package sources

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"
)

const (
	mediaHeaderSize   = 26
	maxVideoPayload   = 2 << 20
	maxAudioPayload   = 3840
	maxMediaID        = 64
	maxMediaMessage   = mediaHeaderSize + 2*maxMediaID + maxVideoPayload
	MediaVersion      = 1
	MediaKindMJPEG    = 1
	MediaKindPCMS16LE = 2
)

var mediaMagic = [4]byte{'N', 'K', 'M', 'F'}

type MediaFrame struct {
	SourceID    string
	SinkID      string
	StreamID    string
	Kind        uint8
	Sequence    uint32
	TimestampUS uint64
	Payload     []byte
}

type FrameIngress interface {
	Ingest(context.Context, MediaFrame) error
	Detach(string)
}

func parseMediaFrame(data []byte) (MediaFrame, error) {
	if len(data) < mediaHeaderSize {
		return MediaFrame{}, errors.New("media frame header is truncated")
	}
	if string(data[:4]) != string(mediaMagic[:]) {
		return MediaFrame{}, errors.New("media frame magic is invalid")
	}
	if data[4] != MediaVersion {
		return MediaFrame{}, fmt.Errorf("unsupported media frame version %d", data[4])
	}
	kind := data[5]
	if kind != MediaKindMJPEG && kind != MediaKindPCMS16LE {
		return MediaFrame{}, fmt.Errorf("unsupported media frame kind %d", kind)
	}
	if binary.BigEndian.Uint16(data[6:8]) != 0 {
		return MediaFrame{}, errors.New("media frame flags are unsupported")
	}
	sinkLength, streamLength := int(data[20]), int(data[21])
	if sinkLength == 0 || sinkLength > maxMediaID || streamLength == 0 || streamLength > maxMediaID {
		return MediaFrame{}, errors.New("media frame identifier length is invalid")
	}
	payloadLength := int(binary.BigEndian.Uint32(data[22:26]))
	if kind == MediaKindMJPEG && payloadLength > maxVideoPayload {
		return MediaFrame{}, errors.New("mjpeg frame exceeds 2 MiB")
	}
	if kind == MediaKindPCMS16LE && payloadLength > maxAudioPayload {
		return MediaFrame{}, errors.New("pcm frame exceeds 40 ms")
	}
	want := mediaHeaderSize + sinkLength + streamLength + payloadLength
	if len(data) != want {
		return MediaFrame{}, fmt.Errorf("media frame is %d bytes, want %d", len(data), want)
	}
	sinkEnd := mediaHeaderSize + sinkLength
	streamEnd := sinkEnd + streamLength
	if !utf8.Valid(data[mediaHeaderSize:sinkEnd]) || !utf8.Valid(data[sinkEnd:streamEnd]) {
		return MediaFrame{}, errors.New("media frame identifiers are not UTF-8")
	}
	return MediaFrame{
		SinkID:      string(data[mediaHeaderSize:sinkEnd]),
		StreamID:    string(data[sinkEnd:streamEnd]),
		Kind:        kind,
		Sequence:    binary.BigEndian.Uint32(data[8:12]),
		TimestampUS: binary.BigEndian.Uint64(data[12:20]),
		Payload:     data[streamEnd:],
	}, nil
}
