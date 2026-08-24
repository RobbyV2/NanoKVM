package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	udcConfigured = "configured"
)

var (
	ErrNoGadget       = errors.New("usb gadget unavailable")
	ErrUnknownProfile = errors.New("unknown profile")
	ErrUnknownMode    = errors.New("unknown hid mode")
	ErrTransient      = errors.New("usb presentation has an active transient mode")
	ErrUDCLoaned      = errors.New("usb gadget: the udc is on loan")
)

// HIDQuiescer is *hid.Hid. The dependency points from hid into this package,
// so the bracket is injected rather than imported. The report writers are the
// same three the hid package releases every key through; they take the mutexes
// Lock takes, so they are only ever called outside the bracket.
type HIDQuiescer interface {
	Lock()
	Unlock()
	CloseNoLock()
	OpenNoLockWithRetry(timeout, delay time.Duration) error
	WriteKeyboardReport([]byte) error
	WriteRelativeMouseReport([]byte) error
	WriteAbsoluteMouseReport([]byte) error
}

// Also *hid.Hid, and optional so no fake HIDQuiescer has to grow a method it
// has nothing to say about. The layout decides which /dev/hidgN each role
// writes to and whether its reports carry a Report ID prefix, so it has to
// reach the writers before they reopen after a rebind.
type HIDRouter interface {
	SetHIDRoutes([]HIDRoute)
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

	wireMu  sync.Mutex
	hid     HIDQuiescer
	media   GadgetObserver
	rebound func(context.Context)

	mu sync.Mutex

	transient *Transient
	loan      string
	nextToken uint64

	statusMu   sync.Mutex
	lastError  *ApplyFailure
	powerCycle bool
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
	m.hid = h
	m.wireMu.Unlock()
	m.pushHIDRoutes(h)
}

func (m *Manager) pushHIDRoutes(h HIDQuiescer) {
	router, ok := h.(HIDRouter)
	if !ok {
		return
	}
	name, err := m.store.Active()
	if err != nil || name == "" {
		return
	}
	profile, err := m.store.LoadProfile(name)
	if err != nil {
		log.Debugf("hid routes: load active profile %s: %s", name, err)
		return
	}
	profile.Normalize()
	router.SetHIDRoutes(HIDRoutes(profile.Functions))
}

func (m *Manager) SetObserver(observer GadgetObserver) {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	m.media = observer
}

// Every gadget mutation below unbinds the UDC and binds it again, which
// destroys and recreates the gadget NIC. The bridge registers here so that br0
// regains the port it just lost; nothing else in this package knows a bridge
// can exist.
func (m *Manager) OnRebind(fn func(context.Context)) {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	m.rebound = fn
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
	snapshot.UDC = m.udcStatus()
	snapshot.LastError, snapshot.PendingPowerCycle = m.pendingStatus(snapshot.UDC)
	return snapshot, nil
}

// The gadget's UDC attribute names the controller it is bound to and says
// nothing more. state and speed sit next to that controller in sysfs, are world
// readable and need none of the privilege Ops exists to gate, so they are read
// here rather than widened into the interface.
func (m *Manager) udcStatus() UDCStatus {
	data, err := m.ops.ReadFile(udcAttr)
	status := UDCStatus{Name: strings.TrimSpace(string(data))}
	status.Bound = err == nil && status.Name != ""
	if status.Name == "" {
		udcs, listErr := m.ops.ListUDC()
		if listErr != nil {
			return status
		}
		status.Name = udcs[0]
	}
	status.State = readUDCAttr(status.Name, "state")
	status.Speed = readUDCAttr(status.Name, "current_speed")
	return status
}

func readUDCAttr(udc, attr string) string {
	data, err := os.ReadFile(filepath.Join(udcDir, udc, attr))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Nothing on this side can perform the power cycle, so the flag is cleared by
// the one thing that proves it happened: a controller reporting configured has
// been enumerated by a host since the reset that set it.
func (m *Manager) pendingStatus(udc UDCStatus) (*ApplyFailure, bool) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if udc.State == udcConfigured {
		m.powerCycle = false
	}
	return m.lastError, m.powerCycle
}

func (m *Manager) recordApply(profile string, err error) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if err == nil {
		m.lastError = nil
		return
	}
	m.lastError = &ApplyFailure{Profile: profile, Message: err.Error(), At: time.Now()}
}

func (m *Manager) requirePowerCycle() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.powerCycle = true
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
	m.recordApply(profile.Name, err)
	if err != nil {
		m.refreshObserver(context.Background())
		return err
	}
	m.notifyObserver(context.Background(), profile, plan)
	return nil
}

