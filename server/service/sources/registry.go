package sources

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultLeaseGrace       = 60 * time.Second
	defaultSubscriberBuffer = 16
)

type scheduleFunc func(time.Duration, func()) func()

type RegistryOptions struct {
	Now              func() time.Time
	Random           io.Reader
	Schedule         scheduleFunc
	LeaseGrace       time.Duration
	SubscriberBuffer int
}

type binding struct {
	BindingView
	token        [32]byte
	expiry       uint64
	cancelExpiry func()
}

type Registry struct {
	mu       sync.Mutex
	sinks    map[string]Slot
	demands  map[string]Demand
	sources  map[string]Source
	bindings map[string]*binding
	subs     map[uint64]chan Event
	nextSub  uint64

	now              func() time.Time
	random           io.Reader
	schedule         scheduleFunc
	leaseGrace       time.Duration
	subscriberBuffer int
}

func NewRegistry(slots []Slot, options RegistryOptions) (*Registry, error) {
	validated, err := validateSlots(slots)
	if err != nil {
		return nil, err
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.Schedule == nil {
		options.Schedule = func(delay time.Duration, callback func()) func() {
			timer := time.AfterFunc(delay, callback)
			return func() { timer.Stop() }
		}
	}
	if options.LeaseGrace <= 0 {
		options.LeaseGrace = defaultLeaseGrace
	}
	if options.SubscriberBuffer <= 0 {
		options.SubscriberBuffer = defaultSubscriberBuffer
	}

	registry := &Registry{
		sinks:            make(map[string]Slot, len(validated)),
		demands:          make(map[string]Demand, len(validated)),
		sources:          make(map[string]Source),
		bindings:         make(map[string]*binding),
		subs:             make(map[uint64]chan Event),
		now:              options.Now,
		random:           options.Random,
		schedule:         options.Schedule,
		leaseGrace:       options.LeaseGrace,
		subscriberBuffer: options.SubscriberBuffer,
	}
	for _, slot := range validated {
		registry.sinks[slot.ID] = slot
	}
	return registry, nil
}

func (r *Registry) RegisterSource(actor Actor, hello Hello) (Source, error) {
	if actor.Username == "" {
		return Source{}, ErrForbidden
	}
	validated, err := validateHello(hello)
	if err != nil {
		return Source{}, fmt.Errorf("%w: %w", ErrInvalidMessage, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id, err := r.newIDLocked("src_", 12, func(candidate string) bool {
		_, exists := r.sources[candidate]
		return exists
	})
	if err != nil {
		return Source{}, fmt.Errorf("source id: %w", err)
	}
	source := Source{
		ID:          id,
		Owner:       actor.Username,
		Agent:       "browser",
		Label:       validated.Label,
		Streams:     validated.Streams,
		ConnectedAt: r.now().UTC(),
	}
	r.sources[id] = source
	copy := cloneSource(source)
	r.emitLocked(Event{Type: "source_added", Source: &copy})
	return copy, nil
}

func (r *Registry) DisconnectSource(sourceID string) {
	r.mu.Lock()
	source, exists := r.sources[sourceID]
	if !exists {
		r.mu.Unlock()
		return
	}
	delete(r.sources, sourceID)
	now := r.now().UTC()
	type expiry struct {
		sinkID     string
		generation uint64
	}
	var expiries []expiry
	for sinkID, current := range r.bindings {
		if current.SourceID != sourceID {
			continue
		}
		current.State = StateOrphaned
		current.ExpiresAt = now.Add(r.leaseGrace)
		current.expiry++
		generation := current.expiry
		expiries = append(expiries, expiry{sinkID: sinkID, generation: generation})
		view := current.BindingView
		r.emitLocked(Event{Type: "binding_state", Binding: &view})
	}
	copy := cloneSource(source)
	r.emitLocked(Event{Type: "source_removed", Source: &copy})
	r.mu.Unlock()

	for _, pending := range expiries {
		cancel := r.schedule(r.leaseGrace, func() { r.expire(pending.sinkID, pending.generation) })
		r.mu.Lock()
		current := r.bindings[pending.sinkID]
		if current != nil && current.State == StateOrphaned && current.expiry == pending.generation {
			current.cancelExpiry = cancel
		} else {
			cancel()
		}
		r.mu.Unlock()
	}
}

func (r *Registry) Claim(actor Actor, sourceID, streamID, sinkID string) (ClaimResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	source, stream, sink, err := r.claimInputsLocked(actor, sourceID, streamID, sinkID)
	if err != nil {
		return ClaimResult{}, err
	}
	if current := r.bindings[sinkID]; current != nil {
		return ClaimResult{}, &OccupiedError{
			SinkID:      sinkID,
			Owner:       current.Owner,
			SourceLabel: current.SourceLabel,
			Since:       current.StartedAt,
		}
	}
	if stream.Kind != sink.Kind {
		return ClaimResult{}, fmt.Errorf("%w: stream kind %q cannot fill %q", ErrInvalidMessage, stream.Kind, sink.Kind)
	}

	current := &binding{BindingView: BindingView{
		SinkID:      sinkID,
		SourceID:    sourceID,
		StreamID:    streamID,
		Owner:       actor.Username,
		SourceLabel: source.Label,
		StreamLabel: stream.Label,
		State:       StateClaimed,
		StartedAt:   r.now().UTC(),
	}}
	if _, err := io.ReadFull(r.random, current.token[:]); err != nil {
		return ClaimResult{}, fmt.Errorf("lease token: %w", err)
	}
	r.bindings[sinkID] = current
	view := current.BindingView
	r.emitLocked(Event{Type: "binding_added", Binding: &view})
	return ClaimResult{Binding: view, Token: base64.RawURLEncoding.EncodeToString(current.token[:])}, nil
}

func (r *Registry) Resume(actor Actor, sourceID, streamID, sinkID, encodedToken string) (BindingView, error) {
	token, err := base64.RawURLEncoding.DecodeString(encodedToken)
	if err != nil || len(token) != 32 {
		return BindingView{}, ErrInvalidToken
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	source, stream, sink, err := r.claimInputsLocked(actor, sourceID, streamID, sinkID)
	if err != nil {
		return BindingView{}, err
	}
	current := r.bindings[sinkID]
	if current == nil || current.State != StateOrphaned || current.Owner != actor.Username ||
		subtle.ConstantTimeCompare(current.token[:], token) != 1 {
		return BindingView{}, ErrInvalidToken
	}
	if !current.ExpiresAt.After(r.now()) {
		r.removeBindingLocked(sinkID, ReasonLeaseExpired)
		return BindingView{}, ErrInvalidToken
	}
	if stream.Kind != sink.Kind {
		return BindingView{}, fmt.Errorf("%w: stream kind %q cannot fill %q", ErrInvalidMessage, stream.Kind, sink.Kind)
	}
	if current.cancelExpiry != nil {
		current.cancelExpiry()
	}
	current.cancelExpiry = nil
	current.expiry++
	current.SourceID = sourceID
	current.StreamID = streamID
	current.SourceLabel = source.Label
	current.StreamLabel = stream.Label
	current.State = StateClaimed
	current.ExpiresAt = time.Time{}
	view := current.BindingView
	r.emitLocked(Event{Type: "binding_state", Binding: &view})
	return view, nil
}

func (r *Registry) Release(actor Actor, sinkID string, reason TerminationReason) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.bindings[sinkID]
	if current == nil {
		return fmt.Errorf("%w: binding %s", ErrNotFound, sinkID)
	}
	if !actor.Admin && current.Owner != actor.Username {
		return ErrForbidden
	}
	if reason == "" {
		reason = ReasonReleased
	}
	r.removeBindingLocked(sinkID, reason)
	return nil
}

func (r *Registry) DisconnectAll(actor Actor) error {
	if !actor.Admin {
		return ErrForbidden
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for sinkID := range r.bindings {
		r.removeBindingLocked(sinkID, ReasonAdminDisconnect)
	}
	return nil
}

func (r *Registry) SyncSlots(slots []Slot) error {
	validated, err := validateSlots(slots)
	if err != nil {
		return err
	}
	next := make(map[string]Slot, len(validated))
	for _, slot := range validated {
		next[slot.ID] = slot
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for sinkID := range r.bindings {
		previous := r.sinks[sinkID]
		current, exists := next[sinkID]
		if !exists {
			r.removeBindingLocked(sinkID, ReasonSlotRemoved)
		} else if current != previous {
			r.removeBindingLocked(sinkID, ReasonSlotChanged)
		}
	}
	r.sinks = next
	for sinkID := range r.demands {
		if _, exists := next[sinkID]; !exists {
			delete(r.demands, sinkID)
		}
	}
	sinks := r.snapshotLocked().Sinks
	r.emitLocked(Event{Type: "sinks_changed", Sinks: sinks})
	return nil
}

func (r *Registry) AuthorizeFrame(sourceID, streamID, sinkID string, kind Kind) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	source, exists := r.sources[sourceID]
	if !exists {
		return fmt.Errorf("%w: source %s", ErrNotFound, sourceID)
	}
	streamFound := false
	for _, stream := range source.Streams {
		if stream.ID == streamID {
			if stream.Kind != kind {
				return fmt.Errorf("%w: stream kind %q", ErrInvalidMessage, stream.Kind)
			}
			streamFound = true
			break
		}
	}
	if !streamFound {
		return fmt.Errorf("%w: stream %s", ErrNotFound, streamID)
	}
	sink, exists := r.sinks[sinkID]
	if !exists {
		return fmt.Errorf("%w: sink %s", ErrNotFound, sinkID)
	}
	if sink.Kind != kind {
		return fmt.Errorf("%w: sink kind %q", ErrInvalidMessage, sink.Kind)
	}
	current := r.bindings[sinkID]
	if current == nil || current.SourceID != sourceID || current.StreamID != streamID || current.State == StateOrphaned {
		return ErrForbidden
	}
	return nil
}

func (r *Registry) SetDemand(sinkID string, demand Demand) error {
	if err := validateDemand(demand); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sinks[sinkID]; !exists {
		return fmt.Errorf("%w: sink %s", ErrNotFound, sinkID)
	}
	if demand.Streaming && demand.Since.IsZero() {
		demand.Since = r.now().UTC()
	}
	if !demand.Streaming {
		demand = Demand{}
	}
	r.demands[sinkID] = demand
	copy := demand
	r.emitLocked(Event{Type: "demand", SinkID: sinkID, Demand: &copy})
	return nil
}

func (r *Registry) SetBindingState(sinkID string, state BindingState) error {
	if state != StateClaimed && state != StateStreaming && state != StateSuspended {
		return fmt.Errorf("invalid binding state %q", state)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.bindings[sinkID]
	if current == nil {
		return fmt.Errorf("%w: binding %s", ErrNotFound, sinkID)
	}
	if current.State == StateOrphaned {
		return errors.New("orphaned binding")
	}
	current.State = state
	view := current.BindingView
	r.emitLocked(Event{Type: "binding_state", Binding: &view})
	return nil
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotLocked()
}

func (r *Registry) Subscribe() (<-chan Event, func()) {
	r.mu.Lock()
	id := r.nextSub
	r.nextSub++
	channel := make(chan Event, r.subscriberBuffer)
	snapshot := r.snapshotLocked()
	channel <- Event{Type: "snapshot", Snapshot: &snapshot}
	r.subs[id] = channel
	r.mu.Unlock()

	var once sync.Once
	return channel, func() {
		once.Do(func() {
			r.mu.Lock()
			if current, exists := r.subs[id]; exists {
				delete(r.subs, id)
				close(current)
			}
			r.mu.Unlock()
		})
	}
}

func (r *Registry) claimInputsLocked(actor Actor, sourceID, streamID, sinkID string) (Source, Stream, Slot, error) {
	if actor.Username == "" {
		return Source{}, Stream{}, Slot{}, ErrForbidden
	}
	source, exists := r.sources[sourceID]
	if !exists {
		return Source{}, Stream{}, Slot{}, fmt.Errorf("%w: source %s", ErrNotFound, sourceID)
	}
	if source.Owner != actor.Username {
		return Source{}, Stream{}, Slot{}, ErrForbidden
	}
	var stream Stream
	found := false
	for _, candidate := range source.Streams {
		if candidate.ID == streamID {
			stream = candidate
			found = true
			break
		}
	}
	if !found {
		return Source{}, Stream{}, Slot{}, fmt.Errorf("%w: stream %s", ErrNotFound, streamID)
	}
	sink, exists := r.sinks[sinkID]
	if !exists {
		return Source{}, Stream{}, Slot{}, fmt.Errorf("%w: sink %s", ErrNotFound, sinkID)
	}
	return source, stream, sink, nil
}

func (r *Registry) expire(sinkID string, generation uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.bindings[sinkID]
	if current == nil || current.State != StateOrphaned || current.expiry != generation {
		return
	}
	if current.ExpiresAt.After(r.now()) {
		return
	}
	r.removeBindingLocked(sinkID, ReasonLeaseExpired)
}

func (r *Registry) removeBindingLocked(sinkID string, reason TerminationReason) {
	current := r.bindings[sinkID]
	if current == nil {
		return
	}
	if current.cancelExpiry != nil {
		current.cancelExpiry()
	}
	view := current.BindingView
	delete(r.bindings, sinkID)
	r.emitLocked(Event{Type: "binding_removed", Binding: &view, Reason: reason})
}

func (r *Registry) emitLocked(event Event) {
	for id, subscriber := range r.subs {
		select {
		case subscriber <- event:
		default:
			delete(r.subs, id)
			close(subscriber)
		}
	}
}

func (r *Registry) snapshotLocked() Snapshot {
	snapshot := Snapshot{
		Sinks:    make([]Sink, 0, len(r.sinks)),
		Sources:  make([]Source, 0, len(r.sources)),
		Bindings: make([]BindingView, 0, len(r.bindings)),
	}
	for _, source := range r.sources {
		snapshot.Sources = append(snapshot.Sources, cloneSource(source))
	}
	for _, current := range r.bindings {
		snapshot.Bindings = append(snapshot.Bindings, current.BindingView)
	}
	for _, slot := range r.sinks {
		sink := Sink{Slot: slot, SlotNumber: slotIndex(slot.ID), Demand: r.demands[slot.ID]}
		if current := r.bindings[slot.ID]; current != nil {
			view := current.BindingView
			sink.Binding = &view
		}
		sink.Output = outputState(sink)
		snapshot.Sinks = append(snapshot.Sinks, sink)
	}
	slices.SortFunc(snapshot.Sources, func(a, b Source) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(snapshot.Bindings, func(a, b BindingView) int { return strings.Compare(a.SinkID, b.SinkID) })
	slices.SortFunc(snapshot.Sinks, func(a, b Sink) int { return strings.Compare(a.ID, b.ID) })
	return snapshot
}

func (r *Registry) newIDLocked(prefix string, size int, exists func(string) bool) (string, error) {
	buffer := make([]byte, size)
	for range 4 {
		if _, err := io.ReadFull(r.random, buffer); err != nil {
			return "", err
		}
		id := prefix + base64.RawURLEncoding.EncodeToString(buffer)
		if !exists(id) {
			return id, nil
		}
	}
	return "", errors.New("id collision")
}

func outputState(sink Sink) OutputState {
	if !sink.Demand.Streaming {
		return OutputIdle
	}
	if sink.Binding != nil && sink.Binding.State == StateStreaming {
		return OutputSource
	}
	if sink.Kind == KindCamera {
		return OutputBlack
	}
	return OutputSilence
}

func validateDemand(demand Demand) error {
	if !demand.Streaming {
		return nil
	}
	if demand.Width < 0 || demand.Width > 7680 || demand.Height < 0 || demand.Height > 4320 || demand.FPS < 0 || demand.FPS > 240 {
		return errors.New("invalid demand")
	}
	return nil
}

func cloneSource(source Source) Source {
	source.Streams = cloneStreams(source.Streams)
	return source
}
