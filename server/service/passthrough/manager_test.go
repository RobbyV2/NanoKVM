package passthrough

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/service/functionfs"
	"NanoKVM-Server/service/presentation"
)

const testUDC = "4340000.usb"

type fakeGadget struct {
	mu            sync.Mutex
	bound         bool
	surrender     int
	reclaim       int
	fail          error
	reclaimed     chan struct{}
	beforeReclaim func()
}

// The lifecycle steps the kernel orders, in the place each one really happens:
// the mkdir registers the ffs instance (presentation.Manager.CreateFunctionFS),
// the mount and the ep0 writes name it (functionfs.openFunctionFS), and the
// link and the bind come last, in the plan StartFunctionFS applies, whose own
// mkdir is the idempotent repeat of the first.
type lifecycleTrace struct {
	mu     sync.Mutex
	events []string
}

func (t *lifecycleTrace) record(event string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *lifecycleTrace) taken() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.events)
}

type fakeHybridGadget struct {
	*fakeGadget
	muHybrid  sync.Mutex
	trace     *lifecycleTrace
	created   int
	started   int
	stopped   int
	recovered int
}

func (g *fakeHybridGadget) CreateFunctionFS(context.Context) error {
	g.muHybrid.Lock()
	defer g.muHybrid.Unlock()
	g.created++
	g.trace.record("configfs mkdir functions/ffs.hybrid")
	return nil
}

func (g *fakeHybridGadget) StartFunctionFS(_ context.Context, _ presentation.FunctionFS) (*presentation.Transient, error) {
	g.muHybrid.Lock()
	defer g.muHybrid.Unlock()
	g.started++
	g.trace.record("configfs mkdir functions/ffs.hybrid")
	g.trace.record("configfs link configs/c.1/ffs.hybrid")
	g.trace.record("configfs bind udc")
	return &presentation.Transient{Token: 7}, nil
}

func (g *fakeHybridGadget) StopFunctionFS(_ context.Context, token uint64) error {
	if token != 7 {
		return errors.New("wrong transient token")
	}
	g.muHybrid.Lock()
	defer g.muHybrid.Unlock()
	g.stopped++
	return nil
}

func (g *fakeHybridGadget) RecoverFunctionFS(context.Context) error {
	g.muHybrid.Lock()
	defer g.muHybrid.Unlock()
	g.recovered++
	return nil
}

type fakeHybridRelay struct {
	closed chan struct{}
	fail   error
	once   sync.Once
}

func newFakeHybridRelay() *fakeHybridRelay { return &fakeHybridRelay{closed: make(chan struct{})} }
func (r *fakeHybridRelay) Run(context.Context) error {
	if r.fail != nil {
		return r.fail
	}
	<-r.closed
	return functionfs.ErrClosed
}
func (r *fakeHybridRelay) Close() error {
	r.once.Do(func() { close(r.closed) })
	return nil
}

type fakeHybridFactory struct {
	relay    *fakeHybridRelay
	trace    *lifecycleTrace
	prepared int
	cleaned  int
}

func (f *fakeHybridFactory) Prepare(Local) (HybridRelay, presentation.FunctionFS, error) {
	f.prepared++
	f.trace.record("functionfs mount hybrid")
	f.trace.record("functionfs write ep0 descriptors")
	return f.relay, presentation.FunctionFS{Interfaces: 1, Endpoints: []presentation.FunctionFSEndpoint{
		{SourceAddress: 0x81, Address: 0x81, Transfer: presentation.EndpointInterrupt, MaxPacket: 8, Interval: 10},
	}}, nil
}

func (f *fakeHybridFactory) Cleanup() error {
	f.cleaned++
	return nil
}

func newFakeGadget() *fakeGadget {
	return &fakeGadget{bound: true, reclaimed: make(chan struct{}, 4)}
}

func TestRemoteHybridSkipsVHCI(t *testing.T) {
	recoveryStatePath = filepath.Join(t.TempDir(), "session.json")
	t.Cleanup(func() { recoveryStatePath = "/etc/kvm/passthrough/session.json" })
	gadget := &fakeHybridGadget{fakeGadget: newFakeGadget()}
	vhci := &fakeVHCI{}
	manager := NewManager(gadget, vhci, &fakeSpawner{}, &fakeModules{})
	factory := &fakeHybridFactory{relay: newFakeHybridRelay()}
	manager.hybrid = factory
	relay := newFakeHybridRelay()
	session, err := manager.StartRemoteHybrid(context.Background(), "Debug adapter", relay, presentation.FunctionFS{
		Interfaces: 1,
		Endpoints: []presentation.FunctionFSEndpoint{{
			SourceAddress: 0x81, Address: 0x81, Transfer: presentation.EndpointBulk, MaxPacket: 64,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.StopSession(session); err != nil {
		t.Fatal(err)
	}
	attached, detached := vhci.calls()
	if len(attached) != 0 || len(detached) != 0 {
		t.Fatalf("remote Hybrid touched VHCI: attach=%v detach=%v", attached, detached)
	}
	if gadget.started != 1 || gadget.stopped != 1 || factory.cleaned != 1 {
		t.Fatalf("lifecycle start=%d stop=%d cleanup=%d", gadget.started, gadget.stopped, factory.cleaned)
	}
}

func (g *fakeGadget) SurrenderUDC() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.fail != nil {
		return "", g.fail
	}
	g.surrender++
	g.bound = false
	return testUDC, nil
}

func (g *fakeGadget) ReclaimUDC() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.beforeReclaim != nil {
		g.beforeReclaim()
	}
	g.reclaim++
	g.bound = true
	g.reclaimed <- struct{}{}
	return nil
}

func (g *fakeGadget) UDCBound() (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.bound, nil
}

func (g *fakeGadget) state() (bool, int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.bound, g.surrender, g.reclaim
}

type fakeVHCI struct {
	mu         sync.Mutex
	device     Device
	port       uint32
	attached   []string
	detached   []uint32
	fail       error
	allowedIso bool
}

func (v *fakeVHCI) isoAllowed() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.allowedIso
}

func (v *fakeVHCI) Attach(_ context.Context, _ string, busID string, allowIso bool) (Attachment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.allowedIso = allowIso

	if v.fail != nil {
		return Attachment{}, v.fail
	}
	v.attached = append(v.attached, busID)
	return Attachment{Port: v.port, Hub: HubHigh, BusID: busID, Device: v.device}, nil
}

func (v *fakeVHCI) Detach(port uint32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.detached = append(v.detached, port)
	return nil
}

// The sysfs scan is the real one: a fake here would leave the mapping from a
// vhci port to a local bus and address untested, and that mapping is what
// usb-proxy is pointed at.
func (v *fakeVHCI) Locate(ctx context.Context, attachment Attachment) (Local, error) {
	return locate(ctx, attachment)
}

func (v *fakeVHCI) calls() ([]string, []uint32) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.attached), slices.Clone(v.detached)
}

