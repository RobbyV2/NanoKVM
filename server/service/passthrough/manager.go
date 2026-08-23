package passthrough

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/functionfs"
	"NanoKVM-Server/service/presentation"

	log "github.com/sirupsen/logrus"
)

const (
	ModeHybrid = "hybrid"
	ModeExact  = "exact"

	portsPerHub      = 8
	locateTimeout    = 5 * time.Second
	locateInterval   = 100 * time.Millisecond
	stopTimeout      = 5 * time.Second
	livenessStrikes  = 5
	proxyReasonBytes = 4 << 10
	proxyReasonLimit = 200
)

var (
	udcClassDir      = "/sys/class/udc"
	proxyLog         = "/tmp/usb-proxy.log"
	livenessInterval = 2 * time.Second
)

var (
	ErrSessionActive = errors.New("passthrough: a session is already running")
	ErrNoSession     = errors.New("passthrough: no session is running")
	ErrRefusedBinary = errors.New("passthrough: refusing to run")
	ErrNotEnumerated = errors.New("passthrough: the imported device did not enumerate")
	ErrAmbiguous     = errors.New("passthrough: imported device location is ambiguous")
	ErrDescriptors   = errors.New("passthrough: cannot validate USB descriptors")
	ErrIsochronous   = errors.New("passthrough: isochronous devices are not supported")
	ErrNoUDCDriver   = errors.New("passthrough: cannot resolve the udc driver")
)

// The presentation manager owns configfs and the gadget lock, so the UDC is
// taken and given back through it rather than through a second writer.
type Gadget interface {
	SurrenderUDC() (string, error)
	ReclaimUDC() error
	UDCBound() (bool, error)
}

type FunctionFSGadget interface {
	CreateFunctionFS(context.Context) error
	StartFunctionFS(context.Context, presentation.FunctionFS) (*presentation.Transient, error)
	StopFunctionFS(context.Context, uint64) error
	RecoverFunctionFS(context.Context) error
}

type HybridRelay interface {
	Run(context.Context) error
	Close() error
}

type HybridFactory interface {
	Prepare(Local) (HybridRelay, presentation.FunctionFS, error)
	Cleanup() error
}

type VHCI interface {
	Attach(ctx context.Context, exporter string, busID string) (Attachment, error)
	Detach(port uint32) error
	Locate(ctx context.Context, attachment Attachment) (Local, error)
}

type Process interface {
	Pid() int
	Terminate() error
	Wait() error
}

type Spawner interface {
	Start(argv []string) (Process, error)
}

// Where the imported device landed on this device's own host stack. usb-proxy
// opens it through libusb, so what it needs is the local bus and address and
// not the exporter's.
type Local struct {
	Bus     uint32
	Address uint32
	Path    string
}

type Session struct {
	Mode      string
	Exporter  string
	BusID     string
	UDC       string
	Port      uint32
	Hub       Hub
	Device    Device
	Local     Local
	Pid       int
	StartedAt time.Time

	proc      Process
	relay     HybridRelay
	token     uint64
	logOffset int64
	stopping  atomic.Bool
	finish    sync.Once
	done      chan struct{}
	exited    chan struct{}
	watched   chan struct{}
	err       error
	remote    bool
}

func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) Err() error { return s.err }

type Manager struct {
	gadget   Gadget
	vhci     VHCI
	proxy    Spawner
	modules  ModuleLoader
	orphans  func() error
	hybrid   HybridFactory
	progress func(int) (proxyProgress, error)

	mu      sync.Mutex
	session *Session
	reason  string
}

var (
	managerOnce    sync.Once
	defaultManager *Manager
)

func GetManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = NewManager(presentation.GetManager(), kernelVHCI{}, execSpawner{}, Insmod{})
	})
	return defaultManager
}

