package presentation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The layout the operator chose on the device that lost its keyboard: all three
// roles on one interface, multiplexed behind a Report ID, nine bytes a report.
func collapsedProfile(t *testing.T, name string) Profile {
	t.Helper()

	profile := standardProfile()
	profile.Name, profile.BuiltIn = name, false
	if err := SetHIDLayout(&profile, [][]HIDRole{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}}); err != nil {
		t.Fatalf("collapse hid layout: %v", err)
	}
	profile.Name, profile.BuiltIn = name, false
	profile.Normalize()
	return profile
}

// The state the device was actually found in: S03usbdev built the stock
// three-HID gadget at boot, and the store went on promising a layout that an
// earlier apply had failed to land and rolled back from.
func divergedManager(t *testing.T) (*Manager, *RecordOps, Profile) {
	t.Helper()

	manager, ops := newTestManager(t)
	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatalf("build the stock gadget: %v", err)
	}

	collapsed := collapsedProfile(t, ProfileCurrent)
	if err := manager.store.SaveProfile(collapsed); err != nil {
		t.Fatalf("save collapsed profile: %v", err)
	}
	if err := manager.store.SetActive(collapsed.Name); err != nil {
		t.Fatalf("set active: %v", err)
	}
	return manager, ops, collapsed
}

type routingHID struct {
	fakeHID
	routes []HIDRoute
}

func (r *routingHID) SetHIDRoutes(routes []HIDRoute) {
	r.routes = append([]HIDRoute(nil), routes...)
}

func linkedNames(ops *RecordOps) []string {
	var names []string
	for link := range ops.Links() {
		if rest, ok := strings.CutPrefix(link, configPrefix+"/"); ok {
			names = append(names, rest)
		}
	}
	slices.Sort(names)
	return names
}

// The bug this whole path exists for. The store promised one HID interface with
// nine-byte report-ID-multiplexed reports; the kernel was presenting the stock
// three, because S03usbdev rebuilds them from scratch on every boot and Migrate
// returns early forever after the first one. Nothing reasserted the profile and
// nothing noticed, so the server wrote nine-byte reports into an eight-byte boot
// keyboard and the host showed HID devices that controlled nothing.
func TestReconcileReassertsAProfileTheBootScriptOverwrote(t *testing.T) {
	manager, ops, collapsed := divergedManager(t)

	if got := linkedNames(ops); !slices.Equal(got, []string{"hid.GS0", "hid.GS1", "hid.GS2"}) {
		t.Fatalf("gadget starts linked as %v, want the stock three", got)
	}

	if err := manager.ReconcileGadget(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if got := linkedNames(ops); !slices.Equal(got, []string{"hid.GS0"}) {
		t.Fatalf("linked = %v, want the collapsed layout's single interface", got)
	}
	length, ok := readLiveUint(ops, configPrefix+"/hid.GS0/report_length")
	if !ok || length != uint64(collapsed.Functions[0].HID.ReportLength) {
		t.Fatalf("report_length = %d ok = %v, want %d", length, ok, collapsed.Functions[0].HID.ReportLength)
	}
}

// A gadget that already matches is not worth an unbind: every apply takes the
// controller away from the attached host and back, and doing that on every boot
// for nothing is its own bug.
func TestReconcileLeavesAMatchingGadgetAlone(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	before := len(ops.Trace())
	if err := manager.ReconcileGadget(ctx); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("reconcile emitted %d ops against a matching gadget, want none", after-before)
	}
}

// The reconcile runs at boot, before anyone is watching. An apply that fails
// halfway and leaves no UDC bound would ship a device with no USB at all, so
// every rung of the ladder ends in a bind and the reconcile reports rather than
// repairs.
func TestReconcileLeavesTheGadgetBoundWhenItCannotLand(t *testing.T) {
	manager, ops, _ := divergedManager(t)

	// Target, rollback and hid-only fallback all fail on the same write.
	path := stringsDir + "/product"
	for _, err := range []error{
		errors.New("target write failed"),
		errors.New("rollback write failed"),
		errors.New("fallback write failed"),
	} {
		ops.FailWriteOnce(path, err)
	}

	err := manager.ReconcileGadget(context.Background())
	if err == nil {
		t.Fatal("reconcile against a gadget that cannot be reasserted returned no error")
	}
	if !strings.Contains(err.Error(), "reassert "+ProfileCurrent) {
		t.Fatalf("err = %v, want the reassert named", err)
	}
	if bound := ops.Bound(); bound != dwc2Device {
		t.Fatalf("bound = %q, want the controller left bound to %q", bound, dwc2Device)
	}
}

