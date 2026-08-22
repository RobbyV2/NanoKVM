package presentation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ModeNormal  = "normal"
	ModeHIDOnly = ProfileHIDOnly

	hidQuiesceTimeout    = 2 * time.Second
	hidQuiesceRetryDelay = 100 * time.Millisecond
)

var (
	ErrNoGadget       = errors.New("usb gadget unavailable")
	ErrUnknownProfile = errors.New("unknown profile")
	ErrUnknownMode    = errors.New("unknown hid mode")
)

// HIDQuiescer is *hid.Hid. The dependency points from hid into this package,
// so the bracket is injected rather than imported.
type HIDQuiescer interface {
	Lock()
	Unlock()
	CloseNoLock()
	OpenNoLockWithRetry(timeout, delay time.Duration) error
}

type Manager struct {
	store *Store
	ops   Ops
	caps  CapabilityTable
	err   error

	wireMu sync.Mutex
	hid    HIDQuiescer

	mu sync.Mutex
}

var (
	managerOnce    sync.Once
	defaultManager *Manager
)

func GetManager() *Manager {
	managerOnce.Do(func() {
		store := NewStore()
		defaultManager = NewManager(store, nil, LoadCapabilities())

		ops, err := NewConfigFSOps(GadgetRoot)
		if err != nil {
			defaultManager.err = fmt.Errorf("%w: %w", ErrNoGadget, err)
			log.Errorf("presentation manager unavailable: %s", defaultManager.err)
			return
		}
		defaultManager.ops = ops

		if err := store.WriteBuiltins(); err != nil {
			log.Errorf("write built-in profiles: %s", err)
		}
		if err := Migrate(store, ops); err != nil {
			log.Errorf("migrate usb presentation: %s", err)
		}
	})
	return defaultManager
}

func NewManager(store *Store, ops Ops, caps CapabilityTable) *Manager {
	return &Manager{store: store, ops: ops, caps: caps}
}

func (m *Manager) SetHID(h HIDQuiescer) {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	m.hid = h
}

func (m *Manager) Snapshot() (Snapshot, error) {
	if err := m.ready(); err != nil {
		return Snapshot{}, err
	}

	active, err := m.store.Active()
	if err != nil {
		return Snapshot{}, fmt.Errorf("read active profile: %w", err)
	}
	lastKnownGood, err := m.store.LastKnownGood()
	if err != nil {
		return Snapshot{}, fmt.Errorf("read last known good profile: %w", err)
	}

	var functions []Function
	if active != "" {
		profile, err := m.store.LoadProfile(active)
		if err != nil {
			return Snapshot{}, fmt.Errorf("load active profile %s: %w", active, err)
		}
		functions = profile.Functions
	}

	snapshot := readSnapshot(m.ops, functions)
	snapshot.Active = active
	snapshot.LastKnownGood = lastKnownGood
	snapshot.Mode = modeOf(active)
	snapshot.Capabilities = m.caps.Source
	return snapshot, nil
}

func (m *Manager) Apply(ctx context.Context, name string) error {
	if err := m.ready(); err != nil {
		return err
	}

	profile, err := m.store.LoadProfile(name)
	if err != nil {
		return fmt.Errorf("load profile %s: %w", name, err)
	}
	if profile.Name == "" {
		return fmt.Errorf("%w: %s", ErrUnknownProfile, name)
	}
	return m.ApplyProfile(ctx, profile)
}

func (m *Manager) ApplyProfile(ctx context.Context, profile Profile) error {
	if err := m.ready(); err != nil {
		return err
	}

	plan, err := Compile(profile, m.caps)
	if err != nil {
		return err
	}
	if err := m.store.SaveProfile(profile); err != nil {
		return fmt.Errorf("save profile %s: %w", profile.Name, err)
	}
	return m.withGadgetLock(func() error { return m.apply(ctx, profile, plan) })
}

// The LUN is runtime state rather than profile state: a reapply rewrites
// removable and inquiry_string but leaves ro, cdrom and file as they were set
// here (H7).
type LUN struct {
	File  string `json:"file"`
	CDROM bool   `json:"cdrom"`
}

