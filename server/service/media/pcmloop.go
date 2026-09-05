package media

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"NanoKVM-Server/service/sources"

	log "github.com/sirupsen/logrus"
)

// pcmOutcome classifies what one non-blocking PCM transfer returned, which is
// the only thing the capture and playback loops need in order to decide what
// to do next. It lives outside the cgo file so the policy is testable on any
// host: the capture loop needs a real gadget PCM and cannot be, and the
// playback loop below runs against pcmRing for the same reason.
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

// pcmTick is the cadence of the playback loop: one 20 ms period per tick.
const pcmTick = 20 * time.Millisecond

// SNDRV_PCM_STATE_RUNNING, the one state in which an idle tick means nothing
// worse than a host that is not draining the endpoint just now.
const pcmStateRunning = 3

// SNDRV_PCM_STATE_PREPARED, the other state an idle tick may legitimately find:
// the ring is ready and simply has not been started yet.
const pcmStatePrepared = 2

// maxWriteBurst bounds how many whole periods one tick may put into the
// playback ring while catching up. High enough to repair a run of dropped
// ticks in one go, low enough that a queue kept full by a source running fast
// cannot hold the loop past its next tick.
const maxWriteBurst = 8

// pcmRing is the playback substream as the loop sees it. Each call answers the
// raw PCM result and whether the device was still open; once Close has run
// under the loop the answer is false, and the loop leaves quietly the way a
// cancelled worker does.
type pcmRing interface {
	write(Packet) (int, bool)
	reset() (int, bool)
	probe() (int64, bool)
}

func runPlayback(ctx context.Context, ring pcmRing, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	ticker := time.NewTicker(pcmTick)
	defer ticker.Stop()
	current, err := fallback(0, 0)
	if err != nil {
		return err
	}
	streaming, failures, successes, resets := false, 0, 0, 0
	sourceActive := false
	currentSource := false
	var generation uint64
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// A Go ticker drops ticks rather than bunching them, and this loop
			// puts exactly one period into the ring per tick while the endpoint
			// takes exactly one out. So every tick the runtime drops is a period
			// the ring never gets back: the level only ever falls, and it falls
			// until the ring underruns and ALSA parks the substream. Measured on
			// the device that was about one underrun a second, and every one of
			// them costs the target host the silence its reset has to prime with.
			// A tick is therefore allowed to write more than one period, but only
			// while the source is actually behind - the queue holds something
			// only when it is, since the browser produces one packet per tick
			// too - so the level is repaired with real audio and never with
			// silence this loop invented.
			//
			// Whether this tick has already reset a broken ring. One recovery
			// per tick is the whole budget: see the underrun branch below.
			recovered := false
		burst:
			for range maxWriteBurst {
				// Only take a new packet once the one in hand has gone out. A tick
				// whose write found no room used to take the next packet anyway
				// and overwrite the packet it was still holding, so every full
				// ring cost the target host 20 ms of audio that the browser had
				// already been told was accepted.
				if !currentSource {
					select {
					case frame := <-frames:
						changed := frame.Generation != generation
						if frame.Reset || len(frame.Data) == 0 {
							current, err = fallback(0, 0)
							if err != nil {
								return err
							}
						} else {
							current = frame
							currentSource = true
						}
						if changed {
							generation = frame.Generation
							rc, open := ring.reset()
							if !open {
								return nil
							}
							if rc < 0 {
								return fmt.Errorf("reset PCM: return %d", rc)
							}
							streaming, failures, successes = false, 0, 0
							demand(sources.Demand{})
						}
					default:
					}
				}
				rc, open := ring.write(current)
				if !open {
					return nil
				}
				switch classifyPCM(rc) {
				case pcmTransferred:
					failures, resets = 0, 0
					successes++
					if !streaming && successes > 4 {
						streaming = true
						demand(sources.Demand{Streaming: true, Since: time.Now().UTC()})
					}
					if currentSource != sourceActive {
						sourceActive = currentSource
						source(sourceActive)
					}
					current, err = fallback(0, 0)
					if err != nil {
						return err
					}
					currentSource = false
					if len(frames) > 0 {
						continue
					}
					break burst
				case pcmIdle:
					// The ring is full: the target host is not draining the
					// endpoint. The packet stays in current and goes out on a
					// later tick, so a host that pauses costs no audio.
					failures++
					successes = 0
					// A host that paused looks exactly like a ring the loop can no
					// longer write to, and only the kernel's own view of the
					// substream tells them apart. Say so once a minute, and only
					// when the answer is not the ordinary one, so a microphone
					// that has gone quiet for a reason is not buried in a log full
					// of a microphone that is merely unused.
					if failures%50 == 0 {
						probe, open := ring.probe()
						if !open {
							return nil
						}
						switch {
						case probe < 0:
							if failures%3000 == 0 {
								log.Warnf("media playback idle for %d ticks: kernel status unavailable (errno %d), so a parked ring cannot be told from a full one", failures, -probe)
							}
						case probe&0xff != pcmStateRunning && probe&0xff != pcmStatePrepared:
							// A ring that is merely full is a host that paused,
							// and waiting is the right answer. A ring the kernel
							// says is parked is not: it will not move another
							// sample until something prepares it. Every route
							// into that state has to end here, whether or not
							// this loop was the one that saw it happen - the bug
							// this guards is a microphone silent for the rest of
							// the session while the browser still sends and the
							// server still acknowledges, and a ring nobody can
							// explain is worth resetting rather than leaving
							// parked.
							log.Warnf("media playback idle for %d ticks in kernel state %d, avail %d: resetting the parked ring", failures, probe&0xff, probe>>8)
							parked, open := ring.reset()
							if !open {
								return nil
							}
							if parked < 0 {
								return fmt.Errorf("reset parked playback PCM: return %d", parked)
							}
							failures = 0
						}
					}
					if streaming && failures >= 3 {
						streaming = false
						demand(sources.Demand{})
						if sourceActive {
							sourceActive = false
							source(false)
						}
					}
					break burst
				}
				// An underrun. This loop is paced by a Go ticker and the endpoint
				// by the host's clock, so a tick that lands late empties the ring
				// and ALSA parks the substream in XRUN. Nothing put it back, so the
				// first late tick silenced the microphone for the rest of the
				// binding while every frame was still acknowledged - the queue
				// filled, the writer never drained it, and hw_ptr never moved
				// again.
				successes = 0
				if recovered {
					// The ring was prepared and primed a moment ago and broke
					// again on the very next write, so this is not a late tick, it
					// is a stream nothing on this side can move: the host has
					// stopped taking audio, or the bus is suspended under it.
					// Resetting again within the same tick used to get the same
					// answer eight times over, and with the priming writes each
					// reset makes that ran the write ioctl thousands of times a
					// second against a host that wanted none of it - 98% of the
					// board's one core with nothing streaming, and the camera pump
					// starved behind it. A real microphone burns nothing while the
					// host is not pulling; this one waits for the next tick.
					break burst
				}
				recovered = true
				resets++
				if resets > 50 {
					return fmt.Errorf("write PCM: return %d after %d resets", rc, resets)
				}
				if resets == 1 || resets%25 == 0 {
					log.Warnf("media playback ring reset after write returned %d (%d in a row)", rc, resets)
				}
				reset, open := ring.reset()
				if !open {
					return nil
				}
				if reset < 0 {
					return fmt.Errorf("reset playback PCM: return %d", reset)
				}
			}
		}
	}
}
