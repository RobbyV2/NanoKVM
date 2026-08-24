//go:build linux

package functionfs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"NanoKVM-Server/service/presentation"

	"golang.org/x/sys/unix"
)

const (
	functionFSMount = "/dev/ffs-hybrid"
	usbFSRoot       = "/dev/bus/usb"
	usbTimeoutMS    = 5000

	iocWrite = 1
	iocRead  = 2

	usbdevfsControl         = uint((iocRead|iocWrite)<<30 | 24<<16 | 'U'<<8)
	usbdevfsSetInterface    = uint(iocRead<<30 | 8<<16 | 'U'<<8 | 4)
	usbdevfsSubmitURB       = uint(iocRead<<30 | 56<<16 | 'U'<<8 | 10)
	usbdevfsDiscardURB      = uint('U'<<8 | 11)
	usbdevfsReapURBNoDelay  = uint(iocWrite<<30 | 8<<16 | 'U'<<8 | 13)
	usbdevfsReset           = uint('U'<<8 | 20)
	usbdevfsClearHalt       = uint(iocRead<<30 | 4<<16 | 'U'<<8 | 21)
	usbdevfsDisconnectClaim = uint(iocRead<<30 | 264<<16 | 'U'<<8 | 27)
)

// How long a cancelled transfer is waited for before the request is abandoned.
// The URB and its buffer stay pinned and pending: only a reap or the closing of
// the device file may release them.
var transferCancelGrace = time.Second

type Prepared struct {
	Image Image
	Relay *Relay
}

func Prepare(devicePath string, bus uint32, address uint32, caps presentation.CapabilityTable) (_ *Prepared, err error) {
	raw, err := readDescriptors(filepath.Join(devicePath, "descriptors"))
	if err != nil {
		return nil, fmt.Errorf("read imported descriptors: %w", err)
	}
	interfaces, err := scanInterfaces(raw)
	if err != nil {
		return nil, err
	}
	if bus > 999 || address > 999 {
		return nil, fmt.Errorf("%w: bus %d address %d", ErrMalformed, bus, address)
	}
	device, err := openUSBDevice(filepath.Join(usbFSRoot, fmt.Sprintf("%03d", bus), fmt.Sprintf("%03d", address)), interfaces, raw)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = device.Close()
		}
	}()

	image, err := Import(raw, device, caps)
	if err != nil {
		return nil, err
	}
	control, endpoints, err := openFunctionFS(image)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			for _, endpoint := range endpoints {
				_ = endpoint.Close()
			}
			_ = control.Close()
		}
	}()
	attachStreams(image, endpoints, device)
	relay, err := NewRelay(image, control, endpoints, device)
	if err != nil {
		return nil, err
	}
	return &Prepared{Image: image, Relay: relay}, nil
}

// Only a locally attached source can be driven with pipelined isochronous URBs,
// so a remote transport keeps its endpoints unwrapped and NewRelay refuses the
// image rather than presenting a camera that would never produce a frame.
func attachStreams(image Image, endpoints map[uint8]DataEndpoint, device USBDevice) {
	source, ok := device.(*linuxDevice)
	if !ok {
		return
	}
	for _, endpoint := range image.Function.Endpoints {
		if endpoint.Transfer != presentation.EndpointIsochronous {
			continue
		}
		file, ok := endpoints[endpoint.Address].(*linuxEndpoint)
		if !ok {
			continue
		}
		number := image.EndpointOwners[endpoint.Address]
		endpoints[endpoint.Address] = newIsochronousEndpoint(file, source, endpoint, number, image.Alternates[number])
	}
}

func PrepareRemote(raw []byte, fetcher Fetcher, device USBDevice, caps presentation.CapabilityTable) (_ *Prepared, err error) {
	image, err := Import(raw, fetcher, caps)
	if err != nil {
		return nil, err
	}
	control, endpoints, err := openFunctionFS(image)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			for _, endpoint := range endpoints {
				_ = endpoint.Close()
			}
			_ = control.Close()
		}
	}()
	attachStreams(image, endpoints, device)
	relay, err := NewRelay(image, control, endpoints, device)
	if err != nil {
		return nil, err
	}
	return &Prepared{Image: image, Relay: relay}, nil
}

