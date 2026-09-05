package presentation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ErrDiverged is what a failed apply carries when the store is left naming a
// profile the gadget is not presenting. It is a statement about the store, not
// about the transaction that failed, and it is derived rather than remembered.
var ErrDiverged = errors.New("the active profile is not live")

var (
	// The reconcile runs inside GetManager, on the boot path, before anything
	// is listening. It is bounded so that it can never be the reason the server
	// does not reach its listener; the configfs writes themselves are
	// synchronous and the context gates the transaction's entry points.
	//
	// Ten seconds because the first caller of GetManager is main.go's
	// startup.Run("usb passthrough", 15s, ...), which abandons its goroutine
	// and records a timeout rather than joining it: a reconcile that outlived
	// that budget would report itself as a passthrough failure and mask a real
	// one. The realistic cost is an apply plus at most enumerateTimeout.
	reconcileBudget = 10 * time.Second

	// Spent only when a host had already reached configured before the reapply,
	// so a device booting with nothing plugged into it is never delayed here.
	enumerateTimeout = 3 * time.Second
	enumeratePoll    = 100 * time.Millisecond
)

// Divergence is what the gadget the kernel bound does not have in common with
// the profile the store calls active. Missing is promised and absent, Extra is
// present and unpromised, Changed is present under the right name carrying the
// wrong descriptor.
type Divergence struct {
	Profile string   `json:"profile"`
	Missing []string `json:"missing,omitempty"`
	Extra   []string `json:"extra,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

func (d Divergence) Empty() bool {
	return len(d.Missing) == 0 && len(d.Extra) == 0 && len(d.Changed) == 0
}

func (d Divergence) String() string {
	var parts []string
	for _, part := range [...]struct {
		label string
		names []string
	}{
		{"is missing", d.Missing},
		{"also carries", d.Extra},
		{"differs in", d.Changed},
	} {
		if len(part.names) != 0 {
			parts = append(parts, part.label+" "+strings.Join(part.names, ", "))
		}
	}
	return strings.Join(parts, "; ")
}

// The gadget as the kernel has it, read back through the same linkage probe the
// snapshot uses and, for HID, through the attributes f_hid publishes. Ops has
// no readdir and is deliberately not given one for this: the instance names a
// profile may hold are a closed set, so walking them costs a handful of reads
// and leaves the privilege boundary exactly where it is.
func liveFunctions(ops Ops, extra []Function) []Function {
	if ops == nil {
		return nil
	}
	return liveFunctionsFrom(ops, readSnapshot(ops, extra).Linked)
}

func liveFunctionsFrom(ops Ops, linked []string) []Function {
	if ops == nil {
		return nil
	}

	live := make([]Function, 0, len(linked))
	hidSeen := 0
	for _, name := range linked {
		kind, instance, ok := splitFunctionName(name)
		if !ok {
			continue
		}
		function := Function{Kind: kind, Instance: instance}
		if kind == FunctionHID {
			hid, ok := readLiveHID(ops, name, hidNodeGuess(instance, hidSeen))
			if !ok {
				continue
			}
			function.HID = hid
			hidSeen++
		}
		live = append(live, function)
	}
	return live
}

// Where the node number comes from when f_hid does not publish dev. The
// instance name is the better guess than the count of HID functions seen so
// far, because the minor is handed out at mkdir in GS order and an unlinked
// GS0 does not give its minor back - hidg_free_inst does that, and this package
// never rmdirs a hid function. A gadget linking only GS1 and GS2 is therefore
// on hidg1 and hidg2, not on hidg0 and hidg1.
func hidNodeGuess(instance string, seen int) int {
	if index := slices.Index(hidInstances[:], instance); index >= 0 {
		return index
	}
	return seen
}

func splitFunctionName(name string) (FunctionKind, string, bool) {
	kind, instance, ok := strings.Cut(name, ".")
	if !ok || kind == "" || instance == "" {
		return "", "", false
	}
	return FunctionKind(kind), instance, true
}

// f_hid publishes protocol, subclass, report_length, report_desc and the device
// number of the /dev/hidgN it owns. None of those shows is guarded by
// opts->refcnt - only the stores are - so all of it is readable while the
// function is linked and running.
//
// dev is what pins the node. hidg_alloc takes the minor from an ida at mkdir
// and names the character device after it, so the minor is the N in /dev/hidgN
// and is a fact rather than the creation-order assumption everything else in
// this package has to make. The position is the fallback for a kernel that does
// not publish it.
func readLiveHID(ops Ops, name string, position int) (*HIDFunction, bool) {
	base := configPrefix + "/" + name + "/"

	desc, err := ops.ReadFile(base + "report_desc")
	if err != nil {
		return nil, false
	}
	length, ok := readLiveUint(ops, base+"report_length")
	if !ok {
		return nil, false
	}
	protocol, _ := readLiveUint(ops, base+"protocol")
	subClass, _ := readLiveUint(ops, base+"subclass")

	hid := &HIDFunction{
		Protocol:     uint8(protocol),
		SubClass:     uint8(subClass),
		ReportLength: uint16(length),
		ReportDesc:   desc,
		DevNodeIndex: position,
	}
	if minor, ok := readLiveMinor(ops, base+"dev"); ok {
		hid.DevNodeIndex = minor
	}
	return hid, true
}

func readLiveUint(ops Ops, rel string) (uint64, bool) {
	data, err := ops.ReadFile(rel)
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 16)
	return value, err == nil
}

// The dev attribute is "major:minor", and the minor is the node number.
func readLiveMinor(ops Ops, rel string) (int, bool) {
	data, err := ops.ReadFile(rel)
	if err != nil {
		return 0, false
	}
	_, minor, ok := strings.Cut(strings.TrimSpace(string(data)), ":")
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(minor)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

// Every ordered group hidLayoutFunctions can build, shortest first: the three
// roles alone, then every ordered pair, then every ordered triple. Fifteen of
// them, which is what makes matching a live descriptor against composed ones
// both cheaper and more exact than a report-descriptor parser would be.
func hidRoleGroups() [][]HIDRole {
	var groups [][]HIDRole
	var walk func(prefix, remaining []HIDRole)
	walk = func(prefix, remaining []HIDRole) {
		if len(prefix) != 0 {
			groups = append(groups, slices.Clone(prefix))
		}
		for i, role := range remaining {
			rest := slices.Concat(remaining[:i:i], remaining[i+1:])
			walk(slices.Concat(prefix, []HIDRole{role}), rest)
		}
	}
	walk(nil, HIDRoles[:])
	slices.SortStableFunc(groups, func(a, b []HIDRole) int { return len(a) - len(b) })
	return groups
}

// Which roles a live interface carries, recovered from the descriptor bytes it
// is presenting. Nothing else on the device can tell a keyboard from a pointer:
// the report descriptor is the whole of the interface contract, and both the
// boot and the standard variant of each role are tried because the subclass a
// composite gave up is not recoverable from anywhere else.
func inferHIDRoles(desc []byte) ([]HIDRole, bool) {
	if len(desc) == 0 {
		return nil, false
	}
	for _, group := range hidRoleGroups() {
		for _, subClass := range [...]uint8{0, 1} {
			composed, _, err := composeHIDReport(group, subClass)
			if err != nil {
				continue
			}
			if bytes.Equal(composed, desc) {
				return slices.Clone(group), true
			}
		}
	}
	return nil, false
}

// The live HID functions with their roles named. A stored profile carrying the
// same descriptor bytes names them outright, which is the only way a descriptor
// this package did not compose - an imported one - keeps its roles at all;
// everything else is matched against the layouts hidLayoutFunctions can build.
// A function that answers to neither is returned with no roles and contributes
// no route, because guessing which node a keyboard is on is the failure mode
// this whole path exists to remove.
func nameHIDRoles(live, stored []Function) []Function {
	named := make([]Function, 0, len(live))
	for _, function := range live {
		if function.Kind != FunctionHID || function.HID == nil {
			continue
		}
		hid := *function.HID
		if roles, ok := storedRoles(stored, hid); ok {
			hid.Roles = roles
		} else if roles, ok := inferHIDRoles(hid.ReportDesc); ok {
			hid.Roles = roles
		} else {
			hid.Roles = nil
		}
		function.HID = &hid
		named = append(named, function)
	}
	return named
}

func storedRoles(stored []Function, live HIDFunction) ([]HIDRole, bool) {
	for _, function := range stored {
		if function.Kind != FunctionHID || function.HID == nil || len(function.HID.Roles) == 0 {
			continue
		}
		if function.HID.ReportLength != live.ReportLength {
			continue
		}
		if bytes.Equal(function.HID.ReportDesc, live.ReportDesc) {
			return slices.Clone(function.HID.Roles), true
		}
	}
	return nil, false
}

// What the gadget has that the profile does not promise, and the other way
// round. Only functions this package can probe through configs/c.1 are
// compared: a camera the profile does not mention cannot be seen there at all,
// and reporting a difference nobody can observe would make every boot reapply.
func compareLayout(profile Profile, live []Function) Divergence {
	divergence := Divergence{Profile: profile.Name}

	want := make(map[string]Function, len(profile.Functions))
	for _, function := range profile.Functions {
		if _, probeable := functionProbeAttr[function.Kind]; probeable {
			want[functionName(function)] = function
		}
	}
	have := make(map[string]Function, len(live))
	for _, function := range live {
		have[functionName(function)] = function
	}

	for _, function := range profile.Functions {
		name := functionName(function)
		if _, probeable := functionProbeAttr[function.Kind]; !probeable {
			continue
		}
		if _, present := have[name]; !present {
			divergence.Missing = append(divergence.Missing, name)
		}
	}
	for _, function := range live {
		name := functionName(function)
		promised, ok := want[name]
		if !ok {
			divergence.Extra = append(divergence.Extra, name)
			continue
		}
		if function.Kind != FunctionHID || function.HID == nil || promised.HID == nil {
			continue
		}
		if !sameHIDShape(*promised.HID, *function.HID) {
			divergence.Changed = append(divergence.Changed, name)
		}
	}
	return divergence
}

// What the attached host reads off the wire, and nothing else. wakeup_on_write
// is kernel-side behaviour with no descriptor of its own and the compiler only
// ever turns it on, so a difference there is not something an apply would fix
// and is not worth an unbind to discover.
func sameHIDShape(want, live HIDFunction) bool {
	return want.Protocol == live.Protocol &&
		want.SubClass == live.SubClass &&
		want.ReportLength == live.ReportLength &&
		bytes.Equal(want.ReportDesc, live.ReportDesc)
}

// The profile the store calls active, or the zero profile when it names none.
func (m *Manager) activeProfile() (Profile, error) {
	name, err := m.store.Active()
	if err != nil {
		return Profile{}, fmt.Errorf("read active profile: %w", err)
	}
	if name == "" {
		return Profile{}, nil
	}
	profile, err := m.store.LoadProfile(name)
	if err != nil {
		return Profile{}, fmt.Errorf("load active profile %s: %w", name, err)
	}
	if profile.Name == "" {
		return Profile{}, fmt.Errorf("%w: %s", ErrUnknownProfile, name)
	}
	profile.Normalize()
	return profile, nil
}

func (m *Manager) divergence(profile Profile) Divergence {
	return compareLayout(profile, liveFunctions(m.ops, profile.Functions))
}

// ReconcileGadget makes the hardware match the profile the store promises.
//
// /etc/init.d/S03usbdev rebuilds the stock three-HID gadget from scratch on
// every boot and knows nothing about the profile store, while Migrate returns
// early the moment an active profile exists. Between them, a layout the
// operator chose and an apply that rolled back leave the store promising one
// gadget and the kernel presenting another - permanently, silently, and with
// the server writing HID reports shaped for the promise into endpoints that
// belong to the other one. This is what closes that, and it is what makes a
// collapsed HID layout survive a reboot at all.
//
// It is not a health check with an opinion: a reconcile that cannot land the
// profile reports it, and hidRoutes then follows the gadget that survived
// rather than the profile that did not.
//
// It never binds. S03usbdev deliberately leaves the UDC unbound so the host
// enumerates once, with the layout the operator actually saved, rather than
// once for the stock gadget and again fifteen seconds later for the real one.
// Windows caches an interface-to-driver mapping per device instance, and the
// second enumeration hands it a map that does not match the first - which is
// how the camera, the NIC and the audio functions ended up at
// CM_PROB_FAILED_START, moving between them from boot to boot depending on
// which driver was mid-start. The same rule holds inside the process: the
// reconcile runs first thing, before the vision library, the HID writers and
// the media pipeline exist, and a bind here followed by a pull-up cycle once
// they did was measured as two enumerations 0.7 s apart at every start. So the
// apply below runs with its bind deferred, every rollback rung included, and
// Attach makes the one bind at the end of start. The init script's watchdog
// is the backstop if this process never gets that far at all.
func (m *Manager) ReconcileGadget(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	if m.transientUp.Load() {
		return ErrTransient
	}
	// What the host had before anything here touched the controller, for
	// Attach's wait: a host that was configured and does not come back after
	// the reapply and the bind is the deactivated gadget confirmEnumeration
	// describes.
	m.startUDC = m.udcStatus()

	profile, err := m.activeProfile()
	if err != nil {
		return err
	}
	// Nothing has claimed the gadget yet, so there is no promise to keep and
	// whatever the init script built stands.
	if profile.Name == "" {
		return nil
	}
	// A hybrid is a transient with a process on the other end of ep0.
	// Reasserting one from a boot path would build a gadget nobody is serving.
	for _, function := range profile.Functions {
		if function.Kind == FunctionFFS {
			return nil
		}
	}

	divergence := m.divergence(profile)
	if divergence.Empty() {
		return nil
	}

	log.Warnf("usb gadget diverges from active profile %s: it %s; reasserting the profile", profile.Name, divergence)
	m.deferBind = true
	defer func() { m.deferBind = false }()
	if err := m.ApplyProfile(ctx, profile); err != nil {
		return fmt.Errorf("reassert %s: %w", profile.Name, err)
	}
	return nil
}

// Attach is the one bind of a start, and the last thing the start does to the
// gadget. Everything that binds at run time - an apply, a LUN change, a PHY
// reset, a reclaim after passthrough - binds for itself and is untouched by
// this; a start reaches here with the profile reconciled, the HID writers and
// the media observer wired, and the controller unbound, whether the init
// script left it so or Detach did on the way out of the previous process. One
// bind then puts a finished device on the bus, and a host that asks for a
// descriptor gets an answer, which is what the pull-up cycle this replaces
// bought at the cost of a second enumeration.
//
// The bind runs inside the HID quiesce bracket, as ReclaimUDC's does, so the
// /dev/hidgN nodes f_hid creates at bind are opened with the routes pushed,
// and the media observer is refreshed after it, so the camera's video node -
// which exists only while the function is bound - is held before the host
// asks for it. A controller that is already bound is left alone: the host is
// enumerated against the gadget that is up, and this is not the place to take
// it away.
func (m *Manager) Attach(ctx context.Context) error {
	if err := m.ready(); err != nil {
		return err
	}
	bound := false
	err := m.withGadgetLock(func() error {
		var bindErr error
		bound, bindErr = m.bindIfUnbound()
		return bindErr
	})
	refreshErr := m.refreshObserver(ctx)
	if err != nil {
		return err
	}
	if bound {
		m.confirmEnumeration(ctx, m.startUDC)
	}
	return refreshErr
}

// The bind Attach makes, and whether it made one.
func (m *Manager) bindIfUnbound() (bool, error) {
	if data, err := m.ops.ReadFile(udcAttr); err == nil && strings.TrimSpace(string(data)) != "" {
		log.Infof("the gadget is already bound to %s; leaving it on the bus", strings.TrimSpace(string(data)))
		return false, nil
	}
	available, err := m.ops.ListUDC()
	if err != nil {
		return false, fmt.Errorf("list udc: %w", err)
	}
	if len(available) == 0 {
		return false, fmt.Errorf("no udc to bind")
	}
	log.Infof("binding the gadget to %s: the init script left the bind to us", available[0])
	return true, m.ensureBound(available[0])
}

// f_uvc calls usb_function_deactivate() when the last V4L2 handle closes, and
// __composite_unbind frees the composite without ever making the matching
// usb_gadget_activate() call, so gadget->deactivated survives a UDC unbind and
// rebind and the pullup never comes back on. A gadget in that state binds
// cleanly, reports the controller in its UDC attribute, and never enumerates.
// The UDC attribute cannot tell the two apart; the controller's own state can.
//
// A host that had reached configured before the reapply and does not reach it
// again is that gadget, and all this side can do about it is say so. A
// controller that was not attached to begin with proves nothing, is not waited
// on, and is why a device booting with no cable in it is never delayed here.
func (m *Manager) confirmEnumeration(ctx context.Context, before UDCStatus) {
	if before.State != udcConfigured {
		return
	}

	deadline := time.Now().Add(enumerateTimeout)
	for {
		if m.udcStatus().State == udcConfigured {
			return
		}
		if ctx.Err() != nil || !time.Now().Before(deadline) {
			break
		}
		time.Sleep(enumeratePoll)
	}
	m.requirePowerCycle()
	log.Warnf("usb gadget bound but did not re-enumerate after reasserting the active profile; the attached host needs the cable re-seated or the device power cycled")
}

// The honest semantics for a failed apply, of the two on offer: the store
// records that its active profile is not live, rather than the stored profile
// reverting with the gadget. The revert is what applyPlan's ladder already
// does - every rung that completes writes the profile it restored back over the
// store and moves the active marker with it - so the only case left is the one
// where every rung failed and the controller carries a half-finished
// transaction that matches no profile at all. There is nothing to revert to
// there, and the store would otherwise go on naming a layout the host is not
// enumerating: the same silent lie that cost this device its keyboard.
//
// It is recorded by derivation rather than by a marker on disk. A flag in
// /etc/kvm/presentation would be one more thing that goes stale, and S03usbdev
// rewriting the gadget out from under it on the next boot is exactly how; the
// comparison itself costs a dozen configfs reads and is true at the moment it
// is asked. So the failed apply carries it, and Snapshot re-derives it for
// every reader after that.
func (m *Manager) reportDivergence() error {
	profile, err := m.activeProfile()
	if err != nil || profile.Name == "" {
		return nil
	}
	divergence := m.divergence(profile)
	if divergence.Empty() {
		return nil
	}
	return fmt.Errorf("%w: the store still names %s but the gadget %s", ErrDiverged, profile.Name, divergence)
}

// Which /dev/hidgN each role writes to, and whether its reports carry a report
// ID prefix, is a property of the gadget the kernel bound and never of the
// profile on disk. The two disagree on every boot until ReconcileGadget lands,
// after an apply that rolled back, and for the whole life of a transient
// hybrid. A nine-byte report-ID-multiplexed report written into the stock
// eight-byte boot keyboard is accepted by the endpoint and understood by
// nobody: the host shows HID devices and nothing is controllable, with no error
// anywhere. So the live gadget decides, and the stored profile is consulted
// only to put role names to a descriptor this package did not compose.
//
// The second result is "there is something to say". With no readable gadget and
// no active profile the routes are left alone, so an unwired Hid keeps the
// historical hidg0/hidg1/hidg2 mapping rather than being told every role is
// absent and opening nothing at all.
func (m *Manager) hidRoutes() ([]HIDRoute, bool) {
	var stored []Function
	if profile, err := m.activeProfile(); err != nil {
		log.Debugf("hid routes: %s", err)
	} else {
		stored = profile.Functions
	}

	if m.ready() == nil {
		if routes := HIDRoutes(nameHIDRoles(liveFunctions(m.ops, stored), stored)); len(routes) != 0 {
			return routes, true
		}
	}
	routes := HIDRoutes(stored)
	return routes, len(routes) != 0
}
