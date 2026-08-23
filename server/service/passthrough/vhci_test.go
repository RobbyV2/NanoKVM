package passthrough

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const statusWithFreePorts = `hub port sta spd dev      sockfd local_busid
hs  0000 006 003 00020003 000009 1-1
hs  0001 004 000 00000000 000000 0-0
hs	0002	004	000	00000000	000000	0-0
hs   0003  004  000  00000000  000000  0-0
hs  0004 004 000 00000000 000000 0-0
hs  0005 004 000 00000000 000000 0-0
hs  0006 004 000 00000000 000000 0-0
hs  0007 004 000 00000000 000000 0-0
ss  0008 006 005 00030002 000010 2-1
ss  0009 004 000 00000000 000000 0-0
ss  0010 004 000 00000000 000000 0-0
ss  0011 004 000 00000000 000000 0-0
ss  0012 004 000 00000000 000000 0-0
ss  0013 004 000 00000000 000000 0-0
ss  0014 004 000 00000000 000000 0-0
ss  0015 004 000 00000000 000000 0-0
`

const statusFull = `hub port sta spd dev      sockfd local_busid
hs  0000 006 003 00020003 000009 1-1
hs  0001 006 003 00020004 000010 1-2
hs  0002 006 002 00020005 000011 1-3
hs  0003 006 003 00020006 000012 1-4
hs  0004 005 003 00020007 000013 1-5
hs  0005 006 003 00020008 000014 1-6
hs  0006 007 003 00020009 000015 1-7
hs  0007 006 003 0002000a 000016 1-8
ss  0008 006 005 00030002 000017 2-1
ss  0009 006 005 00030003 000018 2-2
ss  0010 006 005 00030004 000019 2-3
ss  0011 006 005 00030005 000020 2-4
ss  0012 006 005 00030006 000021 2-5
ss  0013 006 005 00030007 000022 2-6
ss  0014 006 005 00030008 000023 2-7
ss  0015 006 005 00030009 000024 2-8
`

func swapRoot(t *testing.T) string {
	t.Helper()

	previous := vhciRoot
	vhciRoot = t.TempDir()
	t.Cleanup(func() { vhciRoot = previous })
	return vhciRoot
}

func touch(t *testing.T, path string) string {
	t.Helper()

	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	return path
}

func loopback(t *testing.T) *net.TCPConn {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	accepted, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	t.Cleanup(func() { accepted.Close() })

	return conn.(*net.TCPConn)
}

func descriptor(t *testing.T, conn *net.TCPConn) int {
	t.Helper()

	control, err := conn.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn: %v", err)
	}

	var fd int
	if err := control.Control(func(raw uintptr) { fd = int(raw) }); err != nil {
		t.Fatalf("Control: %v", err)
	}
	return fd
}

type exchange struct {
	replies *bytes.Reader
	sent    bytes.Buffer
}

func (e *exchange) Read(p []byte) (int, error)  { return e.replies.Read(p) }
func (e *exchange) Write(p []byte) (int, error) { return e.sent.Write(p) }

func newExchange(replies []byte) *exchange {
	return &exchange{replies: bytes.NewReader(replies)}
}

func TestParseStatusReadsTheRealTable(t *testing.T) {
	entries, err := ParseStatus(strings.NewReader(statusWithFreePorts))
	if err != nil {
		t.Fatalf("ParseStatus: %v", err)
	}
	if len(entries) != 16 {
		t.Fatalf("parsed %d entries, want 16 with the header skipped", len(entries))
	}

	want := PortEntry{Hub: HubHigh, Number: 0, Status: VDevUsed, Speed: SpeedHigh, DevID: 0x00020003, SockFD: 9, BusID: "1-1"}
	if entries[0] != want {
		t.Fatalf("entry 0 = %+v, want %+v", entries[0], want)
	}

	// Tabs and doubled spaces are the same table.
	for _, index := range []int{2, 3} {
		entry := entries[index]
		if entry.Hub != HubHigh || entry.Number != uint32(index) || entry.Status != VDevNull || entry.BusID != "0-0" {
			t.Fatalf("entry %d = %+v, want a free hs port", index, entry)
		}
	}
	if entries[8].Hub != HubSuper || entries[8].Speed != SpeedSuper || entries[8].DevID != 0x00030002 {
		t.Fatalf("entry 8 = %+v, want the used ss port", entries[8])
	}
}

