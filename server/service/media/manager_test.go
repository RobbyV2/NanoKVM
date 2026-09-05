package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
)

type fakeRegistry struct {
	mu      sync.Mutex
	slots   []sources.Slot
	demands map[string]sources.Demand
}

func (r *fakeRegistry) SyncSlots(slots []sources.Slot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slots = append([]sources.Slot(nil), slots...)
	return nil
}

func (r *fakeRegistry) SetDemand(id string, demand sources.Demand) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.demands == nil {
		r.demands = make(map[string]sources.Demand)
	}
	r.demands[id] = demand
	return nil
}

func (r *fakeRegistry) SetBindingState(string, sources.BindingState) error { return nil }

type fakeResolver struct{ nodes map[string]string }

func (r fakeResolver) ResolveVideo(id string) (string, error) { return r.nodes[id], nil }
func (r fakeResolver) ResolveAudio(id string) (string, error) { return r.nodes[id], nil }

func (r fakeResolver) GadgetVideoNodes() ([]string, error) {
	var nodes []string
	for _, node := range r.nodes {
		if strings.HasPrefix(node, "/dev/video") {
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	return nodes, nil
}

// One open per node, counted, so a test can prove the manager never opens a
// gadget video node twice: the second open would leak a deactivation the
// kernel never pays back.
type fakeHolds struct {
	mu     sync.Mutex
	opens  map[string]int
	closes map[string]int
	fail   map[string]error
	block  chan struct{}
	next   int
}

type fakeHolder struct {
	table *fakeHolds
	node  string
	fd    int
}

func (h *fakeHolder) FD() int { return h.fd }

func (h *fakeHolder) Close() error {
	h.table.mu.Lock()
	defer h.table.mu.Unlock()
	h.table.closes[h.node]++
	return nil
}

func newFakeHolds() *fakeHolds {
	return &fakeHolds{opens: map[string]int{}, closes: map[string]int{}, fail: map[string]error{}, next: 100}
}

func (t *fakeHolds) open(node string) (Holder, error) {
	t.mu.Lock()
	t.opens[node]++
	err := t.fail[node]
	block := t.block
	t.next++
	fd := t.next
	t.mu.Unlock()
	if block != nil {
		<-block
	}
	if err != nil {
		return nil, err
	}
	return &fakeHolder{table: t, node: node, fd: fd}, nil
}

func (t *fakeHolds) count(node string) (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.opens[node], t.closes[node]
}

func newTestManager(registry SlotRegistry, resolver NodeResolver, factory OutputFactory) *Manager {
	return newManagerWith(registry, resolver, factory, newFakeHolds().open)
}

type fakeFactory struct {
	mu      sync.Mutex
	outputs map[string]*fakeOutput
	inputs  map[string]*fakeInput
	specs   map[string]SlotSpec
}

func (f *fakeFactory) Open(spec SlotSpec, _ string) (Output, error) {
	output := &fakeOutput{packets: make(chan Packet, 8), closed: make(chan struct{})}
	f.mu.Lock()
	if f.outputs == nil {
		f.outputs = make(map[string]*fakeOutput)
	}
	f.outputs[spec.ID] = output
	if f.specs == nil {
		f.specs = make(map[string]SlotSpec)
	}
	f.specs[spec.ID] = spec
	f.mu.Unlock()
	return output, nil
}

func (f *fakeFactory) OpenInput(spec SlotSpec, _ string) (Input, error) {
	input := &fakeInput{emit: make(chan func(Packet), 1), closed: make(chan struct{})}
	f.mu.Lock()
	if f.inputs == nil {
		f.inputs = make(map[string]*fakeInput)
	}
	f.inputs[spec.ID] = input
	if f.specs == nil {
		f.specs = make(map[string]SlotSpec)
	}
	f.specs[spec.ID] = spec
	f.mu.Unlock()
	return input, nil
}

func (f *fakeFactory) input(id string) *fakeInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inputs[id]
}

// fakeInput stands in for a gadget capture substream: it hands the manager the
// emit callback so a test can push a period through at a moment of its own
// choosing.
type fakeInput struct {
	once   sync.Once
	emit   chan func(Packet)
	closed chan struct{}
}

func (i *fakeInput) Run(ctx context.Context, emit func(Packet), demand func(sources.Demand), active func(bool)) error {
	demand(sources.Demand{Streaming: true, Since: time.Now()})
	active(true)
	select {
	case i.emit <- emit:
	default:
	}
	<-ctx.Done()
	return nil
}

func (i *fakeInput) Close() error {
	i.once.Do(func() { close(i.closed) })
	return nil
}

func (f *fakeFactory) spec(id string) SlotSpec {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.specs[id]
}

type fakeOutput struct {
	once    sync.Once
	packets chan Packet
	closed  chan struct{}
}

type blockedFactory struct{ output *blockedOutput }

func (f *blockedFactory) Open(SlotSpec, string) (Output, error) {
	f.output = &blockedOutput{}
	return f.output, nil
}

func (f *blockedFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

type blockedOutput struct{}

func (*blockedOutput) Run(ctx context.Context, _ <-chan Packet, _ Fallback, demand func(sources.Demand), _ func(bool)) error {
	demand(sources.Demand{Streaming: true, Width: 640, Height: 480, FPS: 30, Since: time.Now()})
	<-ctx.Done()
	return nil
}

func (*blockedOutput) Close() error { return nil }

type idleOutput struct{}

func (*idleOutput) Run(ctx context.Context, _ <-chan Packet, _ Fallback, _ func(sources.Demand), _ func(bool)) error {
	<-ctx.Done()
	return nil
}

func (*idleOutput) Close() error { return nil }

type idleFactory struct{}

func (idleFactory) Open(SlotSpec, string) (Output, error) { return &idleOutput{}, nil }

func (idleFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

func (o *fakeOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	demand(sources.Demand{Streaming: true, Width: 640, Height: 480, FPS: 30, Since: time.Now()})
	packet, _ := fallback(640, 480)
	o.packets <- packet
	for {
		select {
		case <-ctx.Done():
			return nil
		case packet := <-frames:
			source(!packet.Reset)
			o.packets <- packet
		}
	}
}

func (o *fakeOutput) Close() error {
	o.once.Do(func() { close(o.closed) })
	return nil
}

func cameraFunction(instance string) presentation.Function {
	return presentation.Function{Kind: presentation.FunctionUVC, Instance: instance, Video: &presentation.VideoFunction{
		FunctionName: "Camera " + instance, StreamingMaxPacket: 768, StreamingInterval: 1,
		Formats: []presentation.VideoFormat{{Codec: "mjpeg", Frames: []presentation.VideoFrame{{Width: 640, Height: 480, Intervals: []uint32{333333}}}}},
	}}
}

func microphoneFunction(instance string) presentation.Function {
	return presentation.Function{Kind: presentation.FunctionUAC2, Instance: instance, Audio: &presentation.AudioFunction{
		FunctionName: "Microphone " + instance, PChannelMask: 1, PSampleRate: 48000, PSampleSize: 2, RequestNumber: 4,
	}}
}

func jpegFrame(t *testing.T, width, height int) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewGray(image.Rect(0, 0, width, height)), nil); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func waitDemand(t *testing.T, registry *fakeRegistry, id string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		registry.mu.Lock()
		streaming := registry.demands[id].Streaming
		registry.mu.Unlock()
		if streaming {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("%s never became demanded", id)
}

func TestManagerKeepsSlotsIndependentAndBounded(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video8", "uvc.cam1": "/dev/video3"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0"), cameraFunction("cam1")}}
	plan := presentation.Plan{FIFOs: presentation.FIFOAssignment{"uvc.cam0": {128, 768}, "uvc.cam1": {128, 512}}}
	if err := manager.Reconcile(context.Background(), profile, plan); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam1")

	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam1", Kind: sources.MediaKindMJPEG, Sequence: 1, Payload: jpegFrame(t, 640, 480)}
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	select {
	case packet := <-factory.outputs["uvc.cam1"].packets:
		if packet.Sequence == 0 {
			select {
			case packet = <-factory.outputs["uvc.cam1"].packets:
			case <-time.After(time.Second):
				t.Fatal("cam1 fallback was not replaced")
			}
		}
		if packet.Sequence != 1 {
			t.Fatalf("cam1 sequence = %d", packet.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("cam1 did not receive its frame")
	}
	select {
	case packet := <-factory.outputs["uvc.cam0"].packets:
		if packet.Sequence != 0 {
			t.Fatalf("cam0 received cam1 sequence %d", packet.Sequence)
		}
	default:
	}
	if err := manager.Ingest(context.Background(), frame); err == nil {
		t.Fatal("accepted a repeated sequence")
	}
	frame.Sequence = 0
	if err := manager.Ingest(context.Background(), frame); err == nil {
		t.Fatal("accepted an older sequence")
	}
}

func TestDetachDropsQueuedFrameAndSequenceState(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")

	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 7, Payload: jpegFrame(t, 640, 480)}
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	manager.Detach("uvc.cam0")
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatalf("sequence remained stale after detach: %v", err)
	}
}

