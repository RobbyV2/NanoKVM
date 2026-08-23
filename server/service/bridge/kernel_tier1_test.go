//go:build linux && kernelint

package bridge

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/kernelint"
)

const (
	uplinkCIDR  = "10.9.9.2/24"
	gatewayAddr = "10.9.9.1"
	peerNS      = "kernelint-peer"
)

type netFixture struct {
	port    int
	witness *ListenerWitness
	pinger  Pinger
	gadget  Gadget
}

func ip(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("ip %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func newNetFixture(t *testing.T) *netFixture {
	t.Helper()

	dir := t.TempDir()
	for target, value := range map[*string]string{
		&stateDir:      filepath.Join(dir, "network"),
		&uplinkPath:    filepath.Join(dir, "l2-uplink"),
		&gatewayPath:   filepath.Join(dir, "gateway"),
		&resolvPath:    filepath.Join(dir, "resolv.conf"),
		&udhcpcPidPath: filepath.Join(dir, "udhcpc.pid"),
		&noDHCPPath:    filepath.Join(dir, "eth.nodhcp"),
	} {
		previous := *target
		*target = value
		t.Cleanup(func() { *target = previous })
	}
	if err := os.MkdirAll(stateDir, dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(noDHCPPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	// The gateway lives in a second namespace on purpose. ping -I br0 binds to
	// the device, so a gateway address that is local to this namespace is
	// unreachable through it and gate 2 fails for a reason the bridge did not
	// cause.
	ip(t, "link", "set", "lo", "up")
	ip(t, "netns", "add", peerNS)
	t.Cleanup(func() { _ = exec.Command("ip", "netns", "del", peerNS).Run() })
	ip(t, "link", "add", StockUplink, "type", "veth", "peer", "name", "wan0")
	ip(t, "link", "add", GadgetName, "type", "veth", "peer", "name", "host0")
	t.Cleanup(func() {
		for _, dev := range []string{BridgeName, StockUplink, GadgetName} {
			_ = exec.Command("ip", "link", "del", dev).Run()
		}
	})
	ip(t, "link", "set", "wan0", "netns", peerNS)
	ip(t, "netns", "exec", peerNS, "ip", "addr", "add", gatewayAddr+"/24", "dev", "wan0")
	ip(t, "netns", "exec", peerNS, "ip", "link", "set", "wan0", "up")
	ip(t, "addr", "add", uplinkCIDR, "dev", StockUplink)
	for _, dev := range []string{StockUplink, GadgetName, "host0"} {
		ip(t, "link", "set", dev, "up")
	}
	ip(t, "route", "add", "default", "via", gatewayAddr, "dev", StockUplink)

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	port := listener.Addr().(*net.TCPAddr).Port
	return &netFixture{port: port, witness: NewListenerWitness("http", port), pinger: NewPinger(NewCommander())}
}

func (f *netFixture) manager() *Manager {
	return New(Config{
		Scripts:  noopScripts{},
		Firewall: noopFirewall{},
		Pinger:   f.pinger,
		Liveness: f.witness,
		Gadget:   f.gadget,
		Window:   30 * time.Second,
	})
}

type noopScripts struct{}

func (noopScripts) StartEth(context.Context) error { return nil }

type noopFirewall struct{}

func (noopFirewall) Install(context.Context, string) error { return nil }
func (noopFirewall) Remove(context.Context, string) error  { return nil }

type deadPinger struct{}

func (deadPinger) Ping(context.Context, string, string) bool { return false }

func links(t *testing.T) map[string]Link {
	t.Helper()
	all, err := NewIPTool(NewCommander()).Links(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]Link, len(all))
	for _, link := range all {
		byName[link.Name] = link
	}
	return byName
}

func addressesOf(t *testing.T, dev string) []string {
	t.Helper()
	all, err := NewIPTool(NewCommander()).AddrsV4(context.Background(), dev)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(all))
	for _, info := range all {
		out = append(out, info.CIDR())
	}
	return out
}

func TestKernelTier1BridgeEnableMovesTheAddressAndDisableGivesItBack(t *testing.T) {
	kernelint.RequireTier1(t)
	fixture := newNetFixture(t)
	manager := fixture.manager()
	ctx := context.Background()

	stock, err := permanentMAC(StockUplink)
	if err != nil {
		t.Fatal(err)
	}

	rsp, err := manager.Enable(ctx)
	if err != nil {
		t.Fatalf("enable: %v (%s)", err, rsp.Message)
	}
	if rsp.State != proto.BridgeEnabled {
		t.Fatalf("state = %s, message %q, checks %+v", rsp.State, rsp.Message, rsp.Checks)
	}

	live := links(t)
	if live[StockUplink].Master != BridgeName {
		t.Fatalf("%s master = %q", StockUplink, live[StockUplink].Master)
	}
	if live[BridgeName].Address != stock {
		t.Fatalf("%s MAC = %q, want %s's %q", BridgeName, live[BridgeName].Address, StockUplink, stock)
	}
	if got := addressesOf(t, BridgeName); len(got) != 1 || got[0] != uplinkCIDR {
		t.Fatalf("%s addresses = %v", BridgeName, got)
	}
	if got := addressesOf(t, StockUplink); len(got) != 0 {
		t.Fatalf("%s kept %v", StockUplink, got)
	}
	if !rsp.Checks.InboundWeak {
		t.Fatalf("inbound gate %+v, want the SelfConnect fallback", rsp.Checks)
	}
	if ReadUplink() != BridgeName {
		t.Fatalf("uplink file = %q", ReadUplink())
	}

	rsp, err = manager.Disable(ctx)
	if err != nil {
		t.Fatalf("disable: %v (%s)", err, rsp.Message)
	}
	if rsp.State != proto.BridgeDisabled {
		t.Fatalf("state = %s, message %q, checks %+v", rsp.State, rsp.Message, rsp.Checks)
	}
	if _, present := links(t)[BridgeName]; present {
		t.Fatalf("%s survived disable", BridgeName)
	}
	if got := addressesOf(t, StockUplink); len(got) != 1 || got[0] != uplinkCIDR {
		t.Fatalf("%s addresses = %v", StockUplink, got)
	}
	if ReadUplink() != StockUplink {
		t.Fatalf("uplink file = %q", ReadUplink())
	}
}

// The dead-man half: gate 2 fails after eth0 has already been flushed and
// enslaved, and the restore has to undo netlink state it did not record itself.
func TestKernelTier1BridgeRollbackReleasesTheRealPort(t *testing.T) {
	kernelint.RequireTier1(t)
	fixture := newNetFixture(t)
	fixture.pinger = deadPinger{}
	manager := fixture.manager()

	// A clean rollback is not an error: the transaction did what it promised
	// and the state names the outcome.
	rsp, err := manager.Enable(context.Background())
	if err != nil {
		t.Fatalf("rollback itself failed: %v (%s)", err, rsp.Message)
	}
	if rsp.State != proto.BridgeRolledBack {
		t.Fatalf("state = %s, message %q", rsp.State, rsp.Message)
	}
	if !rsp.Checks.Address || rsp.Checks.Gateway {
		t.Fatalf("checks = %+v, want the gateway gate to be the one that failed", rsp.Checks)
	}

	live := links(t)
	if _, present := live[BridgeName]; present {
		t.Fatalf("%s survived the rollback", BridgeName)
	}
	if master := live[StockUplink].Master; master != "" {
		t.Fatalf("%s is still enslaved to %q", StockUplink, master)
	}
	if got := addressesOf(t, StockUplink); len(got) != 1 || got[0] != uplinkCIDR {
		t.Fatalf("%s addresses = %v after rollback", StockUplink, got)
	}
	if ReadUplink() != StockUplink {
		t.Fatalf("uplink file = %q after rollback", ReadUplink())
	}

	pending, err := manager.Store().Pending()
	if err != nil {
		t.Fatal(err)
	}
	if pending != nil {
		t.Fatalf("marker %+v still armed after the rollback", pending)
	}
}

// The gadget port is the half that has to survive a presentation apply. It runs
// against a real veth so the enslave and the release are netlink, not a recorder.
func TestKernelTier1BridgeReattachesTheGadgetPort(t *testing.T) {
	kernelint.RequireTier1(t)
	fixture := newNetFixture(t)
	fixture.gadget = staticGadget(GadgetName)
	manager := fixture.manager()
	ctx := context.Background()

	if rsp, err := manager.Enable(ctx); err != nil {
		t.Fatalf("enable: %v (%s)", err, rsp.Message)
	}
	if master := links(t)[GadgetName].Master; master != BridgeName {
		t.Fatalf("%s master = %q after enable", GadgetName, master)
	}

	ip(t, "link", "set", GadgetName, "nomaster")
	if master := links(t)[GadgetName].Master; master != "" {
		t.Fatalf("%s is still enslaved to %q", GadgetName, master)
	}

	manager.ReattachGadget(ctx)
	if master := links(t)[GadgetName].Master; master != BridgeName {
		t.Fatalf("%s master = %q after ReattachGadget", GadgetName, master)
	}

	if rsp, err := manager.Disable(ctx); err != nil {
		t.Fatalf("disable: %v (%s)", err, rsp.Message)
	}
	if master := links(t)[GadgetName].Master; master != "" {
		t.Fatalf("%s master = %q after disable", GadgetName, master)
	}
}

type staticGadget string

func (g staticGadget) NIC(context.Context) (string, error)             { return string(g), nil }
func (g staticGadget) NetworkProtocol(context.Context) (string, error) { return "ncm", nil }
func (staticGadget) OnRebind(func(context.Context))                    {}
