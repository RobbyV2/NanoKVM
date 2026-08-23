package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type recordingIngress struct{ frames chan MediaFrame }

type recordingSlotManager struct {
	registry    *Registry
	cameras     []string
	microphones []string
}

func (m *recordingSlotManager) SetMediaSlots(_ context.Context, cameras, microphones []string) error {
	m.cameras = append([]string(nil), cameras...)
	m.microphones = append([]string(nil), microphones...)
	var slots []Slot
	for index, label := range cameras {
		slots = append(slots, Slot{ID: fmt.Sprintf("uvc.cam%d", index), Kind: KindCamera, Label: label})
	}
	for index, label := range microphones {
		slots = append(slots, Slot{ID: fmt.Sprintf("uac2.mic%d", index), Kind: KindMicrophone, Label: label})
	}
	return m.registry.SyncSlots(slots)
}

func (r *recordingIngress) Ingest(_ context.Context, frame MediaFrame) error {
	r.frames <- frame
	return nil
}

func (*recordingIngress) Detach(string) {}

func TestSetSinksAppliesThePresentationProfileForAdmins(t *testing.T) {
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

	body, err := json.Marshal(slotsRequest{Slots: []Slot{
		{ID: "uvc.cam0", Kind: KindCamera, Label: "Desk Camera"},
		{ID: "uac2.mic0", Kind: KindMicrophone, Label: "Desk Microphone"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/sinks", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"code":0`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	if len(manager.cameras) != 1 || manager.cameras[0] != "Desk Camera" || len(manager.microphones) != 1 || manager.microphones[0] != "Desk Microphone" {
		t.Fatalf("slots = %v/%v", manager.cameras, manager.microphones)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Sinks) != 2 {
		t.Fatalf("snapshot sinks = %+v", snapshot.Sinks)
	}
}

func TestSetSinksRejectsNonAdminsBeforeProfileMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, nil, RegistryOptions{})
	manager := &recordingSlotManager{registry: registry}
	service := NewServiceWith(registry)
	service.SetSlotManager(manager)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("principal", middleware.Principal{Username: "user", Role: authn.RoleUser})
		c.Next()
	})
	router.PUT("/sinks", service.SetSinks)

	request := httptest.NewRequest(http.MethodPut, "/sinks", strings.NewReader(`{"slots":[]}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if len(manager.cameras)+len(manager.microphones) != 0 || !strings.Contains(recorder.Body.String(), ErrForbidden.Error()) {
		t.Fatalf("response = %s, slots = %v/%v", recorder.Body.String(), manager.cameras, manager.microphones)
	}
}

func TestSourceWebSocketClaimsAndNamesHolder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	router := gin.New()
	router.GET("/events", service.serveEvents)
	router.GET("/alice", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	router.GET("/bob", func(c *gin.Context) { service.serveSource(c, Actor{Username: "bob"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	events := dialWebSocket(t, server.URL, "/events")
	defer events.Close()
	var event Event
	readJSON(t, events, &event)
	if event.Type != "snapshot" {
		t.Fatalf("first event=%s", event.Type)
	}

	alice, aliceReady := connectSource(t, server.URL, "/alice", "Pixel", []Stream{
		{ID: "camera", Kind: KindCamera, Label: "Back camera"},
		{ID: "mic", Kind: KindMicrophone, Label: "Phone microphone"},
	})
	defer alice.Close()
	bob, _ := connectSource(t, server.URL, "/bob", "Laptop", []Stream{{ID: "camera", Kind: KindCamera, Label: "Webcam"}})
	defer bob.Close()

	if err := alice.WriteJSON(controlMessage{Type: "claim", SinkID: "uvc.cam0", StreamID: "camera"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, alice, &claimed)
	if claimed.Type != "claimed" || claimed.Token == "" || claimed.Binding.Owner != "alice" {
		t.Fatalf("claimed=%+v", claimed)
	}
	if claimed.Binding.SourceID != aliceReady.Source.ID {
		t.Fatalf("source id=%s want=%s", claimed.Binding.SourceID, aliceReady.Source.ID)
	}

	if err := bob.WriteJSON(controlMessage{Type: "claim", SinkID: "uvc.cam0", StreamID: "camera"}); err != nil {
		t.Fatal(err)
	}
	var refused controlResponse
	readJSON(t, bob, &refused)
	if refused.Type != "claim_refused" || refused.Message != "slot_occupied" || refused.Owner != "alice" || refused.SourceLabel != "Pixel" {
		t.Fatalf("refused=%+v", refused)
	}

	if err := alice.Close(); err != nil {
		t.Fatal(err)
	}
	for {
		readJSON(t, events, &event)
		if event.Type == "binding_state" && event.Binding != nil && event.Binding.State == StateOrphaned {
			break
		}
	}
	if got := sinkByID(t, registry.Snapshot(), "uvc.cam0").Binding.State; got != StateOrphaned {
		t.Fatalf("binding state=%s", got)
	}
}

func TestSourceWebSocketRejectsUnknownAndOversizedHello(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	unknown := dialWebSocket(t, server.URL, "/source")
	if err := unknown.WriteMessage(websocket.TextMessage, []byte(`{"type":"hello","label":"Phone","streams":[],"admin":true}`)); err != nil {
		t.Fatal(err)
	}
	var response controlResponse
	readJSON(t, unknown, &response)
	if response.Type != "error" || response.Message != "hello required" {
		t.Fatalf("response=%+v", response)
	}
	_ = unknown.Close()

	oversized := dialWebSocket(t, server.URL, "/source")
	payload := `{"type":"hello","label":"` + strings.Repeat("x", maxControlMessage) + `","streams":[]}`
	if err := oversized.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	_ = oversized.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := oversized.ReadMessage(); err == nil {
		t.Fatal("oversized hello kept connection open")
	}
	_ = oversized.Close()
	if len(registry.Snapshot().Sources) != 0 {
		t.Fatal("invalid hello registered a source")
	}
}

func TestSourceWebSocketAcceptsBoundMediaAndAcknowledgesIt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	ingress := &recordingIngress{frames: make(chan MediaFrame, 1)}
	service.SetIngress(ingress)
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, ready := connectSource(t, server.URL, "/source", "Phone", []Stream{{ID: "front", Kind: KindCamera, Label: "Front"}})
	defer connection.Close()
	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: "uvc.cam0", StreamID: "front"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, connection, &claimed)
	if claimed.Type != "claimed" {
		t.Fatalf("claim = %+v", claimed)
	}
	payload := []byte{0xff, 0xd8, 0xff, 0xd9}
	if err := connection.WriteMessage(websocket.BinaryMessage, encodedMediaFrame(MediaKindMJPEG, "uvc.cam0", "front", payload)); err != nil {
		t.Fatal(err)
	}
	var ack controlResponse
	readJSON(t, connection, &ack)
	if ack.Type != "frame_ack" || ack.SinkID != "uvc.cam0" || ack.StreamID != "front" || ack.Sequence != 17 {
		t.Fatalf("ack = %+v", ack)
	}
	select {
	case frame := <-ingress.frames:
		if frame.SourceID != ready.Source.ID || frame.Sequence != 17 || string(frame.Payload) != string(payload) {
			t.Fatalf("frame = %+v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("media frame was not delivered")
	}
}

func TestEventWebSocketGetsFreshSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	actor := Actor{Username: "alice"}
	mustSource(t, registry, actor, "Phone", KindCamera)
	service := NewServiceWith(registry)
	router := gin.New()
	router.GET("/events", service.serveEvents)
	server := httptest.NewServer(router)
	defer server.Close()

	connection := dialWebSocket(t, server.URL, "/events")
	defer connection.Close()
	var event Event
	readJSON(t, connection, &event)
	if event.Type != "snapshot" || event.Snapshot == nil || len(event.Snapshot.Sources) != 1 || event.Snapshot.Sources[0].Owner != "alice" {
		t.Fatalf("event=%+v", event)
	}
}

func connectSource(t *testing.T, serverURL, path, label string, streams []Stream) (*websocket.Conn, controlResponse) {
	t.Helper()
	connection := dialWebSocket(t, serverURL, path)
	if err := connection.WriteJSON(controlMessage{Type: "hello", Label: label, Streams: streams}); err != nil {
		t.Fatal(err)
	}
	var ready controlResponse
	readJSON(t, connection, &ready)
	if ready.Type != "source_ready" || ready.Source == nil || ready.Snapshot == nil {
		t.Fatalf("ready=%+v", ready)
	}
	return connection, ready
}

func dialWebSocket(t *testing.T, serverURL, path string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(serverURL, "http") + path
	connection, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func readJSON(t *testing.T, connection *websocket.Conn, destination any) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := connection.ReadJSON(destination); err != nil {
		t.Fatal(err)
	}
}
