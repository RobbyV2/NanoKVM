package media

import (
	"fmt"
	"strings"

	"NanoKVM-Server/service/startup"
)

// The edges of a camera stream as this side sees them, written to the kernel
// log so a device's dmesg shows where the time between the host's alt 1 and
// the first frame goes. One line per edge, none per frame: every UVC event
// the gadget hands over (with its name and, for a control transfer, what it
// asked), the STREAMON handled with the cost of each ioctl on the way, the
// first frame the kernel finished sending, and the STREAMOFF handled. Each
// line carries the kernel's own timestamp for the edge, the lag before this
// side had handled it, and the offset from the STREAMON event once one has
// been seen, so the host's negotiation, this side's ioctls and the wire can
// be told apart on one clock.
//
// The C side records the marks (see nk_mark in output_linux.go) and Go
// drains them after every step; this file only turns them into lines and is
// pure so a test can read them.

const (
	uvcMarkEvent     = 1
	uvcMarkFirstSent = 2
	uvcMarkGrounded  = 3

	uvcGroundPollErr = 1
	uvcGroundENODEV  = 2
	uvcGroundRefmt   = 3

	// From linux/usb/g_uvc.h: V4L2_EVENT_PRIVATE_START + 0..5.
	uvcEventConnect    = 0x08000000
	uvcEventDisconnect = 0x08000001
	uvcEventStreamOn   = 0x08000002
	uvcEventStreamOff  = 0x08000003
	uvcEventSetup      = 0x08000004
	uvcEventData       = 0x08000005

	uvcVSProbeControl  = 0x01
	uvcVSCommitControl = 0x02
)

type uvcMark struct {
	kind      uint32
	eventType uint32
	request   uint32
	selector  uint32
	length    uint32
	frame     uint32
	interval  uint32
	kernelNS  int64
	handledNS int64
}

// uvcStart is nk_uvc_start's clock: entry, after S_FMT, after REQBUFS, after
// the mmaps, after the QBUFs, after STREAMON. A zero stamp is a step that was
// not reached.
type uvcStart [6]int64

type uvcTrace struct {
	slot string
	emit func(format string, args ...any)
	// The kernel's timestamp of the last STREAMON event, the reference every
	// later line is offset from; cleared by a STREAMOFF so the next
	// negotiation is not measured against a stream that has ended.
	streamOnNS int64
}

func newUVCTrace(slot string) *uvcTrace {
	return &uvcTrace{slot: slot, emit: startup.Kmsg}
}

var uvcRequestNames = map[uint32]string{
	0x01: "SET_CUR", 0x81: "GET_CUR", 0x82: "GET_MIN", 0x83: "GET_MAX",
	0x84: "GET_RES", 0x85: "GET_LEN", 0x86: "GET_INFO", 0x87: "GET_DEF",
}

func uvcSelectorName(selector uint32) string {
	switch selector {
	case uvcVSProbeControl:
		return "probe"
	case uvcVSCommitControl:
		return "commit"
	}
	return fmt.Sprintf("selector 0x%02x", selector)
}

func uvcEventName(eventType uint32) string {
	switch eventType {
	case uvcEventConnect:
		return "connect"
	case uvcEventDisconnect:
		return "disconnect"
	case uvcEventStreamOn:
		return "streamon"
	case uvcEventStreamOff:
		return "streamoff"
	case uvcEventSetup:
		return "setup"
	case uvcEventData:
		return "data"
	}
	return fmt.Sprintf("event 0x%08x", eventType)
}

func monoSeconds(ns int64) string {
	return fmt.Sprintf("%d.%03d", ns/1e9, (ns%1e9)/1e6)
}

func msBetween(from, to int64) int64 {
	return (to - from) / 1e6
}

// The offset from the STREAMON event, or nothing when none has been seen.
func (t *uvcTrace) sinceStreamOn(ns int64) string {
	if t.streamOnNS == 0 {
		return ""
	}
	return fmt.Sprintf(", streamon+%d ms", msBetween(t.streamOnNS, ns))
}

func (t *uvcTrace) event(m uvcMark) {
	switch m.kind {
	case uvcMarkEvent:
		t.uvcEvent(m)
	case uvcMarkFirstSent:
		t.emit("uvc %s: first frame sent at %s%s, %d bytes, seen by this side %d ms later",
			t.slot, monoSeconds(m.kernelNS), t.sinceStreamOn(m.kernelNS), m.length, msBetween(m.kernelNS, m.handledNS))
	case uvcMarkGrounded:
		reason := "unknown"
		switch m.eventType {
		case uvcGroundPollErr:
			reason = "poll reported an error on the queue"
		case uvcGroundENODEV:
			reason = "the gadget marked the queue disconnected"
		case uvcGroundRefmt:
			reason = "the host committed a new geometry"
		}
		t.emit("uvc %s: stream taken down by this side at %s%s: %s; it is raised again at the committed geometry",
			t.slot, monoSeconds(m.kernelNS), t.sinceStreamOn(m.kernelNS), reason)
	}
}

func (t *uvcTrace) uvcEvent(m uvcMark) {
	var detail strings.Builder
	switch m.eventType {
	case uvcEventStreamOn:
		t.streamOnNS = m.kernelNS
	case uvcEventSetup:
		name, ok := uvcRequestNames[m.request]
		if !ok {
			name = fmt.Sprintf("request 0x%02x", m.request)
		}
		fmt.Fprintf(&detail, " %s %s len %d", name, uvcSelectorName(m.selector), m.length)
	case uvcEventData:
		fmt.Fprintf(&detail, " %s frame %d interval %d len %d", uvcSelectorName(m.selector), m.frame, m.interval, m.length)
	}
	handled := "handled"
	if m.eventType == uvcEventStreamOff || m.eventType == uvcEventDisconnect {
		handled = "stream taken down"
	}
	t.emit("uvc %s: event %s%s at %s%s, %s %d ms later",
		t.slot, uvcEventName(m.eventType), detail.String(), monoSeconds(m.kernelNS), t.sinceStreamOn(m.kernelNS), handled, msBetween(m.kernelNS, m.handledNS))
	if m.eventType == uvcEventStreamOff || m.eventType == uvcEventDisconnect {
		t.streamOnNS = 0
	}
}

// started is the STREAMON handled: the host's alt 1 is held at its status
// stage until the STREAMON ioctl at the end of this sequence, so these are
// milliseconds the host spent waiting on this side.
func (t *uvcTrace) started(stamps uvcStart, err error) {
	if err != nil {
		reached := 0
		for i, stamp := range stamps {
			if stamp != 0 {
				reached = i
			}
		}
		t.emit("uvc %s: streamon failed at %s%s after step %d of s_fmt, reqbufs, mmap, qbuf, streamon: %s",
			t.slot, monoSeconds(stamps[reached]), t.sinceStreamOn(stamps[reached]), reached, err)
		return
	}
	t.emit("uvc %s: streamon handled at %s%s: s_fmt %d ms, reqbufs %d ms, mmap %d ms, qbuf %d ms, streamon %d ms; first frame queued at %s%s",
		t.slot, monoSeconds(stamps[5]), t.sinceStreamOn(stamps[5]),
		msBetween(stamps[0], stamps[1]), msBetween(stamps[1], stamps[2]), msBetween(stamps[2], stamps[3]),
		msBetween(stamps[3], stamps[4]), msBetween(stamps[4], stamps[5]),
		monoSeconds(stamps[4]), t.sinceStreamOn(stamps[4]))
}
