package exit

import (
	"sync/atomic"

	"NanoKVM-Server/proto"
)

// Counter is the ByteCounter the manager hands every component of a slot.
type Counter struct {
	up   atomic.Int64
	down atomic.Int64
}

func NewCounter() *Counter { return &Counter{} }

func (c *Counter) AddUp(n int64)   { c.up.Add(n) }
func (c *Counter) AddDown(n int64) { c.down.Add(n) }

func (c *Counter) Totals() proto.ExitBytes {
	return proto.ExitBytes{Up: uint64(max(c.up.Load(), 0)), Down: uint64(max(c.down.Load(), 0))}
}
