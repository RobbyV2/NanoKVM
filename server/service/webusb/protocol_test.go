package webusb

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/service/functionfs"
	"NanoKVM-Server/service/passthrough"
	"NanoKVM-Server/service/presentation"
)

func TestDeviceRoundTripValidatesResponse(t *testing.T) {
	requests := make(chan []byte, 1)
	device := NewDevice(func(data []byte) error {
		requests <- data
		return nil
	})
	result := make(chan error, 1)
	go func() {
		data, err := device.roundTrip(context.Background(), Packet{Type: TypeTransferIn, Endpoint: 0x81, DeclaredLength: 16, TimeoutMS: 1000})
		if err == nil && string(data) != "ok" {
			err = errors.New("wrong payload")
		}
		result <- err
	}()
	request, err := Decode(<-requests)
	if err != nil {
		t.Fatal(err)
	}
	response, _ := Encode(Packet{Type: request.Type | responseBit, ID: request.ID, DeclaredLength: 2, Payload: []byte("ok")})
	if err := device.Receive(response); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestDeviceCancellationIsBounded(t *testing.T) {
	requests := make(chan []byte, 2)
	device := NewDevice(func(data []byte) error { requests <- data; return nil })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := device.roundTrip(ctx, Packet{Type: TypeTransferIn, DeclaredLength: 64, TimeoutMS: 1000})
		done <- err
	}()
	<-requests
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel returned %v", err)
	}
	cancelPacket, err := Decode(<-requests)
	if err != nil || cancelPacket.Type != TypeCancel {
		t.Fatalf("cancel packet=%+v err=%v", cancelPacket, err)
	}
}

func TestDeviceTimeoutCancelsBrowserTransfer(t *testing.T) {
	requests := make(chan []byte, 2)
	device := NewDevice(func(data []byte) error { requests <- data; return nil })
	done := make(chan error, 1)
	go func() {
		_, err := device.roundTrip(context.Background(), Packet{Type: TypeTransferIn, DeclaredLength: 64, TimeoutMS: 1})
		done <- err
	}()
	<-requests
	if err := <-done; !errors.Is(err, functionfs.ErrTransferTime) {
		t.Fatalf("timeout returned %v", err)
	}
	cancelPacket, err := Decode(<-requests)
	if err != nil || cancelPacket.Type != TypeCancel {
		t.Fatalf("cancel packet=%+v err=%v", cancelPacket, err)
	}
}

func TestDeviceBackpressure(t *testing.T) {
	var mu sync.Mutex
	var requests [][]byte
	device := NewDevice(func(data []byte) error {
		mu.Lock()
		requests = append(requests, data)
		mu.Unlock()
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for range MaxPending {
		go device.roundTrip(ctx, Packet{Type: TypeTransferIn, DeclaredLength: 1, TimeoutMS: 1000})
	}
	deadline := time.Now().Add(time.Second)
	for {
		mu.Lock()
		count := len(requests)
		mu.Unlock()
		if count == MaxPending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("requests did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := device.roundTrip(ctx, Packet{Type: TypeTransferIn, DeclaredLength: 1}); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("extra request returned %v", err)
	}
}

func FuzzDecode(f *testing.F) {
	seed, _ := Encode(Packet{Type: TypeTransferIn, ID: 1, DeclaredLength: 8, Payload: []byte{1, 2}})
	f.Add(seed)
	f.Add([]byte("NKUF"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > HeaderBytes+MaxPayloadBytes+1 {
			return
		}
		_, _ = Decode(data)
	})
}

type stoppingStarter struct{ stopped int }

func (s *stoppingStarter) StartRemoteHybrid(context.Context, string, passthrough.HybridRelay, presentation.FunctionFS) (*passthrough.Session, error) {
	return nil, errors.New("unused")
}

func (s *stoppingStarter) StopSession(*passthrough.Session) error {
	s.stopped++
	return nil
}

func TestSessionCloseDisconnectsTheBrowser(t *testing.T) {
	frames := make(chan []byte, 4)
	manager := &stoppingStarter{}
	session := &Session{
		cancel: func() {}, device: NewDevice(func(data []byte) error { frames <- data; return nil }),
		ready: make(chan error, 1), done: make(chan error, 1),
		manager: manager, passthrough: &passthrough.Session{},
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if manager.stopped != 1 {
		t.Fatalf("stopped = %d", manager.stopped)
	}
	select {
	case data := <-frames:
		packet, err := Decode(data)
		if err != nil || packet.Type != TypeDisconnect {
			t.Fatalf("packet=%+v err=%v", packet, err)
		}
	default:
		t.Fatal("close did not disconnect the browser relay")
	}
}
