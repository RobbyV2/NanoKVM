package exit

import (
	"os"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"gopkg.in/yaml.v3"
)

func testConfig(slot Slot) Config {
	return Config{
		Slot: slot.ID, Enabled: true, Mode: proto.ExitModeNative, Token: "k7m2p9vx",
		NIC: NICGadget, DNS: []string{"1.1.1.1", "8.8.8.8"}, MTU: 1280,
	}
}

func TestRenderHevConfigKeys(t *testing.T) {
	slot := MustSlot("0")
	out := RenderHevConfig(slot, testConfig(slot))

	var doc struct {
		Tunnel map[string]any `yaml:"tunnel"`
		Socks5 map[string]any `yaml:"socks5"`
		Misc   map[string]any `yaml:"misc"`
	}
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("hev.yml does not parse: %v\n%s", err, out)
	}
	wantTunnel := map[string]any{"name": "exit0", "mtu": 1280, "ipv4": "198.18.0.1"}
	wantSocks := map[string]any{"port": 10800, "address": "127.0.0.1", "udp": "udp"}
	wantMisc := map[string]any{
		"log-file": "/tmp/exit0-hev.log", "log-level": "warn", "connect-timeout": 12000,
		"tcp-buffer-size": 16384, "udp-recv-buffer-size": 65536, "task-stack-size": 20480,
		"max-session-count": 256,
	}
	for name, pair := range map[string][2]map[string]any{
		"tunnel": {doc.Tunnel, wantTunnel}, "socks5": {doc.Socks5, wantSocks}, "misc": {doc.Misc, wantMisc},
	} {
		got, want := pair[0], pair[1]
		if len(got) != len(want) {
			t.Fatalf("%s has keys %v, want exactly %v", name, keys(got), keys(want))
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("%s.%s = %v (%T), want %v (%T)", name, k, got[k], got[k], v, v)
			}
		}
	}
	// pid-file must never appear: start-stop-daemon owns the pidfile (D8).
	if strings.Contains(out, "pid-file") {
		t.Fatal("hev.yml sets pid-file")
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRenderRestrictYAML(t *testing.T) {
	slot := MustSlot("0")
	out := RenderRestrict(slot, testConfig(slot))

	for _, want := range []string{
		"restrictions:",
		"  - name: exit0",
		`      - !PathPrefix "^exit0$"`,
		`      - !Authorization "^Bearer k7m2p9vx$"`,
		"      - !ReverseTunnel",
		"        protocol: [Socks5]",
		"        port: [10820]",
		"        cidr: [127.0.0.1/32]",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("restrict yaml lacks line %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "!Tunnel") {
		t.Fatal("restrict yaml allows a forward tunnel")
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(out), &node); err != nil {
		t.Fatalf("restrict yaml does not parse: %v", err)
	}
	var tags []string
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if strings.HasPrefix(n.Tag, "!") && !strings.HasPrefix(n.Tag, "!!") {
			tags = append(tags, n.Tag)
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(&node)
	if strings.Join(tags, " ") != "!PathPrefix !Authorization !ReverseTunnel" {
		t.Fatalf("custom tags = %v", tags)
	}
}

func TestRenderEnv(t *testing.T) {
	slot := MustSlot("2")
	cfg := testConfig(slot)
	cfg.Mode = proto.ExitModeWstunnel
	cfg.Pending = true
	cfg.Enabled = false
	cfg.AllowPrivate = true
	out := RenderEnv(slot, cfg, "usb1")

	for _, want := range []string{
		"SLOT=2", "ENABLED=0", "PENDING=1", "MODE=wstunnel", "ALLOW_PRIVATE=1", "MTU=1280", "NIC=usb1",
		`REJECT4="0.0.0.0/8 127.0.0.0/8 169.254.0.0/16 198.18.0.0/15 224.0.0.0/3"`,
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("env lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, cfg.Token) {
		t.Fatal("env carries the token")
	}

	cfg.AllowPrivate = false
	out = RenderEnv(slot, cfg, "usb0; rm -rf /")
	if !strings.Contains(out, "NIC=\n") {
		t.Fatalf("unsafe NIC name reached the env:\n%s", out)
	}
	if !strings.Contains(out, "10.0.0.0/8 100.64.0.0/10 172.16.0.0/12 192.168.0.0/16\"") {
		t.Fatalf("private prefixes missing with allowPrivate=false:\n%s", out)
	}
}

func TestWriteSlotFiles(t *testing.T) {
	useTempDirs(t)
	slot := MustSlot("0")
	if err := writeSlotFiles(slot, testConfig(slot), "usb0"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{slot.HevConfigPath(), slot.RestrictPath(), slot.EnvPath()} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", path, info.Mode().Perm())
		}
	}
}
