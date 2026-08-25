package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type recordingIngress struct {
	frames  chan MediaFrame
	latency map[string]SinkLatency
}

type recordingSlotManager struct {
	registry    *Registry
	cameras     []string
	microphones []string
	speakers    []string
}

func (m *recordingSlotManager) SetMediaSlots(_ context.Context, cameras, microphones, speakers []string) error {
	m.cameras = append([]string(nil), cameras...)
	m.microphones = append([]string(nil), microphones...)
	m.speakers = append([]string(nil), speakers...)
	var slots []Slot
	for index, label := range cameras {
		slots = append(slots, Slot{ID: fmt.Sprintf("uvc.cam%d", index), Kind: KindCamera, Label: label})
	}
	for index, label := range microphones {
		slots = append(slots, Slot{ID: fmt.Sprintf("uac2.mic%d", index), Kind: KindMicrophone, Label: label})
	}
	for index, label := range speakers {
		slots = append(slots, Slot{ID: fmt.Sprintf("uac2.spk%d", index), Kind: KindSpeaker, Label: label})
	}
	return m.registry.SyncSlots(slots)
}

func (r *recordingIngress) Ingest(_ context.Context, frame MediaFrame) error {
	r.frames <- frame
	return nil
}

func (*recordingIngress) Detach(string) {}

func (r *recordingIngress) Latency() map[string]SinkLatency { return r.latency }

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
	if snapshot := registry.Snapshot(); len(snapshot.Sinks) != 3 {
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

type fakeBinarySession struct {
	ready    chan error
	done     chan error
	received chan []byte
	once     sync.Once
}

func (s *fakeBinarySession) Receive(data []byte) error {
	s.received <- append([]byte(nil), data...)
	return nil
}
func (s *fakeBinarySession) Ready() <-chan error { return s.ready }
func (s *fakeBinarySession) Done() <-chan error  { return s.done }
func (s *fakeBinarySession) Close() error {
	s.once.Do(func() { s.done <- nil })
	return nil
}

type fakeUSBBackend struct{ session *fakeBinarySession }

func (b fakeUSBBackend) Start(context.Context, Stream, func([]byte) error) (BinarySession, error) {
	return b.session, nil
}

func TestSourceWebSocketRoutesWebUSBBinary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, nil, RegistryOptions{})
	service := NewServiceWith(registry)
	session := &fakeBinarySession{ready: make(chan error, 1), done: make(chan error, 1), received: make(chan []byte, 1)}
	session.ready <- nil
	service.SetUSBBackend(fakeUSBBackend{session: session})
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice", Admin: true}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/source", "Browser", []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}})
	defer connection.Close()
	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: HybridSinkID, StreamID: "usb"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, connection, &claimed)
	if claimed.Type != "claimed" {
		t.Fatalf("claim=%+v", claimed)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, []byte("NKUF-response")); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-session.received:
		if string(received) != "NKUF-response" {
			t.Fatalf("binary=%q", received)
		}
	case <-time.After(time.Second):
		t.Fatal("binary message was not routed")
	}
	if err := connection.WriteJSON(controlMessage{Type: "release", SinkID: HybridSinkID}); err != nil {
		t.Fatal(err)
	}
	var released controlResponse
	readJSON(t, connection, &released)
	if released.Type != "released" {
		t.Fatalf("release=%+v", released)
	}
}

func TestWebUSBClaimPrecedesStartupError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, nil, RegistryOptions{})
	service := NewServiceWith(registry)
	session := &fakeBinarySession{ready: make(chan error, 1), done: make(chan error, 1), received: make(chan []byte, 1)}
	session.ready <- errors.New("descriptor mismatch")
	service.SetUSBBackend(fakeUSBBackend{session: session})
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice", Admin: true}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/source", "Browser", []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}})
	defer connection.Close()
	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: HybridSinkID, StreamID: "usb"}); err != nil {
		t.Fatal(err)
	}
	var claimed, failed controlResponse
	readJSON(t, connection, &claimed)
	readJSON(t, connection, &failed)
	if claimed.Type != "claimed" || failed.Type != "error" || failed.SinkID != HybridSinkID {
		t.Fatalf("responses = %+v then %+v", claimed, failed)
	}
	// Releasing the binding publishes binding_removed, and the released frame
	// watchTerminations writes for it races the close of the same socket.
	_ = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, _, err := connection.ReadMessage()
		if err == nil {
			continue
		}
		var timeout net.Error
		if errors.As(err, &timeout) && timeout.Timeout() {
			t.Fatal("startup failure left the source socket open")
		}
		break
	}
	if len(registry.Snapshot().Bindings) != 0 {
		t.Fatal("startup failure left the USB sink bound")
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

func TestWebUSBBinaryIsNotChargedToTheMediaBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, nil, RegistryOptions{})
	service := NewServiceWith(registry)
	session := &fakeBinarySession{
		ready: make(chan error, 1), done: make(chan error, 1),
		received: make(chan []byte, mediaMessageLimit+2),
	}
	session.ready <- nil
	service.SetUSBBackend(fakeUSBBackend{session: session})
	router := gin.New()
	router.GET("/source", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice", Admin: true}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/source", "Browser", []Stream{{
		ID: "usb", Kind: KindUSBDevice, Label: "Debug adapter",
		USB: &USBOffer{Profile: "webusb-debug", Configuration: 1, Interfaces: []uint8{0}},
	}})
	defer connection.Close()
	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: HybridSinkID, StreamID: "usb"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, connection, &claimed)
	if claimed.Type != "claimed" {
		t.Fatalf("claim=%+v", claimed)
	}
	for range mediaMessageLimit + 1 {
		if err := connection.WriteMessage(websocket.BinaryMessage, []byte("NKUF-response")); err != nil {
			t.Fatal(err)
		}
	}
	if err := connection.WriteJSON(controlMessage{Type: "release", SinkID: HybridSinkID}); err != nil {
		t.Fatal(err)
	}
	var released controlResponse
	readJSON(t, connection, &released)
	if released.Type != "released" {
		t.Fatalf("release=%+v", released)
	}
	if len(session.received) != mediaMessageLimit+1 {
		t.Fatalf("relayed %d USB frames", len(session.received))
	}
}

