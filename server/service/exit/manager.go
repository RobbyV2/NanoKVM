package exit

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

// The dataplane components the manager owns per slot, as the small interfaces
// this file needs. The constructors in socks.go, mux.go, wsproxy.go, dns.go and
// probe.go satisfy them; newComponents is where they are wired together, and
// tests point it at fakes.
type frontDoor interface {
	Start() error
	Stop()
	SetNative(b Backend)
	SetRelay(gate RelayGate)
	Attached() (state proto.ExitTunnelState, peer *proto.ExitPeer, since *time.Time)
}

type nativeMux interface {
	ServeNative(w http.ResponseWriter, r *http.Request)
	Current() Backend
	CloseAll(reason string)
	SetPinPeer(pin bool)
}

type relayProxy interface {
	RelayGate
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	CloseAll(reason string)
	SetPinPeer(pin bool)
}

type dnsForwarder interface {
	Bind(addr netip.Addr) error
	Stop()
	Bound() bool
	Stats() proto.ExitDNSStats
}

type reachProber interface {
	Start()
	Stop()
	Result() proto.ExitUpstream
}

// componentDeps is what the manager gives the factory: closures that read the
// slot's live config and report events back, so the components never hold a
// Config of their own.
type componentDeps struct {
	Policy       func() Policy
	Bytes        ByteCounter
	OnSession    func(Backend)
	OnClose      func(Backend, string)
	OnPeerChange func(prev, next proto.ExitPeer)
	Upstreams    func() []netip.Addr
	Connected    func() bool
}

type components struct {
	Door  frontDoor
	Mux   nativeMux
	Proxy relayProxy
	DNS   dnsForwarder
	Probe reachProber
}

// Deps are the manager's external dependencies, every one substitutable.
// Components defaults to newComponents (wire.go), the real dataplane.
type Deps struct {
	NIC        NICResolver
	Rebind     Rebinder
	Subscribe  func(func(context.Context))
	Bridge     BridgeGate
	Runner     Runner
	Now        func() time.Time
	Components func(Slot, componentDeps) components
	NICInfo    func(ifname string) (up bool, addr netip.Prefix)
	Limiter    *Limiter
}

type slotState struct {
	slot  Slot
	cfg   Config
	st    State
	comps *components
	bytes *Counter

	nic           proto.ExitNIC
	gw            netip.Addr
	down          DownstreamStatus
	dnsRedirected uint64
	message       string

	lastPeer     *proto.ExitPeer
	peerSource   string
	wasConnected bool
}

// Manager owns every slot's runtime: the enable and disable transactions
// (D23), the watchdog converge (D8), the rebind subscription (D25), the bridge
// gate (D17) and the token gate in front of the dataplane (D10).
type Manager struct {
	deps    Deps
	down    downstream
	limiter *Limiter

	// mu guards slots and every slotState field. opMu serialises the
	// transactions (enable, disable, config, token). convergeMu serialises
	// the converge paths (attach, rebind, watchdog). A converge also takes
	// opMu, but only by TryLock: one that arrives while a transaction runs
	// simply skips, since S94exit start does not check ENABLED and a tick
	// that had passed the enabled check could otherwise queue its start
	// behind Disable's stop and bring the slot back. A transaction's own
	// Rebind fires onRebind on this goroutine, which is why it must not
	// block: Enable's closing convergeSlot covers that rebind itself.
	mu         sync.Mutex
	opMu       sync.Mutex
	convergeMu sync.Mutex
	slots      map[string]*slotState

	initOnce sync.Once
	stop     chan struct{}
	stopOnce sync.Once
}

var (
	managerOnce    sync.Once
	defaultManager *Manager
)

// ErrUnknownSlot is returned for a slot id that has no config.
var ErrUnknownSlot = errors.New("unknown exit slot")

// presentationNIC adapts presentation.Manager to NICResolver: its UDCBound
// returns an error alongside the bool, and an unreadable gadget is unbound.
type presentationNIC struct{ *presentation.Manager }

func (p presentationNIC) UDCBound() bool {
	bound, err := p.Manager.UDCBound()
	return err == nil && bound
}

// fileBridgeGate is D17's evidence, read from the bridge's own files so this
// package does not import the bridge (which imports this one for AnyEnabled).
// The paths are the bridge package's StateDir, UplinkPath, SysClassNetDir and
// RNDISNoDHCPDPath; the pattern is the one S29bridge greps.
type fileBridgeGate struct{}

var bridgeEnabledPattern = regexp.MustCompile(`"enabled"\s*:\s*true`)

func (fileBridgeGate) BridgeActive() (bool, string) {
	if data, err := os.ReadFile("/etc/kvm/presentation/network/last-known-good.json"); err == nil &&
		bridgeEnabledPattern.Match(data) {
		return true, "the L2 bridge is enabled (last-known-good)"
	}
	if data, err := os.ReadFile("/etc/kvm/network/l2-uplink"); err == nil && strings.TrimSpace(string(data)) == "br0" {
		return true, "the management uplink is br0"
	}
	if _, err := os.Stat("/sys/class/net/br0"); err == nil {
		return true, "br0 exists"
	}
	if _, err := os.Stat("/boot/rndis.nodhcpd"); err == nil {
		return true, "/boot/rndis.nodhcpd is set"
	}
	return false, ""
}

