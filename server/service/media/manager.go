package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"slices"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"

	log "github.com/sirupsen/logrus"
)

const (
	videoQueueDepth = 1
	audioQueueDepth = 4
	pcmPacketBytes  = 1920
	stopTimeout     = 2 * time.Second
	videoFallback   = 500 * time.Millisecond
	videoPoll       = 25 * time.Millisecond
	latencyWindow   = time.Second
)

var (
	ErrSinkUnavailable  = errors.New("media sink unavailable")
	ErrUnsupportedFrame = errors.New("unsupported media frame")
	ErrStaleFrame       = errors.New("stale media frame")
	ErrNotDemanded      = errors.New("media sink is not demanded")
	ErrFrameRate        = errors.New("media frame rate exceeded")
)

type SlotRegistry interface {
	SyncSlots([]sources.Slot) error
	SetDemand(string, sources.Demand) error
	SetBindingState(string, sources.BindingState) error
}

type NodeResolver interface {
	ResolveVideo(string) (string, error)
	ResolveAudio(string) (string, error)
	GadgetVideoNodes() ([]string, error)
}

type Output interface {
	Run(context.Context, <-chan Packet, Fallback, func(sources.Demand), func(bool)) error
	Close() error
}

type OutputFactory interface {
	Open(SlotSpec, string) (Output, error)
}

type Packet struct {
	Sequence   uint32
	Generation uint64
	Data       []byte
	Reset      bool
}

type Fallback func(int, int) (Packet, error)

type SlotSpec struct {
	ID    string
	Kind  sources.Kind
	Label string
	Video *presentation.VideoFunction
	Audio *presentation.AudioFunction
	FIFOs []int
	Node  string
	// FD is the descriptor of the one open(2) the manager keeps on a camera's
	// gadget node for as long as the function is linked (see hold.go). The
	// output dups it; it must never close it, and never open the node itself.
	FD int
}

type Manager struct {
	registry SlotRegistry
	resolver NodeResolver
	factory  OutputFactory

	holds *holdTable

	mu      sync.RWMutex
	workers map[string]*worker
}

var _ sources.FrameIngress = (*Manager)(nil)

type worker struct {
	spec   SlotSpec
	output Output
	queue  chan Packet
	cancel context.CancelFunc
	done   chan struct{}

	mu         sync.Mutex
	sequence   map[string]uint32
	generation uint64
	demand     sources.Demand
	rateAt     time.Time
	rateTokens float64
	latency    latencyTracker
}

type pacer struct {
	period time.Duration
	next   time.Time
}

type latencyTracker struct {
	started time.Time
	offset  int64
	frames  int
	sum     int64
	peak    int64
	summary sources.SinkLatency
}

func NewManager(registry SlotRegistry) *Manager {
	return NewManagerWith(registry, NewSysfsResolver("", ""), platformFactory{})
}

func NewManagerWith(registry SlotRegistry, resolver NodeResolver, factory OutputFactory) *Manager {
	return newManagerWith(registry, resolver, factory, holdNode)
}

func newManagerWith(registry SlotRegistry, resolver NodeResolver, factory OutputFactory, open func(string) (Holder, error)) *Manager {
	return &Manager{
		registry: registry, resolver: resolver, factory: factory,
		holds: newHoldTable(open), workers: make(map[string]*worker),
	}
}

func (m *Manager) Applied(ctx context.Context, profile presentation.Profile, plan presentation.Plan) error {
	return m.Reconcile(ctx, profile, plan)
}

// Suspend runs before the gadget is torn down, so it has to give the nodes back
// as well as stop the workers: configfs refuses to unlink a UVC function whose
// video node is still open. The workers go first because their descriptors are
// dups of the held one and the V4L2 handle only closes on the last of them.
func (m *Manager) Suspend() {
	m.mu.Lock()
	workers := m.workers
	m.workers = make(map[string]*worker)
	m.mu.Unlock()
	stopWorkers(workers)
	if _, err := m.holds.hold(context.Background(), nil); err != nil {
		log.Errorf("release uvc nodes before gadget teardown: %s", err)
	}
}