func TestFreePortMatchesTheSpeedClass(t *testing.T) {
	entries, err := ParseStatus(strings.NewReader(statusWithFreePorts))
	if err != nil {
		t.Fatalf("ParseStatus: %v", err)
	}

	for _, want := range []struct {
		speed Speed
		port  uint32
	}{
		{SpeedLow, 1},
		{SpeedFull, 1},
		{SpeedHigh, 1},
		{SpeedSuper, 9},
		{SpeedSuperPlus, 9},
	} {
		got, err := FreePort(entries, want.speed.Hub())
		if err != nil {
			t.Fatalf("FreePort for a %s device: %v", want.speed, err)
		}
		if got != want.port {
			t.Fatalf("FreePort for a %s device = %d, want %d", want.speed, got, want.port)
		}
	}
}

func TestFreePortOnAFullTable(t *testing.T) {
	entries, err := ParseStatus(strings.NewReader(statusFull))
	if err != nil {
		t.Fatalf("ParseStatus: %v", err)
	}
	if len(entries) != 16 {
		t.Fatalf("parsed %d entries, want 16", len(entries))
	}

	for _, hub := range []Hub{HubHigh, HubSuper} {
		if _, err := FreePort(entries, hub); !errors.Is(err, ErrNoFreePort) {
			t.Fatalf("FreePort on a full %s hub = %v, want ErrNoFreePort", hub, err)
		}
	}
}

func TestParseStatusRejectsMalformedLines(t *testing.T) {
	for _, table := range []string{
		"hub port sta spd dev      sockfd local_busid\nhs 0000 004 000 00000000 000000\n",
		"hub port sta spd dev      sockfd local_busid\nxx 0000 004 000 00000000 000000 0-0\n",
		"hub port sta spd dev      sockfd local_busid\nhs 0000 00x 000 00000000 000000 0-0\n",
	} {
		if _, err := ParseStatus(strings.NewReader(table)); !errors.Is(err, ErrMalformed) {
			t.Fatalf("ParseStatus(%q) = %v, want ErrMalformed", table, err)
		}
	}
}

func TestFreePortReadsTheStatusAttribute(t *testing.T) {
	root := swapRoot(t)
	if err := os.WriteFile(filepath.Join(root, statusAttribute), []byte(statusWithFreePorts), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}

	port, err := freePort(SpeedHigh)
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	if port != 1 {
		t.Fatalf("freePort = %d, want 1", port)
	}
}

func TestAttachPayloadIsFourSpaceSeparatedDecimals(t *testing.T) {
	got := AttachPayload(3, 7, Device{BusNum: 2, DevNum: 3}.DevID(), SpeedHigh)
	if want := "3 7 131075 3"; got != want {
		t.Fatalf("AttachPayload = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, "\n\r\t") {
		t.Fatalf("AttachPayload = %q, want no whitespace but the three separators", got)
	}
	if fields := strings.Split(got, " "); len(fields) != 4 {
		t.Fatalf("AttachPayload split into %d fields, want 4", len(fields))
	}
}

func TestAttachWritesItsOwnDescriptorInOneWrite(t *testing.T) {
	root := swapRoot(t)
	attach := touch(t, filepath.Join(root, attachAttribute))
	conn := loopback(t)

	device := Device{BusID: "1-1", BusNum: 2, DevNum: 3, Speed: SpeedHigh}
	if err := attachSocket(conn, 5, device); err != nil {
		t.Fatalf("attachSocket: %v", err)
	}

	raw, err := os.ReadFile(attach)
	if err != nil {
		t.Fatalf("read attach: %v", err)
	}

	want := fmt.Sprintf("5 %d 131075 3", descriptor(t, conn))
	if string(raw) != want {
		t.Fatalf("attach = %q, want %q", raw, want)
	}
	if len(raw) != len(want) {
		t.Fatalf("attach is %d bytes, want the payload's %d with no newline", len(raw), len(want))
	}
}

func TestAttachPropagatesASysfsFailure(t *testing.T) {
	swapRoot(t)

	err := attachSocket(loopback(t), 5, Device{BusID: "1-1", Speed: SpeedHigh})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("attachSocket with no attach attribute = %v, want a wrapped ErrNotExist", err)
	}
}

