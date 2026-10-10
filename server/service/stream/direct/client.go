package direct

import (
	"encoding/binary"
	"sync"
	"time"

	"NanoKVM-Server/service/stream/direct/internal/framequeue"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	frameAckMessage     byte = 2
	streamResyncMessage byte = 3

	defaultQueueFrames = 8
	defaultQueueBytes  = 2 * 1024 * 1024
	writeWait          = 2 * time.Second
	pingPeriod         = 15 * time.Second
)

type client struct {
	conn  *websocket.Conn
	queue *framequeue.Queue

	done       chan struct{}
	writerDone chan struct{}
	closeOnce  sync.Once
}

func newClient(conn *websocket.Conn) *client {
	return &client{
		conn:       conn,
		queue:      framequeue.New(defaultQueueFrames, defaultQueueBytes),
		done:       make(chan struct{}),
		writerDone: make(chan struct{}),
	}
}

func (c *client) start() {
	go c.writeLoop()
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.queue.Close()
		_ = c.conn.Close()
	})
}

func (c *client) wait() {
	<-c.writerDone
}

func (c *client) handleControl(messageType int, data []byte) {
	if messageType != websocket.BinaryMessage || len(data) == 0 {
		return
	}

	switch data[0] {
	case frameAckMessage:
		if len(data) == 9 {
			timestamp := int64(binary.LittleEndian.Uint64(data[1:]))
			c.queue.Acknowledge(timestamp)
		}
	case streamResyncMessage:
		if len(data) == 1 {
			c.queue.RequestResync()
		}
	}
}

func (c *client) writeLoop() {
	defer close(c.writerDone)

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			deadline := time.Now().Add(writeWait)
			if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				c.close()
				return
			}
		default:
		}

		if frame := c.queue.PopForWrite(); frame != nil {
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.close()
				return
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, frame.Payload); err != nil {
				log.Debugf("failed to write h264 frame to %s: %s", c.conn.RemoteAddr(), err)
				c.close()
				return
			}
			continue
		}

		select {
		case <-c.done:
			return
		case <-c.queue.Ready():
		case <-pingTicker.C:
			deadline := time.Now().Add(writeWait)
			if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				c.close()
				return
			}
		}
	}
}
