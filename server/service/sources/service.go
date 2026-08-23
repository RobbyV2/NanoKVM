package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	maxControlMessage  = 64 << 10
	helloTimeout       = 10 * time.Second
	websocketTimeout   = 60 * time.Second
	websocketWriteWait = 5 * time.Second
	pingInterval       = 20 * time.Second
	messageWindow      = 10 * time.Second
	messageLimit       = 64
)

type Service struct {
	registry *Registry
	store    *Store
	initErr  error
	mu       sync.Mutex
}

type controlMessage struct {
	Type     string   `json:"type"`
	Label    string   `json:"label,omitempty"`
	Streams  []Stream `json:"streams,omitempty"`
	SinkID   string   `json:"sink_id,omitempty"`
	StreamID string   `json:"stream_id,omitempty"`
	Token    string   `json:"token,omitempty"`
}

type controlResponse struct {
	Type        string       `json:"type"`
	Message     string       `json:"message,omitempty"`
	Source      *Source      `json:"source,omitempty"`
	Snapshot    *Snapshot    `json:"snapshot,omitempty"`
	Binding     *BindingView `json:"binding,omitempty"`
	Token       string       `json:"token,omitempty"`
	SinkID      string       `json:"sink_id,omitempty"`
	Owner       string       `json:"owner,omitempty"`
	SourceLabel string       `json:"source_label,omitempty"`
	Since       time.Time    `json:"since,omitempty"`
}

type slotsRequest struct {
	Slots []Slot `json:"slots"`
}

var sourceUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     middleware.CheckWebSocketOrigin,
}

func NewService() *Service {
	store := NewStore("")
	slots, loadErr := store.Load()
	registry, registryErr := NewRegistry(slots, RegistryOptions{})
	if registryErr != nil {
		registry, _ = NewRegistry(nil, RegistryOptions{})
	}
	return &Service{registry: registry, store: store, initErr: errors.Join(loadErr, registryErr)}
}

func NewServiceWith(registry *Registry, store *Store) *Service {
	return &Service{registry: registry, store: store}
}

func (s *Service) Get(c *gin.Context) {
	var response proto.Response
	if err := s.ready(); err != nil {
		response.ErrRsp(c, -1, err.Error())
		return
	}
	snapshot := s.registry.Snapshot()
	response.OkRspWithData(c, &snapshot)
}

func (s *Service) SetSinks(c *gin.Context) {
	var response proto.Response
	actor, ok := actorFrom(c)
	if !ok || !actor.Admin {
		response.ErrRsp(c, -1, ErrForbidden.Error())
		return
	}
	var request slotsRequest
	if err := decodeHTTPRequest(c, &request); err != nil {
		response.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := s.store.Save(request.Slots); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	if err := s.registry.ReplaceSlots(actor, request.Slots); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	s.mu.Lock()
	s.initErr = nil
	s.mu.Unlock()
	snapshot := s.registry.Snapshot()
	response.OkRspWithData(c, &snapshot)
}

func (s *Service) Release(c *gin.Context) {
	var response proto.Response
	actor, ok := actorFrom(c)
	if !ok {
		response.ErrRsp(c, -1, ErrForbidden.Error())
		return
	}
	if err := s.registry.Release(actor, c.Param("sink"), ReasonReleased); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	response.OkRsp(c)
}

func (s *Service) DisconnectAll(c *gin.Context) {
	var response proto.Response
	actor, ok := actorFrom(c)
	if !ok || !actor.Admin {
		response.ErrRsp(c, -1, ErrForbidden.Error())
		return
	}
	if err := s.registry.DisconnectAll(actor); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	response.OkRsp(c)
}

func (s *Service) SourceSocket(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok || s.ready() != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	s.serveSource(c, actor)
}

func (s *Service) Events(c *gin.Context) {
	if _, ok := actorFrom(c); !ok || s.ready() != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	s.serveEvents(c)
}

func (s *Service) serveSource(c *gin.Context, actor Actor) {
	connection, err := sourceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Debugf("sources websocket upgrade: %s", err)
		return
	}
	stopWatcher := middleware.WatchWebSocket(c.Request.Context(), connection)
	defer stopWatcher()
	defer connection.Close()
	connection.SetReadLimit(maxControlMessage)
	_ = connection.SetReadDeadline(time.Now().Add(helloTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	})

	message, err := readControl(connection)
	if err != nil || message.Type != "hello" {
		writeControl(connection, controlResponse{Type: "error", Message: "hello required"})
		return
	}
	source, err := s.registry.RegisterSource(actor, Hello{Label: message.Label, Streams: message.Streams})
	if err != nil {
		writeControl(connection, controlResponse{Type: "error", Message: err.Error()})
		return
	}
	defer s.registry.DisconnectSource(source.ID)
	snapshot := s.registry.Snapshot()
	if err := writeControl(connection, controlResponse{Type: "source_ready", Source: &source, Snapshot: &snapshot}); err != nil {
		return
	}
	_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))

	done := make(chan struct{})
	defer close(done)
	go ping(connection, done)
	windowStart, count := time.Now(), 0
	for {
		message, err = readControl(connection)
		if err != nil {
			return
		}
		now := time.Now()
		if now.Sub(windowStart) >= messageWindow {
			windowStart, count = now, 0
		}
		count++
		if count > messageLimit {
			writeControl(connection, controlResponse{Type: "error", Message: "message rate exceeded"})
			return
		}
		if err := s.handleControl(connection, actor, source.ID, message); err != nil {
			return
		}
		_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	}
}

