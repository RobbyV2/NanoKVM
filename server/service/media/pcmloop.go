package media

import "syscall"

// pcmOutcome classifies what one non-blocking PCM transfer returned, which is
// the only thing the capture and playback loops need in order to decide what
// to do next. It lives outside the cgo file so the policy is testable on any
// host: the loops themselves need a real gadget PCM and cannot be.
type pcmOutcome int

const (
	// pcmTransferred: a whole period moved.
	pcmTransferred pcmOutcome = iota
	// pcmIdle: the endpoint had nothing to give or nowhere to put it this
	// tick. That is the ordinary quiet host, not a fault, and the loop only
	// waits for the next tick.
	pcmIdle
	// pcmBroken: the ring under- or overran, or the host tore the stream
	// down. ALSA parks the substream and will not move another sample until
	// something prepares it again, so the loop must reset before it can make
	// progress. Doing nothing here is what left a microphone silent for the
	// rest of its binding after the first late tick.
	pcmBroken
)

func classifyPCM(rc int) pcmOutcome {
	switch {
	case rc == 0:
		return pcmTransferred
	case rc == -int(syscall.EAGAIN):
		return pcmIdle
	default:
		return pcmBroken
	}
}

// maxCaptureDrain bounds how many whole 20 ms periods one tick may take from a
// capture device before yielding. High enough to swallow any realistic
// scheduling delay, low enough that a device handing back frames forever
// cannot hold the loop.
const maxCaptureDrain = 8