// The safety property that matters most. If the reconcile is impossible or its
// apply fails, the server has to drive the layout that is actually bound rather
// than go on writing reports shaped for the one the store remembers. A user
// with a mismatched profile still gets a working keyboard.
func TestHIDRoutesFollowTheLiveGadgetRatherThanTheStore(t *testing.T) {
	manager, _, collapsed := divergedManager(t)

	// What the store promises, for contrast: one node, three report IDs.
	if got := HIDRoutes(collapsed.Functions); len(got) != 3 || got[0].Path != "/dev/hidg0" || got[0].ReportID != 1 {
		t.Fatalf("stored routes = %+v, want the collapsed single-node layout", got)
	}

	h := &routingHID{}
	manager.SetHID(h)

	want := []HIDRoute{
		{Role: HIDRoleKeyboard, Path: "/dev/hidg0", Length: 8},
		{Role: HIDRoleRelative, Path: "/dev/hidg1", Length: 4},
		{Role: HIDRoleAbsolute, Path: "/dev/hidg2", Length: 6},
	}
	if !slices.Equal(h.routes, want) {
		t.Fatalf("routes = %+v, want the live three-interface layout %+v", h.routes, want)
	}
}

// f_hid publishes the device number of the node it owns, and the minor is the N
// in /dev/hidgN. Reading it back beats counting link order, which is the
// assumption that breaks the moment anything renumbers.
func TestHIDRoutesTakeTheNodeNumberFromTheKernel(t *testing.T) {
	manager, ops, _ := divergedManager(t)
	for instance, dev := range map[string]string{"GS0": "238:3", "GS1": "238:4", "GS2": "238:5"} {
		if err := ops.Seed(functionsDir+"/hid."+instance+"/dev", []byte(dev+"\n")); err != nil {
			t.Fatalf("seed dev: %v", err)
		}
	}

	h := &routingHID{}
	manager.SetHID(h)

	want := []string{"/dev/hidg3", "/dev/hidg4", "/dev/hidg5"}
	var got []string
	for _, route := range h.routes {
		got = append(got, route.Path)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("route paths = %v, want %v", got, want)
	}
}

// A live descriptor this package did not compose cannot be decoded into roles,
// but a stored profile carrying the same bytes names them outright. Without
// that, an imported profile would lose its routing the moment the routes
// started following the gadget.
func TestHIDRoutesNameAnImportedDescriptorFromTheStore(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	custom := standardProfile()
	custom.Name, custom.BuiltIn = ProfileCurrent, false
	custom.Functions[0].HID.ReportDesc = append(slices.Clone(descKeyboardStandard), 0xC0)
	custom.Normalize()
	if err := manager.ApplyProfile(ctx, custom); err != nil {
		t.Fatalf("apply custom profile: %v", err)
	}
	if _, ok := inferHIDRoles(custom.Functions[0].HID.ReportDesc); ok {
		t.Fatal("the custom descriptor was decodable after all; pick a different one")
	}
	_ = ops

	h := &routingHID{}
	manager.SetHID(h)

	if len(h.routes) == 0 || h.routes[0].Role != HIDRoleKeyboard || h.routes[0].Path != "/dev/hidg0" {
		t.Fatalf("routes = %+v, want the stored profile's role names on the live nodes", h.routes)
	}
}

// With nothing readable to follow, the historical hidg0/hidg1/hidg2 mapping is
// what an unwired Hid keeps. Telling it every role is absent would leave it
// opening nothing at all.
func TestHIDRoutesAreLeftAloneWithNothingToSay(t *testing.T) {
	useTestPresentationDir(t)
	useTestBootDir(t)
	manager := NewManager(NewStore(), nil, staticV1)

	h := &routingHID{}
	manager.SetHID(h)
	if h.routes != nil {
		t.Fatalf("routes = %+v, want none pushed", h.routes)
	}
}

