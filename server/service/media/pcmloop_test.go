package media

import (
	"syscall"
	"testing"
	"time"

	"NanoKVM-Server/service/sources"
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

// A microphone's frames arrive over a wireless link from a browser pacing
// itself by its own AudioContext clock, so they bunch. The bucket's long run
// rate still holds them to 50 a second; the burst only decides how much of that
// bunching survives the door. At audioQueueDepth - four packets, 80 ms - it did
// not survive: a 20 second recording against the device refused 22 perfectly
// good frames as "frame rate exceeded", each one 20 ms the host never heard.
func TestAudioBurstAbsorbsAWholeRoundTripOfBunching(t *testing.T) {
	w := &worker{spec: SlotSpec{Kind: sources.KindMicrophone}}
	start := time.Now()
	// A quarter second of frames that all arrive at once, which is what one
	// stalled and then released wireless round trip looks like from here.
	admitted := 0
	for i := 0; i < 12; i++ {
		if w.allowFrame(start, sources.Demand{Streaming: true}) {
			admitted++
		}
	}
	if admitted != 12 {
		t.Fatalf("admitted %d of 12 frames arriving together, want all of them", admitted)
	}
	// The long run rate is still the point of the bucket: a source that keeps
	// going without waiting has to be refused once the burst is spent.
	spent := 0
	for i := 0; i < audioRateBurst*2; i++ {
		if !w.allowFrame(start, sources.Demand{Streaming: true}) {
			spent++
		}
	}
	if spent == 0 {
		t.Fatal("a source that never pauses was never refused, so the rate is not bounded at all")
	}
	// And a second of real time buys back a second of frames, no more.
	w.allowFrame(start.Add(time.Second), sources.Demand{Streaming: true})
	if w.rateTokens > audioRateBurst {
		t.Fatalf("tokens %v exceed the burst ceiling %d", w.rateTokens, audioRateBurst)
	}
}

// The playback loop's idle branch waits for the next tick and nothing else, so
// whatever nk_pcm_write reports as EAGAIN had better be a ring the loop can
// write to later. A parked substream is not: ALSA will not move another sample
// until something prepares it. Classifying the kernel's EPIPE as pcmBroken is
// what sends it down the reset path, and it is the difference between one late
// tick costing 20 ms and one late tick costing the rest of the session - on the
// device, an unrecovered XRUN sat with hw_ptr == appl_ptr for 541 seconds while
// a browser pushed 50 packets a second at it.
func TestAParkedRingIsBrokenAndNotIdle(t *testing.T) {
	for _, rc := range []int{-int(syscall.EPIPE), -int(syscall.ESTRPIPE), -int(syscall.ENODEV)} {
		if got := classifyPCM(rc); got != pcmBroken {
			t.Errorf("classifyPCM(%d) = %d, want pcmBroken: the idle branch never resets", rc, got)
		}
	}
}
