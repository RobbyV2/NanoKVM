package bridge

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

// The exact shapes iproute2 emits, kept verbatim rather than hand-simplified.
// A struct tag that does not match one of these decodes to a zero value in
// silence, and the first place that shows up is a restore that puts nothing
// back, on a device nobody can reach to look at it.
const (
	realLinkJSON = `[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,` +
		`"qdisc":"noqueue","operstate":"UNKNOWN","group":"default","txqlen":1000,` +
		`"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},` +
		`{"ifindex":2,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,` +
		`"qdisc":"pfifo_fast","master":"br0","operstate":"UP","group":"default","txqlen":1000,` +
		`"link_type":"ether","address":"3e:7c:1a:2b:3c:4d","broadcast":"ff:ff:ff:ff:ff:ff"}]`

	realAddrJSON = `[{"ifindex":2,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],` +
		`"mtu":1500,"qdisc":"pfifo_fast","operstate":"UP","group":"default","txqlen":1000,` +
		`"link_type":"ether","address":"3e:7c:1a:2b:3c:4d","broadcast":"ff:ff:ff:ff:ff:ff",` +
		`"addr_info":[{"family":"inet","local":"192.168.1.50","prefixlen":24,` +
		`"broadcast":"192.168.1.255","scope":"global","label":"eth0",` +
		`"valid_life_time":4294967295,"preferred_life_time":4294967295},` +
		`{"family":"inet6","local":"fe80::3c7c:1aff:fe2b:3c4d","prefixlen":64,"scope":"link",` +
		`"valid_life_time":4294967295,"preferred_life_time":4294967295}]}]`

	realRouteJSON = `[{"dst":"default","gateway":"192.168.1.1","dev":"br0","protocol":"dhcp",` +
		`"prefsrc":"192.168.1.50","metric":100,"flags":[]},` +
		`{"dst":"192.168.1.0/24","dev":"br0","protocol":"kernel","scope":"link",` +
		`"prefsrc":"192.168.1.50","flags":[]}]`
)

func TestDecodeRealIPOutput(t *testing.T) {
	var links []Link
	if err := json.Unmarshal([]byte(realLinkJSON), &links); err != nil {
		t.Fatalf("decode links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("decoded %d links, want 2", len(links))
	}
	if links[1].Name != "eth0" || links[1].Master != "br0" || links[1].Address != testMAC {
		t.Fatalf("eth0 decoded as %+v", links[1])
	}
	if !links[1].Up() || !links[1].Running() {
		t.Fatalf("eth0 flags %v did not decode into Up/Running", links[1].Flags)
	}

	var addrs []Addr
	if err := json.Unmarshal([]byte(realAddrJSON), &addrs); err != nil {
		t.Fatalf("decode addrs: %v", err)
	}
	if len(addrs[0].AddrInfo) != 2 {
		t.Fatalf("decoded %d addr_info entries, want 2", len(addrs[0].AddrInfo))
	}
	if got := addrs[0].AddrInfo[0].CIDR(); got != "192.168.1.50/24" {
		t.Fatalf("CIDR = %q", got)
	}

	var routes []Route
	if err := json.Unmarshal([]byte(realRouteJSON), &routes); err != nil {
		t.Fatalf("decode routes: %v", err)
	}
	if routes[0].Dst != "default" || routes[0].Gateway != "192.168.1.1" || routes[0].Dev != "br0" {
		t.Fatalf("default route decoded as %+v", routes[0])
	}
}

// TestSnapshotRoundTrip is the property the whole rollback rests on: what comes
// back off disk is what went onto it. Every field is populated, because a field
// that is dropped in serialisation is a field the restore silently does not put
// back.
func TestSnapshotRoundTrip(t *testing.T) {
	h := newHarness(t)

	original := &Snapshot{
		CapturedAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
		Links: []Link{
			{Index: 2, Name: "eth0", Flags: []string{"UP", "LOWER_UP"}, MTU: 1500,
				Address: testMAC, OperState: "UP", LinkType: "ether"},
			{Index: 3, Name: "usb0", Flags: []string{"UP"}, MTU: 1500,
				Master: "br0", Address: "48:da:35:6e:11:22", OperState: "UP", LinkType: "ether"},
		},
		Addrs: []Addr{
			{Index: 2, Name: "eth0", AddrInfo: []AddrInfo{
				{Family: "inet", Local: "192.168.1.50", PrefixLen: 24,
					Scope: "global", Broadcast: "192.168.1.255"},
				{Family: "inet6", Local: "fe80::1", PrefixLen: 64, Scope: "link"},
			}},
		},
		Routes: []Route{
			{Dst: "default", Gateway: "192.168.1.1", Dev: "eth0", Protocol: "dhcp", Metric: 100},
			{Dst: "192.168.1.0/24", Dev: "eth0", Protocol: "kernel", PrefSrc: "192.168.1.50"},
		},
		ResolvConf:        "nameserver 192.168.1.1\nsearch lan\n",
		ResolvConfPresent: true,
		Gateway:           "192.168.1.1\n",
		GatewayPresent:    true,
		Uplink:            "br0",
		UplinkPresent:     true,
		StaticPath:        true,
	}

	path, err := h.store.WriteSnapshot(original)
	if err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	restored, err := h.store.ReadSnapshot(path)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Fatalf("round trip differs.\n got: %+v\nwant: %+v", restored, original)
	}

	// The record is the only thing standing between a failed apply and physical
	// access, so it is not world-readable.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("snapshot mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestReadSnapshotOfAMissingFile(t *testing.T) {
	h := newHarness(t)

	if _, err := h.store.ReadSnapshot(h.store.SnapshotPath()); err == nil {
		t.Fatal("ReadSnapshot of a missing file succeeded")
	}
}

