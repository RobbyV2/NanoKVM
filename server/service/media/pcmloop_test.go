package media

import (
	"syscall"
	"testing"
)

// The playback loop used to treat every non-zero return the same way: count a
// failure and try the identical write again next tick. An underrun leaves the
// substream parked in XRUN, so that write fails forever and the microphone
// goes silent for the rest of its binding while the source is still told every
// frame was accepted. The two cases have to be told apart.
func TestClassifyPCMSeparatesAQuietHostFromABrokenStream(t *testing.T) {
	cases := []struct {
		name string
		rc   int
		want pcmOutcome
	}{
		{"a whole period moved", 0, pcmTransferred},
		{"nothing to do this tick", -int(syscall.EAGAIN), pcmIdle},
		{"the ring under- or overran", -int(syscall.EPIPE), pcmBroken},
		{"the host suspended the stream", -int(syscall.ESTRPIPE), pcmBroken},
		{"the node went away", -int(syscall.ENODEV), pcmBroken},
		{"the stream is in the wrong state", -int(syscall.EBADFD), pcmBroken},
	}
	for _, test := range cases {
		if got := classifyPCM(test.rc); got != test.want {
			t.Errorf("%s: classifyPCM(%d) = %d, want %d", test.name, test.rc, got, test.want)
		}
	}
}

// EAGAIN must never ask for a reset: a host that stops draining the endpoint
// is ordinary, and resetting on it would throw away audio already queued.
func TestClassifyPCMNeverResetsOnEAGAIN(t *testing.T) {
	if classifyPCM(-int(syscall.EAGAIN)) == pcmBroken {
		t.Fatal("a quiet host must not be recovered from")
	}
}
