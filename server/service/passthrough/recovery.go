package passthrough

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"NanoKVM-Server/utils"
)

var (
	recoveryStatePath = "/etc/kvm/passthrough/session.json"
	procRoot          = "/proc"
)

type recoveryState struct {
	Port    uint32 `json:"port"`
	Reclaim bool   `json:"reclaim"`
}

func saveRecoveryState(state recoveryState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode passthrough recovery state: %w", err)
	}
	if err := utils.WriteFileAtomic(recoveryStatePath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("save passthrough recovery state: %w", err)
	}
	return nil
}

func loadRecoveryState() (recoveryState, error) {
	data, err := os.ReadFile(recoveryStatePath)
	if err != nil {
		return recoveryState{}, err
	}
	var state recoveryState
	if err := json.Unmarshal(data, &state); err != nil {
		return recoveryState{}, fmt.Errorf("decode passthrough recovery state: %w", err)
	}
	return state, nil
}

func clearRecoveryState() error {
	err := os.Remove(recoveryStatePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("clear passthrough recovery state: %w", err)
	}
	return utils.SyncDir(filepath.Dir(recoveryStatePath))
}

func stopProxyOrphans() error {
	pids, err := proxyPIDs()
	if err != nil {
		return err
	}

	var result error
	for _, pid := range pids {
		if err := stopOrphan(pid); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func proxyPIDs() ([]int, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procRoot, err)
	}

	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join(procRoot, entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if first, _, _ := bytes.Cut(cmdline, []byte{0}); string(first) == proxyBinary {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func stopOrphan(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("stop orphan usb-proxy %d: %w", pid, err)
	}

	if waitProcessExit(pid, stopTimeout) {
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("kill orphan usb-proxy %d: %w", pid, err)
	}
	if !waitProcessExit(pid, time.Second) {
		return fmt.Errorf("kill orphan usb-proxy %d: process still exists", pid)
	}
	return nil
}

func waitProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(procRoot, strconv.Itoa(pid))); errors.Is(err, os.ErrNotExist) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
