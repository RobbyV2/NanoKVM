package exit

import (
	"fmt"
	"testing"
	"time"
)

func newTestLimiter() (*Limiter, *time.Time) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	l := NewLimiter()
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLimiterBacksOffExponentiallyPerSource(t *testing.T) {
	l, now := newTestLimiter()

	for i := 0; i < limiterFreeAttempts; i++ {
		if lock := l.Fail("a"); lock != 0 {
			t.Fatalf("attempt %d locked for %s before the free attempts were spent", i+1, lock)
		}
	}
	if l.Locked("a") {
		t.Fatal("locked within the free attempts")
	}

	want := limiterBase
	for i := 0; i < 5; i++ {
		if lock := l.Fail("a"); lock != want {
			t.Fatalf("failure %d locked for %s, want %s", limiterFreeAttempts+i+1, lock, want)
		}
		if !l.Locked("a") {
			t.Fatal("not locked after a failure past the free attempts")
		}
		if l.Locked("b") {
			t.Fatal("another source is locked: the limiter is per source only")
		}
		*now = now.Add(want)
		if l.Locked("a") {
			t.Fatal("still locked once the lockout elapsed")
		}
		want *= 2
	}

	for i := 0; i < 40; i++ {
		l.Fail("a")
	}
	if lock := l.Fail("a"); lock != limiterCap {
		t.Fatalf("lockout %s exceeds the cap %s", lock, limiterCap)
	}
}

func TestLimiterResetAndIdleForget(t *testing.T) {
	l, now := newTestLimiter()
	for i := 0; i < limiterFreeAttempts+2; i++ {
		l.Fail("a")
	}
	if !l.Locked("a") {
		t.Fatal("expected a lockout")
	}
	l.Reset("a")
	if l.Locked("a") || l.Len() != 0 {
		t.Fatal("Reset left the source tracked")
	}

	for i := 0; i < limiterFreeAttempts+1; i++ {
		l.Fail("a")
	}
	*now = now.Add(limiterIdle + time.Minute)
	if l.Locked("a") {
		t.Fatal("an idle source stayed locked")
	}
	if lock := l.Fail("a"); lock != 0 {
		t.Fatalf("failure count survived the idle window: locked for %s", lock)
	}
}

func TestLimiterMapIsBounded(t *testing.T) {
	l, _ := newTestLimiter()
	l.max = 50
	for i := 0; i < 200; i++ {
		l.Fail(fmt.Sprintf("10.0.%d.%d", i/256, i%256))
	}
	if l.Len() > 50 {
		t.Fatalf("limiter tracks %d sources, cap is 50", l.Len())
	}
}

func TestSourceKey(t *testing.T) {
	tests := map[string]string{
		"203.0.113.7:51234":         "203.0.113.7",
		"[2001:db8:1:2:3:4:5:6]:80": "2001:db8:1:2::/64",
		"[::ffff:10.1.2.3]:1":       "10.1.2.3",
		"garbage":                   "garbage",
	}
	for in, want := range tests {
		if got := SourceKey(in); got != want {
			t.Errorf("SourceKey(%q) = %q, want %q", in, got, want)
		}
	}
}
