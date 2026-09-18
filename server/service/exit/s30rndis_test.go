package exit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// s30 runs kvmapp/system/init.d/S30rndis for real under sh. The script names
// its roots absolutely, so, like the bridge package's addressing test, every
// one it reaches is relocated into a sandbox by rewriting the source, and
// /proc with them: the udhcpd match reads /proc/<pid>/cmdline, and the fake
// table holds real pids. The udhcpd shim forks a sleeper that outlives the
// script and registers it under its own pid with the command line the real
// server would have, so "how many servers are running" is a kill(pid, 0) on
// real processes and the script's kill is a real kill.
type s30 struct {
	t      *testing.T
	root   string
	trace  string
	proc   string
	pids   string
	script string
	env    []string
}

func newS30(t *testing.T) *s30 {
	t.Helper()
	root := t.TempDir()
	h := &s30{
		t:      t,
		root:   root,
		trace:  filepath.Join(root, "trace"),
		proc:   filepath.Join(root, "proc"),
		pids:   filepath.Join(root, "udhcpd.pids"),
		script: filepath.Join(root, S30Script),
	}
	for _, dir := range []string{h.proc, filepath.Join(root, "bin"), filepath.Join(root, "etc"), filepath.Join(root, "boot")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(h.killSleepers)

	// The device from the bridge bug: the linked NCM function came up as usb1.
	gadget := filepath.Join(root, "sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0")
	if err := os.MkdirAll(gadget, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(gadget, "ifname"):               "usb1\n",
		filepath.Join(root, "boot/usb.ncm"):           "",
		filepath.Join(root, "boot/rndis.ipv4_prefix"): "10.7.8\n",
		filepath.Join(root, "etc/profile"):            "",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeExec(t, filepath.Join(root, "bin", "ip"), "#!/bin/sh\necho \"ip $*\" >> \"$STUB_TRACE\"\n")
	writeExec(t, filepath.Join(root, "bin", "udhcpd"), `#!/bin/sh
echo "udhcpd $*" >> "$STUB_TRACE"
sleep 300 </dev/null >/dev/null 2>&1 &
pid=$!
mkdir -p "$FAKE_PROC/$pid"
printf 'udhcpd\0%s\0%s\0' "$1" "$2" > "$FAKE_PROC/$pid/cmdline"
echo "$pid" >> "$STUB_PIDS"
`)

	source, err := os.ReadFile(filepath.Join("..", "..", "..", "kvmapp", "system", "init.d", S30Script))
	if err != nil {
		t.Fatalf("read S30rndis: %v", err)
	}
	body := string(source)
	for from, to := range map[string]string{
		"/sys/kernel/config": filepath.Join(root, "sys/kernel/config"),
		"/boot/":             filepath.Join(root, "boot") + "/",
		"/etc/profile":       filepath.Join(root, "etc/profile"),
		"/etc/udhcpd.":       filepath.Join(root, "etc") + "/udhcpd.",
		"/proc/":             h.proc + "/",
	} {
		body = strings.ReplaceAll(body, from, to)
	}
	if err := os.WriteFile(h.script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	h.env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"STUB_TRACE="+h.trace,
		"STUB_PIDS="+h.pids,
		"FAKE_PROC="+h.proc,
	)
	return h
}

// run is one `S30rndis <action>`; a non-zero exit fails the test.
func (h *s30) run(action string) {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", h.script, action)
	cmd.Env = h.env
	if out, err := cmd.CombinedOutput(); err != nil {
		h.t.Fatalf("S30rndis %s: %v\n%s", action, err, out)
	}
}

// sleeper starts a detached process the way the udhcpd shim does and lists it
// in the fake /proc under the given command line. It stands for a process the
// script did not start: another interface's udhcpd, or something unrelated.
func (h *s30) sleeper(cmdline ...string) int {
	h.t.Helper()
	out, err := exec.Command("/bin/sh", "-c", "sleep 300 </dev/null >/dev/null 2>&1 & echo $!").Output()
	if err != nil {
		h.t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		h.t.Fatal(err)
	}
	f, err := os.OpenFile(h.pids, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		h.t.Fatal(err)
	}
	if _, err := f.WriteString(strconv.Itoa(pid) + "\n"); err != nil {
		h.t.Fatal(err)
	}
	_ = f.Close()
	dir := filepath.Join(h.proc, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(strings.Join(cmdline, "\x00")+"\x00"), 0o644); err != nil {
		h.t.Fatal(err)
	}
	return pid
}

// shimPIDs is every udhcpd the shim started, oldest first, taken from the
// list the shim appends to; sleepers the test started are in it too.
func (h *s30) shimPIDs() []int {
	h.t.Helper()
	data, err := os.ReadFile(h.pids)
	if err != nil {
		h.t.Fatal(err)
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			h.t.Fatalf("pid list: %q", line)
		}
		pids = append(pids, pid)
	}
	return pids
}

func alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func (h *s30) killSleepers() {
	data, err := os.ReadFile(h.pids)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 1 {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

func (h *s30) traceLines() []string {
	h.t.Helper()
	data, err := os.ReadFile(h.trace)
	if err != nil {
		h.t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func countLines(lines []string, want string) int {
	n := 0
	for _, l := range lines {
		if l == want {
			n++
		}
	}
	return n
}

// The exit manager runs `S30rndis start` after every gadget rebind, and each
// rebind recreates the gadget netdev with no address and administratively
// down. On the deployed unit six rebinds left six udhcpd processes, all bound
// to 0.0.0.0:67, and the NIC stayed DOWN until someone raised it by hand.
// start is therefore a converge: it raises the link and replaces the udhcpd
// already serving this interface, leaving every other process alone; stop
// still takes every gadget NIC's server down.
func TestGadgetAddressingStartIsIdempotent(t *testing.T) {
	h := newS30(t)
	conf := filepath.Join(h.root, "etc/udhcpd.usb1.conf")
	// A previous boot's server on the interface the NIC used to have, which
	// start must leave alone and stop must take down, and a process whose
	// command line merely mentions the config path, which nothing may touch.
	other := h.sleeper("udhcpd", "-S", filepath.Join(h.root, "etc/udhcpd.usb0.conf"))
	bystander := h.sleeper("tail", "-f", conf)

	h.run("start")
	h.run("start")

	lines := h.traceLines()
	for _, want := range []string{
		"ip addr add 10.7.8.1/24 dev usb1",
		"ip link set dev usb1 up",
		"udhcpd -S " + conf,
	} {
		if n := countLines(lines, want); n != 2 {
			t.Fatalf("%q issued %d times, want 2\ntrace:\n%s", want, n, strings.Join(lines, "\n"))
		}
	}
	for i, l := range lines {
		if strings.HasPrefix(l, "udhcpd ") && (i == 0 || lines[i-1] != "ip link set dev usb1 up") {
			t.Fatalf("the link is not raised before udhcpd starts\ntrace:\n%s", strings.Join(lines, "\n"))
		}
	}

	pids := h.shimPIDs()
	if len(pids) != 4 {
		t.Fatalf("pids = %v, want two sleepers and two shim starts", pids)
	}
	first, second := pids[2], pids[3]
	if alive(first) {
		t.Fatalf("the first start's udhcpd %d survived the second start", first)
	}
	if !alive(second) {
		t.Fatalf("the second start's udhcpd %d is not running", second)
	}
	if !alive(other) {
		t.Fatalf("start killed the other interface's udhcpd %d", other)
	}
	if !alive(bystander) {
		t.Fatalf("start killed a bystander %d whose argv names the config", bystander)
	}

	h.run("stop")
	if alive(second) {
		t.Fatalf("stop left this interface's udhcpd %d running", second)
	}
	if alive(other) {
		t.Fatalf("stop left the other interface's udhcpd %d running", other)
	}
	if !alive(bystander) {
		t.Fatalf("stop killed a bystander %d", bystander)
	}
}
