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
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// downstreamTimeout bounds one S94exit or S30rndis invocation. The scripts
// are lock-serialised and idempotent, so a hang here is a wedged iptables or
// a daemon that will not start, and the manager wants an error rather than a
// stuck transaction.
const downstreamTimeout = 60 * time.Second

// downstreamWaitDelay is how long Run waits for the stdout and stderr pipes
// to close after the script has exited or been killed. cmd.Wait otherwise
// blocks for as long as any descendant holds the write end: the with_lock
// subshell after the deadline killed only its parent, or a daemon that did
// not detach its stdio. S94exit's daemons are started through
// start-stop-daemon -b (stdio to /dev/null) and a shell that redirects to
// the log before it execs the binary, so on a healthy device the pipes close
// with the script and this delay is never spent.
const downstreamWaitDelay = time.Second

// execRunner is the real Runner: one process per call, stdout returned trimmed,
// stderr folded into the error. The script runs in its own process group so
// the deadline kills everything it started and not just the script itself,
// and Wait is bounded by WaitDelay so an orphan holding the pipe cannot wedge
// the manager's transaction (S94-2).
type execRunner struct {
	timeout   time.Duration // zero means downstreamTimeout
	waitDelay time.Duration // zero means downstreamWaitDelay
}

func (r execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	timeout, waitDelay := r.timeout, r.waitDelay
	if timeout == 0 {
		timeout = downstreamTimeout
	}
	if waitDelay == 0 {
		waitDelay = downstreamWaitDelay
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// Setpgid makes the script the leader of a new process group; the
	// with_lock subshell and anything it runs stay in that group, while the
	// daemons leave it through start-stop-daemon's setsid. Cancel then signals
	// the group, so the deadline releases the slot lock and closes the pipes
	// instead of leaving a subshell holding both.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if errors.Is(err, exec.ErrWaitDelay) {
		// The script exited 0 and only a descendant kept the pipe open past
		// WaitDelay. That is the script's success; what it printed before
		// the pipe was abandoned is the output.
		err = nil
	}
	msg := strings.TrimSpace(stderr.String())
	if err != nil {
		if msg == "" {
			return out, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	// A script that exits 0 still warns on stderr about what it had to do
	// differently (a daemon started on /dev/null because its log's
	// filesystem is full), and the server log is the only place an operator
	// sees that; each line is logged on its own under the script and its
	// arguments.
	for _, line := range strings.Split(msg, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			log.Warnf("exit: %s %s: %s", filepath.Base(name), strings.Join(args, " "), line)
		}
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
	Spawns        daemonSpawns
	Raw           string
}

// daemonSpawns are the counters `S94exit status` prints beside the vector:
// how many times start_daemon has spawned each daemon since its counter was
// last removed by stop. A daemon found alive is not a spawn, so a count that
// grows between two watchdog ticks is a daemon that died in between. An
// older script prints neither key and both stay 0.
type daemonSpawns struct {
	Hev      uint64
	Wstunnel uint64
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
// 0|1, nic and gw as strings, dns_redirected, hev_spawns and wstunnel_spawns
// as counters. Unknown keys are skipped so a newer script still parses.
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
		case "hev_spawns":
			if n, err := strconv.ParseUint(value, 10, 64); err == nil {
				report.Spawns.Hev = n
			}
		case "wstunnel_spawns":
			if n, err := strconv.ParseUint(value, 10, 64); err == nil {
				report.Spawns.Wstunnel = n
			}
		default:
			continue
		}
		seen++
	}
	return report, seen > 0
}
