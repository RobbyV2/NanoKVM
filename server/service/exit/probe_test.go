package exit

import (
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func TestProberThroughFrontDoor(t *testing.T) {
	h := newFrontDoor(t)
	var connected atomic.Bool
	var refuse atomic.Bool
	var opens atomic.Int32
	f := NewFakeExit()
	f.OnOpen = func(_ uint32, _ byte, dst netip.AddrPort) {
		if dst != probeTarget {
			t.Errorf("probe opened %s, want %s", dst, probeTarget)
		}
		opens.Add(1)
	}
	f.Connect = func(netip.AddrPort) (net.Conn, byte) {
		if refuse.Load() {
			return nil, RepConnRefused
		}
		return pipeConnect(netip.AddrPort{})
	}
	h.attachNative(t, f)

	p := NewProber(h.slot, connected.Load)
	p.Start()
	t.Cleanup(p.Stop)

	// Nothing is probed while no exit is connected.
	time.Sleep(probeInitialDelay + 3*probeInterval)
	if r := p.Result(); r.Reachable || r.CheckedAt != nil {
		t.Fatalf("result while disconnected %+v", r)
	}
	if opens.Load() != 0 {
		t.Fatal("probe dialled while disconnected")
	}

	connected.Store(true)
	waitFor(t, "reachable", func() bool { return p.Result().Reachable })
	r := p.Result()
	if r.CheckedAt == nil || time.Since(*r.CheckedAt) > 2*time.Second || r.LatencyMs < 0 || r.LatencyMs > 1000 {
		t.Fatalf("result %+v", r)
	}
	if opens.Load() == 0 {
		t.Fatal("no OPEN reached the exit")
	}

	// The exit cannot reach the target any more.
	refuse.Store(true)
	waitFor(t, "unreachable", func() bool { r := p.Result(); return !r.Reachable && r.CheckedAt != nil })

	// Disconnected again: unreachable, no more dials.
	connected.Store(false)
	waitFor(t, "unreachable after disconnect", func() bool { return !p.Result().Reachable })
	before := opens.Load()
	time.Sleep(3 * probeInterval)
	if opens.Load() != before {
		t.Fatal("probe dialled after disconnect")
	}

	// Stop is idempotent and returns promptly.
	done := make(chan struct{})
	go func() { p.Stop(); p.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop hung")
	}
}

func TestProberWithoutFrontDoor(t *testing.T) {
	slot := MustSlot("1")
	// Nothing listens where the front door should be: the probe fails fast
	// and reports unreachable rather than hanging.
	setTestSocksAddr(slot, "127.0.0.1:1")
	p := NewProber(slot, func() bool { return true })
	p.Start()
	t.Cleanup(p.Stop)
	waitFor(t, "checked", func() bool { return p.Result().CheckedAt != nil })
	if r := p.Result(); r.Reachable || r.LatencyMs != 0 {
		t.Fatalf("result %+v", r)
	}
}