func readDescriptors(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxDescriptorBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxDescriptorBytes {
		return nil, fmt.Errorf("%w: descriptor stream exceeds %d bytes", ErrMalformed, MaxDescriptorBytes)
	}
	return data, nil
}

func Cleanup() error {
	err := unix.Unmount(functionFSMount, 0)
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOENT) {
		err = nil
	}
	removeErr := os.Remove(functionFSMount)
	if errors.Is(removeErr, os.ErrNotExist) || errors.Is(removeErr, syscall.EBUSY) {
		removeErr = nil
	}
	return errors.Join(err, removeErr)
}

func scanInterfaces(raw []byte) ([]uint8, error) {
	_, _, descriptors, err := splitDescriptors(raw)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint8]bool)
	var interfaces []uint8
	for _, item := range descriptors {
		if item.data[1] != 4 || item.data[3] != 0 || seen[item.data[2]] {
			continue
		}
		seen[item.data[2]] = true
		interfaces = append(interfaces, item.data[2])
	}
	if len(interfaces) == 0 || len(interfaces) > MaxInterfaces {
		return nil, fmt.Errorf("%w: interface count %d", ErrMalformed, len(interfaces))
	}
	return interfaces, nil
}

type linuxControl struct {
	file *os.File
}

func openFunctionFS(image Image) (*linuxControl, map[uint8]DataEndpoint, error) {
	if err := Cleanup(); err != nil {
		return nil, nil, err
	}
	if err := os.Mkdir(functionFSMount, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create FunctionFS mount: %w", err)
	}
	if err := unix.Mount("hybrid", functionFSMount, "functionfs", unix.MS_NOSUID|unix.MS_NODEV, ""); err != nil {
		_ = os.Remove(functionFSMount)
		return nil, nil, fmt.Errorf("mount FunctionFS: %w", err)
	}
	fail := func(err error) (*linuxControl, map[uint8]DataEndpoint, error) {
		_ = Cleanup()
		return nil, nil, err
	}
	ep0, err := os.OpenFile(filepath.Join(functionFSMount, "ep0"), os.O_RDWR, 0)
	if err != nil {
		return fail(fmt.Errorf("open FunctionFS ep0: %w", err))
	}
	control := &linuxControl{file: ep0}
	if _, err := writeAll(ep0, image.Descriptors); err != nil {
		_ = ep0.Close()
		return fail(fmt.Errorf("write FunctionFS descriptors: %w", diagnoseDescriptorWrite(err, image.Descriptors)))
	}
	if _, err := writeAll(ep0, image.StringTable); err != nil {
		_ = ep0.Close()
		return fail(fmt.Errorf("write FunctionFS strings: %w", err))
	}

	endpoints := make(map[uint8]DataEndpoint, len(image.Function.Endpoints))
	for _, endpoint := range image.Function.Endpoints {
		file, err := os.OpenFile(filepath.Join(functionFSMount, functionFSEndpointName(endpoint.Address)), os.O_RDWR, 0)
		if err != nil {
			for _, opened := range endpoints {
				_ = opened.Close()
			}
			_ = ep0.Close()
			return fail(fmt.Errorf("open FunctionFS endpoint 0x%02x: %w", endpoint.Address, err))
		}
		endpoints[endpoint.Address] = &linuxEndpoint{file: file, address: endpoint.Address}
	}
	return control, endpoints, nil
}

// ffs_do_single_desc switches on bDescriptorType and has no case for
// USB_DT_CS_INTERFACE or USB_DT_CS_ENDPOINT, so a block carrying either falls
// through to its default and the ep0 write is EINVAL with nothing but a
// pr_vdebug line to say why. Every CDC and every video function is made of them.
func diagnoseDescriptorWrite(err error, block []byte) error {
	if !errors.Is(err, syscall.EINVAL) {
		return err
	}
	for offset := 20; offset+1 < len(block); {
		length := int(block[offset])
		if length < 2 {
			break
		}
		if block[offset+1] == 0x24 || block[offset+1] == 0x25 {
			return fmt.Errorf("%w: this kernel's ffs_do_single_desc has no case for class-specific descriptor type 0x%02x, so a CDC or video function cannot be presented through FunctionFS: %w", ErrUnsupported, block[offset+1], err)
		}
		offset += length
	}
	return err
}