func (m *Manager) Reconcile(ctx context.Context, profile presentation.Profile, plan presentation.Plan) error {
	specs, slots := deriveSlots(profile, plan)
	if err := m.registry.SyncSlots(slots); err != nil {
		return fmt.Errorf("sync media slots: %w", err)
	}

	// The holds come first and cover every gadget video node, not only the ones
	// a slot resolves to. A node exists only while its UVC function is linked,
	// and every linked UVC function is holding the controller deactivated, so
	// an unheld node is a dark gadget: no HID, no gadget NIC, no virtual disk.
	// A camera whose slot cannot be resolved therefore still gets its node
	// held, and costs its own stream rather than the whole device.
	holders, holdErr := m.holdVideoNodes(ctx)

	next := make(map[string]*worker, len(specs))
	claimed := make(map[string]string, len(specs))
	var failures []error
	if holdErr != nil {
		failures = append(failures, holdErr)
	}
	for _, spec := range specs {
		node, err := m.resolve(spec)
		if err != nil {
			if spec.Kind == sources.KindCamera {
				log.Errorf("media slot %s: %s: the node stays held, so the gadget stays on the bus", spec.ID, err)
			}
			failures = append(failures, fmt.Errorf("%s: %w", spec.ID, err))
			continue
		}
		spec.Node = node
		// Two slots on one node is the shape a partly bound profile takes when
		// the kernel cannot name its functions. One held descriptor is still
		// one, so the bus is safe, but the frames would go to whichever slot
		// wrote last, so the second claim fails instead.
		if owner, taken := claimed[node]; taken {
			failures = append(failures, fmt.Errorf("%s: %w: %s already resolves to %s", spec.ID, ErrNodeIdentityAmbiguous, node, owner))
			continue
		}
		claimed[node] = spec.ID
		if spec.Kind == sources.KindCamera {
			holder, held := holders[node]
			if !held {
				failures = append(failures, fmt.Errorf("%s: %w: %s is not held", spec.ID, ErrSinkUnavailable, node))
				continue
			}
			spec.FD = holder.FD()
		}
		output, err := m.factory.Open(spec, node)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", spec.ID, err))
			continue
		}
		fallback := fallbackFor(spec)
		depth := videoQueueDepth
		if spec.Kind == sources.KindMicrophone {
			depth = audioQueueDepth
		}
		workerCtx, cancel := context.WithCancel(context.Background())
		current := &worker{
			spec: spec, output: output, queue: make(chan Packet, depth), cancel: cancel,
			done: make(chan struct{}), sequence: make(map[string]uint32),
		}
		next[spec.ID] = current
		go m.run(workerCtx, current, fallback)
	}

	if err := ctx.Err(); err != nil {
		stopWorkers(next)
		return err
	}
	m.mu.Lock()
	previous := m.workers
	m.workers = next
	m.mu.Unlock()
	stopWorkers(previous)
	return errors.Join(failures...)
}

