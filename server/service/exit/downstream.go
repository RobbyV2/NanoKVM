package exit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// downstreamTimeout bounds one S94exit or S30rndis invocation. The scripts
// are lock-serialised and idempotent, so a hang here is a wedged iptables or
// a daemon that will not start, and the manager wants an error rather than a
// stuck transaction.
const downstreamTimeout = 60 * time.Second

// execRunner is the real Runner: one process per call, stdout returned trimmed,
// stderr folded into the error.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, downstreamTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return out, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	return out, nil
}

// downstream runs the two init scripts through a Runner and parses what
// S94exit status prints.
type downstream struct {
	run Runner
}

// scriptPath prefers the installed copy and falls back to the seed, so the
// manager works on a device whose /etc/init.d has not been refreshed yet.
func scriptPath(name string) (string, error) {
	for _, dir := range []string{InitDir, InitSeedDir} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s not found in %s or %s", name, InitDir, InitSeedDir)
}

func (d downstream) s94(ctx context.Context, args ...string) (string, error) {
	script, err := scriptPath(S94Script)
	if err != nil {
		return "", err
	}
	return d.run.Run(ctx, script, args...)
}

func (d downstream) s30(ctx context.Context, action string) error {
	script, err := scriptPath(S30Script)
	if err != nil {
		return err
	}
	_, err = d.run.Run(ctx, script, action)
	return err
}

// Start is `S94exit start <n>`: a converge, safe to repeat (D8).
func (d downstream) Start(ctx context.Context, slot Slot) error {
	_, err := d.s94(ctx, "start", slot.ID)
	return err
}

// Stop is `S94exit stop <n>`, the exact reverse of Start.
func (d downstream) Stop(ctx context.Context, slot Slot) error {
	_, err := d.s94(ctx, "stop", slot.ID)
	return err
}

// StartRNDIS re-addresses the gadget NIC and restarts udhcpd with whatever
// options the gadget.route marker now calls for; RestartRNDIS does the same
// after a stop, which is what the enable and disable transactions run.
func (d downstream) StartRNDIS(ctx context.Context) error   { return d.s30(ctx, "start") }
func (d downstream) RestartRNDIS(ctx context.Context) error { return d.s30(ctx, "restart") }

// statusReport is the parsed output of `S94exit status <n>`. Down.DNS is left
// false here: the forwarder's own bind state fills it in (types.go).
type statusReport struct {
	Down          DownstreamStatus
	NIC           string
	Gateway       string
	DNSRedirected uint64
	Raw           string
}

// ErrStatusUnparsed is returned when the script printed nothing that looks
// like the key=value vector, which means it did not run at all.
var ErrStatusUnparsed = errors.New("S94exit status printed no status vector")

// Status runs `S94exit status <n>`. The script exits non-zero whenever any
// element of the vector is missing, which is a report and not a failure, so
// the exit code is folded into the vector and only an unparsable output is
// an error.
func (d downstream) Status(ctx context.Context, slot Slot) (statusReport, error) {
	out, runErr := d.s94(ctx, "status", slot.ID)
	report, ok := parseStatus(out)
	if !ok {
		if runErr != nil {
			return report, fmt.Errorf("%w: %v", ErrStatusUnparsed, runErr)
		}
		return report, ErrStatusUnparsed
	}
	return report, nil
}

// parseStatus reads `key=value` lines: forward routing tun hev wstunnel nat as
// 0|1, nic and gw as strings, dns_redirected as a counter.
func parseStatus(out string) (statusReport, bool) {
	report := statusReport{Raw: out}
	seen := 0
	for _, line := range strings.Split(out, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		flag := value == "1"
		switch key {
		case "forward":
			report.Down.Forward = flag
		case "routing":
			report.Down.Routing = flag
		case "tun":
			report.Down.Tun = flag
		case "hev":
			report.Down.Hev = flag
		case "wstunnel":
			report.Down.Wstunnel = flag
		case "nat":
			report.Down.NAT = flag
		case "nic":
			report.NIC = value
		case "gw":
			report.Gateway = value
		case "dns_redirected":
			if n, err := strconv.ParseUint(value, 10, 64); err == nil {
				report.DNSRedirected = n
			}
		default:
			continue
		}
		seen++
	}
	return report, seen > 0
}
