package media

import (
	"context"
	"fmt"
	"sort"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// f_uvc sets func.bind_deactivated, so usb_add_function() calls
// usb_function_deactivate() before it binds a camera and the controller keeps
// its soft-disconnect asserted for the whole gadget: HID, the gadget NIC and
// mass storage go off the bus with it. The count only comes back down when
// something opens the function's V4L2 node, because uvc_v4l2_open() is what
// calls usb_function_activate(). Nothing else on the device opens that node
// until a client streams, so a linked camera used to mean no USB at all.
//
// cdev->deactivations is a counter and the two halves are not symmetric:
// usb_function_activate() refuses to decrement past zero (WARN_ON, -EINVAL)
// while uvc_v4l2_release() always increments, so a second overlapping open
// leaks a deactivation that no later open can pay back. The invariant this
// file exists to keep is therefore:
//
//	one open(2) per linked UVC function, released only when the function is
//	unlinked.
//
// Every descriptor the streaming path needs is a dup(2) of the one held here.
// dup shares the struct file, so no second uvc_v4l2_open()/_release() pair ever
// runs and the streaming path can come and go without touching the count.
const (
	holdSettle = 5 * time.Second
)

// Holder is one open file description on a gadget video node.
type Holder interface {
	FD() int
	Close() error
}

type fdHolder struct {
	node string
	fd   int
}

func (h *fdHolder) FD() int { return h.fd }

func (h *fdHolder) Close() error {
	if h.fd < 0 {
		return nil
	}
	fd := h.fd
	h.fd = -1
	return syscall.Close(fd)
}

// The flags match what the streaming layer needs from the descriptor it dups:
// V4L2 event dequeues and buffer dequeues in the UVC loop rely on O_NONBLOCK
// returning EAGAIN rather than sleeping.
func holdNode(node string) (Holder, error) {
	fd, err := syscall.Open(node, syscall.O_RDWR|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return &fdHolder{node: node, fd: fd}, nil
}

type holdRequest struct {
	nodes []string
	reply chan map[string]Holder
}

// holdTable owns every held descriptor from a single goroutine. One owner is
// what makes "one open per node" structural instead of a race to reason about:
// no two opens, and no open racing a close, can ever overlap on a node. A
// caller that gives up waiting therefore leaks nothing, which is what lets
// every entry point put a deadline on the wait instead of blocking a request
// goroutine until the browser gives up.
type holdTable struct {
	requests chan holdRequest
	open     func(string) (Holder, error)
	settle   time.Duration
}

func newHoldTable(open func(string) (Holder, error)) *holdTable {
	if open == nil {
		open = holdNode
	}
	table := &holdTable{requests: make(chan holdRequest), open: open, settle: holdSettle}
	go table.serve()
	return table
}

func (t *holdTable) serve() {
	held := make(map[string]Holder)
	// Nodes whose close(2) never came back. The descriptor is still open, so
	// re-opening would be the second uvc_v4l2_open() this file exists to
	// prevent, and the count never comes back down. Retiring the node keeps
	// the invariant at the cost of that one camera.
	retired := make(map[string]bool)
	for request := range t.requests {
		wanted := make(map[string]bool, len(request.nodes))
		for _, node := range request.nodes {
			wanted[node] = true
		}
		// Release first. A node that is no longer wanted belongs to a function
		// the next apply is about to unlink, and configfs refuses the unlink
		// with -EBUSY while the node is open.
		for node, holder := range held {
			if wanted[node] {
				continue
			}
			delete(held, node)
			if !t.release(node, holder) {
				retired[node] = true
			}
		}
		result := make(map[string]Holder, len(request.nodes))
		for _, node := range request.nodes {
			if retired[node] {
				continue
			}
			holder, ok := held[node]
			if !ok {
				var err error
				var settled bool
				holder, err, settled = t.acquire(node)
				if !settled {
					// The open is still in the kernel and may yet hand back a
					// descriptor no one is tracking. Opening again to find out
					// is the second uvc_v4l2_open() the invariant forbids.
					retired[node] = true
				}
				if err != nil {
					// Loud: a node that will not open is a gadget that stays
					// off the bus, which costs the operator every USB function
					// at once, not just this camera.
					log.Errorf("hold uvc node %s: %s: the gadget stays deactivated until it opens", node, err)
					continue
				}
				held[node] = holder
			}
			result[node] = holder
		}
		request.reply <- result
	}
}

// This goroutine is the single owner of every hold, and both halves of its work
// are syscalls into the UVC gadget driver: close(2) runs uvc_v4l2_release() and
// open(2) runs uvc_v4l2_open(). A kernel that does not return from either used
// to wedge the owner for the life of the process - every later hold() then hit
// the settle timeout, Suspend() failed forever, and the manager refused every
// apply with ErrMediaBusy. That costs the operator the whole gadget: HID cannot
// be relinked, the camera cannot come back, passthrough cannot start, and
// nothing short of a reboot recovers it.
//
// Measured: closing a video node whose function had already been unbound oopsed
// in uvcg_video_enable() and left the caller unkillable in D state, so the
// wedge was not hypothetical. The kernel side of that is fixed, but a syscall
// that never returns must not be able to take the manager with it, so both
// halves run on their own goroutine against a deadline.
func (t *holdTable) release(node string, holder Holder) bool {
	done := make(chan error, 1)
	go func() { done <- holder.Close() }()
	timer := time.NewTimer(t.settle)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			log.Errorf("release uvc node %s: %s", node, err)
		}
		return true
	case <-timer.C:
		// Deliberately not an error to the caller: Suspend() asks the kernel
		// who still holds the node through confirmClosed(), which is the
		// honest check and stays accurate whenever the release does finish.
		log.Errorf("release uvc node %s did not return within %s; abandoning the descriptor and retiring the node", node, t.settle)
		return false
	}
}

// The third result says whether the open(2) settled at all. It did not when the
// deadline expired first, and the caller must then retire the node: the syscall
// is still in the kernel and may still produce a descriptor this table is not
// tracking, so asking for the node again would be the second open.
func (t *holdTable) acquire(node string) (Holder, error, bool) {
	type opened struct {
		holder Holder
		err    error
	}
	done := make(chan opened, 1)
	go func() {
		holder, err := t.open(node)
		done <- opened{holder: holder, err: err}
	}()
	timer := time.NewTimer(t.settle)
	defer timer.Stop()
	select {
	case result := <-done:
		return result.holder, result.err, true
	case <-timer.C:
		return nil, fmt.Errorf("open %s did not return within %s", node, t.settle), false
	}
}

// hold converges the held set on nodes and reports what is held. The wait is
// bounded on both halves so a kernel that never returns from open(2) costs a
// legible error and not a request that hangs until the client times out.
func (t *holdTable) hold(ctx context.Context, nodes []string) (map[string]Holder, error) {
	sorted := append([]string(nil), nodes...)
	sort.Strings(sorted)
	reply := make(chan map[string]Holder, 1)
	timer := time.NewTimer(t.settle)
	defer timer.Stop()
	select {
	case t.requests <- holdRequest{nodes: sorted, reply: reply}:
	case <-timer.C:
		return nil, fmt.Errorf("uvc node holds are still settling after %s", t.settle)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case result := <-reply:
		return result, nil
	case <-timer.C:
		return nil, fmt.Errorf("uvc node holds did not settle within %s", t.settle)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
