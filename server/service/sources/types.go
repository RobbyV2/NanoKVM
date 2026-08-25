package sources

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	SchemaVersion = 1
	MaxSlots      = 8
	MaxStreams    = 16
	MaxFormats    = 8
)

type Kind string

const (
	KindCamera     Kind = "camera"
	KindMicrophone Kind = "microphone"
	// A speaker slot runs the other way: the target host plays audio into the
	// gadget's USB OUT endpoint and this device reads it, so frames leave the
	// sink for the browser instead of arriving from it. Everything else about
	// a slot - claims, leases, takeover, terminations - is unchanged.
	KindSpeaker   Kind = "speaker"
	KindUSBDevice Kind = "usb_device"
	HybridSinkID       = "ffs.hybrid"
)

type BindingState string

const (
	StateClaimed   BindingState = "claimed"
	StateStreaming BindingState = "streaming"
	StateOrphaned  BindingState = "orphaned"
	StateSuspended BindingState = "suspended"
)

type OutputState string

const (
	OutputIdle    OutputState = "idle"
	OutputSource  OutputState = "source"
	OutputBlack   OutputState = "black"
	OutputSilence OutputState = "silence"
)

type TerminationReason string

const (
	ReasonReleased        TerminationReason = "released"
	ReasonLeaseExpired    TerminationReason = "lease_expired"
	ReasonAdminDisconnect TerminationReason = "admin_disconnect"
	ReasonSlotRemoved     TerminationReason = "slot_removed"
	ReasonSlotChanged     TerminationReason = "slot_changed"
	ReasonTakenOver       TerminationReason = "taken_over"
)

type Format struct {
	Codec      string `json:"codec"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	FPS        int    `json:"fps,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
}

type Stream struct {
	ID      string    `json:"id"`
	Kind    Kind      `json:"kind"`
	Label   string    `json:"label"`
	Formats []Format  `json:"formats,omitempty"`
	USB     *USBOffer `json:"usb,omitempty"`
}

type USBOffer struct {
	Profile       string  `json:"profile"`
	Configuration uint8   `json:"configuration"`
	Interfaces    []uint8 `json:"interfaces"`
}

type Source struct {
	ID          string    `json:"id"`
	Owner       string    `json:"owner"`
	Agent       string    `json:"agent"`
	Label       string    `json:"label"`
	Streams     []Stream  `json:"streams"`
	ConnectedAt time.Time `json:"connected_at"`
}

type Slot struct {
	ID    string `json:"id"`
	Kind  Kind   `json:"kind"`
	Label string `json:"label"`
	// What a target host reads as this slot's interface string. Empty means
	// the kernel has no writable function_name for the slot's function, so
	// the host shows the kernel's own name whatever the label says. Compared
	// by value in SyncSlots, which is what makes a rename re-enumerate.
	HostName string `json:"host_name,omitempty"`
}

type Sink struct {
	Slot
	SlotNumber int          `json:"slot"`
	Demand     Demand       `json:"demand"`
	Output     OutputState  `json:"output"`
	Binding    *BindingView `json:"binding"`
	Latency    *SinkLatency `json:"latency,omitempty"`
}