// What the plan would do to the linkage that is up now, and what the rollback a
// failed apply runs would then do. The target is resolved through the same
// prepareRecovery the transaction uses, so the preview names the profile the
// operator actually lands on rather than a second guess at it.
func (m *Manager) outcomes(plan Plan) (*Outcome, *Outcome) {
	if m.ready() != nil {
		return nil, nil
	}
	before, err := m.Snapshot()
	if err != nil {
		return nil, nil
	}
	applied := plan.Outcome(before)

	recovery, err := m.prepareRecovery()
	if err != nil {
		return &applied, nil
	}
	rollback := recovery.plan.Outcome(Snapshot{Linked: applied.Linked})
	return &applied, &rollback
}

// The active profile with its media functions preserved, which is the base
// every layout edit and every media edit has to start from so that changing
// one does not silently drop the other.
func (m *Manager) currentProfile() (Profile, error) {
	active, err := m.store.Active()
	if err != nil {
		return Profile{}, fmt.Errorf("read active profile: %w", err)
	}
	profile := standardProfile()
	if active != "" && active != ProfileHIDOnly && active != ProfileHybrid {
		loaded, loadErr := m.store.LoadProfile(active)
		if loadErr != nil {
			return Profile{}, fmt.Errorf("load active profile %s: %w", active, loadErr)
		}
		if loaded.Name != "" {
			profile = loaded
		}
	}
	if profile.BuiltIn {
		profile.Name = ProfileCurrent
		profile.BuiltIn = false
	}
	profile.Normalize()
	return profile, nil
}

func (m *Manager) SetHIDLayout(ctx context.Context, groups [][]HIDRole) error {
	profile, err := m.currentProfile()
	if err != nil {
		return err
	}
	if err := SetHIDLayout(&profile, groups); err != nil {
		return err
	}
	return m.ApplyProfile(ctx, profile)
}

func (m *Manager) SetMediaSlots(ctx context.Context, cameras, microphones []string) error {
	if len(cameras)+len(microphones) > 8 {
		return fmt.Errorf("media slots exceed 8")
	}
	profile, err := m.currentProfile()
	if err != nil {
		return err
	}
	functions := profile.Functions[:0]
	for _, function := range profile.Functions {
		if function.Kind != FunctionUVC && function.Kind != FunctionUAC2 {
			functions = append(functions, function)
		}
	}
	profile.Functions = functions
	named := m.caps.Functions[FunctionUVC].Attributes[UVCAttrFunctionName]
	for index, label := range cameras {
		profile.Functions = append(profile.Functions, defaultCamera(index, label, named))
	}
	named = m.caps.Functions[FunctionUAC2].Attributes[UAC2AttrFunctionName]
	for index, label := range microphones {
		profile.Functions = append(profile.Functions, defaultMicrophone(index, label, named))
	}
	profile.Normalize()
	return m.ApplyProfile(ctx, profile)
}

// The slot label doubles as the host-visible name, but only where the kernel
// can carry one: a device without the attribute must still be able to change
// how many slots it has.
func defaultCamera(index int, label string, named bool) Function {
	frames := []VideoFrame{
		{Width: 1280, Height: 720, Intervals: []uint32{333333, 666666}},
		{Width: 640, Height: 480, Intervals: []uint32{333333, 666666}},
		{Width: 320, Height: 240, Intervals: []uint32{333333, 666666}},
		{Width: 160, Height: 120, Intervals: []uint32{333333, 666666}},
	}
	video := &VideoFunction{
		FunctionName: label, Formats: []VideoFormat{{Codec: "mjpeg", Frames: frames}},
		StreamingMaxPacket: 768, StreamingInterval: 1,
	}
	if named {
		video.HostName = &label
	}
	return Function{Kind: FunctionUVC, Instance: fmt.Sprintf("cam%d", index), Video: video}
}

func defaultMicrophone(index int, label string, named bool) Function {
	audio := &AudioFunction{
		FunctionName: label, PChannelMask: 1, PSampleRate: 48000, PSampleSize: 2,
		CChannelMask: 0, CSampleRate: 48000, CSampleSize: 2, RequestNumber: 4,
	}
	if named {
		audio.HostName = &label
	}
	return Function{Kind: FunctionUAC2, Instance: fmt.Sprintf("mic%d", index), Audio: audio}
}