func TestDetachWritesABarePortNumber(t *testing.T) {
	root := swapRoot(t)
	detach := touch(t, filepath.Join(root, detachAttribute))

	if err := Detach(7); err != nil {
		t.Fatalf("Detach: %v", err)
	}

	raw, err := os.ReadFile(detach)
	if err != nil {
		t.Fatalf("read detach: %v", err)
	}
	if string(raw) != "7" {
		t.Fatalf("detach = %q, want %q", raw, "7")
	}
}

func TestImportSendsOneRequestAndDecodesTheReply(t *testing.T) {
	want := sampleDevice()
	body, err := want.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	reply := OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusOK}.Encode()
	conn := newExchange(append(reply, body...))

	got, err := Import(conn, "1-1")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if got != want {
		t.Fatalf("Import = %+v, want %+v", got, want)
	}

	request, err := EncodeImportRequest("1-1")
	if err != nil {
		t.Fatalf("EncodeImportRequest: %v", err)
	}
	if !bytes.Equal(conn.sent.Bytes(), request) {
		t.Fatalf("sent % x, want % x", conn.sent.Bytes(), request)
	}
}

func TestImportRejectsANonZeroStatus(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusDeviceBusy}.Encode()

	_, err := Import(newExchange(reply), "1-1")
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("Import = %v, want ErrRejected", err)
	}
	if !strings.Contains(err.Error(), StatusDeviceBusy.String()) {
		t.Fatalf("Import error %q does not name the status", err)
	}
}

func TestImportRejectsAnUnexpectedCode(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion, Code: 0x0004, Status: StatusOK}.Encode()

	if _, err := Import(newExchange(reply), "1-1"); !errors.Is(err, ErrUnexpectedCode) {
		t.Fatalf("Import = %v, want ErrUnexpectedCode", err)
	}
}

func TestImportRejectsAnUnexpectedVersion(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion + 1, Code: CodeRepImport, Status: StatusOK}.Encode()

	if _, err := Import(newExchange(reply), "1-1"); !errors.Is(err, ErrVersion) {
		t.Fatalf("Import = %v, want ErrVersion", err)
	}
}

func TestImportRejectsUnexpectedDeviceMetadata(t *testing.T) {
	tests := map[string]Device{
		"busid":  {BusID: "2-1", BusNum: 1, DevNum: 2, Speed: SpeedHigh},
		"busnum": {BusID: "1-1", BusNum: 0x10000, DevNum: 2, Speed: SpeedHigh},
		"devnum": {BusID: "1-1", BusNum: 1, DevNum: 0x10000, Speed: SpeedHigh},
		"speed":  {BusID: "1-1", BusNum: 1, DevNum: 2, Speed: SpeedUnknown},
	}

	for name, device := range tests {
		t.Run(name, func(t *testing.T) {
			body, err := device.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			reply := OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusOK}.Encode()

			if _, err := Import(newExchange(append(reply, body...)), "1-1"); !errors.Is(err, ErrUnexpectedDevice) {
				t.Fatalf("Import = %v, want ErrUnexpectedDevice", err)
			}
		})
	}
}

func TestImportRejectsATruncatedDevice(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusOK}.Encode()

	if _, err := Import(newExchange(append(reply, make([]byte, DeviceSize-1)...)), "1-1"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Import = %v, want an unexpected EOF", err)
	}
}

func TestImportRejectsABadBusIDBeforeWriting(t *testing.T) {
	conn := newExchange(nil)

	if _, err := Import(conn, "1-1; reboot"); !errors.Is(err, ErrBusID) {
		t.Fatalf("Import = %v, want ErrBusID", err)
	}
	if conn.sent.Len() != 0 {
		t.Fatalf("sent %d bytes for a rejected busid", conn.sent.Len())
	}
}