func functionFSEndpointName(address uint8) string {
	return fmt.Sprintf("ep%02x", address)
}

func (c *linuxControl) NextEvent() (Event, error) {
	data := make([]byte, 12)
	if _, err := io.ReadFull(c.file, data); err != nil {
		return Event{}, err
	}
	return DecodeEvent(data)
}

func (c *linuxControl) ReadControl(data []byte) (int, error) {
	return unix.Read(int(c.file.Fd()), data)
}
func (c *linuxControl) WriteControl(data []byte) (int, error) {
	return unix.Write(int(c.file.Fd()), data)
}

func (c *linuxControl) Stall(setup Setup) error {
	if setup.RequestType&0x80 != 0 {
		_, err := unix.Read(int(c.file.Fd()), nil)
		return err
	}
	_, err := unix.Write(int(c.file.Fd()), nil)
	return err
}

func (c *linuxControl) Close() error {
	return c.file.Close()
}

type linuxEndpoint struct {
	file    *os.File
	address uint8
}

func (e *linuxEndpoint) Read(data []byte) (int, error)  { return e.file.Read(data) }
func (e *linuxEndpoint) Write(data []byte) (int, error) { return e.file.Write(data) }
func (e *linuxEndpoint) Close() error                   { return e.file.Close() }
func (e *linuxEndpoint) Stall() error {
	if e.address&0x80 != 0 {
		_, err := unix.Read(int(e.file.Fd()), nil)
		return err
	}
	_, err := unix.Write(int(e.file.Fd()), nil)
	return err
}

type usbControl struct {
	RequestType uint8
	Request     uint8
	Value       uint16
	Index       uint16
	Length      uint16
	Timeout     uint32
	_           uint32
	Data        unsafe.Pointer
}

type usbURB struct {
	Type         uint8
	Endpoint     uint8
	_            uint16
	Status       int32
	Flags        uint32
	Buffer       unsafe.Pointer
	BufferLength int32
	ActualLength int32
	StartFrame   int32
	StreamID     uint32
	ErrorCount   int32
	Signal       uint32
	UserContext  unsafe.Pointer
}

type setInterface struct {
	Interface uint32
	Alternate uint32
}

type usbISOPacket struct {
	Length       uint32
	ActualLength uint32
	Status       uint32
}

// usbdevfs_urb ends in a flexible iso_frame_desc array, so the packet
// descriptors have to be contiguous with the URB the ioctl is handed.
type usbISORequest struct {
	urb     usbURB
	packets [maxISOPackets]usbISOPacket
}

type disconnectClaim struct {
	Interface uint32
	Flags     uint32
	Driver    [256]byte
}

type usbRequest struct {
	ctx         context.Context
	urb         usbURB
	iso         *usbISORequest
	packets     int
	buffer      []byte
	pinner      runtime.Pinner
	done        chan transferResult
	discard     bool
	directionIn bool
}

func (r *usbRequest) block() *usbURB {
	if r.iso != nil {
		return &r.iso.urb
	}
	return &r.urb
}

type transferResult struct {
	data    []byte
	lengths []int
	err     error
}

type linuxDevice struct {
	file       *os.File
	interfaces []uint8
	expected   []byte
	requests   chan *usbRequest
	close      chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
	controlMu  sync.Mutex
}

func openUSBDevice(path string, interfaces []uint8, expected []byte) (*linuxDevice, error) {
	if unsafe.Sizeof(uintptr(0)) != 8 || unsafe.Sizeof(usbControl{}) != 24 || unsafe.Sizeof(usbURB{}) != 56 {
		return nil, fmt.Errorf("%w: unsupported usbfs ABI", ErrUnsupported)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open imported device: %w", err)
	}
	device := &linuxDevice{
		file: file, interfaces: slices.Clone(interfaces), expected: slices.Clone(expected),
		requests: make(chan *usbRequest), close: make(chan struct{}), done: make(chan struct{}),
	}
	for _, number := range interfaces {
		claim := disconnectClaim{Interface: uint32(number)}
		if err := ioctl(file.Fd(), usbdevfsDisconnectClaim, unsafe.Pointer(&claim)); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("claim imported interface %d: %w", number, err)
		}
	}
	go device.reactor()
	return device, nil
}

