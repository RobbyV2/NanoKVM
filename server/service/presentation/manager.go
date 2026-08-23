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
	ErrTransient      = errors.New("usb presentation has an active transient mode")
)

// HIDQuiescer is *hid.Hid. The dependency points from hid into this package,
// so the bracket is injected rather than imported.
type HIDQuiescer interface {
	Lock()
	Unlock()
	CloseNoLock()
	OpenNoLockWithRetry(timeout, delay time.Duration) error
}

type GadgetObserver interface {
	Suspend()
	Applied(context.Context, Profile, Plan) error
}

type Manager struct {
	store *Store
	ops   Ops
	caps  CapabilityTable
	err   error

	wireMu sync.Mutex
	hid    HIDQuiescer
	media  GadgetObserver

	mu sync.Mutex

	transient *Transient
	nextToken uint64
}

type Transient struct {
	Token    uint64
	Profile  Profile
	recovery recoveryPlan
	udc      string
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

func (m *Manager) SetObserver(observer GadgetObserver) {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	m.media = observer
}

func (m *Manager) Snapshot() (Snapshot, error) {
	if err := m.ready(); err != nil {
		return Snapshot{}, err
	}

	active, err := m.store.Active()
	if err != nil {
		return Snapshot{}, fmt.Errorf("read active profile: %w", err)
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
	snapshot.Mode = modeOf(active)

	// A profile the capability table now rejects still describes a gadget that
	// is running, so the accounting failing is not the snapshot failing: the
	// budget is reported as far as it got and the caller reads the linkage.
	endpoints, err := AccountEndpoints(functions, m.caps)
	if err != nil {
		log.Debugf("endpoint accounting for %s: %s", active, err)
	}
	snapshot.Endpoints = endpoints
	snapshot.Headroom = endpoints.Headroom(m.caps)
	if fifos, err := SeatFIFOs(functions, m.caps); err == nil {
		snapshot.FIFOs = fifos
	}
	return snapshot, nil
}

// The bridge's step 13 asks for the gadget NIC by name. It is reported only
// when a network function is actually linked into configs/c.1, since a profile
// that merely mentions one leaves no usb0 for "ip link set usb0 master br0" to
// find. Snapshot probes the linkage rather than a /boot sentinel, so an NCM
// gadget stops being reported as an RNDIS one (H10).
func (m *Manager) NIC(context.Context) (string, error) {
	snapshot, err := m.Snapshot()
	if err != nil {
		return "", err
	}
	if !snapshot.HasNetwork() {
		return "", nil
	}
	return GadgetNIC, nil
}

// What the gadget is presenting to the attached host, empty when it presents no
// network at all. The bridge reports it and never sets it: the choice decides
// what the gadget looks like whether or not a bridge exists, so it belongs to
// the USB profile.
func (m *Manager) NetworkProtocol(context.Context) (string, error) {
	snapshot, err := m.Snapshot()
	if err != nil {
		return "", err
	}
	return string(snapshot.NetworkKind()), nil
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
	for _, function := range profile.Functions {
		if function.Kind == FunctionFFS {
			return ErrTransient
		}
	}

	plan, err := Compile(profile, m.caps)
	if err != nil {
		return err
	}
	observer := m.observer()
	if observer != nil {
		observer.Suspend()
	}
	err = m.withGadgetLock(func() error { return m.apply(ctx, profile, plan) })
	if err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.notifyObserver(context.Background(), profile, plan)
	return nil
}

func (m *Manager) SetMediaSlots(ctx context.Context, cameras, microphones []string) error {
	if len(cameras)+len(microphones) > 8 {
		return fmt.Errorf("media slots exceed 8")
	}
	active, err := m.store.Active()
	if err != nil {
		return fmt.Errorf("read active profile: %w", err)
	}
	profile := standardProfile()
	if active != "" && active != ProfileHIDOnly && active != ProfileHybrid {
		loaded, loadErr := m.store.LoadProfile(active)
		if loadErr != nil {
			return fmt.Errorf("load active profile %s: %w", active, loadErr)
		}
		if loaded.Name != "" {
			profile = loaded
		}
	}
	if profile.BuiltIn {
		profile.Name = ProfileCurrent
		profile.BuiltIn = false
	}
	functions := profile.Functions[:0]
	for _, function := range profile.Functions {
		if function.Kind != FunctionUVC && function.Kind != FunctionUAC2 {
			functions = append(functions, function)
		}
	}
	profile.Functions = functions
	for index, label := range cameras {
		profile.Functions = append(profile.Functions, defaultCamera(index, label))
	}
	for index, label := range microphones {
		profile.Functions = append(profile.Functions, defaultMicrophone(index, label))
	}
	profile.Normalize()
	return m.ApplyProfile(ctx, profile)
}

func defaultCamera(index int, label string) Function {
	frames := []VideoFrame{
		{Width: 1280, Height: 720, Intervals: []uint32{333333, 666666}},
		{Width: 640, Height: 480, Intervals: []uint32{333333, 666666}},
		{Width: 320, Height: 240, Intervals: []uint32{333333, 666666}},
		{Width: 160, Height: 120, Intervals: []uint32{333333, 666666}},
	}
	return Function{Kind: FunctionUVC, Instance: fmt.Sprintf("cam%d", index), Video: &VideoFunction{
		FunctionName: label, Formats: []VideoFormat{{Codec: "mjpeg", Frames: frames}},
		StreamingMaxPacket: 768, StreamingInterval: 1,
	}}
}

func defaultMicrophone(index int, label string) Function {
	return Function{Kind: FunctionUAC2, Instance: fmt.Sprintf("mic%d", index), Audio: &AudioFunction{
		FunctionName: label, PChannelMask: 1, PSampleRate: 48000, PSampleSize: 2,
		CChannelMask: 0, CSampleRate: 48000, CSampleSize: 2, RequestNumber: 4,
	}}
}

func (m *Manager) StartFunctionFS(ctx context.Context, function FunctionFS) (*Transient, error) {
	if err := m.ready(); err != nil {
		return nil, err
	}
	profile, err := m.hybridProfile(function)
	if err != nil {
		return nil, err
	}
	plan, err := Compile(profile, m.caps)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.transient != nil {
		m.mu.Unlock()
		return nil, ErrTransient
	}
	observer := m.observer()
	if observer != nil {
		observer.Suspend()
	}
	var recovery recoveryPlan
	var udc string
	err = m.withHIDQuiesced(func() error {
		var applyErr error
		recovery, udc, applyErr = m.applyPlan(ctx, profile, plan, false)
		return applyErr
	})
	if err != nil {
		m.mu.Unlock()
		m.refreshObserver(context.Background())
		return nil, err
	}
	m.nextToken++
	state := &Transient{Token: m.nextToken, Profile: profile, recovery: recovery, udc: udc}
	m.transient = state
	m.mu.Unlock()
	m.notifyObserver(context.Background(), profile, plan)
	return state, nil
}

func (m *Manager) StopFunctionFS(ctx context.Context, token uint64) error {
	if err := m.ready(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.transient == nil || m.transient.Token != token {
		m.mu.Unlock()
		return ErrTransient
	}
	state := m.transient
	if observer := m.observer(); observer != nil {
		observer.Suspend()
	}
	err := m.withHIDQuiesced(func() error { return m.restoreFunctionFS(ctx, state) })
	if err == nil {
		m.transient = nil
	}
	m.mu.Unlock()
	if err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.notifyObserver(context.Background(), state.recovery.profile, state.recovery.plan)
	return nil
}

func (m *Manager) RecoverFunctionFS(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.transient != nil {
		m.mu.Unlock()
		return ErrTransient
	}
	if observer := m.observer(); observer != nil {
		observer.Suspend()
	}
	var recovered Profile
	var recoveredPlan Plan
	err := m.withHIDQuiesced(func() (result error) {
		udcs, err := m.ops.ListUDC()
		if err != nil {
			return err
		}
		udc := udcs[0]
		defer func() {
			if result != nil {
				result = errors.Join(result, m.ensureBound(udc))
			}
		}()
		if err := m.ops.UnbindUDC(); err != nil {
			return err
		}
		if err := m.ops.Remove(configPrefix + "/ffs.hybrid"); err != nil {
			return err
		}
		name, err := m.store.Active()
		if err != nil {
			return err
		}
		if name == "" || name == ProfileHybrid {
			name = ProfileStandard
		}
		profile, err := m.store.LoadProfile(name)
		if err != nil || profile.Name == "" {
			profile = standardProfile()
		}
		plan, err := Compile(profile, m.caps)
		if err != nil {
			return err
		}
		recovered, recoveredPlan = profile, plan
		_, _, err = m.applyPlan(ctx, profile, plan, true)
		return err
	})
	m.mu.Unlock()
	if err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.notifyObserver(context.Background(), recovered, recoveredPlan)
	return nil
}

func (m *Manager) hybridProfile(function FunctionFS) (Profile, error) {
	name, err := m.store.Active()
	if err != nil {
		return Profile{}, err
	}
	base := standardProfile()
	if name != "" && name != ProfileHIDOnly && name != ProfileHybrid {
		if loaded, loadErr := m.store.LoadProfile(name); loadErr == nil && loaded.Name != "" {
			base = loaded
		}
	}
	hid := standardProfile().Functions[:2]
	base.Name, base.BuiltIn, base.OSDesc, base.Descriptors = ProfileHybrid, false, nil, nil
	base.Functions = append(append([]Function(nil), hid...), Function{Kind: FunctionFFS, Instance: "hybrid", FFS: &function})
	base.Normalize()
	return base, base.Validate()
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

// Raw-gadget runs on the same UDC and udc->driver is a single pointer, so
// passthrough can only start once this gadget is unbound. HID is closed and
// stays closed: f_hid deletes /dev/hidgN in hidg_unbind, so there is nothing to
// reopen until ReclaimUDC binds again, which is why this is not withGadgetLock.
func (m *Manager) SurrenderUDC() (string, error) {
	if err := m.ready(); err != nil {
		return "", err
	}

	observer := m.observer()
	if observer != nil {
		observer.Suspend()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transient != nil {
		return "", ErrTransient
	}

	udcs, err := m.ops.ListUDC()
	if err != nil {
		m.refreshObserver(context.Background())
		return "", fmt.Errorf("surrender udc: %w", err)
	}

	h := m.quiescer()
	if h != nil {
		h.Lock()
		h.CloseNoLock()
		h.Unlock()
	}
	// The unbind is what makes the closed devices unreopenable. If it fails the
	// gadget still owns the UDC and /dev/hidgN is still there, so leaving the
	// devices closed would cost a keyboard for a session that never started.
	if err := m.ops.UnbindUDC(); err != nil {
		reopen := reopenHID(h)
		m.refreshObserver(context.Background())
		return "", errors.Join(fmt.Errorf("surrender udc: %w", err), reopen)
	}
	return udcs[0], nil
}

func (m *Manager) ReclaimUDC() error {
	if err := m.ready(); err != nil {
		return err
	}
	if err := m.withGadgetLock(m.bind); err != nil {
		return err
	}
	m.refreshObserver(context.Background())
	return nil
}

func (m *Manager) UDCBound() (bool, error) {
	if err := m.ready(); err != nil {
		return false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.ops.ReadFile(udcAttr)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", udcAttr, err)
	}
	return strings.TrimSpace(string(data)) != "", nil
}

func (m *Manager) Rebind(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	if observer := m.observer(); observer != nil {
		observer.Suspend()
	}
	if err := m.withGadgetLock(func() error { return m.rebind(ctx) }); err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.refreshObserver(context.Background())
	return nil
}

// H5: ResetPHY polls for the controller to come back instead of sleeping a
// fixed second, and rebinds afterwards, since the dwc2 unbind drops the UDC.
func (m *Manager) ResetPHY(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	if observer := m.observer(); observer != nil {
		observer.Suspend()
	}
	err := m.withGadgetLock(func() error {
		if err := m.ops.ResetPHY(ctx); err != nil {
			return fmt.Errorf("reset usb phy: %w", err)
		}
		return m.bind()
	})
	if err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.refreshObserver(context.Background())
	return nil
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
	if m.transient != nil {
		return ErrTransient
	}
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

func reopenHID(h HIDQuiescer) error {
	if h == nil {
		return nil
	}

	h.Lock()
	defer h.Unlock()
	if err := h.OpenNoLockWithRetry(hidQuiesceTimeout, hidQuiesceRetryDelay); err != nil {
		return fmt.Errorf("reopen hid devices: %w", err)
	}
	return nil
}

func (m *Manager) quiescer() HIDQuiescer {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	return m.hid
}

func (m *Manager) observer() GadgetObserver {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	return m.media
}

func (m *Manager) RefreshObserver(ctx context.Context) error {
	return m.refreshObserver(ctx)
}

func (m *Manager) refreshObserver(ctx context.Context) error {
	observer := m.observer()
	if observer == nil {
		return nil
	}
	active, err := m.store.Active()
	if err != nil || active == "" {
		return err
	}
	profile, err := m.store.LoadProfile(active)
	if err != nil {
		return err
	}
	plan, err := Compile(profile, m.caps)
	if err != nil {
		return err
	}
	return observer.Applied(ctx, profile, plan)
}

func (m *Manager) notifyObserver(ctx context.Context, profile Profile, plan Plan) {
	observer := m.observer()
	if observer == nil {
		return
	}
	if err := observer.Applied(ctx, profile, plan); err != nil {
		log.Errorf("media gadget reconcile: %s", err)
	}
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
