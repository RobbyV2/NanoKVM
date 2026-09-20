package main

import (
	"context"
	"net"
	"net/netip"
	"sync"
)

// stream is one multiplexed connection: a TCP socket or a UDP socket bound
// for this stream, plus the flow-control state the spec keeps per stream.
type stream struct {
	id    uint32
	proto byte
	conn  net.Conn
	pc    net.PacketConn

	sendCredit *credit
	queue      *queue
	ctx        context.Context
	cancel     context.CancelFunc

	mu        sync.Mutex
	remoteEOF bool
	localEOF  bool
	aborted   bool
	consumed  int
}

func (st *stream) markRemoteEOF() (both bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.remoteEOF = true
	return st.localEOF
}

func (st *stream) markLocalEOF() (both bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.localEOF = true
	return st.remoteEOF
}

func (st *stream) remoteEOFSeen() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.remoteEOF
}

func (st *stream) isAborted() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.aborted
}

// consume records n bytes written to the socket and returns the WINDOW
// increment to send once half the stream window has been spent.
func (st *stream) consume(n, window int) int {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.consumed += n
	if window > 0 && st.consumed >= window/2 {
		inc := st.consumed
		st.consumed = 0
		return inc
	}
	return 0
}

func (st *stream) abort() {
	st.mu.Lock()
	st.aborted = true
	st.mu.Unlock()
	st.cancel()
	st.sendCredit.close(errStreamReset)
	if st.conn != nil {
		_ = st.conn.Close()
	}
	if st.pc != nil {
		_ = st.pc.Close()
	}
}

type queued struct {
	data []byte
	dst  netip.AddrPort
	eof  bool
}

// queue is an unbounded FIFO fed by the reader; the credit this end granted
// is what bounds it in practice.
type queue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []queued
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(item queued) {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.mu.Unlock()
	q.cond.Broadcast()
}

func (q *queue) pop(ctx context.Context) (queued, bool) {
	stop := context.AfterFunc(ctx, func() {
		q.mu.Lock()
		q.mu.Unlock() //nolint:staticcheck // the empty critical section orders the broadcast after a waiter's check
		q.cond.Broadcast()
	})
	defer stop()
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 {
		if ctx.Err() != nil {
			return queued{}, false
		}
		q.cond.Wait()
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}