// GetManager builds the singleton with the real dependencies and initialises
// it. Callable before the routes exist, which is what the post-attach hook
// needs.
func GetManager() *Manager {
	managerOnce.Do(func() {
		pm := presentation.GetManager()
		defaultManager = NewManager(Deps{
			NIC:       presentationNIC{pm},
			Rebind:    pm,
			Subscribe: pm.OnRebind,
			Bridge:    fileBridgeGate{},
			Runner:    execRunner{},
		})
		defaultManager.Init()
	})
	return defaultManager
}

// NewManager builds a manager without initialising it. Zero deps take the
// real implementation.
func NewManager(deps Deps) *Manager {
	if deps.Runner == nil {
		deps.Runner = execRunner{}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Components == nil {
		deps.Components = newComponents
	}
	if deps.NICInfo == nil {
		deps.NICInfo = readNICInfo
	}
	if deps.Limiter == nil {
		deps.Limiter = NewLimiter()
	}
	if deps.Bridge == nil {
		deps.Bridge = fileBridgeGate{}
	}
	return &Manager{
		deps:    deps,
		down:    downstream{run: deps.Runner},
		limiter: deps.Limiter,
		slots:   make(map[string]*slotState),
		stop:    make(chan struct{}),
	}
}

// readNICInfo is the real NICInfo: link state and the first 10.x address.
func readNICInfo(ifname string) (bool, netip.Prefix) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return false, netip.Prefix{}
	}
	up := iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0
	addrs, err := iface.Addrs()
	if err != nil {
		return up, netip.Prefix{}
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil || ip4[0] != 10 {
			continue
		}
		ones, _ := ipnet.Mask.Size()
		return up, netip.PrefixFrom(netip.AddrFrom4([4]byte(ip4)), ones)
	}
	return up, netip.Prefix{}
}

// Init installs the scripts, loads the slots (creating slot 0 disabled with a
// fresh token when absent), disarms anything left pending, starts every
// enabled slot's runtime, subscribes to rebinds and starts the watchdog.
func (m *Manager) Init() {
	m.initOnce.Do(func() {
		ctx := context.Background()
		if err := installScripts(); err != nil {
			log.Warnf("exit: install init scripts: %s", err)
		}

		cfgs, err := LoadAll()
		if err != nil {
			log.Errorf("exit: load slot configs: %s", err)
		}
		if !hasSlot(cfgs, "0") {
			cfg, err := DefaultConfig(MustSlot("0"))
			if err != nil {
				log.Errorf("exit: create slot 0: %s", err)
			} else if err := m.persist(cfg, ""); err != nil {
				log.Errorf("exit: write slot 0: %s", err)
			} else {
				cfgs = append(cfgs, cfg)
			}
		}

		for _, cfg := range cfgs {
			slot := MustSlot(cfg.Slot)
			st, err := LoadState(slot)
			if err != nil {
				log.Warnf("exit: slot %s state: %s", slot.ID, err)
			}
			s := &slotState{slot: slot, cfg: cfg, st: st, bytes: NewCounter()}
			s.lastPeer = st.PreviousPeer
			if cfg.Pending {
				// A transaction that never finished: converge to off (D18).
				log.Warnf("exit: slot %s was left pending; disabling", slot.ID)
				s.cfg.Enabled, s.cfg.Pending = false, false
				_ = RemoveGadgetRoute(slot)
				if err := m.persist(s.cfg, ""); err != nil {
					log.Errorf("exit: slot %s: %s", slot.ID, err)
				}
				if err := m.down.Stop(ctx, slot); err != nil {
					log.Warnf("exit: slot %s: %s", slot.ID, err)
				}
			}
			m.mu.Lock()
			m.slots[slot.ID] = s
			m.mu.Unlock()
		}

		for _, s := range m.snapshot() {
			if !s.cfg.Enabled {
				continue
			}
			if err := m.ensureBinaries(s.cfg); err != nil {
				log.Errorf("exit: slot %s: %s", s.slot.ID, err)
			}
			if err := m.down.Start(ctx, s.slot); err != nil {
				log.Warnf("exit: slot %s: %s", s.slot.ID, err)
			}
			_, gw := m.resolveNIC(ctx, s)
			if err := m.startComponents(s, gw); err != nil {
				log.Errorf("exit: slot %s: %s", s.slot.ID, err)
				m.setMessage(s, err.Error())
			}
			m.refreshDownstream(ctx, s)
		}

		if m.deps.Subscribe != nil {
			m.deps.Subscribe(m.onRebind)
		}
		go m.watchdog()
	})
}

// Close stops the watchdog, for tests.
func (m *Manager) Close() {
	m.stopOnce.Do(func() { close(m.stop) })
}

func hasSlot(cfgs []Config, id string) bool {
	for _, cfg := range cfgs {
		if cfg.Slot == id {
			return true
		}
	}
	return false
}

func (m *Manager) snapshot() []*slotState {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*slotState, 0, len(m.slots))
	for _, s := range m.slots {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].slot.ID < out[j].slot.ID })
	return out
}

func (m *Manager) get(slot Slot) (*slotState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[slot.ID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSlot, slot.ID)
	}
	return s, nil
}

func (m *Manager) setMessage(s *slotState, msg string) {
	m.mu.Lock()
	s.message = msg
	m.mu.Unlock()
}

