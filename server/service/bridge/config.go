package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

const (
	BridgeName  = "br0"
	StockUplink = "eth0"
	GadgetName  = "usb0"

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
)

const (
	snapshotName      = "snapshot.json"
	pendingName       = "pending.json"
	lastKnownGoodName = "last-known-good.json"

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
)

var (
	ErrRecoveryInterface = errors.New("bridge: wlan0 is the out-of-band recovery path and is never enslaved")
	ErrNotEnslavable     = errors.New("bridge: interface is not enslavable")
	ErrBusy              = errors.New("bridge: another apply is in progress")
	ErrNoSnapshot        = errors.New("bridge: snapshot not found")
)

// The closed set of interfaces ever passed to "ip link set <dev> master br0".
// wlan0 is absent by construction, so the rule is enforced by the data rather
// than by a check every call site has to remember.
var enslavable = map[string]bool{
	StockUplink: true,
	GadgetName:  true,
}

// IFNAMSIZ is 16 including the NUL. Names read back out of a snapshot pass
// through this before they can reach an argv.
var deviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,14}$`)

// wlan0 is reported distinctly because that rejection has a cause worth
// surfacing to an operator.
func checkEnslavable(name string) error {
	if name == RecoveryName {
		return ErrRecoveryInterface
	}
	if !enslavable[name] {
		return fmt.Errorf("%w: %q", ErrNotEnslavable, name)
	}
	return nil
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

	data, err := os.ReadFile(s.pendingPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var pending Pending
	if err := json.Unmarshal(data, &pending); err != nil {
		return nil, fmt.Errorf("decode pending: %w", err)
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