func TestManagerRejectsUndeclaredFormats(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")
	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 1, Payload: jpegFrame(t, 320, 240)}
	if err := manager.Ingest(context.Background(), frame); err == nil {
		t.Fatal("accepted an undeclared resolution")
	}
}

func TestManagerRejectsMJPEGBeyondDeclaredBuffer(t *testing.T) {
	payload := jpegFrame(t, 640, 480)
	padding := make([]byte, 640*480*2+1-len(payload))
	payload = append(append(append([]byte(nil), payload[:len(payload)-2]...), padding...), 0xff, 0xd9)
	frame := sources.MediaFrame{Kind: sources.MediaKindMJPEG, Payload: payload}
	if _, _, err := validateFrame(SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}, frame); !errors.Is(err, ErrUnsupportedFrame) {
		t.Fatalf("err = %v, want ErrUnsupportedFrame", err)
	}
}

func TestManagerKeepsOnlyLatestVideoFrame(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &blockedFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")
	payload := jpegFrame(t, 640, 480)
	for sequence := uint32(1); sequence <= 2; sequence++ {
		frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: sequence, Payload: payload}
		if err := manager.Ingest(context.Background(), frame); err != nil {
			t.Fatal(err)
		}
	}
	worker := manager.workers["uvc.cam0"]
	if len(worker.queue) != 1 {
		t.Fatalf("queue length = %d, want 1", len(worker.queue))
	}
	if packet := <-worker.queue; packet.Sequence != 2 {
		t.Fatalf("queued sequence = %d, want 2", packet.Sequence)
	}
	manager.Detach("uvc.cam0")
	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 2, Payload: payload}
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	packet := <-worker.queue
	if packet.Reset || packet.Generation != 1 || packet.Sequence != 2 {
		t.Fatalf("post-detach packet = %+v, want generation 1 sequence 2", packet)
	}
}

