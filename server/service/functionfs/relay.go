package functionfs

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"NanoKVM-Server/service/presentation"
)

const stallRetryDelay = 10 * time.Millisecond

// Every data transfer is bounded so a source that answers neither the transfer
// nor its cancellation cannot hold a relay loop forever. A var so the tests can
// shorten it.
var transferTimeout = 5 * time.Second

var (
	ErrClosed       = errors.New("functionfs: relay closed")
	ErrTransfer     = errors.New("functionfs: transfer failed")
	ErrTransferTime = errors.New("functionfs: transfer timed out")
	ErrStall        = errors.New("functionfs: endpoint stalled")
)

type EventType uint8

const (
	EventBind EventType = iota
	EventUnbind
	EventEnable
	EventDisable
	EventSetup
	EventSuspend
	EventResume
)

type Setup struct {
	RequestType uint8
	Request     uint8
	Value       uint16
	Index       uint16
	Length      uint16
}

type Event struct {
	Type  EventType
	Setup Setup
}

type ControlEndpoint interface {
	NextEvent() (Event, error)
	ReadControl([]byte) (int, error)
	WriteControl([]byte) (int, error)
	Stall(Setup) error
	Close() error
}

type DataEndpoint interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Stall() error
	Close() error
}

type USBDevice interface {
	Control(context.Context, Setup, []byte) ([]byte, error)
	Transfer(context.Context, Endpoint, []byte) ([]byte, error)
	ClearHalt(uint8) error
	Reset() error
	Close() error
}

type Endpoint struct {
	SourceAddress uint8
	Address       uint8
	Transfer      string
	MaxPacket     uint16
}

type Relay struct {
	image     Image
	control   ControlEndpoint
	endpoints map[uint8]DataEndpoint
	device    USBDevice

	closeOnce sync.Once
	closed    chan struct{}
}

func NewRelay(image Image, control ControlEndpoint, endpoints map[uint8]DataEndpoint, device USBDevice) (*Relay, error) {
	if control == nil || device == nil || len(endpoints) != len(image.Function.Endpoints) {
		return nil, fmt.Errorf("%w: incomplete relay", ErrMalformed)
	}
	for _, endpoint := range image.Function.Endpoints {
		if endpoints[endpoint.Address] == nil {
			return nil, fmt.Errorf("%w: endpoint 0x%02x is missing", ErrMalformed, endpoint.Address)
		}
	}
	return &Relay{image: image, control: control, endpoints: endpoints, device: device, closed: make(chan struct{})}, nil
}

func (r *Relay) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer r.Close()
	go func() {
		select {
		case <-ctx.Done():
			_ = r.Close()
		case <-r.closed:
		}
	}()

	errorsOut := make(chan error, len(r.endpoints)+1)
	var transferCancel context.CancelFunc = func() {}
	var transfers sync.WaitGroup
	start := func() {
		transferCancel()
		transfers.Wait()
		var transferContext context.Context
		transferContext, transferCancel = context.WithCancel(ctx)
		limit := MaxTransferBytes
		if share := (256 << 10) / len(r.endpoints); share < limit {
			limit = share
		}
		for _, endpoint := range r.image.Function.Endpoints {
			endpoint := endpoint
			transfers.Add(1)
			go func() {
				defer transfers.Done()
				if err := r.transferLoop(transferContext, endpoint, limit); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, ErrClosed) {
					errorsOut <- err
				}
			}()
		}
	}
	stop := func() {
		transferCancel()
		transfers.Wait()
	}
	defer stop()

	events := make(chan Event)
	go func() {
		for {
			event, err := r.control.NextEvent()
			if err != nil {
				errorsOut <- err
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.closed:
			return ErrClosed
		case err := <-errorsOut:
			return err
		case event := <-events:
			switch event.Type {
			case EventEnable:
				start()
			case EventSuspend:
				stop()
			case EventResume:
				start()
			case EventDisable:
				stop()
				if err := r.device.Reset(); err != nil {
					return fmt.Errorf("reset imported device: %w", err)
				}
			case EventSetup:
				if err := r.handleSetup(ctx, event.Setup); err != nil {
					return err
				}
			case EventUnbind:
				return ErrClosed
			}
		}
	}
}

func (r *Relay) transferLoop(ctx context.Context, endpoint presentation.FunctionFSEndpoint, limit int) error {
	file := r.endpoints[endpoint.Address]
	size := limit
	if endpoint.Transfer == presentation.EndpointInterrupt {
		size = int(endpoint.MaxPacket)
	}
	buffer := make([]byte, size)
	source := Endpoint{
		SourceAddress: endpoint.SourceAddress,
		Address:       endpoint.Address,
		Transfer:      string(endpoint.Transfer),
		MaxPacket:     endpoint.MaxPacket,
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if endpoint.Address&0x80 != 0 {
			data, err := r.transfer(ctx, source, buffer)
			// An input endpoint with nothing to report times out by design, and
			// the source holds its data until the next poll, so re-arming the
			// transfer loses nothing.
			if errors.Is(err, ErrTransferTime) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if errors.Is(err, ErrStall) {
				if err := file.Stall(); err != nil {
					return err
				}
				if err := waitStall(ctx); err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("source endpoint 0x%02x: %w", endpoint.SourceAddress, err)
			}
			if len(data) == 0 {
				continue
			}
			if _, err := writeAll(file, data); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("functionfs endpoint 0x%02x: %w", endpoint.Address, err)
			}
			continue
		}

		n, err := file.Read(buffer)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("functionfs endpoint 0x%02x: %w", endpoint.Address, err)
		}
		if n == 0 {
			continue
		}
		if _, err := r.transfer(ctx, source, buffer[:n]); errors.Is(err, ErrStall) {
			if err := file.Stall(); err != nil {
				return err
			}
			if err := waitStall(ctx); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("source endpoint 0x%02x: %w", endpoint.SourceAddress, err)
		}
	}
}

