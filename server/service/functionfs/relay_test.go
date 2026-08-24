package functionfs

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
)

type fakeControl struct {
	events  chan Event
	writes  chan []byte
	closed  chan struct{}
	once    sync.Once
	mu      sync.Mutex
	reads   int
	stalls  int
	payload []byte
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
	defer f.mu.Unlock()
	f.reads++
	if len(data) == 0 {
		return 0, nil
	}
	if len(f.payload) == 0 {
		return 0, io.EOF
	}
	n := copy(data, f.payload)
	f.payload = f.payload[n:]
	return n, nil
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
	reads  chan []byte
	once   sync.Once
}

func newFakeEndpoint() *fakeEndpoint {
	return &fakeEndpoint{closed: make(chan struct{}), reads: make(chan []byte, 1)}
}
func (f *fakeEndpoint) Read(data []byte) (int, error) {
	select {
	case payload := <-f.reads:
		return copy(data, payload), nil
	case <-f.closed:
		return 0, ErrClosed
	}
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

func TestRelayRearmsAnInputTransferThatTimesOut(t *testing.T) {
	previous := transferTimeout
	transferTimeout = 20 * time.Millisecond
	defer func() { transferTimeout = previous }()

	control := newFakeControl()
	device := &fakeDevice{}
	relay, err := NewRelay(relayImage(), control, map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, device)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- relay.Run(context.Background()) }()
	control.events <- Event{Type: EventEnable}
	waitTransfers(t, device, 3)

	select {
	case err := <-done:
		t.Fatalf("an idle input endpoint ended the relay: %v", err)
	default:
	}
	_ = relay.Close()
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("Run returned %v", err)
	}
}

func TestRelayFailsAnOutputTransferThatTimesOut(t *testing.T) {
	previous := transferTimeout
	transferTimeout = 20 * time.Millisecond
	defer func() { transferTimeout = previous }()

	control := newFakeControl()
	endpoint := newFakeEndpoint()
	relay, err := NewRelay(outputRelayImage(), control, map[uint8]DataEndpoint{0x01: endpoint}, &fakeDevice{})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- relay.Run(context.Background()) }()
	control.events <- Event{Type: EventEnable}
	endpoint.reads <- []byte{1, 2, 3, 4}

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a timed out output transfer did not end the relay")
	}
}

func outputRelayImage() Image {
	return Image{
		Interfaces: map[uint8]uint8{4: 0}, Endpoints: map[uint8]uint8{0x03: 0x01},
		Function: presentation.FunctionFS{Interfaces: 1, Endpoints: []presentation.FunctionFSEndpoint{
			{SourceAddress: 0x03, Address: 0x01, Transfer: presentation.EndpointBulk, MaxPacket: 512},
		}},
	}
}

type fakeStream struct {
	*fakeEndpoint
	mu       sync.Mutex
	payloads []int
	stops    int
	err      error
}

func newFakeStream() *fakeStream { return &fakeStream{fakeEndpoint: newFakeEndpoint()} }

func (f *fakeStream) Start(payload int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads = append(f.payloads, payload)
	return f.err
}

func (f *fakeStream) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	return nil
}

func streamRelayImage() Image {
	return Image{
		Interfaces: map[uint8]uint8{1: 0}, Endpoints: map[uint8]uint8{0x83: 0x81},
		Alternates: map[uint8]uint8{1: 3}, EndpointOwners: map[uint8]uint8{0x81: 1},
		Function: presentation.FunctionFS{Interfaces: 1, Endpoints: []presentation.FunctionFSEndpoint{
			{SourceAddress: 0x83, Address: 0x81, Transfer: presentation.EndpointIsochronous, MaxPacket: 768, Interval: 1},
		}},
	}
}

func videoCommitPayload(size uint32) []byte {
	data := make([]byte, 26)
	data[2], data[3] = 1, 1
	binary.LittleEndian.PutUint32(data[22:26], size)
	return data
}

