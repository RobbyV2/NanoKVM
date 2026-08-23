package passthrough

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	VHCIRoot    = "/sys/devices/platform/vhci_hcd.0"
	ExporterTCP = 3240

	statusAttribute = "status"
	attachAttribute = "attach"
	detachAttribute = "detach"

	keepAlive   = 30 * time.Second
	listTimeout = 10 * time.Second
)

// var rather than const so the tests point the package at t.TempDir(), the way
// bridge/config.go swaps its state directory.
var vhciRoot = VHCIRoot

type Hub string

const (
	HubHigh  Hub = "hs"
	HubSuper Hub = "ss"
)

// This defconfig builds one controller with VHCI_HC_PORTS 8, so ports 0..7 are
// hs and 8..15 are ss in the one status file.
func (s Speed) Hub() Hub {
	if s == SpeedSuper || s == SpeedSuperPlus {
		return HubSuper
	}
	return HubHigh
}

type VDevStatus uint32

const (
	VDevNull        VDevStatus = 4
	VDevNotAssigned VDevStatus = 5
	VDevUsed        VDevStatus = 6
	VDevError       VDevStatus = 7
)

var (
	ErrNoFreePort      = errors.New("passthrough: no free vhci port")
	ErrMalformed       = errors.New("passthrough: malformed vhci status line")
	ErrNotTCP          = errors.New("passthrough: exporter connection is not tcp")
	ErrExporterAddress = errors.New("passthrough: exporter address is not allowed")
)

type PortEntry struct {
	Hub    Hub
	Number uint32
	Status VDevStatus
	Speed  Speed
	DevID  uint32
	SockFD int
	BusID  string
}

type Attachment struct {
	Port   uint32
	Hub    Hub
	BusID  string
	Device Device
}

func Attach(ctx context.Context, addr string, busID string) (Attachment, error) {
	if err := guardRemote(ctx, addr, busID); err != nil {
		return Attachment{}, err
	}

	conn, err := dial(ctx, addr)
	if err != nil {
		return Attachment{}, err
	}
	// attach_store keeps its own reference to the struct file, so the kernel
	// still owns the socket once this descriptor goes away.
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return Attachment{}, fmt.Errorf("set exchange deadline: %w", err)
		}
	}

	device, err := Import(conn, busID)
	if err != nil {
		return Attachment{}, err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return Attachment{}, fmt.Errorf("clear exchange deadline: %w", err)
	}

	port, err := freePort(device.Speed)
	if err != nil {
		return Attachment{}, err
	}
	if err := attachSocket(conn, port, device); err != nil {
		return Attachment{}, err
	}
	return Attachment{Port: port, Hub: device.Speed.Hub(), BusID: busID, Device: device}, nil
}

func Detach(port uint32) error {
	return writeAttribute(detachAttribute, strconv.FormatUint(uint64(port), 10))
}

func List(ctx context.Context, addr string) ([]RemoteDevice, error) {
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	conn, err := dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set exchange deadline: %w", err)
	}
	return Devlist(conn)
}

var listExporter = List

// A device the backend cannot relay is refused here, before a vhci port is
// taken and long before the UDC is surrendered: finding out afterwards costs
// the operator the keyboard the gadget was serving. An exporter that will not
// answer a devlist is not itself a refusal — stock usbip exporters that only
// serve imports keep working, and the endpoint descriptors are checked again
// once the device is enumerated locally.
func guardRemote(ctx context.Context, addr string, busID string) error {
	devices, err := listExporter(ctx, addr)
	if err != nil {
		if errors.Is(err, ErrExporterAddress) {
			return err
		}
		log.Debugf("passthrough: %s did not list its devices: %s", addr, err)
		return nil
	}
	for _, device := range devices {
		if device.BusID != busID {
			continue
		}
		if refusal := device.Refusal(); refusal != "" {
			return fmt.Errorf("%w: %s", ErrIsochronous, refusal)
		}
	}
	return nil
}

func Devlist(conn io.ReadWriter) ([]RemoteDevice, error) {
	if _, err := conn.Write(EncodeDevlistRequest()); err != nil {
		return nil, fmt.Errorf("send devlist request: %w", err)
	}

	raw := make([]byte, HeaderSize+CountSize)
	if _, err := io.ReadFull(conn, raw); err != nil {
		return nil, fmt.Errorf("read devlist reply header: %w", err)
	}
	reply, err := DecodeOpCommon(raw[:HeaderSize])
	if err != nil {
		return nil, err
	}
	if reply.Version != ProtocolVersion {
		return nil, fmt.Errorf("%w: 0x%04x", ErrVersion, reply.Version)
	}
	if reply.Code != CodeRepDevlist {
		return nil, fmt.Errorf("%w: 0x%04x", ErrUnexpectedCode, reply.Code)
	}
	if reply.Status != StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrRejected, reply.Status)
	}

	count := binary.BigEndian.Uint32(raw[HeaderSize:])
	if count > maxDevices {
		return nil, fmt.Errorf("%w: exporter listed %d devices", ErrUnexpectedDevice, count)
	}
	devices := make([]RemoteDevice, 0, count)
	for index := uint32(0); index < count; index++ {
		device, err := readRemoteDevice(conn, index)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
}

