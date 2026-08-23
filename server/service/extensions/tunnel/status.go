package tunnel

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"NanoKVM-Server/proto"
	log "github.com/sirupsen/logrus"
)

var (
	procDir     = "/proc"
	pidDir      = "/var/run"
	initDir     = "/etc/init.d"
	initSeedDir = "/kvmapp/system/init.d"
)

const (
	watchdogInterval = 30 * time.Second
	healthMaxAge     = 90 * time.Second
	logTailBytes     = 64 << 10
)

func pidPath(name proto.TunnelName) string {
	return filepath.Join(pidDir, string(name)+".pid")
}

func initScriptPath(name proto.TunnelName) string {
	return filepath.Join(initDir, "S97"+string(name))
}

func initSeedPath(name proto.TunnelName) string {
	return filepath.Join(initSeedDir, "S97"+string(name))
}

func logPath(name proto.TunnelName) string {
	return filepath.Join(logDir, string(name)+".log")
}

func pidOf(name proto.TunnelName) (int, bool) {
	data, err := os.ReadFile(pidPath(name))
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}

	cmdline, err := os.ReadFile(filepath.Join(procDir, strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return 0, false
	}

	argv := strings.Split(string(cmdline), "\x00")
	if len(argv) == 0 || argv[0] == "" {
		return 0, false
	}

	switch filepath.Base(argv[0]) {
	case string(name), string(name) + ".cmd":
		return pid, true
	default:
		return 0, false
	}
}

func logTail(name proto.TunnelName, n int) []string {
	if n <= 0 {
		return nil
	}

	file, err := os.Open(logPath(name))
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil
	}

	size := info.Size()
	offset := int64(0)
	if size > logTailBytes {
		offset = size - logTailBytes
	}

	buffer := make([]byte, size-offset)
	read, err := file.ReadAt(buffer, offset)
	if err != nil && read == 0 {
		return nil
	}

	content := string(buffer[:read])
	if offset > 0 {
		if index := strings.IndexByte(content, '\n'); index >= 0 {
			content = content[index+1:]
		}
	}

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func isEnabled(name proto.TunnelName) bool {
	_, err := os.Stat(initScriptPath(name))
	return err == nil
}

// The init script's presence is what boot reads, but an installer that copies
// /kvmapp/system/init.d over /etc/init.d writes it without being asked. The
// config file holds what the user asked for, and that is what wins.
func intendedEnabled(name proto.TunnelName) bool {
	cfg, err := loadConfig(name)
	if err != nil || cfg.Enabled == nil {
		return isEnabled(name)
	}
	return *cfg.Enabled
}

func setIntendedEnabled(name proto.TunnelName, enabled bool) error {
	cfg, err := loadConfig(name)
	if err != nil {
		return err
	}
	cfg.Enabled = &enabled
	return saveConfig(name, cfg)
}

// Reconcile runs once at server start, after rcS has already run whatever was
// in /etc/init.d, and makes the enabled set match the recorded intent.
func Reconcile(names []proto.TunnelName) {
	for _, name := range names {
		cfg, err := loadConfig(name)
		if err != nil {
			log.Errorf("failed to read %s config: %s", name, err)
			continue
		}

		enabled := isEnabled(name)

		if cfg.Enabled == nil {
			if !enabled {
				continue
			}
			if err := setIntendedEnabled(name, true); err != nil {
				log.Errorf("failed to adopt %s boot intent: %s", name, err)
			}
			continue
		}

		// Copied rather than only restored when missing: an update ships a new
		// script in /kvmapp that nothing else here would ever install.
		if *cfg.Enabled {
			if err := enableInitScript(name); err != nil {
				log.Errorf("failed to restore %s init script: %s", name, err)
			}
			continue
		}

		if !enabled {
			continue
		}

		_ = runInitScript(name, "stop")
		if err := os.Remove(initScriptPath(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Errorf("failed to remove %s init script: %s", name, err)
		}
		log.Debugf("tunnel %s disabled again after it was reinstalled", name)
	}
}

func isInstalled(name proto.TunnelName) bool {
	if _, err := os.Stat(binaryFile(name)); err == nil {
		return true
	}
	_, err := os.Stat(seedPath(name))
	return err == nil
}

func isConfigured(cfg Config) bool {
	if strings.TrimSpace(cfg.Args) != "" {
		return true
	}
	for _, value := range cfg.Env {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func currentState(name proto.TunnelName) (proto.TunnelState, string) {
	spec, ok := specOf(name)
	if !ok {
		return proto.TunnelError, "unknown tunnel"
	}

	if !isInstalled(name) {
		return proto.TunnelNotInstall, ""
	}

	cfg, err := loadConfig(name)
	if err != nil {
		return proto.TunnelError, err.Error()
	}
	if !isConfigured(cfg) {
		return proto.TunnelNotConfigured, ""
	}

	if _, running := pidOf(name); !running {
		if !intendedEnabled(name) {
			return proto.TunnelStopped, ""
		}

		message := "service is enabled but not running"
		if lines := logTail(name, 1); len(lines) > 0 {
			message = lines[0]
		}
		return proto.TunnelError, message
	}

	if spec.HealthFile == "" {
		return proto.TunnelRunning, ""
	}

	info, err := os.Stat(spec.HealthFile)
	if err != nil || time.Since(info.ModTime()) > healthMaxAge {
		return proto.TunnelRunning, ""
	}
	return proto.TunnelConnected, ""
}

func StartWatchdog(names []proto.TunnelName) {
	if len(names) == 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(watchdogInterval)
		defer ticker.Stop()

		for range ticker.C {
			revive(names)
		}
	}()
}

func revive(names []proto.TunnelName) {
	for _, name := range names {
		if !intendedEnabled(name) {
			continue
		}
		if _, running := pidOf(name); running {
			continue
		}

		if err := runInitScript(name, "start"); err != nil {
			log.Errorf("failed to restart tunnel %s: %s", name, err)
			continue
		}
		log.Debugf("tunnel %s restarted", name)
	}
}