func (s *Service) handleControl(connection *websocket.Conn, actor Actor, sourceID string, message controlMessage) error {
	switch message.Type {
	case "claim":
		result, err := s.registry.Claim(actor, sourceID, message.StreamID, message.SinkID)
		if err != nil {
			var occupied *OccupiedError
			if errors.As(err, &occupied) {
				return writeControl(connection, controlResponse{
					Type:        "claim_refused",
					Message:     "slot_occupied",
					SinkID:      occupied.SinkID,
					Owner:       occupied.Owner,
					SourceLabel: occupied.SourceLabel,
					Since:       occupied.Since,
				})
			}
			return writeControl(connection, controlResponse{Type: "error", Message: err.Error()})
		}
		return writeControl(connection, controlResponse{Type: "claimed", Binding: &result.Binding, Token: result.Token})
	case "release":
		if err := s.registry.Release(actor, message.SinkID, ReasonReleased); err != nil {
			return writeControl(connection, controlResponse{Type: "error", Message: err.Error()})
		}
		return writeControl(connection, controlResponse{Type: "released", SinkID: message.SinkID})
	case "resume":
		binding, err := s.registry.Resume(actor, sourceID, message.StreamID, message.SinkID, message.Token)
		if err != nil {
			return writeControl(connection, controlResponse{Type: "error", Message: err.Error()})
		}
		return writeControl(connection, controlResponse{Type: "resumed", Binding: &binding})
	default:
		return writeControl(connection, controlResponse{Type: "error", Message: "unknown message type"})
	}
}

func (s *Service) serveEvents(c *gin.Context) {
	connection, err := sourceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Debugf("sources events upgrade: %s", err)
		return
	}
	stopWatcher := middleware.WatchWebSocket(c.Request.Context(), connection)
	defer stopWatcher()
	defer connection.Close()
	connection.SetReadLimit(1024)
	_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	})

	events, unsubscribe := s.registry.Subscribe()
	defer unsubscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case event, open := <-events:
			if !open || writeEvent(connection, event) != nil {
				return
			}
		case <-ticker.C:
			if connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(websocketWriteWait)) != nil {
				return
			}
		case <-done:
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (s *Service) ready() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initErr
}

func actorFrom(c *gin.Context) (Actor, bool) {
	principal, ok := middleware.CurrentPrincipal(c)
	if !ok {
		return Actor{}, false
	}
	return Actor{Username: principal.Username, Admin: principal.Role == authn.RoleAdmin}, true
}

func decodeHTTPRequest(c *gin.Context, destination any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxControlMessage)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	return requireJSONEnd(decoder)
}

func readControl(connection *websocket.Conn) (controlMessage, error) {
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		return controlMessage{}, err
	}
	if messageType != websocket.TextMessage {
		return controlMessage{}, errors.New("control messages must be JSON text")
	}
	var message controlMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&message); err != nil {
		return controlMessage{}, fmt.Errorf("decode control message: %w", err)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return controlMessage{}, fmt.Errorf("decode control message: %w", err)
	}
	return message, nil
}

func writeControl(connection *websocket.Conn, response controlResponse) error {
	_ = connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
	return connection.WriteJSON(response)
}

func writeEvent(connection *websocket.Conn, event Event) error {
	_ = connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
	return connection.WriteJSON(event)
}

func ping(connection *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(websocketWriteWait)) != nil {
				return
			}
		case <-done:
			return
		}
	}
}