func TestWithExporterPortDefaultsTo3240(t *testing.T) {
	for addr, want := range map[string]string{
		"192.168.1.10":    "192.168.1.10:3240",
		"127.0.0.1:12345": "127.0.0.1:12345",
		"exporter.local":  "exporter.local:3240",
		"[fe80::1]":       "[fe80::1]:3240",
		"[fe80::1]:3240":  "[fe80::1]:3240",
	} {
		if got := withExporterPort(addr); got != want {
			t.Fatalf("withExporterPort(%q) = %q, want %q", addr, got, want)
		}
	}
}

func devlistReply(t *testing.T, devices ...RemoteDevice) []byte {
	t.Helper()

	out := OpCommon{Version: ProtocolVersion, Code: CodeRepDevlist, Status: StatusOK}.Encode()
	count := make([]byte, CountSize)
	binary.BigEndian.PutUint32(count, uint32(len(devices)))
	out = append(out, count...)

	for _, device := range devices {
		device.NumInterfaces = uint8(len(device.Interfaces))
		body, err := device.Device.Encode()
		if err != nil {
			t.Fatalf("Encode %s: %v", device.BusID, err)
		}
		out = append(out, body...)
		for _, iface := range device.Interfaces {
			out = append(out, iface.Encode()...)
		}
	}
	return out
}

func allowAnyRoute(t *testing.T) {
	t.Helper()

	previous := exporterRoute
	exporterRoute = func(net.IP) error { return nil }
	t.Cleanup(func() { exporterRoute = previous })
}

func stubList(t *testing.T, devices []RemoteDevice, err error) *int {
	t.Helper()

	calls := 0
	previous := listExporter
	listExporter = func(context.Context, string) ([]RemoteDevice, error) {
		calls++
		return devices, err
	}
	t.Cleanup(func() { listExporter = previous })
	return &calls
}

func TestDevlistDecodesDevicesAndInterfaces(t *testing.T) {
	webcam := RemoteDevice{Device: sampleDevice(), Interfaces: []Interface{
		{Class: 0x0e, SubClass: 0x01},
		{Class: 0x0e, SubClass: 0x02},
	}}
	webcam.BusID = "1-1"
	bare := RemoteDevice{Device: sampleDevice()}
	bare.BusID = "1-2"

	conn := newExchange(devlistReply(t, webcam, bare))
	devices, err := Devlist(conn)
	if err != nil {
		t.Fatalf("Devlist: %v", err)
	}
	if !bytes.Equal(conn.sent.Bytes(), EncodeDevlistRequest()) {
		t.Fatalf("sent % x, want % x", conn.sent.Bytes(), EncodeDevlistRequest())
	}
	if len(devices) != 2 {
		t.Fatalf("decoded %d devices, want 2", len(devices))
	}
	if devices[0].BusID != "1-1" || len(devices[0].Interfaces) != 2 {
		t.Fatalf("device 0 = %+v, want 1-1 with two interfaces", devices[0])
	}
	if devices[0].Interfaces[1] != (Interface{Class: 0x0e, SubClass: 0x02}) {
		t.Fatalf("interface 1 = %+v, want the streaming interface", devices[0].Interfaces[1])
	}
	if devices[1].BusID != "1-2" || len(devices[1].Interfaces) != 0 {
		t.Fatalf("device 1 = %+v, want 1-2 with no interfaces", devices[1])
	}
}

func TestDevlistRejectsAnExporterThatDoesNotSpeakIt(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusOK}.Encode()

	if _, err := Devlist(newExchange(append(reply, 0, 0, 0, 0))); !errors.Is(err, ErrUnexpectedCode) {
		t.Fatalf("Devlist = %v, want ErrUnexpectedCode", err)
	}
}

func TestDevlistRefusesAnAbsurdDeviceCount(t *testing.T) {
	reply := OpCommon{Version: ProtocolVersion, Code: CodeRepDevlist, Status: StatusOK}.Encode()
	count := make([]byte, CountSize)
	binary.BigEndian.PutUint32(count, maxDevices+1)

	_, err := Devlist(newExchange(append(reply, count...)))
	if !errors.Is(err, ErrUnexpectedDevice) {
		t.Fatalf("Devlist = %v, want ErrUnexpectedDevice", err)
	}
}

