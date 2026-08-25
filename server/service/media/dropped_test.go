package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
)

// A source that overruns the device by byte rate stays inside every
// frames-per-second limit, so the rate limiter admits it and the queue fills.
// The oldest frame is then thrown away to make room, which is the right trade
// for latency - but it used to be silent, and the source was acknowledged for a
// frame nobody kept. Measured on hardware: 136KB frames at 30fps produced 141
// acks and zero errors while the achieved rate collapsed to 11fps.
func TestAFullQueueCountsTheFrameItThrowsAway(t *testing.T) {
	registry := &fakeRegistry{}
	manager := newTestManager(registry, fakeResolver{nodes: map[string]string{"uvc.cam0": "/dev/video0"}}, &blockedFactory{})
	profile := presentation.Profile{Functions: []presentation.Function{cameraFunction("cam0")}}
	if err := manager.Reconcile(context.Background(), profile, presentation.Plan{}); err != nil {
		t.Fatal(err)
	}
	defer manager.Suspend()
	waitDemand(t, registry, "uvc.cam0")

	payload := jpegFrame(t, 640, 480)
	deadline := time.Now().Add(3 * time.Second)
	for seq := 1; time.Now().Before(deadline); seq++ {
		frame := sources.MediaFrame{
			SourceID: "source", StreamID: "front", SinkID: "uvc.cam0",
			Kind: sources.MediaKindMJPEG, Sequence: uint32(seq),
			TimestampUS: uint64(time.Now().UnixMicro()), Payload: payload,
		}
		// The limiter refusing a frame is not the case under test; it means the
		// source was told, which is the behaviour that already worked.
		if err := manager.Ingest(context.Background(), frame); err != nil && !errors.Is(err, ErrFrameRate) {
			t.Fatalf("ingest %d: %v", seq, err)
		}
		worker := manager.workers["uvc.cam0"]
		worker.mu.Lock()
		dropped := worker.latency.dropped
		worker.mu.Unlock()
		if dropped > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("filled the queue against a blocked output and never counted a dropped frame")
}

// The count has to reach the sink summary, which is what the snapshot carries
// back to the source, and it has to reset with the window or it would only ever
// grow and stop meaning "drops right now".
func TestLatencySummaryCarriesDroppedFramesAndResets(t *testing.T) {
	var tracker latencyTracker
	tracker.started = time.Now().Add(-2 * latencyWindow)
	tracker.dropped = 3

	now := time.Now()
	tracker.observe(now, uint64(now.Add(-5*time.Millisecond).UnixMicro()))

	if tracker.summary.Dropped != 3 {
		t.Fatalf("summary reports %d dropped, want 3", tracker.summary.Dropped)
	}
	if tracker.dropped != 0 {
		t.Fatalf("counter is %d after the window closed, want it reset", tracker.dropped)
	}
}