func (m *Manager) Ingest(ctx context.Context, frame sources.MediaFrame) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.RLock()
	w := m.workers[frame.SinkID]
	m.mu.RUnlock()
	if w == nil {
		return fmt.Errorf("%w: %s", ErrSinkUnavailable, frame.SinkID)
	}
	width, height, err := validateFrame(w.spec, frame)
	if err != nil {
		return err
	}
	key := frame.SourceID + "\x00" + frame.StreamID
	w.mu.Lock()
	demand := w.demand
	if !demand.Streaming {
		w.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrNotDemanded, frame.SinkID)
	}
	if w.spec.Kind == sources.KindCamera && (width != demand.Width || height != demand.Height) {
		w.mu.Unlock()
		return fmt.Errorf("%w: host requests %dx%d, frame is %dx%d", ErrUnsupportedFrame, demand.Width, demand.Height, width, height)
	}
	now := time.Now()
	if !w.allowFrame(now, demand) {
		w.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrFrameRate, frame.SinkID)
	}
	previous, seen := w.sequence[key]
	if seen && int32(frame.Sequence-previous) <= 0 {
		w.mu.Unlock()
		return fmt.Errorf("%w: sequence %d follows %d", ErrStaleFrame, frame.Sequence, previous)
	}
	w.sequence[key] = frame.Sequence
	w.latency.observe(now, frame.TimestampUS)
	packet := Packet{Sequence: frame.Sequence, Generation: w.generation, Data: append([]byte(nil), frame.Payload...)}
	defer w.mu.Unlock()

	select {
	case w.queue <- packet:
		return nil
	default:
	}
	select {
	case <-w.queue:
	default:
	}
	select {
	case w.queue <- packet:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) Detach(sinkID string) {
	m.mu.RLock()
	w := m.workers[sinkID]
	m.mu.RUnlock()
	if w == nil {
		return
	}
	w.mu.Lock()
	clear(w.sequence)
	w.generation++
	w.rateAt = time.Time{}
	w.rateTokens = 0
	w.latency = latencyTracker{}
	generation := w.generation
	w.mu.Unlock()
	for {
		select {
		case <-w.queue:
		default:
			select {
			case w.queue <- Packet{Generation: generation, Reset: true}:
			default:
			}
			return
		}
	}
}

func (w *worker) allowFrame(now time.Time, demand sources.Demand) bool {
	rate, burst := float64(50), float64(audioQueueDepth)
	if w.spec.Kind == sources.KindCamera {
		rate, burst = float64(demand.FPS), float64(videoQueueDepth+1)
		if rate <= 0 {
			return false
		}
	}
	if w.rateAt.IsZero() {
		w.rateAt, w.rateTokens = now, burst
	} else {
		w.rateTokens = min(burst, w.rateTokens+now.Sub(w.rateAt).Seconds()*rate)
		w.rateAt = now
	}
	if w.rateTokens < 1 {
		return false
	}
	w.rateTokens--
	return true
}

func (t *latencyTracker) observe(now time.Time, stampUS uint64) {
	if stampUS == 0 {
		return
	}
	skew := now.UnixMicro() - int64(stampUS)
	if t.started.IsZero() {
		t.started, t.offset = now, skew
	}
	if skew < t.offset {
		t.offset = skew
	}
	sample := skew - t.offset
	t.frames++
	t.sum += sample
	if sample > t.peak {
		t.peak = sample
	}
	if now.Sub(t.started) < latencyWindow {
		return
	}
	t.summary = sources.SinkLatency{
		Frames:    t.frames,
		AvgMS:     float64(t.sum) / float64(t.frames) / 1000,
		PeakMS:    float64(t.peak) / 1000,
		BaseMS:    float64(t.offset) / 1000,
		UpdatedAt: now.UTC(),
	}
	t.started, t.frames, t.sum, t.peak = now, 0, 0, 0
}

func (m *Manager) Latency() map[string]sources.SinkLatency {
	m.mu.RLock()
	defer m.mu.RUnlock()
	summaries := make(map[string]sources.SinkLatency, len(m.workers))
	for id, w := range m.workers {
		w.mu.Lock()
		summary := w.latency.summary
		w.mu.Unlock()
		if summary.Frames > 0 {
			summaries[id] = summary
		}
	}
	return summaries
}

func (p *pacer) due(now time.Time, fps int) (int, bool) {
	period := time.Duration(0)
	if fps > 0 {
		period = time.Second / time.Duration(fps)
	}
	if period != p.period {
		p.period, p.next = period, now
	}
	if period > 0 && now.Before(p.next) {
		return int((min(p.next.Sub(now), videoPoll) + time.Millisecond - 1) / time.Millisecond), false
	}
	p.next = p.next.Add(period)
	if p.next.Before(now) {
		p.next = now.Add(period)
	}
	return int(videoPoll / time.Millisecond), true
}

