//go:build linux

package functionfs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	aioRead      = 0
	aioWrite     = 1
	aioBlockSize = 64
	aioEventSize = 32
)

// How long close waits for the kernel to hand back every request it was given.
var aioDrainGrace = 2 * time.Second

var ErrAIOLeaked = errors.New("functionfs: aio queue leaked")

// struct iocb and struct io_event, which have this layout on every 64-bit
// little-endian port. runtime_linux_test.go pins the sizes.
type aioControlBlock struct {
	Data      uint64
	Key       uint32
	RWFlags   int32
	Opcode    uint16
	ReqPrio   int16
	Fd        uint32
	Buf       uint64
	Bytes     uint64
	Offset    int64
	Reserved2 uint64
	Flags     uint32
	ResFd     uint32
}

type aioEvent struct {
	Data uint64
	Obj  uint64
	Res  int64
	Res2 int64
}

type aioRequest struct {
	slot   int
	length int
	offset int64
	write  bool
}

type aioCompletion struct {
	slot   int
	result int64
}

// One io_submit carries a whole batch, so the syscall rate is the batch rate and
// not the transfer rate. f_fs allocates a usb_request per submission and returns
// -EIOCBQUEUED without waiting, so there is no per-endpoint depth limit to
// respect on the kernel side; the depth here is chosen for the jitter it hides.
//
// Lifetime: nothing on a completion path frees anything. The pool, the aio
// context and the endpoint file are released only by close, and only once every
// submitted request has been returned by io_getevents. A drain that does not
// finish inside aioDrainGrace leaks all three rather than unmapping memory the
// kernel may still be writing into.
type aioQueue struct {
	context  uintptr
	fd       int
	depth    int
	slotSize int
	inFlight int
	region   []byte
	pool     []byte
	blocks   []aioControlBlock
	pointers []uint64
	events   []aioEvent
	locked   bool
}

