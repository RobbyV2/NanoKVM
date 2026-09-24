package assistant

import (
	"errors"
	"sync"
)

// Context images live in memory, per device (the extension kept them per
// browser profile). T and a server restart clear them.
const maxContextBytes = 16 << 20

var ErrContextsFull = errors.New("context images exceed 16 MB; clear them with T")

type Contexts struct {
	mu     sync.Mutex
	images []Image
	size   int
}

func (c *Contexts) Add(img Image) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.size+len(img.B64) > maxContextBytes {
		return len(c.images), ErrContextsFull
	}
	c.images = append(c.images, img)
	c.size += len(img.B64)
	return len(c.images), nil
}

func (c *Contexts) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.images)
}

func (c *Contexts) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.images = nil
	c.size = 0
}

func (c *Contexts) Snapshot() []Image {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Image(nil), c.images...)
}