// persist writes the config and the three rendered files together, so the
// env S94exit reads never disagrees with the json the manager reads.
func (m *Manager) persist(cfg Config, nic string) error {
	if err := SaveConfig(cfg); err != nil {
		return err
	}
	return writeSlotFiles(MustSlot(cfg.Slot), cfg, nic)
}

func (m *Manager) ensureBinaries(cfg Config) error {
	if _, err := ensureBinary(HevBinary); err != nil {
		return fmt.Errorf("%s: %w", HevBinary, err)
	}
	if cfg.Mode == proto.ExitModeWstunnel {
		if _, err := ensureBinary(WstunnelBinary); err != nil {
			return fmt.Errorf("%s: %w", WstunnelBinary, err)
		}
	}
	return nil
}

// resolveNIC re-reads the gadget netdev (only once the UDC is bound, rejecting
// the unregistered placeholder), records it in <n>/nic, and reads its link
// state and 10.x address. It returns the ifname and the gateway address.
func (m *Manager) resolveNIC(ctx context.Context, s *slotState) (string, netip.Addr) {
	m.mu.Lock()
	nic := s.nic
	m.mu.Unlock()

	if m.deps.NIC != nil && m.deps.NIC.UDCBound() {
		if name, err := m.deps.NIC.NIC(ctx); err == nil && name != "" && !strings.Contains(name, "(") && !strings.Contains(name, "%") {
			nic.Ifname = name
		} else if err == nil && name == "" {
			nic.Ifname = ""
		}
		if protocol, err := m.deps.NIC.NetworkProtocol(ctx); err == nil {
			nic.Protocol = protocol
		}
	}

	var gw netip.Addr
	nic.Up, nic.Address = false, ""
	if nic.Ifname != "" {
		up, prefix := m.deps.NICInfo(nic.Ifname)
		nic.Up = up
		if prefix.IsValid() {
			nic.Address = prefix.String()
			gw = prefix.Addr()
		}
		_ = utils.WriteFileAtomic(s.slot.NicPath(), []byte(nic.Ifname+"\n"), 0o600)
	}

	m.mu.Lock()
	s.nic = nic
	s.gw = gw
	m.mu.Unlock()
	return nic.Ifname, gw
}

// refreshDownstream runs `S94exit status` and records the vector.
func (m *Manager) refreshDownstream(ctx context.Context, s *slotState) {
	report, err := m.down.Status(ctx, s.slot)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		s.down = DownstreamStatus{}
		return
	}
	s.down = report.Down
	s.dnsRedirected = report.DNSRedirected
	if s.comps != nil {
		s.down.DNS = s.comps.DNS.Bound()
	}
}

// startComponents builds and starts the slot's dataplane and binds the DNS
// forwarder to gw when it is known. Bind errors are returned; the front door's
// listen error is fatal to an enable.
func (m *Manager) startComponents(s *slotState, gw netip.Addr) error {
	m.mu.Lock()
	if s.comps != nil {
		m.mu.Unlock()
		return nil
	}
	cfg := s.cfg
	m.mu.Unlock()

	deps := componentDeps{
		Policy:       func() Policy { return m.config(s).Policy() },
		Bytes:        s.bytes,
		OnSession:    func(b Backend) { m.onSession(s, b) },
		OnClose:      func(b Backend, reason string) { m.onClose(s, b, reason) },
		OnPeerChange: func(prev, next proto.ExitPeer) { m.onPeerChange(s, prev, next) },
		Upstreams:    func() []netip.Addr { return m.config(s).Upstreams() },
		Connected:    func() bool { return m.connected(s) },
	}
	c := m.deps.Components(s.slot, deps)
	if err := c.Door.Start(); err != nil {
		return fmt.Errorf("socks front door %s: %w", s.slot.SocksAddr(), err)
	}
	c.Mux.SetPinPeer(cfg.PinPeer)
	c.Proxy.SetPinPeer(cfg.PinPeer)
	if cfg.Mode == proto.ExitModeWstunnel {
		c.Door.SetRelay(c.Proxy)
	}
	var bindErr error
	if gw.IsValid() {
		if err := c.DNS.Bind(gw); err != nil {
			bindErr = fmt.Errorf("dns forwarder on %s:53: %w", gw, err)
		}
	}
	c.Probe.Start()

	m.mu.Lock()
	s.comps = &c
	m.mu.Unlock()
	return bindErr
}

// stopComponents detaches the slot's dataplane under mu and stops it outside,
// since CloseAll may call the hooks back.
func (m *Manager) stopComponents(s *slotState) {
	m.mu.Lock()
	c := s.comps
	s.comps = nil
	m.mu.Unlock()
	if c == nil {
		return
	}
	c.Probe.Stop()
	c.DNS.Stop()
	c.Mux.CloseAll("slot stopped")
	c.Proxy.CloseAll("slot stopped")
	c.Door.SetNative(nil)
	c.Door.SetRelay(nil)
	c.Door.Stop()
}

func (m *Manager) config(s *slotState) Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.cfg
}

// view snapshots the fields a transaction reads before it starts running
// commands, in one critical section.
func (m *Manager) view(s *slotState) (cfg Config, nic string, c *components, gw netip.Addr) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.cfg, s.nic.Ifname, s.comps, s.gw
}

