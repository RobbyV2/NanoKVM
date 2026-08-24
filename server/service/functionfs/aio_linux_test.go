//go:build linux

package functionfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"
)

func TestAIOStructuresMatchTheKernelABI(t *testing.T) {
	if unsafe.Sizeof(aioControlBlock{}) != aioBlockSize || unsafe.Sizeof(aioEvent{}) != aioEventSize {
		t.Fatalf("aio sizes = %d/%d, want %d/%d", unsafe.Sizeof(aioControlBlock{}), unsafe.Sizeof(aioEvent{}), aioBlockSize, aioEventSize)
	}
	var block aioControlBlock
	base := uintptr(unsafe.Pointer(&block))
	for name, want := range map[string]uintptr{
		"aio_lio_opcode": 16, "aio_fildes": 20, "aio_buf": 24, "aio_nbytes": 32, "aio_offset": 40,
	} {
		var got uintptr
		switch name {
		case "aio_lio_opcode":
			got = uintptr(unsafe.Pointer(&block.Opcode)) - base
		case "aio_fildes":
			got = uintptr(unsafe.Pointer(&block.Fd)) - base
		case "aio_buf":
			got = uintptr(unsafe.Pointer(&block.Buf)) - base
		case "aio_nbytes":
			got = uintptr(unsafe.Pointer(&block.Bytes)) - base
		case "aio_offset":
			got = uintptr(unsafe.Pointer(&block.Offset)) - base
		}
		if got != want {
			t.Fatalf("%s at offset %d, want %d", name, got, want)
		}
	}
}

// io_submit works on any file, so the mechanism is testable without a gadget
// endpoint: a whole batch goes in with one syscall and comes back through
// io_getevents carrying the slot each completion belongs to.
func aioBatchRoundTrip(t *testing.T) {
	t.Helper()
	const depth, slot = 16, 64

	payload := bytes.Repeat([]byte("nanokvm-"), depth*slot/8)
	for index := range payload {
		payload[index] ^= byte(index)
	}
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	queue, err := newAIOQueue(int(file.Fd()), depth, slot)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]aioRequest, depth)
	for index := range requests {
		requests[index] = aioRequest{slot: index, length: slot, offset: int64(index * slot)}
	}
	if err := queue.submit(requests); err != nil {
		t.Fatalf("submit %d reads: %v", depth, err)
	}
	if queue.inFlight != depth {
		t.Fatalf("in flight = %d after one io_submit of %d", queue.inFlight, depth)
	}

	completions := make([]aioCompletion, depth)
	seen := make(map[int]bool, depth)
	for len(seen) < depth {
		n, err := queue.wait(1, completions, time.Second)
		if err != nil {
			t.Fatalf("io_getevents: %v", err)
		}
		if n == 0 {
			t.Fatalf("io_getevents returned nothing with %d in flight", queue.inFlight)
		}
		for _, completion := range completions[:n] {
			if completion.result != slot {
				t.Fatalf("slot %d returned %d, want %d", completion.slot, completion.result, slot)
			}
			if seen[completion.slot] {
				t.Fatalf("slot %d completed twice", completion.slot)
			}
			seen[completion.slot] = true
			want := payload[completion.slot*slot : (completion.slot+1)*slot]
			if !bytes.Equal(queue.buffer(completion.slot), want) {
				t.Fatalf("slot %d read %x, want %x", completion.slot, queue.buffer(completion.slot), want)
			}
		}
	}
	if queue.inFlight != 0 {
		t.Fatalf("in flight = %d after every completion", queue.inFlight)
	}
	if err := queue.close(); err != nil {
		t.Fatalf("close a drained queue: %v", err)
	}
}

func TestAIOSubmitsAWholeBatchInOneSyscall(t *testing.T) { aioBatchRoundTrip(t) }

func TestAIOWritesReachTheFile(t *testing.T) {
	const depth, slot = 8, 32
	path := filepath.Join(t.TempDir(), "sink")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	queue, err := newAIOQueue(int(file.Fd()), depth, slot)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]aioRequest, depth)
	for index := range requests {
		for byteIndex := range queue.buffer(index) {
			queue.buffer(index)[byteIndex] = byte(index)
		}
		requests[index] = aioRequest{slot: index, length: slot, offset: int64(index * slot), write: true}
	}
	if err := queue.submit(requests); err != nil {
		t.Fatal(err)
	}
	if err := queue.drain(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := queue.close(); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for index := range depth {
		if !bytes.Equal(written[index*slot:(index+1)*slot], bytes.Repeat([]byte{byte(index)}, slot)) {
			t.Fatalf("slot %d landed as %x", index, written[index*slot:(index+1)*slot])
		}
	}
}

// The teardown invariant, asserted from the outside: a queue with a request the
// kernel has not returned is leaked, not unmapped.
func TestAIOCloseLeaksRatherThanUnmappingUnderTheKernel(t *testing.T) {
	previous := aioDrainGrace
	aioDrainGrace = 20 * time.Millisecond
	defer func() { aioDrainGrace = previous }()

	queue, err := newAIOQueue(0, 4, 64)
	if err != nil {
		t.Fatal(err)
	}
	queue.inFlight = 1
	if err := queue.close(); !errors.Is(err, ErrAIOLeaked) {
		t.Fatalf("close with a request outstanding = %v, want ErrAIOLeaked", err)
	}
	queue.inFlight = 0
	if err := queue.close(); err != nil {
		t.Fatal(err)
	}
}
