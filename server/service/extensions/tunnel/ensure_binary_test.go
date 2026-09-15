package tunnel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBinaryFromTracksTheSeed(t *testing.T) {
	bins, seeds := useTestBinDirs(t)
	seed := filepath.Join(seeds, "hev-socks5-tunnel.gz")
	if err := os.WriteFile(seed, gzipBytes(t, []byte("v1")), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := EnsureBinaryFrom("hev-socks5-tunnel", seed)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(bins, "hev-socks5-tunnel") {
		t.Fatalf("path = %s", path)
	}
	if got, _ := os.ReadFile(path); string(got) != "v1" {
		t.Fatalf("extracted %q", got)
	}
	record := filepath.Join(bins, ".hev-socks5-tunnel.seed")
	if _, err := os.Stat(record); err != nil {
		t.Fatalf("no seed record: %v", err)
	}

	// Same seed: the binary is left alone even if it was modified in place.
	if err := os.WriteFile(path, []byte("touched"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureBinaryFrom("hev-socks5-tunnel", seed); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "touched" {
		t.Fatal("an unchanged seed re-extracted the binary")
	}

	// New seed: re-extracted once.
	if err := os.WriteFile(seed, gzipBytes(t, []byte("v2")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureBinaryFrom("hev-socks5-tunnel", seed); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "v2" {
		t.Fatalf("new seed not extracted: %q", got)
	}

	// Custom binary: never replaced, whatever the seed says.
	if err := os.WriteFile(filepath.Join(bins, ".hev-socks5-tunnel.custom"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seed, gzipBytes(t, []byte("v3")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureBinaryFrom("hev-socks5-tunnel", seed); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "v2" {
		t.Fatalf("custom binary replaced: %q", got)
	}

	// Missing seed with a present binary is fine; with none it is an error.
	if _, err := EnsureBinaryFrom("hev-socks5-tunnel", filepath.Join(seeds, "nope.gz")); err != nil {
		t.Fatalf("present binary, absent seed: %v", err)
	}
	if _, err := EnsureBinaryFrom("other", filepath.Join(seeds, "nope.gz")); err == nil {
		t.Fatal("absent binary and seed returned no error")
	}
	if _, err := EnsureBinaryFrom("../escape", seed); err == nil {
		t.Fatal("path traversal in the name accepted")
	}
}

// EnsureBinary through the tunnel seed dir is what binaryPath now is.
func TestEnsureBinaryUsesTheTunnelSeedDir(t *testing.T) {
	bins, seeds := useTestBinDirs(t)
	if err := os.WriteFile(filepath.Join(seeds, "wstunnel.gz"), gzipBytes(t, []byte("ws")), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := EnsureBinary("wstunnel")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(bins, "wstunnel") {
		t.Fatalf("path = %s", path)
	}
}
