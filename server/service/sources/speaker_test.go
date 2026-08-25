package sources

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
)

type fakeEgress struct {
	mu       sync.Mutex
	deliver  func(MediaFrame) error
	attached int
	detached int
	err      error
}

func (e *fakeEgress) Attach(sinkID string, deliver func(MediaFrame) error) (func(), error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return nil, e.err
	}
	e.attached++
	e.deliver = deliver
	return func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.detached++
		e.deliver = nil
	}, nil
}

func (e *fakeEgress) send(frame MediaFrame) error {
	e.mu.Lock()
	deliver := e.deliver
	e.mu.Unlock()
	if deliver == nil {
		return errors.New("nobody is attached")
	}
	return deliver(frame)
}

func (e *fakeEgress) counts() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.attached, e.detached
}

func speakerSlots() []Slot {
	return []Slot{{ID: "uac2.spk0", Kind: KindSpeaker, Label: "Speaker 1"}}
}

// A frame this server sends must be one it would itself accept, or the browser
// and the gadget disagree about the wire the first time either changes.
func TestEncodedFrameParsesBack(t *testing.T) {
	payload := bytes.Repeat([]byte{0x11, 0x22}, 960)
	data, err := encodeMediaFrame(MediaFrame{
		SinkID: "uac2.spk0", StreamID: "spk", Kind: MediaKindPCMS16LE,
		Sequence: 7, TimestampUS: 1234567, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := parseMediaFrame(data)
	if err != nil {
		t.Fatalf("parseMediaFrame(encodeMediaFrame(..)) = %v", err)
	}
	if frame.SinkID != "uac2.spk0" || frame.StreamID != "spk" || frame.Sequence != 7 || frame.TimestampUS != 1234567 {
		t.Fatalf("frame = %+v", frame)
	}
	if !bytes.Equal(frame.Payload, payload) {
		t.Fatal("payload did not survive the round trip")
	}
	if _, err := encodeMediaFrame(MediaFrame{SinkID: "uac2.spk0", StreamID: "spk", Kind: MediaKindPCMS16LE, Payload: make([]byte, maxAudioPayload+1)}); err == nil {
		t.Fatal("an oversized payload must be refused before it reaches the wire")
	}
}

func TestSpeakerClaimStreamsAudioToTheBrowser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, speakerSlots(), RegistryOptions{})
	service := NewServiceWith(registry)
	egress := &fakeEgress{}
	service.SetEgress(egress)
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/source", "Browser", []Stream{{
		ID: "spk", Kind: KindSpeaker, Label: "Playback",
		Formats: []Format{{Codec: "pcm_s16le", SampleRate: 48000, Channels: 1}},
	}})
	defer connection.Close()

	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: "uac2.spk0", StreamID: "spk"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, connection, &claimed)
	if claimed.Type != "claimed" {
		t.Fatalf("claim = %+v", claimed)
	}
	if attached, _ := egress.counts(); attached != 1 {
		t.Fatalf("attached = %d, want 1", attached)
	}

	payload := bytes.Repeat([]byte{0x7f, 0x00}, 960)
	if err := egress.send(MediaFrame{SinkID: "uac2.spk0", Kind: MediaKindPCMS16LE, Sequence: 3, TimestampUS: 99, Payload: payload}); err != nil {
		t.Fatal(err)
	}

	_ = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("message type = %d, want binary", messageType)
	}
	frame, err := parseMediaFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	if frame.SinkID != "uac2.spk0" || frame.StreamID != "spk" || frame.Sequence != 3 || !bytes.Equal(frame.Payload, payload) {
		t.Fatalf("frame = %+v", frame)
	}

	if err := connection.WriteJSON(controlMessage{Type: "release", SinkID: "uac2.spk0"}); err != nil {
		t.Fatal(err)
	}
	var released controlResponse
	readJSON(t, connection, &released)
	if released.Type != "released" {
		t.Fatalf("release = %+v", released)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, detached := egress.counts(); detached == 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("releasing a speaker never detached its capture")
}

// A media backend that cannot capture must refuse quickly and give the slot
// back, never leave the browser holding a claim that will never produce audio.
func TestSpeakerClaimFailsFastWithoutCapture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, speakerSlots(), RegistryOptions{})
	service := NewServiceWith(registry)
	service.SetEgress(&fakeEgress{err: errors.New("hw:2,0 is not open")})
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/source", "Browser", []Stream{{
		ID: "spk", Kind: KindSpeaker, Label: "Playback",
		Formats: []Format{{Codec: "pcm_s16le", SampleRate: 48000, Channels: 1}},
	}})
	defer connection.Close()

	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: "uac2.spk0", StreamID: "spk"}); err != nil {
		t.Fatal(err)
	}
	var refused controlResponse
	readJSON(t, connection, &refused)
	if refused.Type != "error" || !strings.Contains(refused.Message, "hw:2,0") {
		t.Fatalf("claim = %+v", refused)
	}
	if binding := registry.Snapshot().Bindings; len(binding) != 0 {
		t.Fatalf("the slot stayed claimed: %+v", binding)
	}
}

func TestSpeakerSinkRefusesBrowserAudio(t *testing.T) {
	registry := mustRegistry(t, speakerSlots(), RegistryOptions{})
	source, err := registry.RegisterSource(Actor{Username: "alice"}, Hello{Label: "Browser", Streams: []Stream{
		{ID: "spk", Kind: KindSpeaker, Label: "Playback", Formats: []Format{{Codec: "pcm_s16le", SampleRate: 48000, Channels: 1}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Claim(Actor{Username: "alice"}, source.ID, "spk", "uac2.spk0"); err != nil {
		t.Fatal(err)
	}
	// handleMedia maps every PCM frame to a microphone, so a speaker sink can
	// never be written to over the source socket.
	if err := registry.AuthorizeFrame(source.ID, "spk", "uac2.spk0", KindMicrophone); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("AuthorizeFrame = %v, want a refusal", err)
	}
}

func TestSetSinksSplitsSpeakersFromMicrophones(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, nil, RegistryOptions{})
	manager := &recordingSlotManager{registry: registry}
	service := NewServiceWith(registry)
	service.SetSlotManager(manager)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("principal", middleware.Principal{Username: "admin", Role: authn.RoleAdmin})
		c.Next()
	})
	router.PUT("/sinks", service.SetSinks)

	body := `{"slots":[{"id":"uac2.mic0","kind":"microphone","label":"Microphone 1"},{"id":"uac2.spk0","kind":"speaker","label":"Speaker 1"}]}`
	request := httptest.NewRequest(http.MethodPut, "/sinks", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(manager.microphones) != 1 || manager.microphones[0] != "Microphone 1" {
		t.Fatalf("microphones = %v", manager.microphones)
	}
	if len(manager.speakers) != 1 || manager.speakers[0] != "Speaker 1" {
		t.Fatalf("speakers = %v", manager.speakers)
	}
}
