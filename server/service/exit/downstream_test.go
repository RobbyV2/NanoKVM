package exit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	report, ok := parseStatus("forward=1\nrouting=1\ntun=0\nhev=1\nwstunnel=1\nnat=0\nnic=usb0\ngw=10.1.2.1\ndns_redirected=17\n")
	if !ok {
		t.Fatal("status not recognised")
	}
	if !report.Down.Forward || !report.Down.Routing || report.Down.Tun || !report.Down.Hev || !report.Down.Wstunnel || report.Down.NAT {
		t.Fatalf("vector = %+v", report.Down)
	}
	if report.Down.DNS {
		t.Fatal("dns is the forwarder's to report, not the script's")
	}
	if report.NIC != "usb0" || report.Gateway != "10.1.2.1" || report.DNSRedirected != 17 {
		t.Fatalf("extras = %+v", report)
	}

	if _, ok := parseStatus("S94exit: invalid slot 'x'\n"); ok {
		t.Fatal("garbage recognised as a status vector")
	}
}

type scriptedRunner struct {
	calls []string
	out   map[string]string
	err   map[string]error
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	r.calls = append(r.calls, key)
	return r.out[key], r.err[key]
}

func TestDownstreamStatusFoldsTheExitCode(t *testing.T) {
	root := useTempDirs(t)
	writeExec(t, InitSeedDir+"/"+S94Script, "#!/bin/sh\n")
	writeExec(t, InitSeedDir+"/"+S30Script, "#!/bin/sh\n")
	_ = root

	key := InitSeedDir + "/S94exit status 0"
	r := &scriptedRunner{
		out: map[string]string{key: "forward=1\nrouting=0\ntun=0\nhev=0\nwstunnel=1\nnat=0\nnic=\ngw=\ndns_redirected=0"},
		err: map[string]error{key: errors.New("exit status 1")},
	}
	d := downstream{run: r}
	report, err := d.Status(context.Background(), MustSlot("0"))
	if err != nil {
		t.Fatalf("a non-zero status with a vector is a report, got %v", err)
	}
	if !report.Down.Forward || report.Down.Hev {
		t.Fatalf("vector = %+v", report.Down)
	}

	r.out[key] = ""
	if _, err := d.Status(context.Background(), MustSlot("0")); !errors.Is(err, ErrStatusUnparsed) {
		t.Fatalf("empty output: %v", err)
	}

	if err := d.Start(context.Background(), MustSlot("0")); err != nil {
		t.Fatal(err)
	}
	if err := d.RestartRNDIS(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.calls[len(r.calls)-2] != InitSeedDir+"/S94exit start 0" || r.calls[len(r.calls)-1] != InitSeedDir+"/S30rndis restart" {
		t.Fatalf("calls = %v", r.calls)
	}
}

// S94-2: cmd.Wait does not return when the script has exited but a child it
// left behind still holds the stdout pipe, and the deadline used to kill only
// the script, never the with_lock subshell or a stuck child under it. The
// runner must come back within its WaitDelay of the exit or the deadline.
func TestExecRunnerReturnsWhenAnOrphanHoldsTheStdoutPipe(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	// The orphan writes its own pid so the test can check on it and clean up.
	orphan := "sleep 30 &\necho $! > " + pidFile + "\n"
	writeExec(t, filepath.Join(dir, "exits.sh"), "#!/bin/sh\necho forward=1\n"+orphan+"exit 0\n")
	writeExec(t, filepath.Join(dir, "hangs.sh"), "#!/bin/sh\necho started\n"+orphan+"wait\n")

	orphanPid := func() int {
		t.Helper()
		data, err := os.ReadFile(pidFile)
		if err != nil {
			t.Fatal(err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		return pid
	}
	alive := func(pid int) bool { return syscall.Kill(pid, 0) == nil }

	r := execRunner{timeout: 300 * time.Millisecond, waitDelay: 300 * time.Millisecond}
	const bound = 2 * time.Second

	// A script that exited 0 is a success even though a descendant holds the
	// pipe: the output it printed is returned and the orphan is left alone
	// (a daemon that did not detach is not the converge's failure).
	start := time.Now()
	out, err := r.Run(context.Background(), filepath.Join(dir, "exits.sh"))
	if elapsed := time.Since(start); elapsed > bound {
		t.Fatalf("Run took %v with an orphan on stdout", elapsed)
	}
	if err != nil || out != "forward=1" {
		t.Fatalf("Run = %q, %v", out, err)
	}
	pid := orphanPid()
	if !alive(pid) {
		t.Fatal("a script that exited on its own had its orphan killed")
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)

	// A script that hangs is killed at the deadline together with everything
	// in its process group, so nothing is left holding the pipe or the lock.
	start = time.Now()
	_, err = r.Run(context.Background(), filepath.Join(dir, "hangs.sh"))
	if elapsed := time.Since(start); elapsed > bound {
		t.Fatalf("Run took %v on a hung script", elapsed)
	}
	if err == nil || !strings.Contains(err.Error(), "hangs.sh") {
		t.Fatalf("hung script: %v", err)
	}
	pid = orphanPid()
	deadline := time.Now().Add(time.Second)
	for alive(pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if alive(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatal("the deadline killed the script but not its process group")
	}
}
