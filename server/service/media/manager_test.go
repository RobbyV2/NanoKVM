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
	payload := jpegFrame(t, 640, 480)
	for sequence := uint32(1); sequence <= 3; sequence++ {
		err := manager.Ingest(context.Background(), sources.MediaFrame{SourceID: "source", StreamID: "front", SinkID: "uvc.cam0", Kind: sources.MediaKindMJPEG, Sequence: sequence, Payload: payload})
		if sequence < 3 && err != nil {
			t.Fatal(err)
		}
		if sequence == 3 && !errors.Is(err, ErrFrameRate) {
			t.Fatalf("third frame err = %v, want ErrFrameRate", err)
		}
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