func NewManager(gadget Gadget, vhci VHCI, proxy Spawner, modules ModuleLoader) *Manager {
	return &Manager{
		gadget:   gadget,
		vhci:     vhci,
		proxy:    proxy,
		modules:  modules,
		orphans:  stopProxyOrphans,
		hybrid:   functionFSFactory{},
		progress: readProxyProgress,
	}
}

// Start preserves the Exact behavior of the original API.
func (m *Manager) Start(ctx context.Context, exporter string, busID string) (*Session, error) {
	return m.StartMode(ctx, exporter, busID, ModeExact)
}

// A refusal is the operator's only account of what went wrong, so it is kept
// where the status can report it instead of only being returned once. A start
// refused because a session is already running says nothing about that session
// and must not overwrite what it reported.
func (m *Manager) StartMode(ctx context.Context, exporter string, busID string, mode string) (*Session, error) {
	session, err := m.startMode(ctx, exporter, busID, mode)
	if err != nil && !errors.Is(err, ErrSessionActive) {
		m.setReason(err.Error())
	}
	return session, err
}

func (m *Manager) setReason(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reason = reason
}

func (m *Manager) startMode(ctx context.Context, exporter string, busID string, mode string) (*Session, error) {
	if mode == "" {
		mode = ModeHybrid
	}
	if mode == ModeHybrid {
		return m.startHybrid(ctx, exporter, busID)
	}
	if mode != ModeExact {
		return nil, fmt.Errorf("%w: mode %q", ErrDescriptors, mode)
	}
	return m.startExact(ctx, exporter, busID)
}

func (m *Manager) startExact(ctx context.Context, exporter string, busID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		return nil, ErrSessionActive
	}
	m.reason = ""
	binary, err := installProxy()
	if err != nil {
		return nil, err
	}
	if err := m.modules.Load(sessionModules...); err != nil {
		return nil, err
	}

	attachment, err := m.vhci.Attach(ctx, exporter, busID)
	if err != nil {
		return nil, err
	}
	state := recoveryState{Port: attachment.Port, Mode: ModeExact}
	if err := saveRecoveryState(state); err != nil {
		return nil, errors.Join(err, m.vhci.Detach(attachment.Port))
	}

	local, err := m.vhci.Locate(ctx, attachment)
	if err != nil {
		return nil, errors.Join(err, m.detach(state))
	}
	if err := validateDescriptors(local.Path); err != nil {
		return nil, errors.Join(err, m.detach(state))
	}

	state.Reclaim = true
	if err := saveRecoveryState(state); err != nil {
		return nil, errors.Join(err, m.detach(state))
	}

	udc, err := m.gadget.SurrenderUDC()
	if err != nil {
		return nil, errors.Join(err, m.restore(state))
	}

	driver, err := udcDriver(udc)
	if err != nil {
		return nil, errors.Join(err, m.restore(state))
	}

	offset := proxyLogSize()
	proc, err := m.proxy.Start(proxyArgv(binary, udc, driver, local))
	if err != nil {
		return nil, errors.Join(err, m.restore(state))
	}

	session := &Session{
		Mode:      ModeExact,
		Exporter:  exporter,
		BusID:     busID,
		UDC:       udc,
		Port:      attachment.Port,
		Hub:       attachment.Hub,
		Device:    attachment.Device,
		Local:     local,
		Pid:       proc.Pid(),
		StartedAt: time.Now(),
		proc:      proc,
		logOffset: offset,
		done:      make(chan struct{}),
		exited:    make(chan struct{}),
		watched:   make(chan struct{}),
	}
	m.session = session

	go m.supervise(session)
	go m.watchProxy(session, livenessInterval)
	log.Debugf("passthrough: %s from %s on port %d, pid %d", busID, exporter, session.Port, session.Pid)
	return session, nil
}

