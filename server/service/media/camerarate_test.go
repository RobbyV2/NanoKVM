package media

import (
	"testing"
	"time"

	"NanoKVM-Server/service/sources"
)

// The browser sends camera frames through a two-frame window: it holds at most
// two unacknowledged, so frames reach the server in pairs rather than one every
// period, and a wireless link moves each pair a few milliseconds either way.
// That is the arrival pattern the admission bucket actually sees, and the burst
// it used to carry - two, borrowed from the queue depth - had exactly zero
// margin against it. The pair drained both tokens and the next pair arrived
// with precisely 2.0 refilled, so any jitter at all landed one of them early
// and refused a frame the source had every right to send. A refused frame is
// dropped, which is a visible stutter rather than an overload being shed.
func TestCameraBucketAbsorbsAckGatedJitter(t *testing.T) {
	const (
		fps     = 30
		seconds = 20
		jitter  = 5 * time.Millisecond
	)
	worker := &worker{spec: SlotSpec{Kind: sources.KindCamera}}
	demand := sources.Demand{Streaming: true, Width: 640, Height: 480, FPS: fps}

	start := time.Now()
	period := time.Second / fps
	refused, sent := 0, 0
	// Pairs leave every two periods, and the jitter alternates sign so the long
	// run rate stays exactly the negotiated one: only the bunching moves.
	for pair := range seconds * fps / 2 {
		skew := jitter
		if pair%2 == 1 {
			skew = -jitter
		}
		at := start.Add(time.Duration(pair) * 2 * period).Add(skew)
		for range 2 {
			sent++
			if !worker.allowFrame(at, demand) {
				refused++
			}
		}
	}
	if refused != 0 {
		t.Fatalf("refused %d of %d frames sent at the negotiated rate", refused, sent)
	}
}

// The refill still bounds a source that simply will not stop, which is the only
// thing the bucket is there for.
func TestCameraBucketStillBoundsARunawaySource(t *testing.T) {
	const fps = 30
	worker := &worker{spec: SlotSpec{Kind: sources.KindCamera}}
	demand := sources.Demand{Streaming: true, Width: 640, Height: 480, FPS: fps}

	start := time.Now()
	admitted := 0
	// Ten times the negotiated rate for two seconds.
	for tick := range 10 * fps * 2 {
		at := start.Add(time.Duration(tick) * time.Second / (10 * fps))
		if worker.allowFrame(at, demand) {
			admitted++
		}
	}
	// Two seconds of refill plus one bucketful, and not a frame more.
	if ceiling := 2*fps + max(videoRateBurst, fps/2); admitted > ceiling {
		t.Fatalf("admitted %d frames in two seconds, want no more than %d", admitted, ceiling)
	}
}
