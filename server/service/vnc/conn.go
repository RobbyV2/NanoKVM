package vnc

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputcontrol"
	"NanoKVM-Server/service/vm/jiggler"

	log "github.com/sirupsen/logrus"
)

const (
	msgSetPixelFormat           = 0
	msgSetEncodings             = 2
	msgFramebufferUpdateRequest = 3
	msgKeyEvent                 = 4
	msgPointerEvent             = 5
	msgClientCutText            = 6

	manualPreemptTimeout = 2 * time.Second
	writeTimeout         = 30 * time.Second
)

var errNoTightSupport = errors.New("client does not support the Tight encoding")

type inputEvent struct {
	kind     inputcontrol.ManualReportKind
	report   []byte
	held     bool
	cooldown bool
}

type conn struct {
	server *Server
	net    net.Conn
	reader *bufio.Reader

	manual   *inputcontrol.ManualSession
	input    chan inputEvent
	keyboard chan hid.QueuedReport
	mouse    chan hid.QueuedReport
	inputs   sync.WaitGroup
	workers  sync.WaitGroup

	keys    *keyboardState
	pointer pointerState

	mutex       sync.Mutex
	tight       bool
	desktopSize bool
	requests    uint64
	full        bool
	width       uint16
	height      uint16
	wake        chan struct{}
}

func newConn(server *Server, netConn net.Conn) *conn {
	return &conn{
		server:   server,
		net:      netConn,
		reader:   bufio.NewReaderSize(netConn, 4096),
		manual:   inputcontrol.NewManualSession(controlmode.GetManager(), inputcontrol.GetCoordinator()),
		input:    make(chan inputEvent, 200),
		keyboard: make(chan hid.QueuedReport, 200),
		mouse:    make(chan hid.QueuedReport, 200),
		keys:     newKeyboardState(),
		wake:     make(chan struct{}, 1),
	}
}

func (c *conn) serve() {
	defer c.close()

	c.server.pump.acquire()
	defer c.server.pump.release()

	if err := c.handshake(); err != nil {
		log.Debugf("vnc handshake with %s failed: %s", c.net.RemoteAddr(), err)
		return
	}

	c.inputs.Add(1)
	go func() {
		defer c.inputs.Done()
		c.runInput()
	}()

	c.workers.Add(2)
	go func() {
		defer c.workers.Done()
		hid.GetHid().KeyboardReports(c.keyboard)
	}()
	go func() {
		defer c.workers.Done()
		hid.GetHid().MouseReports(c.mouse)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		if err := c.readMessages(); err != nil && !errors.Is(err, io.EOF) {
			log.Debugf("vnc client %s disconnected: %s", c.net.RemoteAddr(), err)
		}
	}()

	if err := c.writeUpdates(ctx); err != nil {
		log.Debugf("vnc update loop for %s ended: %s", c.net.RemoteAddr(), err)
	}
}

func (c *conn) handshake() error {
	_ = c.net.SetDeadline(time.Now().Add(time.Minute))
	defer func() { _ = c.net.SetDeadline(time.Time{}) }()

	if err := negotiateVersion(c.reader, c.net); err != nil {
		return err
	}

	creds, err := negotiateSecurity(c.reader, c.net, c.server.AllowNone)
	if err != nil {
		return err
	}

	authErr := c.server.verify(c.net.RemoteAddr(), creds)
	if err := writeSecurityResult(c.net, authErr); err != nil {
		return err
	}
	if authErr != nil {
		return authErr
	}

	if err := readClientInit(c.reader); err != nil {
		return err
	}

	width, height, _, _ := c.server.Screen()
	if first := c.server.pump.wait(firstFrameWait); first.version != 0 {
		width, height = first.width, first.height
	} else if width == 0 || height == 0 {
		width, height = fallbackWidth, fallbackHeight
	}

	c.width = width
	c.height = height
	return writeServerInit(c.net, width, height, "NanoKVM")
}