type fakeProcess struct {
	exit      chan struct{}
	terminate chan struct{}
	once      sync.Once
}

func newFakeProcess() *fakeProcess {
	return &fakeProcess{exit: make(chan struct{}), terminate: make(chan struct{}, 1)}
}

func (p *fakeProcess) Pid() int { return 4242 }

func (p *fakeProcess) Terminate() error {
	p.terminate <- struct{}{}
	p.kill()
	return nil
}

func (p *fakeProcess) kill() {
	p.once.Do(func() { close(p.exit) })
}

func (p *fakeProcess) Wait() error {
	<-p.exit
	return errors.New("exit status 1")
}

type fakeSpawner struct {
	mu      sync.Mutex
	argv    [][]string
	process *fakeProcess
	fail    error
}

func (s *fakeSpawner) Start(argv []string) (Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fail != nil {
		return nil, s.fail
	}
	s.argv = append(s.argv, slices.Clone(argv))
	s.process = newFakeProcess()
	return s.process, nil
}

func (s *fakeSpawner) last() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.argv) == 0 {
		return nil
	}
	return s.argv[len(s.argv)-1]
}

type fakeModules struct {
	mu     sync.Mutex
	loaded []Module
	fail   error
}

func (f *fakeModules) Load(modules ...Module) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.fail != nil {
		return f.fail
	}
	f.loaded = append(f.loaded, modules...)
	return nil
}

func newTestManager(t *testing.T) (*Manager, *fakeGadget, *fakeVHCI, *fakeSpawner) {
	t.Helper()

	device := Device{
		BusID:     "1-1",
		BusNum:    1,
		DevNum:    4,
		Speed:     SpeedHigh,
		IDVendor:  0x046d,
		IDProduct: 0xc31c,
	}
	fakeSysfs(t, 5, device, 3, 7)
	fakeProxySeed(t, proxyELF)

	gadget, vhci, spawner := newFakeGadget(), &fakeVHCI{device: device, port: 5}, &fakeSpawner{}
	return NewManager(gadget, vhci, spawner, &fakeModules{}), gadget, vhci, spawner
}

// A vhci tree holding the device the exporter described, plus the udc class
// entry whose driver symlink names the raw-gadget driver.
func fakeSysfs(t *testing.T, port uint32, device Device, bus int, address int) {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, "usb3", "3-6")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "speed"), []byte("480\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	attrs := map[string]string{
		"devpath":   "6",
		"idVendor":  hex4(device.IDVendor),
		"idProduct": hex4(device.IDProduct),
		"busnum":    "3\n",
		"devnum":    "7\n",
	}
	if port%portsPerHub+1 != 6 || bus != 3 || address != 7 {
		t.Fatalf("fakeSysfs is pinned to port 5, bus 3, address 7")
	}
	for name, value := range attrs {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "descriptors"), []byte{
		18, 1, 0, 2, 0, 0, 0, 64, 0x6d, 0x04, 0x1c, 0xc3, 0, 1, 1, 2, 3, 1,
		9, 2, 25, 0, 1, 1, 0, 0x80, 50,
		9, 4, 0, 0, 1, 0, 0, 0, 0,
		7, 5, 0x81, 0x03, 8, 0, 10,
	}, 0o644); err != nil {
		t.Fatal(err)
	}

	udcDir := filepath.Join(root, "udc", testUDC, "device")
	if err := os.MkdirAll(udcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../bus/platform/drivers/dwc2", filepath.Join(udcDir, "driver")); err != nil {
		t.Fatal(err)
	}

	previousVHCI, previousUDC := vhciRoot, udcClassDir
	vhciRoot, udcClassDir = root, filepath.Join(root, "udc")
	t.Cleanup(func() { vhciRoot, udcClassDir = previousVHCI, previousUDC })
}

