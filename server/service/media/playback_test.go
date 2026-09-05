package media

import (
	"context"
	"sync"
	"syscall"
	"testing"
	"time"

	"NanoKVM-Server/service/sources"
)

// A playback ring that answers every write from a script and counts what the
// loop asked of it. Resets always succeed, as they do on the device: prepare
// empties the ring and the priming writes go into an empty ring.
type scriptedRing struct {
	mu     sync.Mutex
	answer func(write int) int
	writes int
	resets int
}

func (r *scriptedRing) write(Packet) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	return r.answer(r.writes), true
}

func (r *scriptedRing) reset() (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resets++
	return 0, true
}

func (r *scriptedRing) probe() (int64, bool) { return pcmStateRunning, true }

func (r *scriptedRing) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writes, r.resets
}

func silentPeriod(int, int) (Packet, error) {
	return Packet{Data: make([]byte, pcmPacketBytes)}, nil
}

// Runs the playback loop against ring for the window, then cancels it and
// reports how it ended.
func playFor(t *testing.T, ring pcmRing, frames <-chan Packet, window time.Duration, source func(bool)) error {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runPlayback(ctx, ring, frames, silentPeriod, func(sources.Demand) {}, source) }()
	timer := time.NewTimer(window)
	defer timer.Stop()
	select {
	case err := <-done:
		cancel()
		return err
	case <-timer.C:
	}
	cancel()
	select {
	case err := <-done:
		return err
	case <-time.After(stopTimeout):
		t.Fatal("the playback loop did not leave on cancel")
		return nil
	}
}

// Measured on the device with nothing streaming and the host's bus suspended:
// the microphone's writer at 98% of the board's one core, the write ioctl
// failing with EPIPE about 5000 times a second, and the camera pump starved
// behind it. Every failed write reset the ring and tried again within the same
// tick, eight times over, and each reset primed the ring with four more
// writes. A host that is not pulling audio has to cost nothing: one write, one
// recovery and one retry per tick, then the loop waits for the next tick.
func TestAWriteThatKeepsFailingWaitsForTheNextTick(t *testing.T) {
	ring := &scriptedRing{answer: func(int) int { return -int(syscall.EPIPE) }}
	window := 10 * pcmTick
	if err := playFor(t, ring, make(chan Packet), window, func(bool) {}); err != nil {
		t.Fatalf("runPlayback() = %v, want nil: a ring that stays broken for %s is not yet a dead one", err, window)
	}
	writes, resets := ring.counts()
	// The runtime drops ticks rather than bunching them, so a tick already
	// pending when the loop starts is the only one this window can gain.
	ticks := int(window/pcmTick) + 1
	if writes > 2*ticks {
		t.Fatalf("%d writes in %s, want at most two a tick (%d): the loop is spinning inside the tick", writes, window, 2*ticks)
	}
	if resets > ticks {
		t.Fatalf("%d resets in %s, want at most one a tick (%d)", resets, window, ticks)
	}
	if writes < 2 || resets < 1 {
		t.Fatalf("%d writes and %d resets: the ring was never recovered and retried at all", writes, resets)
	}
}

// The other half of the budget: a ring that broke because one tick landed
// late is recovered once and the packet in hand goes out on the retry, so an
// ordinary underrun still costs the host nothing but the silence the reset
// primes with.
func TestAnUnderrunIsRecoveredOnceAndThePacketStillGoesOut(t *testing.T) {
	ring := &scriptedRing{answer: func(write int) int {
		if write == 1 {
			return -int(syscall.EPIPE)
		}
		return 0
	}}
	frames := make(chan Packet, 1)
	frames <- Packet{Data: make([]byte, pcmPacketBytes)}
	var mu sync.Mutex
	accepted := false
	source := func(active bool) {
		mu.Lock()
		defer mu.Unlock()
		accepted = accepted || active
	}
	if err := playFor(t, ring, frames, 5*pcmTick, source); err != nil {
		t.Fatalf("runPlayback() = %v, want nil", err)
	}
	writes, resets := ring.counts()
	if resets != 1 {
		t.Fatalf("%d resets for one underrun, want exactly one", resets)
	}
	if writes < 2 {
		t.Fatalf("%d writes: the packet was never retried after the reset", writes)
	}
	mu.Lock()
	defer mu.Unlock()
	if !accepted {
		t.Fatal("the browser's packet never reached the ring after the recovery")
	}
}

// A ring that will not recover is an error the supervisor reopens on, with its
// own backoff, rather than a loop that retries forever. That verdict is paced
// by the same tick, so it takes a second to reach, and the loop leaves the
// same way it always did.
func TestARingThatNeverRecoversIsGivenUpAtTheTickRate(t *testing.T) {
	if testing.Short() {
		t.Skip("waits out fifty ticks")
	}
	ring := &scriptedRing{answer: func(int) int { return -int(syscall.EPIPE) }}
	start := time.Now()
	err := playFor(t, ring, make(chan Packet), 3*time.Second, func(bool) {})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("runPlayback() = nil, want the reset ceiling reported")
	}
	if elapsed < 40*pcmTick {
		t.Fatalf("gave up after %s, want the fifty resets spread over their own ticks", elapsed)
	}
}
