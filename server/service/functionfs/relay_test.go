package functionfs

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
)

type fakeControl struct {
	events chan Event
	writes chan []byte
	closed chan struct{}
	once   sync.Once
	mu     sync.Mutex
	reads  int
	stalls int
}

func newFakeControl() *fakeControl {
	return &fakeControl{events: make(chan Event, 8), writes: make(chan []byte, 8), closed: make(chan struct{})}
}

func (f *fakeControl) NextEvent() (Event, error) {
	select {
	case event := <-f.events:
		return event, nil
	case <-f.closed:
		return Event{}, ErrClosed
	}
}

func (f *fakeControl) ReadControl(data []byte) (int, error) {
	f.mu.Lock()
	f.reads++
	f.mu.Unlock()
	if len(data) == 0 {
		return 0, nil
	}
	return 0, io.EOF
}
func (f *fakeControl) WriteControl(data []byte) (int, error) {
	f.writes <- append([]byte(nil), data...)
	return len(data), nil
}
func (f *fakeControl) Stall(Setup) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stalls++
	return nil
}
func (f *fakeControl) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

type fakeEndpoint struct {
	closed chan struct{}
	once   sync.Once
}

func newFakeEndpoint() *fakeEndpoint { return &fakeEndpoint{closed: make(chan struct{})} }
func (f *fakeEndpoint) Read([]byte) (int, error) {
	<-f.closed
	return 0, ErrClosed
}
func (f *fakeEndpoint) Write(data []byte) (int, error) { return len(data), nil }
func (f *fakeEndpoint) Stall() error                   { return nil }
func (f *fakeEndpoint) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

type fakeDevice struct {
	mu         sync.Mutex
	setup      Setup
	resets     int
	closed     bool
	report     []byte
	controlErr error
	controls   int
	clears     []uint8
	transfer   chan struct{}
	transfers  int
	signal     sync.Once
}

func (f *fakeDevice) Control(_ context.Context, setup Setup, _ []byte) ([]byte, error) {
	f.mu.Lock()
	f.setup = setup
	f.controls++
	f.mu.Unlock()
	if f.controlErr != nil {
		return nil, f.controlErr
	}
	return append([]byte(nil), f.report...), nil
}
func (f *fakeDevice) Transfer(ctx context.Context, _ Endpoint, _ []byte) ([]byte, error) {
	f.mu.Lock()
	f.transfers++
	f.mu.Unlock()
	if f.transfer != nil {
		f.signal.Do(func() { close(f.transfer) })
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRelaySuspendResumesWithoutReset(t *testing.T) {
	control := newFakeControl()
	device := &fakeDevice{transfer: make(chan struct{})}
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- relay.Run(context.Background()) }()
	control.events <- Event{Type: EventEnable}
	waitTransfers(t, device, 1)
	control.events <- Event{Type: EventSuspend}
	control.events <- Event{Type: EventResume}
	waitTransfers(t, device, 2)
	device.mu.Lock()
	resets := device.resets
	device.mu.Unlock()
	if resets != 0 {
		t.Fatalf("suspend resets = %d", resets)
	}
	_ = relay.Close()
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("Run returned %v", err)
	}
}

func waitTransfers(t *testing.T, device *fakeDevice, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		device.mu.Lock()
		got := device.transfers
		device.mu.Unlock()
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("transfers = %d, want %d", got, want)
		}
		time.Sleep(time.Millisecond)
	}
}
func (f *fakeDevice) ClearHalt(endpoint uint8) error {
	f.mu.Lock()
	f.clears = append(f.clears, endpoint)
	f.mu.Unlock()
	return nil
}
func (f *fakeDevice) Reset() error {
	f.mu.Lock()
	f.resets++
	f.mu.Unlock()
	return nil
}
func (f *fakeDevice) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func relayImage() Image {
	return Image{
		Interfaces: map[uint8]uint8{4: 0}, Endpoints: map[uint8]uint8{0x83: 0x81},
		Function: presentation.FunctionFS{Interfaces: 1, Endpoints: []presentation.FunctionFSEndpoint{
			{SourceAddress: 0x83, Address: 0x81, Transfer: presentation.EndpointInterrupt, MaxPacket: 8, Interval: 10},
		}},
	}
}

