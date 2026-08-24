package bootslot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	BootDir   = "/boot"
	Confirmed = "/run/nanokvm-kernel-confirmed"
	Pending   = "/kvmapp/kernel_pending"
	Installed = "/kvmapp/kernel.version"

	cmdlinePath = "/proc/cmdline"
	statePrefix = "ab_state="
)

// StateCommitted and StateTrial are the two values ab_state carries.
const (
	StateCommitted = "committed"
	StateTrial     = "trial"
)

// SlotTrial and SlotGood are the values u-boot passes in nanokvm_slot.
const (
	SlotTrial = "trial"
	SlotGood  = "good"
)

var (
	ErrNoState  = errors.New("uEnv.txt carries no ab_state")
	ErrState    = errors.New("unknown ab_state")
	ErrNotReady = errors.New("trial kernel never reached a serving state; the next boot rolls back")
)

type Paths struct {
	Root          string
	Cmdline       string
	Pending       string
	ConfirmPath   string
	InstalledPath string
}

func Default() Paths {
	return Paths{
		Root:          BootDir,
		Cmdline:       cmdlinePath,
		Pending:       Pending,
		ConfirmPath:   Confirmed,
		InstalledPath: Installed,
	}
}

func (p Paths) uenv() string    { return filepath.Join(p.Root, "uEnv.txt") }
func (p Paths) bootcnt() string { return filepath.Join(p.Root, "bootcnt") }
func (p Paths) good() string    { return filepath.Join(p.Root, "boot.sd") }

// Alt is the trial slot an OTA writes over. Nothing else in /boot may be
// created: the partition holds under 2 MiB free, less than one kernel.
func (p Paths) Alt() string { return filepath.Join(p.Root, "boot.alt") }

// Slot reports the slot the running kernel booted from. Empty when the
// bootloader predates the A/B policy.
func (p Paths) Slot() string {
	data, err := os.ReadFile(p.Cmdline)
	if err != nil {
		return ""
	}
	for _, field := range strings.Fields(string(data)) {
		if value, ok := strings.CutPrefix(field, "nanokvm_slot="); ok {
			return value
		}
	}
	return ""
}

// setState rewrites only the ab_state line. Every other byte, including the
// trailing "\n\0" that env import scans for and the sdboot= last line, is
// preserved.
func (p Paths) setState(state string) error {
	data, err := os.ReadFile(p.uenv())
	if err != nil {
		return err
	}
	start := 0
	if !bytes.HasPrefix(data, []byte(statePrefix)) {
		start = bytes.Index(data, append([]byte("\n"), statePrefix...))
		if start < 0 {
			return ErrNoState
		}
		start++
	}
	end := bytes.IndexByte(data[start:], '\n')
	if end < 0 {
		return ErrNoState
	}
	end += start

	var out bytes.Buffer
	out.Write(data[:start])
	out.WriteString(statePrefix + state)
	out.Write(data[end:])
	return replace(p.uenv(), out.Bytes())
}

// SetState arms or disarms the trial slot. Step 3 of an OTA install, and the
// only writer of ab_state besides Confirm.
func (p Paths) SetState(state string) error {
	if state != StateCommitted && state != StateTrial {
		return fmt.Errorf("%w: %q", ErrState, state)
	}
	return p.setState(state)
}

// ResetBootCount zeroes the counter u-boot increments on every trial attempt.
func (p Paths) ResetBootCount() error {
	return replace(p.bootcnt(), make([]byte, 4))
}

// MarkPending records the version being tried, so a rollback can name it.
func (p Paths) MarkPending(version string) error {
	return replace(p.Pending, []byte(strings.TrimSpace(version)+"\n"))
}

// ClearRollback drops the marker a rollback was reported from, so the warning
// does not outlive the operator having seen it.
func (p Paths) ClearRollback() error {
	if err := os.Remove(p.Pending); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InstalledVersion is the kernel version that last passed its trial. Empty on
// a device whose kernel has only ever come from a flashed image.
func (p Paths) InstalledVersion() string {
	data, err := os.ReadFile(p.InstalledPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// replace writes through a temporary file in the same directory so a torn
// write leaves the previous policy intact.
func replace(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, "."+filepath.Base(path)+".new")
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer handle.Close()
	_ = handle.Sync()
	return nil
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Confirm marks a trial kernel good. It signals the trial guard first, so a
// slow commit cannot be interrupted by the guard's deadline; a commit that
// fails after that simply rolls back on the next boot, because ab_state is
// still trial and bootcnt has already been incremented.
func (p Paths) Confirm() error {
	if p.Slot() != SlotTrial {
		return nil
	}
	if err := os.WriteFile(p.ConfirmPath, nil, 0o644); err != nil {
		return fmt.Errorf("signal trial guard: %w", err)
	}
	if err := copyFile(p.good(), p.Alt()); err != nil {
		return fmt.Errorf("commit kernel: %w", err)
	}
	if err := p.setState("committed"); err != nil {
		return fmt.Errorf("commit boot policy: %w", err)
	}
	if err := p.ResetBootCount(); err != nil {
		return fmt.Errorf("reset boot counter: %w", err)
	}
	if version, err := os.ReadFile(p.Pending); err == nil {
		_ = replace(p.InstalledPath, version)
	}
	_ = os.Remove(p.Pending)
	return nil
}

// RolledBack reports the kernel version that failed its trial, when the
// running kernel is the committed one and a trial was outstanding.
func (p Paths) RolledBack() (string, bool) {
	if p.Slot() != SlotGood {
		return "", false
	}
	data, err := os.ReadFile(p.Pending)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

// Ready is the gate a trial kernel has to pass before it is committed.
// Reaching userspace is not enough: a kernel that boots with no working NIC
// leaves a device nobody can reach, and that has to roll back, so both probes
// have to agree before Confirm runs.
type Ready struct {
	Serving  func() bool
	Routable func() bool
	Timeout  time.Duration
	Interval time.Duration
}

func (p Paths) ConfirmWhenReady(ready Ready) error {
	if p.Slot() != SlotTrial {
		return nil
	}
	deadline := time.Now().Add(ready.Timeout)
	for time.Now().Before(deadline) {
		if ready.Serving() && ready.Routable() {
			return p.Confirm()
		}
		time.Sleep(ready.Interval)
	}
	return ErrNotReady
}

// Routable reports whether any interface holds a non-loopback address, which
// is what separates a working device from one nobody can reach.
func Routable() bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	return routable(addrs)
}

func routable(addrs []net.Addr) bool {
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() {
			return true
		}
	}
	return false
}