func (m *Manager) run(ctx context.Context, w *worker, fallback Fallback) {
	defer close(w.done)
	defer w.output.Close()
	if err := w.output.Run(ctx, w.queue, fallback, func(demand sources.Demand) {
		if !demand.Streaming {
			demand = sources.Demand{}
		}
		w.mu.Lock()
		w.demand = demand
		w.mu.Unlock()
		_ = m.registry.SetDemand(w.spec.ID, demand)
		state := sources.StateSuspended
		if demand.Streaming {
			state = sources.StateClaimed
		}
		_ = m.registry.SetBindingState(w.spec.ID, state)
	}, func(active bool) {
		state := sources.StateClaimed
		if active {
			state = sources.StateStreaming
		}
		_ = m.registry.SetBindingState(w.spec.ID, state)
	}); err != nil {
		log.Errorf("media output %s: %s: the node stays held, so the gadget stays on the bus", w.spec.ID, err)
	}
	_ = m.registry.SetDemand(w.spec.ID, sources.Demand{})
	_ = m.registry.SetBindingState(w.spec.ID, sources.StateSuspended)
}

// holdVideoNodes converges the held set on what the gadget currently exposes.
// It is bounded: a node that will not open, or a hold that will not settle, is
// an error the caller reports now, never a request that hangs.
func (m *Manager) holdVideoNodes(ctx context.Context) (map[string]Holder, error) {
	nodes, err := m.resolver.GadgetVideoNodes()
	if err != nil {
		log.Errorf("list gadget video nodes: %s", err)
		return nil, fmt.Errorf("list gadget video nodes: %w", err)
	}
	holders, err := m.holds.hold(ctx, nodes)
	if err != nil {
		log.Errorf("hold gadget video nodes %v: %s", nodes, err)
		return nil, err
	}
	var missing []string
	for _, node := range nodes {
		if _, held := holders[node]; !held {
			missing = append(missing, node)
		}
	}
	if len(missing) > 0 {
		return holders, fmt.Errorf("%w: gadget video nodes %v stayed closed, so the controller stays deactivated", ErrSinkUnavailable, missing)
	}
	return holders, nil
}

func (m *Manager) resolve(spec SlotSpec) (string, error) {
	if spec.Kind == sources.KindCamera {
		return m.resolver.ResolveVideo(spec.ID)
	}
	return m.resolver.ResolveAudio(spec.ID)
}

func deriveSlots(profile presentation.Profile, plan presentation.Plan) ([]SlotSpec, []sources.Slot) {
	var specs []SlotSpec
	var slots []sources.Slot
	for _, function := range profile.Functions {
		name := string(function.Kind) + "." + function.Instance
		switch function.Kind {
		case presentation.FunctionUVC:
			specs = append(specs, SlotSpec{ID: name, Kind: sources.KindCamera, Label: function.Video.FunctionName, Video: function.Video, FIFOs: slices.Clone(plan.FIFOs[name])})
			slots = append(slots, sources.Slot{ID: name, Kind: sources.KindCamera, Label: function.Video.FunctionName, HostName: plan.MediaNames[name]})
		case presentation.FunctionUAC2:
			specs = append(specs, SlotSpec{ID: name, Kind: sources.KindMicrophone, Label: function.Audio.FunctionName, Audio: function.Audio, FIFOs: slices.Clone(plan.FIFOs[name])})
			slots = append(slots, sources.Slot{ID: name, Kind: sources.KindMicrophone, Label: function.Audio.FunctionName, HostName: plan.MediaNames[name]})
		}
	}
	return specs, slots
}

