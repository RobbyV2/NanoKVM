//go:build linux

package functionfs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"NanoKVM-Server/service/presentation"
)

const (
	maxISOPackets = 32
	urbISOASAP    = 0x02
)

// A group is one URB of isoPacketsPerTransfer microframes, so eight groups of
// eight packets buffer (8-1)*8*125us = 7ms of scheduling jitter behind 64
// isochronous transfers, which is also 64 iocbs and one io_submit per
// millisecond. Eight packets rather than thirty-two because the reactor reaps
// completed URBs on a 1ms tick, and a URB longer than that tick would make the
// reap period, not the buffer, the latency floor.
var (
	isoPacketsPerTransfer = 8
	isoTransfersInFlight  = 8
	isoRealtimePriority   = 10
	isoWaitTimeout        = 20 * time.Millisecond
)

type Stream interface {
	Start(payload int) error
	Stop() error
}

type isochronousEndpoint struct {
	*linuxEndpoint
	device    *linuxDevice
	source    Endpoint
	number    uint8
	alternate uint8
	packet    int

	mu     sync.Mutex
	stream *isochronousStream
}

func newIsochronousEndpoint(file *linuxEndpoint, device *linuxDevice, endpoint presentation.FunctionFSEndpoint, number uint8, alternate uint8) *isochronousEndpoint {
	return &isochronousEndpoint{
		linuxEndpoint: file, device: device, number: number, alternate: alternate,
		packet: int(endpoint.MaxPacket) * (int(endpoint.Mult) + 1),
		source: Endpoint{
			SourceAddress: endpoint.SourceAddress, Address: endpoint.Address,
			Transfer: string(endpoint.Transfer), MaxPacket: endpoint.MaxPacket,
		},
	}
}

func (e *isochronousEndpoint) Start(payload int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.stopLocked(); err != nil {
		return err
	}
	slot := e.packet
	if payload > 0 && payload < slot {
		slot = payload
	}
	if err := e.device.SetAlternate(e.number, e.alternate); err != nil {
		return fmt.Errorf("set imported interface %d to alternate setting %d: %w", e.number, e.alternate, err)
	}
	queue, err := newAIOQueue(int(e.file.Fd()), isoTransfersInFlight*isoPacketsPerTransfer, slot)
	if err != nil {
		return err
	}
	stream := &isochronousStream{
		device: e.device, queue: queue, source: e.source,
		groups: isoTransfersInFlight, perGroup: isoPacketsPerTransfer, slot: slot,
		free: make(chan int, isoTransfersInFlight), ready: make(chan readyGroup, isoTransfersInFlight),
	}
	stream.ctx, stream.cancel = context.WithCancel(context.Background())
	for group := range stream.groups {
		stream.free <- group
	}
	stream.wait.Add(2)
	go stream.sourceLoop()
	go stream.sinkLoop()
	e.stream = stream
	return nil
}

func (e *isochronousEndpoint) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stopLocked()
}

func (e *isochronousEndpoint) stopLocked() error {
	if e.stream == nil {
		return nil
	}
	err := e.stream.stop()
	e.stream = nil
	if err != nil {
		return err
	}
	return e.device.SetAlternate(e.number, 0)
}

func (e *isochronousEndpoint) Close() error {
	return errors.Join(e.Stop(), e.linuxEndpoint.Close())
}

type readyGroup struct {
	group   int
	lengths []int
}

// Lifetime, in one sentence: neither loop frees anything, stop is the only
// release, and it releases only after both loops have exited and the kernel has
// returned every URB and every iocb. The pool is simultaneously the usbfs
// transfer buffer and the aio buffer, so a source transfer that outlives its
// cancellation leaks the whole stream rather than letting munmap race a DMA.
type isochronousStream struct {
	device   *linuxDevice
	queue    *aioQueue
	source   Endpoint
	groups   int
	perGroup int
	slot     int

	free  chan int
	ready chan readyGroup

	ctx    context.Context
	cancel context.CancelFunc
	wait   sync.WaitGroup

	mu      sync.Mutex
	failure error
	leaked  int
}

func (s *isochronousStream) fail(err error) {
	s.mu.Lock()
	if s.failure == nil {
		s.failure = err
	}
	s.mu.Unlock()
	s.cancel()
}