func (m *Manager) connected(s *slotState) bool {
	m.mu.Lock()
	c := s.comps
	m.mu.Unlock()
	if c == nil {
		return false
	}
	state, _, _ := c.Door.Attached()
	return state == proto.ExitConnected
}

// onSession is the mux hook after WELCOME: the front door is pointed at the
// new session and the peer is recorded, with a supersede noted when the
// address changed (D12).
func (m *Manager) onSession(s *slotState, b Backend) {
	peer := b.Peer()
	m.mu.Lock()
	c := s.comps
	m.notePeerLocked(s, peer)
	m.mu.Unlock()
	if c != nil {
		c.Door.SetNative(b)
	}
	m.saveState(s)
}

func (m *Manager) onClose(s *slotState, b Backend, reason string) {
	m.mu.Lock()
	c := s.comps
	m.mu.Unlock()
	if c == nil {
		return
	}
	// Only the current session detaches the door: a superseded session's
	// close must not undo the new session's attach.
	if cur := c.Mux.Current(); cur == nil || cur == b {
		c.Door.SetNative(nil)
	}
	m.markDisconnected(s)
	log.Infof("exit: slot %s session from %s closed: %s", s.slot.ID, b.Peer().Addr, reason)
}

func (m *Manager) onPeerChange(s *slotState, prev, next proto.ExitPeer) {
	m.mu.Lock()
	m.notePeerLocked(s, next)
	m.mu.Unlock()
	m.saveState(s)
}

// notePeerLocked records a connecting peer. Callers hold mu.
func (m *Manager) notePeerLocked(s *slotState, peer proto.ExitPeer) {
	now := m.deps.Now()
	if s.lastPeer != nil && s.lastPeer.Addr != peer.Addr {
		log.Warnf("exit: slot %s exit superseded: %s -> %s", s.slot.ID, s.lastPeer.Addr, peer.Addr)
		prev := *s.lastPeer
		s.st.PreviousPeer = &prev
		s.st.PeerChangedAt = &now
	}
	p := peer
	s.lastPeer = &p
	s.peerSource = SourceKey(peer.Addr)
	s.st.LastConnectedAt = &now
	s.wasConnected = true
}

// markDisconnected persists lastConnectedAt on the connected -> disconnected
// edge, and only then, so a 3 s status poll never writes flash.
func (m *Manager) markDisconnected(s *slotState) {
	m.mu.Lock()
	was := s.wasConnected
	if was {
		now := m.deps.Now()
		s.st.LastConnectedAt = &now
		s.wasConnected = false
	}
	m.mu.Unlock()
	if was {
		m.saveState(s)
	}
}

func (m *Manager) saveState(s *slotState) {
	m.mu.Lock()
	st := s.st
	m.mu.Unlock()
	if err := SaveState(s.slot, st); err != nil {
		log.Warnf("exit: slot %s state: %s", s.slot.ID, err)
	}
}