func TestManagerRejectsBurstBeyondSlotRate(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &blockedFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")
	// The demand is 30 fps, so the bucket carries half a second of frames. A
	// wireless link that hands over a bunch that size is jitter, not a runaway
	// source, and every frame in it has to be admitted; the frame after the
	// burst is the one the long run rate refuses.
	payload := jpegFrame(t, 640, 480)
	const burst = 15
	for sequence := uint32(1); sequence <= burst; sequence++ {
		err := manager.Ingest(context.Background(), sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: sequence, Payload: payload})
		if err != nil {
			t.Fatalf("frame %d of the burst: %v", sequence, err)
		}
	}
	err := manager.Ingest(context.Background(), sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: burst + 1, Payload: payload})
	if !errors.Is(err, ErrFrameRate) {
		t.Fatalf("frame past the burst err = %v, want ErrFrameRate", err)
	}
}

func TestManagerRejectsFramesWithoutHostDemand(t *testing.T) {
	registry := &fakeRegistry{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, idleFactory{})
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 1, Payload: jpegFrame(t, 640, 480)}
	if err := manager.Ingest(context.Background(), frame); !errors.Is(err, ErrNotDemanded) {
		t.Fatalf("err = %v, want ErrNotDemanded", err)
	}
}

func TestManagerSynchronizesSlotsBeforeNodeDiscoveryFails(t *testing.T) {
	registry := &fakeRegistry{}
	manager := newTestManager(registry, failingAudioResolver{}, &fakeFactory{})
	profile := presentation.Profile{Functions: []presentation.Function{microphoneFunction("mic0"), microphoneFunction("mic1")}}
	err := manager.Reconcile(context.Background(), profile, presentation.Plan{})
	if err == nil {
		t.Fatal("missing audio identity did not fail closed")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if len(registry.slots) != 2 || registry.slots[0].ID != "uac2.mic0" || registry.slots[1].ID != "uac2.mic1" {
		t.Fatalf("slots = %+v", registry.slots)
	}
}

type failingAudioResolver struct{}

func (failingAudioResolver) ResolveVideo(string) (string, error) { return "", ErrNodeNotFound }
func (failingAudioResolver) ResolveAudio(string) (string, error) {
	return "", ErrAudioIdentityUnavailable
}
func (failingAudioResolver) GadgetVideoNodes() ([]string, error) { return nil, nil }

func TestSuspendClosesEverySlot(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0", "uac2.mic0": "hw:1,0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0"), microphoneFunction("mic0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	manager.Suspend()
	for id, output := range factory.outputs {
		select {
		case <-output.closed:
		default:
			t.Fatalf("%s was not closed", id)
		}
	}
}

func TestFallbackBlackFrameIsThreeComponentYCbCrAndCached(t *testing.T) {
	spec := SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}
	fallback := fallbackFor(spec)
	first, err := fallback(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	components, width, height := jpegFrameHeader(t, first.Data)
	if components != 3 {
		t.Fatalf("component count = %d, want 3", components)
	}
	if width != 640 || height != 480 {
		t.Fatalf("frame = %dx%d, want 640x480", width, height)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(first.Data))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded.(*image.YCbCr); !ok {
		t.Fatalf("decoded %T, want *image.YCbCr", decoded)
	}
	if bounds := decoded.Bounds(); bounds.Dx() != 640 || bounds.Dy() != 480 {
		t.Fatalf("decoded bounds = %v", bounds)
	}
	if r, g, b, _ := decoded.At(320, 240).RGBA(); r > 0x0808 || g > 0x0808 || b > 0x0808 {
		t.Fatalf("centre pixel = %d %d %d, want black", r, g, b)
	}
	second, err := fallback(640, 480)
	if err != nil {
		t.Fatal(err)
	}
	if &second.Data[0] != &first.Data[0] {
		t.Fatal("black frame was re-encoded instead of cached")
	}
}

func jpegFrameHeader(t *testing.T, data []byte) (int, int, int) {
	t.Helper()
	for i := 2; i+9 < len(data); {
		marker := data[i+1]
		if data[i] != 0xff {
			t.Fatalf("offset %d is not a marker", i)
		}
		if marker == 0xd8 || marker == 0xd9 || (marker >= 0xd0 && marker <= 0xd7) {
			i += 2
			continue
		}
		if marker == 0xc0 {
			return int(data[i+9]), int(binary.BigEndian.Uint16(data[i+7:])), int(binary.BigEndian.Uint16(data[i+5:]))
		}
		i += 2 + int(binary.BigEndian.Uint16(data[i+2:]))
	}
	t.Fatal("no baseline SOF0 marker")
	return 0, 0, 0
}

func TestPacerHoldsSubmissionUntilTheNegotiatedInterval(t *testing.T) {
	base := time.Unix(1700000000, 0)
	var pace pacer
	if timeout, submit := pace.due(base, 0); !submit || timeout != 25 {
		t.Fatalf("undemanded step = (%d, %v), want (25, true)", timeout, submit)
	}
	if _, submit := pace.due(base, 30); !submit {
		t.Fatal("first 30 fps step did not submit")
	}
	if timeout, submit := pace.due(base.Add(time.Millisecond), 30); submit || timeout != 25 {
		t.Fatalf("step 1 ms in = (%d, %v), want (25, false)", timeout, submit)
	}
	if timeout, submit := pace.due(base.Add(20*time.Millisecond), 30); submit || timeout != 14 {
		t.Fatalf("step 20 ms in = (%d, %v), want (14, false)", timeout, submit)
	}
	if _, submit := pace.due(base.Add(34*time.Millisecond), 30); !submit {
		t.Fatal("step past the 33.3 ms interval did not submit")
	}
	if _, submit := pace.due(base.Add(40*time.Millisecond), 30); submit {
		t.Fatal("second frame submitted 6 ms after the first")
	}
	if _, submit := pace.due(base.Add(time.Second), 30); !submit {
		t.Fatal("late step did not submit")
	}
	if _, submit := pace.due(base.Add(time.Second+time.Millisecond), 30); submit {
		t.Fatal("a late step burst instead of resynchronizing")
	}
	if _, submit := pace.due(base.Add(time.Second+2*time.Millisecond), 15); !submit {
		t.Fatal("renegotiated interval did not submit immediately")
	}
	if _, submit := pace.due(base.Add(time.Second+40*time.Millisecond), 15); submit {
		t.Fatal("15 fps submitted after 38 ms")
	}
}

func TestLatencyTrackerSummarizesEachWindow(t *testing.T) {
	base := time.Unix(1700000000, 0)
	var tracker latencyTracker
	observe := func(offset, skew time.Duration) {
		now := base.Add(offset)
		tracker.observe(now, uint64(now.Add(-skew).UnixMicro()))
	}
	observe(0, 100*time.Millisecond)
	observe(300*time.Millisecond, 160*time.Millisecond)
	if tracker.summary.Frames != 0 {
		t.Fatalf("summary published mid-window: %+v", tracker.summary)
	}
	observe(600*time.Millisecond, 130*time.Millisecond)
	observe(time.Second, 100*time.Millisecond)
	summary := tracker.summary
	if summary.Frames != 4 || summary.AvgMS != 22.5 || summary.PeakMS != 60 || summary.BaseMS != 100 {
		t.Fatalf("summary = %+v, want 4 frames avg 22.5 peak 60 base 100", summary)
	}
	if !summary.UpdatedAt.Equal(base.Add(time.Second).UTC()) {
		t.Fatalf("summary updated at %s", summary.UpdatedAt)
	}
	observe(1200*time.Millisecond, 80*time.Millisecond)
	if tracker.offset != 80000 {
		t.Fatalf("baseline = %d us, want 80000", tracker.offset)
	}
	if tracker.summary != summary {
		t.Fatal("summary changed mid-window")
	}
}

func TestIngestMeasuresFrameLatency(t *testing.T) {
	registry := &fakeRegistry{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, &blockedFactory{})
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")
	sent := time.Now().Add(-120 * time.Millisecond)
	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 1, TimestampUS: uint64(sent.UnixMicro()), Payload: jpegFrame(t, 640, 480)}
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	worker := manager.workers["uvc.cam0"]
	worker.mu.Lock()
	frames, offset := worker.latency.frames, worker.latency.offset
	started := worker.latency.started
	worker.mu.Unlock()
	if frames != 1 {
		t.Fatalf("recorded %d frames, want 1", frames)
	}
	if offset < 120000 {
		t.Fatalf("baseline = %d us, want at least 120000", offset)
	}
	closing := started.Add(time.Second)
	worker.mu.Lock()
	worker.latency.observe(closing, uint64(closing.Add(-time.Duration(offset)*time.Microsecond-60*time.Millisecond).UnixMicro()))
	worker.mu.Unlock()
	summary, reported := manager.Latency()["uvc.cam0"]
	if !reported || summary.Frames != 2 || summary.PeakMS != 60 {
		t.Fatalf("summary = %+v reported = %v, want 2 frames peaking at 60 ms", summary, reported)
	}
	manager.Detach("uvc.cam0")
	if len(manager.Latency()) != 0 {
		t.Fatalf("latency of a detached source survived: %+v", manager.Latency())
	}
}

