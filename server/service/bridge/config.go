package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

const (
	BridgeName  = "br0"
	StockUplink = "eth0"

	// The name the gadget NIC has on a device where nothing else is holding it,
	// and nothing more than that. It is a default for a script that has no other
	// answer, never the authority for the live name.
	//
	// gether_setup names the netdev with "usb%d". Applying a profile unlinks the
	// outgoing net function from configs/c.1, but unlinking does not destroy an
	// f_ncm or f_rndis: only rmdir does, and gether_setup holds the netdev until
	// then. An orphaned function therefore keeps usb0 and the live one comes up
	// as usb1. presentation.Manager.NIC reads the linked function's configfs
	// ifname and is the only authority for which name is live.
	StockGadgetName = "usb0"

	// wlan0 is the out-of-band recovery path and is never enslaved.
	RecoveryName = "wlan0"
)

const (
	StateDir   = "/etc/kvm/presentation/network"
	UplinkPath = "/etc/kvm/network/l2-uplink"

	// Read verbatim by system_state.cpp in preference to its own
	// "ip route | grep eth0", so an OTA that ships a server without a new
	// kvm_system still reports a gateway on the OLED.
	GatewayPath = "/etc/kvm/gateway"

	// Captured and restored because S30eth:55 appends rather than replaces.
	ResolvPath = "/etc/resolv.conf"

	// Keeps the literal eth0 spelling under every uplink: it names S30eth's one
	// DHCP client rather than an interface, and deriving it from the uplink
	// would leave stop unable to find a client start launched.
	UdhcpcPidPath = "/run/udhcpc.eth0.pid"

	NoDHCPPath     = "/boot/eth.nodhcp"
	SysClassNetDir = "/sys/class/net"

	// S30rndis puts the standalone 10.x.y.1/24 on the gadget NIC and starts a
	// udhcpd on it unless this flag exists. Both are wrong once the NIC is a
	// bridge port: the address belongs to a subnet the bridge does not carry,
	// and the DHCP server is then sitting on the bridged segment, where it can
	// answer the target ahead of the LAN's own server with a lease whose config
	// names no router and no DNS.
	RNDISNoDHCPDPath = "/boot/rndis.nodhcpd"

	// S30rndis generates one of these per interface and names it in the argv of
	// the udhcpd it starts, which is the only handle this package has on that
	// process. The name is derived from the interface rather than fixed: a
	// device whose NIC came up as usb1 has /etc/udhcpd.usb1.conf, and a match
	// that only knows usb0 leaves a DHCP server running on the bridged segment.
	UdhcpdConfDir = "/etc"
)

const (
	snapshotName      = "snapshot.json"
	pendingName       = "pending.json"
	lastKnownGoodName = "last-known-good.json"

	// Where S29bridge moves an armed marker it acted on. The script has to run
	// before S30eth and therefore before the server exists, so it performs the
	// restore itself; the rename is what leaves the server something to report
	// instead of a device that is silently back on eth0 for no recorded reason.
	recoveredName = "recovered.json"

	dirMode  os.FileMode = 0o755
	fileMode os.FileMode = 0o600
)

// Long enough for udhcpc's -t 10 -T 1 to run to exhaustion twice, short enough
// that an operator watching a dead session does not power-cycle first.
const DefaultWindow = 60 * time.Second

// var rather than const so the package tests against t.TempDir(), the way
// tunnel/config_test.go swaps configDir.
var (
	stateDir      = StateDir
	uplinkPath    = UplinkPath
	gatewayPath   = GatewayPath
	resolvPath    = ResolvPath
	udhcpcPidPath = UdhcpcPidPath
	noDHCPPath    = NoDHCPPath
	sysClassNet   = SysClassNetDir

	rndisNoDHCPDPath = RNDISNoDHCPDPath
	udhcpdConfDir    = UdhcpdConfDir
	procDir          = "/proc"
)

var (
	ErrRecoveryInterface = errors.New("bridge: wlan0 is the out-of-band recovery path and is never enslaved")
	ErrNotEnslavable     = errors.New("bridge: interface is not enslavable")
	ErrBusy              = errors.New("bridge: another apply is in progress")
	ErrPreflight         = errors.New("bridge: enable refused before anything was changed")
	ErrNoSnapshot        = errors.New("bridge: snapshot not found")
)

