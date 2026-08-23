//go:build linux && kernelint

package passthrough

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/service/kernelint"
)

const (
	exporterHost = "10.9.10.7"
	exporterCIDR = exporterHost + "/32"
)

func ip(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
		t.Fatalf("ip %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

// A usbip exporter that answers a devlist and then an OP_REQ_IMPORT, holding
// the import socket open because that is what attach_store needs to take a
// port. usbip-host is compiled out of every GitHub runner kernel, so the export
// side cannot be the real one; the import side under test is.
//
// It takes a private address of its own rather than 127.0.0.1 because
// allowedExporter refuses loopback and every port but 3240, and it accepts more
// than once because Attach dials twice: guardRemote reads the devlist before
// the import. An empty devlist is the honest answer from a stub that exports
// nothing, and leaves guardRemote with no matching busID to refuse.
func kernelExporter(t *testing.T, device Device) string {
	t.Helper()

	ip(t, "link", "set", "lo", "up")
	ip(t, "addr", "add", exporterCIDR, "dev", "lo")
	t.Cleanup(func() { _ = exec.Command("ip", "addr", "del", exporterCIDR, "dev", "lo").Run() })

	addr := net.JoinHostPort(exporterHost, strconv.Itoa(ExporterTCP))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	body, err := device.Encode()
	if err != nil {
		t.Fatal(err)
	}
	imported := append(OpCommon{Version: ProtocolVersion, Code: CodeRepImport, Status: StatusOK}.Encode(), body...)
	listed := append(OpCommon{Version: ProtocolVersion, Code: CodeRepDevlist, Status: StatusOK}.Encode(), make([]byte, CountSize)...)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				header := make([]byte, HeaderSize)
				if _, err := io.ReadFull(conn, header); err != nil {
					return
				}
				request, err := DecodeOpCommon(header)
				if err != nil {
					return
				}
				if request.Code == CodeReqDevlist {
					_, _ = conn.Write(listed)
					return
				}
				if _, err := io.ReadFull(conn, make([]byte, ImportRequestSize-HeaderSize)); err != nil {
					return
				}
				_, _ = conn.Write(imported)
				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()
	return addr
}

func kernelStatus(t *testing.T) []PortEntry {
	t.Helper()
	file, err := os.Open(filepath.Join(vhciRoot, statusAttribute))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	entries, err := ParseStatus(file)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func kernelPort(t *testing.T, hub Hub, number uint32) PortEntry {
	t.Helper()
	for _, entry := range kernelStatus(t) {
		if entry.Hub == hub && entry.Number == number {
			return entry
		}
	}
	t.Fatalf("port %s/%d is absent from the real vhci status", hub, number)
	return PortEntry{}
}

func TestKernelTier1VHCIAttachTakesARealPort(t *testing.T) {
	kernelint.RequireTier1VHCI(t)

	device := Device{
		Path: "/sys/devices/platform/dummy_hcd.0/usb1/1-1", BusID: "1-1",
		BusNum: 1, DevNum: 2, Speed: SpeedHigh, IDVendor: 0x1d6b, IDProduct: 0x0104,
		ConfigurationValue: 1, NumConfigurations: 1, NumInterfaces: 1,
	}
	addr := kernelExporter(t, device)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attachment, err := Attach(ctx, addr, "1-1")
	if err != nil {
		t.Fatal(err)
	}
	detached := false
	t.Cleanup(func() {
		if !detached {
			_ = Detach(attachment.Port)
		}
	})

	if attachment.Hub != HubHigh {
		t.Fatalf("hub = %s, want %s for a high-speed device", attachment.Hub, HubHigh)
	}
	if attachment.Port >= portsPerHub {
		t.Fatalf("port %d is outside the hs range", attachment.Port)
	}

	// The port leaves VDevNull the moment attach_store takes the socket. The
	// speed and devid columns, and VDevUsed itself, are filled in by the
	// enumeration the kernel then drives over that socket, which a stub
	// exporter never completes.
	if status := kernelPort(t, HubHigh, attachment.Port).Status; status == VDevNull {
		t.Fatalf("port %d is still free after the attach", attachment.Port)
	}

	if err := Detach(attachment.Port); err != nil {
		t.Fatal(err)
	}
	detached = true

	deadline := time.Now().Add(5 * time.Second)
	for {
		if kernelPort(t, HubHigh, attachment.Port).Status == VDevNull {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("port %d never returned to VDevNull", attachment.Port)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// The ss half of the same status file. VHCI_HC_PORTS is a build-time constant
// and the hs/ss split is what freePort reads out of it, so a kernel built with
// a different port count fails here rather than silently attaching to the wrong
// hub.
func TestKernelTier1VHCISuperSpeedTakesTheSuperHub(t *testing.T) {
	kernelint.RequireTier1VHCI(t)

	device := Device{
		Path: "/sys/devices/platform/dummy_hcd.0/usb2/2-1", BusID: "2-1",
		BusNum: 2, DevNum: 3, Speed: SpeedSuper, IDVendor: 0x1d6b, IDProduct: 0x0104,
		ConfigurationValue: 1, NumConfigurations: 1, NumInterfaces: 1,
	}
	addr := kernelExporter(t, device)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attachment, err := Attach(ctx, addr, "2-1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Detach(attachment.Port) })

	if attachment.Hub != HubSuper {
		t.Fatalf("hub = %s, want %s", attachment.Hub, HubSuper)
	}
	if entry := kernelPort(t, HubSuper, attachment.Port); entry.Status == VDevNull {
		t.Fatalf("port %d is still free after the attach", attachment.Port)
	}
}