func (c *conn) readMessages() error {
	for {
		messageType, err := c.reader.ReadByte()
		if err != nil {
			return err
		}

		switch messageType {
		case msgSetPixelFormat:
			if err := c.readPixelFormat(); err != nil {
				return err
			}
		case msgSetEncodings:
			if err := c.readEncodings(); err != nil {
				return err
			}
		case msgFramebufferUpdateRequest:
			if err := c.readUpdateRequest(); err != nil {
				return err
			}
		case msgKeyEvent:
			if err := c.readKeyEvent(); err != nil {
				return err
			}
		case msgPointerEvent:
			if err := c.readPointerEvent(); err != nil {
				return err
			}
		case msgClientCutText:
			if err := c.discardCutText(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown client message type %d", messageType)
		}
	}
}

func (c *conn) readPixelFormat() error {
	var message [19]byte
	if _, err := io.ReadFull(c.reader, message[:]); err != nil {
		return err
	}

	bitsPerPixel := message[3]
	trueColour := message[6]
	if trueColour == 0 || (bitsPerPixel != 16 && bitsPerPixel != 32) {
		return fmt.Errorf("unsupported pixel format: %d bpp, true colour %d", bitsPerPixel, trueColour)
	}
	return nil
}

func (c *conn) readEncodings() error {
	var header [3]byte
	if _, err := io.ReadFull(c.reader, header[:]); err != nil {
		return err
	}

	count := int(binary.BigEndian.Uint16(header[1:3]))
	encodings := make([]byte, count*4)
	if _, err := io.ReadFull(c.reader, encodings); err != nil {
		return err
	}

	tight := false
	desktopSize := false
	for index := 0; index < count; index++ {
		switch int32(binary.BigEndian.Uint32(encodings[index*4:])) {
		case encodingTight:
			tight = true
		case encodingDesktopSize:
			desktopSize = true
		}
	}

	c.mutex.Lock()
	c.tight = tight
	c.desktopSize = desktopSize
	c.mutex.Unlock()

	if !tight {
		return errNoTightSupport
	}
	return nil
}

func (c *conn) readUpdateRequest() error {
	var message [9]byte
	if _, err := io.ReadFull(c.reader, message[:]); err != nil {
		return err
	}

	c.mutex.Lock()
	c.requests++
	if message[0] == 0 {
		c.full = true
	}
	c.mutex.Unlock()

	select {
	case c.wake <- struct{}{}:
	default:
	}
	return nil
}

func (c *conn) readKeyEvent() error {
	var message [7]byte
	if _, err := io.ReadFull(c.reader, message[:]); err != nil {
		return err
	}

	report := c.keys.key(binary.BigEndian.Uint32(message[3:7]), message[0] != 0)
	if report == nil {
		return nil
	}

	c.submit(inputEvent{kind: inputcontrol.ManualKeyboard, report: report, held: keyboardHeld(report), cooldown: true})
	return nil
}

func (c *conn) readPointerEvent() error {
	var message [5]byte
	if _, err := io.ReadFull(c.reader, message[:]); err != nil {
		return err
	}

	c.mutex.Lock()
	width, height := c.width, c.height
	c.mutex.Unlock()

	report := c.pointer.pointer(
		message[0],
		binary.BigEndian.Uint16(message[1:3]),
		binary.BigEndian.Uint16(message[3:5]),
		width,
		height,
	)
	c.submit(inputEvent{
		kind:     inputcontrol.ManualAbsoluteMouse,
		report:   report,
		held:     report[0] != 0,
		cooldown: report[0] != 0 || report[5] != 0,
	})
	return nil
}

func (c *conn) discardCutText() error {
	var header [7]byte
	if _, err := io.ReadFull(c.reader, header[:]); err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(header[3:7])
	_, err := io.CopyN(io.Discard, c.reader, int64(length))
	return err
}

// submit hands the event to the input worker. Reserving manual control can
// block for seconds, and the framebuffer update requests share this read loop,
// so the reservation must never happen here.
func (c *conn) submit(event inputEvent) {
	select {
	case c.input <- event:
	default:
		log.Debugf("vnc input dropped for %s: queue is full", c.net.RemoteAddr())
	}
}

func (c *conn) runInput() {
	for event := range c.input {
		c.queue(event)
	}
}

func (c *conn) queue(event inputEvent) {
	queue := c.mouse
	if event.kind == inputcontrol.ManualKeyboard {
		queue = c.keyboard
	}

	ctx, cancel := context.WithTimeout(context.Background(), manualPreemptTimeout)
	defer cancel()

	reservation, err := c.manual.ReserveWithCooldown(ctx, event.kind, event.held, event.cooldown, c.server.AllowInput)
	if err != nil {
		if !errors.Is(err, inputcontrol.ErrManualInputBlocked) {
			log.Errorf("vnc input failed to acquire control: %s", err)
		}
		return
	}

	queued := hid.QueuedReport{
		Data:               event.report,
		Execute:            c.manual.Execute,
		Complete:           reservation.Complete,
		ResetKeyboard:      func() { c.manual.Reset(inputcontrol.ManualKeyboard) },
		ResetRelativeMouse: func() { c.manual.Reset(inputcontrol.ManualRelativeMouse) },
		ResetAbsoluteMouse: func() { c.manual.Reset(inputcontrol.ManualAbsoluteMouse) },
	}
	queue <- queued
	jiggler.GetJiggler().Update()
}

func keyboardHeld(report []byte) bool {
	if report[0] != 0 {
		return true
	}
	for _, key := range report[2:] {
		if key != 0 {
			return true
		}
	}
	return false
}

// writeUpdates answers one framebuffer update request at a time. Requests are
// counted rather than flagged so a request that arrives while an update is
// being written is not swallowed by the write that was already in flight.
func (c *conn) writeUpdates(ctx context.Context) error {
	var served uint64
	var sent uint64

	for {
		current, changed := c.server.pump.snapshot()

		c.mutex.Lock()
		requests := c.requests
		supported := c.desktopSize
		pending := c.tight && requests != served && current.version != 0
		resized := pending && (current.width != c.width || current.height != c.height)
		update := pending && !resized && (c.full || current.version != sent)
		if resized {
			c.width, c.height = current.width, current.height
			c.full = true
		}
		if update {
			c.full = false
		}
		c.mutex.Unlock()

		switch {
		case resized:
			if !supported {
				return fmt.Errorf("screen resized to %dx%d and the client cannot follow", current.width, current.height)
			}
			_ = c.net.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := writeDesktopSize(c.net, current.width, current.height); err != nil {
				return err
			}
			served = requests
			sent = 0
			continue
		case update:
			_ = c.net.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := writeTightJPEG(c.net, current); err != nil {
				return err
			}
			served = requests
			sent = current.version
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		case <-c.wake:
		}
	}
}

func (c *conn) close() {
	_ = c.net.Close()
	close(c.input)
	c.inputs.Wait()
	close(c.keyboard)
	close(c.mouse)
	c.workers.Wait()
	c.manual.Close()
}