func addLocalDevice(t *testing.T, root string, bus string, rootSpeed string) {
	t.Helper()

	parent := filepath.Join(root, "usb"+bus)
	dir := filepath.Join(parent, bus+"-6")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	attrs := map[string]string{
		filepath.Join(parent, "speed"):  rootSpeed + "\n",
		filepath.Join(dir, "devpath"):   "6\n",
		filepath.Join(dir, "idVendor"):  "046d\n",
		filepath.Join(dir, "idProduct"): "c31c\n",
		filepath.Join(dir, "busnum"):    bus + "\n",
		filepath.Join(dir, "devnum"):    "7\n",
	}
	for path, value := range attrs {
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindLocalRefusesAnAmbiguousHubLocation(t *testing.T) {
	root := t.TempDir()
	previous := vhciRoot
	vhciRoot = root
	t.Cleanup(func() { vhciRoot = previous })

	addLocalDevice(t, root, "3", "480")
	addLocalDevice(t, root, "4", "480")
	device := Device{BusID: "1-1", IDVendor: 0x046d, IDProduct: 0xc31c}
	if _, err := findLocal("6", HubHigh, device); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("findLocal = %v, want %v", err, ErrAmbiguous)
	}
}

func TestFindLocalSelectsTheAttachmentHub(t *testing.T) {
	root := t.TempDir()
	previous := vhciRoot
	vhciRoot = root
	t.Cleanup(func() { vhciRoot = previous })

	addLocalDevice(t, root, "3", "480")
	addLocalDevice(t, root, "4", "10000")
	device := Device{BusID: "1-1", IDVendor: 0x046d, IDProduct: 0xc31c}
	local, err := findLocal("6", HubSuper, device)
	if err != nil {
		t.Fatal(err)
	}
	if local.Bus != 4 || local.Address != 7 {
		t.Fatalf("local = %+v, want bus 4 address 7", local)
	}
}

const proxyELF = "\x7fELFusb-proxy"

// The binary is never on disk in a test, so every Start here goes through the
// extractor the device relies on.
func fakeProxySeed(t *testing.T, content string) string {
	t.Helper()

	root := t.TempDir()
	seed := filepath.Join(root, "seed", "usb-proxy.gz")
	if err := os.MkdirAll(filepath.Dir(seed), 0o755); err != nil {
		t.Fatal(err)
	}

	file, err := os.Create(seed)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := io.WriteString(writer, content); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(writer.Close(), file.Close()); err != nil {
		t.Fatal(err)
	}

	previousBinary, previousSeed, previousState := proxyBinary, proxySeed, recoveryStatePath
	proxyBinary, proxySeed = filepath.Join(root, "bin", "usb-proxy"), seed
	recoveryStatePath = filepath.Join(root, "state", "session.json")
	t.Cleanup(func() {
		proxyBinary, proxySeed, recoveryStatePath = previousBinary, previousSeed, previousState
	})
	return root
}

func TestStartExtractsTheSeededProxyOnce(t *testing.T) {
	manager, _, _, _ := newTestManager(t)

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	defer func() { _ = manager.Stop() }()

	info, err := os.Stat(proxyBinary)
	if err != nil {
		t.Fatalf("the seed was never extracted: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}

	raw, err := os.ReadFile(proxyBinary)
	if err != nil || string(raw) != proxyELF {
		t.Fatalf("extracted %q, want %q (err %v)", raw, proxyELF, err)
	}

	// A second install must not rewrite a binary someone may have replaced.
	if err := os.WriteFile(proxyBinary, []byte("custom"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := installProxy(); err != nil {
		t.Fatalf("installProxy: %s", err)
	}
	if raw, err := os.ReadFile(proxyBinary); err != nil || string(raw) != "custom" {
		t.Fatalf("installProxy overwrote an existing binary with %q", raw)
	}
}

func TestStartWithoutASeedKeepsTheGadget(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)
	if err := os.Remove(proxySeed); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); !errors.Is(err, ErrNoProxy) {
		t.Fatalf("start = %v, want %v", err, ErrNoProxy)
	}

	attached, detached := vhci.calls()
	if len(attached) != 0 || len(detached) != 0 {
		t.Fatalf("a start with no proxy touched vhci: attached = %v, detached = %v", attached, detached)
	}
	if bound, surrendered, _ := gadget.state(); !bound || surrendered != 0 {
		t.Fatalf("bound = %t, surrenders = %d, want true, 0", bound, surrendered)
	}
}

func TestStartTakesTheGadgetAndStopRestoresIt(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)

	session, err := manager.Start(context.Background(), "10.0.0.5:3240", "1-1")
	if err != nil {
		t.Fatalf("start: %s", err)
	}

	if bound, surrendered, _ := gadget.state(); bound || surrendered != 1 {
		t.Fatalf("after start: bound = %t, surrenders = %d, want false, 1", bound, surrendered)
	}
	if session.Local.Bus != 3 || session.Local.Address != 7 {
		t.Fatalf("located %d:%d, want 3:7", session.Local.Bus, session.Local.Address)
	}

	// The device selector is the local bus and address, never the vendor and
	// product pair usb-proxy would match a local device with.
	want := []string{proxyBinary, "--device", testUDC, "--driver", "dwc2", "--device_bus", "3", "--device_addr", "7", "--auto_remap_endpoints"}
	if argv := spawner.last(); !slices.Equal(argv, want) {
		t.Fatalf("argv = %v, want %v", argv, want)
	}

	status := manager.Status()
	if !status.Active || !status.HIDSurrendered || status.Port != 5 || status.Device.IDVendor != "046d" {
		t.Fatalf("status = %+v", status)
	}

	if err := manager.Stop(); err != nil {
		t.Fatalf("stop: %s", err)
	}
	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 1 {
		t.Fatalf("after stop: bound = %t, reclaims = %d, want true, 1", bound, reclaimed)
	}

	attached, detached := vhci.calls()
	if !slices.Equal(attached, []string{"1-1"}) || !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("attached = %v, detached = %v", attached, detached)
	}
	if manager.Status().Active {
		t.Fatal("a stopped session still reports active")
	}
}

func TestHybridKeepsBootHIDAndRestoresTheGadget(t *testing.T) {
	manager, base, vhci, spawner := newTestManager(t)
	gadget := &fakeHybridGadget{fakeGadget: base}
	factory := &fakeHybridFactory{relay: newFakeHybridRelay()}
	manager.gadget = gadget
	manager.hybrid = factory

	session, err := manager.StartMode(context.Background(), "10.0.0.5", "1-1", ModeHybrid, false)
	if err != nil {
		t.Fatal(err)
	}
	if session.Mode != ModeHybrid || manager.Status().HIDSurrendered {
		t.Fatalf("Hybrid status = %+v", manager.Status())
	}
	if len(spawner.last()) != 0 {
		t.Fatal("Hybrid spawned the Exact proxy")
	}
	modules := manager.modules.(*fakeModules)
	modules.mu.Lock()
	loaded := slices.Clone(modules.loaded)
	modules.mu.Unlock()
	if !slices.Equal(loaded, []Module{ModuleUSBIPCore, ModuleVHCI}) {
		t.Fatalf("Hybrid modules = %v", loaded)
	}
	if err := manager.Stop(); err != nil {
		t.Fatal(err)
	}
	gadget.muHybrid.Lock()
	started, stopped := gadget.started, gadget.stopped
	gadget.muHybrid.Unlock()
	if started != 1 || stopped != 1 || factory.prepared != 1 || factory.cleaned != 1 {
		t.Fatalf("start/stop/prepare/cleanup = %d/%d/%d/%d", started, stopped, factory.prepared, factory.cleaned)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v", detached)
	}
}

// 9.1: mount -t functionfs hybrid returns ENODEV until configfs holds
// functions/ffs.hybrid, and the UDC cannot bind before ep0 carries the
// descriptors, so this sequence is the whole of Hybrid working on a real kernel.
func TestHybridRegistersTheFFSInstanceBeforeMounting(t *testing.T) {
	manager, base, _, _ := newTestManager(t)
	trace := &lifecycleTrace{}
	gadget := &fakeHybridGadget{fakeGadget: base, trace: trace}
	factory := &fakeHybridFactory{relay: newFakeHybridRelay(), trace: trace}
	manager.gadget = gadget
	manager.hybrid = factory

	if _, err := manager.StartMode(context.Background(), "10.0.0.5", "1-1", ModeHybrid, false); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = manager.Stop() }()

	want := []string{
		"configfs mkdir functions/ffs.hybrid",
		"functionfs mount hybrid",
		"functionfs write ep0 descriptors",
		"configfs mkdir functions/ffs.hybrid",
		"configfs link configs/c.1/ffs.hybrid",
		"configfs bind udc",
	}
	events := trace.taken()
	if !slices.Equal(events, want) {
		t.Fatalf("lifecycle = %v, want %v", events, want)
	}
}

