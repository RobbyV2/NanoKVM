package sources

import (
	"encoding/binary"
	"errors"
	"testing"
)

func encodedMediaFrame(kind byte, sink, stream string, payload []byte) []byte {
	data := make([]byte, mediaHeaderSize+len(sink)+len(stream)+len(payload))
	copy(data[:4], mediaMagic[:])
	data[4], data[5] = MediaVersion, kind
	binary.BigEndian.PutUint32(data[8:12], 17)
	binary.BigEndian.PutUint64(data[12:20], 123456)
	data[20], data[21] = byte(len(sink)), byte(len(stream))
	binary.BigEndian.PutUint32(data[22:26], uint32(len(payload)))
	copy(data[26:], sink)
	copy(data[26+len(sink):], stream)
	copy(data[26+len(sink)+len(stream):], payload)
	return data
}

func TestParseMediaFrame(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	frame, err := parseMediaFrame(encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", payload))
	if err != nil {
		t.Fatal(err)
	}
	if frame.SinkID != "uvc.cam0" || frame.StreamID != "front" || frame.Kind != MediaKindMJPEG || frame.Sequence != 17 || frame.TimestampUS != 123456 {
		t.Fatalf("frame = %+v", frame)
	}
	if string(frame.Payload) != string(payload) {
		t.Fatalf("payload = %v", frame.Payload)
	}
}

func TestParseMediaFrameRejectsUnboundedOrAmbiguousInput(t *testing.T) {
	tests := map[string]func() []byte{
		"truncated": func() []byte { return []byte("NKMF") },
		"version": func() []byte {
			data := encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", nil)
			data[4] = 2
			return data
		},
		"flags": func() []byte {
			data := encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", nil)
			data[7] = 1
			return data
		},
		"length": func() []byte {
			data := encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", nil)
			binary.BigEndian.PutUint32(data[22:26], 1)
			return data
		},
		"empty payload": func() []byte {
			return encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", nil)
		},
		"video size": func() []byte {
			data := encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", nil)
			binary.BigEndian.PutUint32(data[22:26], maxVideoPayload+1)
			return data
		},
		"audio size": func() []byte {
			data := encodedMediaFrame(MediaKindPCMS16LE, "uac2.mic0", "mic", nil)
			binary.BigEndian.PutUint32(data[22:26], maxAudioPayload+1)
			return data
		},
	}
	for name, makeData := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseMediaFrame(makeData()); err == nil {
				t.Fatal("accepted invalid media frame")
			}
		})
	}
}

func TestAuthorizeFrameRequiresExactLiveBinding(t *testing.T) {
	registry, err := NewRegistry([]Slot{{ID: "uvc.cam0", Kind: KindCamera, Label: "Camera"}}, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	source, err := registry.RegisterSource(Actor{Username: "hari"}, Hello{Label: "Phone", Streams: []Stream{{ID: "front", Kind: KindCamera, Label: "Front"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(Actor{Username: "hari"}, source.ID, "front", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}
	if err := registry.AuthorizeFrame(source.ID, "front", "uvc.cam0", KindCamera); err != nil {
		t.Fatal(err)
	}
	for name, err := range map[string]error{
		"source": registry.AuthorizeFrame("other", "front", "uvc.cam0", KindCamera),
		"stream": registry.AuthorizeFrame(source.ID, "other", "uvc.cam0", KindCamera),
		"kind":   registry.AuthorizeFrame(source.ID, "front", "uvc.cam0", KindMicrophone),
	} {
		if err == nil {
			t.Fatalf("%s mismatch was authorized", name)
		}
	}
	registry.DisconnectSource(source.ID)
	if err := registry.AuthorizeFrame(source.ID, "front", "uvc.cam0", KindCamera); err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("disconnected source err = %v", err)
	}
}

func FuzzParseMediaFrame(f *testing.F) {
	f.Add(encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", []byte{0xff, 0xd8, 0xff, 0xd9}))
	f.Add(encodedMediaFrame(MediaKindPCMS16LE, "uac2.mic0", "mic", make([]byte, 1920)))
	f.Add([]byte("NKMF"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseMediaFrame(data)
	})
}