func (m *Manager) startHybrid(ctx context.Context, exporter string, busID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		return nil, ErrSessionActive
	}
	m.reason = ""
	gadget, ok := m.gadget.(FunctionFSGadget)
	if !ok {
		return nil, fmt.Errorf("%w: FunctionFS presentation is unavailable", ErrDescriptors)
	}
	if err := m.modules.Load(ModuleUSBIPCore, ModuleVHCI); err != nil {
		return nil, err
	}
	attachment, err := m.vhci.Attach(ctx, exporter, busID)
	if err != nil {
		return nil, err
	}
	state := recoveryState{Port: attachment.Port, Mode: ModeHybrid}
	if err := saveRecoveryState(state); err != nil {
		return nil, errors.Join(err, m.vhci.Detach(attachment.Port))
	}
	local, err := m.vhci.Locate(ctx, attachment)
	if err != nil {
		return nil, errors.Join(err, m.detach(state))
	}
	// The ffs instance has to exist in configfs before Prepare mounts it.
	if err := gadget.CreateFunctionFS(ctx); err != nil {
		return nil, errors.Join(err, m.detach(state))
	}
	relay, function, err := m.hybrid.Prepare(local)
	if err != nil {
		return nil, errors.Join(err, m.detach(state))
	}
	transient, err := gadget.StartFunctionFS(ctx, function)
	if err != nil {
		return nil, errors.Join(err, relay.Close(), m.hybrid.Cleanup(), m.detach(state))
	}
	session := &Session{
		Mode: ModeHybrid, Exporter: exporter, BusID: busID, Port: attachment.Port,
		Hub: attachment.Hub, Device: attachment.Device, Local: local, StartedAt: time.Now(),
		relay: relay, token: transient.Token, done: make(chan struct{}),
	}
	m.session = session
	go m.superviseHybrid(session)
	log.Debugf("passthrough: Hybrid %s from %s on port %d", busID, exporter, session.Port)
	return session, nil
}

func (m *Manager) StartRemoteHybrid(ctx context.Context, label string, relay HybridRelay, function presentation.FunctionFS) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		return nil, ErrSessionActive
	}
	gadget, ok := m.gadget.(FunctionFSGadget)
	if !ok {
		return nil, fmt.Errorf("%w: FunctionFS presentation is unavailable", ErrDescriptors)
	}
	state := recoveryState{Mode: ModeHybrid, Source: "webusb"}
	if err := saveRecoveryState(state); err != nil {
		return nil, err
	}
	transient, err := gadget.StartFunctionFS(ctx, function)
	if err != nil {
		return nil, errors.Join(err, relay.Close(), m.hybrid.Cleanup(), clearRecoveryState())
	}
	session := &Session{
		Mode: ModeHybrid, Exporter: "browser", BusID: label, StartedAt: time.Now(),
		relay: relay, token: transient.Token, done: make(chan struct{}), remote: true,
	}
	m.session = session
	go m.superviseHybrid(session)
	return session, nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	session := m.session
	m.mu.Unlock()

	if session == nil {
		return ErrNoSession
	}
	if session.Mode == ModeHybrid {
		m.finishHybrid(session, nil)
		<-session.done
		return session.err
	}
	session.stopping.Store(true)
	if err := session.proc.Terminate(); err != nil {
		return err
	}

	<-session.done
	return session.err
}

func (m *Manager) StopSession(session *Session) error {
	m.mu.Lock()
	if m.session != session {
		m.mu.Unlock()
		return ErrNoSession
	}
	m.mu.Unlock()
	if session.Mode != ModeHybrid {
		return errors.New("passthrough: remote session is not Hybrid")
	}
	m.finishHybrid(session, nil)
	<-session.done
	return session.err
}

func (m *Manager) Close() error {
	m.mu.Lock()
	active := m.session != nil
	m.mu.Unlock()
	if !active {
		return nil
	}
	return m.Stop()
}

