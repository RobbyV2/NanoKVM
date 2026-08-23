package vnc

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"NanoKVM-Server/service/hid"
)

func testJPEG(marker byte) []byte {
	jpeg := bytes.Repeat([]byte{marker}, 512)
	jpeg[0], jpeg[1] = 0xff, 0xd8
	return jpeg
}

func startServer(t *testing.T, server *Server) net.Addr {
	t.Helper()

	if server.Screen == nil {
		server.Screen = func() (uint16, uint16, uint16, int) { return 640, 480, 80, 30 }
	}
	if server.ReadJPEG == nil {
		server.ReadJPEG = func(uint16, uint16, uint16) ([]byte, int) { return testJPEG(0x41), 0 }
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() { _ = server.Serve(listener) }()
	return listener.Addr()
}

type testClient struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func dial(t *testing.T, addr net.Addr) *testClient {
	t.Helper()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatalf("dial: %s", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	return &testClient{t: t, conn: conn, r: bufio.NewReader(conn)}
}

func (c *testClient) read(n int) []byte {
	c.t.Helper()

	buffer := make([]byte, n)
	if _, err := io.ReadFull(c.r, buffer); err != nil {
		c.t.Fatalf("read %d bytes: %s", n, err)
	}
	return buffer
}

func (c *testClient) write(data []byte) {
	c.t.Helper()

	if _, err := c.conn.Write(data); err != nil {
		c.t.Fatalf("write: %s", err)
	}
}

func (c *testClient) handshake(username string, password string) (uint16, uint16) {
	c.t.Helper()

	c.read(12)
	c.write([]byte(protocolVersion))

	types := c.read(1)
	c.read(int(types[0]))
	c.write([]byte{securityVeNCrypt})

	c.read(2)
	c.write([]byte{0, 2})
	c.read(1)
	count := c.read(1)
	c.read(int(count[0]) * 4)
	c.write(binary.BigEndian.AppendUint32(nil, venCryptPlain))

	credentials := binary.BigEndian.AppendUint32(nil, uint32(len(username)))
	credentials = binary.BigEndian.AppendUint32(credentials, uint32(len(password)))
	credentials = append(credentials, username...)
	credentials = append(credentials, password...)
	c.write(credentials)

	if result := c.read(4); binary.BigEndian.Uint32(result) != 0 {
		length := binary.BigEndian.Uint32(c.read(4))
		c.t.Fatalf("authentication rejected: %s", c.read(int(length)))
	}

	c.write([]byte{1})
	init := c.read(24)
	c.read(int(binary.BigEndian.Uint32(init[20:24])))

	return binary.BigEndian.Uint16(init[0:2]), binary.BigEndian.Uint16(init[2:4])
}

func (c *testClient) setEncodings(encodings ...int32) {
	c.t.Helper()

	message := []byte{msgSetEncodings, 0}
	message = binary.BigEndian.AppendUint16(message, uint16(len(encodings)))
	for _, encoding := range encodings {
		message = binary.BigEndian.AppendUint32(message, uint32(encoding))
	}
	c.write(message)
}

func (c *testClient) requestUpdate(incremental byte) {
	c.t.Helper()

	message := []byte{msgFramebufferUpdateRequest, incremental}
	message = binary.BigEndian.AppendUint16(message, 0)
	message = binary.BigEndian.AppendUint16(message, 0)
	message = binary.BigEndian.AppendUint16(message, 1920)
	message = binary.BigEndian.AppendUint16(message, 1080)
	c.write(message)
}

func (c *testClient) readUpdate() []byte {
	c.t.Helper()

	header := c.read(16)
	if header[0] != 0 || binary.BigEndian.Uint16(header[2:4]) != 1 {
		c.t.Fatalf("unexpected update header % x", header)
	}
	if int32(binary.BigEndian.Uint32(header[12:16])) != encodingTight {
		c.t.Fatalf("encoding = %d, want Tight", int32(binary.BigEndian.Uint32(header[12:16])))
	}
	if control := c.read(1); control[0] != tightJPEG {
		c.t.Fatalf("tight compression control = %#x, want %#x", control[0], tightJPEG)
	}

	first := c.read(1)[0]
	length := int(first & 0x7f)
	if first&0x80 != 0 {
		second := c.read(1)[0]
		length |= int(second&0x7f) << 7
		if second&0x80 != 0 {
			length |= int(c.read(1)[0]) << 14
		}
	}
	return c.read(length)
}

func TestSessionDeliversHardwareJPEGAndSurvivesMissingHID(t *testing.T) {
	restore := authenticate
	authenticate = func(username string, password string) (string, bool, error) {
		return username, username == "admin" && password == "hunter2hunter2", nil
	}
	t.Cleanup(func() { authenticate = restore })

	jpeg := testJPEG(0x5a)
	addr := startServer(t, &Server{
		ReadJPEG: func(uint16, uint16, uint16) ([]byte, int) { return jpeg, 0 },
	})

	client := dial(t, addr)
	width, height := client.handshake("admin", "hunter2hunter2")
	if width != 640 || height != 480 {
		t.Fatalf("framebuffer = %dx%d, want 640x480", width, height)
	}

	client.setEncodings(encodingTight, encodingDesktopSize)
	client.requestUpdate(0)
	if got := client.readUpdate(); !bytes.Equal(got, jpeg) {
		t.Fatal("first update did not carry the encoder's JPEG unchanged")
	}

	// /dev/hidg* is absent here, so every HID write fails the way it does while
	// USB passthrough holds the UDC. The session must not notice.
	client.write([]byte{msgKeyEvent, 1, 0, 0, 0, 0, 0, 'a'})
	client.write([]byte{msgKeyEvent, 0, 0, 0, 0, 0, 0, 'a'})
	pointer := []byte{msgPointerEvent, rfbButtonLeft}
	pointer = binary.BigEndian.AppendUint16(pointer, 320)
	pointer = binary.BigEndian.AppendUint16(pointer, 240)
	client.write(pointer)

	client.requestUpdate(0)
	if got := client.readUpdate(); !bytes.Equal(got, jpeg) {
		t.Fatal("session stopped serving frames after HID writes failed")
	}
}

func TestSessionWritesHIDReportsOnceTheDeviceReturns(t *testing.T) {
	device := hid.HID2
	if err := os.Remove(device); err != nil && !os.IsNotExist(err) {
		t.Skipf("cannot control %s in this environment: %s", device, err)
	}

	// Hid is a process-wide singleton that caches one descriptor per device and
	// only drops it when a write fails. On the board that is enough: when the
	// UDC goes away the character device stops accepting reports and the failed
	// write reopens it. Here the device is a regular file, and writing to an
	// unlinked regular file never fails, so a descriptor this test left open on
	// a previous run in the same process would keep absorbing the reports and
	// the freshly created file would stay empty. Drop it so the device is
	// genuinely absent, and drop it again afterwards so the next test starts
	// from the same place.
	hid.GetHid().Close()
	t.Cleanup(func() {
		hid.GetHid().Close()
		_ = os.Remove(device)
	})

	addr := startServer(t, &Server{AllowNone: true})
	client := dial(t, addr)

	client.read(12)
	client.write([]byte(protocolVersion))
	types := client.read(1)
	client.read(int(types[0]))
	client.write([]byte{securityNone})
	client.read(4)
	client.write([]byte{1})
	init := client.read(24)
	client.read(int(binary.BigEndian.Uint32(init[20:24])))
	client.setEncodings(encodingTight)

	pointer := []byte{msgPointerEvent, 0}
	pointer = binary.BigEndian.AppendUint16(pointer, 0)
	pointer = binary.BigEndian.AppendUint16(pointer, 0)
	client.write(pointer)
	time.Sleep(500 * time.Millisecond)

	created, err := os.OpenFile(device, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		t.Skipf("cannot create %s in this environment: %s", device, err)
	}
	_ = created.Close()

	pointer = []byte{msgPointerEvent, 0}
	pointer = binary.BigEndian.AppendUint16(pointer, 639)
	pointer = binary.BigEndian.AppendUint16(pointer, 479)
	client.write(pointer)

	want := []byte{0x00, 0xff, 0x7f, 0xff, 0x7f, 0x00}
	deadline := time.Now().Add(5 * time.Second)
	for {
		reports, readErr := os.ReadFile(device)
		if readErr == nil && bytes.Contains(reports, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no HID report reached %s after it came back: % x (%v)", device, reports, readErr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestSessionAnswersARequestThatArrivesDuringAnUpdate(t *testing.T) {
	jpeg := testJPEG(0x33)
	jpeg = append(jpeg, bytes.Repeat([]byte{0x44}, 3<<20)...)

	addr := startServer(t, &Server{
		AllowNone: true,
		Screen:    func() (uint16, uint16, uint16, int) { return 640, 480, 80, 30 },
		ReadJPEG:  func(uint16, uint16, uint16) ([]byte, int) { return jpeg, 0 },
	})
	client := dial(t, addr)

	client.read(12)
	client.write([]byte(protocolVersion))
	types := client.read(1)
	client.read(int(types[0]))
	client.write([]byte{securityNone})
	client.read(4)
	client.write([]byte{1})
	init := client.read(24)
	client.read(int(binary.BigEndian.Uint32(init[20:24])))
	client.setEncodings(encodingTight)

	// The payload is larger than the socket buffer, so the second request is
	// parsed while the first update is still being written.
	client.requestUpdate(0)
	time.Sleep(500 * time.Millisecond)
	client.requestUpdate(0)

	if got := client.readUpdate(); !bytes.Equal(got, jpeg) {
		t.Fatal("first update was truncated")
	}
	if got := client.readUpdate(); !bytes.Equal(got, jpeg) {
		t.Fatal("the request that arrived during the first update was never answered")
	}
}

func TestSessionRefusesClientWithoutTight(t *testing.T) {
	addr := startServer(t, &Server{AllowNone: true})
	client := dial(t, addr)

	client.read(12)
	client.write([]byte(protocolVersion))
	types := client.read(1)
	client.read(int(types[0]))
	client.write([]byte{securityNone})
	client.read(4)
	client.write([]byte{1})
	init := client.read(24)
	client.read(int(binary.BigEndian.Uint32(init[20:24])))

	client.setEncodings(0, 1, 16)
	client.requestUpdate(0)

	if _, err := client.r.Read(make([]byte, 1)); err == nil {
		t.Fatal("a client without Tight support was served instead of disconnected")
	}
}