func TestCaptureRecordsEverything(t *testing.T) {
	h := newHarness(t)
	writeFile(t, gatewayPath, "192.168.1.1\n")
	writeFile(t, noDHCPPath, "192.168.1.50/24 192.168.1.1\n")

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	if len(snapshot.Links) == 0 || len(snapshot.Addrs) == 0 || len(snapshot.Routes) == 0 {
		t.Fatalf("capture is missing an ip section: %+v", snapshot)
	}
	if !snapshot.ResolvConfPresent || snapshot.ResolvConf == "" {
		t.Error("resolv.conf was not captured")
	}
	if !snapshot.GatewayPresent || snapshot.Gateway != "192.168.1.1\n" {
		t.Errorf("gateway captured as %q, %v", snapshot.Gateway, snapshot.GatewayPresent)
	}
	if !snapshot.StaticPath {
		t.Error("the static branch was not detected from /boot/eth.nodhcp")
	}

	// An absent l2-uplink is the stock state and must be recorded as absent
	// rather than as the string "eth0": a restore writes the file back only when
	// it was there, so that a disabled device is byte-identical to one that
	// never bridged.
	if snapshot.UplinkPresent {
		t.Error("an absent l2-uplink was captured as present")
	}
	if got := snapshot.UplinkName(); got != StockUplink {
		t.Errorf("UplinkName = %q, want eth0", got)
	}
}

func TestCaptureRecordsAnExistingUplink(t *testing.T) {
	h := newHarness(t)
	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !snapshot.UplinkPresent || snapshot.Uplink != BridgeName {
		t.Fatalf("uplink captured as %q, %v", snapshot.Uplink, snapshot.UplinkPresent)
	}
}

// IPv4 filters inet6, which is the same bug as S30eth:59's "grep inet".
func TestSnapshotIPv4IgnoresIPv6(t *testing.T) {
	snapshot := &Snapshot{Addrs: []Addr{
		{Name: "eth0", AddrInfo: []AddrInfo{
			{Family: "inet6", Local: "fe80::1", PrefixLen: 64},
			{Family: "inet", Local: "192.168.1.50", PrefixLen: 24},
			{Family: "inet", Local: "10.0.0.5", PrefixLen: 8},
		}},
		{Name: "wlan0", AddrInfo: []AddrInfo{{Family: "inet", Local: "10.9.9.1", PrefixLen: 24}}},
	}}

	got := snapshot.IPv4("eth0")
	if len(got) != 2 || got[0].Local != "192.168.1.50" || got[1].Local != "10.0.0.5" {
		t.Fatalf("IPv4(eth0) = %+v", got)
	}

	// A device carrying nothing but a link-local reports no IPv4 at all.
	only6 := &Snapshot{Addrs: []Addr{
		{Name: "br0", AddrInfo: []AddrInfo{{Family: "inet6", Local: "fe80::1", PrefixLen: 64}}},
	}}
	if got := only6.IPv4("br0"); len(got) != 0 {
		t.Fatalf("IPv4 on a link-local-only device = %+v", got)
	}
}

