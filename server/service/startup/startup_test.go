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
