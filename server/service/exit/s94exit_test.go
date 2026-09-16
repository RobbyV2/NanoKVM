package exit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// s94 runs kvmapp/system/init.d/S94exit under sh with PATH pointing at the
// recording stubs in testdata/stubs, the idiom presentation/testdata uses for
// the gadget scripts. Every root the script reaches is relocated by env.
type s94 struct {
	t       *testing.T
	root    string
	state   string
	trace   string
	sysNet  string
	gadget  string
	exitDir string
	runDir  string
	env     []string
}

func newS94(t *testing.T) *s94 {
	t.Helper()
	root := t.TempDir()
	stubs, err := filepath.Abs(filepath.Join("testdata", "stubs"))
	if err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "..", "kvmapp", "system", "init.d", S94Script))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}

	h := &s94{
		t: t, root: root,
		state:   filepath.Join(root, "state"),
		trace:   filepath.Join(root, "trace"),
		sysNet:  filepath.Join(root, "sys/class/net"),
		gadget:  filepath.Join(root, "sys/kernel/config/usb_gadget/g0/configs/c.1"),
		exitDir: filepath.Join(root, "etc/kvm/exit"),
		runDir:  filepath.Join(root, "var/run"),
	}
	for _, dir := range []string{h.state, h.sysNet, h.gadget, h.exitDir, h.runDir, filepath.Join(root, "bin"), filepath.Join(root, "tmp"), filepath.Join(root, "seed")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The built-in chains the script jumps from.
	for _, chain := range []string{"iptables/filter.FORWARD", "iptables/nat.PREROUTING", "iptables/nat.POSTROUTING", "ip6tables/filter.FORWARD"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(h.state, chain)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(h.state, chain), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The daemons are present so no seed extraction runs. They live until
	// killed, like the real ones, so a pidfile the stub start-stop-daemon
	// writes names a process whose command line carries the binary's path.
	for _, bin := range []string{HevBinary, WstunnelBinary} {
		writeExec(t, filepath.Join(root, "bin", bin), "#!/bin/sh\nwhile :; do sleep 1; done\n")
	}
	t.Cleanup(h.killDaemons)

	h.env = append(os.Environ(),
		"PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
		"STUB_STATE="+h.state,
		"STUB_TRACE="+h.trace,
		"EXIT_DIR="+h.exitDir,
		"BIN_DIR="+filepath.Join(root, "bin"),
		"SEED_DIR="+filepath.Join(root, "seed"),
		"RUN_DIR="+h.runDir,
		"LOG_DIR="+filepath.Join(root, "tmp"),
		"SYS_NET="+h.sysNet,
		"GADGET_CONFIG="+h.gadget,
		"S94EXIT_SCRIPT="+script,
	)
	return h
}

// killDaemons reaps every process the stub start-stop-daemon spawned.
func (h *s94) killDaemons() {
	data, err := os.ReadFile(filepath.Join(h.state, "daemons"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 1 {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
	}
}

// daemonWrapper is the -c body start_daemon hands /bin/sh: redirect to the
// log named by $1, then become the daemon.
const daemonWrapper = `exec >>"$1" 2>&1; shift; exec "$@"`

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

// gadgetNIC registers the live gadget NIC the way configfs and sysfs do.
func (h *s94) gadgetNIC(name, addr string) {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Join(h.gadget, "ncm.usb0"), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.gadget, "ncm.usb0", "ifname"), []byte(name+"\n"), 0o644); err != nil {
		h.t.Fatal(err)
	}
	if strings.Contains(name, "(") {
		return
	}
	if err := os.MkdirAll(filepath.Join(h.sysNet, name), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if addr != "" {
		if err := os.WriteFile(filepath.Join(h.state, "addr."+name), []byte(addr+"\n"), 0o644); err != nil {
			h.t.Fatal(err)
		}
	}
}

func (h *s94) writeEnv(slot Slot, cfg Config, nic string) {
	h.t.Helper()
	old := ConfigDir
	ConfigDir = h.exitDir
	defer func() { ConfigDir = old }()
	if err := writeSlotFiles(slot, cfg, nic); err != nil {
		h.t.Fatal(err)
	}
}

func (h *s94) run(args ...string) (string, int) {
	h.t.Helper()
	_ = os.Remove(h.trace)
	script := ""
	for _, kv := range h.env {
		if strings.HasPrefix(kv, "S94EXIT_SCRIPT=") {
			script = strings.TrimPrefix(kv, "S94EXIT_SCRIPT=")
		}
	}
	cmd := exec.Command("sh", append([]string{script}, args...)...)
	cmd.Env = h.env
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		h.t.Fatalf("run S94exit %v: %v\n%s", args, err, out)
	}
	return string(out), code
}

func (h *s94) traceLines() []string {
	data, _ := os.ReadFile(h.trace)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func (h *s94) stateFile(rel string) string {
	data, _ := os.ReadFile(filepath.Join(h.state, rel))
	return string(data)
}

func (h *s94) count(lines []string, substr string) int {
	n := 0
	for _, l := range lines {
		if strings.Contains(l, substr) {
			n++
		}
	}
	return n
}

func TestS94exitConvergeIsIdempotent(t *testing.T) {
	h := newS94(t)
	slot := MustSlot("0")
	cfg := testConfig(slot)
	h.gadgetNIC("usb0", "10.1.2.1/24")
	h.writeEnv(slot, cfg, "usb0")

	out, code := h.run("start", "0")
	if code != 0 {
		t.Fatalf("first start exited %d:\n%s", code, out)
	}
	first := h.traceLines()
	rules := h.stateFile("rules")
	chain := h.stateFile("iptables/filter.EXIT0")
	nat := h.stateFile("iptables/nat.EXIT0_NAT")
	masq := h.stateFile("iptables/nat.EXIT0_MASQ")
	forward := h.stateFile("iptables/filter.FORWARD")
	table := h.stateFile("route.100")

	// D6: the essentials of the first converge.
	for _, want := range []string{
		"ip tuntap add dev exit0 mode tun",
		"ip addr replace 198.18.0.1/32 dev exit0",
		"ip link set dev exit0 mtu 1280 up",
		"ip route replace default dev exit0 table 100",
		"ip route replace unreachable default metric 4294967295 table 100",
		"ip rule add pref 1001 iif usb0 unreachable",
		"ip rule add pref 1000 iif usb0 lookup 100",
		"sysctl -q -w net.ipv4.ip_forward=1",
		"sysctl -q -w net.ipv4.conf.exit0.rp_filter=2",
		"sysctl -q -w net.ipv4.conf.usb0.route_localnet=0",
		"sysctl -q -w net.ipv6.conf.usb0.disable_ipv6=1",
		// The daemon runs through a shell that redirects to its log after the
		// daemonising fork and execs the binary (-b would hand it /dev/null).
		"start-stop-daemon -S -bmq -p " + h.runDir + "/exit0-hev.pid -x /bin/sh -- -c " + daemonWrapper + " sh " + h.root + "/tmp/exit0-hev.log " + h.root + "/bin/hev-socks5-tunnel " + h.exitDir + "/0/hev.yml",
		"ip6tables -A FORWARD -i usb0 -j REJECT --reject-with icmp6-adm-prohibited",
	} {
		if h.count(first, want) != 1 {
			t.Errorf("first start: %q appears %d times, want once\n%s", want, h.count(first, want), strings.Join(first, "\n"))
		}
	}
	if h.count(first, "wstunnel") != 0 {
		t.Errorf("native mode started wstunnel:\n%s", strings.Join(first, "\n"))
	}
	if strings.Count(rules, "\n") != 2 {
		t.Fatalf("rules after first start:\n%s", rules)
	}
	// The fence is added before the lookup so the consumer is never between them.
	if idx := strings.Index(strings.Join(first, "\n"), "pref 1001"); idx < 0 || idx > strings.Index(strings.Join(first, "\n"), "pref 1000") {
		t.Errorf("fence rule was not added before the lookup rule")
	}
	if forward != "-j EXIT0\n" {
		t.Fatalf("FORWARD jump:\n%s", forward)
	}
	wantChain := []string{
		"-i usb0 -d 0.0.0.0/8 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 127.0.0.0/8 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 169.254.0.0/16 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 198.18.0.0/15 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 224.0.0.0/3 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 10.0.0.0/8 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 100.64.0.0/10 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 172.16.0.0/12 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -d 192.168.0.0/16 -j REJECT --reject-with icmp-admin-prohibited",
		"-i usb0 -o exit0 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1240",
		"-i exit0 -o usb0 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1240",
		"-i usb0 -o exit0 -j ACCEPT",
		"-i exit0 -o usb0 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT",
		"-i usb0 -j REJECT --reject-with icmp-admin-prohibited",
		"-o usb0 ! -i exit0 -j REJECT",
	}
	if chain != strings.Join(wantChain, "\n")+"\n" {
		t.Fatalf("EXIT0 chain:\n%s\nwant:\n%s", chain, strings.Join(wantChain, "\n"))
	}
	wantNAT := "-i usb0 ! -d 10.1.2.1 -p udp --dport 53 -j DNAT --to-destination 10.1.2.1:53\n" +
		"-i usb0 ! -d 10.1.2.1 -p tcp --dport 53 -j DNAT --to-destination 10.1.2.1:53\n"
	if nat != wantNAT {
		t.Fatalf("EXIT0_NAT chain:\n%s", nat)
	}
	if masq != "-s 10.1.2.0/24 -o exit0 -j MASQUERADE\n" {
		t.Fatalf("EXIT0_MASQ chain:\n%s", masq)
	}
	if table != "default dev exit0\nunreachable default metric 4294967295\n" {
		t.Fatalf("table 100:\n%s", table)
	}

	// Second converge: identical state, no duplicate adds, daemon left alone.
	out, code = h.run("start", "0")
	if code != 0 {
		t.Fatalf("second start exited %d:\n%s", code, out)
	}
	second := h.traceLines()
	if h.count(second, "ip tuntap add") != 0 {
		t.Error("second start recreated the tun")
	}
	if h.count(second, "start-stop-daemon -S") != 0 {
		t.Errorf("second start restarted a live daemon:\n%s", strings.Join(second, "\n"))
	}
	if h.count(second, "ip rule del") != 0 {
		t.Errorf("second start deleted rules that name the live NIC:\n%s", strings.Join(second, "\n"))
	}
	if h.count(second, "iptables -I FORWARD") != 0 || h.count(second, "iptables -t nat -I") != 0 {
		t.Errorf("second start duplicated a chain jump:\n%s", strings.Join(second, "\n"))
	}
	// The chains are replaced in one iptables-restore transaction, never
	// flushed and refilled rule by rule while the FORWARD jump is live: an
	// empty EXIT0 falls through to the ACCEPT policy for the whole refill.
	for _, run := range [][]string{first, second} {
		if h.count(run, "iptables-restore -n") != 1 || h.count(run, "iptables -F ") != 0 || h.count(run, "iptables -A ") != 0 || h.count(run, "iptables -t nat -F ") != 0 || h.count(run, "iptables -t nat -A ") != 0 {
			t.Errorf("chains were not replaced in one transaction:\n%s", strings.Join(run, "\n"))
		}
	}
	restored := h.stateFile("iptables-restore.last")
	for _, want := range []string{"*filter\n:EXIT0 - [0:0]\n-A EXIT0 ", "\nCOMMIT\n*nat\n:EXIT0_NAT - [0:0]\n:EXIT0_MASQ - [0:0]\n-A EXIT0_NAT ", "-A EXIT0_MASQ -s 10.1.2.0/24 -o exit0 -j MASQUERADE\nCOMMIT\n"} {
		if !strings.Contains(restored, want) {
			t.Errorf("iptables-restore input lacks %q:\n%s", want, restored)
		}
	}
	if kept, _ := os.ReadFile(filepath.Join(h.runDir, "exit0.rules")); string(kept) != restored {
		t.Errorf("the rules file under RUN_DIR is not what was fed to iptables-restore:\n%s\nvs\n%s", kept, restored)
	}
	for name, before := range map[string]string{
		"rules": rules, "iptables/filter.EXIT0": chain, "iptables/nat.EXIT0_NAT": nat,
		"iptables/nat.EXIT0_MASQ": masq, "iptables/filter.FORWARD": forward, "route.100": table,
	} {
		if after := h.stateFile(name); after != before {
			t.Errorf("%s changed on the second converge:\n%s\nwas:\n%s", name, after, before)
		}
	}
	if h.stateFile("ip6tables/filter.FORWARD") != "-i usb0 -j REJECT --reject-with icmp6-adm-prohibited\n" {
		t.Errorf("ip6tables rule duplicated or missing:\n%s", h.stateFile("ip6tables/filter.FORWARD"))
	}
}

func TestS94exitPrunesRulesForARenamedNIC(t *testing.T) {
	h := newS94(t)
	slot := MustSlot("0")
	h.gadgetNIC("usb0", "10.1.2.1/24")
	h.writeEnv(slot, testConfig(slot), "usb0")
	if _, code := h.run("start", "0"); code != 0 {
		t.Fatal("start failed")
	}
	// The gadget was rebuilt and the NIC came back as usb1.
	h.gadgetNIC("usb1", "10.1.2.1/24")
	if _, code := h.run("start", "0"); code != 0 {
		t.Fatal("second start failed")
	}
	rules := h.stateFile("rules")
	if strings.Contains(rules, "usb0") || strings.Count(rules, "\n") != 2 || !strings.Contains(rules, "iif usb1 lookup 100") {
		t.Fatalf("rules after the rename:\n%s", rules)
	}
	trace := strings.Join(h.traceLines(), "\n")
	if !strings.Contains(trace, "ip rule del pref 1000 iif usb0") || !strings.Contains(trace, "ip rule del pref 1001 iif usb0") {
		t.Fatalf("stale rules were not pruned:\n%s", trace)
	}
}

func TestS94exitSkipsTheNICHalfWithoutAGadgetNetdev(t *testing.T) {
	h := newS94(t)
	slot := MustSlot("0")
	h.gadgetNIC("(unnamed net_device)", "")
	h.writeEnv(slot, testConfig(slot), "")

	out, code := h.run("start", "0")
	if code != 0 {
		t.Fatalf("start without a NIC exited %d:\n%s", code, out)
	}
	trace := strings.Join(h.traceLines(), "\n")
	if !strings.Contains(trace, "ip tuntap add dev exit0") || !strings.Contains(trace, "start-stop-daemon -S") {
		t.Fatalf("NIC-independent half did not come up:\n%s", trace)
	}
	if strings.Contains(trace, "ip rule add") || strings.Contains(trace, "iptables -N") {
		t.Fatalf("NIC-keyed half ran with no netdev:\n%s", trace)
	}
	if !strings.Contains(out, "no gadget NIC registered yet") {
		t.Fatalf("no log line about the missing NIC:\n%s", out)
	}

	out, code = h.run("status", "0")
	if code == 0 {
		t.Fatalf("status reported healthy with no rules:\n%s", out)
	}
	report, ok := parseStatus(out)
	if !ok || report.Down.Routing || !report.Down.Forward || !report.Down.Hev || report.Down.Tun || report.Down.NAT || !report.Down.Wstunnel {
		t.Fatalf("status vector = %+v (ok=%v):\n%s", report.Down, ok, out)
	}
}

func TestS94exitStatusVectorAndStop(t *testing.T) {
	h := newS94(t)
	slot := MustSlot("0")
	cfg := testConfig(slot)
	cfg.Mode = "wstunnel"
	h.gadgetNIC("usb0", "10.1.2.1/24")
	h.writeEnv(slot, cfg, "usb0")

	if out, code := h.run("start", "0"); code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	trace := strings.Join(h.traceLines(), "\n")
	if !strings.Contains(trace, "-p "+h.runDir+"/exit0-wstunnel.pid -x /bin/sh -- -c "+daemonWrapper+" sh "+h.root+"/tmp/exit0-wstunnel.log "+h.root+"/bin/wstunnel server --restrict-config "+h.exitDir+"/0/wstunnel-restrict.yml ws://127.0.0.1:10810") {
		t.Fatalf("wstunnel server not started for Mode B:\n%s", trace)
	}

	// hev has not attached yet: carrier 0, everything else up.
	out, code := h.run("status", "0")
	report, ok := parseStatus(out)
	if !ok {
		t.Fatalf("status output:\n%s", out)
	}
	if code == 0 || report.Down.Tun {
		t.Fatalf("status with carrier 0: code=%d vector=%+v", code, report.Down)
	}
	if !report.Down.Forward || !report.Down.Routing || !report.Down.Hev || !report.Down.Wstunnel || !report.Down.NAT {
		t.Fatalf("vector = %+v:\n%s", report.Down, out)
	}
	if report.NIC != "usb0" || report.Gateway != "10.1.2.1" || report.DNSRedirected != 4 {
		t.Fatalf("extras = %+v", report)
	}

	if err := os.WriteFile(filepath.Join(h.sysNet, "exit0", "carrier"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code = h.run("status", "0")
	if code != 0 {
		t.Fatalf("status with everything up exited %d:\n%s", code, out)
	}

	// Mode switched back to native: the converge stops wstunnel.
	cfg.Mode = "native"
	h.writeEnv(slot, cfg, "usb0")
	if out, code := h.run("start", "0"); code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	if strings.Contains(strings.Join(h.traceLines(), "\n"), "start-stop-daemon -S -bmq -p "+h.runDir+"/exit0-wstunnel.pid") {
		t.Fatal("wstunnel restarted in native mode")
	}
	if _, err := os.Stat(filepath.Join(h.runDir, "exit0-wstunnel.pid")); !os.IsNotExist(err) {
		t.Fatal("wstunnel pidfile survived the switch to native")
	}
	// The converge flushed the NAT chain (counters zeroed) after carrying its
	// 4 hits over; the stub reports 4 live hits again, so the total is 8.
	out, _ = h.run("status", "0")
	if report, ok := parseStatus(out); !ok || report.DNSRedirected != 8 {
		t.Fatalf("dns_redirected after a second converge = %d (ok=%v), want 8:\n%s", report.DNSRedirected, ok, out)
	}

	// Stop reverses everything by identity.
	if out, code := h.run("stop", "0"); code != 0 {
		t.Fatalf("stop exited %d:\n%s", code, out)
	}
	trace = strings.Join(h.traceLines(), "\n")
	for _, want := range []string{
		"start-stop-daemon -K -q -p " + h.runDir + "/exit0-hev.pid",
		"iptables -D FORWARD -j EXIT0",
		"iptables -F EXIT0", "iptables -X EXIT0",
		"iptables -t nat -D PREROUTING -j EXIT0_NAT", "iptables -t nat -X EXIT0_NAT",
		"iptables -t nat -D POSTROUTING -j EXIT0_MASQ", "iptables -t nat -X EXIT0_MASQ",
		"ip6tables -D FORWARD -i usb0 -j REJECT --reject-with icmp6-adm-prohibited",
		"ip rule del pref 1000", "ip rule del pref 1001",
		"ip route flush table 100",
		"ip link del exit0",
	} {
		if !strings.Contains(trace, want) {
			t.Errorf("stop did not run %q:\n%s", want, trace)
		}
	}
	if h.stateFile("rules") != "" || h.stateFile("iptables/filter.FORWARD") != "" || h.stateFile("iptables/nat.PREROUTING") != "" || h.stateFile("iptables/nat.POSTROUTING") != "" || h.stateFile("ip6tables/filter.FORWARD") != "" {
		t.Fatalf("stop left state behind: rules=%q FORWARD=%q PRE=%q POST=%q v6=%q",
			h.stateFile("rules"), h.stateFile("iptables/filter.FORWARD"), h.stateFile("iptables/nat.PREROUTING"), h.stateFile("iptables/nat.POSTROUTING"), h.stateFile("ip6tables/filter.FORWARD"))
	}
	for _, gone := range []string{"iptables/filter.EXIT0", "iptables/nat.EXIT0_NAT", "iptables/nat.EXIT0_MASQ", "route.100"} {
		if _, err := os.Stat(filepath.Join(h.state, gone)); !os.IsNotExist(err) {
			t.Errorf("%s survived stop", gone)
		}
	}
	if _, err := os.Stat(filepath.Join(h.sysNet, "exit0")); !os.IsNotExist(err) {
		t.Error("tun survived stop")
	}
	if _, err := os.Stat(filepath.Join(h.runDir, "exit0-hev.pid")); !os.IsNotExist(err) {
		t.Error("hev pidfile survived stop")
	}
	if _, err := os.Stat(filepath.Join(h.runDir, "exit0.dns_redirected")); !os.IsNotExist(err) {
		t.Error("dns_redirected total survived stop")
	}

	// A second stop is harmless.
	if out, code := h.run("stop", "0"); code != 0 {
		t.Fatalf("second stop exited %d:\n%s", code, out)
	}
}

func TestS94exitNoArgumentHonoursEnabledAndPending(t *testing.T) {
	h := newS94(t)
	h.gadgetNIC("usb0", "10.1.2.1/24")

	enabled := testConfig(MustSlot("0"))
	h.writeEnv(MustSlot("0"), enabled, "usb0")
	pending := testConfig(MustSlot("1"))
	pending.Pending = true
	h.writeEnv(MustSlot("1"), pending, "usb0")
	disabled := testConfig(MustSlot("2"))
	disabled.Enabled = false
	h.writeEnv(MustSlot("2"), disabled, "usb0")

	if out, code := h.run("start"); code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	trace := strings.Join(h.traceLines(), "\n")
	if !strings.Contains(trace, "ip tuntap add dev exit0") {
		t.Error("enabled slot 0 was not started")
	}
	if strings.Contains(trace, "exit1") || strings.Contains(trace, "exit2") {
		t.Errorf("a pending or disabled slot was started:\n%s", trace)
	}

	if out, code := h.run("stop"); code != 0 {
		t.Fatalf("stop exited %d:\n%s", code, out)
	}
	trace = strings.Join(h.traceLines(), "\n")
	for _, n := range []string{"0", "1", "2"} {
		if !strings.Contains(trace, "ip rule del pref "+strconv.Itoa(1000+2*int(n[0]-'0'))) {
			t.Errorf("stop skipped slot %s:\n%s", n, trace)
		}
	}

	if out, code := h.run("start", "x"); code == 0 {
		t.Fatalf("invalid slot accepted:\n%s", out)
	}
	if _, code := h.run("frobnicate"); code == 0 {
		t.Fatal("unknown verb accepted")
	}
}

// A pidfile whose pid is alive but belongs to another process (pids wrap
// every few hours on the device) is not the daemon: status must not report
// it, stop must not signal it, and start must start a real one.
func TestS94exitIgnoresARecycledPid(t *testing.T) {
	h := newS94(t)
	slot := MustSlot("0")
	h.gadgetNIC("usb0", "10.1.2.1/24")
	h.writeEnv(slot, testConfig(slot), "usb0")
	if out, code := h.run("start", "0"); code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	pidfile := filepath.Join(h.runDir, "exit0-hev.pid")
	if err := os.WriteFile(filepath.Join(h.sysNet, "exit0", "carrier"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := h.run("status", "0"); code != 0 {
		t.Fatalf("status with the real daemon exited %d:\n%s", code, out)
	}

	// hev died and its pid was handed to this test process, which is alive
	// and is not hev.
	h.killDaemons()
	impostor := strconv.Itoa(os.Getpid()) + "\n"
	if err := os.WriteFile(pidfile, []byte(impostor), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := h.run("status", "0")
	report, ok := parseStatus(out)
	if !ok || report.Down.Hev || code == 0 {
		t.Fatalf("a recycled pid was reported as hev (code=%d):\n%s", code, out)
	}

	if out, code := h.run("stop", "0"); code != 0 {
		t.Fatalf("stop exited %d:\n%s", code, out)
	}
	if trace := strings.Join(h.traceLines(), "\n"); strings.Contains(trace, "start-stop-daemon -K -q -p "+pidfile) {
		t.Fatalf("stop signalled a pid that is not hev:\n%s", trace)
	}
	if _, err := os.Stat(pidfile); !os.IsNotExist(err) {
		t.Fatal("stop left the stale pidfile behind")
	}

	if err := os.WriteFile(pidfile, []byte(impostor), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := h.run("start", "0"); code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	if trace := strings.Join(h.traceLines(), "\n"); !strings.Contains(trace, "start-stop-daemon -S -bmq -p "+pidfile) {
		t.Fatalf("start trusted a recycled pid and did not restart hev:\n%s", trace)
	}
	if now, _ := os.ReadFile(pidfile); string(now) == impostor {
		t.Fatal("the pidfile still names the impostor after start")
	}
}

// Without iptables-restore on PATH the same rules file is applied rule by
// rule through iptables, ending in the same state.
func TestS94exitFallsBackToIptablesWithoutRestore(t *testing.T) {
	h := newS94(t)
	stubs, _ := filepath.Abs(filepath.Join("testdata", "stubs"))
	entries, err := os.ReadDir(stubs)
	if err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(h.root, "stubs-without-restore")
	if err := os.MkdirAll(partial, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "iptables-restore" {
			continue
		}
		if err := os.Symlink(filepath.Join(stubs, e.Name()), filepath.Join(partial, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
	for i, kv := range h.env {
		if strings.HasPrefix(kv, "PATH=") {
			h.env[i] = "PATH=" + partial + string(os.PathListSeparator) + os.Getenv("PATH")
		}
	}

	slot := MustSlot("0")
	h.gadgetNIC("usb0", "10.1.2.1/24")
	h.writeEnv(slot, testConfig(slot), "usb0")
	out, code := h.run("start", "0")
	if code != 0 {
		t.Fatalf("start exited %d:\n%s", code, out)
	}
	trace := h.traceLines()
	if h.count(trace, "iptables-restore") != 0 || h.count(trace, "iptables -t filter -F EXIT0") != 1 || h.count(trace, "iptables -t nat -F EXIT0_NAT") != 1 || h.count(trace, "iptables -t nat -F EXIT0_MASQ") != 1 {
		t.Fatalf("fallback did not flush and refill through iptables:\n%s", strings.Join(trace, "\n"))
	}
	if got := strings.Count(h.stateFile("iptables/filter.EXIT0"), "\n"); got != 15 {
		t.Fatalf("EXIT0 has %d rules, want 15:\n%s", got, h.stateFile("iptables/filter.EXIT0"))
	}
	if h.stateFile("iptables/nat.EXIT0_NAT") != "-i usb0 ! -d 10.1.2.1 -p udp --dport 53 -j DNAT --to-destination 10.1.2.1:53\n-i usb0 ! -d 10.1.2.1 -p tcp --dport 53 -j DNAT --to-destination 10.1.2.1:53\n" {
		t.Fatalf("EXIT0_NAT chain:\n%s", h.stateFile("iptables/nat.EXIT0_NAT"))
	}
	if h.stateFile("iptables/nat.EXIT0_MASQ") != "-s 10.1.2.0/24 -o exit0 -j MASQUERADE\n" {
		t.Fatalf("EXIT0_MASQ chain:\n%s", h.stateFile("iptables/nat.EXIT0_MASQ"))
	}
	if h.stateFile("iptables/filter.FORWARD") != "-j EXIT0\n" {
		t.Fatalf("FORWARD jump:\n%s", h.stateFile("iptables/filter.FORWARD"))
	}
}

// stop restores the sysctls start changed to what they were, except that
// ip_forward is left alone while another slot or tailscaled still needs it,
// and whenever forwarding stays on the iif fence outlives stop: a consumer
// whose lease still names this box as router (D5, refused rebind) must be
// refused, not forwarded onto the LAN with a 10.x source.
func TestS94exitStopRestoresSysctlsAndFencesWhileForwardingStaysOn(t *testing.T) {
	start := func(t *testing.T, h *s94, slot string) {
		t.Helper()
		s := MustSlot(slot)
		h.writeEnv(s, testConfig(s), "usb0")
		if out, code := h.run("start", slot); code != 0 {
			t.Fatalf("start %s exited %d:\n%s", slot, code, out)
		}
	}
	stop := func(t *testing.T, h *s94, slot string) string {
		t.Helper()
		out, code := h.run("stop", slot)
		if code != 0 {
			t.Fatalf("stop %s exited %d:\n%s", slot, code, out)
		}
		return out
	}
	sysctl := func(h *s94, key string) string {
		return strings.TrimSpace(h.stateFile("sysctl/" + key))
	}

	t.Run("forwarding was off", func(t *testing.T) {
		h := newS94(t)
		h.gadgetNIC("usb0", "10.1.2.1/24")
		start(t, h, "0")
		if sysctl(h, "net.ipv4.ip_forward") != "1" || sysctl(h, "net.ipv6.conf.usb0.disable_ipv6") != "1" {
			t.Fatalf("start did not set the sysctls: forward=%q disable_ipv6=%q", sysctl(h, "net.ipv4.ip_forward"), sysctl(h, "net.ipv6.conf.usb0.disable_ipv6"))
		}
		out := stop(t, h, "0")
		for key, want := range map[string]string{"net.ipv4.ip_forward": "0", "net.ipv6.conf.usb0.disable_ipv6": "0", "net.ipv4.conf.usb0.route_localnet": "0"} {
			if got := sysctl(h, key); got != want {
				t.Errorf("%s after stop = %q, want %q", key, got, want)
			}
		}
		if h.stateFile("rules") != "" {
			t.Errorf("rules left after stop with forwarding off:\n%s", h.stateFile("rules"))
		}
		if _, err := os.Stat(filepath.Join(h.runDir, "exit0.sysctl")); !os.IsNotExist(err) {
			t.Error("the sysctl record survived stop")
		}
		if strings.Contains(out, "fence") {
			t.Errorf("stop kept a fence with forwarding off:\n%s", out)
		}
	})

	t.Run("forwarding was on before the first start", func(t *testing.T) {
		h := newS94(t)
		h.gadgetNIC("usb0", "10.1.2.1/24")
		if err := os.MkdirAll(filepath.Join(h.state, "sysctl"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(h.state, "sysctl", "net.ipv4.ip_forward"), []byte("1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		start(t, h, "0")
		// A watchdog converge must not overwrite the record with our own value.
		if out, code := h.run("start", "0"); code != 0 {
			t.Fatalf("second start exited %d:\n%s", code, out)
		}
		out := stop(t, h, "0")
		if sysctl(h, "net.ipv4.ip_forward") != "1" {
			t.Errorf("stop zeroed ip_forward that was on before the first start")
		}
		if sysctl(h, "net.ipv6.conf.usb0.disable_ipv6") != "0" {
			t.Errorf("disable_ipv6 after stop = %q, want 0", sysctl(h, "net.ipv6.conf.usb0.disable_ipv6"))
		}
		rules := h.stateFile("rules")
		if rules != "1001:\tfrom all iif usb0 unreachable\n" {
			t.Errorf("rules after stop with forwarding on:\n%q\nwant only the fence", rules)
		}
		if !strings.Contains(out, "fence") {
			t.Errorf("stop did not say the fence stays:\n%s", out)
		}
	})

	t.Run("tailscaled started after us", func(t *testing.T) {
		h := newS94(t)
		h.gadgetNIC("usb0", "10.1.2.1/24")
		start(t, h, "0")
		if err := os.WriteFile(filepath.Join(h.runDir, "tailscaled.pid"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stop(t, h, "0")
		if sysctl(h, "net.ipv4.ip_forward") != "1" {
			t.Errorf("stop zeroed ip_forward under a running tailscaled")
		}
		if !strings.Contains(h.stateFile("rules"), "iif usb0 unreachable") || strings.Contains(h.stateFile("rules"), "lookup") {
			t.Errorf("rules after stop under tailscaled:\n%s", h.stateFile("rules"))
		}
	})

	t.Run("another slot is up", func(t *testing.T) {
		h := newS94(t)
		h.gadgetNIC("usb0", "10.1.2.1/24")
		start(t, h, "0")
		start(t, h, "1")
		stop(t, h, "0")
		if sysctl(h, "net.ipv4.ip_forward") != "1" {
			t.Errorf("stop of slot 0 zeroed ip_forward under slot 1")
		}
		rules := h.stateFile("rules")
		if !strings.Contains(rules, "1001:\tfrom all iif usb0 unreachable") || strings.Contains(rules, "1000:") || !strings.Contains(rules, "1002:") || !strings.Contains(rules, "1003:") {
			t.Errorf("rules after stopping slot 0 with slot 1 up:\n%s", rules)
		}
		stop(t, h, "1")
		if sysctl(h, "net.ipv4.ip_forward") != "0" {
			t.Errorf("ip_forward after the last slot stopped = %q, want 0", sysctl(h, "net.ipv4.ip_forward"))
		}
		if h.stateFile("rules") != "" {
			t.Errorf("rules left after the last stop:\n%s", h.stateFile("rules"))
		}
	})
}