func (m *Manager) Recover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		return ErrSessionActive
	}
	state, err := loadRecoveryState()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.Join(err, m.orphans())
	}
	if err := m.orphans(); err != nil {
		return err
	}
	if state.Mode == ModeHybrid {
		gadget, ok := m.gadget.(FunctionFSGadget)
		if !ok {
			return fmt.Errorf("%w: FunctionFS presentation is unavailable", ErrDescriptors)
		}
		restoreErr := gadget.RecoverFunctionFS(context.Background())
		cleanupErr := m.hybrid.Cleanup()
		var detachErr error
		if state.Source != "webusb" {
			detachErr = m.vhci.Detach(state.Port)
		}
		err := errors.Join(restoreErr, cleanupErr, detachErr)
		if err == nil {
			err = clearRecoveryState()
		}
		return err
	}
	return m.restore(state)
}

func (m *Manager) superviseHybrid(session *Session) {
	err := session.relay.Run(context.Background())
	if !errors.Is(err, functionfs.ErrClosed) && !errors.Is(err, context.Canceled) {
		log.Warnf("passthrough: Hybrid relay exited: %s", err)
	}
	m.finishHybrid(session, err)
}

func (m *Manager) finishHybrid(session *Session, relayErr error) {
	session.finish.Do(func() {
		if errors.Is(relayErr, functionfs.ErrClosed) || errors.Is(relayErr, context.Canceled) {
			relayErr = nil
		}
		gadget := m.gadget.(FunctionFSGadget)
		restoreErr := gadget.StopFunctionFS(context.Background(), session.token)
		closeErr := session.relay.Close()
		cleanupErr := m.hybrid.Cleanup()
		var detachErr error
		if !session.remote {
			detachErr = m.vhci.Detach(session.Port)
		}
		cleanupResult := errors.Join(restoreErr, closeErr, cleanupErr, detachErr)
		if cleanupResult == nil {
			cleanupResult = clearRecoveryState()
		}
		session.err = errors.Join(relayErr, cleanupResult)
		m.mu.Lock()
		if relayErr != nil {
			m.reason = relayErr.Error()
		}
		if m.session == session {
			m.session = nil
		}
		m.mu.Unlock()
		close(session.done)
	})
}

// The watchdog. A crashed proxy takes the keyboard and the mouse with it, so
// the restore hangs off the proxy's exit and covers a stop, an unexpected exit
// and a crash with one path.
func (m *Manager) supervise(session *Session) {
	waitErr := session.proc.Wait()
	close(session.exited)
	<-session.watched

	m.mu.Lock()
	defer m.mu.Unlock()

	if !session.stopping.Load() && m.reason == "" {
		m.reason = proxyReason(session.logOffset)
	}
	session.err = m.restore(recoveryState{Port: session.Port, Reclaim: true})
	if m.session == session {
		m.session = nil
	}
	if waitErr != nil {
		log.Warnf("passthrough: usb-proxy exited: %s", waitErr)
	}
	if session.err != nil {
		log.Errorf("passthrough: restore after usb-proxy exited: %s", session.err)
	}
	close(session.done)
}

// The gadget comes back before the port goes away: a user without a keyboard is
// a worse state than a vhci port that outlives its proxy by a moment.
func (m *Manager) restore(state recoveryState) error {
	bound, boundErr := m.gadget.UDCBound()
	var reclaimErr error
	if boundErr == nil && state.Reclaim && !bound {
		reclaimErr = m.gadget.ReclaimUDC()
	}
	detachErr := m.vhci.Detach(state.Port)
	err := errors.Join(boundErr, reclaimErr, detachErr)
	if err == nil {
		err = clearRecoveryState()
	}
	return err
}

func (m *Manager) detach(state recoveryState) error {
	err := m.vhci.Detach(state.Port)
	if err == nil {
		err = clearRecoveryState()
	}
	return err
}

