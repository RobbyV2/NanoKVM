package media

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func recordingTrace() (*uvcTrace, *[]string) {
	var lines []string
	trace := &uvcTrace{slot: "uvc.cam0", emit: func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}}
	return trace, &lines
}

// One stream as the host drives it: the probe and commit control transfers,
// the STREAMON, the start, the first frame the kernel sent, the STREAMOFF.
// Every line names its edge, stamps it on the kernel's clock, and after the
// STREAMON carries the offset from it, so the device log can be read as one
// timeline.
func TestUVCTraceWritesOneLinePerEdgeOnTheStreamOnClock(t *testing.T) {
	trace, lines := recordingTrace()
	const base = int64(5527_000_000_000)

	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventSetup, request: 0x87, selector: uvcVSProbeControl, length: 26, kernelNS: base, handledNS: base + 1_500_000})
	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventData, selector: uvcVSCommitControl, frame: 1, interval: 333333, length: 26, kernelNS: base + 200_000_000, handledNS: base + 200_400_000})
	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventStreamOn, kernelNS: base + 1_000_000_000, handledNS: base + 1_002_000_000})
	trace.started(uvcStart{base + 1_003_000_000, base + 1_004_000_000, base + 1_007_000_000, base + 1_009_000_000, base + 1_013_000_000, base + 1_015_000_000}, nil)
	trace.event(uvcMark{kind: uvcMarkFirstSent, length: 16384, kernelNS: base + 1_040_000_000, handledNS: base + 1_045_000_000})
	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventStreamOff, kernelNS: base + 13_700_000_000, handledNS: base + 13_702_000_000})
	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventSetup, request: 0x81, selector: uvcVSProbeControl, length: 26, kernelNS: base + 20_000_000_000, handledNS: base + 20_001_000_000})

	want := []string{
		"uvc uvc.cam0: event setup GET_DEF probe len 26 at 5527.000, handled 1 ms later",
		"uvc uvc.cam0: event data commit frame 1 interval 333333 len 26 at 5527.200, handled 0 ms later",
		"uvc uvc.cam0: event streamon at 5528.000, streamon+0 ms, handled 2 ms later",
		"uvc uvc.cam0: streamon handled at 5528.015, streamon+15 ms: s_fmt 1 ms, reqbufs 3 ms, mmap 2 ms, qbuf 4 ms, streamon 2 ms; first frame queued at 5528.013, streamon+13 ms",
		"uvc uvc.cam0: first frame sent at 5528.040, streamon+40 ms, 16384 bytes, seen by this side 5 ms later",
		"uvc uvc.cam0: event streamoff at 5540.700, streamon+12700 ms, stream taken down 2 ms later",
		"uvc uvc.cam0: event setup GET_CUR probe len 26 at 5547.000, handled 1 ms later",
	}
	if len(*lines) != len(want) {
		t.Fatalf("%d lines, want %d:\n%s", len(*lines), len(want), strings.Join(*lines, "\n"))
	}
	for i, line := range *lines {
		if line != want[i] {
			t.Fatalf("line %d = %q, want %q", i, line, want[i])
		}
	}
}

// A start that fails says which ioctl it got to, and a stream this side took
// down on its own says why.
func TestUVCTraceNamesTheFailedStepAndTheGroundReason(t *testing.T) {
	trace, lines := recordingTrace()
	const base = int64(100_000_000_000)
	trace.event(uvcMark{kind: uvcMarkEvent, eventType: uvcEventStreamOn, kernelNS: base, handledNS: base})
	trace.started(uvcStart{base + 1_000_000, base + 2_000_000, 0, 0, 0, 0}, errors.New("errno 12"))
	trace.event(uvcMark{kind: uvcMarkGrounded, eventType: uvcGroundENODEV, kernelNS: base + 500_000_000, handledNS: base + 500_000_000})

	want := []string{
		"uvc uvc.cam0: event streamon at 100.000, streamon+0 ms, handled 0 ms later",
		"uvc uvc.cam0: streamon failed at 100.002, streamon+2 ms after step 1 of s_fmt, reqbufs, mmap, qbuf, streamon: errno 12",
		"uvc uvc.cam0: stream taken down by this side at 100.500, streamon+500 ms: the gadget marked the queue disconnected; it is raised again at the committed geometry",
	}
	if len(*lines) != len(want) {
		t.Fatalf("%d lines, want %d:\n%s", len(*lines), len(want), strings.Join(*lines, "\n"))
	}
	for i, line := range want {
		if (*lines)[i] != line {
			t.Fatalf("line %d = %q, want %q", i, (*lines)[i], line)
		}
	}
}