func TestRelayStartsTheStreamAtTheSizeTheCommitNamed(t *testing.T) {
	control := newFakeControl()
	control.payload = videoCommitPayload(768)
	stream := newFakeStream()
	device := &fakeDevice{}
	relay, err := NewRelay(streamRelayImage(), control, map[uint8]DataEndpoint{0x81: stream}, device)
	if err != nil {
		t.Fatal(err)
	}
	setup := Setup{RequestType: 0x21, Request: 0x01, Value: 0x0200, Index: 0, Length: 26}
	if err := relay.handleSetup(context.Background(), setup); err != nil {
		t.Fatal(err)
	}
	device.mu.Lock()
	forwarded := device.setup
	device.mu.Unlock()
	if forwarded.Index != 1 || forwarded.Value != 0x0200 {
		t.Fatalf("commit forwarded to the source as %+v, want interface 1 and VS_COMMIT_CONTROL", forwarded)
	}
	stream.mu.Lock()
	started := len(stream.payloads)
	stream.mu.Unlock()
	if started != 0 {
		t.Fatal("the commit started the stream on its own; only the alternate edge may")
	}
	if err := relay.startStreams(); err != nil {
		t.Fatal(err)
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if len(stream.payloads) != 1 || stream.payloads[0] != 768 {
		t.Fatalf("stream starts = %v, want one at dwMaxPayloadTransferSize 768", stream.payloads)
	}
}

func TestRelayLeavesTheStreamAloneOnAProbe(t *testing.T) {
	control := newFakeControl()
	control.payload = videoCommitPayload(768)
	stream := newFakeStream()
	relay, err := NewRelay(streamRelayImage(), control, map[uint8]DataEndpoint{0x81: stream}, &fakeDevice{})
	if err != nil {
		t.Fatal(err)
	}
	setup := Setup{RequestType: 0x21, Request: 0x01, Value: 0x0100, Index: 0, Length: 26}
	if err := relay.handleSetup(context.Background(), setup); err != nil {
		t.Fatal(err)
	}
	if len(relay.payloads) != 0 {
		t.Fatalf("VS_PROBE_CONTROL sized the slot: %v", relay.payloads)
	}
}

func TestRelayRefusesAnIsochronousEndpointWithNoStream(t *testing.T) {
	_, err := NewRelay(streamRelayImage(), newFakeControl(), map[uint8]DataEndpoint{0x81: newFakeEndpoint()}, &fakeDevice{})
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "0x81") {
		t.Fatalf("NewRelay() error = %v, want an ErrUnsupported naming endpoint 0x81", err)
	}
}

// SET_INTERFACE(alt != 0) starts an isochronous stream and alt 0 stops it, for
// video and audio alike; the patched f_fs reports the pair as ENABLE and DISABLE.
// A host sends the stop routinely, so it has to drain the pipeline, and it is not
// a deconfigure: resetting the source there would drop the negotiated format.
func TestRelayStartsAndStopsTheStreamOnTheAlternateEdge(t *testing.T) {
	control := newFakeControl()
	stream := newFakeStream()
	device := &fakeDevice{}
	relay, err := NewRelay(streamRelayImage(), control, map[uint8]DataEndpoint{0x81: stream}, device)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- relay.Run(context.Background()) }()
	control.events <- Event{Type: EventEnable}
	waitStream(t, stream, func(starts, stops int) bool { return starts == 1 }, "the alternate edge did not start the stream")
	control.events <- Event{Type: EventDisable}
	waitStream(t, stream, func(starts, stops int) bool { return stops == 1 }, "alternate 0 left the isochronous stream running")
	device.mu.Lock()
	resets := device.resets
	device.mu.Unlock()
	if resets != 0 {
		t.Fatalf("a stream stop reset the source %d times", resets)
	}
	_ = relay.Close()
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatalf("Run returned %v", err)
	}
}

func waitStream(t *testing.T, stream *fakeStream, ok func(starts, stops int) bool, message string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		stream.mu.Lock()
		starts, stops := len(stream.payloads), stream.stops
		stream.mu.Unlock()
		if ok(starts, stops) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		time.Sleep(time.Millisecond)
	}
}
