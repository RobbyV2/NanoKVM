package framequeue

import (
	"encoding/binary"
	"sync"
)

const maxFlowWindow = 8

type Frame struct {
	Key       bool
	Timestamp int64
	Payload   []byte
}

type Queue struct {
	mutex sync.Mutex
	wake  chan struct{}

	frames      []*Frame
	queuedBytes int
	inFlight    []int64

	maxFrames int
	maxBytes  int
	window    int

	flowControlled     bool
	waitingForKeyframe bool
	closed             bool
}

// New creates a bounded queue that starts at the next keyframe.
func New(maxFrames int, maxBytes int) *Queue {
	return &Queue{
		wake:               make(chan struct{}, 1),
		maxFrames:          maxFrames,
		maxBytes:           maxBytes,
		waitingForKeyframe: true,
	}
}

func (q *Queue) EnableFlowControl(window int) {
	if window < 1 {
		window = 1
	}
	if window > maxFlowWindow {
		window = maxFlowWindow
	}

	q.mutex.Lock()
	q.clearFramesLocked()
	q.inFlight = q.inFlight[:0]
	q.window = window
	q.flowControlled = true
	q.waitingForKeyframe = true
	q.mutex.Unlock()
}

func (q *Queue) offer(frame *Frame) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return false
	}

	if frame.Key {
		q.clearFramesLocked()
		if q.flowControlled && len(q.inFlight) >= q.window {
			q.waitingForKeyframe = true
			return false
		}
		q.waitingForKeyframe = false
		q.pushLocked(frame)
		q.signalLocked()
		return true
	}

	if q.waitingForKeyframe {
		return false
	}

	if q.flowControlled {
		if len(q.frames)+len(q.inFlight) >= q.window {
			q.clearFramesLocked()
			q.waitingForKeyframe = true
			return false
		}
	} else if len(q.frames) >= q.maxFrames || q.queuedBytes+len(frame.Payload) > q.maxBytes {
		q.clearFramesLocked()
		q.waitingForKeyframe = true
		return false
	}

	q.pushLocked(frame)
	q.signalLocked()
	return true
}

func (q *Queue) PopForWrite() *Frame {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed || len(q.frames) == 0 {
		return nil
	}

	frame := q.frames[0]
	q.frames[0] = nil
	q.frames = q.frames[1:]
	q.queuedBytes -= len(frame.Payload)
	if q.flowControlled {
		q.inFlight = append(q.inFlight, frame.Timestamp)
	}

	return frame
}

func (q *Queue) Acknowledge(timestamp int64) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	acknowledged := 0
	for acknowledged < len(q.inFlight) && q.inFlight[acknowledged] <= timestamp {
		acknowledged++
	}
	if acknowledged == 0 {
		return
	}

	clear(q.inFlight[:acknowledged])
	q.inFlight = q.inFlight[acknowledged:]
}

func (q *Queue) RequestResync() {
	q.mutex.Lock()
	q.clearFramesLocked()
	q.inFlight = q.inFlight[:0]
	q.waitingForKeyframe = true
	q.mutex.Unlock()
}

func (q *Queue) Close() {
	q.mutex.Lock()
	q.closed = true
	q.clearFramesLocked()
	q.inFlight = q.inFlight[:0]
	q.mutex.Unlock()
}

func (q *Queue) pushLocked(frame *Frame) {
	q.frames = append(q.frames, frame)
	q.queuedBytes += len(frame.Payload)
}

func (q *Queue) clearFramesLocked() {
	clear(q.frames)
	q.frames = q.frames[:0]
	q.queuedBytes = 0
}

func (q *Queue) signalLocked() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func newFrame(isKeyFrame bool, timestamp int64, data []byte) *Frame {
	payload := make([]byte, 9+len(data))
	if isKeyFrame {
		payload[0] = 1
	}
	binary.LittleEndian.PutUint64(payload[1:9], uint64(timestamp))
	copy(payload[9:], data)

	return &Frame{
		Key:       isKeyFrame,
		Timestamp: timestamp,
		Payload:   payload,
	}
}

// Broadcast distributes every captured access unit, even when all flow windows
// are full. The shared encoder keeps advancing in that case. Each blocked queue
// must observe the dropped frame and wait for a new keyframe, otherwise the next
// delivered delta references a picture the client never received.
func Broadcast(queues []*Queue, key bool, timestamp int64, data []byte) {
	frame := newFrame(key, timestamp, data)
	for _, q := range queues {
		q.offer(frame)
	}
}

// Ready is signaled when a frame is available for the transport writer.
func (q *Queue) Ready() <-chan struct{} { return q.wake }