// The fixed half of the set of interfaces ever passed to
// "ip link set <dev> master br0". eth0 can stay data because WriteUplink
// refuses to name anything but br0 or eth0, so the uplink is never something
// else.
var enslavable = map[string]bool{
	StockUplink: true,
}

// The moving half. The gadget NIC used to be the constant usb0 and sat in the
// map beside eth0; it cannot, now that an orphaned net function can push the
// live one to usb1. What survives of "a closed set enforced by data" is the
// shape: gether_setup allocates the netdev name from "usb%d" and nothing else
// on this device is called usbN, so wlan0 is still absent by construction and
// so is every other interface an operator might be reached over.
//
// The shape is the floor, not the whole rule. The only caller that can put a
// usbN through it is enslaveGadget, which enslaves what presentation.Manager
// named as the live NIC a moment earlier and nothing else, so in practice the
// set is eth0 plus exactly that name.
var gadgetNIC = regexp.MustCompile(`^usb[0-9]+$`)

// IFNAMSIZ is 16 including the NUL. Names read back out of a snapshot pass
// through this before they can reach an argv.
var deviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,14}$`)

// wlan0 is reported distinctly because that rejection has a cause worth
// surfacing to an operator.
func checkEnslavable(name string) error {
	if name == RecoveryName {
		return ErrRecoveryInterface
	}
	if !enslavable[name] && !gadgetNIC.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrNotEnslavable, name)
	}
	return nil
}

// The config S30rndis writes for a gadget NIC, and the string that identifies
// its udhcpd in the process table.
func udhcpdGadgetConf(nic string) string {
	return filepath.Join(udhcpdConfDir, "udhcpd."+nic+".conf")
}

// Whether an argv names any gadget NIC's udhcpd config. The whole usbN family
// is matched rather than one name: a protocol switch renames the NIC without
// stopping the udhcpd the previous boot started, so the process still to kill
// before enslaving usb1 is very often the one holding /etc/udhcpd.usb0.conf.
func namesGadgetDHCPDConf(args []string) bool {
	for _, arg := range args {
		dir, file := filepath.Split(arg)
		if filepath.Clean(dir) != filepath.Clean(udhcpdConfDir) {
			continue
		}
		name, ok := strings.CutPrefix(file, "udhcpd.")
		if !ok {
			continue
		}
		if name, ok = strings.CutSuffix(name, ".conf"); ok && gadgetNIC.MatchString(name) {
			return true
		}
	}
	return false
}

func safeDevice(name string) bool {
	return deviceName.MatchString(name)
}

// Pending carries the snapshot path rather than the snapshot, so writing it is
// a single block and a boot-time reader can act on it without parsing a
// capture. SnapshotPath is absolute: a relative one would resolve against
// whatever working directory the recovering process happened to have.
type Pending struct {
	Operation    string `json:"operation"`
	SnapshotPath string `json:"snapshotPath"`

	ArmedAt  time.Time `json:"armedAt"`
	Deadline time.Time `json:"deadline"`
}

func (p Pending) Expired(now time.Time) bool {
	return !now.Before(p.Deadline)
}

func (p Pending) Remaining(now time.Time) time.Duration {
	if d := p.Deadline.Sub(now); d > 0 {
		return d
	}
	return 0
}

// Written before pending.json is removed, so a crash in the disarm sequence
// leaves a device a boot-time check still restores rather than one that comes
// up half-applied with no record of it.
type LastKnownGood struct {
	Enabled   bool               `json:"enabled"`
	Uplink    string             `json:"uplink"`
	State     proto.BridgeState  `json:"state"`
	Checks    proto.BridgeChecks `json:"checks"`
	Message   string             `json:"message"`
	AppliedAt time.Time          `json:"appliedAt"`
}

// Every file is 0600 through utils.AtomicFile, so a power cut leaves either the
// whole old file or the whole new one and never a truncated marker.
type Store struct {
	mu  sync.Mutex
	dir string
}

func NewStore() *Store {
	return &Store{dir: stateDir}
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) SnapshotPath() string { return filepath.Join(s.dir, snapshotName) }

func (s *Store) pendingPath() string { return filepath.Join(s.dir, pendingName) }

func (s *Store) lastKnownGoodPath() string { return filepath.Join(s.dir, lastKnownGoodName) }

func (s *Store) recoveredPath() string { return filepath.Join(s.dir, recoveredName) }

// AtomicFile fsyncs before the rename, which is what makes step 1 meaningful:
// a power cut after this point leaves a complete record on disk.
func (s *Store) WriteSnapshot(snapshot *Snapshot) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", s.dir, err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode snapshot: %w", err)
	}

	path := s.SnapshotPath()
	if err := utils.WriteFileAtomic(path, append(data, '\n'), fileMode); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ReadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoSnapshot, path)
		}
		return nil, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode snapshot %s: %w", path, err)
	}
	return &snapshot, nil
}

// Arm must be called before the first mutation and returns only once the marker
// is durable.
func (s *Store) Arm(pending Pending) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", s.dir, err)
	}

	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return fmt.Errorf("encode pending: %w", err)
	}
	return utils.WriteFileAtomic(s.pendingPath(), append(data, '\n'), fileMode)
}

// A corrupt marker is an error rather than a nil: treating it as "nothing
// armed" is the one interpretation that skips a restore.
func (s *Store) Pending() (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readMarker(s.pendingPath())
}

// The marker S29bridge already acted on. It is the armed record verbatim, moved
// by a rename, so it names the operation that was interrupted and when it was
// armed without the shell having had to author a file.
func (s *Store) Recovered() (*Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readMarker(s.recoveredPath())
}

// Removed only once the outcome it describes is durable, so an interruption
// between the two leaves it to be adopted again on the next boot rather than
// losing it.
func (s *Store) ClearRecovered() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.recoveredPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return utils.SyncDir(s.dir)
}

func readMarker(path string) (*Pending, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var pending Pending
	if err := json.Unmarshal(data, &pending); err != nil {
		return nil, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return &pending, nil
}

// The directory is synced so the removal survives a crash rather than sitting
// in an unflushed dirent.
func (s *Store) Disarm() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.pendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return utils.SyncDir(s.dir)
}

func (s *Store) WriteLastKnownGood(lkg LastKnownGood) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", s.dir, err)
	}

	data, err := json.MarshalIndent(lkg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode last-known-good: %w", err)
	}
	return utils.WriteFileAtomic(s.lastKnownGoodPath(), append(data, '\n'), fileMode)
}

func (s *Store) LastKnownGood() (*LastKnownGood, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.lastKnownGoodPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var lkg LastKnownGood
	if err := json.Unmarshal(data, &lkg); err != nil {
		return nil, fmt.Errorf("decode last-known-good: %w", err)
	}
	return &lkg, nil
}

// Commit is the step-12 disarm. The outcome is published first and the marker
// removed second, so an interruption between them leaves an armed marker over a
// written outcome and the boot-time check restores. The reverse order would
// leave the bridge live, no marker, and a stale outcome claiming it is not.
func (s *Store) Commit(lkg LastKnownGood) error {
	if err := s.WriteLastKnownGood(lkg); err != nil {
		return err
	}
	return s.Disarm()
}

// An absent file is eth0, the stock state, and is not an error.
func ReadUplink() string {
	data, err := os.ReadFile(uplinkPath)
	if err != nil {
		return StockUplink
	}

	name := trimLine(string(data))
	if name == "" || !safeDevice(name) {
		return StockUplink
	}
	return name
}

func WriteUplink(name string) error {
	if name != BridgeName && name != StockUplink {
		return fmt.Errorf("bridge: refusing to write uplink %q", name)
	}
	return utils.WriteFileAtomic(uplinkPath, []byte(name+"\n"), 0o644)
}

// Returns every reader to eth0 through the absent-file fallback, leaving the
// on-disk state byte-identical to a device that never bridged.
func RemoveUplink() error {
	if err := os.Remove(uplinkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return utils.SyncDir(filepath.Dir(uplinkPath))
}

func ReadGateway() (string, bool) {
	data, err := os.ReadFile(gatewayPath)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// Validated as IPv4 first, so nothing else reaches a file another process
// parses verbatim.
func WriteGateway(gateway string) error {
	if !isIPv4(gateway) {
		return fmt.Errorf("bridge: refusing to write gateway %q", gateway)
	}
	return utils.WriteFileAtomic(gatewayPath, []byte(gateway+"\n"), 0o644)
}

func restoreFile(path, content string, present bool, mode os.FileMode) error {
	if !present {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return utils.SyncDir(filepath.Dir(path))
	}
	return utils.WriteFileAtomic(path, []byte(content), mode)
}