func TestSnapshotQueries(t *testing.T) {
	snapshot := &Snapshot{
		Links: []Link{
			{Name: "eth0", Master: "br0"},
			{Name: "usb0"},
		},
		Routes: []Route{
			{Dst: "192.168.1.0/24", Dev: "br0"},
			{Dst: "default", Gateway: "192.168.1.1", Dev: "br0"},
		},
		ResolvConf: "search lan\nnameserver 192.168.1.1\nnameserver 8.8.8.8\n",
	}

	if master, ok := snapshot.Master("eth0"); !ok || master != "br0" {
		t.Errorf("Master(eth0) = %q, %v", master, ok)
	}
	if _, ok := snapshot.Master("usb0"); ok {
		t.Error("Master(usb0) reported a master for a device with none")
	}
	if _, ok := snapshot.Master("nope"); ok {
		t.Error("Master of an absent device reported a master")
	}

	route, ok := snapshot.DefaultRoute()
	if !ok || route.Gateway != "192.168.1.1" {
		t.Errorf("DefaultRoute = %+v, %v", route, ok)
	}
	if _, ok := snapshot.DefaultRouteVia("eth0"); ok {
		t.Error("DefaultRouteVia(eth0) matched a route through br0")
	}

	if !snapshot.HasNameserver("192.168.1.1") {
		t.Error("HasNameserver missed a line that is there")
	}
	if snapshot.HasNameserver("1.1.1.1") {
		t.Error("HasNameserver matched a line that is not there")
	}
}

// TestRestorePutsTheFilesBackFirst covers the ordering the restore depends on:
// S30eth reads l2-uplink to decide which device to address, so restoring the
// file after re-running the script would address the wrong one.
func TestRestorePutsTheFilesBackFirst(t *testing.T) {
	h := newHarness(t)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// Move the device to the post-enable state.
	if err := WriteUplink(BridgeName); err != nil {
		t.Fatalf("WriteUplink: %v", err)
	}
	writeFile(t, gatewayPath, "10.0.0.1\n")
	writeFile(t, resolvPath, "nameserver 10.0.0.1\nnameserver 10.0.0.2\n")
	h.net.links[BridgeName] = &Link{Index: 90, Name: BridgeName, Address: testMAC}
	h.net.links[StockUplink].Master = BridgeName

	var uplinkAtStartEth string
	h.net.onCall = func(line string) {
		if line == S30ethScript+" start" {
			uplinkAtStartEth = ReadUplink()
		}
	}

	if err := h.mgr.Restore(context.Background(), snapshot); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if uplinkAtStartEth != StockUplink {
		t.Errorf("S30eth ran with uplink %q, want eth0: the file was restored late",
			uplinkAtStartEth)
	}
	if _, err := os.Stat(uplinkPath); !os.IsNotExist(err) {
		t.Error("the l2-uplink file was written back where the capture shows none")
	}
	if _, ok := ReadGateway(); ok {
		t.Error("the gateway file was written back where the capture shows none")
	}
	if resolv, _ := readFile(resolvPath); resolv != "nameserver 192.168.1.1\n" {
		t.Errorf("resolv.conf = %q, want the captured content", resolv)
	}
	if master := h.net.links[StockUplink].Master; master != "" {
		t.Errorf("eth0 is still enslaved to %q", master)
	}
	if _, exists := h.net.links[BridgeName]; exists {
		t.Error("br0 was not deleted")
	}
}

// A restore of a capture that already had the bridge leaves br0 alone: a failed
// disable has to land back on a device that still exists.
func TestRestoreKeepsABridgeTheCaptureHad(t *testing.T) {
	h := newHarness(t)
	enabled(t, h)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := h.mgr.Restore(context.Background(), snapshot); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if _, exists := h.net.links[BridgeName]; !exists {
		t.Fatal("a restore of a capture that had br0 deleted it")
	}
	if got := ReadUplink(); got != BridgeName {
		t.Fatalf("uplink = %q, want br0", got)
	}
	notInTrace(t, h.net.trace(), "ip link delete br0")
}

// The DHCP branch at S30eth:68 is backgrounded with no fallback address of any
// kind, so a rollback whose udhcpc never takes a lease would otherwise leave
// the device with no IPv4 and nothing that retries. The captured address is
// replayed only in that case.
func TestRestoreReplaysTheAddressWhenDHCPProducesNothing(t *testing.T) {
	h := newHarness(t)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	h.net.dhcpFails = true
	delete(h.net.addrs, StockUplink)

	if err := h.mgr.Restore(context.Background(), snapshot); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	addrs := h.net.addrs[StockUplink]
	if len(addrs) != 1 || addrs[0].Local != "192.168.1.50" {
		t.Fatalf("eth0 addresses after restore = %+v, want the captured 192.168.1.50", addrs)
	}
}

func TestRestoreDoesNotReplayWhenDHCPSucceeds(t *testing.T) {
	h := newHarness(t)

	snapshot, err := Capture(context.Background(), h.mgr.ip)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := h.mgr.Restore(context.Background(), snapshot); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	notInTrace(t, h.net.trace(), "-batch")
	if addrs := h.net.addrs[StockUplink]; len(addrs) != 1 {
		t.Fatalf("eth0 has %d addresses after restore, want the single lease", len(addrs))
	}
}
