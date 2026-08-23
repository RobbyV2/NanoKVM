package vnc

import (
	"encoding/binary"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	encodingTight       int32 = 7
	encodingDesktopSize int32 = -223

	tightJPEG byte = 0x90

	// A Tight rectangle length is a 3-byte compact integer at most.
	maxTightPayload = 1<<22 - 1
)

type frame struct {
	data    []byte
	width   uint16
	height  uint16
	version uint64
}

type framePump struct {
	read  func(width uint16, height uint16, quality uint16) ([]byte, int)
	setup func() (width uint16, height uint16, quality uint16, fps int)

	mutex   sync.Mutex
	refs    int
	current frame
	changed chan struct{}
	stop    chan struct{}
}

func newFramePump(
	setup func() (uint16, uint16, uint16, int),
	read func(uint16, uint16, uint16) ([]byte, int),
) *framePump {
	return &framePump{
		read:    read,
		setup:   setup,
		changed: make(chan struct{}),
	}
}

func (p *framePump) acquire() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.refs++
	if p.refs > 1 {
		return
	}

	p.stop = make(chan struct{})
	go p.run(p.stop)
}

func (p *framePump) release() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.refs--
	if p.refs > 0 {
		return
	}

	p.refs = 0
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
	p.current = frame{}
}

func (p *framePump) run(stop chan struct{}) {
	_, _, _, fps := p.setup()
	ticker := time.NewTicker(frameInterval(fps))
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		width, height, quality, current := p.setup()
		if current != fps {
			fps = current
			ticker.Reset(frameInterval(fps))
		}

		data, result := p.read(width, height, quality)
		if result < 0 || result == 5 || len(data) == 0 {
			continue
		}
		if len(data) > maxTightPayload {
			log.Warnf("dropping %d byte frame: larger than a Tight rectangle can carry", len(data))
			continue
		}
		p.publish(data, width, height)
	}
}

func frameInterval(fps int) time.Duration {
	if fps <= 0 {
		fps = 30
	}
	return time.Second / time.Duration(fps)
}

func (p *framePump) publish(data []byte, width uint16, height uint16) {
	p.mutex.Lock()
	p.current = frame{data: data, width: width, height: height, version: p.current.version + 1}
	changed := p.changed
	p.changed = make(chan struct{})
	p.mutex.Unlock()

	close(changed)
}

func (p *framePump) snapshot() (frame, <-chan struct{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.current, p.changed
}

// writeTightJPEG emits one framebuffer update carrying the hardware encoder's
// JPEG as a single Tight rectangle, so no pixel data is ever decoded here.
func writeTightJPEG(w io.Writer, f frame) error {
	message := make([]byte, 0, len(f.data)+32)
	message = appendUpdateHeader(message, f.width, f.height, encodingTight)
	message = append(message, tightJPEG)
	message = appendCompactLength(message, len(f.data))
	message = append(message, f.data...)

	_, err := w.Write(message)
	return err
}

func writeDesktopSize(w io.Writer, width uint16, height uint16) error {
	message := appendUpdateHeader(make([]byte, 0, 16), width, height, encodingDesktopSize)

	_, err := w.Write(message)
	return err
}

// appendUpdateHeader writes a framebuffer update carrying exactly one
// rectangle anchored at the origin.
func appendUpdateHeader(buffer []byte, width uint16, height uint16, encoding int32) []byte {
	buffer = append(buffer, 0, 0)
	buffer = binary.BigEndian.AppendUint16(buffer, 1)
	buffer = binary.BigEndian.AppendUint16(buffer, 0)
	buffer = binary.BigEndian.AppendUint16(buffer, 0)
	buffer = binary.BigEndian.AppendUint16(buffer, width)
	buffer = binary.BigEndian.AppendUint16(buffer, height)
	return binary.BigEndian.AppendUint32(buffer, uint32(encoding))
}

func appendCompactLength(buffer []byte, length int) []byte {
	buffer = append(buffer, byte(length&0x7f))
	if length <= 0x7f {
		return buffer
	}

	buffer[len(buffer)-1] |= 0x80
	buffer = append(buffer, byte((length>>7)&0x7f))
	if length <= 0x3fff {
		return buffer
	}

	buffer[len(buffer)-1] |= 0x80
	return append(buffer, byte((length>>14)&0xff))
}
