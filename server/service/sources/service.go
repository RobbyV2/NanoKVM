package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	maxBinaryMessage   = (64 << 10) + 32
	usbFrameMagic      = "NKUF"
	helloTimeout       = 10 * time.Second
	websocketTimeout   = 60 * time.Second
	websocketWriteWait = 5 * time.Second
	pingInterval       = 20 * time.Second
	messageWindow      = 10 * time.Second
	messageLimit       = 64
	mediaMessageLimit  = MaxSlots * 60 * 10
)

type Service struct {
	registry *Registry
	ingress  FrameIngress
	egress   FrameEgress
	slots    SlotManager
	mu       sync.Mutex
	usb      USBBackend
	active   map[string]BinarySession
	// One detach function per speaker slot a browser is draining. A slot
	// carries at most one binding, so at most one entry per sink.
	speakers map[string]func()
}

type BinarySession interface {
	Receive([]byte) error
	Ready() <-chan error
	Done() <-chan error
	Close() error
}

type USBBackend interface {
	Start(context.Context, Stream, func([]byte) error) (BinarySession, error)
}

type SlotManager interface {
	SetMediaSlots(context.Context, []string, []string, []string) error
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
	Type        string            `json:"type"`
	Message     string            `json:"message,omitempty"`
	Source      *Source           `json:"source,omitempty"`
	Snapshot    *Snapshot         `json:"snapshot,omitempty"`
	Binding     *BindingView      `json:"binding,omitempty"`
	Token       string            `json:"token,omitempty"`
	SinkID      string            `json:"sink_id,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	SourceLabel string            `json:"source_label,omitempty"`
	Since       time.Time         `json:"since,omitempty"`
	StreamID    string            `json:"stream_id,omitempty"`
	Sequence    uint32            `json:"sequence,omitempty"`
	Takeover    string            `json:"takeover,omitempty"`
	Reason      TerminationReason `json:"reason,omitempty"`
}

type frameFailure struct {
	frame MediaFrame
	err   error
}

type slotsRequest struct {
	Slots []Slot `json:"slots"`
}

type bindingRequest struct {
	SourceID string `json:"source_id"`
	StreamID string `json:"stream_id"`
	SinkID   string `json:"sink_id,omitempty"`
	Token    string `json:"token,omitempty"`
}

type claimRefusal struct {
	SinkID      string    `json:"sink_id"`
	Owner       string    `json:"owner"`
	SourceLabel string    `json:"source_label"`
	Since       time.Time `json:"since"`
	Takeover    string    `json:"takeover"`
}

func (e *frameFailure) Error() string { return e.err.Error() }
func (e *frameFailure) Unwrap() error { return e.err }

var sourceUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     middleware.CheckWebSocketOrigin,
}

func NewService() *Service {
	registry, _ := NewRegistry(nil, RegistryOptions{})
	return &Service{registry: registry, active: make(map[string]BinarySession), speakers: make(map[string]func())}
}

func NewServiceWith(registry *Registry) *Service {
	return &Service{registry: registry, active: make(map[string]BinarySession), speakers: make(map[string]func())}
}

func (s *Service) SetIngress(ingress FrameIngress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ingress = ingress
	if egress, ok := ingress.(FrameEgress); ok {
		s.egress = egress
	}
}

func (s *Service) SetEgress(egress FrameEgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.egress = egress
}

func (s *Service) SetSlotManager(manager SlotManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slots = manager
}

func (s *Service) SetUSBBackend(backend USBBackend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usb = backend
}

func (s *Service) usbBackend() USBBackend {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usb
}

func (s *Service) Registry() *Registry { return s.registry }

func (s *Service) Get(c *gin.Context) {
	var response proto.Response
	snapshot := s.registry.Snapshot()
	s.attachLatency(snapshot.Sinks)
	response.OkRspWithData(c, &snapshot)
}

// The registry never sees a frame, so the skew is measured in the media worker
// and joined onto the sinks here, where it is read.
func (s *Service) attachLatency(sinks []Sink) {
	s.mu.Lock()
	ingress := s.ingress
	s.mu.Unlock()
	if ingress == nil {
		return
	}
	measured := ingress.Latency()
	for index := range sinks {
		if summary, ok := measured[sinks[index].ID]; ok {
			sinks[index].Latency = &summary
		}
	}
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
	slots, err := validateSlots(request.Slots)
	if err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	var cameras, microphones, speakers []string
	for _, slot := range slots {
		switch slot.Kind {
		case KindCamera:
			cameras = append(cameras, slot.Label)
		case KindSpeaker:
			speakers = append(speakers, slot.Label)
		default:
			microphones = append(microphones, slot.Label)
		}
	}
	s.mu.Lock()
	manager := s.slots
	s.mu.Unlock()
	if manager == nil {
		response.ErrRsp(c, -2, "media profile manager is unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
	defer cancel()
	if err := manager.SetMediaSlots(ctx, cameras, microphones, speakers); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
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
	s.detach(c.Param("sink"))
	s.stopUSB(c.Param("sink"))
	response.OkRsp(c)
}

func (s *Service) Claim(c *gin.Context) {
	var response proto.Response
	actor, request, ok := bindingArguments(c)
	if !ok {
		return
	}
	if err := s.requireMediaStream(request); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	result, err := s.registry.Claim(actor, request.SourceID, request.StreamID, request.SinkID)
	if err != nil {
		writeClaimError(c, actor, err)
		return
	}
	response.OkRspWithData(c, &result)
}

func (s *Service) Resume(c *gin.Context) {
	var response proto.Response
	actor, request, ok := bindingArguments(c)
	if !ok {
		return
	}
	if err := s.requireMediaStream(request); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	binding, err := s.registry.Resume(actor, request.SourceID, request.StreamID, request.SinkID, request.Token)
	if err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	response.OkRspWithData(c, &binding)
}

func (s *Service) Takeover(c *gin.Context) {
	var response proto.Response
	actor, request, ok := bindingArguments(c)
	if !ok {
		return
	}
	if err := s.requireMediaStream(request); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	result, err := s.registry.Takeover(actor, request.SourceID, request.StreamID, request.SinkID)
	if err != nil {
		writeClaimError(c, actor, err)
		return
	}
	s.detach(request.SinkID)
	response.OkRspWithData(c, &result)
}

func (s *Service) requireMediaStream(request bindingRequest) error {
	stream, err := s.registry.Stream(request.SourceID, request.StreamID)
	if err != nil {
		return fmt.Errorf("%w: stream %s", ErrNotFound, request.StreamID)
	}
	if stream.Kind == KindUSBDevice {
		return fmt.Errorf("%w: USB devices bind over the source socket", ErrForbidden)
	}
	return nil
}

func bindingArguments(c *gin.Context) (Actor, bindingRequest, bool) {
	var response proto.Response
	actor, ok := actorFrom(c)
	if !ok {
		response.ErrRsp(c, -1, ErrForbidden.Error())
		return Actor{}, bindingRequest{}, false
	}
	var request bindingRequest
	if err := decodeHTTPRequest(c, &request); err != nil {
		response.ErrRsp(c, -1, "invalid arguments")
		return Actor{}, bindingRequest{}, false
	}
	if sink := c.Param("sink"); sink != "" {
		request.SinkID = sink
	}
	return actor, request, true
}

func writeClaimError(c *gin.Context, actor Actor, err error) {
	var response proto.Response
	var occupied *OccupiedError
	if !errors.As(err, &occupied) {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	response.Err(-3, "slot_occupied")
	response.Data = claimRefusal{
		SinkID:      occupied.SinkID,
		Owner:       occupied.Owner,
		SourceLabel: occupied.SourceLabel,
		Since:       occupied.Since,
		Takeover:    takeoverHint(actor),
	}
	c.JSON(http.StatusOK, &response)
}

func takeoverHint(actor Actor) string {
	if actor.Admin {
		return "immediate"
	}
	return "refused"
}

func (s *Service) DisconnectAll(c *gin.Context) {
	var response proto.Response
	actor, ok := actorFrom(c)
	if !ok || !actor.Admin {
		response.ErrRsp(c, -1, ErrForbidden.Error())
		return
	}
	snapshot := s.registry.Snapshot()
	if err := s.registry.DisconnectAll(actor); err != nil {
		response.ErrRsp(c, -2, err.Error())
		return
	}
	for _, binding := range snapshot.Bindings {
		s.detach(binding.SinkID)
	}
	s.stopAllUSB()
	s.stopAllSpeakers()
	current := s.registry.Snapshot()
	response.OkRspWithData(c, &current)
}

func (s *Service) SourceSocket(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	s.serveSource(c, actor)
}

func (s *Service) Events(c *gin.Context) {
	if _, ok := actorFrom(c); !ok {
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
	writer := &sourceWriter{connection: connection}
	connection.SetReadLimit(maxControlMessage)
	_ = connection.SetReadDeadline(time.Now().Add(helloTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	})

	message, err := readControl(connection)
	if err != nil || message.Type != "hello" {
		writer.JSON(controlResponse{Type: "error", Message: "hello required"})
		return
	}
	source, err := s.registry.RegisterSource(actor, Hello{Label: message.Label, Streams: message.Streams})
	if err != nil {
		writer.JSON(controlResponse{Type: "error", Message: err.Error()})
		return
	}
	defer func() {
		snapshot := s.registry.Snapshot()
		s.registry.DisconnectSource(source.ID)
		for _, binding := range snapshot.Bindings {
			if binding.SourceID == source.ID {
				s.detach(binding.SinkID)
				s.stopSpeaker(binding.SinkID)
			}
		}
	}()
	snapshot := s.registry.Snapshot()
	if err := writer.JSON(controlResponse{Type: "source_ready", Source: &source, Snapshot: &snapshot}); err != nil {
		return
	}
	_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	connection.SetReadLimit(maxMediaMessage)

	events, unsubscribe := s.registry.Subscribe()
	defer unsubscribe()
	done := make(chan struct{})
	defer close(done)
	ctx, cancel := context.WithCancel(c.Request.Context())
	var binarySession BinarySession
	defer func() {
		cancel()
		if binarySession != nil {
			_ = binarySession.Close()
		}
	}()
	go pingWriter(writer, done)
	go watchTerminations(writer, source.ID, events, done)
	windowStart, count, mediaCount := time.Now(), 0, 0
	for {
		messageType, data, readErr := connection.ReadMessage()
		err = readErr
		if err != nil {
			return
		}
		if messageType == websocket.BinaryMessage {
			if len(data) >= 4 && string(data[:4]) == usbFrameMagic {
				if binarySession == nil || len(data) > maxBinaryMessage || binarySession.Receive(data) != nil {
					_ = writer.JSON(controlResponse{Type: "error", Message: "invalid USB transfer"})
					return
				}
				_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
				continue
			}
			now := time.Now()
			if now.Sub(windowStart) >= messageWindow {
				windowStart, count, mediaCount = now, 0, 0
			}
			mediaCount++
			if mediaCount > mediaMessageLimit {
				writer.JSON(controlResponse{Type: "error", Message: "media frame rate exceeded"})
				return
			}
			if err := s.handleMedia(c.Request.Context(), writer, source.ID, data); err != nil {
				response := controlResponse{Type: "frame_error", Message: err.Error()}
				var failure *frameFailure
				if errors.As(err, &failure) {
					response.SinkID = failure.frame.SinkID
					response.StreamID = failure.frame.StreamID
					response.Sequence = failure.frame.Sequence
				}
				if writer.JSON(response) != nil {
					return
				}
			}
			_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
			continue
		}
		if len(data) > maxControlMessage {
			writer.JSON(controlResponse{Type: "error", Message: "control message is too large"})
			return
		}
		message, err = decodeControl(messageType, data)
		if err != nil {
			writer.JSON(controlResponse{Type: "error", Message: err.Error()})
			return
		}
		now := time.Now()
		if now.Sub(windowStart) >= messageWindow {
			windowStart, count, mediaCount = now, 0, 0
		}
		count++
		if count > messageLimit {
			writer.JSON(controlResponse{Type: "error", Message: "message rate exceeded"})
			return
		}
		if err := s.handleControl(ctx, writer, actor, source.ID, message, &binarySession); err != nil {
			return
		}
		_ = connection.SetReadDeadline(time.Now().Add(websocketTimeout))
	}
}

func (s *Service) handleMedia(ctx context.Context, writer *sourceWriter, sourceID string, data []byte) error {
	frame, err := parseMediaFrame(data)
	if err != nil {
		return err
	}
	frame.SourceID = sourceID
	kind := KindCamera
	if frame.Kind == MediaKindPCMS16LE {
		kind = KindMicrophone
	}
	if err := s.registry.AuthorizeFrame(sourceID, frame.StreamID, frame.SinkID, kind); err != nil {
		return &frameFailure{frame: frame, err: err}
	}
	s.mu.Lock()
	ingress := s.ingress
	s.mu.Unlock()
	if ingress == nil {
		return &frameFailure{frame: frame, err: errors.New("media output is unavailable")}
	}
	if err := ingress.Ingest(ctx, frame); err != nil {
		// A sink the host has stopped reading refuses every frame until it
		// starts again, and the browser deliberately keeps capturing across
		// that gap so the picture resumes instantly. Reporting each refusal as
		// a frame error put "media sink is not demanded" in front of the
		// operator thirty times a second for something no operator can act on.
		// The frame is finished either way; acknowledging it keeps the source's
		// in-flight window moving and says nothing that is not true.
		if !errors.Is(err, ErrNotDemanded) {
			return &frameFailure{frame: frame, err: err}
		}
	}
	return writer.JSON(controlResponse{
		Type: "frame_ack", SinkID: frame.SinkID, StreamID: frame.StreamID, Sequence: frame.Sequence,
	})
}

func (s *Service) handleControl(ctx context.Context, writer *sourceWriter, actor Actor, sourceID string, message controlMessage, active *BinarySession) error {
	switch message.Type {
	case "claim":
		result, err := s.registry.Claim(actor, sourceID, message.StreamID, message.SinkID)
		if err != nil {
			var occupied *OccupiedError
			if errors.As(err, &occupied) {
				return writer.JSON(controlResponse{
					Type:        "claim_refused",
					Message:     "slot_occupied",
					SinkID:      occupied.SinkID,
					Owner:       occupied.Owner,
					SourceLabel: occupied.SourceLabel,
					Since:       occupied.Since,
					Takeover:    takeoverHint(actor),
				})
			}
			return writer.JSON(controlResponse{Type: "error", Message: err.Error()})
		}
		if result.Stream.Kind == KindUSBDevice {
			backend := s.usbBackend()
			if backend == nil || *active != nil {
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return writer.JSON(controlResponse{Type: "error", Message: "WebUSB relay is unavailable"})
			}
			session, startErr := backend.Start(ctx, result.Stream, writer.Binary)
			if startErr != nil {
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return writer.JSON(controlResponse{Type: "error", Message: startErr.Error()})
			}
			*active = session
			s.trackUSB(message.SinkID, session)
			if err := writer.JSON(controlResponse{Type: "claimed", Binding: &result.Binding, Token: result.Token}); err != nil {
				_ = session.Close()
				s.untrackUSB(message.SinkID, session)
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				*active = nil
				return err
			}
			s.watchUSB(ctx, writer, actor, message.SinkID, session)
			return nil
		}
		if result.Stream.Kind == KindSpeaker {
			if err := s.startSpeaker(writer, message.SinkID, message.StreamID); err != nil {
				// The refusal goes out before the slot is given back, so the
				// browser reads why it failed rather than a bare release the
				// termination watcher would otherwise deliver first.
				refused := writer.JSON(controlResponse{Type: "error", Message: err.Error(), SinkID: message.SinkID})
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return refused
			}
			if err := writer.JSON(controlResponse{Type: "claimed", Binding: &result.Binding, Token: result.Token}); err != nil {
				s.stopSpeaker(message.SinkID)
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return err
			}
			return nil
		}
		return writer.JSON(controlResponse{Type: "claimed", Binding: &result.Binding, Token: result.Token})
	case "release":
		if err := s.registry.Release(actor, message.SinkID, ReasonReleased); err != nil {
			return writer.JSON(controlResponse{Type: "error", Message: err.Error()})
		}
		s.detach(message.SinkID)
		s.stopSpeaker(message.SinkID)
		if *active != nil {
			session := *active
			s.untrackUSB(message.SinkID, session)
			_ = session.Close()
			*active = nil
		}
		return writer.JSON(controlResponse{Type: "released", SinkID: message.SinkID})
	case "resume":
		binding, err := s.registry.Resume(actor, sourceID, message.StreamID, message.SinkID, message.Token)
		if err != nil {
			return writer.JSON(controlResponse{Type: "error", Message: err.Error()})
		}
		stream, err := s.registry.Stream(sourceID, message.StreamID)
		if err == nil && stream.Kind == KindUSBDevice {
			backend := s.usbBackend()
			if backend == nil || *active != nil {
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return writer.JSON(controlResponse{Type: "error", Message: "WebUSB relay is unavailable"})
			}
			session, startErr := backend.Start(ctx, stream, writer.Binary)
			if startErr != nil {
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return writer.JSON(controlResponse{Type: "error", Message: startErr.Error()})
			}
			*active = session
			s.trackUSB(message.SinkID, session)
			if err := writer.JSON(controlResponse{Type: "resumed", Binding: &binding}); err != nil {
				_ = session.Close()
				s.untrackUSB(message.SinkID, session)
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				*active = nil
				return err
			}
			s.watchUSB(ctx, writer, actor, message.SinkID, session)
			return nil
		}
		if err == nil && stream.Kind == KindSpeaker {
			if err := s.startSpeaker(writer, message.SinkID, message.StreamID); err != nil {
				refused := writer.JSON(controlResponse{Type: "error", Message: err.Error(), SinkID: message.SinkID})
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return refused
			}
			if err := writer.JSON(controlResponse{Type: "resumed", Binding: &binding}); err != nil {
				s.stopSpeaker(message.SinkID)
				_ = s.registry.Release(actor, message.SinkID, ReasonReleased)
				return err
			}
			return nil
		}
		return writer.JSON(controlResponse{Type: "resumed", Binding: &binding})
	default:
		return writer.JSON(controlResponse{Type: "error", Message: "unknown message type"})
	}
}

func (s *Service) watchUSB(ctx context.Context, writer *sourceWriter, actor Actor, sinkID string, session BinarySession) {
	go func() {
		defer s.untrackUSB(sinkID, session)
		if err := <-session.Ready(); err != nil {
			_ = writer.JSON(controlResponse{Type: "error", Message: err.Error(), SinkID: sinkID})
			if ctx.Err() == nil {
				_ = s.registry.Release(actor, sinkID, ReasonReleased)
				_ = writer.Close()
			}
			return
		}
		_ = s.registry.SetBindingState(sinkID, StateStreaming)
		if err := <-session.Done(); err != nil {
			_ = writer.JSON(controlResponse{Type: "error", Message: err.Error(), SinkID: sinkID})
		}
		if ctx.Err() != nil {
			return
		}
		if s.isUSBTracked(sinkID, session) {
			_ = s.registry.Release(actor, sinkID, ReasonReleased)
			_ = writer.Close()
		}
	}()
}

// startSpeaker points a speaker slot's capture at this socket. Encoding runs on
// the media backend's delivery goroutine and every write carries the socket's
// own deadline, so a browser that stops reading loses its own stream and
// nothing else.
func (s *Service) startSpeaker(writer *sourceWriter, sinkID, streamID string) error {
	s.mu.Lock()
	egress := s.egress
	s.mu.Unlock()
	if egress == nil {
		return errors.New("speaker capture is unavailable")
	}
	detach, err := egress.Attach(sinkID, func(frame MediaFrame) error {
		frame.StreamID = streamID
		data, err := encodeMediaFrame(frame)
		if err != nil {
			return err
		}
		return writer.Binary(data)
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	previous := s.speakers[sinkID]
	s.speakers[sinkID] = detach
	s.mu.Unlock()
	if previous != nil {
		previous()
	}
	return nil
}

func (s *Service) stopSpeaker(sinkID string) {
	s.mu.Lock()
	detach := s.speakers[sinkID]
	delete(s.speakers, sinkID)
	s.mu.Unlock()
	if detach != nil {
		detach()
	}
}

func (s *Service) stopAllSpeakers() {
	s.mu.Lock()
	detachers := s.speakers
	s.speakers = make(map[string]func())
	s.mu.Unlock()
	for _, detach := range detachers {
		detach()
	}
}

func (s *Service) trackUSB(sinkID string, session BinarySession) {
	s.mu.Lock()
	s.active[sinkID] = session
	s.mu.Unlock()
}

func (s *Service) untrackUSB(sinkID string, session BinarySession) {
	s.mu.Lock()
	if s.active[sinkID] == session {
		delete(s.active, sinkID)
	}
	s.mu.Unlock()
}

func (s *Service) isUSBTracked(sinkID string, session BinarySession) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[sinkID] == session
}

func (s *Service) stopUSB(sinkID string) {
	s.mu.Lock()
	session := s.active[sinkID]
	delete(s.active, sinkID)
	s.mu.Unlock()
	if session != nil {
		_ = session.Close()
	}
}

func (s *Service) stopAllUSB() {
	s.mu.Lock()
	sessions := s.active
	s.active = make(map[string]BinarySession)
	s.mu.Unlock()
	for _, session := range sessions {
		_ = session.Close()
	}
}

func (s *Service) detach(sinkID string) {
	s.mu.Lock()
	ingress := s.ingress
	s.mu.Unlock()
	if ingress != nil {
		ingress.Detach(sinkID)
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

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func readControl(connection *websocket.Conn) (controlMessage, error) {
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		return controlMessage{}, err
	}
	return decodeControl(messageType, data)
}

func decodeControl(messageType int, data []byte) (controlMessage, error) {
	if messageType != websocket.TextMessage {
		return controlMessage{}, errors.New("control messages must be JSON text")
	}
	if len(data) > maxControlMessage {
		return controlMessage{}, errors.New("control message is too large")
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

type sourceWriter struct {
	mu         sync.Mutex
	connection *websocket.Conn
}

func (w *sourceWriter) JSON(response controlResponse) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return writeControl(w.connection, response)
}

func (w *sourceWriter) Binary(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
	return w.connection.WriteMessage(websocket.BinaryMessage, data)
}

func (w *sourceWriter) Ping() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(websocketWriteWait))
}

func (w *sourceWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.connection.Close()
}

func writeControl(connection *websocket.Conn, response controlResponse) error {
	_ = connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
	return connection.WriteJSON(response)
}

func writeEvent(connection *websocket.Conn, event Event) error {
	_ = connection.SetWriteDeadline(time.Now().Add(websocketWriteWait))
	return connection.WriteJSON(event)
}

func watchTerminations(writer *sourceWriter, sourceID string, events <-chan Event, done <-chan struct{}) {
	for {
		select {
		case event, open := <-events:
			if !open {
				_ = writer.Close()
				return
			}
			if event.Type != "binding_removed" || event.Binding == nil || event.Binding.SourceID != sourceID {
				continue
			}
			if writer.JSON(controlResponse{Type: "released", SinkID: event.Binding.SinkID, Reason: event.Reason}) != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func pingWriter(writer *sourceWriter, done <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if writer.Ping() != nil {
				return
			}
		case <-done:
			return
		}
	}
}