func TestRelayReversesControlRecipient(t *testing.T) {
	image := relayImage()
	control := newFakeControl()
	device := &fakeDevice{report: []byte{1, 2, 3}}
	relay, err := NewRelay(image, control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	setup := Setup{RequestType: 0x82, Request: 6, Value: 1, Index: 0x81, Length: 3}
	if err := relay.handleSetup(context.Background(), setup); err != nil {
		t.Fatal(err)
	}
	device.mu.Lock()
	got := device.setup.Index
	device.mu.Unlock()
	if got != 0x83 {
		t.Fatalf("source endpoint = 0x%02x, want 0x83", got)
	}
	if data := <-control.writes; string(data) != string(device.report) {
		t.Fatalf("control response = %x", data)
	}
}

func TestRelayAcknowledgesZeroLengthControl(t *testing.T) {
	control := newFakeControl()
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, &fakeDevice{})
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.handleSetup(context.Background(), Setup{RequestType: 0x01, Request: 9}); err != nil {
		t.Fatal(err)
	}
	control.mu.Lock()
	reads := control.reads
	control.mu.Unlock()
	if reads != 1 {
		t.Fatalf("zero-length OUT reads = %d", reads)
	}
	if err := relay.handleSetup(context.Background(), Setup{RequestType: 0x81, Request: 1}); err != nil {
		t.Fatal(err)
	}
	if data := <-control.writes; len(data) != 0 {
		t.Fatalf("zero-length IN wrote %x", data)
	}
}

func TestRelayClearsHaltOnce(t *testing.T) {
	control := newFakeControl()
	device := &fakeDevice{}
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	setup := Setup{RequestType: 0x02, Request: 1, Index: 0x81}
	if err := relay.handleSetup(context.Background(), setup); err != nil {
		t.Fatal(err)
	}
	device.mu.Lock()
	defer device.mu.Unlock()
	if len(device.clears) != 1 || device.clears[0] != 0x83 || device.controls != 0 {
		t.Fatalf("clears = %x, controls = %d", device.clears, device.controls)
	}
}

func TestRelayPropagatesControlStall(t *testing.T) {
	control := newFakeControl()
	device := &fakeDevice{controlErr: ErrStall}
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.handleSetup(context.Background(), Setup{RequestType: 0x81, Request: 1}); err != nil {
		t.Fatal(err)
	}
	control.mu.Lock()
	stalls := control.stalls
	control.mu.Unlock()
	if stalls != 1 {
		t.Fatalf("stalls = %d", stalls)
	}
}

func TestRelayDisableResetsAndRestarts(t *testing.T) {
	control := newFakeControl()
	device := &fakeDevice{transfer: make(chan struct{})}
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- relay.Run(context.Background()) }()
	control.events <- Event{Type: EventEnable}
	select {
	case <-device.transfer:
	case <-time.After(time.Second):
		t.Fatal("transfer did not start")
	}
	control.events <- Event{Type: EventDisable}
	deadline := time.Now().Add(time.Second)
	for {
		device.mu.Lock()
		resets := device.resets
		device.mu.Unlock()
		if resets == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("disable did not reset source")
		}
		time.Sleep(time.Millisecond)
	}
	_ = relay.Close()
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("Run returned %v", err)
	}
}

func TestRelayConcurrentClose(t *testing.T) {
	for range 32 {
		control := newFakeControl()
		relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, &fakeDevice{})
		if err != nil {
			t.Fatal(err)
		}
		var group sync.WaitGroup
		for range 8 {
			group.Add(1)
			go func() {
				defer group.Done()
				_ = relay.Close()
			}()
		}
		group.Wait()
	}
}

func TestDecodeEventBounds(t *testing.T) {
	if _, err := DecodeEvent(make([]byte, 11)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("short event returned %v", err)
	}
	event := make([]byte, 12)
	event[8] = 7
	if _, err := DecodeEvent(event); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unknown event returned %v", err)
	}
}
