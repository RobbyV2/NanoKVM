package exit

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"NanoKVM-Server/proto"
)

var (
	probeInterval     = ProbeInterval
	probeTimeout      = ProbeTimeout
	probeInitialDelay = 2 * time.Second
	probeTarget       = netip.MustParseAddrPort("1.1.1.1:443")
)

// Prober is the reachability probe of D20: while an exit is connected it
// dials probeTarget through the slot's SOCKS front door every probeInterval
// and records reachability and latency. Nothing is probed when no exit is
// connected; the last result is then reported as unreachable.
type Prober struct {
	slot      Slot
	connected func() bool

	mu     sync.Mutex
	result proto.ExitUpstream
	stop   chan struct{}
	once   sync.Once
	wg     sync.WaitGroup
}

// NewProber builds a prober; nothing runs until Start.
func NewProber(slot Slot, connected func() bool) *Prober {
	if connected == nil {
		connected = func() bool { return false }
	}
	return &Prober{slot: slot, connected: connected, stop: make(chan struct{})}
}

// Start begins the probe loop.
func (p *Prober) Start() {
	p.wg.Add(1)
	go p.loop()
}

// Stop ends the loop and waits for it.
func (p *Prober) Stop() {
	p.once.Do(func() { close(p.stop) })
	p.wg.Wait()
}

// Result is the last probe outcome.
func (p *Prober) Result() proto.ExitUpstream {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := p.result
	if r.CheckedAt != nil {
		t := *r.CheckedAt
		r.CheckedAt = &t
	}
	return r
}

func (p *Prober) loop() {
	defer p.wg.Done()
	first := time.NewTimer(probeInitialDelay)
	defer first.Stop()
	select {
	case <-first.C:
	case <-p.stop:
		return
	}
	p.tick()
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.tick()
		case <-p.stop:
			return
		}
	}
}

// tick runs one probe, or marks the result unreachable without dialling
// when no exit is connected.
func (p *Prober) tick() {
	if !p.connected() {
		p.mu.Lock()
		p.result.Reachable = false
		p.result.LatencyMs = 0
		p.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	start := time.Now()
	c, err := dialThroughFrontDoor(ctx, p.slot, probeTarget)
	latency := time.Since(start)
	if c != nil {
		_ = c.Close()
	}
	now := time.Now()
	p.mu.Lock()
	p.result = proto.ExitUpstream{
		Reachable: err == nil,
		LatencyMs: latency.Milliseconds(),
		CheckedAt: &now,
	}
	if err != nil {
		p.result.LatencyMs = 0
	}
	p.mu.Unlock()
}