// What the fake gadget above stands in for: the real CreateFunctionFS is the
// configfs mkdir and nothing else.
func TestCreateFunctionFSMakesTheConfigFSInstance(t *testing.T) {
	ops := &recordMkdirOps{}
	gadget := presentation.NewManager(presentation.NewStore(), ops, presentation.LoadCapabilities())
	var _ FunctionFSGadget = gadget

	if err := gadget.CreateFunctionFS(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ops.mkdirs, []string{"functions/ffs.hybrid"}) {
		t.Fatalf("mkdirs = %v, want [functions/ffs.hybrid]", ops.mkdirs)
	}
}

type recordMkdirOps struct {
	mkdirs []string
}

func (o *recordMkdirOps) Mkdir(rel string) error {
	o.mkdirs = append(o.mkdirs, rel)
	return nil
}

func (o *recordMkdirOps) WriteFile(string, []byte) error  { return nil }
func (o *recordMkdirOps) ReadFile(string) ([]byte, error) { return nil, nil }
func (o *recordMkdirOps) Symlink(string, string) error    { return nil }
func (o *recordMkdirOps) Remove(string) error             { return nil }
func (o *recordMkdirOps) RemoveDir(string) error          { return nil }
func (o *recordMkdirOps) ListUDC() ([]string, error)      { return []string{testUDC}, nil }
func (o *recordMkdirOps) BindUDC(string) error            { return nil }
func (o *recordMkdirOps) UnbindUDC() error                { return nil }
func (o *recordMkdirOps) SetOTGRole(string) error         { return nil }
func (o *recordMkdirOps) ResetPHY(context.Context) error  { return nil }