// mkdir functions/ffs.hybrid is what registers the ffs instance named "hybrid";
// mount -t functionfs hybrid returns ENODEV until it exists. The caller mounts
// and writes ep0 between this and StartFunctionFS, which links and binds.
func (m *Manager) CreateFunctionFS(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transient != nil {
		return ErrTransient
	}
	if err := m.loanHeld(); err != nil {
		return err
	}
	return m.ops.Mkdir(functionsDir + "/ffs.hybrid")
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
	if err := m.loanHeld(); err != nil {
		m.mu.Unlock()
		return nil, err
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
	if err := m.loanHeld(); err != nil {
		m.mu.Unlock()
		return err
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
		m.requirePowerCycle()
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
	// The operator's own layout, not the built-in one, capped at the two HID
	// interfaces a hybrid leaves room for.
	var hid []Function
	for _, function := range base.Functions {
		if function.Kind == FunctionHID && len(hid) < 2 {
			hid = append(hid, function)
		}
	}
	if len(hid) == 0 {
		hid = standardProfile().Functions[:2]
	}
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

	// Refuse before suspending, the way StartFunctionFS and RecoverFunctionFS
	// do. Suspend is undone only by the observer's Applied, which the refusals
	// below never reach, so suspending first strands the media pipeline on a
	// surrender that never happened.
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transient != nil {
		return "", ErrTransient
	}
	if err := m.loanHeld(); err != nil {
		return "", err
	}
	if observer := m.observer(); observer != nil {
		observer.Suspend()
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
	m.loan = fmt.Sprintf("a usb passthrough session (usb-proxy on udc %s)", udcs[0])
	return udcs[0], nil
}

// The loan ends when the borrower gives the controller back, whether or not the
// bind that follows succeeds. A failed bind leaves the UDC free, so keeping the
// loan would refuse every mutator for the rest of the boot, which is worse than
// the rebind it was there to prevent.
func (m *Manager) ReclaimUDC() error {
	if err := m.ready(); err != nil {
		return err
	}
	m.mu.Lock()
	m.loan = ""
	m.mu.Unlock()
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
		m.requirePowerCycle()
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
	if err := m.loanHeld(); err != nil {
		return err
	}
	return m.withHIDQuiesced(fn)
}

// udc->driver is one pointer and usb-proxy drives it from another process, so
// every mutator here would rebind the controller out from under a running
// passthrough session. The kernel is the record of who holds it: the borrower
// has it only while this gadget's UDC attribute is empty, so a loan standing
// against a bound gadget is stale and is dropped rather than believed.
// Callers hold m.mu.
func (m *Manager) loanHeld() error {
	if m.loan == "" {
		return nil
	}
	data, err := m.ops.ReadFile(udcAttr)
	if err == nil && strings.TrimSpace(string(data)) != "" {
		m.loan = ""
		return nil
	}
	return fmt.Errorf("%w to %s; stop the session before changing the usb gadget", ErrUDCLoaned, m.loan)
}

func (m *Manager) withHIDQuiesced(fn func() error) (err error) {
	h := m.quiescer()
	if h == nil {
		return fn()
	}

	h.Lock()
	h.CloseNoLock()
	defer func() {
		m.pushHIDRoutes(h)
		reopen := h.OpenNoLockWithRetry(hidQuiesceTimeout, hidQuiesceRetryDelay)
		h.Unlock()
		if reopen != nil {
			if err == nil {
				err = fmt.Errorf("reopen hid devices: %w", reopen)
			}
			return
		}
		releaseHID(h)
	}()
	return fn()
}

// All-keys-up in the three report shapes service/hid writes, sent once the
// devices are back and the mutexes are handed over. A rebind can leave a
// modifier held down on the host, and a release report is safe at any moment,
// so the bracket ends with one for every device it reopened. The write is best
// effort: a host that has not enumerated the gadget refuses it, and that is a
// statement about the host rather than about the gadget.
func releaseHID(h HIDQuiescer) {
	err := errors.Join(
		h.WriteKeyboardReport(make([]byte, 8)),
		h.WriteRelativeMouseReport(make([]byte, 4)),
		h.WriteAbsoluteMouseReport(make([]byte, 6)),
	)
	if err != nil {
		log.Warnf("release hid state after rebind: %s", err)
	}
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

func (m *Manager) notifyRebound(ctx context.Context) {
	m.wireMu.Lock()
	fn := m.rebound
	m.wireMu.Unlock()
	if fn != nil {
		fn(ctx)
	}
}

func (m *Manager) RefreshObserver(ctx context.Context) error {
	return m.refreshObserver(ctx)
}

func (m *Manager) refreshObserver(ctx context.Context) error {
	m.notifyRebound(ctx)
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
	m.notifyRebound(ctx)
	observer := m.observer()
	if observer == nil {
		return
	}
	if err := observer.Applied(ctx, profile, plan); err != nil {
		log.Errorf("media gadget reconcile: %s", err)
	}
}

func (m *Manager) Err() error {
	return m.err
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
