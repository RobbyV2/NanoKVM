package bridge

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

const (
	operationEnable  = "enable"
	operationDisable = "disable"

	// restoreWindow bounds a rollback. It is generous because a rollback runs
	// on a device whose network is already wrong, and because S30eth start on
	// the DHCP branch can sit through udhcpc's -t 10 -T 1.
	restoreWindow = 2 * time.Minute
)

// The presentation manager's half of step 13. It may be nil: the twelve steps
// that hold the management address must work on a device with no gadget NIC.
// NIC returns "" when the active profile has none.
type Gadget interface {
	NIC(ctx context.Context) (string, error)
}

func (m *Manager) lock() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inflight {
		return ErrBusy
	}
	m.inflight = true
	return nil
}

func (m *Manager) unlock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inflight = false
}

// Steps 1 and 2. The snapshot is fsynced before this returns, so from step 3's
// first change there is a complete record on disk, and the watchdog is running
// before that change. The order is load-bearing: a marker naming a snapshot not
// yet written is one a boot-time check cannot act on.
func (m *Manager) begin(ctx context.Context, operation string) (*Snapshot, *deadman, error) {
	// Step 1. Snapshot the full network state to disk.
	snapshot, err := Capture(ctx, m.ip)
	if err != nil {
		return nil, nil, fmt.Errorf("snapshot: %w", err)
	}

	snapshotPath, err := m.store.WriteSnapshot(snapshot)
	if err != nil {
		return nil, nil, fmt.Errorf("write snapshot: %w", err)
	}

	// Step 2. Arm the dead-man.
	dm, err := m.arm(operation, snapshotPath)
	if err != nil {
		return nil, nil, err
	}

	return snapshot, dm, nil
}

