package exit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func useTempDirs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	old := []string{ConfigDir, BinDir, SeedDir, InitSeedDir, InitDir, RunDir, LogDir}
	ConfigDir = filepath.Join(root, "etc/kvm/exit")
	BinDir = filepath.Join(root, "etc/kvm/bin")
	SeedDir = filepath.Join(root, "kvmapp/exit")
	InitSeedDir = filepath.Join(root, "kvmapp/system/init.d")
	InitDir = filepath.Join(root, "etc/init.d")
	RunDir = filepath.Join(root, "var/run")
	LogDir = filepath.Join(root, "tmp")
	t.Cleanup(func() {
		ConfigDir, BinDir, SeedDir, InitSeedDir, InitDir, RunDir, LogDir = old[0], old[1], old[2], old[3], old[4], old[5], old[6]
	})
	return root
}

func TestTokenAlphabetAndLength(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		token, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(token) != TokenLength {
			t.Fatalf("token %q has length %d, want %d", token, len(token), TokenLength)
		}
		for _, r := range token {
			if !strings.ContainsRune(tokenAlphabet, r) {
				t.Fatalf("token %q carries %q, outside the alphabet", token, r)
			}
			if strings.ContainsRune("0o1li", r) {
				t.Fatalf("token %q carries ambiguous symbol %q", token, r)
			}
		}
		if !ValidToken(token) {
			t.Fatalf("ValidToken(%q) = false", token)
		}
		if seen[token] {
			t.Fatalf("token %q drawn twice in 200 draws", token)
		}
		seen[token] = true
	}
	if ValidToken("k7m2p9v") || ValidToken("k7m2p9vx0") || ValidToken("K7M2P9VX") {
		t.Fatal("ValidToken accepted a token outside the alphabet or length")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	useTempDirs(t)
	slot := MustSlot("0")

	cfg, err := DefaultConfig(slot)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.Pending || cfg.Mode != proto.ExitModeNative || cfg.MTU != DefaultMTU || cfg.NIC != NICGadget {
		t.Fatalf("default config = %+v", cfg)
	}
	if got := cfg.Upstreams(); len(got) != 2 || got[0].String() != "1.1.1.1" {
		t.Fatalf("default upstreams = %v", got)
	}

	cfg.Enabled = true
	cfg.Mode = proto.ExitModeWstunnel
	cfg.AllowPrivate = true
	cfg.DNS = []string{"9.9.9.9", "bogus", ""}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(slot.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}

	back, ok, err := LoadConfig(slot)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if back.Token != cfg.Token || !back.Enabled || back.Mode != proto.ExitModeWstunnel || !back.AllowPrivate {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if got := back.Upstreams(); len(got) != 1 || got[0].String() != "9.9.9.9" {
		t.Fatalf("upstreams after round trip = %v", got)
	}
	if !back.Policy().AllowPrivate {
		t.Fatal("policy does not follow allowPrivate")
	}

	if _, ok, err := LoadConfig(MustSlot("3")); err != nil || ok {
		t.Fatalf("absent slot: ok=%v err=%v", ok, err)
	}

	all, err := LoadAll()
	if err != nil || len(all) != 1 || all[0].Slot != "0" {
		t.Fatalf("LoadAll = %+v, %v", all, err)
	}
	if slot, ok := AnyEnabled(); !ok || slot != "0" {
		t.Fatalf("AnyEnabled = %q, %v", slot, ok)
	}

	back.Enabled = false
	if err := SaveConfig(back); err != nil {
		t.Fatal(err)
	}
	if _, ok := AnyEnabled(); ok {
		t.Fatal("AnyEnabled after disable")
	}
}

func TestSaveConfigRefusesAForeignToken(t *testing.T) {
	useTempDirs(t)
	cfg, _ := DefaultConfig(MustSlot("0"))
	cfg.Token = "not-a-token"
	if err := SaveConfig(cfg); err == nil {
		t.Fatal("saved a config with a token outside the alphabet")
	}
}

func TestStateAndMarkerRoundTrip(t *testing.T) {
	useTempDirs(t)
	slot := MustSlot("0")

	st, err := LoadState(slot)
	if err != nil || st.LastConnectedAt != nil {
		t.Fatalf("absent state = %+v, %v", st, err)
	}
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	st.LastConnectedAt = &now
	st.PreviousPeer = &proto.ExitPeer{Addr: "203.0.113.7", Transport: proto.ExitModeNative}
	if err := SaveState(slot, st); err != nil {
		t.Fatal(err)
	}
	back, err := LoadState(slot)
	if err != nil || back.LastConnectedAt == nil || !back.LastConnectedAt.Equal(now) || back.PreviousPeer.Addr != "203.0.113.7" {
		t.Fatalf("state round trip = %+v, %v", back, err)
	}

	if err := WriteGadgetRoute(slot); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(GadgetRoutePath())
	if err != nil || strings.TrimSpace(string(data)) != "0" {
		t.Fatalf("marker = %q, %v", data, err)
	}
	// Another slot never removes a marker it does not own.
	if err := RemoveGadgetRoute(MustSlot("1")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GadgetRoutePath()); err != nil {
		t.Fatal("marker removed by a slot that does not own it")
	}
	if err := RemoveGadgetRoute(slot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GadgetRoutePath()); !os.IsNotExist(err) {
		t.Fatal("marker still present after the owner removed it")
	}
	if err := RemoveGadgetRoute(slot); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}
