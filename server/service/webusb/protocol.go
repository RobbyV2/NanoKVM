package webusb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"NanoKVM-Server/service/functionfs"
)

const (
	HeaderBytes      = 32
	MaxPayloadBytes  = functionfs.MaxTransferBytes
	MaxPending       = 8
	MaxPendingBytes  = 256 << 10
	DefaultTimeoutMS = 5000
)

const (
	TypeControl uint8 = iota + 1
	TypeTransferIn
	TypeTransferOut
	TypeClearHalt
	TypeReset
	TypeCancel
	TypeDisconnect
)

const responseBit = 0x80

const (
	StatusOK uint8 = iota
	StatusStall
	StatusTimeout
	StatusDisconnected
	StatusCanceled
	StatusIO
)

var (
	ErrProtocol     = errors.New("webusb: protocol error")
	ErrBackpressure = errors.New("webusb: transfer limit reached")
	ErrDisconnected = errors.New("webusb: device disconnected")
)

type Packet struct {
	Type           uint8
	Status         uint8
	Flags          uint8
	ID             uint32
	Endpoint       uint8
	RequestType    uint8
	Request        uint8
	Transfer       uint8
	Value          uint16
	Index          uint16
	DeclaredLength uint32
	TimeoutMS      uint32
	Payload        []byte
}

func Encode(packet Packet) ([]byte, error) {
	if len(packet.Payload) > MaxPayloadBytes || packet.DeclaredLength > MaxPayloadBytes {
		return nil, fmt.Errorf("%w: payload length", ErrProtocol)
	}
	data := make([]byte, HeaderBytes+len(packet.Payload))
	copy(data, "NKUF")
	data[4], data[5], data[6], data[7] = 1, packet.Type, packet.Status, packet.Flags
	binary.BigEndian.PutUint32(data[8:12], packet.ID)
	data[12], data[13], data[14], data[15] = packet.Endpoint, packet.RequestType, packet.Request, packet.Transfer
	binary.BigEndian.PutUint16(data[16:18], packet.Value)
	binary.BigEndian.PutUint16(data[18:20], packet.Index)
	binary.BigEndian.PutUint32(data[20:24], packet.DeclaredLength)
	binary.BigEndian.PutUint32(data[24:28], uint32(len(packet.Payload)))
	binary.BigEndian.PutUint32(data[28:32], packet.TimeoutMS)
	copy(data[HeaderBytes:], packet.Payload)
	return data, nil
}

func Decode(data []byte) (Packet, error) {
	if len(data) < HeaderBytes || string(data[:4]) != "NKUF" || data[4] != 1 {
		return Packet{}, fmt.Errorf("%w: header", ErrProtocol)
	}
	length := binary.BigEndian.Uint32(data[24:28])
	declared := binary.BigEndian.Uint32(data[20:24])
	if length > MaxPayloadBytes || declared > MaxPayloadBytes || int(length) != len(data)-HeaderBytes {
		return Packet{}, fmt.Errorf("%w: length", ErrProtocol)
	}
	return Packet{
		Type: data[5], Status: data[6], Flags: data[7], ID: binary.BigEndian.Uint32(data[8:12]),
		Endpoint: data[12], RequestType: data[13], Request: data[14], Transfer: data[15],
		Value: binary.BigEndian.Uint16(data[16:18]), Index: binary.BigEndian.Uint16(data[18:20]),
		DeclaredLength: declared, TimeoutMS: binary.BigEndian.Uint32(data[28:32]),
		Payload: slices.Clone(data[HeaderBytes:]),
	}, nil
}

type response struct {
	packet Packet
	err    error
}

type pendingRequest struct {
	waiter   chan response
	typeID   uint8
	reserved uint32
}

type Device struct {
	send func([]byte) error

	mu           sync.Mutex
	next         uint32
	pending      map[uint32]pendingRequest
	pendingBytes int
	closed       bool
}

func NewDevice(send func([]byte) error) *Device {
	return &Device{send: send, pending: make(map[uint32]pendingRequest)}
}

func (d *Device) Descriptor(kind uint8, index uint8, recipient uint16, limit int) ([]byte, error) {
	if limit < 0 || limit > functionfs.MaxControlBytes {
		return nil, fmt.Errorf("%w: descriptor limit", ErrProtocol)
	}
	return d.roundTrip(context.Background(), Packet{
		Type: TypeControl, RequestType: 0x80, Request: 6, Value: uint16(kind)<<8 | uint16(index),
		Index: recipient, DeclaredLength: uint32(limit), TimeoutMS: DefaultTimeoutMS,
	})
}

func (d *Device) Control(ctx context.Context, setup functionfs.Setup, data []byte) ([]byte, error) {
	declared := uint32(len(data))
	if setup.RequestType&0x80 != 0 {
		declared = uint32(setup.Length)
	}
	return d.roundTrip(ctx, Packet{
		Type: TypeControl, RequestType: setup.RequestType, Request: setup.Request, Value: setup.Value,
		Index: setup.Index, DeclaredLength: declared, TimeoutMS: DefaultTimeoutMS, Payload: slices.Clone(data),
	})
}