// The browser-to-gadget skew the media worker measured over the last window.
// AvgMS and PeakMS are relative to BaseMS, the smallest skew yet seen, because
// the stamp is the browser's wall clock and the two clocks need not agree.
type SinkLatency struct {
	Frames int `json:"frames"`
	// Frames this sink threw away in the window because its queue was full.
	// A source that overruns the device by byte rate stays inside any
	// frames-per-second limit, so nothing else tells it to back off: every
	// frame it manages to push is acknowledged while older ones are quietly
	// discarded. Reporting the discards is what lets it lower its bitrate.
	Dropped   int       `json:"dropped"`
	AvgMS     float64   `json:"avg_ms"`
	PeakMS    float64   `json:"peak_ms"`
	BaseMS    float64   `json:"base_ms"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Demand struct {
	Streaming bool      `json:"streaming"`
	Width     int       `json:"width,omitempty"`
	Height    int       `json:"height,omitempty"`
	FPS       int       `json:"fps,omitempty"`
	Since     time.Time `json:"since,omitempty"`
}

type BindingView struct {
	SinkID      string       `json:"sink_id"`
	SourceID    string       `json:"source_id"`
	StreamID    string       `json:"stream_id"`
	Owner       string       `json:"owner"`
	SourceLabel string       `json:"source_label"`
	StreamLabel string       `json:"stream_label"`
	State       BindingState `json:"state"`
	StartedAt   time.Time    `json:"started_at"`
	ExpiresAt   time.Time    `json:"expires_at,omitempty"`
}

type Snapshot struct {
	Sinks    []Sink        `json:"sinks"`
	Sources  []Source      `json:"sources"`
	Bindings []BindingView `json:"bindings"`
}

type Event struct {
	Type     string            `json:"type"`
	Snapshot *Snapshot         `json:"snapshot,omitempty"`
	Sink     *Sink             `json:"sink,omitempty"`
	Sinks    []Sink            `json:"sinks,omitempty"`
	Source   *Source           `json:"source,omitempty"`
	Binding  *BindingView      `json:"binding,omitempty"`
	SinkID   string            `json:"sink_id,omitempty"`
	SourceID string            `json:"source_id,omitempty"`
	Reason   TerminationReason `json:"reason,omitempty"`
	Demand   *Demand           `json:"demand,omitempty"`
}

type Actor struct {
	Username string
	Admin    bool
}

type Hello struct {
	Label   string   `json:"label"`
	Streams []Stream `json:"streams"`
}

type ClaimResult struct {
	Binding BindingView `json:"binding"`
	Token   string      `json:"token"`
	Stream  Stream      `json:"-"`
}

var (
	ErrForbidden      = errors.New("forbidden")
	ErrNotFound       = errors.New("not found")
	ErrOccupied       = errors.New("slot occupied")
	ErrInvalidToken   = errors.New("invalid lease token")
	ErrInvalidMessage = errors.New("invalid message")

	idPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	cameraSlotID  = regexp.MustCompile(`^uvc\.cam([0-9])$`)
	micSlotID     = regexp.MustCompile(`^uac2\.mic([0-9])$`)
	speakerSlotID = regexp.MustCompile(`^uac2\.spk([0-9])$`)
	codecPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,23}$`)
)

type OccupiedError struct {
	SinkID      string
	Owner       string
	SourceLabel string
	Since       time.Time
}

func (e *OccupiedError) Error() string {
	return fmt.Sprintf("%s: %s by %s", ErrOccupied, e.SinkID, e.Owner)
}

func (e *OccupiedError) Unwrap() error { return ErrOccupied }

func validateSlots(slots []Slot) ([]Slot, error) {
	if len(slots) > MaxSlots {
		return nil, fmt.Errorf("slots: %d exceeds %d", len(slots), MaxSlots)
	}
	result := slices.Clone(slots)
	seen := make(map[string]struct{}, len(result))
	indices := map[Kind][]int{KindCamera: nil, KindMicrophone: nil, KindSpeaker: nil}
	for i := range result {
		result[i].Label = strings.TrimSpace(result[i].Label)
		if _, exists := seen[result[i].ID]; exists {
			return nil, fmt.Errorf("slots: duplicate id %q", result[i].ID)
		}
		if err := validateSlot(result[i]); err != nil {
			return nil, fmt.Errorf("slots[%d]: %w", i, err)
		}
		seen[result[i].ID] = struct{}{}
		indices[result[i].Kind] = append(indices[result[i].Kind], slotIndex(result[i].ID))
	}
	for kind, values := range indices {
		slices.Sort(values)
		for want, got := range values {
			if got != want {
				return nil, fmt.Errorf("slots: %s indices must be contiguous from zero", kind)
			}
		}
	}
	slices.SortFunc(result, func(a, b Slot) int { return strings.Compare(a.ID, b.ID) })
	return result, nil
}

func validateSlot(slot Slot) error {
	if err := validateLabel("label", slot.Label); err != nil {
		return err
	}
	if slot.HostName != "" {
		if err := validateLabel("host name", slot.HostName); err != nil {
			return err
		}
	}
	var pattern *regexp.Regexp
	switch slot.Kind {
	case KindCamera:
		pattern = cameraSlotID
	case KindMicrophone:
		pattern = micSlotID
	case KindSpeaker:
		pattern = speakerSlotID
	default:
		return fmt.Errorf("kind %q", slot.Kind)
	}
	if !pattern.MatchString(slot.ID) {
		return fmt.Errorf("id %q does not match kind %q", slot.ID, slot.Kind)
	}
	return nil
}

func slotIndex(id string) int {
	match := cameraSlotID.FindStringSubmatch(id)
	if match == nil {
		match = micSlotID.FindStringSubmatch(id)
	}
	if match == nil {
		match = speakerSlotID.FindStringSubmatch(id)
	}
	if len(match) != 2 {
		return 0
	}
	index, _ := strconv.Atoi(match[1])
	return index
}