func (l LUN) inquiry() string {
	if l.CDROM {
		return InquiryStringCDROM
	}
	return InquiryString
}

func (l LUN) backingFile() string {
	if l.File == "" {
		return DefaultDiskFile
	}
	return l.File
}

func (m *Manager) LUN() (LUN, error) {
	if err := m.ready(); err != nil {
		return LUN{}, err
	}
	return m.readLUN()
}

func (m *Manager) SetLUN(ctx context.Context, lun LUN) error {
	if err := m.ready(); err != nil {
		return err
	}
	return m.withGadgetLock(func() error { return m.setLUN(ctx, lun) })
}

func (m *Manager) Rebind(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	return m.withGadgetLock(func() error { return m.rebind(ctx) })
}

// H5: ResetPHY polls for the controller to come back instead of sleeping a
// fixed second, and rebinds afterwards, since the dwc2 unbind drops the UDC.
func (m *Manager) ResetPHY(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	return m.withGadgetLock(func() error {
		if err := m.ops.ResetPHY(ctx); err != nil {
			return fmt.Errorf("reset usb phy: %w", err)
		}
		return m.bind()
	})
}

// D6: mode resolves in three tiers. The active profile is the truth; an exact
// bcdDevice match covers a gadget configured before the marker was written
// deliberately; any 0x05xx is the gadget core default on some 5.x kernel, which
// is what normal mode used to be identified by.
func (m *Manager) Mode() (string, error) {
	active, err := m.store.Active()
	if err != nil {
		return "", fmt.Errorf("read active profile: %w", err)
	}
	if active != "" {
		return modeOf(active), nil
	}
	if err := m.ready(); err != nil {
		return "", err
	}

	data, err := m.ops.ReadFile(attrBCDDevice)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", attrBCDDevice, err)
	}
	return modeFromBCDDevice(strings.TrimSpace(string(data)))
}

// D3: mode is manager state, not the filename of the init script the OTA keeps
// overwriting. Normal mode returns to the migrated profile so the user's
// network and disk functions survive the round trip.
func (m *Manager) SetMode(ctx context.Context, mode string) error {
	switch mode {
	case ModeHIDOnly:
		return m.Apply(ctx, ProfileHIDOnly)
	case ModeNormal:
		if profile, err := m.store.LoadProfile(ProfileCurrent); err == nil && profile.Name != "" {
			return m.ApplyProfile(ctx, profile)
		}
		return m.Apply(ctx, ProfileStandard)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownMode, mode)
	}
}

// R1.3: the HID mutex is the only mutual exclusion the four gadget mutators
// have today, so the gadget lock wraps it here and in no other place.
func (m *Manager) withGadgetLock(fn func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.withHIDQuiesced(fn)
}

func (m *Manager) withHIDQuiesced(fn func() error) (err error) {
	h := m.quiescer()
	if h == nil {
		return fn()
	}

	h.Lock()
	h.CloseNoLock()
	defer func() {
		reopen := h.OpenNoLockWithRetry(hidQuiesceTimeout, hidQuiesceRetryDelay)
		h.Unlock()
		if reopen != nil && err == nil {
			err = fmt.Errorf("reopen hid devices: %w", reopen)
		}
	}()
	return fn()
}

func (m *Manager) quiescer() HIDQuiescer {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	return m.hid
}

func (m *Manager) ready() error {
	if m.err != nil {
		return m.err
	}
	if m.ops == nil {
		return ErrNoGadget
	}
	return nil
}

var bcdDeviceModes = map[string]string{
	BCDDeviceNormal:  ModeNormal,
	BCDDeviceHIDOnly: ModeHIDOnly,
}

func modeFromBCDDevice(marker string) (string, error) {
	if mode, ok := bcdDeviceModes[marker]; ok {
		return mode, nil
	}
	if value, err := strconv.ParseUint(strings.TrimPrefix(marker, "0x"), 16, 16); err == nil && value>>8 == 0x05 {
		return ModeNormal, nil
	}
	return "", fmt.Errorf("%w: bcdDevice %q", ErrUnknownMode, marker)
}

func modeOf(profile string) string {
	if profile == ProfileHIDOnly {
		return ModeHIDOnly
	}
	return ModeNormal
}
