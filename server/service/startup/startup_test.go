package startup

import (
	"errors"
	"testing"
	"time"
)

func reset() {
	mu.Lock()
	defer mu.Unlock()
	failed = map[string]string{}
	order = nil
}

func TestRunRecordsSuccessAndError(t *testing.T) {
	reset()

	Run("ok", time.Second, func() error { return nil })
	Run("broken", time.Second, func() error { return errors.New("no such device") })

	report := Report()
	if len(report) != 2 {
		t.Fatalf("got %d steps, want 2", len(report))
	}
	if report[0].Name != "ok" || report[0].Error != "" {
		t.Fatalf("got %+v, want ok with no error", report[0])
	}
	if report[1].Error != "no such device" {
		t.Fatalf("got %q, want %q", report[1].Error, "no such device")
	}
}

func TestRunSurvivesAPanic(t *testing.T) {
	reset()

	Run("panicking", time.Second, func() error { panic("gadget vanished") })

	report := Report()
	if len(report) != 1 || report[0].Error != "panic: gadget vanished" {
		t.Fatalf("got %+v, want the panic recorded", report)
	}
}

// The listener must come up even when a hardware call never returns, so Run
// returns on its budget and leaves the step running.
func TestRunReturnsOnBudget(t *testing.T) {
	reset()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	started := time.Now()
	Run("stuck", 50*time.Millisecond, func() error {
		<-release
		return nil
	})

	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Run blocked for %s on a step that never returns", elapsed)
	}
	report := Report()
	if len(report) != 1 || report[0].Error != "timed out after 50ms" {
		t.Fatalf("got %+v, want the timeout recorded", report)
	}
}

// The way out is the mirror of the way in: a step that never returns costs its
// budget and nothing more, the steps after it still run, and the results say
// which one took the time.
func TestStopRunsEveryStepInOrderAndAbandonsTheStuckOne(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	var order []string
	started := time.Now()
	results := Stop(
		Step{Name: "media", Budget: time.Second, Run: func() error { order = append(order, "media"); return nil }},
		Step{Name: "vision", Budget: 50 * time.Millisecond, Run: func() error { <-release; return nil }},
		Step{Name: "passthrough", Budget: time.Second, Run: func() error { order = append(order, "passthrough"); return errors.New("no session") }},
		Step{Name: "gadget", Budget: time.Second, Run: func() error { panic("controller gone") }},
	)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Stop blocked for %s on a step that never returns", elapsed)
	}
	if len(order) != 2 || order[0] != "media" || order[1] != "passthrough" {
		t.Fatalf("steps ran as %v, want media then passthrough with the stuck one skipped over", order)
	}
	if len(results) != 4 {
		t.Fatalf("got %d results, want 4", len(results))
	}
	if r := results[0]; r.Name != "media" || r.Err != nil || r.Abandoned {
		t.Fatalf("media = %+v, want done", r)
	}
	if r := results[1]; !r.Abandoned || r.Elapsed != 50*time.Millisecond || r.String() != "vision did not finish within 50ms" {
		t.Fatalf("vision = %+v (%s), want abandoned on its budget", r, r)
	}
	if r := results[2]; r.Abandoned || r.Err == nil || r.Err.Error() != "no session" {
		t.Fatalf("passthrough = %+v, want its error carried", r)
	}
	if r := results[3]; r.Err == nil || r.Err.Error() != "panic: controller gone" {
		t.Fatalf("gadget = %+v, want the panic recovered into an error", r)
	}
}

func TestFailRecordsAndClears(t *testing.T) {
	reset()

	Fail("presentation", errors.New("usb gadget unavailable"))
	if report := Report(); len(report) != 1 || report[0].Error != "usb gadget unavailable" {
		t.Fatalf("got %+v, want the error recorded", report)
	}

	Fail("presentation", nil)
	if report := Report(); len(report) != 1 || report[0].Error != "" {
		t.Fatalf("got %+v, want the error cleared", report)
	}
}