func TestAttachRefusesAnUnrelayableDeviceBeforeTakingAPort(t *testing.T) {
	root := swapRoot(t)
	if err := os.WriteFile(filepath.Join(root, statusAttribute), []byte(statusWithFreePorts), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	touch(t, filepath.Join(root, attachAttribute))

	headset := RemoteDevice{Device: sampleDevice(), Interfaces: []Interface{{Class: 0x01, SubClass: 0x02}}}
	headset.BusID = "1-1"
	stubList(t, []RemoteDevice{headset}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Attach(ctx, "10.0.0.5", "1-1", false)
	if !errors.Is(err, ErrIsochronous) {
		t.Fatalf("Attach = %v, want ErrIsochronous", err)
	}
	if !strings.Contains(err.Error(), "1-1") || !strings.Contains(err.Error(), "046d:c31c") {
		t.Fatalf("Attach error %q does not name the device", err)
	}

	raw, readErr := os.ReadFile(filepath.Join(root, attachAttribute))
	if readErr != nil {
		t.Fatalf("read attach: %v", readErr)
	}
	if len(raw) != 0 {
		t.Fatalf("attach = %q, want nothing written for a refused device", raw)
	}
}

func TestAttachStillWorksWithAnExporterThatHasNoDevlist(t *testing.T) {
	swapRoot(t)
	allowAnyRoute(t)
	calls := stubList(t, nil, ErrUnexpectedCode)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Attach(ctx, "192.0.2.7", "1-1", false)
	if *calls != 1 {
		t.Fatalf("devlist attempted %d times, want 1", *calls)
	}
	if errors.Is(err, ErrIsochronous) || errors.Is(err, ErrUnexpectedCode) {
		t.Fatalf("Attach = %v, want the import to be tried anyway", err)
	}
}

func TestExporterPolicyAllowsOnlyRoutableUnicastOnTheUsbipPort(t *testing.T) {
	allowAnyRoute(t)

	for _, addr := range []string{
		"127.0.0.1",
		"[::1]",
		"169.254.169.254",
		"[fe80::1]",
		"224.0.0.1",
		"0.0.0.0",
		"255.255.255.255",
		"10.0.0.5:8080",
		"10.0.0.5:22",
	} {
		if _, err := allowedExporter(context.Background(), addr); !errors.Is(err, ErrExporterAddress) {
			t.Fatalf("allowedExporter(%q) = %v, want ErrExporterAddress", addr, err)
		}
	}

	for addr, want := range map[string]string{
		"10.0.0.5":           "10.0.0.5:3240",
		"192.168.1.10:3240":  "192.168.1.10:3240",
		"[2001:db8::1]":      "[2001:db8::1]:3240",
		"[2001:db8::1]:3240": "[2001:db8::1]:3240",
	} {
		got, err := allowedExporter(context.Background(), addr)
		if err != nil {
			t.Fatalf("allowedExporter(%q): %v", addr, err)
		}
		if got != want {
			t.Fatalf("allowedExporter(%q) = %q, want %q", addr, got, want)
		}
	}
}

func TestExporterPolicyRefusesATargetWithNoRoute(t *testing.T) {
	previous := exporterRoute
	exporterRoute = func(net.IP) error { return errors.New("network is unreachable") }
	t.Cleanup(func() { exporterRoute = previous })

	if _, err := allowedExporter(context.Background(), "10.0.0.5"); !errors.Is(err, ErrExporterAddress) {
		t.Fatalf("allowedExporter of an unroutable target = %v, want ErrExporterAddress", err)
	}
}

func TestExporterPolicyDialsTheAddressItChecked(t *testing.T) {
	allowAnyRoute(t)

	previous := lookupExporter
	lookupExporter = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.0.2.7")}, nil
	}
	t.Cleanup(func() { lookupExporter = previous })

	got, err := allowedExporter(context.Background(), "exporter.local")
	if err != nil {
		t.Fatalf("allowedExporter: %v", err)
	}
	if got != "192.0.2.7:3240" {
		t.Fatalf("allowedExporter = %q, want the checked address and not the name", got)
	}
}
