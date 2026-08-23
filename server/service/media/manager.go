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
}

type Manager struct {
	registry SlotRegistry
	resolver NodeResolver
	factory  OutputFactory

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
}

func NewManager(registry SlotRegistry) *Manager {
	return NewManagerWith(registry, NewSysfsResolver("", ""), platformFactory{})
}

func NewManagerWith(registry SlotRegistry, resolver NodeResolver, factory OutputFactory) *Manager {
	return &Manager{registry: registry, resolver: resolver, factory: factory, workers: make(map[string]*worker)}
}

func (m *Manager) Applied(ctx context.Context, profile presentation.Profile, plan presentation.Plan) error {
	return m.Reconcile(ctx, profile, plan)
}

func (m *Manager) Suspend() {
	m.mu.Lock()
	workers := m.workers
	m.workers = make(map[string]*worker)
	m.mu.Unlock()
	stopWorkers(workers)
}

func (m *Manager) Reconcile(ctx context.Context, profile presentation.Profile, plan presentation.Plan) error {
	specs, slots := deriveSlots(profile, plan)
	if err := m.registry.SyncSlots(slots); err != nil {
		return fmt.Errorf("sync media slots: %w", err)
	}

	next := make(map[string]*worker, len(specs))
	var failures []error
	for _, spec := range specs {
		node, err := m.resolve(spec)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", spec.ID, err))
			continue
		}
		spec.Node = node
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
	if !w.allowFrame(time.Now(), demand) {
		w.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrFrameRate, frame.SinkID)
	}
	previous, seen := w.sequence[key]
	if seen && int32(frame.Sequence-previous) <= 0 {
		w.mu.Unlock()
		return fmt.Errorf("%w: sequence %d follows %d", ErrStaleFrame, frame.Sequence, previous)
	}
	w.sequence[key] = frame.Sequence
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
		log.Errorf("media output %s: %s", w.spec.ID, err)
	}
	_ = m.registry.SetDemand(w.spec.ID, sources.Demand{})
	_ = m.registry.SetBindingState(w.spec.ID, sources.StateSuspended)
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
			slots = append(slots, sources.Slot{ID: name, Kind: sources.KindCamera, Label: function.Video.FunctionName})
		case presentation.FunctionUAC2:
			specs = append(specs, SlotSpec{ID: name, Kind: sources.KindMicrophone, Label: function.Audio.FunctionName, Audio: function.Audio, FIFOs: slices.Clone(plan.FIFOs[name])})
			slots = append(slots, sources.Slot{ID: name, Kind: sources.KindMicrophone, Label: function.Audio.FunctionName})
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

func fallbackFor(spec SlotSpec) Fallback {
	return func(width, height int) (Packet, error) {
		if spec.Kind == sources.KindMicrophone {
			return Packet{Data: make([]byte, pcmPacketBytes)}, nil
		}
		if width == 0 || height == 0 {
			frame := spec.Video.Formats[0].Frames[0]
			width, height = int(frame.Width), int(frame.Height)
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
		black := image.NewGray(image.Rect(0, 0, width, height))
		var encoded bytes.Buffer
		if err := jpeg.Encode(&encoded, black, &jpeg.Options{Quality: 70}); err != nil {
			return Packet{}, err
		}
		return Packet{Data: encoded.Bytes()}, nil
	}
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
