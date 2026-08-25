package media

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
)

func speakerFunction(instance string) presentation.Function {
	return presentation.Function{Kind: presentation.FunctionUAC2, Instance: instance, Audio: &presentation.AudioFunction{
		FunctionName: "Speaker " + instance, PChannelMask: 0, PSampleRate: 48000, PSampleSize: 2,
		CChannelMask: 1, CSampleRate: 48000, CSampleSize: 2, RequestNumber: 4,
	}}
}

func speakerProfile() presentation.Profile {
	return presentation.Profile{Functions: []presentation.Function{microphoneFunction("mic0"), speakerFunction("spk0")}}
}

// The channel masks are the only direction marker the kernel has, so they are
// the only one the slot table may use.
func TestSpeakerSlotComesFromTheOUTChannelMask(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.mic0": "hw:1,0", "uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	if err := manager.Reconcile(context.Background(), speakerProfile(), presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	registry.mu.Lock()
	kinds := map[string]sources.Kind{}
	for _, slot := range registry.slots {
		kinds[slot.ID] = slot.Kind
	}
	registry.mu.Unlock()
	if kinds["uac2.mic0"] != sources.KindMicrophone || kinds["uac2.spk0"] != sources.KindSpeaker {
		t.Fatalf("slot kinds = %v", kinds)
	}
	if factory.input("uac2.spk0") == nil {
		t.Fatal("a speaker must be opened for capture, not playback")
	}
	if factory.input("uac2.mic0") != nil {
		t.Fatal("a microphone must not be opened for capture")
	}
}

func TestCapturedAudioReachesTheAttachedBrowser(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{speakerFunction("spk0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	frames := make(chan sources.MediaFrame, 4)
	detach, err := manager.Attach("uac2.spk0", func(frame sources.MediaFrame) error {
		frames <- frame
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	input := factory.input("uac2.spk0")
	if input == nil {
		t.Fatal("no capture input was opened")
	}
	var emit func(Packet)
	select {
	case emit = <-input.emit:
	case <-time.After(time.Second):
		t.Fatal("capture never started")
	}

	period := make([]byte, pcmPacketBytes)
	period[7] = 0x42
	emit(Packet{Data: period})
	emit(Packet{Data: period})

	for want := range uint32(2) {
		select {
		case frame := <-frames:
			if frame.SinkID != "uac2.spk0" || frame.Kind != sources.MediaKindPCMS16LE {
				t.Fatalf("frame = %+v", frame)
			}
			if frame.Sequence != want {
				t.Fatalf("sequence = %d, want %d", frame.Sequence, want)
			}
			if len(frame.Payload) != pcmPacketBytes || frame.Payload[7] != 0x42 {
				t.Fatalf("payload = %d bytes", len(frame.Payload))
			}
			if frame.TimestampUS == 0 {
				t.Fatal("frame carries no timestamp")
			}
		case <-time.After(time.Second):
			t.Fatalf("frame %d never arrived", want)
		}
	}

	detach()
	emit(Packet{Data: period})
	select {
	case frame := <-frames:
		t.Fatalf("a detached browser still received %+v", frame)
	case <-time.After(50 * time.Millisecond):
	}
}

// The gadget's reader must never be the thing that waits. A browser that stops
// draining loses packets; the capture loop keeps its cadence.
func TestASlowBrowserNeverStallsCapture(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{speakerFunction("spk0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	release := make(chan struct{})
	delivered := make(chan struct{}, 1024)
	detach, err := manager.Attach("uac2.spk0", func(sources.MediaFrame) error {
		<-release
		delivered <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer detach()

	input := factory.input("uac2.spk0")
	var emit func(Packet)
	select {
	case emit = <-input.emit:
	case <-time.After(time.Second):
		t.Fatal("capture never started")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 500 {
			emit(Packet{Data: make([]byte, pcmPacketBytes)})
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("capture blocked on a browser that was not reading")
	}
	close(release)
}

func TestAttachRefusesASinkTheBrowserFills(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.mic0": "hw:1,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{microphoneFunction("mic0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	if _, err := manager.Attach("uac2.mic0", func(sources.MediaFrame) error { return nil }); !errors.Is(err, ErrUnsupportedFrame) {
		t.Fatalf("Attach(microphone) = %v, want a refusal", err)
	}
	if _, err := manager.Attach("uac2.spk9", func(sources.MediaFrame) error { return nil }); !errors.Is(err, ErrSinkUnavailable) {
		t.Fatalf("Attach(unknown) = %v, want a refusal", err)
	}
}

func TestSpeakerSinkAcceptsNoBrowserFrames(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{speakerFunction("spk0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	err := manager.Ingest(context.Background(), sources.MediaFrame{
		SinkID: "uac2.spk0", StreamID: "s", Kind: sources.MediaKindPCMS16LE,
		Payload: make([]byte, pcmPacketBytes),
	})
	if !errors.Is(err, ErrUnsupportedFrame) || !strings.Contains(err.Error(), "accepts none") {
		t.Fatalf("Ingest(speaker) = %v, want a refusal", err)
	}
}

// Every apply rebuilds every worker, whether or not the slot changed. A browser
// whose binding survives that must keep hearing the target, not go quiet behind
// a binding that still reads as live.
func TestAReconcileDoesNotOrphanAListeningBrowser(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{speakerFunction("spk0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	frames := make(chan sources.MediaFrame, 4)
	if _, err := manager.Attach("uac2.spk0", func(frame sources.MediaFrame) error {
		frames <- frame
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	first := factory.input("uac2.spk0")
	select {
	case <-first.emit:
	case <-time.After(time.Second):
		t.Fatal("capture never started")
	}

	// A second reconcile of the same profile, which is what any unrelated
	// setting change produces.
	factory.mu.Lock()
	factory.inputs = nil
	factory.mu.Unlock()
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	second := factory.input("uac2.spk0")
	if second == nil {
		t.Fatal("the reconcile opened no capture")
	}
	var emit func(Packet)
	select {
	case emit = <-second.emit:
	case <-time.After(time.Second):
		t.Fatal("the rebuilt capture never started")
	}
	emit(Packet{Data: make([]byte, pcmPacketBytes)})
	select {
	case frame := <-frames:
		if frame.SinkID != "uac2.spk0" {
			t.Fatalf("frame = %+v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("the listening browser was orphaned by the reconcile")
	}
}

func TestARemovedSpeakerStopsItsListener(t *testing.T) {
	registry := &fakeRegistry{}
	factory := &fakeFactory{}
	resolver := fakeResolver{nodes: map[string]string{"uac2.spk0": "hw:2,0"}}
	manager := newTestManager(registry, resolver, factory)

	profile := presentation.Profile{Functions: []presentation.Function{speakerFunction("spk0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()

	stopped := make(chan struct{})
	if _, err := manager.Attach("uac2.spk0", func(sources.MediaFrame) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(stopped)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			manager.mu.RLock()
			_, present := manager.listeners["uac2.spk0"]
			manager.mu.RUnlock()
			if !present {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	if err := manager.Reconcile(context.Background(), presentation.Profile{}, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	<-stopped
	manager.mu.RLock()
	_, present := manager.listeners["uac2.spk0"]
	manager.mu.RUnlock()
	if present {
		t.Fatal("a removed speaker kept its listener")
	}
}
