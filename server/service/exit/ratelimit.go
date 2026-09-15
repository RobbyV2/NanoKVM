package exit

import (
	"net"
	"net/netip"
	"sync"
	"time"
)

// Limiter is the D10 per-source token attempt limiter: always on, keyed on the
// request's RemoteAddr (IPv6 by /64), exponential backoff after a few free
// attempts, and a bounded map so an attacker cannot grow it without limit.
// There is no global pause: one source being locked never affects another.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*attempt
	now     func() time.Time
	max     int
}

type attempt struct {
	failures    int
	lastFailed  time.Time
	lockedUntil time.Time
}

const (
	limiterFreeAttempts = 3
	limiterBase         = time.Second
	limiterCap          = 15 * time.Minute
	limiterIdle         = time.Hour
	limiterMaxEntries   = 3000
)

func NewLimiter() *Limiter {
	return &Limiter{entries: make(map[string]*attempt), now: time.Now, max: limiterMaxEntries}
}

// Locked reports whether source is inside a lockout. A correct token from a
// locked source is still rejected (D10).
func (l *Limiter) Locked(source string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.entries[source]
	if !ok {
		return false
	}
	now := l.now()
	if now.Sub(a.lastFailed) > limiterIdle {
		delete(l.entries, source)
		return false
	}
	return now.Before(a.lockedUntil)
}

// Fail records a rejected attempt and returns the lockout it imposed (zero
// while the source is still within its free attempts).
func (l *Limiter) Fail(source string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()

	a, ok := l.entries[source]
	if !ok {
		if len(l.entries) >= l.max {
			l.evict(now)
		}
		a = &attempt{}
		l.entries[source] = a
	}
	if !a.lastFailed.IsZero() && now.Sub(a.lastFailed) > limiterIdle {
		a.failures = 0
	}
	a.failures++
	a.lastFailed = now

	over := a.failures - limiterFreeAttempts
	if over <= 0 {
		return 0
	}
	lock := limiterBase << uint(min(over-1, 20))
	if lock > limiterCap || lock <= 0 {
		lock = limiterCap
	}
	a.lockedUntil = now.Add(lock)
	return lock
}

// Reset forgets a source, for the peer whose token was just regenerated (D11).
func (l *Limiter) Reset(source string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, source)
}

// Len is the number of tracked sources, for tests.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// evict drops expired entries and, if the map is still full, the whole map:
// as in auth/brute_force.go, forgetting attempts is the lesser evil next to
// unbounded growth. Callers hold l.mu.
func (l *Limiter) evict(now time.Time) {
	for key, a := range l.entries {
		if now.After(a.lockedUntil) && now.Sub(a.lastFailed) > limiterIdle {
			delete(l.entries, key)
		}
	}
	if len(l.entries) >= l.max {
		l.entries = make(map[string]*attempt)
	}
}

// SourceKey reduces a RemoteAddr to the limiter key: the IPv4 address, or the
// /64 of an IPv6 address, since a single host commonly owns a whole /64.
func SourceKey(remoteAddr string) string {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(64)
	if err != nil {
		return addr.String()
	}
	return prefix.String()
}