func (d *linuxDevice) Descriptor(kind uint8, index uint8, recipient uint16, limit int) ([]byte, error) {
	if limit <= 0 || limit > MaxControlBytes {
		return nil, ErrEndpointSize
	}
	setup := descriptorSetup(kind, index, recipient, limit)
	return d.Control(context.Background(), setup, nil)
}

func descriptorSetup(kind uint8, index uint8, recipient uint16, limit int) Setup {
	requestType := uint8(0x80)
	if kind == 0x21 || kind == 0x22 {
		requestType = 0x81
	}
	return Setup{RequestType: requestType, Request: 6, Value: uint16(kind)<<8 | uint16(index), Index: recipient, Length: uint16(limit)}
}

func (d *linuxDevice) Control(ctx context.Context, setup Setup, data []byte) ([]byte, error) {
	if setup.Length > MaxControlBytes || setup.RequestType&0x80 == 0 && len(data) != int(setup.Length) {
		return nil, ErrEndpointSize
	}
	buffer := data
	if setup.RequestType&0x80 != 0 {
		buffer = make([]byte, setup.Length)
	}
	var pointer unsafe.Pointer
	if len(buffer) != 0 {
		pointer = unsafe.Pointer(&buffer[0])
	}
	transfer := usbControl{
		RequestType: setup.RequestType, Request: setup.Request, Value: setup.Value,
		Index: setup.Index, Length: setup.Length, Timeout: usbTimeoutMS, Data: pointer,
	}
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n, err := ioctlResult(d.file.Fd(), usbdevfsControl, unsafe.Pointer(&transfer))
	runtime.KeepAlive(buffer)
	if err != nil {
		return nil, normalizeUSBError(err)
	}
	if n < 0 || n > len(buffer) {
		return nil, fmt.Errorf("%w: control returned %d bytes", ErrTransfer, n)
	}
	if setup.RequestType&0x80 != 0 {
		return slices.Clone(buffer[:n]), nil
	}
	return nil, nil
}

func (d *linuxDevice) Transfer(ctx context.Context, endpoint Endpoint, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > MaxTransferBytes {
		return nil, ErrEndpointSize
	}
	request := &usbRequest{ctx: ctx, buffer: make([]byte, len(data)), done: make(chan transferResult, 1), directionIn: endpoint.SourceAddress&0x80 != 0}
	if !request.directionIn {
		copy(request.buffer, data)
	}
	typeID := uint8(3)
	if endpoint.Transfer == string(presentation.EndpointInterrupt) {
		typeID = 1
	}
	request.urb = usbURB{Type: typeID, Endpoint: endpoint.SourceAddress, BufferLength: int32(len(request.buffer))}
	request.urb.Buffer = unsafe.Pointer(&request.buffer[0])
	request.pinner.Pin(&request.urb)
	request.pinner.Pin(&request.buffer[0])
	if err := d.enqueue(ctx, request); err != nil {
		return nil, err
	}
	select {
	case result := <-request.done:
		return result.data, normalizeUSBError(result.err)
	case <-ctx.Done():
	case <-d.close:
		return nil, ErrClosed
	}

	grace := time.NewTimer(transferCancelGrace)
	defer grace.Stop()
	select {
	case result := <-request.done:
		return result.data, normalizeUSBError(result.err)
	case <-grace.C:
		return nil, fmt.Errorf("%w: endpoint 0x%02x did not answer cancellation", ErrTransfer, endpoint.SourceAddress)
	case <-d.close:
		return nil, ErrClosed
	}
}

func (d *linuxDevice) enqueue(ctx context.Context, request *usbRequest) error {
	select {
	case d.requests <- request:
		return nil
	case <-ctx.Done():
		request.pinner.Unpin()
		return ctx.Err()
	case <-d.close:
		request.pinner.Unpin()
		return ErrClosed
	}
}