// detached is the context a transaction runs on: the caller's values, none
// of its cancellation. The handlers pass gin's request context, which dies
// with the client connection (a closed tab, the UI's request timeout), and
// exec.CommandContext then refuses to start anything, so a transaction that
// had already mutated the config or the device could not finish or roll
// back. execRunner's own per-command timeout is the bound instead.
func detached(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

// Enable is the D23 transaction. Every failure after the first mutation runs
// the rollback and returns the failing step's error, which status carries as
// message.
func (m *Manager) Enable(ctx context.Context, slot Slot) error {
	ctx = detached(ctx)
	m.opMu.Lock()
	defer m.opMu.Unlock()

	s, err := m.get(slot)
	if err != nil {
		return err
	}
	if m.config(s).Enabled {
		return nil
	}

	// Refuse-checks: nothing has been written yet.
	if active, why := m.deps.Bridge.BridgeActive(); active {
		return m.fail(s, fmt.Errorf("the gadget NIC belongs to the L2 bridge: %s; disable the bridge first", why))
	}
	if m.deps.NIC == nil {
		return m.fail(s, errors.New("usb presentation unavailable"))
	}
	protocol, err := m.deps.NIC.NetworkProtocol(ctx)
	if err != nil {
		return m.fail(s, fmt.Errorf("read the gadget network function: %w", err))
	}
	if protocol == "" {
		return m.fail(s, errors.New("the USB profile links no network function: turn on the USB Network Adapter under Network"))
	}

	cfg := m.config(s)
	if err := m.ensureBinaries(cfg); err != nil {
		return m.fail(s, err)
	}

	nic, gw := m.resolveNIC(ctx, s)

	// Arm: files first, then the config with pending set (D18, D23).
	cfg.Enabled, cfg.Pending = false, true
	if err := m.persist(cfg, nic); err != nil {
		return m.fail(s, fmt.Errorf("write slot files: %w", err))
	}
	m.mu.Lock()
	s.cfg = cfg
	m.mu.Unlock()

	rollback := func(step string, cause error) error {
		err := fmt.Errorf("%s: %w", step, cause)
		log.Errorf("exit: slot %s enable failed at %s; rolling back", slot.ID, err)
		m.stopComponents(s)
		if stopErr := m.down.Stop(ctx, slot); stopErr != nil {
			log.Warnf("exit: slot %s rollback: %s", slot.ID, stopErr)
		}
		_ = RemoveGadgetRoute(slot)
		cfg := m.config(s)
		cfg.Enabled, cfg.Pending = false, false
		if persistErr := m.persist(cfg, nic); persistErr != nil {
			log.Errorf("exit: slot %s rollback: %s", slot.ID, persistErr)
		}
		m.mu.Lock()
		s.cfg = cfg
		m.mu.Unlock()
		return m.fail(s, err)
	}

	if err := m.down.Start(ctx, slot); err != nil {
		return rollback("S94exit start", err)
	}
	report, err := m.waitForHev(ctx, slot)
	if err != nil {
		return rollback("S94exit status", err)
	}
	if !report.Down.Hev {
		return rollback("S94exit status", fmt.Errorf("%s is not running; see %s", HevBinary, slot.LogPath("hev")))
	}
	if cfg.Mode == proto.ExitModeWstunnel && !report.Down.Wstunnel {
		return rollback("S94exit status", fmt.Errorf("%s server is not running; see %s", WstunnelBinary, slot.LogPath("wstunnel")))
	}

	// The DNS forwarder needs the gadget address; S30rndis puts it there.
	if !gw.IsValid() {
		if err := m.down.StartRNDIS(ctx); err != nil {
			log.Warnf("exit: slot %s: %s", slot.ID, err)
		}
		nic, gw = m.resolveNIC(ctx, s)
	}
	if !gw.IsValid() {
		return rollback("gadget address", fmt.Errorf("the gadget NIC %q has no 10.x address to bind DNS on", nic))
	}

	if err := m.startComponents(s, gw); err != nil {
		return rollback("start listeners", err)
	}

	// Commit.
	if err := WriteGadgetRoute(slot); err != nil {
		return rollback("write gadget.route", err)
	}
	cfg.Enabled, cfg.Pending = true, false
	if err := m.persist(cfg, nic); err != nil {
		return rollback("write config", err)
	}
	m.mu.Lock()
	s.cfg = cfg
	s.message = ""
	m.mu.Unlock()

	if err := m.down.RestartRNDIS(ctx); err != nil {
		return rollback("S30rndis restart", err)
	}
	// One more converge so the DNAT rule sees the (possibly new) address.
	if err := m.down.Start(ctx, slot); err != nil {
		log.Warnf("exit: slot %s: %s", slot.ID, err)
	}

	// Re-lease the consumer, best effort (D5). The rebind fires the D25
	// subscriber on its own; this closing converge covers a rebind that was
	// refused, and takes convergeMu like every other converge so it never
	// runs beside the watchdog's.
	if m.deps.Rebind != nil {
		if err := m.deps.Rebind.Rebind(ctx); err != nil {
			m.setMessage(s, "re-plug the USB cable to pick up the new gateway: "+err.Error())
		}
	}
	m.convergeMu.Lock()
	m.convergeSlot(ctx, s)
	m.convergeMu.Unlock()
	return nil
}

// hevStartBudget is how long an enable waits for hev to show a pid:
// start-stop-daemon returns before hev has opened the tun, and a daemon that
// dies on this CPU is what the wait catches. Wall time, not the injected
// clock, because it paces real polls.
var hevStartBudget = 3 * time.Second

func (m *Manager) waitForHev(ctx context.Context, slot Slot) (statusReport, error) {
	deadline := time.Now().Add(hevStartBudget)
	for {
		report, err := m.down.Status(ctx, slot)
		if err != nil {
			return report, err
		}
		if report.Down.Hev || !time.Now().Before(deadline) {
			return report, nil
		}
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case <-time.After(hevStartBudget / 15):
		}
	}
}

func (m *Manager) fail(s *slotState, err error) error {
	m.setMessage(s, err.Error())
	return err
}

// Disable removes the marker and flips enabled first, so a crash midway
// converges to off, then restarts udhcpd without the options, stops the
// listeners, tears the downstream down and re-leases the consumer (D23).
func (m *Manager) Disable(ctx context.Context, slot Slot) error {
	ctx = detached(ctx)
	m.opMu.Lock()
	defer m.opMu.Unlock()

	s, err := m.get(slot)
	if err != nil {
		return err
	}
	cfg, nic, c, _ := m.view(s)
	if !cfg.Enabled && !cfg.Pending && c == nil {
		return nil
	}

	_ = RemoveGadgetRoute(slot)
	cfg.Enabled, cfg.Pending = false, false
	if err := m.persist(cfg, nic); err != nil {
		return m.fail(s, fmt.Errorf("write config: %w", err))
	}
	m.mu.Lock()
	s.cfg = cfg
	s.message = ""
	m.mu.Unlock()

	var firstErr error
	if err := m.down.RestartRNDIS(ctx); err != nil {
		firstErr = err
	}
	m.stopComponents(s)
	if err := m.down.Stop(ctx, slot); err != nil && firstErr == nil {
		firstErr = err
	}
	m.mu.Lock()
	s.down = DownstreamStatus{}
	s.dnsRedirected = 0
	m.mu.Unlock()
	if m.deps.Rebind != nil {
		if err := m.deps.Rebind.Rebind(ctx); err != nil {
			m.setMessage(s, "re-plug the USB cable to drop the old gateway: "+err.Error())
		}
	}
	m.markDisconnected(s)
	if firstErr != nil {
		m.setMessage(s, firstErr.Error())
	}
	return firstErr
}