func TestManagerRejectsEmptyPayloads(t *testing.T) {
	camera := SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}
	if _, _, err := validateFrame(camera, sources.MediaFrame{Kind: sources.MediaKindMJPEG}); !errors.Is(err, ErrUnsupportedFrame) {
		t.Fatalf("empty MJPEG err = %v, want ErrUnsupportedFrame", err)
	}
	microphone := SlotSpec{ID: "uac2.mic0", Kind: sources.KindMicrophone, Audio: microphoneFunction("mic0").Audio}
	if _, _, err := validateFrame(microphone, sources.MediaFrame{Kind: sources.MediaKindPCMS16LE}); !errors.Is(err, ErrUnsupportedFrame) {
		t.Fatalf("empty PCM err = %v, want ErrUnsupportedFrame", err)
	}
}

func TestPacketSpanNeverIndexesAnEmptyPayload(t *testing.T) {
	base, size := packetSpan(Packet{Generation: 1, Reset: true})
	if base == nil || size != 0 {
		t.Fatalf("reset span = (%v, %d), want a usable pointer and 0", base, size)
	}
	data := []byte{0xff, 0xd8, 0xff, 0xd9}
	if base, size = packetSpan(Packet{Data: data}); base != &data[0] || size != len(data) {
		t.Fatalf("payload span = (%v, %d), want (%v, %d)", base, size, &data[0], len(data))
	}
}