func (d *linuxDevice) reactor() {
	defer close(d.done)
	pending := make(map[*usbURB]*usbRequest)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	var dead error
	for {
		select {
		case request := <-d.requests:
			if dead != nil {
				d.complete(request, nil, dead)
			} else if err := ioctl(d.file.Fd(), usbdevfsSubmitURB, unsafe.Pointer(request.block())); err != nil {
				d.complete(request, nil, err)
			} else {
				pending[request.block()] = request
			}
		case <-ticker.C:
			if dead != nil {
				continue
			}
			for _, request := range pending {
				if request.ctx.Err() != nil && !request.discard {
					request.discard = true
					if err := ioctl(d.file.Fd(), usbdevfsDiscardURB, unsafe.Pointer(request.block())); err != nil && !errors.Is(err, syscall.EINVAL) {
						dead = err
						break
					}
				}
			}
			if dead == nil {
				dead = d.reap(pending)
			}
			if dead != nil {
				_ = d.file.Close()
				for urb, request := range pending {
					d.complete(request, nil, dead)
					delete(pending, urb)
				}
			}
		case <-d.close:
			for _, request := range pending {
				_ = ioctl(d.file.Fd(), usbdevfsDiscardURB, unsafe.Pointer(request.block()))
			}
			_ = d.file.Close()
			for _, request := range pending {
				d.complete(request, nil, ErrClosed)
			}
			return
		}
	}
}

func (d *linuxDevice) reap(pending map[*usbURB]*usbRequest) error {
	for {
		var pointer unsafe.Pointer
		err := ioctl(d.file.Fd(), usbdevfsReapURBNoDelay, unsafe.Pointer(&pointer))
		if errors.Is(err, syscall.EAGAIN) {
			return nil
		}
		if err != nil {
			return err
		}
		urb := (*usbURB)(pointer)
		request := pending[urb]
		if request == nil {
			continue
		}
		delete(pending, urb)
		// A transfer that completed is delivered even when its deadline passed
		// while it was in flight; only a transfer that failed reports why it
		// was cancelled.
		if urb.Status != 0 {
			err := error(syscall.Errno(-urb.Status))
			if ctxErr := request.ctx.Err(); ctxErr != nil {
				err = ctxErr
			}
			d.complete(request, nil, err)
			continue
		}
		if request.packets != 0 {
			d.completeISO(request)
			continue
		}
		if urb.ActualLength < 0 || int(urb.ActualLength) > len(request.buffer) {
			d.complete(request, nil, fmt.Errorf("%w: source returned %d bytes", ErrTransfer, urb.ActualLength))
			continue
		}
		var data []byte
		if request.directionIn {
			data = slices.Clone(request.buffer[:urb.ActualLength])
		}
		d.complete(request, data, nil)
	}
}

func (d *linuxDevice) complete(request *usbRequest, data []byte, err error) {
	request.pinner.Unpin()
	request.done <- transferResult{data: data, err: err}
}

// An isochronous URB carries its errors per packet, not in urb.status, and a
// short or missing packet is ordinary rather than a failure of the transfer.
func (d *linuxDevice) completeISO(request *usbRequest) {
	lengths := make([]int, request.packets)
	for index := range lengths {
		packet := request.iso.packets[index]
		if packet.Status != 0 || int(packet.ActualLength) > int(packet.Length) {
			continue
		}
		lengths[index] = int(packet.ActualLength)
	}
	request.pinner.Unpin()
	request.done <- transferResult{lengths: lengths}
}

// The source keeps every alternate setting the camera declared, so the one the
// host asked for on the presented interface has to be translated back and set on
// the imported device before its isochronous endpoint carries anything.
func (d *linuxDevice) SetAlternate(number uint8, alternate uint8) error {
	value := setInterface{Interface: uint32(number), Alternate: uint32(alternate)}
	return ioctl(d.file.Fd(), usbdevfsSetInterface, unsafe.Pointer(&value))
}