// SetConfig applies the editable fields. A mode change restarts the slot's
// listeners and converges the daemons; an MTU change restarts the downstream
// (hev reads its MTU at start); a policy change reapplies the chain.
func (m *Manager) SetConfig(ctx context.Context, slot Slot, req proto.SetExitConfigReq) error {
	ctx = detached(ctx)
	m.opMu.Lock()
	defer m.opMu.Unlock()

	s, err := m.get(slot)
	if err != nil {
		return err
	}
	old, nic, _, gw := m.view(s)
	cfg := old
	if req.Mode != nil {
		cfg.Mode = *req.Mode
	}
	if req.DNS != nil {
		cfg.DNS = append([]string(nil), (*req.DNS)...)
	}
	if req.MTU != nil {
		cfg.MTU = *req.MTU
	}
	if req.AllowPrivate != nil {
		cfg.AllowPrivate = *req.AllowPrivate
	}
	if req.PinPeer != nil {
		cfg.PinPeer = *req.PinPeer
	}
	cfg.Normalize()
	if cfg.Mode == proto.ExitModeWstunnel && cfg.Enabled {
		if err := m.ensureBinaries(cfg); err != nil {
			return m.fail(s, err)
		}
	}
	if err := m.persist(cfg, nic); err != nil {
		return m.fail(s, fmt.Errorf("write config: %w", err))
	}
	m.mu.Lock()
	s.cfg = cfg
	c := s.comps
	m.mu.Unlock()

	if !cfg.Enabled {
		return nil
	}
	if c != nil && cfg.PinPeer != old.PinPeer {
		c.Mux.SetPinPeer(cfg.PinPeer)
		c.Proxy.SetPinPeer(cfg.PinPeer)
	}
	switch {
	case cfg.MTU != old.MTU:
		if err := m.down.Stop(ctx, slot); err != nil {
			log.Warnf("exit: slot %s: %s", slot.ID, err)
		}
		fallthrough
	case cfg.Mode != old.Mode:
		m.stopComponents(s)
		if err := m.down.Start(ctx, slot); err != nil {
			return m.fail(s, err)
		}
		if err := m.startComponents(s, gw); err != nil {
			return m.fail(s, err)
		}
	case cfg.AllowPrivate != old.AllowPrivate:
		if err := m.down.Start(ctx, slot); err != nil {
			return m.fail(s, err)
		}
	}
	m.refreshDownstream(ctx, s)
	return nil
}

// RegenerateToken is D11: new token, restriction yaml rewritten (wstunnel
// hot-reloads it), every session on the slot closed, the connected peer's
// limiter entry cleared.
func (m *Manager) RegenerateToken(ctx context.Context, slot Slot) (string, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	s, err := m.get(slot)
	if err != nil {
		return "", err
	}
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	cfg, nic, _, _ := m.view(s)
	cfg.Token = token
	cfg.TokenCreatedAt = m.deps.Now().UTC().Truncate(time.Second)
	if err := m.persist(cfg, nic); err != nil {
		return "", m.fail(s, fmt.Errorf("write config: %w", err))
	}
	m.mu.Lock()
	s.cfg = cfg
	c := s.comps
	source := s.peerSource
	m.mu.Unlock()

	if c != nil {
		c.Mux.CloseAll("token regenerated")
		c.Proxy.CloseAll("token regenerated")
	}
	if source != "" {
		m.limiter.Reset(source)
	}
	return token, nil
}

// Disconnect drops the active exit without touching anything else.
func (m *Manager) Disconnect(slot Slot) error {
	s, err := m.get(slot)
	if err != nil {
		return err
	}
	m.mu.Lock()
	c := s.comps
	m.mu.Unlock()
	if c != nil {
		c.Mux.CloseAll("disconnected by operator")
		c.Proxy.CloseAll("disconnected by operator")
	}
	return nil
}

// OnAttach is the post-attach hook the router calls once the UDC is bound:
// the gadget netdev exists from this point (D25). The router's context is
// the startup budget's, which may be past its deadline by now; the converge
// runs regardless, bounded by execRunner's per-command timeout, since a
// converge cut short leaves the consumer a NIC with no address and no lease
// until the next watchdog tick.
func (m *Manager) OnAttach(ctx context.Context) { m.onRebind(detached(ctx)) }

// onRebind is the presentation manager's rebind subscriber (D25).
func (m *Manager) onRebind(ctx context.Context) {
	if !m.opMu.TryLock() {
		return // a transaction is running; it converges on its own
	}
	defer m.opMu.Unlock()
	m.convergeMu.Lock()
	defer m.convergeMu.Unlock()
	for _, s := range m.snapshot() {
		if m.config(s).Enabled {
			m.convergeSlot(ctx, s)
		}
	}
}