func TestAdminDisconnectStopsTheSourceCapture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	router := gin.New()
	router.GET("/alice", func(c *gin.Context) { service.serveSource(c, Actor{Username: "alice"}) })
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _ := connectSource(t, server.URL, "/alice", "Pixel", []Stream{{ID: "stream", Kind: KindCamera, Label: "Back camera"}})
	defer connection.Close()
	if err := connection.WriteJSON(controlMessage{Type: "claim", SinkID: "uvc.cam0", StreamID: "stream"}); err != nil {
		t.Fatal(err)
	}
	var claimed controlResponse
	readJSON(t, connection, &claimed)
	if claimed.Type != "claimed" {
		t.Fatalf("claim = %+v", claimed)
	}

	admin := gin.New()
	admin.Use(func(c *gin.Context) {
		c.Set("principal", middleware.Principal{Username: "root", Role: authn.RoleAdmin})
		c.Next()
	})
	admin.DELETE("/bindings", service.DisconnectAll)
	recorder := httptest.NewRecorder()
	admin.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/bindings", nil))

	var disconnected struct {
		Code int      `json:"code"`
		Data Snapshot `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &disconnected); err != nil {
		t.Fatal(err)
	}
	if disconnected.Code != 0 || len(disconnected.Data.Bindings) != 0 || len(disconnected.Data.Sinks) == 0 {
		t.Fatalf("disconnect response = %s", recorder.Body.String())
	}

	var revoked controlResponse
	readJSON(t, connection, &revoked)
	if revoked.Type != "released" || revoked.SinkID != "uvc.cam0" || revoked.Reason != ReasonAdminDisconnect {
		t.Fatalf("revocation = %+v", revoked)
	}
}

func TestDisconnectAllRejectsNonAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	alice := Actor{Username: "alice"}
	source := mustSource(t, registry, alice, "Pixel", KindCamera)
	if _, err := registry.Claim(alice, source.ID, "stream", "uvc.cam0"); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("principal", middleware.Principal{Username: "alice", Role: authn.RoleUser})
		c.Next()
	})
	router.DELETE("/bindings", service.DisconnectAll)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/bindings", nil))

	var refused struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &refused); err != nil {
		t.Fatal(err)
	}
	// -1 rather than -2: the registry refuses this too, but only the handler
	// can tell the caller it was the role and not the sweep that failed.
	if refused.Code != -1 || refused.Msg != ErrForbidden.Error() {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if len(registry.Snapshot().Bindings) != 1 {
		t.Fatal("refused disconnect still dropped the binding")
	}
}

func TestRESTClaimRefusesAndAdminTakesOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	alice := mustSource(t, registry, Actor{Username: "alice"}, "Pixel", KindCamera)
	bob := mustSource(t, registry, Actor{Username: "bob"}, "Laptop", KindCamera)
	aliceBody := fmt.Sprintf(`{"source_id":%q,"stream_id":"stream","sink_id":"uvc.cam0"}`, alice.ID)
	bobBody := fmt.Sprintf(`{"source_id":%q,"stream_id":"stream","sink_id":"uvc.cam0"}`, bob.ID)
	bobTakeover := fmt.Sprintf(`{"source_id":%q,"stream_id":"stream"}`, bob.ID)

	granted := postBinding(t, service, middleware.Principal{Username: "alice", Role: authn.RoleUser}, "/bindings", aliceBody)
	var claim struct {
		Code int         `json:"code"`
		Data ClaimResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(granted), &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Code != 0 || claim.Data.Token == "" || claim.Data.Binding.Owner != "alice" {
		t.Fatalf("claim = %s", granted)
	}

	for _, path := range []struct{ url, body string }{
		{url: "/bindings", body: bobBody},
		{url: "/bindings/uvc.cam0/takeover", body: bobTakeover},
	} {
		payload := postBinding(t, service, middleware.Principal{Username: "bob", Role: authn.RoleUser}, path.url, path.body)
		var refusal struct {
			Msg  string       `json:"msg"`
			Data claimRefusal `json:"data"`
		}
		if err := json.Unmarshal([]byte(payload), &refusal); err != nil {
			t.Fatal(err)
		}
		if refusal.Msg != "slot_occupied" || refusal.Data.Owner != "alice" || refusal.Data.SourceLabel != "Pixel" || refusal.Data.Takeover != "refused" {
			t.Fatalf("%s refusal = %s", path.url, payload)
		}
	}
	if binding := sinkByID(t, registry.Snapshot(), "uvc.cam0").Binding; binding == nil || binding.Owner != "alice" {
		t.Fatalf("refused takeover disturbed the incumbent: %+v", binding)
	}

	events, cancel := registry.Subscribe()
	defer cancel()
	<-events
	taken := postBinding(t, service, middleware.Principal{Username: "bob", Role: authn.RoleAdmin}, "/bindings/uvc.cam0/takeover", bobTakeover)
	if !strings.Contains(taken, `"code":0`) {
		t.Fatalf("takeover = %s", taken)
	}
	binding := sinkByID(t, registry.Snapshot(), "uvc.cam0").Binding
	if binding == nil || binding.Owner != "bob" || binding.SourceID != bob.ID {
		t.Fatalf("binding after takeover = %+v", binding)
	}
	removed := <-events
	if removed.Type != "binding_removed" || removed.Reason != ReasonTakenOver || removed.Binding.Owner != "alice" {
		t.Fatalf("termination event = %+v", removed)
	}
	if added := <-events; added.Type != "binding_added" || added.Binding.Owner != "bob" {
		t.Fatalf("takeover event = %+v", added)
	}
}

func TestRESTResumeRestoresAnOrphanedLease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, testSlots, RegistryOptions{})
	service := NewServiceWith(registry)
	alice := Actor{Username: "alice"}
	first := mustSource(t, registry, alice, "Pixel", KindCamera)
	claim, err := registry.Claim(alice, first.ID, "stream", "uvc.cam0")
	if err != nil {
		t.Fatal(err)
	}
	registry.DisconnectSource(first.ID)
	second := mustSource(t, registry, alice, "Pixel reloaded", KindCamera)

	principal := middleware.Principal{Username: "alice", Role: authn.RoleUser}
	rejected := postBinding(t, service, principal, "/bindings/uvc.cam0/resume",
		fmt.Sprintf(`{"source_id":%q,"stream_id":"stream","token":"wrong"}`, second.ID))
	if !strings.Contains(rejected, ErrInvalidToken.Error()) {
		t.Fatalf("bad token resume = %s", rejected)
	}
	resumed := postBinding(t, service, principal, "/bindings/uvc.cam0/resume",
		fmt.Sprintf(`{"source_id":%q,"stream_id":"stream","token":%q}`, second.ID, claim.Token))
	var response struct {
		Code int         `json:"code"`
		Data BindingView `json:"data"`
	}
	if err := json.Unmarshal([]byte(resumed), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 || response.Data.State != StateClaimed || response.Data.SourceID != second.ID {
		t.Fatalf("resume = %s", resumed)
	}
}

func postBinding(t *testing.T, service *Service, principal middleware.Principal, path, body string) string {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("principal", principal)
		c.Next()
	})
	router.POST("/bindings", service.Claim)
	router.POST("/bindings/:sink/resume", service.Resume)
	router.POST("/bindings/:sink/takeover", service.Takeover)
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s = %d %s", path, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

func TestSinkListCarriesTheMeasuredLatency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := mustRegistry(t, []Slot{{ID: "uvc.cam0", Kind: KindCamera, Label: "Camera"}}, RegistryOptions{})
	service := NewServiceWith(registry)
	service.SetIngress(&recordingIngress{latency: map[string]SinkLatency{
		"uvc.cam0": {Frames: 30, AvgMS: 42, PeakMS: 91, BaseMS: 7},
	}})
	router := gin.New()
	router.GET("/sources", service.Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/sources", nil))

	var response struct {
		Data struct {
			Sinks []Sink `json:"sinks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var latency *SinkLatency
	for _, sink := range response.Data.Sinks {
		if sink.ID == "uvc.cam0" {
			latency = sink.Latency
		} else if sink.Latency != nil {
			t.Fatalf("sink %s reports latency it never measured: %+v", sink.ID, *sink.Latency)
		}
	}
	if latency == nil {
		t.Fatal("the sink list carries no latency for a measured sink")
	}
	if latency.Frames != 30 || latency.AvgMS != 42 || latency.PeakMS != 91 || latency.BaseMS != 7 {
		t.Fatalf("latency = %+v", *latency)
	}
}