func (d *linuxDevice) beginISO(ctx context.Context, endpoint Endpoint, buffer []byte, packets int, packet int) (*usbRequest, error) {
	if packets <= 0 || packets > maxISOPackets || packet <= 0 || len(buffer) != packets*packet {
		return nil, fmt.Errorf("%w: %d isochronous packets of %d in %d bytes", ErrEndpointSize, packets, packet, len(buffer))
	}
	request := &usbRequest{
		ctx: ctx, iso: &usbISORequest{}, packets: packets, buffer: buffer,
		done: make(chan transferResult, 1), directionIn: endpoint.SourceAddress&0x80 != 0,
	}
	request.iso.urb = usbURB{
		Type: 0, Endpoint: endpoint.SourceAddress, Flags: urbISOASAP,
		BufferLength: int32(len(buffer)), Buffer: unsafe.Pointer(&buffer[0]), StreamID: uint32(packets),
	}
	for index := range packets {
		request.iso.packets[index].Length = uint32(packet)
	}
	// The buffer is the mlocked aio pool rather than Go memory, so only the URB
	// and its packet descriptors need pinning.
	request.pinner.Pin(request.iso)
	if err := d.enqueue(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

func (d *linuxDevice) ClearHalt(endpoint uint8) error {
	value := uint32(endpoint)
	return ioctl(d.file.Fd(), usbdevfsClearHalt, unsafe.Pointer(&value))
}

func (d *linuxDevice) Reset() error {
	d.controlMu.Lock()
	defer d.controlMu.Unlock()
	if err := ioctl(d.file.Fd(), usbdevfsReset, nil); err != nil {
		return err
	}
	for _, number := range d.interfaces {
		claim := disconnectClaim{Interface: uint32(number)}
		if err := ioctl(d.file.Fd(), usbdevfsDisconnectClaim, unsafe.Pointer(&claim)); err != nil {
			return err
		}
	}
	device, err := d.descriptorLocked(1, 0, 0, 18)
	if err != nil {
		return err
	}
	header, err := d.descriptorLocked(2, 0, 0, 9)
	if err != nil || len(header) != 9 {
		return errors.Join(err, ErrMalformed)
	}
	total := int(binary.LittleEndian.Uint16(header[2:4]))
	if total < 9 || total > MaxDescriptorBytes-18 {
		return ErrMalformed
	}
	config, err := d.descriptorLocked(2, 0, 0, total)
	if err != nil {
		return err
	}
	if !bytes.Equal(append(device, config...), d.expected) {
		return fmt.Errorf("%w: descriptors changed after reset", ErrAmbiguous)
	}
	return nil
}

func (d *linuxDevice) descriptorLocked(kind uint8, index uint8, recipient uint16, limit int) ([]byte, error) {
	buffer := make([]byte, limit)
	transfer := usbControl{RequestType: 0x80, Request: 6, Value: uint16(kind)<<8 | uint16(index), Index: recipient, Length: uint16(limit), Timeout: usbTimeoutMS, Data: unsafe.Pointer(&buffer[0])}
	n, err := ioctlResult(d.file.Fd(), usbdevfsControl, unsafe.Pointer(&transfer))
	runtime.KeepAlive(buffer)
	if err != nil || n < 0 || n > len(buffer) {
		return nil, errors.Join(err, ErrTransfer)
	}
	return slices.Clone(buffer[:n]), nil
}

func (d *linuxDevice) Close() error {
	var result error
	d.closeOnce.Do(func() {
		close(d.close)
		<-d.done
		result = d.file.Close()
		if errors.Is(result, os.ErrClosed) {
			result = nil
		}
	})
	return result
}

func ioctl(fd uintptr, request uint, pointer unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(request), uintptr(pointer))
	if errno != 0 {
		return errno
	}
	return nil
}

func ioctlResult(fd uintptr, request uint, pointer unsafe.Pointer) (int, error) {
	value, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(request), uintptr(pointer))
	if errno != 0 {
		return 0, errno
	}
	return int(value), nil
}

func normalizeUSBError(err error) error {
	if errors.Is(err, syscall.EPIPE) {
		return ErrStall
	}
	return err
}