// convergeSlot is one D25 pass for an enabled slot: re-resolve the NIC,
// converge the downstream, address the NIC if S30rndis has not, and rebind the
// forwarder to the current address.
func (m *Manager) convergeSlot(ctx context.Context, s *slotState) {
	m.mu.Lock()
	prevGW := s.gw
	m.mu.Unlock()

	nic, gw := m.resolveNIC(ctx, s)
	if err := m.down.Start(ctx, s.slot); err != nil {
		log.Warnf("exit: slot %s converge: %s", s.slot.ID, err)
	}
	if nic != "" && !gw.IsValid() {
		if err := m.down.StartRNDIS(ctx); err != nil {
			log.Warnf("exit: slot %s: %s", s.slot.ID, err)
		}
		_, gw = m.resolveNIC(ctx, s)
		if gw.IsValid() {
			if err := m.down.Start(ctx, s.slot); err != nil {
				log.Warnf("exit: slot %s converge: %s", s.slot.ID, err)
			}
		}
	}

	m.mu.Lock()
	c := s.comps
	m.mu.Unlock()
	if c == nil && gw.IsValid() {
		// The listeners never came up (a taken SOCKS port at boot, a mode
		// switch whose restart failed): retry them here like every other
		// part of the slot, or hev keeps dialling a closed port for good.
		if err := m.startComponents(s, gw); err != nil {
			log.Warnf("exit: slot %s: %s", s.slot.ID, err)
			m.setMessage(s, err.Error())
		} else {
			m.setMessage(s, "")
		}
		m.mu.Lock()
		c = s.comps
		m.mu.Unlock()
	}
	if c != nil && gw.IsValid() && (gw != prevGW || !c.DNS.Bound()) {
		if err := c.DNS.Bind(gw); err != nil {
			log.Warnf("exit: slot %s: dns forwarder on %s: %s", s.slot.ID, gw, err)
		}
	}
	m.refreshDownstream(ctx, s)
}

// watchdog is D8's 30 s converge for every enabled slot.
func (m *Manager) watchdog() {
	ticker := time.NewTicker(WatchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.Tick(context.Background())
		}
	}
}

// Tick is one watchdog pass, exported for tests. It yields to a running
// transaction: the next tick is 30 s away and the transaction leaves the
// slot converged.
func (m *Manager) Tick(ctx context.Context) {
	if !m.opMu.TryLock() {
		return
	}
	defer m.opMu.Unlock()
	m.convergeMu.Lock()
	defer m.convergeMu.Unlock()
	for _, s := range m.snapshot() {
		if !m.config(s).Enabled {
			m.stopOrphan(ctx, s)
			continue
		}
		m.convergeSlot(ctx, s)
		if !m.connected(s) {
			m.markDisconnected(s)
		}
	}
}

// stopOrphan is the watchdog's pass over a disabled slot: a downstream that
// is still up (the server died between Disable's config flip and its S94exit
// stop) is stopped, since nothing else ever would; boot skips disabled slots
// and the transactions are over.
func (m *Manager) stopOrphan(ctx context.Context, s *slotState) {
	if m.config(s).Pending {
		return
	}
	report, err := m.down.Status(ctx, s.slot)
	if err != nil {
		return
	}
	if !report.Down.Hev && !report.Down.Routing && !report.Down.NAT {
		return
	}
	log.Warnf("exit: slot %s is disabled but its downstream is up (hev=%v routing=%v nat=%v); stopping it",
		s.slot.ID, report.Down.Hev, report.Down.Routing, report.Down.NAT)
	if err := m.down.Stop(ctx, s.slot); err != nil {
		log.Warnf("exit: slot %s: %s", s.slot.ID, err)
	}
}

// Status is the per-slot status the UI polls (D19).
func (m *Manager) Status(slot Slot) (proto.GetExitStatusRsp, error) {
	s, err := m.get(slot)
	if err != nil {
		return proto.GetExitStatusRsp{}, err
	}
	return m.status(s), nil
}

// Slots lists every slot's status in slot order.
func (m *Manager) Slots() proto.GetExitSlotsRsp {
	rsp := proto.GetExitSlotsRsp{Slots: []proto.GetExitStatusRsp{}}
	for _, s := range m.snapshot() {
		rsp.Slots = append(rsp.Slots, m.status(s))
	}
	return rsp
}

func (m *Manager) status(s *slotState) proto.GetExitStatusRsp {
	m.mu.Lock()
	c := s.comps
	rsp := proto.GetExitStatusRsp{
		Slot:            s.slot.ID,
		Enabled:         s.cfg.Enabled,
		Pending:         s.cfg.Pending,
		Mode:            s.cfg.Mode,
		Token:           s.cfg.Token,
		Tunnel:          proto.ExitDisconnected,
		PreviousPeer:    s.st.PreviousPeer,
		PeerChangedAt:   s.st.PeerChangedAt,
		LastConnectedAt: s.st.LastConnectedAt,
		NIC:             s.nic,
		Downstream:      s.down,
		DNS:             proto.ExitDNSStats{Redirected: s.dnsRedirected},
		Bytes:           s.bytes.Totals(),
		Message:         s.message,
	}
	m.mu.Unlock()

	if c != nil {
		state, peer, since := c.Door.Attached()
		rsp.Tunnel = state
		rsp.Peer = peer
		rsp.ConnectedAt = since
		if state == proto.ExitConnected && since != nil {
			rsp.UptimeSeconds = int64(m.deps.Now().Sub(*since).Seconds())
			m.mu.Lock()
			s.wasConnected = true
			if s.st.LastConnectedAt == nil {
				now := m.deps.Now()
				s.st.LastConnectedAt = &now
				rsp.LastConnectedAt = &now
			}
			m.mu.Unlock()
		}
		stats := c.DNS.Stats()
		rsp.DNS.Queries, rsp.DNS.Failures = stats.Queries, stats.Failures
		rsp.Downstream.DNS = c.DNS.Bound()
		rsp.Upstream = c.Probe.Result()
	}
	return rsp
}