// The store must not go on naming a layout the controller is not presenting.
// applyPlan's ladder reverts the store along with the gadget on every rung that
// completes; this is the case where none of them did.
func TestFailedApplyRecordsThatTheActiveProfileIsNotLive(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()
	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	path := stringsDir + "/product"
	for _, err := range []error{
		errors.New("target write failed"),
		errors.New("rollback write failed"),
		errors.New("fallback write failed"),
	} {
		ops.FailWriteOnce(path, err)
	}

	err := manager.ApplyProfile(ctx, collapsedProfile(t, "target"))
	if !errors.Is(err, ErrDiverged) {
		t.Fatalf("err = %v, want it to carry %v", err, ErrDiverged)
	}
	if !strings.Contains(err.Error(), "hid.GS1") {
		t.Fatalf("err = %v, want the functions the gadget is missing named", err)
	}

	snapshot, snapErr := manager.Snapshot()
	if snapErr != nil {
		t.Fatalf("snapshot: %v", snapErr)
	}
	if snapshot.Diverged == nil {
		t.Fatal("snapshot reports no divergence against a gadget that is missing two of its three hid interfaces")
	}
	if snapshot.Diverged.Profile != ProfileStandard {
		t.Fatalf("divergence names %q, want the profile the store still calls active", snapshot.Diverged.Profile)
	}
}

// A gadget that matches its profile is not reported as diverged, or the field
// says nothing.
func TestSnapshotReportsNoDivergenceAgainstAMatchingGadget(t *testing.T) {
	manager, _ := newTestManager(t)
	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}

	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Diverged != nil {
		t.Fatalf("diverged = %+v, want none", snapshot.Diverged)
	}
}

// Unlinking a net function does not destroy it. gether_setup holds the netdev
// until rmdir, so an unlinked rndis.usb0 goes on owning the name usb0 and the
// ncm function that replaced it is handed usb1, which is how a bridge port ends
// up on a netdev with no carrier and no function behind it.
func TestApplyReleasesTheDirectoryOfAReplacedNetworkFunction(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	rndis := profileForFlags(flags{rndis: true})
	rndis.Name, rndis.BuiltIn = "netrndis", false
	rndis.Normalize()
	if err := manager.ApplyProfile(ctx, rndis); err != nil {
		t.Fatalf("apply rndis profile: %v", err)
	}

	ncm := profileForFlags(flags{ncm: true})
	ncm.Name, ncm.BuiltIn = "netncm", false
	ncm.Normalize()
	if err := manager.ApplyProfile(ctx, ncm); err != nil {
		t.Fatalf("apply ncm profile: %v", err)
	}

	released := false
	for _, op := range ops.Trace() {
		if op.Kind == OpRmdir && op.Path == functionsDir+"/rndis.usb0" {
			released = true
		}
		// The one rule this must not break: removing a hid function directory
		// returns its /dev/hidgN minor to the ida and renumbers the rest.
		if op.Kind == OpRmdir && strings.HasPrefix(op.Path, functionsDir+"/hid.") {
			t.Fatalf("op %s %s removed a hid function directory", op.Kind, op.Path)
		}
	}
	if !released {
		t.Fatalf("functions/rndis.usb0 survived the switch to ncm, so it still owns the name %s", GadgetNIC)
	}
}