func validateFrame(spec SlotSpec, frame sources.MediaFrame) (int, int, error) {
	if spec.Kind == sources.KindCamera {
		if frame.Kind != sources.MediaKindMJPEG {
			return 0, 0, fmt.Errorf("%w: %s accepts MJPEG", ErrUnsupportedFrame, spec.ID)
		}
		if len(frame.Payload) < 4 || len(frame.Payload) > 2<<20 || !bytes.Equal(frame.Payload[:2], []byte{0xff, 0xd8}) || !bytes.Equal(frame.Payload[len(frame.Payload)-2:], []byte{0xff, 0xd9}) {
			return 0, 0, fmt.Errorf("%w: invalid MJPEG payload", ErrUnsupportedFrame)
		}
		config, err := jpeg.DecodeConfig(bytes.NewReader(frame.Payload))
		if err != nil {
			return 0, 0, fmt.Errorf("%w: decode MJPEG: %v", ErrUnsupportedFrame, err)
		}
		for _, format := range spec.Video.Formats {
			for _, candidate := range format.Frames {
				if config.Width == int(candidate.Width) && config.Height == int(candidate.Height) {
					if len(frame.Payload) > config.Width*config.Height*2 {
						return 0, 0, fmt.Errorf("%w: MJPEG exceeds the declared %dx%d frame buffer", ErrUnsupportedFrame, config.Width, config.Height)
					}
					return config.Width, config.Height, nil
				}
			}
		}
		return 0, 0, fmt.Errorf("%w: %dx%d is not declared by %s", ErrUnsupportedFrame, config.Width, config.Height, spec.ID)
	}
	if frame.Kind != sources.MediaKindPCMS16LE || len(frame.Payload) != pcmPacketBytes {
		return 0, 0, fmt.Errorf("%w: %s accepts 20 ms mono S16LE 48000 Hz packets", ErrUnsupportedFrame, spec.ID)
	}
	return 0, 0, nil
}

var emptyPacket [1]byte

func packetSpan(p Packet) (*byte, int) {
	if len(p.Data) == 0 {
		return &emptyPacket[0], 0
	}
	return &p.Data[0], len(p.Data)
}

func fallbackFor(spec SlotSpec) Fallback {
	cache := make(map[[2]int][]byte, 1)
	return func(width, height int) (Packet, error) {
		if spec.Kind == sources.KindMicrophone {
			return Packet{Data: make([]byte, pcmPacketBytes)}, nil
		}
		if width == 0 || height == 0 {
			frame := spec.Video.Formats[0].Frames[0]
			width, height = int(frame.Width), int(frame.Height)
		}
		if data, cached := cache[[2]int{width, height}]; cached {
			return Packet{Data: data}, nil
		}
		declared := false
		for _, format := range spec.Video.Formats {
			for _, frame := range format.Frames {
				declared = declared || width == int(frame.Width) && height == int(frame.Height)
			}
		}
		if !declared {
			return Packet{}, fmt.Errorf("fallback %dx%d is not declared", width, height)
		}
		data, err := encodeBlackFrame(width, height)
		if err != nil {
			return Packet{}, err
		}
		cache[[2]int{width, height}] = data
		return Packet{Data: data}, nil
	}
}

func encodeBlackFrame(width, height int) ([]byte, error) {
	black := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
	for i := range black.Cb {
		black.Cb[i], black.Cr[i] = 128, 128
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, black, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func stopWorkers(workers map[string]*worker) {
	for _, worker := range workers {
		worker.cancel()
		_ = worker.output.Close()
	}
	deadline := time.NewTimer(stopTimeout)
	defer deadline.Stop()
	for _, worker := range workers {
		select {
		case <-worker.done:
		case <-deadline.C:
			return
		}
	}
}

func parseALSANode(node string) (int, int, error) {
	var card, device int
	if _, err := fmt.Sscanf(strings.TrimSpace(node), "hw:%d,%d", &card, &device); err != nil || card < 0 || device < 0 {
		return 0, 0, fmt.Errorf("invalid ALSA node %q", node)
	}
	return card, device, nil
}