// Config is the editable part of the slot config, without the token.
func (m *Manager) Config(slot Slot) (proto.GetExitConfigRsp, error) {
	s, err := m.get(slot)
	if err != nil {
		return proto.GetExitConfigRsp{}, err
	}
	cfg := m.config(s)
	return proto.GetExitConfigRsp{
		Slot: cfg.Slot, Mode: cfg.Mode, DNS: append([]string{}, cfg.DNS...), MTU: cfg.MTU,
		AllowPrivate: cfg.AllowPrivate, PinPeer: cfg.PinPeer,
	}, nil
}

// Commands templates both modes' commands from the request (D15).
func (m *Manager) Commands(slot Slot, r *http.Request) (proto.GetExitCommandsRsp, error) {
	s, err := m.get(slot)
	if err != nil {
		return proto.GetExitCommandsRsp{}, err
	}
	return Commands(slot, m.config(s), OriginOf(r)), nil
}

const logTailLines = 200

var tokenShaped = regexp.MustCompile(`\b[a-z2-9]{8}\b`)

// Logs tails both daemon logs with token-shaped values redacted.
func (m *Manager) Logs(slot Slot) (proto.GetExitLogsRsp, error) {
	s, err := m.get(slot)
	if err != nil {
		return proto.GetExitLogsRsp{}, err
	}
	token := m.config(s).Token
	redact := func(lines []string) []string {
		out := make([]string, 0, len(lines))
		for _, line := range lines {
			if token != "" {
				line = strings.ReplaceAll(line, token, "********")
			}
			out = append(out, tokenShaped.ReplaceAllString(line, "********"))
		}
		return out
	}
	return proto.GetExitLogsRsp{
		Hev:      redact(tailFile(slot.LogPath("hev"), logTailLines)),
		Wstunnel: redact(tailFile(slot.LogPath("wstunnel"), logTailLines)),
	}, nil
}

func tailFile(path string, n int) []string {
	const tailBytes = 64 << 10
	file, err := os.Open(path)
	if err != nil {
		return []string{}
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return []string{}
	}
	offset := int64(0)
	if info.Size() > tailBytes {
		offset = info.Size() - tailBytes
	}
	buf := make([]byte, info.Size()-offset)
	read, err := file.ReadAt(buf, offset)
	if err != nil && read == 0 {
		return []string{}
	}
	content := string(buf[:read])
	if offset > 0 {
		if i := strings.IndexByte(content, '\n'); i >= 0 {
			content = content[i+1:]
		}
	}
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// Gate is the token-gated surface under /exit/<slot>/ (D10). It returns false
// for every rejection, and the caller writes the one 404 gin would write for
// an unknown route, so an unknown slot, a wrong token, a locked source and a
// disabled slot are indistinguishable from the outside.
func (m *Manager) Gate(w http.ResponseWriter, r *http.Request, slotID, rest string) bool {
	source := SourceKey(r.RemoteAddr)
	if m.limiter.Locked(source) {
		return false
	}

	presented := bearerToken(r.Header.Get("Authorization"))
	slot, err := ParseSlot(slotID)
	var s *slotState
	if err == nil {
		s, _ = m.get(slot)
	}
	// Compare against something whatever happened, so an unknown slot costs
	// the same time as a wrong token. The compare only ever counts when the
	// slot's token is one this package issued: a config whose token is empty
	// or otherwise invalid (loadConfig repairs those, but the gate does not
	// rely on it) would otherwise match an absent or malformed header, since
	// two empty strings compare equal.
	expected := strings.Repeat("x", TokenLength)
	var cfg Config
	var c *components
	if s != nil {
		m.mu.Lock()
		cfg = s.cfg
		c = s.comps
		m.mu.Unlock()
	}
	valid := s != nil && ValidToken(cfg.Token)
	if valid {
		expected = cfg.Token
	}
	match := subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) == 1
	ok := valid && match
	if s == nil || !ok {
		// A token attempt that failed: unknown slot or wrong token.
		m.limiter.Fail(source)
		return false
	}
	if !cfg.Enabled || c == nil {
		// The right token on a slot that is off (or mid-transaction) leaks
		// nothing and is not counted: the exit's own reconnect loop while the
		// operator has the slot disabled must not lock the exit out of the
		// slot it is about to be re-enabled on.
		return false
	}

	switch {
	case rest == "/native":
		if cfg.Mode != proto.ExitModeNative {
			return false
		}
		c.Mux.ServeNative(w, r)
		return true
	case clientNames[strings.TrimPrefix(rest, "/")]:
		var fingerprint string
		origin := OriginOf(r)
		if origin.TLS() {
			if cert, err := readCertificate(); err == nil {
				fingerprint = cert.Fingerprint
			}
		}
		body, found := RenderClient(strings.TrimPrefix(rest, "/"), slot, cfg, origin, fingerprint)
		if !found {
			return false
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return true
	default:
		if cfg.Mode != proto.ExitModeWstunnel {
			return false
		}
		c.Proxy.ServeHTTP(w, r)
		return true
	}
}

func bearerToken(header string) string {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