func (s *isochronousStream) groupBuffer(group int) []byte {
	return s.queue.span(group*s.perGroup, (group+1)*s.perGroup)
}

func (s *isochronousStream) release(group int) {
	select {
	case s.free <- group:
	default:
	}
}

func (s *isochronousStream) take(block bool) (int, bool) {
	if !block {
		select {
		case group := <-s.free:
			return group, true
		default:
			return 0, false
		}
	}
	select {
	case group := <-s.free:
		return group, true
	case <-s.ctx.Done():
		return 0, false
	}
}

// Transfers are awaited in submission order, which is what keeps a video frame's
// packets in the order the camera produced them.
func (s *isochronousStream) sourceLoop() {
	defer s.wait.Done()
	_ = raiseRealtime(isoRealtimePriority)
	var groups []int
	var requests []*usbRequest
	defer func() { s.drainSource(requests) }()

	for {
		for len(requests) < s.groups {
			group, ok := s.take(len(requests) == 0)
			if !ok {
				if len(requests) == 0 {
					return
				}
				break
			}
			request, err := s.device.beginISO(s.ctx, s.source, s.groupBuffer(group), s.perGroup, s.slot)
			if err != nil {
				s.release(group)
				s.fail(err)
				return
			}
			groups = append(groups, group)
			requests = append(requests, request)
		}
		group, request := groups[0], requests[0]
		select {
		case result := <-request.done:
			groups, requests = groups[1:], requests[1:]
			if result.err != nil {
				s.release(group)
				s.fail(fmt.Errorf("source endpoint 0x%02x: %w", s.source.SourceAddress, result.err))
				return
			}
			select {
			case s.ready <- readyGroup{group: group, lengths: result.lengths}:
			case <-s.ctx.Done():
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *isochronousStream) drainSource(requests []*usbRequest) {
	grace := time.NewTimer(transferCancelGrace)
	defer grace.Stop()
	for index, request := range requests {
		select {
		case <-request.done:
		case <-grace.C:
			s.mu.Lock()
			s.leaked = len(requests) - index
			s.mu.Unlock()
			return
		}
	}
}

func (s *isochronousStream) sinkLoop() {
	defer s.wait.Done()
	_ = raiseRealtime(isoRealtimePriority)
	outstanding := make([]int, s.groups)
	batch := make([]aioRequest, 0, s.queue.depth)
	completions := make([]aioCompletion, s.queue.depth)

	for {
		item, ok := s.nextReady(s.queue.inFlight == 0)
		if !ok && s.ctx.Err() != nil {
			return
		}
		if ok {
			batch = batch[:0]
			for index, length := range item.lengths {
				if length <= 0 {
					continue
				}
				batch = append(batch, aioRequest{slot: item.group*s.perGroup + index, length: length, write: true})
			}
			outstanding[item.group] = len(batch)
			if len(batch) == 0 {
				s.release(item.group)
			} else if err := s.queue.submit(batch); err != nil {
				s.fail(fmt.Errorf("functionfs endpoint 0x%02x: %w", s.source.Address, err))
				return
			}
		}
		if s.queue.inFlight == 0 {
			continue
		}
		n, err := s.queue.wait(s.perGroup, completions, isoWaitTimeout)
		if err != nil {
			s.fail(fmt.Errorf("functionfs endpoint 0x%02x: %w", s.source.Address, err))
			return
		}
		for _, completion := range completions[:n] {
			group := completion.slot / s.perGroup
			outstanding[group]--
			if outstanding[group] == 0 {
				s.release(group)
			}
		}
	}
}

func (s *isochronousStream) nextReady(block bool) (readyGroup, bool) {
	if !block {
		select {
		case item := <-s.ready:
			return item, true
		default:
			return readyGroup{}, false
		}
	}
	select {
	case item := <-s.ready:
		return item, true
	case <-s.ctx.Done():
		return readyGroup{}, false
	}
}

func (s *isochronousStream) stop() error {
	s.cancel()
	s.wait.Wait()
	s.mu.Lock()
	failure, leaked := s.failure, s.leaked
	s.mu.Unlock()
	if leaked != 0 {
		return errors.Join(failure, fmt.Errorf("%w: %d source transfers on endpoint 0x%02x never answered cancellation", ErrAIOLeaked, leaked, s.source.SourceAddress))
	}
	return errors.Join(failure, s.queue.close())
}