func TestHybridRecoveryIsPersistentAndOrdered(t *testing.T) {
	manager, base, vhci, _ := newTestManager(t)
	gadget := &fakeHybridGadget{fakeGadget: base}
	factory := &fakeHybridFactory{relay: newFakeHybridRelay()}
	manager.gadget = gadget
	manager.hybrid = factory
	manager.orphans = func() error { return nil }
	if err := saveRecoveryState(recoveryState{Port: 5, Mode: ModeHybrid}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Recover(); err != nil {
		t.Fatal(err)
	}
	gadget.muHybrid.Lock()
	recovered := gadget.recovered
	gadget.muHybrid.Unlock()
	if recovered != 1 || factory.cleaned != 1 {
		t.Fatalf("recovered/cleaned = %d/%d", recovered, factory.cleaned)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v", detached)
	}
	if _, err := os.Stat(recoveryStatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery marker survives: %v", err)
	}
}

func TestAProxyThatExitsRestoresTheGadget(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}

	spawner.process.kill()

	select {
	case <-gadget.reclaimed:
	case <-time.After(2 * time.Second):
		t.Fatal("a dead proxy left the gadget unbound: no reclaim within 2s")
	}
	waitIdle(t, manager)

	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 1 {
		t.Fatalf("after the proxy died: bound = %t, reclaims = %d, want true, 1", bound, reclaimed)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
	if err := manager.Stop(); !errors.Is(err, ErrNoSession) {
		t.Fatalf("stop after a crash: err = %v, want %v", err, ErrNoSession)
	}
}

func TestCloseStopsTheProxyAndClearsRecoveryState(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	if _, err := os.Stat(recoveryStatePath); err != nil {
		t.Fatalf("recovery state after start: %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close: %s", err)
	}
	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 1 {
		t.Fatalf("after close: bound = %t, reclaims = %d, want true, 1", bound, reclaimed)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
	if _, err := os.Stat(recoveryStatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery state survived close: %v", err)
	}
}

func TestRecoverStopsTheOrphanBeforeReclaimingTheGadget(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)
	gadget.bound = false
	if err := saveRecoveryState(recoveryState{Port: 5, Reclaim: true}); err != nil {
		t.Fatal(err)
	}

	called := false
	gadget.beforeReclaim = func() {
		if !called {
			t.Fatal("gadget reclaimed before orphan proxy stopped")
		}
	}
	manager.orphans = func() error {
		called = true
		return nil
	}
	if err := manager.Recover(); err != nil {
		t.Fatalf("recover: %s", err)
	}
	if !called {
		t.Fatal("recover did not stop orphan proxies")
	}
	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 1 {
		t.Fatalf("after recover: bound = %t, reclaims = %d, want true, 1", bound, reclaimed)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
	if _, err := os.Stat(recoveryStatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery state survived recovery: %v", err)
	}
}

func TestRecoverLeavesAnAlreadyBoundGadgetAlone(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)
	if err := saveRecoveryState(recoveryState{Port: 5}); err != nil {
		t.Fatal(err)
	}
	manager.orphans = func() error { return nil }

	if err := manager.Recover(); err != nil {
		t.Fatalf("recover: %s", err)
	}
	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 0 {
		t.Fatalf("after recover: bound = %t, reclaims = %d, want true, 0", bound, reclaimed)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
}

func TestRecoverDoesNotReclaimWhileAnOrphanMayStillOwnTheUDC(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)
	gadget.bound = false
	if err := saveRecoveryState(recoveryState{Port: 5, Reclaim: true}); err != nil {
		t.Fatal(err)
	}
	refused := errors.New("orphan survived")
	manager.orphans = func() error { return refused }

	if err := manager.Recover(); !errors.Is(err, refused) {
		t.Fatalf("recover = %v, want %v", err, refused)
	}
	if bound, _, reclaimed := gadget.state(); bound || reclaimed != 0 {
		t.Fatalf("after failed recovery: bound = %t, reclaims = %d, want false, 0", bound, reclaimed)
	}
	if _, detached := vhci.calls(); len(detached) != 0 {
		t.Fatalf("detached = %v, want none", detached)
	}
}

func waitIdle(t *testing.T, manager *Manager) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for manager.Status().Active {
		if time.Now().After(deadline) {
			t.Fatal("the session was still active 2s after the proxy exited")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestASecondSessionIsRefused(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	if _, err := manager.Start(context.Background(), "10.0.0.9", "2-1"); !errors.Is(err, ErrSessionActive) {
		t.Fatalf("second start: err = %v, want %v", err, ErrSessionActive)
	}

	attached, detached := vhci.calls()
	if !slices.Equal(attached, []string{"1-1"}) || len(detached) != 0 {
		t.Fatalf("the refused session touched vhci: attached = %v, detached = %v", attached, detached)
	}
	if _, surrendered, reclaimed := gadget.state(); surrendered != 1 || reclaimed != 0 {
		t.Fatalf("the refused session touched the gadget: surrenders = %d, reclaims = %d", surrendered, reclaimed)
	}
	if err := manager.Stop(); err != nil {
		t.Fatalf("stop: %s", err)
	}
}

func TestAFailedSpawnGivesTheGadgetBack(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)
	spawner.fail = errors.New("no such file")

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err == nil {
		t.Fatal("start succeeded with a spawner that cannot spawn")
	}
	if bound, surrendered, reclaimed := gadget.state(); !bound || surrendered != 1 || reclaimed != 1 {
		t.Fatalf("bound = %t, surrenders = %d, reclaims = %d, want true, 1, 1", bound, surrendered, reclaimed)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
	if manager.Status().Active {
		t.Fatal("a failed start left a session behind")
	}
}

func TestIsochronousDescriptorsAreRejectedBeforeTheGadgetIsTouched(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)
	descriptors := filepath.Join(vhciRoot, "usb3", "3-6", "descriptors")
	raw := []byte{
		18, 1, 0, 2, 0, 0, 0, 64, 0x6d, 0x04, 0x1c, 0xc3, 0, 1, 1, 2, 3, 1,
		9, 2, 25, 0, 1, 1, 0, 0x80, 50,
		9, 4, 0, 0, 1, 0, 0, 0, 0,
		7, 5, 0x81, 0x01, 0, 3, 1,
	}
	if err := os.WriteFile(descriptors, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); !errors.Is(err, ErrIsochronous) {
		t.Fatalf("start = %v, want %v", err, ErrIsochronous)
	}
	if reason := manager.Status().Reason; !strings.Contains(reason, "isochronous") {
		t.Fatalf("status reason = %q, want the refusal", reason)
	}
	if bound, surrendered, reclaimed := gadget.state(); !bound || surrendered != 0 || reclaimed != 0 {
		t.Fatalf("bound = %t, surrenders = %d, reclaims = %d, want true, 0, 0", bound, surrendered, reclaimed)
	}
	if argv := spawner.last(); argv != nil {
		t.Fatalf("proxy ran for an isochronous device: %v", argv)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
}

// The refusal has to name the endpoint that caused it. A device with a bulk
// pair and one isochronous IN is refused for 0x83 and for nothing else.
func TestTheIsochronousRefusalNamesTheOffendingEndpoint(t *testing.T) {
	manager, _, _, _ := newTestManager(t)
	descriptors := filepath.Join(vhciRoot, "usb3", "3-6", "descriptors")
	raw := []byte{
		18, 1, 0, 2, 0, 0, 0, 64, 0x6d, 0x04, 0x1c, 0xc3, 0, 1, 1, 2, 3, 1,
		9, 2, 39, 0, 1, 1, 0, 0x80, 50,
		9, 4, 0, 0, 3, 0, 0, 0, 0,
		7, 5, 0x81, 0x02, 0, 2, 0,
		7, 5, 0x02, 0x02, 0, 2, 0,
		7, 5, 0x83, 0x01, 0, 3, 1,
	}
	if err := os.WriteFile(descriptors, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := manager.StartMode(context.Background(), "10.0.0.5", "1-1", ModeExact, false)
	if !errors.Is(err, ErrIsochronous) {
		t.Fatalf("start = %v, want %v", err, ErrIsochronous)
	}
	if !strings.Contains(err.Error(), "endpoint 0x83") {
		t.Fatalf("refusal = %q, want it to name endpoint 0x83", err)
	}
	if reason := manager.Status().Reason; !strings.Contains(reason, "endpoint 0x83") {
		t.Fatalf("status reason = %q, want it to name endpoint 0x83", reason)
	}
}

// The guard is an opt-out, not a wall: a start that allowed isochronous
// transfers surrenders the UDC and runs the proxy for the same descriptors the
// default refuses, and the proxy is told the isochronous batch to use.
func TestAllowingIsochronousStartsTheProxyForAStreamingDevice(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)
	descriptors := filepath.Join(vhciRoot, "usb3", "3-6", "descriptors")
	raw := []byte{
		18, 1, 0, 2, 0, 0, 0, 64, 0x6d, 0x04, 0x1c, 0xc3, 0, 1, 1, 2, 3, 1,
		9, 2, 25, 0, 1, 1, 0, 0x80, 50,
		9, 4, 0, 0, 1, 0, 0, 0, 0,
		7, 5, 0x81, 0x01, 0, 3, 1,
	}
	if err := os.WriteFile(descriptors, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.StartMode(context.Background(), "10.0.0.5", "1-1", ModeExact, true); err != nil {
		t.Fatalf("start with isochronous allowed = %v", err)
	}
	if !vhci.isoAllowed() {
		t.Fatal("the attach was not told isochronous transfers were allowed")
	}
	if bound, surrendered, _ := gadget.state(); bound || surrendered != 1 {
		t.Fatalf("bound = %t, surrenders = %d, want false, 1", bound, surrendered)
	}
	argv := spawner.last()
	if argv == nil {
		t.Fatal("the proxy did not run for an allowed isochronous device")
	}
	if !slices.Contains(argv, "--iso_batch_size") {
		t.Fatalf("argv = %v, want it to carry --iso_batch_size", argv)
	}
	if i := slices.Index(argv, "--iso_batch_size"); argv[i+1] != strconv.Itoa(isoBatchPackets) {
		t.Fatalf("--iso_batch_size = %q, want %d", argv[i+1], isoBatchPackets)
	}
}

// A plain session must keep the argv it always had, so enabling the opt-in is
// the only thing that changes what the proxy is asked to do.
func TestADefaultStartDoesNotMentionIsochronousInTheArgv(t *testing.T) {
	argv := proxyArgv("/etc/kvm/bin/usb-proxy", "4340000.usb", "dwc2", Local{Bus: 3, Address: 7}, false)
	if slices.Contains(argv, "--iso_batch_size") {
		t.Fatalf("argv = %v, want no isochronous flag", argv)
	}
}

func TestMalformedDescriptorsAreRejectedBeforeTheGadgetIsTouched(t *testing.T) {
	manager, gadget, vhci, _ := newTestManager(t)
	descriptors := filepath.Join(vhciRoot, "usb3", "3-6", "descriptors")
	if err := os.WriteFile(descriptors, []byte{7, 5, 0x81}, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); !errors.Is(err, ErrDescriptors) {
		t.Fatalf("start = %v, want %v", err, ErrDescriptors)
	}
	if bound, surrendered, _ := gadget.state(); !bound || surrendered != 0 {
		t.Fatalf("bound = %t, surrenders = %d, want true, 0", bound, surrendered)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
}

// The three steps before the gadget is touched. Each must leave the port
// unattached, the gadget bound and no session behind.
func TestAFailureBeforeTheGadgetLeavesEverythingAlone(t *testing.T) {
	refused := errors.New("refused")

	for name, breakIt := range map[string]func(*Manager, *fakeGadget, *fakeVHCI){
		"modules": func(m *Manager, _ *fakeGadget, _ *fakeVHCI) {
			m.modules = &fakeModules{fail: refused}
		},
		"attach": func(_ *Manager, _ *fakeGadget, v *fakeVHCI) { v.fail = refused },
	} {
		t.Run(name, func(t *testing.T) {
			manager, gadget, vhci, _ := newTestManager(t)
			breakIt(manager, gadget, vhci)

			if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); !errors.Is(err, refused) {
				t.Fatalf("start = %v, want %v", err, refused)
			}
			if _, detached := vhci.calls(); len(detached) != 0 {
				t.Fatalf("detached = %v, want none", detached)
			}
			if bound, surrendered, reclaimed := gadget.state(); !bound || surrendered != 0 || reclaimed != 0 {
				t.Fatalf("bound = %t, surrenders = %d, reclaims = %d", bound, surrendered, reclaimed)
			}
			if manager.Status().Active {
				t.Fatal("a failed start left a session behind")
			}
		})
	}
}

// A gadget that will not give up the UDC has nothing to reclaim, so the port is
// detached and the gadget is left exactly as it was found.
func TestAGadgetThatRefusesToSurrenderDetachesThePort(t *testing.T) {
	manager, gadget, vhci, spawner := newTestManager(t)
	refused := errors.New("udc busy")
	gadget.fail = refused

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); !errors.Is(err, refused) {
		t.Fatalf("start = %v, want %v", err, refused)
	}
	if _, detached := vhci.calls(); !slices.Equal(detached, []uint32{5}) {
		t.Fatalf("detached = %v, want [5]", detached)
	}
	if bound, surrendered, reclaimed := gadget.state(); !bound || surrendered != 0 || reclaimed != 0 {
		t.Fatalf("bound = %t, surrenders = %d, reclaims = %d, want true, 0, 0", bound, surrendered, reclaimed)
	}
	if argv := spawner.last(); argv != nil {
		t.Fatalf("the proxy ran without a udc: %v", argv)
	}
	if manager.Status().Active {
		t.Fatal("a failed start left a session behind")
	}
}

func TestStartLoadsTheModulesItNeeds(t *testing.T) {
	manager, _, _, _ := newTestManager(t)
	modules := &fakeModules{}
	manager.modules = modules

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	defer func() { _ = manager.Stop() }()

	want := []Module{ModuleUSBIPCore, ModuleVHCI, ModuleRawGadget}
	if !slices.Equal(modules.loaded, want) {
		t.Fatalf("loaded = %v, want %v", modules.loaded, want)
	}
}

func fakeModuleTree(t *testing.T, files ...string) *int {
	t.Helper()

	root, sys := t.TempDir(), t.TempDir()
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte("ko"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	previousRoots, previousSys, previousInsmod := moduleRoots, sysModuleDir, insmod
	moduleRoots, sysModuleDir = []string{root}, sys

	calls := 0
	insmod = func(path string) error {
		calls++
		return os.MkdirAll(filepath.Join(sys, moduleNameOf(path)), 0o755)
	}
	t.Cleanup(func() {
		moduleRoots, sysModuleDir, insmod = previousRoots, previousSys, previousInsmod
	})
	return &calls
}

func moduleNameOf(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), ".ko")
	return strings.ReplaceAll(name, "-", "_")
}

func TestLoadingModulesIsIdempotent(t *testing.T) {
	calls := fakeModuleTree(t, "usbip-core.ko", "vhci-hcd.ko", "raw_gadget.ko")

	for range 3 {
		if err := (Insmod{}).Load(sessionModules...); err != nil {
			t.Fatalf("load: %s", err)
		}
	}
	if *calls != len(sessionModules) {
		t.Fatalf("insmod ran %d times, want %d", *calls, len(sessionModules))
	}

	for _, module := range sessionModules {
		loaded, err := module.Loaded()
		if err != nil || !loaded {
			t.Fatalf("%s loaded = %t, err = %v", module, loaded, err)
		}
	}
}

func TestAModuleMissingFromTheImageIsNamed(t *testing.T) {
	fakeModuleTree(t, "usbip-core.ko", "vhci-hcd.ko")

	err := (Insmod{}).Load(sessionModules...)
	if !errors.Is(err, ErrModuleMissing) {
		t.Fatalf("load: err = %v, want %v", err, ErrModuleMissing)
	}
	if !strings.Contains(err.Error(), string(ModuleRawGadget)) {
		t.Fatalf("error %q does not name the missing module", err)
	}
}

func TestTheSpawnerRefusesAnyBinaryButTheProxy(t *testing.T) {
	for _, argv := range [][]string{nil, {"/bin/sh", "-c", "id"}, {"usb-proxy"}} {
		if _, err := (execSpawner{}).Start(argv); !errors.Is(err, ErrRefusedBinary) {
			t.Fatalf("start %v: err = %v, want %v", argv, err, ErrRefusedBinary)
		}
	}
}

// The three things the log has to say: why this proxy refused the device, that
// the next session does not inherit it, and that a stop the operator asked for
// is not a failure at all.
func TestARefusedProxyReportsWhyFromItsLog(t *testing.T) {
	manager, _, _, spawner := newTestManager(t)
	previous := proxyLog
	proxyLog = filepath.Join(t.TempDir(), "usb-proxy.log")
	t.Cleanup(func() { proxyLog = previous })

	appendLog := func(line string) {
		t.Helper()
		file, err := os.OpenFile(proxyLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteString(line); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	appendLog("Failed to remap endpoint 0x81 (interface 0, alt 0)\n")
	spawner.process.kill()
	waitIdle(t, manager)
	if reason := manager.Status().Reason; reason != "Failed to remap endpoint 0x81 (interface 0, alt 0)" {
		t.Fatalf("refusal reason = %q", reason)
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("second start: %s", err)
	}
	spawner.process.kill()
	waitIdle(t, manager)
	if reason := manager.Status().Reason; reason != "" {
		t.Fatalf("a silent proxy inherited %q", reason)
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("third start: %s", err)
	}
	appendLog("Received SIGTERM, stopping...\n")
	if err := manager.Stop(); err != nil {
		t.Fatalf("stop: %s", err)
	}
	if reason := manager.Status().Reason; reason != "" {
		t.Fatalf("a stop the operator asked for reported %q", reason)
	}
}

func TestAProxyThatStopsMakingProgressIsTerminated(t *testing.T) {
	manager, gadget, _, spawner := newTestManager(t)
	previous := livenessInterval
	livenessInterval = time.Millisecond
	t.Cleanup(func() { livenessInterval = previous })
	manager.progress = func(int) (proxyProgress, error) {
		return proxyProgress{cpu: 1234, blocked: true}, nil
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	select {
	case <-spawner.process.terminate:
	case <-time.After(2 * time.Second):
		t.Fatal("the watchdog left a stuck proxy holding the UDC")
	}
	waitIdle(t, manager)

	if bound, _, reclaimed := gadget.state(); !bound || reclaimed != 1 {
		t.Fatalf("after the watchdog fired: bound = %t, reclaims = %d, want true, 1", bound, reclaimed)
	}
	if reason := manager.Status().Reason; !strings.Contains(reason, "made no progress") {
		t.Fatalf("status reason = %q", reason)
	}
}

// A proxy waiting on a quiet host looks blocked too, and killing it would cost
// the operator the session, so only a proxy that also burns no CPU is stopped.
func TestAProxyThatKeepsWorkingIsLeftAlone(t *testing.T) {
	manager, _, _, spawner := newTestManager(t)
	previous := livenessInterval
	livenessInterval = time.Millisecond
	t.Cleanup(func() { livenessInterval = previous })
	var cpu atomic.Uint64
	manager.progress = func(int) (proxyProgress, error) {
		return proxyProgress{cpu: cpu.Add(1), blocked: true}, nil
	}

	if _, err := manager.Start(context.Background(), "10.0.0.5", "1-1"); err != nil {
		t.Fatalf("start: %s", err)
	}
	select {
	case <-spawner.process.terminate:
		t.Fatal("the watchdog killed a proxy that was still working")
	case <-time.After(200 * time.Millisecond):
	}
	if !manager.Status().Active {
		t.Fatal("the session ended on its own")
	}
	if err := manager.Stop(); err != nil {
		t.Fatalf("stop: %s", err)
	}
}

func TestProxyProgressReadsEveryThread(t *testing.T) {
	root := t.TempDir()
	previous := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = previous })

	writeThreadStat := func(tid int, state string, user int, system int) {
		t.Helper()
		dir := filepath.Join(root, "4242", "task", strconv.Itoa(tid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		// The comm field holds a space and a parenthesis of its own.
		stat := fmt.Sprintf("%d (usb-proxy (hs)) %s%s %d %d 0 0\n", tid, state, strings.Repeat(" 0", 10), user, system)
		if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeThreadStat(4242, "D", 7, 9)
	writeThreadStat(4243, "D", 3, 4)
	progress, err := readProxyProgress(4242)
	if err != nil {
		t.Fatal(err)
	}
	if !progress.blocked || progress.cpu != 23 {
		t.Fatalf("progress = %+v, want blocked with 23 jiffies", progress)
	}

	writeThreadStat(4243, "S", 3, 4)
	progress, err = readProxyProgress(4242)
	if err != nil {
		t.Fatal(err)
	}
	if progress.blocked {
		t.Fatalf("one interruptible thread still reads as blocked: %+v", progress)
	}
}

func TestAHybridRelayFailureReachesTheStatus(t *testing.T) {
	manager, base, _, _ := newTestManager(t)
	gadget := &fakeHybridGadget{fakeGadget: base}
	relay := newFakeHybridRelay()
	relay.fail = fmt.Errorf("source endpoint 0x83: %w", functionfs.ErrTransferTime)
	manager.gadget = gadget
	manager.hybrid = &fakeHybridFactory{relay: relay}

	session, err := manager.StartMode(context.Background(), "10.0.0.5", "1-1", ModeHybrid, false)
	if err != nil {
		t.Fatalf("start: %s", err)
	}
	<-session.Done()
	if reason := manager.Status().Reason; !strings.Contains(reason, "transfer timed out") {
		t.Fatalf("status reason = %q", reason)
	}
}