func (r *Relay) transfer(ctx context.Context, source Endpoint, data []byte) ([]byte, error) {
	bounded, cancel := context.WithTimeout(ctx, transferTimeout)
	defer cancel()
	return r.device.Transfer(bounded, source, data)
}

func (r *Relay) handleSetup(ctx context.Context, setup Setup) error {
	if setup.Length > MaxControlBytes {
		return fmt.Errorf("%w: control length %d", ErrEndpointSize, setup.Length)
	}
	setup = r.reverseSetup(setup)
	if setup.RequestType&0x80 != 0 {
		data := r.cachedDescriptor(setup)
		var err error
		if data == nil {
			data, err = r.device.Control(ctx, setup, nil)
		}
		if err != nil {
			if stallErr := r.control.Stall(setup); stallErr != nil {
				return errors.Join(fmt.Errorf("control IN: %w", err), stallErr)
			}
			return nil
		}
		if len(data) > int(setup.Length) {
			data = data[:setup.Length]
		}
		if len(data) == 0 {
			_, err = r.control.WriteControl(nil)
		} else {
			_, err = writeAll(controlWriter{r.control}, data)
		}
		return err
	}

	data := make([]byte, setup.Length)
	if len(data) == 0 {
		if _, err := r.control.ReadControl(nil); err != nil {
			return fmt.Errorf("control OUT status: %w", err)
		}
	} else {
		n, err := io.ReadFull(controlReader{r.control}, data)
		if err != nil || n != len(data) {
			return fmt.Errorf("control OUT data: %w", err)
		}
	}
	if setup.RequestType == 0x02 && setup.Request == 1 && setup.Value == 0 && setup.Length == 0 && setup.Index&0xff00 == 0 {
		return r.device.ClearHalt(uint8(setup.Index))
	}
	_, err := r.device.Control(ctx, setup, data)
	return err
}

func waitStall(ctx context.Context) error {
	timer := time.NewTimer(stallRetryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *Relay) reverseSetup(setup Setup) Setup {
	switch setup.RequestType & 0x1f {
	case 1:
		mapped := uint8(setup.Index)
		for source, target := range r.image.Interfaces {
			if target == mapped {
				setup.Index = setup.Index&0xff00 | uint16(source)
				break
			}
		}
	case 2:
		mapped := uint8(setup.Index)
		for source, target := range r.image.Endpoints {
			if target == mapped {
				setup.Index = setup.Index&0xff00 | uint16(source)
				break
			}
		}
	}
	return setup
}

func (r *Relay) cachedDescriptor(setup Setup) []byte {
	if setup.Request != 6 || setup.RequestType&0x1f != 1 {
		return nil
	}
	mapped := uint8(setup.Index)
	for source, target := range r.image.Interfaces {
		if source != uint8(setup.Index) {
			continue
		}
		mapped = target
		break
	}
	switch uint8(setup.Value >> 8) {
	case 0x21:
		return slices.Clone(r.image.HIDDescriptors[mapped])
	case 0x22:
		return slices.Clone(r.image.HIDReports[mapped])
	default:
		return nil
	}
}

func (r *Relay) Close() error {
	var result error
	r.closeOnce.Do(func() {
		close(r.closed)
		for _, endpoint := range r.endpoints {
			result = errors.Join(result, endpoint.Close())
		}
		result = errors.Join(result, r.control.Close(), r.device.Close())
	})
	return result
}

type controlReader struct{ endpoint ControlEndpoint }

func (r controlReader) Read(data []byte) (int, error) { return r.endpoint.ReadControl(data) }

type controlWriter struct{ endpoint ControlEndpoint }

func (w controlWriter) Write(data []byte) (int, error) { return w.endpoint.WriteControl(data) }

func writeAll(writer interface{ Write([]byte) (int, error) }, data []byte) (int, error) {
	written := 0
	for written < len(data) {
		n, err := writer.Write(data[written:])
		written += n
		if err != nil {
			return written, err
		}
		if n == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func DecodeEvent(data []byte) (Event, error) {
	if len(data) != 12 || data[8] > uint8(EventResume) {
		return Event{}, fmt.Errorf("%w: functionfs event", ErrMalformed)
	}
	return Event{Type: EventType(data[8]), Setup: Setup{
		RequestType: data[0], Request: data[1], Value: binary.LittleEndian.Uint16(data[2:4]),
		Index: binary.LittleEndian.Uint16(data[4:6]), Length: binary.LittleEndian.Uint16(data[6:8]),
	}}, nil
}