// Status carries the passthrough response plus why the running session is in
// trouble, or why the last one was refused or died. usb-proxy writes that only
// to its log, which nothing else reads.
type Status struct {
	proto.GetPassthroughRsp
	Reason string `json:"reason,omitempty"`
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.session
	if session == nil {
		return Status{Reason: m.reason}
	}
	return Status{Reason: m.reason, GetPassthroughRsp: proto.GetPassthroughRsp{
		Active:         true,
		Mode:           session.Mode,
		Exporter:       session.Exporter,
		UDC:            session.UDC,
		Port:           session.Port,
		Hub:            string(session.Hub),
		Bus:            session.Local.Bus,
		Address:        session.Local.Address,
		Pid:            session.Pid,
		HIDSurrendered: session.Mode == ModeExact,
		StartedAt:      session.StartedAt,
		Device: &proto.PassthroughDevice{
			BusID:     session.BusID,
			IDVendor:  hex4(session.Device.IDVendor),
			IDProduct: hex4(session.Device.IDProduct),
			Speed:     session.Device.Speed.String(),
			Class:     session.Device.DeviceClass,
		},
	}}
}

// proc.Wait only ever hears about a proxy that exits. One that keeps running
// without relaying anything still holds the UDC, and the operator loses the
// keyboard until the board is rebooted. usb-proxy cannot be asked how it is
// doing, but the kernel says how its threads wait: every wait raw-gadget and
// libusb make on its behalf is interruptible, so a thread parked in
// uninterruptible sleep is stuck inside the kernel rather than idle between
// packets. Only when that holds for every thread, with no CPU time accruing
// across several probes, is the proxy terminated: a proxy that is merely
// waiting on a quiet host looks exactly like a busy one from out here, and
// killing it would cost the operator the session it is for.
func (m *Manager) watchProxy(session *Session, interval time.Duration) {
	defer close(session.watched)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	last, err := m.progress(session.Pid)
	if err != nil {
		return
	}
	strikes := 0
	for {
		select {
		case <-session.exited:
			return
		case <-ticker.C:
		}
		current, err := m.progress(session.Pid)
		if err != nil {
			return
		}
		if current.blocked && current.cpu == last.cpu {
			strikes++
		} else {
			strikes = 0
		}
		last = current
		if strikes < livenessStrikes {
			continue
		}
		reason := fmt.Sprintf("usb-proxy %d made no progress for %s and every thread is blocked in the kernel", session.Pid, interval*livenessStrikes)
		log.Warnf("passthrough: %s", reason)
		m.setReason(reason)
		_ = session.proc.Terminate()
		return
	}
}

type proxyProgress struct {
	cpu     uint64
	blocked bool
}

func readProxyProgress(pid int) (proxyProgress, error) {
	root := filepath.Join(procRoot, strconv.Itoa(pid), "task")
	entries, err := os.ReadDir(root)
	if err != nil {
		return proxyProgress{}, err
	}
	if len(entries) == 0 {
		return proxyProgress{}, fmt.Errorf("usb-proxy %d has no threads", pid)
	}
	progress := proxyProgress{blocked: true}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), "stat"))
		if err != nil {
			continue
		}
		state, cpu, err := parseThreadStat(data)
		if err != nil {
			return proxyProgress{}, err
		}
		progress.cpu += cpu
		if state != 'D' {
			progress.blocked = false
		}
	}
	return progress, nil
}

// The comm field is parenthesised and may hold spaces and parentheses of its
// own, so the fields are counted from the last one it closes: state, then the
// ten fields before utime and stime.
func parseThreadStat(data []byte) (byte, uint64, error) {
	comm := bytes.LastIndexByte(data, ')')
	if comm < 0 {
		return 0, 0, errors.New("thread stat has no comm field")
	}
	fields := strings.Fields(string(data[comm+1:]))
	if len(fields) < 13 || len(fields[0]) != 1 {
		return 0, 0, fmt.Errorf("thread stat has %d fields", len(fields))
	}
	user, userErr := strconv.ParseUint(fields[11], 10, 64)
	system, systemErr := strconv.ParseUint(fields[12], 10, 64)
	if err := errors.Join(userErr, systemErr); err != nil {
		return 0, 0, err
	}
	return fields[0][0], user + system, nil
}