// The name is only free for the replacement if the release happens before the
// plan recreates it. rmdir on a still-linked function is -EBUSY, and removing
// one before the UDC is unbound can wedge the gadget, so the release sits
// between the unlink and the mkdir and nowhere else.
func TestTheReleaseSitsBetweenTheUnlinkAndTheReplacement(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx := context.Background()

	rndis := profileForFlags(flags{rndis: true})
	rndis.Name, rndis.BuiltIn = "netrndis", false
	rndis.Normalize()
	if err := manager.ApplyProfile(ctx, rndis); err != nil {
		t.Fatalf("apply rndis profile: %v", err)
	}

	mark := len(ops.Trace())
	ncm := profileForFlags(flags{ncm: true})
	ncm.Name, ncm.BuiltIn = "netncm", false
	ncm.Normalize()
	if err := manager.ApplyProfile(ctx, ncm); err != nil {
		t.Fatalf("apply ncm profile: %v", err)
	}

	unbind, unlink, rmdir, mkdir := -1, -1, -1, -1
	for i, op := range ops.Trace()[mark:] {
		switch {
		case op.Kind == OpUnbind && unbind < 0:
			unbind = i
		case op.Kind == OpUnlink && op.Path == configPrefix+"/rndis.usb0":
			unlink = i
		case op.Kind == OpRmdir && op.Path == functionsDir+"/rndis.usb0":
			rmdir = i
		case op.Kind == OpMkdir && op.Path == functionsDir+"/ncm.usb0" && mkdir < 0:
			mkdir = i
		}
	}
	if unbind < 0 || unlink < 0 || rmdir < 0 || mkdir < 0 {
		t.Fatalf("unbind=%d unlink=%d rmdir=%d mkdir=%d, want all four", unbind, unlink, rmdir, mkdir)
	}
	if !(unbind < unlink && unlink < rmdir && rmdir < mkdir) {
		t.Fatalf("order unbind=%d unlink=%d rmdir=%d mkdir=%d, want unbind < unlink < rmdir < mkdir",
			unbind, unlink, rmdir, mkdir)
	}
}

// gether asks the kernel for "usb%d" and gets the first free number, which is
// not necessarily zero. The bridge puts whatever comes back on a command line,
// so it has to be the kernel's answer and not the function instance's name.
func TestNICReportsTheInterfaceTheKernelActuallyNamed(t *testing.T) {
	manager, ops := newTestManager(t)
	seed(t, ops, functionsDir+"/ncm.usb0/dev_addr")
	seed(t, ops, configPrefix+"/ncm.usb0/dev_addr")
	for _, rel := range []string{functionsDir + "/ncm.usb0/ifname", configPrefix + "/ncm.usb0/ifname"} {
		if err := ops.Seed(rel, []byte("usb1\n")); err != nil {
			t.Fatalf("seed ifname: %v", err)
		}
	}

	nic, err := manager.NIC(context.Background())
	if err != nil {
		t.Fatalf("NIC: %v", err)
	}
	if nic != "usb1" {
		t.Fatalf("NIC = %q, want the live interface name usb1", nic)
	}
}

