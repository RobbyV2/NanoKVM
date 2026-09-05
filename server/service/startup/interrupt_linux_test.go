package startup

import (
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// The kernel's struct sigaction on the 64-bit Linux ports Go supports.
type kernelSigaction struct {
	handler  uintptr
	flags    uint64
	restorer uintptr
	mask     uint64
}

// What musl's system() does for the life of its child: an rt_sigaction to the
// kernel behind the runtime's back, leaving SIGINT ignored.
func ignoreInKernel(t *testing.T) {
	t.Helper()
	act := kernelSigaction{handler: 1} // SIG_IGN
	_, _, errno := syscall.RawSyscall6(syscall.SYS_RT_SIGACTION, uintptr(syscall.SIGINT),
		uintptr(unsafe.Pointer(&act)), 0, 8, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}
}

func expectInterrupt(t *testing.T, ch <-chan os.Signal, want bool, what string) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case sig := <-ch:
		if sig != syscall.SIGINT {
			t.Fatalf("%s: got %v, want SIGINT", what, sig)
		}
		if !want {
			t.Fatalf("%s: SIGINT reached Go", what)
		}
	case <-time.After(300 * time.Millisecond):
		if want {
			t.Fatalf("%s: SIGINT did not reach Go", what)
		}
	}
}

func TestReassertInterruptTakesTheSignalBackFromAnIgnoredDisposition(t *testing.T) {
	ch := Interrupts()
	expectInterrupt(t, ch, true, "baseline")
	if got := interruptDisposition(); got != "caught" {
		t.Fatalf("disposition at baseline: got %q, want caught", got)
	}

	ignoreInKernel(t)
	if got := interruptDisposition(); got != "ignored" {
		t.Fatalf("disposition after SIG_IGN: got %q, want ignored", got)
	}
	expectInterrupt(t, ch, false, "after SIG_IGN")

	ReassertInterrupt("test")
	if got := interruptDisposition(); got != "caught" {
		t.Fatalf("disposition after reassert: got %q, want caught", got)
	}
	expectInterrupt(t, ch, true, "after reassert")

	// A second time, since the shells that cause this run for the life of
	// the process and the reassert has to work every time, not once.
	ignoreInKernel(t)
	expectInterrupt(t, ch, false, "after SIG_IGN again")
	ReassertInterrupt("test again")
	expectInterrupt(t, ch, true, "after reassert again")
}

func TestReassertInterruptLeavesACaughtSignalCaught(t *testing.T) {
	ch := Interrupts()
	ReassertInterrupt("nothing wrong")
	expectInterrupt(t, ch, true, "after a reassert with nothing to fix")
}

func TestDispositionFromStatus(t *testing.T) {
	// SIGHUP is bit 0, SIGINT bit 1, SIGPIPE bit 12.
	status := "Name:\tNanoKVM-Server\nSigPnd:\t0000000000000000\nSigBlk:\t0000000000000000\nSigIgn:\t0000000000001000\nSigCgt:\tfffffffe7fc1fefe\n"
	if got := dispositionFromStatus(status, syscall.SIGINT); got != "caught" {
		t.Errorf("caught: got %q", got)
	}
	if got := dispositionFromStatus(status, syscall.SIGHUP); got != "default" {
		t.Errorf("default: got %q", got)
	}
	if got := dispositionFromStatus(status, syscall.SIGPIPE); got != "ignored" {
		t.Errorf("ignored: got %q", got)
	}
	if got := dispositionFromStatus("Name:\tx\n", syscall.SIGINT); got != "" {
		t.Errorf("unreadable: got %q", got)
	}
}