func readRemoteDevice(conn io.Reader, index uint32) (RemoteDevice, error) {
	body := make([]byte, DeviceSize)
	if _, err := io.ReadFull(conn, body); err != nil {
		return RemoteDevice{}, fmt.Errorf("read usbip_usb_device %d: %w", index, err)
	}
	device, err := DecodeDevice(body)
	if err != nil {
		return RemoteDevice{}, err
	}

	interfaces := make([]Interface, 0, device.NumInterfaces)
	for number := 0; number < int(device.NumInterfaces); number++ {
		raw := make([]byte, InterfaceSize)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return RemoteDevice{}, fmt.Errorf("read usbip_usb_interface %d of %s: %w", number, device.BusID, err)
		}
		iface, err := DecodeInterface(raw)
		if err != nil {
			return RemoteDevice{}, err
		}
		interfaces = append(interfaces, iface)
	}
	return RemoteDevice{Device: device, Interfaces: interfaces}, nil
}

func Import(conn io.ReadWriter, busID string) (Device, error) {
	request, err := EncodeImportRequest(busID)
	if err != nil {
		return Device{}, err
	}
	if _, err := conn.Write(request); err != nil {
		return Device{}, fmt.Errorf("send import request for %s: %w", busID, err)
	}

	raw := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, raw); err != nil {
		return Device{}, fmt.Errorf("read import reply header: %w", err)
	}
	reply, err := DecodeOpCommon(raw)
	if err != nil {
		return Device{}, err
	}
	if reply.Version != ProtocolVersion {
		return Device{}, fmt.Errorf("%w: 0x%04x", ErrVersion, reply.Version)
	}
	if reply.Code != CodeRepImport {
		return Device{}, fmt.Errorf("%w: 0x%04x", ErrUnexpectedCode, reply.Code)
	}
	if reply.Status != StatusOK {
		return Device{}, fmt.Errorf("%w: %s: %s", ErrRejected, busID, reply.Status)
	}

	body := make([]byte, DeviceSize)
	if _, err := io.ReadFull(conn, body); err != nil {
		return Device{}, fmt.Errorf("read usbip_usb_device: %w", err)
	}
	device, err := DecodeDevice(body)
	if err != nil {
		return Device{}, err
	}
	if device.BusID != busID {
		return Device{}, fmt.Errorf("%w: requested %s, received %s", ErrUnexpectedDevice, busID, device.BusID)
	}
	if device.BusNum > 0xffff || device.DevNum > 0xffff {
		return Device{}, fmt.Errorf("%w: bus %d device %d exceeds vhci devid", ErrUnexpectedDevice, device.BusNum, device.DevNum)
	}
	if device.Speed < SpeedLow || device.Speed > SpeedSuperPlus {
		return Device{}, fmt.Errorf("%w: %s", ErrUnexpectedDevice, device.Speed)
	}
	return device, nil
}

func ParseStatus(reader io.Reader) ([]PortEntry, error) {
	var entries []PortEntry

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == "hub" {
			continue
		}
		if len(fields) != 7 {
			return nil, fmt.Errorf("%w: %q has %d columns", ErrMalformed, line, len(fields))
		}

		entry, err := parsePortEntry(fields)
		if err != nil {
			return nil, fmt.Errorf("%w: %q: %w", ErrMalformed, line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read vhci status: %w", err)
	}
	return entries, nil
}

func FreePort(entries []PortEntry, hub Hub) (uint32, error) {
	for _, entry := range entries {
		if entry.Hub == hub && entry.Status == VDevNull {
			return entry.Number, nil
		}
	}
	return 0, fmt.Errorf("%w on the %s hub", ErrNoFreePort, hub)
}

func AttachPayload(port uint32, sockfd int, devID uint32, speed Speed) string {
	return fmt.Sprintf("%d %d %d %d", port, sockfd, devID, uint32(speed))
}

// The kernel's attach_store does sockfd_lookup against the calling process fd
// table, so the descriptor written has to be one this process owns and still
// holds. Control pins it for the duration of the callback, which is where the
// sysfs write happens.
func attachSocket(conn syscall.Conn, port uint32, device Device) error {
	control, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("raw socket for port %d: %w", port, err)
	}

	var writeErr error
	if err := control.Control(func(fd uintptr) {
		writeErr = writeAttribute(attachAttribute, AttachPayload(port, int(fd), device.DevID(), device.Speed))
	}); err != nil {
		return fmt.Errorf("pin socket for port %d: %w", port, err)
	}
	if writeErr != nil {
		return fmt.Errorf("attach %s to port %d: %w", device.BusID, port, writeErr)
	}
	return nil
}