func newAIOQueue(fd int, depth int, slotSize int) (*aioQueue, error) {
	if depth <= 0 || slotSize <= 0 || depth > 1024 || slotSize > MaxTransferBytes {
		return nil, fmt.Errorf("%w: aio queue depth %d slot %d", ErrEndpointSize, depth, slotSize)
	}
	header := depth * (aioBlockSize + 8 + aioEventSize)
	header = (header + os.Getpagesize() - 1) &^ (os.Getpagesize() - 1)
	region, err := unix.Mmap(-1, 0, header+depth*slotSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
	if err != nil {
		return nil, fmt.Errorf("map aio pool: %w", err)
	}
	queue := &aioQueue{fd: fd, depth: depth, slotSize: slotSize, region: region, pool: region[header:]}
	queue.blocks = unsafe.Slice((*aioControlBlock)(unsafe.Pointer(&region[0])), depth)
	queue.pointers = unsafe.Slice((*uint64)(unsafe.Pointer(&region[depth*aioBlockSize])), depth)
	queue.events = unsafe.Slice((*aioEvent)(unsafe.Pointer(&region[depth*(aioBlockSize+8)])), depth)
	// A reclaim fault inside a microframe costs the slot, and ZRAM makes one
	// cheap for the kernel to take. Failing to lock is not worth refusing over.
	if err := unix.Mlock(region); err == nil {
		queue.locked = true
	}
	if _, _, errno := unix.Syscall(unix.SYS_IO_SETUP, uintptr(depth), uintptr(unsafe.Pointer(&queue.context)), 0); errno != 0 {
		_ = unix.Munmap(region)
		return nil, fmt.Errorf("create aio context of %d: %w", depth, errno)
	}
	return queue, nil
}

func (q *aioQueue) buffer(slot int) []byte {
	return q.pool[slot*q.slotSize : (slot+1)*q.slotSize : (slot+1)*q.slotSize]
}

func (q *aioQueue) span(from, to int) []byte {
	return q.pool[from*q.slotSize : to*q.slotSize : to*q.slotSize]
}

func (q *aioQueue) submit(requests []aioRequest) error {
	if len(requests) == 0 {
		return nil
	}
	for index, request := range requests {
		if request.slot < 0 || request.slot >= q.depth || request.length < 0 || request.length > q.slotSize {
			return fmt.Errorf("%w: aio slot %d length %d", ErrEndpointSize, request.slot, request.length)
		}
		block := &q.blocks[request.slot]
		*block = aioControlBlock{
			Data:   uint64(request.slot),
			Opcode: aioRead,
			Fd:     uint32(q.fd),
			Buf:    uint64(uintptr(unsafe.Pointer(&q.pool[request.slot*q.slotSize]))),
			Bytes:  uint64(request.length),
			Offset: request.offset,
		}
		if request.write {
			block.Opcode = aioWrite
		}
		q.pointers[index] = uint64(uintptr(unsafe.Pointer(block)))
	}
	// io_submit reports -errno only when it took nothing, and the count when it
	// took a prefix, so the count is trustworthy exactly when errno is clear.
	submitted, _, errno := unix.Syscall(unix.SYS_IO_SUBMIT, q.context, uintptr(len(requests)), uintptr(unsafe.Pointer(&q.pointers[0])))
	if errno != 0 {
		return errno
	}
	q.inFlight += int(submitted)
	if int(submitted) != len(requests) {
		return fmt.Errorf("%w: io_submit took %d of %d requests", ErrTransfer, submitted, len(requests))
	}
	return nil
}

func (q *aioQueue) wait(minimum int, out []aioCompletion, timeout time.Duration) (int, error) {
	if q.inFlight == 0 || len(out) == 0 {
		return 0, nil
	}
	maximum := min(len(out), q.inFlight, q.depth)
	minimum = min(minimum, maximum)
	var deadline *unix.Timespec
	if timeout >= 0 {
		spec := unix.NsecToTimespec(int64(timeout))
		deadline = &spec
	}
	n, _, errno := unix.Syscall6(unix.SYS_IO_GETEVENTS, q.context, uintptr(minimum), uintptr(maximum),
		uintptr(unsafe.Pointer(&q.events[0])), uintptr(unsafe.Pointer(deadline)), 0)
	if errno == syscall.EINTR {
		return 0, nil
	}
	if errno != 0 {
		return 0, errno
	}
	q.inFlight -= int(n)
	for index := range int(n) {
		out[index] = aioCompletion{slot: int(q.events[index].Data), result: q.events[index].Res}
	}
	return int(n), nil
}

func (q *aioQueue) drain(grace time.Duration) error {
	completions := make([]aioCompletion, q.depth)
	deadline := time.Now().Add(grace)
	for q.inFlight > 0 {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("%w: %d requests still in flight", ErrAIOLeaked, q.inFlight)
		}
		if _, err := q.wait(1, completions, remaining); err != nil {
			return err
		}
	}
	return nil
}

func (q *aioQueue) close() error {
	err := q.drain(aioDrainGrace)
	if err != nil {
		return err
	}
	if _, _, errno := unix.Syscall(unix.SYS_IO_DESTROY, q.context, 0, 0); errno != 0 {
		return errno
	}
	if q.locked {
		_ = unix.Munlock(q.region)
	}
	return unix.Munmap(q.region)
}

// The relay loops all block in a syscall, which is what makes SCHED_FIFO safe
// here: sched_rt_runtime_exceeded lives outside CONFIG_RT_GROUP_SCHED, so a
// thread that spins is throttled for 50 ms of every second and loses 400
// microframes, while a thread that blocks accrues almost no rt_time. Only the
// calling thread is affected, and only while this goroutine owns it; every other
// goroutine keeps the normal scheduler. A refusal, which is what an unprivileged
// process gets, leaves the loop running at normal priority.
func raiseRealtime(priority int) error {
	runtime.LockOSThread()
	param := struct{ priority int32 }{priority: int32(priority)}
	if _, _, errno := unix.Syscall(unix.SYS_SCHED_SETSCHEDULER, 0, uintptr(unix.SCHED_FIFO), uintptr(unsafe.Pointer(&param))); errno != 0 {
		return errno
	}
	return nil
}