// f_uvc's usb_function_deactivate() outlives the composite that took it, so a
// gadget can bind, report its controller, and never turn the pullup back on.
// A host that had reached configured and does not reach it again is that
// gadget, and the only thing this side can do about it is say so.
func TestReconcileFlagsAGadgetThatBoundButNeverCameBack(t *testing.T) {
	manager, ops, _ := divergedManager(t)

	dir := t.TempDir()
	oldUDCDir, oldTimeout, oldPoll := udcDir, enumerateTimeout, enumeratePoll
	udcDir, enumerateTimeout, enumeratePoll = dir, 20*time.Millisecond, time.Millisecond
	t.Cleanup(func() { udcDir, enumerateTimeout, enumeratePoll = oldUDCDir, oldTimeout, oldPoll })

	state := filepath.Join(dir, dwc2Device, "state")
	if err := os.MkdirAll(filepath.Dir(state), 0o755); err != nil {
		t.Fatalf("make udc dir: %v", err)
	}
	if err := os.WriteFile(state, []byte(udcConfigured+"\n"), 0o600); err != nil {
		t.Fatalf("write udc state: %v", err)
	}

	// The reapply takes the controller away from the host, and this one never
	// gets it back.
	ops.OnBind(func() {
		_ = os.WriteFile(state, []byte("not attached\n"), 0o600)
	})

	if err := manager.ReconcileGadget(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snapshot.PendingPowerCycle {
		t.Fatal("a gadget that bound and never enumerated again is not reported as needing a power cycle")
	}
}

// The wait is only ever spent on a host that was there. A device booting with
// nothing plugged into it must not be delayed by it at all.
func TestReconcileDoesNotWaitOnAHostThatWasNeverAttached(t *testing.T) {
	manager, _, _ := divergedManager(t)

	dir := t.TempDir()
	oldUDCDir, oldTimeout := udcDir, enumerateTimeout
	udcDir, enumerateTimeout = dir, time.Hour
	t.Cleanup(func() { udcDir, enumerateTimeout = oldUDCDir, oldTimeout })

	done := make(chan error, 1)
	go func() { done <- manager.ReconcileGadget(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reconcile waited for a host that was never attached")
	}
}

// A transient owns the gadget on purpose, and reasserting a profile from under
// the process serving ep0 would take the functionfs away from it.
func TestReconcileRefusesWhileATransientIsUp(t *testing.T) {
	manager, _, _ := divergedManager(t)
	manager.mu.Lock()
	manager.setTransient(&Transient{Token: 1})
	manager.mu.Unlock()

	if err := manager.ReconcileGadget(context.Background()); !errors.Is(err, ErrTransient) {
		t.Fatalf("err = %v, want %v", err, ErrTransient)
	}
}

func TestHIDRoleGroupsCoverEveryLayoutTheCompilerCanBuild(t *testing.T) {
	groups := hidRoleGroups()
	if len(groups) != 15 {
		t.Fatalf("%d groups, want 15", len(groups))
	}
	for _, group := range groups {
		if err := ValidateHIDLayout([][]HIDRole{group}); err != nil {
			t.Fatalf("group %v: %v", group, err)
		}
		desc, _, err := composeHIDReport(group, 0)
		if err != nil {
			t.Fatalf("compose %v: %v", group, err)
		}
		roles, ok := inferHIDRoles(desc)
		if !ok || !slices.Equal(roles, group) {
			t.Fatalf("inferHIDRoles(%v) = %v ok = %v", group, roles, ok)
		}
	}
}

// withHIDQuiesced holds the HID lock for the whole bracket and pushes routes
// from its defer. A router that locks again there deadlocks against itself with
// every HID handle closed, which on hardware was a keyboard that died the
// instant an apply started and a save that never returned.
func TestQuiescedRoutePushDoesNotRelockTheHID(t *testing.T) {
	quiescer := &reentrantQuiescer{}
	manager := NewManager(NewStore(), nil, staticV1)
	manager.SetHID(quiescer)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = manager.withHIDQuiesced(func() error { return nil })
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the quiesce bracket never returned: its route push took the HID lock it already holds")
	}
	if quiescer.locked.Load() != 0 {
		t.Fatalf("HID left locked %d deep after the bracket", quiescer.locked.Load())
	}
}

// Models sync.Mutex's one property that matters here: it is not reentrant.
type reentrantQuiescer struct {
	locked atomic.Int32
	routes atomic.Int32
}

func (q *reentrantQuiescer) Lock() {
	if !q.locked.CompareAndSwap(0, 1) {
		panic("HID locked while already held: sync.Mutex would block here forever")
	}
}
func (q *reentrantQuiescer) Unlock()                                                { q.locked.Store(0) }
func (q *reentrantQuiescer) CloseNoLock()                                           {}
func (q *reentrantQuiescer) SetHIDRoutesLocked([]HIDRoute)                          { q.routes.Add(1) }
func (q *reentrantQuiescer) SetHIDRoutes(r []HIDRoute)                              { q.Lock(); q.routes.Add(1); q.Unlock() }
func (q *reentrantQuiescer) OpenNoLockWithRetry(time.Duration, time.Duration) error { return nil }
func (q *reentrantQuiescer) WriteKeyboardReport([]byte) error                       { return nil }
func (q *reentrantQuiescer) WriteRelativeMouseReport([]byte) error                  { return nil }
func (q *reentrantQuiescer) WriteAbsoluteMouseReport([]byte) error                  { return nil }

// Saving a panel without changing anything recompiles the same profile. Running
// the transaction anyway unbinds the UDC and binds it again, which the host
// reads as an unplug: it drops whatever camera stream was open and takes the
// keyboard away for the duration. A plan that asks for nothing must touch
// nothing.
func TestAnUnchangedProfileLeavesTheGadgetAlone(t *testing.T) {
	manager, ops := newTestManager(t)
	profile := standardProfile()
	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	settled := len(ops.Trace())

	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	// The guard above turns an empty plan into a no-op. One op still survives a
	// recompile of the same profile: osDesc() re-links os_desc/c.1 through
	// compiler.link, which always emits unlink+symlink whether or not the link
	// already points where it should. That single op is enough to make the plan
	// non-empty, so the transaction still runs and the host still sees an
	// unplug. Pinned rather than waved at: if anything else starts surviving an
	// unchanged apply, this fails and says what.
	var touched []string
	for _, op := range ops.Trace()[settled:] {
		switch op.Kind {
		case OpUnlink, OpRmdir, OpBind, OpUnbind:
			touched = append(touched, op.Kind.String()+" "+op.Path)
		}
	}
	// What it costs today, exactly. The os_desc unlink is the only real change,
	// and it drags the whole unbind/bind transaction with it - which the host
	// reads as an unplug and which drops an open camera stream. Fixing it means
	// teaching Snapshot to record the os_desc link so Reconcile can drop the
	// redundant relink; then the plan is empty, the guard in applyPlan fires,
	// and this list becomes empty too.
	want := "unbind UDC, unlink os_desc/c.1, bind UDC"
	if got := strings.Join(touched, ", "); got != want {
		t.Fatalf("an unchanged apply touched the gadget with [%s], want [%s]", got, want)
	}
}

// The redundant relink, isolated. compiler.link cannot see the current state so
// osDesc() always emits unlink+symlink for os_desc/c.1; when the link is already
// in place that pair changes nothing, and leaving it in is what keeps an
// otherwise empty plan non-empty - which drags the whole unbind/bind
// transaction along and reads to the host as an unplug.
func TestReconcileDropsARelinkThatChangesNothing(t *testing.T) {
	plan := Plan{Ops: []Op{
		{Kind: OpUnlink, Path: osDescDir + "/" + configName},
		{Kind: OpSymlink, Path: osDescDir + "/" + configName, Target: configPrefix},
	}}

	linked := Reconcile(Snapshot{OSDescLinked: true}, plan)
	if len(linked.Ops) != 0 {
		t.Fatalf("os_desc already linked, but the plan kept %d op(s): %v", len(linked.Ops), linked.Ops)
	}

	// With the link absent the pair is the whole point and must survive.
	absent := Reconcile(Snapshot{OSDescLinked: false}, plan)
	if len(absent.Ops) != 2 {
		t.Fatalf("os_desc not linked, but the plan kept %d op(s), want both", len(absent.Ops))
	}
}

// The other half of the rule: osDesc() emits a lone unlink, with use=0 and no
// symlink after it, when the last network function goes. That is a removal, not
// a redundant relink, and suppressing it leaves the gadget answering the 0xEE
// string request for a profile with nothing to describe (H11).
func TestReconcileKeepsALoneOSDescUnlink(t *testing.T) {
	plan := Plan{Ops: []Op{
		{Kind: OpWrite, Path: osDescDir + "/use", Data: []byte("0")},
		{Kind: OpUnlink, Path: osDescDir + "/" + configName},
	}}

	got := Reconcile(Snapshot{OSDescLinked: true}, plan)
	var unlinks int
	for _, op := range got.Ops {
		if op.Kind == OpUnlink && op.Path == osDescDir+"/"+configName {
			unlinks++
		}
	}
	if unlinks != 1 {
		t.Fatalf("the removal unlink was dropped: plan is %v", got.Ops)
	}
}

// S03usbdev leaves the UDC unbound so the host enumerates once, with the final
// layout. That only holds if every path out of the reconcile binds: an apply
// binds as part of its transaction, and the paths that decide there is nothing
// to apply have to bind for themselves. Miss one and the device never reaches
// a host at all, which is worse than the double enumeration it replaces.
func TestReconcileBindsWhenItDecidesNotToApply(t *testing.T) {
	manager, ops := newTestManager(t)
	profile := standardProfile()
	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatalf("seed apply: %v", err)
	}
	if err := ops.UnbindUDC(); err != nil {
		t.Fatalf("unbind: %v", err)
	}
	if got := ops.Bound(); got != "" {
		t.Fatalf("precondition: still bound to %q", got)
	}

	// The gadget now matches the profile, so the reconcile has nothing to apply.
	if err := manager.ReconcileGadget(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got := ops.Bound(); got == "" {
		t.Fatal("the reconcile found nothing to apply and left the controller unbound")
	}
}