func freePort(speed Speed) (uint32, error) {
	path := filepath.Join(vhciRoot, statusAttribute)
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	entries, err := ParseStatus(file)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return FreePort(entries, speed.Hub())
}

// One write of the whole payload with no trailing newline: the sysfs store
// handler parses a single buffer, and utils.AtomicFile cannot be used because
// an attribute is written in place and never renamed over.
func writeAttribute(name string, payload string) error {
	path := filepath.Join(vhciRoot, name)
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}

	written, err := file.Write([]byte(payload))
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("write %q to %s: %w", payload, path, err)
	}
	if written != len(payload) {
		_ = file.Close()
		return fmt.Errorf("short write to %s: %d of %d bytes", path, written, len(payload))
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

func dial(ctx context.Context, addr string) (*net.TCPConn, error) {
	target, err := allowedExporter(ctx, addr)
	if err != nil {
		return nil, err
	}

	conn, err := (&net.Dialer{KeepAlive: keepAlive}).DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, fmt.Errorf("dial exporter %s: %w", target, err)
	}
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: %s", ErrNotTCP, target)
	}

	if err := tcp.SetNoDelay(true); err != nil {
		_ = tcp.Close()
		return nil, fmt.Errorf("set TCP_NODELAY on %s: %w", target, err)
	}
	if err := tcp.SetKeepAlive(true); err != nil {
		_ = tcp.Close()
		return nil, fmt.Errorf("set SO_KEEPALIVE on %s: %w", target, err)
	}
	return tcp, nil
}

// The allowed set, written out rather than left to whatever a request names:
// the registered usbip port, and a global unicast address this device has a
// route to. Loopback, the link-local range that holds the cloud metadata
// address, multicast and every other port are refused before a socket is
// opened, so an admin request cannot turn the device into a dialler. The
// resolved address is what gets dialled, not the name, so a second answer
// cannot arrive between the check and the connection.
func allowedExporter(ctx context.Context, addr string) (string, error) {
	host, port, err := net.SplitHostPort(withExporterPort(addr))
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrExporterAddress, addr)
	}
	if port != strconv.Itoa(ExporterTCP) {
		return "", fmt.Errorf("%w: %s port %s, only the usbip port %d is dialled", ErrExporterAddress, host, port, ExporterTCP)
	}

	ips, err := lookupExporter(ctx, host)
	if err != nil {
		return "", fmt.Errorf("%w: resolve %s: %w", ErrExporterAddress, host, err)
	}
	for _, ip := range ips {
		if !ip.IsGlobalUnicast() || ip.IsLinkLocalUnicast() {
			continue
		}
		if err := exporterRoute(ip); err != nil {
			continue
		}
		return net.JoinHostPort(ip.String(), port), nil
	}
	return "", fmt.Errorf("%w: %s is not a routable unicast address", ErrExporterAddress, host)
}

var lookupExporter = func(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return []net.IP{ip}, nil
	}
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

// A route lookup and nothing more: connecting a UDP socket makes the kernel
// pick the route it would use without putting a packet on the wire, which is
// the only honest answer to whether the target is reachable from here.
var exporterRoute = func(ip net.IP) error {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: ip, Port: ExporterTCP})
	if err != nil {
		return err
	}
	return conn.Close()
}

func withExporterPort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		addr = strings.TrimSuffix(strings.TrimPrefix(addr, "["), "]")
	}
	return net.JoinHostPort(addr, strconv.Itoa(ExporterTCP))
}

func parsePortEntry(fields []string) (PortEntry, error) {
	hub := Hub(fields[0])
	if hub != HubHigh && hub != HubSuper {
		return PortEntry{}, fmt.Errorf("unknown hub %q", fields[0])
	}

	var failure error
	number := func(text string, base int) uint64 {
		value, err := strconv.ParseUint(text, base, 32)
		if err != nil && failure == nil {
			failure = err
		}
		return value
	}

	entry := PortEntry{
		Hub:    hub,
		Number: uint32(number(fields[1], 10)),
		Status: VDevStatus(number(fields[2], 10)),
		Speed:  Speed(number(fields[3], 10)),
		DevID:  uint32(number(fields[4], 16)),
		SockFD: int(number(fields[5], 10)),
		BusID:  fields[6],
	}
	if failure != nil {
		return PortEntry{}, failure
	}
	return entry, nil
}