func validateHello(hello Hello) (Hello, error) {
	hello.Label = strings.TrimSpace(hello.Label)
	if err := validateLabel("label", hello.Label); err != nil {
		return Hello{}, err
	}
	if len(hello.Streams) == 0 || len(hello.Streams) > MaxStreams {
		return Hello{}, fmt.Errorf("streams: want 1..%d", MaxStreams)
	}
	hello.Streams = cloneStreams(hello.Streams)
	seen := make(map[string]struct{}, len(hello.Streams))
	for i := range hello.Streams {
		hello.Streams[i].Label = strings.TrimSpace(hello.Streams[i].Label)
		stream := hello.Streams[i]
		if !idPattern.MatchString(stream.ID) {
			return Hello{}, fmt.Errorf("streams[%d]: invalid id", i)
		}
		if _, exists := seen[stream.ID]; exists {
			return Hello{}, fmt.Errorf("streams[%d]: duplicate id", i)
		}
		seen[stream.ID] = struct{}{}
		if stream.Kind != KindCamera && stream.Kind != KindMicrophone && stream.Kind != KindSpeaker && stream.Kind != KindUSBDevice {
			return Hello{}, fmt.Errorf("streams[%d]: invalid kind", i)
		}
		if err := validateLabel("stream label", stream.Label); err != nil {
			return Hello{}, fmt.Errorf("streams[%d]: %w", i, err)
		}
		if len(stream.Formats) > MaxFormats {
			return Hello{}, fmt.Errorf("streams[%d]: too many formats", i)
		}
		if stream.Kind == KindUSBDevice {
			if len(stream.Formats) != 0 || stream.USB == nil || !idPattern.MatchString(stream.USB.Profile) || stream.USB.Configuration == 0 || len(stream.USB.Interfaces) == 0 || len(stream.USB.Interfaces) > 16 {
				return Hello{}, fmt.Errorf("streams[%d]: invalid USB offer", i)
			}
			interfaces := make(map[uint8]bool, len(stream.USB.Interfaces))
			for _, number := range stream.USB.Interfaces {
				if interfaces[number] {
					return Hello{}, fmt.Errorf("streams[%d]: duplicate USB interface", i)
				}
				interfaces[number] = true
			}
			continue
		}
		if stream.USB != nil {
			return Hello{}, fmt.Errorf("streams[%d]: USB metadata on media stream", i)
		}
		for j, format := range stream.Formats {
			if err := format.validate(stream.Kind); err != nil {
				return Hello{}, fmt.Errorf("streams[%d].formats[%d]: %w", i, j, err)
			}
		}
	}
	return hello, nil
}

func (f Format) validate(kind Kind) error {
	if !codecPattern.MatchString(f.Codec) {
		return errors.New("invalid codec")
	}
	if kind == KindCamera {
		if f.Codec != "mjpeg" {
			return fmt.Errorf("unsupported camera codec %q", f.Codec)
		}
		allowed := map[[2]int]bool{{1280, 720}: true, {640, 480}: true, {320, 240}: true, {160, 120}: true}
		if !allowed[[2]int{f.Width, f.Height}] || (f.FPS != 15 && f.FPS != 30) {
			return errors.New("unsupported MJPEG size or rate")
		}
		if f.SampleRate != 0 || f.Channels != 0 {
			return errors.New("audio fields on camera")
		}
		return nil
	}
	if f.Codec != "pcm_s16le" || f.SampleRate != 48000 || f.Channels != 1 {
		return fmt.Errorf("%s requires mono pcm_s16le at 48000 Hz", kind)
	}
	if f.Width != 0 || f.Height != 0 || f.FPS != 0 {
		return fmt.Errorf("video fields on %s", kind)
	}
	return nil
}

func validateLabel(name, value string) error {
	if value == "" || len(value) > 80 || !utf8.ValidString(value) {
		return fmt.Errorf("%s must contain 1..80 UTF-8 bytes", name)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains control characters", name)
		}
	}
	return nil
}

func cloneStreams(streams []Stream) []Stream {
	result := make([]Stream, len(streams))
	for i := range streams {
		result[i] = streams[i]
		result[i].Formats = slices.Clone(streams[i].Formats)
		if streams[i].USB != nil {
			usb := *streams[i].USB
			usb.Interfaces = slices.Clone(streams[i].USB.Interfaces)
			result[i].USB = &usb
		}
	}
	return result
}
