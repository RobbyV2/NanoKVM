package framequeue

import (
	"encoding/binary"
	"strconv"
	"testing"
)

func flowQueue(window int) *Queue {
	q := New(8, 2*1024*1024)
	q.EnableFlowControl(window)
	return q
}

func broadcast(queues []*Queue, key bool, timestamp int64) {
	Broadcast(queues, key, timestamp, []byte{0, 0, 0, 1, 1})
}

func take(t *testing.T, q *Queue, timestamp int64) *Frame {
	t.Helper()
	f := q.PopForWrite()
	if f == nil || f.Timestamp != timestamp {
		t.Fatalf("frame = %+v, want timestamp %d", f, timestamp)
	}
	return f
}

func noFrame(t *testing.T, q *Queue) {
	t.Helper()
	if f := q.PopForWrite(); f != nil {
		t.Fatalf("unexpected frame %+v: a skipped reference requires a new keyframe", f)
	}
}

func TestEveryBlockedClientWaitsForKeyframeAfterCaptureAdvances(t *testing.T) {
	for _, count := range []int{1, 2} {
		t.Run(strconv.Itoa(count)+" clients", func(t *testing.T) {
			queues := make([]*Queue, count)
			for i := range queues {
				queues[i] = flowQueue(2)
			}
			broadcast(queues, true, 1)
			for _, q := range queues {
				take(t, q, 1)
			}
			broadcast(queues, false, 2)
			for _, q := range queues {
				take(t, q, 2)
			}

			// The source continues encoding while every client's two-frame window
			// is full. Frame 4 references frame 3, which nobody receives.
			broadcast(queues, false, 3)
			for _, q := range queues {
				q.Acknowledge(2)
			}
			broadcast(queues, false, 4)
			for _, q := range queues {
				noFrame(t, q)
			}

			broadcast(queues, true, 5)
			for _, q := range queues {
				take(t, q, 5)
			}
			broadcast(queues, false, 6)
			for _, q := range queues {
				take(t, q, 6)
			}
		})
	}
}

func TestBlockedConsumerDoesNotInterruptHealthyConsumer(t *testing.T) {
	slow, fast := flowQueue(2), flowQueue(2)
	queues := []*Queue{slow, fast}
	for timestamp := int64(1); timestamp <= 4; timestamp++ {
		broadcast(queues, timestamp == 1, timestamp)
		take(t, fast, timestamp)
		fast.Acknowledge(timestamp)
		if timestamp <= 2 {
			take(t, slow, timestamp)
		}
	}
	slow.Acknowledge(2)
	broadcast(queues, false, 5)
	noFrame(t, slow)
	take(t, fast, 5)
}

func TestSkippedKeyframeAlsoInvalidatesReferences(t *testing.T) {
	q := flowQueue(1)
	queues := []*Queue{q}
	broadcast(queues, true, 1)
	take(t, q, 1)
	broadcast(queues, true, 2)
	q.Acknowledge(1)
	broadcast(queues, false, 3)
	noFrame(t, q)
	broadcast(queues, true, 4)
	take(t, q, 4)
}

func TestUnsentFramesAreDiscardedWhenTheWindowOverflows(t *testing.T) {
	q := flowQueue(2)
	queues := []*Queue{q}
	broadcast(queues, true, 1)
	broadcast(queues, false, 2)
	broadcast(queues, false, 3)
	noFrame(t, q)
	broadcast(queues, true, 4)
	take(t, q, 4)
}

func TestUncontrolledConsumerOverflowRequiresKeyframe(t *testing.T) {
	q := New(2, 1024)
	queues := []*Queue{q}
	broadcast(queues, true, 1)
	broadcast(queues, false, 2)
	broadcast(queues, false, 3)
	noFrame(t, q)
	broadcast(queues, false, 4)
	noFrame(t, q)
	broadcast(queues, true, 5)
	take(t, q, 5)
}

func TestAcknowledgementDoesNotDiscardNewerInFlightFrames(t *testing.T) {
	q := flowQueue(2)
	queues := []*Queue{q}
	broadcast(queues, true, 1)
	take(t, q, 1)
	broadcast(queues, false, 2)
	take(t, q, 2)
	q.Acknowledge(1)
	broadcast(queues, false, 3)
	take(t, q, 3)
	broadcast(queues, false, 4)
	q.Acknowledge(3)
	broadcast(queues, false, 5)
	noFrame(t, q)
}

func TestResyncAndCloseDoNotLeakQueuedFrames(t *testing.T) {
	q := flowQueue(2)
	queues := []*Queue{q}
	broadcast(queues, true, 1)
	q.RequestResync()
	broadcast(queues, false, 2)
	noFrame(t, q)
	broadcast(queues, true, 3)
	q.Close()
	broadcast(queues, true, 4)
	noFrame(t, q)
}

func TestWireFrameOwnsItsPayloadAndSignalsWriter(t *testing.T) {
	q := flowQueue(2)
	data := []byte{0, 0, 0, 1, 0x65}
	Broadcast([]*Queue{q}, true, 1234567, data)
	data[4] = 0
	select {
	case <-q.Ready():
	default:
		t.Fatal("writer was not signaled")
	}
	f := take(t, q, 1234567)
	if f.Payload[0] != 1 || binary.LittleEndian.Uint64(f.Payload[1:9]) != 1234567 || f.Payload[13] != 0x65 {
		t.Fatalf("unexpected wire frame %x", f.Payload)
	}
}
