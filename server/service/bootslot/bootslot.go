package bootslot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	BootDir   = "/boot"
	Confirmed = "/run/nanokvm-kernel-confirmed"
	Pending   = "/kvmapp/kernel_pending"

	cmdlinePath = "/proc/cmdline"
	statePrefix = "ab_state="
)

// SlotTrial and SlotGood are the values u-boot passes in nanokvm_slot.
const (
	SlotTrial = "trial"
	SlotGood  = "good"
)

var ErrNoState = errors.New("uEnv.txt carries no ab_state")

type Paths struct {
	Root        string
	Cmdline     string
	Pending     string
	ConfirmPath string
}

func Default() Paths {
	return Paths{Root: BootDir, Cmdline: cmdlinePath, Pending: Pending, ConfirmPath: Confirmed}
}

func (p Paths) uenv() string    { return filepath.Join(p.Root, "uEnv.txt") }
func (p Paths) bootcnt() string { return filepath.Join(p.Root, "bootcnt") }
func (p Paths) good() string    { return filepath.Join(p.Root, "boot.sd") }
func (p Paths) alt() string     { return filepath.Join(p.Root, "boot.alt") }

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
	if err := copyFile(p.good(), p.alt()); err != nil {
		return fmt.Errorf("commit kernel: %w", err)
	}
	if err := p.setState("committed"); err != nil {
		return fmt.Errorf("commit boot policy: %w", err)
	}
	if err := replace(p.bootcnt(), make([]byte, 4)); err != nil {
		return fmt.Errorf("reset boot counter: %w", err)
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
