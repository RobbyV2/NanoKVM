package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

var testSlots = []Slot{
	{ID: "uvc.cam0", Kind: KindCamera, Label: "NanoKVM Camera 1"},
	{ID: "uac2.mic0", Kind: KindMicrophone, Label: "NanoKVM Microphone"},
}

func TestConcurrentClaimsHaveOneWinner(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	const contenders = 64
	type contender struct {
		actor  Actor
		source Source
	}
	clients := make([]contender, contenders)
	for i := range clients {
		clients[i].actor = Actor{Username: fmt.Sprintf("user%d", i)}
		clients[i].source = mustSource(t, registry, clients[i].actor, fmt.Sprintf("Phone %d", i), KindCamera)
	}

	start := make(chan struct{})
	results := make(chan struct {
		index int
		err   error
	}, contenders)
	var workers sync.WaitGroup
	for i := range clients {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			_, err := registry.Claim(clients[index].actor, clients[index].source.ID, "stream", "uvc.cam0")
			results <- struct {
				index int
				err   error
			}{index: index, err: err}
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)

	winner := -1
	var refusals []struct {
		index int
		err   error
	}
	for result := range results {
		if result.err == nil {
			if winner != -1 {
				t.Fatalf("multiple winners: %d and %d", winner, result.index)
			}
			winner = result.index
		} else {
			refusals = append(refusals, result)
		}
	}
	if winner == -1 || len(refusals) != contenders-1 {
		t.Fatalf("winner=%d refusals=%d", winner, len(refusals))
	}
	for _, refusal := range refusals {
		var occupied *OccupiedError
		if !errors.As(refusal.err, &occupied) {
			t.Fatalf("claim %d: %v", refusal.index, refusal.err)
		}
		if occupied.Owner != clients[winner].actor.Username || occupied.SourceLabel != clients[winner].source.Label {
			t.Fatalf("refusal names %q/%q, want %q/%q", occupied.Owner, occupied.SourceLabel, clients[winner].actor.Username, clients[winner].source.Label)
		}
	}
}