func proxyLogSize() int64 {
	info, err := os.Stat(proxyLog)
	if err != nil {
		return 0
	}
	return info.Size()
}

// usb-proxy prints why it will not relay a device and exits, and the message
// goes nowhere but its log. Only what this session appended is read, so an
// older session's failure is never reported as this one's.
func proxyReason(offset int64) string {
	file, err := os.Open(proxyLog)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.Size() <= offset {
		return ""
	}
	if info.Size()-offset > proxyReasonBytes {
		offset = info.Size() - proxyReasonBytes
	}
	data := make([]byte, info.Size()-offset)
	n, _ := file.ReadAt(data, offset)
	lines := strings.Split(string(data[:n]), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}
		if len(line) > proxyReasonLimit {
			line = line[:proxyReasonLimit]
		}
		return line
	}
	return ""
}

type functionFSFactory struct{}

func (functionFSFactory) Prepare(local Local) (HybridRelay, presentation.FunctionFS, error) {
	prepared, err := functionfs.Prepare(local.Path, local.Bus, local.Address, presentation.LoadCapabilities())
	if err != nil {
		return nil, presentation.FunctionFS{}, err
	}
	return prepared.Relay, prepared.Image.Function, nil
}

func (functionFSFactory) Cleanup() error { return functionfs.Cleanup() }

// usb-proxy matches on vendor and product id, which would just as happily bind
// a local device carrying the same pair, so the imported device is named by the
// bus and address vhci gave it. Every element here is a number from the import
// or a name read out of sysfs; no request string reaches an argv.
func proxyArgv(binary string, udc string, driver string, local Local) []string {
	return []string{
		binary,
		"--device", udc,
		"--driver", driver,
		"--device_bus", strconv.FormatUint(uint64(local.Bus), 10),
		"--device_addr", strconv.FormatUint(uint64(local.Address), 10),
		"--auto_remap_endpoints",
	}
}

// raw-gadget is opened against a UDC name and its driver name, and the default
// is the dummy_udc pair. The driver is read back rather than hardcoded so it
// cannot drift from the device the presentation manager just unbound.
func udcDriver(udc string) (string, error) {
	target, err := os.Readlink(filepath.Join(udcClassDir, udc, "device", "driver"))
	if err != nil {
		return "", fmt.Errorf("%w for %s: %w", ErrNoUDCDriver, udc, err)
	}

	driver := filepath.Base(target)
	if driver == "." || driver == string(filepath.Separator) {
		return "", fmt.Errorf("%w for %s: %q", ErrNoUDCDriver, udc, target)
	}
	return driver, nil
}

type kernelVHCI struct{}

func (kernelVHCI) Attach(ctx context.Context, exporter string, busID string) (Attachment, error) {
	return Attach(ctx, exporter, busID)
}

func (kernelVHCI) Detach(port uint32) error {
	return Detach(port)
}

func (kernelVHCI) Locate(ctx context.Context, attachment Attachment) (Local, error) {
	return locate(ctx, attachment)
}

// vhci hands the device to the local host stack, which enumerates it and gives
// it a bus and address of its own. Ports 0..7 are the hs root hub's 1..8 and
// 8..15 are the ss root hub's, so the port index is what ties an attachment
// back to a sysfs device.
func locate(ctx context.Context, attachment Attachment) (Local, error) {
	devpath := strconv.FormatUint(uint64(attachment.Port%portsPerHub+1), 10)
	deadline := time.Now().Add(locateTimeout)

	for {
		local, err := findLocal(devpath, attachment.Hub, attachment.Device)
		if err == nil {
			return local, nil
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return Local{}, err
		}

		select {
		case <-ctx.Done():
		case <-time.After(locateInterval):
		}
	}
}