func TestSlotsCarryTheNameTheHostWillRead(t *testing.T) {
	registry := &fakeRegistry{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0", "uac2.mic0": "hw:0,0"}}, &fakeFactory{})
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0"), microphoneFunction("mic0")}}
	plan := presentation.Plan{MediaNames: map[string]string{"uvc.cam0": "Desk Camera"}}
	if err := manager.Reconcile(context.Background(), profile, plan); err != nil {
		t.Fatal(err)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.slots[0].HostName != "Desk Camera" {
		t.Fatalf("camera host name = %q, want the name the plan writes", registry.slots[0].HostName)
	}
	if registry.slots[1].HostName != "" {
		t.Fatalf("microphone host name = %q, want empty where the kernel cannot carry one", registry.slots[1].HostName)
	}
}

// The gadget the operator actually runs: three HID functions, a NIC and the
// virtual disk, plus one camera. f_uvc deactivates the whole composite device
// on bind, so this profile is the one that proves adding a camera does not cost
// the keyboard, the network and the disk.
func compositeProfile() presentation.Profile {
	return presentation.Profile{Functions: []presentation.Function{
		{Kind: presentation.FunctionHID, Instance: "GS0"},
		{Kind: presentation.FunctionHID, Instance: "GS1"},
		{Kind: presentation.FunctionHID, Instance: "GS2"},
		{Kind: presentation.FunctionNCM, Instance: "usb0"},
		{Kind: presentation.FunctionMassStorage, Instance: "disk0"},
		cameraFunction("cam0"),
	}}
}

func TestCameraNodeIsHeldOnceForTheLifetimeOfTheFunction(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	holds := newFakeHolds()
	manager := newManagerWith(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory, holds.open)
	if err := manager.Reconcile(context.Background(), compositeProfile(), presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("after apply the node was opened %d times and closed %d, want 1 and 0", opens, closes)
	}
	if fd := factory.spec("uvc.cam0").FD; fd == 0 {
		t.Fatal("the camera output was opened without the held descriptor, so it opened the node itself")
	}
	// usb_function_activate refuses to decrement cdev->deactivations past zero
	// while uvc_v4l2_release always increments it, so a second open of a node
	// that is already held leaks a deactivation no later open can pay back.
	if err := manager.Reconcile(context.Background(), compositeProfile(), presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("reapplying the same profile opened the node %d times and closed it %d, want 1 and 0", opens, closes)
	}
	manager.Suspend()
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 1 {
		t.Fatalf("after suspend the node was opened %d times and closed %d, want 1 and 1", opens, closes)
	}
}

type listResolver struct {
	mu    sync.Mutex
	nodes []string
	video map[string]string
	err   error
}

func (r *listResolver) ResolveVideo(id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return "", r.err
	}
	return r.video[id], nil
}

func (r *listResolver) ResolveAudio(string) (string, error) { return "", ErrNodeNotFound }

func (r *listResolver) GadgetVideoNodes() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.nodes...), nil
}

func (r *listResolver) set(nodes ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes = nodes
}

func TestUnresolvableCameraStillHoldsItsNode(t *testing.T) {
	registry := &fakeRegistry{}
	holds := newFakeHolds()
	resolver := &listResolver{nodes: []string{"/dev/video0", "/dev/video1"}, err: ErrNodeIdentityAmbiguous}
	manager := newManagerWith(registry, resolver, &fakeFactory{}, holds.open)
	defer manager.Suspend()
	err := manager.Reconcile(context.Background(), presentation.Profile{
		Functions: []presentation.Function{cameraFunction("cam0"), cameraFunction("cam1")},
	}, presentation.Plan{})
	if !errors.Is(err, ErrNodeIdentityAmbiguous) {
		t.Fatalf("err = %v, want the ambiguous identity reported", err)
	}
	for _, node := range []string{"/dev/video0", "/dev/video1"} {
		if opens, closes := holds.count(node); opens != 1 || closes != 0 {
			t.Fatalf("%s was opened %d times and closed %d, want 1 and 0: an unheld node keeps the whole gadget deactivated", node, opens, closes)
		}
	}
}