// Steps 1 through 12 hold the management address and are all inside the armed
// window. Step 13 runs after the disarm: enslaving usb0 is reversible and never
// touches the address the caller is talking to.
func (m *Manager) Enable(ctx context.Context) (proto.SetBridgeRsp, error) {
	if err := m.lock(); err != nil {
		return proto.SetBridgeRsp{State: proto.BridgePending, Message: err.Error()}, err
	}
	defer m.unlock()

	snapshot, dm, err := m.begin(ctx, operationEnable)
	if err != nil {
		return proto.SetBridgeRsp{State: proto.BridgeDisabled, Uplink: ReadUplink(), Message: err.Error()}, err
	}
	armedAt := m.now()

	// Step 3. Create br0 with STP off and its MAC pinned to eth0's permanent
	// address, then bring it up.
	//
	// The MAC is read before anything is enslaved. Once eth0 is a port the
	// bridge has already elected an address of its own, and reading then would
	// pin whatever the election produced rather than the identity the network's
	// DHCP reservation was made against.
	mac, err := permanentMAC(StockUplink)
	if err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	if err := m.ip.AddBridge(ctx, BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	if err := m.ip.SetMAC(ctx, BridgeName, mac); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	if err := m.ip.SetUp(ctx, BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 4. Kill any running udhcpc, so no lease renewal re-adds an address
	// to a device that is about to become a port.
	if err := killUdhcpc(); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 5. eth0's addresses and default route were captured in step 1;
	// take them off the device now.
	if err := m.ip.FlushAddr(ctx, StockUplink); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	// A device with no default route makes this fail, and that is the state the
	// step exists to produce, so the error is not one.
	_ = m.ip.DeleteDefaultRoute(ctx, StockUplink)

	// Step 6. Enslave eth0 and bring it up. SetMaster is the only path to
	// "ip link set ... master" and its device set never contains wlan0.
	if err := m.ip.SetMaster(ctx, StockUplink, BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	if err := m.ip.SetUp(ctx, StockUplink); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 7. Publish the uplink name, which every later step and every other
	// component reads through.
	if err := WriteUplink(BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 8. Address br0, replaying eth0's captured address on the static path
	// and letting S30eth run udhcpc on the DHCP one.
	if err := m.address(ctx, BridgeName, StockUplink, snapshot); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 9. Publish the resulting gateway for the unpatched kvm_system.
	m.publishGateway(ctx, BridgeName)

	// Step 10. Re-install the five firewall rules against br0 and delete the
	// five naming eth0; the boot-time "-C || -A" block never removes anything.
	if err := m.firewall.Install(ctx, BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	_ = m.firewall.Remove(ctx, StockUplink)

	// Step 11. Verify. All three gates, or nothing is disarmed.
	checks, reason := m.verify(ctx, BridgeName, armedAt)
	if !checks.Passed() {
		return m.rollback(ctx, dm, snapshot, checks, reason)
	}

	// Step 12. Disarm, recording the outcome before removing the marker.
	rsp, err := m.commit(dm, true, BridgeName, checks)
	if err != nil || rsp.State != proto.BridgeEnabled {
		return rsp, err
	}

	// Step 13. Enslave usb0, outside the armed window. Its worst case is an
	// attached host with no network, not a device with no management plane, so
	// a failure here is logged and reported rather than rolled back onto the
	// address the caller is using.
	if err := m.enslaveGadget(ctx); err != nil {
		rsp.Message = fmt.Sprintf("bridge enabled; gadget NIC not enslaved: %s", err)
	}
	return rsp, nil
}

// Step 13 tears the device down after the disarm, so a failed verification
// restores onto a bridge that still exists.
func (m *Manager) Disable(ctx context.Context) (proto.SetBridgeRsp, error) {
	if err := m.lock(); err != nil {
		return proto.SetBridgeRsp{State: proto.BridgePending, Message: err.Error()}, err
	}
	defer m.unlock()

	// Step 1 and step 2, exactly as in enable.
	snapshot, dm, err := m.begin(ctx, operationDisable)
	if err != nil {
		return proto.SetBridgeRsp{State: proto.BridgeEnabled, Uplink: ReadUplink(), Message: err.Error()}, err
	}
	armedAt := m.now()

	// Step 3. Release usb0 first, leaving the gadget function itself as the
	// profile has it: an unbridged usb0 is the stock state.
	if _, enslaved := snapshot.Master(GadgetName); enslaved {
		if err := m.ip.SetNoMaster(ctx, GadgetName); err != nil {
			return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
		}
	}

	// Step 4. Kill any running udhcpc and remove the pidfile.
	if err := killUdhcpc(); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 5. br0's addresses and default route were captured in step 1;
	// take them off the bridge now.
	if err := m.ip.FlushAddr(ctx, BridgeName); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	_ = m.ip.DeleteDefaultRoute(ctx, BridgeName)

	// Step 6. Release eth0 and bring it up.
	if err := m.ip.SetNoMaster(ctx, StockUplink); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	if err := m.ip.SetUp(ctx, StockUplink); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 7. Remove the uplink file, so the absent-file fallback returns every
	// reader to eth0 and the on-disk state is byte-identical to a device that
	// never had the bridge.
	if err := RemoveUplink(); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 8. Address eth0, by the same static replay or S30eth split.
	if err := m.address(ctx, StockUplink, BridgeName, snapshot); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}

	// Step 9. Rewrite the gateway file.
	m.publishGateway(ctx, StockUplink)

	// Step 10. Re-install the five rules against eth0, delete the five naming
	// br0.
	if err := m.firewall.Install(ctx, StockUplink); err != nil {
		return m.rollback(ctx, dm, snapshot, proto.BridgeChecks{}, err.Error())
	}
	_ = m.firewall.Remove(ctx, BridgeName)

	// Step 11. Verify the same three checks against eth0.
	checks, reason := m.verify(ctx, StockUplink, armedAt)
	if !checks.Passed() {
		return m.rollback(ctx, dm, snapshot, checks, reason)
	}

	// Step 12. Disarm.
	rsp, err := m.commit(dm, false, StockUplink, checks)
	if err != nil || rsp.State != proto.BridgeDisabled {
		return rsp, err
	}

	// Step 13. Tear the device down, after the disarm.
	if err := m.ip.SetDown(ctx, BridgeName); err != nil {
		log.Warnf("bridge: %s down: %s", BridgeName, err)
	}
	if err := m.ip.DeleteLink(ctx, BridgeName); err != nil {
		log.Warnf("bridge: delete %s: %s", BridgeName, err)
	}
	return rsp, nil
}

// Step 11. The three gates are not redundant: an address proves the device is
// configured, a gateway that answers proves the network layer works, and
// neither proves a client can reach the management plane. S95nanokvm's
// DROP OUTPUT tcp --sport 8000 exists to make a reachable host refuse to answer
// on a port. The first failure returns, so checks names the gate that failed.
func (m *Manager) verify(ctx context.Context, uplink string, since time.Time) (proto.BridgeChecks, string) {
	var checks proto.BridgeChecks

	// Gate 1. An IPv4 address on the uplink, via ip -4 rather than a grep that
	// also matches inet6.
	addrs, err := m.ip.AddrsV4(ctx, uplink)
	if err != nil {
		return checks, fmt.Sprintf("cannot read addresses on %s: %s", uplink, err)
	}
	if len(addrs) == 0 {
		return checks, fmt.Sprintf("no IPv4 address on %s", uplink)
	}
	checks.Address = true
	address := addrs[0].Local

	// Gate 2. A default route through the uplink whose gateway answers.
	routes, err := m.ip.Routes(ctx)
	if err != nil {
		return checks, fmt.Sprintf("cannot read routes: %s", err)
	}

	gateway := ""
	for _, route := range routes {
		if route.Dst == "default" && route.Dev == uplink && route.Gateway != "" {
			gateway = route.Gateway
			break
		}
	}
	if gateway == "" {
		return checks, fmt.Sprintf("no default route through %s", uplink)
	}
	if !m.pinger.Ping(ctx, uplink, gateway) {
		return checks, fmt.Sprintf("gateway %s does not answer on %s", gateway, uplink)
	}
	checks.Gateway = true

	// Gate 3. Inbound liveness: the management plane actually answered.
	if m.live.Observed(address, since) {
		checks.Inbound = true
		return checks, ""
	}

	// The fallback proves the listener and the local delivery path rather than
	// the wire, and is recorded as the weaker of the two so an operator reading
	// the result knows which one they got.
	if err := m.live.SelfConnect(ctx, address); err != nil {
		return checks, fmt.Sprintf("management plane not reachable at %s: %s", address, err)
	}
	checks.Inbound, checks.InboundWeak = true, true
	return checks, ""
}

// Step 12. The outcome is durable before the marker is removed and before the
// response is written, because the caller losing its connection partway through
// is the expected failure mode: it reads the result back from GET.
func (m *Manager) commit(dm *deadman, enabled bool, uplink string, checks proto.BridgeChecks) (proto.SetBridgeRsp, error) {
	if !dm.disarm() {
		// The deadline fired first. The watchdog has already restored and
		// recorded its own outcome; overwriting it here would claim a success
		// that was undone.
		return proto.SetBridgeRsp{
			State:   proto.BridgeRolledBack,
			Uplink:  ReadUplink(),
			Checks:  checks,
			Message: "dead-man deadline expired before verification completed",
		}, nil
	}

	state := proto.BridgeDisabled
	if enabled {
		state = proto.BridgeEnabled
	}

	lkg := LastKnownGood{
		Enabled:   enabled,
		Uplink:    uplink,
		State:     state,
		Checks:    checks,
		AppliedAt: m.now().UTC(),
	}
	if err := m.store.Commit(lkg); err != nil {
		return proto.SetBridgeRsp{State: state, Uplink: uplink, Checks: checks, Message: err.Error()}, err
	}

	return proto.SetBridgeRsp{State: state, Uplink: uplink, Checks: checks}, nil
}

// The single failure exit for both transactions.
func (m *Manager) rollback(
	ctx context.Context,
	dm *deadman,
	snapshot *Snapshot,
	checks proto.BridgeChecks,
	reason string,
) (proto.SetBridgeRsp, error) {
	uplink := snapshot.UplinkName()

	if !dm.disarm() {
		// The deadline beat this path to the restore. disarm waited for the
		// watchdog to finish, so the device is already back and recorded.
		return proto.SetBridgeRsp{
			State:   proto.BridgeRolledBack,
			Uplink:  uplink,
			Checks:  checks,
			Message: "dead-man deadline expired: " + reason,
		}, nil
	}

	// WithoutCancel, because the context that reached here is very often the
	// one that just expired, and a restore that inherits a cancelled context
	// restores nothing at all.
	restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restoreWindow)
	defer cancel()

	log.Warnf("bridge: rolling back: %s", reason)

	state := proto.BridgeRolledBack
	message := reason
	if err := m.Restore(restoreCtx, snapshot); err != nil {
		state = proto.BridgeFailed
		message = fmt.Sprintf("%s; rollback incomplete: %s", reason, err)
		log.Errorf("bridge: %s", message)
	}

	lkg := LastKnownGood{
		Enabled:   uplink == BridgeName,
		Uplink:    uplink,
		State:     state,
		Checks:    checks,
		Message:   message,
		AppliedAt: m.now().UTC(),
	}
	if err := m.store.Commit(lkg); err != nil {
		log.Errorf("bridge: record rollback: %s", err)
	}

	return proto.SetBridgeRsp{State: state, Uplink: uplink, Checks: checks, Message: message}, nil
}

// Step 8. The split is not an optimisation: S30eth:55 appends a nameserver line
// rather than replacing one, so re-running the script on a static device
// accumulates a duplicate on every apply. The static path therefore replays the
// captured address and route in one ip -batch. The DHCP path has no such
// problem and re-runs the script, which reads the uplink file step 7 wrote.
func (m *Manager) address(ctx context.Context, to, from string, snapshot *Snapshot) error {
	if snapshot.StaticPath && len(snapshot.IPv4(from)) > 0 {
		if err := m.replay(ctx, to, snapshotMovedTo(snapshot, from, to)); err != nil {
			return err
		}
		return m.ensureNameserver(snapshot, to)
	}
	return m.scripts.StartEth(ctx)
}

// Relabels the captured addresses and default route onto another device, so
// replay is shared between the capture-side device and the target.
func snapshotMovedTo(snapshot *Snapshot, from, to string) *Snapshot {
	moved := &Snapshot{}

	for _, info := range snapshot.IPv4(from) {
		moved.Addrs = append(moved.Addrs, Addr{Name: to, AddrInfo: []AddrInfo{info}})
	}
	if route, ok := snapshot.DefaultRouteVia(from); ok {
		route.Dev = to
		moved.Routes = append(moved.Routes, route)
	} else if route, ok := snapshot.DefaultRoute(); ok {
		route.Dev = to
		moved.Routes = append(moved.Routes, route)
	}
	return moved
}

// Re-adds the captured nameserver only when it is genuinely missing, which is
// what keeps this idempotent where S30eth:55 is not.
func (m *Manager) ensureNameserver(snapshot *Snapshot, dev string) error {
	gateway := m.gatewayOf(snapshot, dev)
	if gateway == "" || !snapshot.HasNameserver(gateway) {
		return nil
	}

	current, _ := readFile(resolvPath)
	line := "nameserver " + gateway
	for _, existing := range strings.Split(current, "\n") {
		if strings.TrimSpace(existing) == line {
			return nil
		}
	}

	if current != "" && !strings.HasSuffix(current, "\n") {
		current += "\n"
	}
	return utils.WriteFileAtomic(resolvPath, []byte(current+line+"\n"), 0o644)
}

func (m *Manager) gatewayOf(snapshot *Snapshot, dev string) string {
	if route, ok := snapshot.DefaultRouteVia(dev); ok {
		return route.Gateway
	}
	if route, ok := snapshot.DefaultRoute(); ok {
		return route.Gateway
	}
	return ""
}

// Step 9. A failure is logged rather than rolled back: the gateway file is an
// escape hatch for a display, not part of the management path.
func (m *Manager) publishGateway(ctx context.Context, uplink string) {
	routes, err := m.ip.Routes(ctx)
	if err != nil {
		log.Warnf("bridge: read routes for gateway file: %s", err)
		return
	}

	for _, route := range routes {
		if route.Dst == "default" && route.Dev == uplink && isIPv4(route.Gateway) {
			if err := WriteGateway(route.Gateway); err != nil {
				log.Warnf("bridge: write %s: %s", gatewayPath, err)
			}
			return
		}
	}
}

// Step 13 of enable, outside the armed window and a no-op with no gadget NIC.
func (m *Manager) enslaveGadget(ctx context.Context) error {
	if m.gadget == nil {
		return nil
	}

	nic, err := m.gadget.NIC(ctx)
	if err != nil {
		return err
	}
	if nic == "" {
		return nil
	}

	// A profile that somehow named wlan0 is refused rather than trusted because
	// it came from the presentation manager.
	if err := checkEnslavable(nic); err != nil {
		return err
	}
	if err := m.ip.SetMaster(ctx, nic, BridgeName); err != nil {
		return err
	}

	// Its own small rollback: an enslaved port that never forwards is a silent
	// black hole for the host's frames, where the stock state fails visibly.
	if err := m.ip.SetUp(ctx, nic); err != nil {
		if release := m.ip.SetNoMaster(ctx, nic); release != nil {
			log.Errorf("bridge: %s left enslaved and down: %s", nic, release)
		}
		return err
	}
	return nil
}

// Reads the live device rather than the record, so an operator comparing the
// two can see a device that drifted from it.
func (m *Manager) Status(ctx context.Context) (proto.GetBridgeRsp, error) {
	rsp := proto.GetBridgeRsp{
		State:  proto.BridgeDisabled,
		Uplink: ReadUplink(),
	}

	if lkg, err := m.store.LastKnownGood(); err == nil && lkg != nil {
		rsp.LastApply = &proto.BridgeApply{
			State:     lkg.State,
			Uplink:    lkg.Uplink,
			Enabled:   lkg.Enabled,
			Checks:    lkg.Checks,
			Message:   lkg.Message,
			AppliedAt: lkg.AppliedAt,
		}
		rsp.State = lkg.State
	}

	if pending, err := m.store.Pending(); err == nil && pending != nil {
		rsp.State = proto.BridgePending
		rsp.Pending = &proto.BridgeArmed{
			Operation:    pending.Operation,
			SnapshotPath: pending.SnapshotPath,
			ArmedAt:      pending.ArmedAt,
			Deadline:     pending.Deadline,
		}
	}

	links, err := m.ip.Links(ctx)
	if err != nil {
		return rsp, err
	}

	for _, link := range links {
		switch {
		case link.Name == BridgeName:
			rsp.Exists = true
			rsp.MAC = link.Address
		case link.Master == BridgeName:
			rsp.Ports = append(rsp.Ports, proto.BridgePort{
				Name:  link.Name,
				State: link.OperState,
				Up:    link.Up(),
			})
		}
	}

	if rsp.Exists {
		if addrs, err := m.ip.AddrsV4(ctx, BridgeName); err == nil && len(addrs) > 0 {
			rsp.Address = addrs[0].CIDR()
		}
	}

	if routes, err := m.ip.Routes(ctx); err == nil {
		for _, route := range routes {
			if route.Dst == "default" && route.Dev == rsp.Uplink {
				rsp.Gateway = route.Gateway
				break
			}
		}
	}

	return rsp, nil
}

// For the case where all three gates passed and the operator still cannot reach
// the device from somewhere verification did not test. Reached over the wlan0
// AP or a serial console.
func (m *Manager) Revert(ctx context.Context) error {
	if err := m.lock(); err != nil {
		return err
	}
	defer m.unlock()

	path := m.store.SnapshotPath()
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%w: %s", ErrNoSnapshot, path)
	}

	return m.restoreFrom(ctx, Pending{Operation: "revert", SnapshotPath: path, ArmedAt: m.now()})
}