func findLocal(devpath string, hub Hub, device Device) (Local, error) {
	matches, err := filepath.Glob(filepath.Join(vhciRoot, "usb*", "*-*"))
	if err != nil {
		return Local{}, fmt.Errorf("scan %s: %w", vhciRoot, err)
	}

	var found []Local
	for _, dir := range matches {
		if attribute(dir, "devpath") != devpath {
			continue
		}
		rootSpeed, err := strconv.ParseFloat(attribute(filepath.Dir(dir), "speed"), 64)
		if err != nil || (rootSpeed >= 5000) != (hub == HubSuper) {
			continue
		}
		if attribute(dir, "idVendor") != hex4(device.IDVendor) || attribute(dir, "idProduct") != hex4(device.IDProduct) {
			continue
		}

		bus, busErr := strconv.ParseUint(attribute(dir, "busnum"), 10, 32)
		address, addressErr := strconv.ParseUint(attribute(dir, "devnum"), 10, 32)
		if err := errors.Join(busErr, addressErr); err != nil {
			return Local{}, fmt.Errorf("read bus and address of %s: %w", dir, err)
		}
		found = append(found, Local{Bus: uint32(bus), Address: uint32(address), Path: dir})
	}
	if len(found) > 1 {
		return Local{}, fmt.Errorf("%w: %s on the %s root port %s", ErrAmbiguous, device.BusID, hub, devpath)
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return Local{}, fmt.Errorf("%w: %s on root port %s", ErrNotEnumerated, device.BusID, devpath)
}

func attribute(dir string, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func validateDescriptors(devicePath string) error {
	raw, err := os.ReadFile(filepath.Join(devicePath, "descriptors"))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDescriptors, err)
	}
	for offset := 0; offset < len(raw); {
		if len(raw)-offset < 2 {
			return fmt.Errorf("%w: truncated descriptor at %d", ErrDescriptors, offset)
		}
		length := int(raw[offset])
		if length < 2 || offset+length > len(raw) {
			return fmt.Errorf("%w: invalid length %d at %d", ErrDescriptors, length, offset)
		}
		if raw[offset+1] == 5 {
			if length < 4 {
				return fmt.Errorf("%w: truncated endpoint at %d", ErrDescriptors, offset)
			}
			if raw[offset+3]&0x03 == 0x01 {
				return ErrIsochronous
			}
		}
		offset += length
	}
	return nil
}

func hex4(value uint16) string {
	return fmt.Sprintf("%04x", value)
}

type execSpawner struct{}

// The single spawn point, taking an already split argv the way bridge's
// Commander does, so a string that somehow reached here still cannot name a
// binary the design did not intend.
func (execSpawner) Start(argv []string) (Process, error) {
	if len(argv) == 0 || argv[0] != proxyBinary {
		return nil, fmt.Errorf("%w: %v", ErrRefusedBinary, argv)
	}

	file, err := os.OpenFile(proxyLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", proxyLog, err)
	}

	// Not CommandContext: the proxy outlives the request that started it, and
	// its lifetime is the session the watchdog supervises.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = file, file

	if err := cmd.Start(); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("start %s: %w", argv[0], err)
	}
	return &execProcess{cmd: cmd, log: file}, nil
}

type execProcess struct {
	cmd    *exec.Cmd
	log    *os.File
	exited atomic.Bool
}

func (p *execProcess) Pid() int {
	return p.cmd.Process.Pid
}

func (p *execProcess) Terminate() error {
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("terminate usb-proxy: %w", err)
	}

	time.AfterFunc(stopTimeout, func() {
		if p.exited.Load() {
			return
		}
		_ = p.cmd.Process.Kill()
	})
	return nil
}

func (p *execProcess) Wait() error {
	err := p.cmd.Wait()
	p.exited.Store(true)
	_ = p.log.Close()
	return err
}