func TestSourceOffersCameraAndMicrophoneIndependently(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	actor := Actor{Username: "alice"}
	source, err := registry.RegisterSource(actor, Hello{Label: "Pixel", Streams: []Stream{
		{ID: "back", Kind: KindCamera, Label: "Back camera"},
		{ID: "mic", Kind: KindMicrophone, Label: "Microphone"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(actor, source.ID, "back", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(actor, source.ID, "mic", "uac2.mic0"); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Bindings) != 2 {
		t.Fatalf("bindings=%d", len(snapshot.Bindings))
	}
}

func TestDisconnectResumeAndExpiry(t *testing.T) {
	clock := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	scheduler := &manualScheduler{}
	random := bytes.NewReader(bytes.Join([][]byte{
		bytes.Repeat([]byte{0x01}, 12),
		bytes.Repeat([]byte{0x5a}, 32),
		bytes.Repeat([]byte{0x02}, 12),
		bytes.Repeat([]byte{0x03}, 12),
	}, nil))
	registry := mustRegistry(t, testSlots, RegistryOptions{
		Now:        func() time.Time { return clock },
		Random:     random,
		Schedule:   scheduler.Schedule,
		LeaseGrace: time.Minute,
	})
	alice := Actor{Username: "alice"}
	first := mustSource(t, registry, alice, "Pixel", KindCamera)
	claim, err := registry.Claim(alice, first.ID, "stream", "uvc.cam0")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.SetDemand("uvc.cam0", Demand{Streaming: true, Width: 1280, Height: 720, FPS: 30}); err != nil {
		t.Fatal(err)
	}
	if err := registry.SetBindingState("uvc.cam0", StateStreaming); err != nil {
		t.Fatal(err)
	}
	registry.DisconnectSource(first.ID)
	sink := sinkByID(t, registry.Snapshot(), "uvc.cam0")
	if sink.Binding.State != StateOrphaned || sink.Output != OutputBlack {
		t.Fatalf("orphaned sink=%+v", sink)
	}

	bob := Actor{Username: "bob"}
	bobSource := mustSource(t, registry, bob, "Laptop", KindCamera)
	if _, err := registry.Resume(bob, bobSource.ID, "stream", "uvc.cam0", claim.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong owner resume: %v", err)
	}
	second := mustSource(t, registry, alice, "Pixel reloaded", KindCamera)
	if _, err := registry.Resume(alice, second.ID, "stream", "uvc.cam0", claim.Token+"x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong token resume: %v", err)
	}
	resumed, err := registry.Resume(alice, second.ID, "stream", "uvc.cam0", claim.Token)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != StateClaimed || !resumed.ExpiresAt.IsZero() || resumed.SourceID != second.ID {
		t.Fatalf("resumed=%+v", resumed)
	}
	scheduler.Run()
	if len(registry.Snapshot().Bindings) != 1 {
		t.Fatal("cancelled expiry removed resumed binding")
	}

	registry.DisconnectSource(second.ID)
	clock = clock.Add(61 * time.Second)
	scheduler.Run()
	if len(registry.Snapshot().Bindings) != 0 {
		t.Fatal("expired binding survived")
	}
}

func TestFallbackOutputTracksDemandAndBinding(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	for _, test := range []struct {
		id   string
		want OutputState
	}{
		{id: "uvc.cam0", want: OutputBlack},
		{id: "uac2.mic0", want: OutputSilence},
	} {
		if err := registry.SetDemand(test.id, Demand{Streaming: true}); err != nil {
			t.Fatal(err)
		}
		if got := sinkByID(t, registry.Snapshot(), test.id).Output; got != test.want {
			t.Fatalf("%s output=%s want=%s", test.id, got, test.want)
		}
	}
	actor := Actor{Username: "alice"}
	source := mustSource(t, registry, actor, "Phone", KindCamera)
	if _, err := registry.Claim(actor, source.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}
	if got := sinkByID(t, registry.Snapshot(), "uvc.cam0").Output; got != OutputBlack {
		t.Fatalf("claimed output=%s", got)
	}
	if err := registry.SetBindingState("uvc.cam0", StateStreaming); err != nil {
		t.Fatal(err)
	}
	if got := sinkByID(t, registry.Snapshot(), "uvc.cam0").Output; got != OutputSource {
		t.Fatalf("streaming output=%s", got)
	}
}

func TestSyncSlotsTerminatesChangedBindings(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	actor := Actor{Username: "alice"}
	camera := mustSource(t, registry, actor, "Camera", KindCamera)
	microphone := mustSource(t, registry, actor, "Mic", KindMicrophone)
	if _, err := registry.Claim(actor, camera.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(actor, microphone.ID, "stream", "uac2.mic0"); err != nil {
		t.Fatal(err)
	}
	events, cancel := registry.Subscribe()
	defer cancel()
	<-events

	if err := registry.SyncSlots([]Slot{
		{ID: "uac2.mic0", Kind: KindMicrophone, Label: "Changed microphone"},
	}); err != nil {
		t.Fatal(err)
	}
	reasons := make(map[TerminationReason]bool)
	for i := 0; i < 3; i++ {
		event := <-events
		if event.Type == "binding_removed" {
			reasons[event.Reason] = true
		}
	}
	if !reasons[ReasonSlotRemoved] || !reasons[ReasonSlotChanged] {
		t.Fatalf("reasons=%v", reasons)
	}
	if len(registry.Snapshot().Bindings) != 0 {
		t.Fatal("slot replacement retained bindings")
	}
}

func TestSlowSubscriberIsClosed(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{SubscriberBuffer: 1})
	events, cancel := registry.Subscribe()
	defer cancel()
	actor := Actor{Username: "alice"}
	mustSource(t, registry, actor, "Phone", KindCamera)
	if event, open := <-events; !open || event.Type != "snapshot" {
		t.Fatalf("first event=%+v open=%v", event, open)
	}
	if _, open := <-events; open {
		t.Fatal("slow subscriber remained open")
	}
}

func TestTokensNeverEnterSnapshotsOrEvents(t *testing.T) {
	random := bytes.NewReader(bytes.Repeat([]byte{0xa5}, 256))
	registry := mustRegistry(t, testSlots, RegistryOptions{Random: random})
	events, cancel := registry.Subscribe()
	defer cancel()
	<-events
	actor := Actor{Username: "alice"}
	source := mustSource(t, registry, actor, "Phone", KindCamera)
	<-events
	claim, err := registry.Claim(actor, source.ID, "stream", "uvc.cam0")
	if err != nil {
		t.Fatal(err)
	}
	event := <-events
	data, err := json.Marshal(struct {
		Snapshot Snapshot `json:"snapshot"`
		Event    Event    `json:"event"`
	}{Snapshot: registry.Snapshot(), Event: event})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), claim.Token) {
		t.Fatal("lease token leaked")
	}
}

func TestWebUSBSinkRequiresAdmin(t *testing.T) {
	registry := mustRegistry(t, nil, RegistryOptions{})
	user := Actor{Username: "alice"}
	source, err := registry.RegisterSource(user, Hello{Label: "Browser", Streams: []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(user, source.ID, "usb", HybridSinkID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin claim returned %v", err)
	}
	admin := Actor{Username: "alice", Admin: true}
	if _, err := registry.Claim(admin, source.ID, "usb", HybridSinkID); err != nil {
		t.Fatal(err)
	}
}

func TestWebUSBStreamIDsMustBeUnique(t *testing.T) {
	offer := Stream{ID: "usb", Kind: KindUSBDevice, Label: "Adapter", USB: &USBOffer{
		Profile: "webusb-adapter", Configuration: 1, Interfaces: []uint8{0},
	}}
	if _, err := validateHello(Hello{Label: "Browser", Streams: []Stream{offer, offer}}); err == nil {
		t.Fatal("validateHello() accepted duplicate WebUSB stream IDs")
	}
}

func TestRejectsHostileMetadata(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	actor := Actor{Username: "alice"}
	tests := []Hello{
		{Label: "phone\nforged", Streams: []Stream{{ID: "stream", Kind: KindCamera, Label: "Camera"}}},
		{Label: "Phone", Streams: []Stream{{ID: "../stream", Kind: KindCamera, Label: "Camera"}}},
		{Label: "Phone", Streams: []Stream{{ID: "stream", Kind: "unknown", Label: "Camera"}}},
		{Label: "Phone", Streams: []Stream{{ID: "stream", Kind: KindCamera, Label: "Camera", Formats: []Format{{Codec: "mjpeg;rm", Width: 640, Height: 480, FPS: 30}}}}},
		{Label: "Phone", Streams: []Stream{{ID: "stream", Kind: KindMicrophone, Label: "Mic", Formats: []Format{{Codec: "pcm", Width: 1, SampleRate: 48000, Channels: 1}}}}},
	}
	for i, hello := range tests {
		if _, err := registry.RegisterSource(actor, hello); !errors.Is(err, ErrInvalidMessage) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

type manualScheduler struct {
	mu   sync.Mutex
	jobs []*manualJob
}

type manualJob struct {
	callback func()
	canceled bool
}

func (s *manualScheduler) Schedule(_ time.Duration, callback func()) func() {
	job := &manualJob{callback: callback}
	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		job.canceled = true
		s.mu.Unlock()
	}
}

func (s *manualScheduler) Run() {
	s.mu.Lock()
	jobs := s.jobs
	s.jobs = nil
	s.mu.Unlock()
	for _, job := range jobs {
		s.mu.Lock()
		canceled := job.canceled
		s.mu.Unlock()
		if !canceled {
			job.callback()
		}
	}
}

func mustRegistry(t *testing.T, slots []Slot, options RegistryOptions) *Registry {
	t.Helper()
	registry, err := NewRegistry(slots, options)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func mustSource(t *testing.T, registry *Registry, actor Actor, label string, kind Kind) Source {
	t.Helper()
	source, err := registry.RegisterSource(actor, Hello{Label: label, Streams: []Stream{{ID: "stream", Kind: kind, Label: label}}})
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func sinkByID(t *testing.T, snapshot Snapshot, id string) Sink {
	t.Helper()
	for _, sink := range snapshot.Sinks {
		if sink.ID == id {
			return sink
		}
	}
	t.Fatalf("sink %s not found", id)
	return Sink{}
}

func TestReleaseRefusesANonOwner(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	alice := Actor{Username: "alice"}
	source := mustSource(t, registry, alice, "Pixel", KindCamera)
	if _, err := registry.Claim(alice, source.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}

	bob := Actor{Username: "bob"}
	if err := registry.Release(bob, "uvc.cam0", ReasonReleased); !errors.Is(err, ErrForbidden) {
		t.Fatalf("stranger release: err = %v, want %v", err, ErrForbidden)
	}
	if len(registry.Snapshot().Bindings) != 1 {
		t.Fatal("refused release still dropped the binding")
	}

	if err := registry.Release(Actor{Username: "root", Admin: true}, "uvc.cam0", ReasonReleased); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectAllRefusesANonAdmin(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	alice := Actor{Username: "alice"}
	source := mustSource(t, registry, alice, "Pixel", KindCamera)
	if _, err := registry.Claim(alice, source.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}

	// The owner of the only binding is still not allowed to sweep the table.
	if err := registry.DisconnectAll(alice); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin disconnect: err = %v, want %v", err, ErrForbidden)
	}
	if len(registry.Snapshot().Bindings) != 1 {
		t.Fatal("refused disconnect still dropped the binding")
	}
}

// Resume carries the same USB restriction as Claim because a lease outlives the
// role that was checked when it was taken: the grace window is where an admin
// who has since been demoted would otherwise walk back onto the hybrid sink.
func TestResumeRefusesANonAdminOnTheUSBSink(t *testing.T) {
	registry := mustRegistry(t, nil, RegistryOptions{})
	admin := Actor{Username: "alice", Admin: true}
	source, err := registry.RegisterSource(admin, Hello{Label: "Browser", Streams: []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	claim, err := registry.Claim(admin, source.ID, "usb", HybridSinkID)
	if err != nil {
		t.Fatal(err)
	}
	registry.DisconnectSource(source.ID)

	demoted := Actor{Username: "alice"}
	next, err := registry.RegisterSource(demoted, Hello{Label: "Browser", Streams: []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resume(demoted, next.ID, "usb", HybridSinkID, claim.Token); !errors.Is(err, ErrForbidden) {
		t.Fatalf("demoted resume: err = %v, want %v", err, ErrForbidden)
	}
	if sinkByID(t, registry.Snapshot(), HybridSinkID).Binding.State != StateOrphaned {
		t.Fatal("refused resume still reattached the lease")
	}

	if _, err := registry.Resume(admin, next.ID, "usb", HybridSinkID, claim.Token); err != nil {
		t.Fatal(err)
	}
}

// The host re-enumerates when the interface string changes, so a binding that
// survived the rename would be streaming into descriptors its browser never saw.
func TestSyncSlotsTerminatesOnAHostNameChangeAlone(t *testing.T) {
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	actor := Actor{Username: "alice"}
	camera := mustSource(t, registry, actor, "Camera", KindCamera)
	if _, err := registry.Claim(actor, camera.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}
	events, cancel := registry.Subscribe()
	defer cancel()
	<-events

	slots := append([]Slot(nil), testSlots...)
	for i := range slots {
		if slots[i].ID == "uvc.cam0" {
			slots[i].HostName = "Desk Camera"
		}
	}
	if err := registry.SyncSlots(slots); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		event := <-events
		if event.Type == "binding_removed" {
			if event.Reason != ReasonSlotChanged {
				t.Fatalf("reason = %s, want %s", event.Reason, ReasonSlotChanged)
			}
			return
		}
	}
	t.Fatal("renaming the host-visible interface string left the binding up")
}