func (d *Device) Transfer(ctx context.Context, endpoint functionfs.Endpoint, data []byte) ([]byte, error) {
	typeID := TypeTransferOut
	declared := uint32(len(data))
	payload := slices.Clone(data)
	if endpoint.SourceAddress&0x80 != 0 {
		typeID, payload = TypeTransferIn, nil
	}
	transfer := uint8(2)
	if endpoint.Transfer == "interrupt" {
		transfer = 3
	}
	return d.roundTrip(ctx, Packet{
		Type: typeID, Endpoint: endpoint.SourceAddress, Transfer: transfer,
		DeclaredLength: declared, TimeoutMS: DefaultTimeoutMS, Payload: payload,
	})
}

func (d *Device) ClearHalt(endpoint uint8) error {
	_, err := d.roundTrip(context.Background(), Packet{Type: TypeClearHalt, Endpoint: endpoint, TimeoutMS: DefaultTimeoutMS})
	return err
}

func (d *Device) Reset() error {
	_, err := d.roundTrip(context.Background(), Packet{Type: TypeReset, TimeoutMS: DefaultTimeoutMS})
	return err
}

func (d *Device) Receive(data []byte) error {
	packet, err := Decode(data)
	if err != nil {
		return err
	}
	if packet.Type&responseBit == 0 || packet.ID == 0 {
		return fmt.Errorf("%w: unsolicited packet", ErrProtocol)
	}
	d.mu.Lock()
	pending, exists := d.pending[packet.ID]
	if exists {
		delete(d.pending, packet.ID)
		d.pendingBytes -= int(pending.reserved)
	}
	d.mu.Unlock()
	if !exists {
		return fmt.Errorf("%w: unknown request %d", ErrProtocol, packet.ID)
	}
	if packet.Type != pending.typeID|responseBit || uint32(len(packet.Payload)) > pending.reserved {
		pending.waiter <- response{err: fmt.Errorf("%w: mismatched response", ErrProtocol)}
		return fmt.Errorf("%w: mismatched response", ErrProtocol)
	}
	pending.waiter <- response{packet: packet, err: statusError(packet.Status)}
	return nil
}

func (d *Device) roundTrip(ctx context.Context, packet Packet) ([]byte, error) {
	if packet.DeclaredLength > MaxPayloadBytes || len(packet.Payload) > MaxPayloadBytes {
		return nil, fmt.Errorf("%w: transfer length", ErrProtocol)
	}
	waiter := make(chan response, 1)
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, ErrDisconnected
	}
	if len(d.pending) >= MaxPending || d.pendingBytes+int(packet.DeclaredLength) > MaxPendingBytes {
		d.mu.Unlock()
		return nil, ErrBackpressure
	}
	d.next++
	if d.next == 0 {
		d.next++
	}
	packet.ID = d.next
	d.pending[packet.ID] = pendingRequest{waiter: waiter, typeID: packet.Type, reserved: packet.DeclaredLength}
	d.pendingBytes += int(packet.DeclaredLength)
	d.mu.Unlock()

	encoded, err := Encode(packet)
	if err == nil {
		err = d.send(encoded)
	}
	if err != nil {
		d.remove(packet.ID, packet.DeclaredLength)
		return nil, err
	}
	timeout := time.Duration(packet.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = DefaultTimeoutMS * time.Millisecond
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-waiter:
		return result.packet.Payload, result.err
	case <-ctx.Done():
		d.remove(packet.ID, packet.DeclaredLength)
		cancel, _ := Encode(Packet{Type: TypeCancel, ID: packet.ID})
		_ = d.send(cancel)
		return nil, ctx.Err()
	case <-timer.C:
		d.remove(packet.ID, packet.DeclaredLength)
		cancel, _ := Encode(Packet{Type: TypeCancel, ID: packet.ID})
		_ = d.send(cancel)
		return nil, functionfs.ErrTransferTime
	}
}

func (d *Device) remove(id uint32, length uint32) {
	d.mu.Lock()
	if _, ok := d.pending[id]; ok {
		delete(d.pending, id)
		d.pendingBytes -= int(length)
	}
	d.mu.Unlock()
}

func (d *Device) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	waiters := d.pending
	d.pending = make(map[uint32]pendingRequest)
	d.pendingBytes = 0
	d.mu.Unlock()
	for _, pending := range waiters {
		pending.waiter <- response{err: ErrDisconnected}
	}
	packet, _ := Encode(Packet{Type: TypeDisconnect})
	return d.send(packet)
}

func statusError(status uint8) error {
	switch status {
	case StatusOK:
		return nil
	case StatusStall:
		return functionfs.ErrStall
	case StatusTimeout:
		return functionfs.ErrTransferTime
	case StatusCanceled:
		return context.Canceled
	case StatusDisconnected:
		return ErrDisconnected
	default:
		return functionfs.ErrTransfer
	}
}