func TestNodesAreReleasedWhenTheirFunctionIsUnlinked(t *testing.T) {
	registry := &fakeRegistry{}
	holds := newFakeHolds()
	resolver := &listResolver{nodes: []string{"/dev/video0"}, video: map[string]string{"uvc.cam0": "/dev/video0"}}
	manager := newManagerWith(registry, resolver, &fakeFactory{}, holds.open)
	if err := manager.Reconcile(context.Background(), compositeProfile(), presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("node opened %d times and closed %d, want 1 and 0", opens, closes)
	}
	resolver.set()
	if err := manager.Reconcile(context.Background(), presentation.Profile{}, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 1 {
		t.Fatalf("after the camera was dropped the node was opened %d times and closed %d, want 1 and 1: configfs refuses to unlink a function whose node is still open", opens, closes)
	}
}

func TestAHoldThatNeverReturnsFailsInsteadOfHanging(t *testing.T) {
	registry := &fakeRegistry{}
	holds := newFakeHolds()
	holds.block = make(chan struct{})
	defer close(holds.block)
	manager := newManagerWith(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, &fakeFactory{}, holds.open)
	manager.holds.settle = 50 * time.Millisecond
	started := time.Now()
	err := manager.Reconcile(context.Background(), compositeProfile(), presentation.Plan{})
	if err == nil {
		t.Fatal("a node that never opens must be reported, not waited on")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("reconcile blocked for %s on a node that never opens", elapsed)
	}
}

func TestTwoCamerasCannotClaimOneNode(t *testing.T) {
	registry := &fakeRegistry{}
	holds := newFakeHolds()
	resolver := &listResolver{
		nodes: []string{"/dev/video0"},
		video: map[string]string{"uvc.cam0": "/dev/video0", "uvc.cam1": "/dev/video0"},
	}
	factory := &fakeFactory{}
	manager := newManagerWith(registry, resolver, factory, holds.open)
	defer manager.Suspend()
	err := manager.Reconcile(context.Background(), presentation.Profile{
		Functions: []presentation.Function{cameraFunction("cam0"), cameraFunction("cam1")},
	}, presentation.Plan{})
	if !errors.Is(err, ErrNodeIdentityAmbiguous) {
		t.Fatalf("err = %v, want the second claim on the node refused", err)
	}
	if opens, _ := holds.count("/dev/video0"); opens != 1 {
		t.Fatalf("node opened %d times, want 1", opens)
	}
	if _, streamed := factory.outputs["uvc.cam1"]; streamed {
		t.Fatal("both slots streamed to one node")
	}
}

// An output whose first run fails the way the gadget's own queue does when the
// host resets the bus: -ENODEV out of a QBUF, once, and the loop returns it.
// Every run after that behaves. The point of the test is that there is a run
// after that at all - the node stays held either way, so a slot that stops
// filling itself is a camera the host still sees and that has gone black.
type flakyOutput struct {
	factory *flakyFactory
	release chan struct{}
}

func (o *flakyOutput) Run(ctx context.Context, _ <-chan Packet, _ Fallback, demand func(sources.Demand), _ func(bool)) error {
	o.factory.mu.Lock()
	o.factory.runs++
	first := o.factory.runs == 1
	o.factory.mu.Unlock()
	if first {
		return errors.New("write UVC: errno 19")
	}
	demand(sources.Demand{Streaming: true, Width: 640, Height: 480, FPS: 30, Since: time.Now()})
	<-ctx.Done()
	return nil
}

func (o *flakyOutput) Close() error { return nil }

type flakyFactory struct {
	mu    sync.Mutex
	opens int
	runs  int
}

func (f *flakyFactory) Open(SlotSpec, string) (Output, error) {
	f.mu.Lock()
	f.opens++
	f.mu.Unlock()
	return &flakyOutput{factory: f}, nil
}

func (f *flakyFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

func TestManagerReopensASlotWhoseOutputFailed(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &flakyFactory{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	// The slot has to come back on its own: nothing else rebuilds a worker
	// short of an admin applying a profile.
	waitDemand(t, registry, "uvc.cam0")
	factory.mu.Lock()
	opens, runs := factory.opens, factory.runs
	factory.mu.Unlock()
	if opens < 2 || runs < 2 {
		t.Fatalf("opens = %d, runs = %d, want the node reopened and run again", opens, runs)
	}
}

// The kernel's video node, scripted. Every step the loop takes hands the test
// the frame it carried and then waits for the test to answer it, so the test
// plays the host: it raises the stream, drops it, and counts how often the loop
// raises the stream on its own.
type scriptedVideo struct {
	calls  chan Packet
	steps  chan videoStep
	starts chan Packet
	closed chan struct{}
}

func newScriptedVideo() *scriptedVideo {
	return &scriptedVideo{calls: make(chan Packet, 64), steps: make(chan videoStep), starts: make(chan Packet, 64), closed: make(chan struct{})}
}

func (v *scriptedVideo) step(current Packet, _ int, _ bool) (videoStep, error) {
	v.calls <- current
	select {
	case step := <-v.steps:
		return step, nil
	case <-v.closed:
		return videoStep{}, errVideoClosed
	}
}

func (v *scriptedVideo) start(current Packet) error {
	v.starts <- current
	return nil
}

// A stepped loop under test, with the host having committed 640x480 at 30 fps
// and started the stream. On return the loop is waiting inside a poll whose
// frame has already been read, so the test's next action - a frame sent, an
// edge answered - is what the poll after it carries. The returned function
// answers the waiting poll with no edge and returns the frame of the one that
// follows.
func startScriptedVideo(t *testing.T) (*scriptedVideo, chan Packet, chan sources.Demand, func() Packet) {
	t.Helper()
	video := newScriptedVideo()
	frames := make(chan Packet, 8)
	demands := make(chan sources.Demand, 8)
	spec := SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runVideo(ctx, frames, fallbackFor(spec), func(demand sources.Demand) { demands <- demand }, func(bool) {}, video)
	}()
	t.Cleanup(func() {
		close(video.closed)
		cancel()
		if err := <-done; err != nil {
			t.Errorf("runVideo returned %v", err)
		}
	})
	<-video.calls
	video.steps <- videoStep{edge: edgeStreamOn, width: 640, height: 480, fps: 30}
	if started := <-video.starts; !bytes.Equal(started.Data, blackFrame(t, spec, 640, 480)) {
		t.Fatal("the stream did not open on the black frame at the committed size")
	}
	if demand := <-demands; !demand.Streaming || demand.Width != 640 || demand.Height != 480 || demand.FPS != 30 {
		t.Fatalf("demand after STREAMON = %+v, want 640x480 at 30 fps", demand)
	}
	<-video.calls
	next := func() Packet {
		t.Helper()
		video.steps <- videoStep{width: 640, height: 480, fps: 30}
		select {
		case current := <-video.calls:
			return current
		case <-time.After(time.Second):
			t.Fatal("the loop stopped polling the node")
			return Packet{}
		}
	}
	return video, frames, demands, next
}

// countEncodes swaps the black-frame encoder for one that counts, for the life
// of the test.
func countEncodes(t *testing.T) *atomic.Int32 {
	t.Helper()
	var encodes atomic.Int32
	previous := encodeBlack
	encodeBlack = func(width, height int) ([]byte, error) {
		encodes.Add(1)
		return previous(width, height)
	}
	t.Cleanup(func() { encodeBlack = previous })
	return &encodes
}

func threeGeometries(id string) SlotSpec {
	return SlotSpec{ID: id, Kind: sources.KindCamera, Video: &presentation.VideoFunction{
		FunctionName: "Camera", StreamingMaxPacket: 768, StreamingInterval: 1,
		Formats: []presentation.VideoFormat{{Codec: "mjpeg", Frames: []presentation.VideoFrame{
			{Width: 640, Height: 480, Intervals: []uint32{333333}},
			{Width: 320, Height: 240, Intervals: []uint32{333333}},
			{Width: 160, Height: 120, Intervals: []uint32{333333}},
		}}},
	}}
}

// The stream opens on a black frame at whatever size the host committed, and
// the host may commit any of the declared sizes. Encoding that frame on the
// STREAMON is a visible part of the wait for the first picture on this board,
// so every declared geometry is encoded before the node is first polled, and
// the STREAMON only looks its frame up.
func TestBlackFramesAreReadyBeforeTheHostAsks(t *testing.T) {
	encodes := countEncodes(t)
	spec := threeGeometries("uvc.cam0")
	fallback := fallbackFor(spec)
	if err := warmFallback(spec, fallback); err != nil {
		t.Fatal(err)
	}
	if got := encodes.Load(); got != 3 {
		t.Fatalf("warmFallback encoded %d frames, want one per declared geometry (3)", got)
	}

	video := newScriptedVideo()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runVideo(ctx, make(chan Packet), fallback, func(sources.Demand) {}, func(bool) {}, video)
	}()
	t.Cleanup(func() {
		close(video.closed)
		cancel()
		if err := <-done; err != nil {
			t.Errorf("runVideo returned %v", err)
		}
	})
	// The host commits the smallest mode, not the first one, so a cache that
	// only held the first geometry would encode here.
	<-video.calls
	video.steps <- videoStep{edge: edgeStreamOn, width: 160, height: 120, fps: 30}
	started := <-video.starts
	if got := encodes.Load(); got != 3 {
		t.Fatalf("the STREAMON encoded %d more frames, want none", got-3)
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(started.Data))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 160 || config.Height != 120 {
		t.Fatalf("the stream opened on a %dx%d frame, want the committed 160x120", config.Width, config.Height)
	}
}

// An output that reports how many black frames had been encoded by the time it
// was run, and what asking for them costs after that.
type warmProbeFactory struct {
	encodes  *atomic.Int32
	atRun    chan int32
	afterAsk chan int32
}

func (f *warmProbeFactory) Open(SlotSpec, string) (Output, error) { return f, nil }

func (f *warmProbeFactory) OpenInput(SlotSpec, string) (Input, error) {
	return nil, errors.New("no capture in this factory")
}

func (f *warmProbeFactory) Run(ctx context.Context, _ <-chan Packet, fallback Fallback, _ func(sources.Demand), _ func(bool)) error {
	f.atRun <- f.encodes.Load()
	for _, geometry := range [][2]int{{640, 480}, {320, 240}, {160, 120}} {
		if _, err := fallback(geometry[0], geometry[1]); err != nil {
			return err
		}
	}
	f.afterAsk <- f.encodes.Load()
	<-ctx.Done()
	return nil
}

func (f *warmProbeFactory) Close() error { return nil }

// The worker does the encoding, before it runs its output, so a reconcile is
// not held up by it and the node is not polled until the frames are ready.
func TestWorkerEncodesEveryBlackFrameBeforeRunningItsOutput(t *testing.T) {
	encodes := countEncodes(t)
	factory := &warmProbeFactory{encodes: encodes, atRun: make(chan int32, 1), afterAsk: make(chan int32, 1)}
	manager := newTestManager(&fakeRegistry{}, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory)
	function := cameraFunction("cam0")
	function.Video.Formats = threeGeometries("uvc.cam0").Video.Formats
	if err := manager.Reconcile(context.Background(), presentation.Profile{Functions: []presentation.Function{function}}, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	select {
	case atRun := <-factory.atRun:
		if atRun != 3 {
			t.Fatalf("%d frames encoded when the output started, want all 3 declared geometries", atRun)
		}
	case <-time.After(time.Second):
		t.Fatal("the output never ran")
	}
	if afterAsk := <-factory.afterAsk; afterAsk != 3 {
		t.Fatalf("asking for the declared geometries encoded %d more frames, want none", afterAsk-3)
	}
}

func BenchmarkEncodeBlackFrame640x480(b *testing.B) {
	for b.Loop() {
		if _, err := encodeBlackFrame(640, 480); err != nil {
			b.Fatal(err)
		}
	}
}

func blackFrame(t *testing.T, spec SlotSpec, width, height int) []byte {
	t.Helper()
	packet, err := fallbackFor(spec)(width, height)
	if err != nil {
		t.Fatal(err)
	}
	return packet.Data
}

// A browser that releases, refreshes, or is taken over moves the queue to a new
// generation, and the loop used to answer that by taking the stream down and
// raising it again - a STREAMOFF and STREAMON on the node with the host still
// at its streaming alternate setting, which the host saw as the camera freezing.
// A source change may change which bytes go out and nothing else.
func TestASourceChangeNeverRestartsTheStream(t *testing.T) {
	video, frames, _, next := startScriptedVideo(t)

	first := jpegFrame(t, 640, 480)
	frames <- Packet{Sequence: 1, Generation: 1, Data: first}
	if got := next(); !bytes.Equal(got.Data, first) {
		t.Fatal("the first source's frame was not the next thing sent")
	}
	second := append(append([]byte(nil), first[:len(first)-2]...), 0x00, 0xff, 0xd9)
	frames <- Packet{Sequence: 1, Generation: 2, Data: second}
	if got := next(); !bytes.Equal(got.Data, second) {
		t.Fatal("the second source's frame was not the next thing sent")
	}
	frames <- Packet{Generation: 3, Reset: true}
	spec := SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}
	if got := next(); !bytes.Equal(got.Data, blackFrame(t, spec, 640, 480)) {
		t.Fatal("a source that let go was not replaced by the black frame at the committed size")
	}
	select {
	case <-video.starts:
		t.Fatal("a source change raised the stream again")
	default:
	}
}

// The host is the only thing that moves the stream. Its STREAMOFF drops the
// demand so the browser stops encoding; its STREAMON raises the stream again at
// the committed size and the demand comes back.
func TestAHostStreamOffThenStreamOnResumes(t *testing.T) {
	video, frames, demands, next := startScriptedVideo(t)
	frames <- Packet{Sequence: 1, Generation: 1, Data: jpegFrame(t, 640, 480)}
	next()

	video.steps <- videoStep{edge: edgeStreamOff, width: 640, height: 480, fps: 30}
	if demand := <-demands; demand.Streaming {
		t.Fatalf("demand after STREAMOFF = %+v, want none", demand)
	}
	<-video.calls
	video.steps <- videoStep{edge: edgeStreamOn, width: 640, height: 480, fps: 30}
	spec := SlotSpec{ID: "uvc.cam0", Kind: sources.KindCamera, Video: cameraFunction("cam0").Video}
	select {
	case started := <-video.starts:
		if !bytes.Equal(started.Data, blackFrame(t, spec, 640, 480)) {
			t.Fatal("the stream did not reopen on the black frame at the committed size")
		}
	case <-time.After(time.Second):
		t.Fatal("the host's STREAMON did not raise the stream")
	}
	if demand := <-demands; !demand.Streaming {
		t.Fatalf("demand after STREAMON = %+v, want streaming", demand)
	}
}

// A browser letting go of a slot reaches the queue and nothing past it: the
// node is not closed or reopened, the hold on it is untouched, the worker is
// the same one, and the host's demand stands.
func TestABrowserDetachNeverReachesTheNode(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	holds := newFakeHolds()
	manager := newManagerWith(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, factory, holds.open)
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")
	output := factory.outputs["uvc.cam0"]
	worker := manager.workers["uvc.cam0"]

	frame := sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: 1, Payload: jpegFrame(t, 640, 480)}
	if err := manager.Ingest(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	manager.Detach("uvc.cam0")

	deadline := time.After(time.Second)
	for reset := false; !reset; {
		select {
		case packet := <-output.packets:
			reset = packet.Reset
		case <-deadline:
			t.Fatal("the output never saw the source let go")
		}
	}
	if opens, closes := holds.count("/dev/video0"); opens != 1 || closes != 0 {
		t.Fatalf("detach left the node opened %d times and closed %d, want 1 and 0", opens, closes)
	}
	select {
	case <-output.closed:
		t.Fatal("detach closed the output")
	default:
	}
	if factory.outputs["uvc.cam0"] != output || manager.workers["uvc.cam0"] != worker {
		t.Fatal("detach rebuilt the worker or reopened the output")
	}
	registry.mu.Lock()
	streaming := registry.demands["uvc.cam0"].Streaming
	registry.mu.Unlock()
	if !streaming {
		t.Fatal("detach dropped the host's demand")
	}
}
