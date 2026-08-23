package config

import (
	"os"
	"path/filepath"
	"testing"
)

// A configuration written before VNC existed has no vnc section at all, so the
// round trip has to supply the port rather than persist a zero that would make
// the server listen on an arbitrary one.
func TestReadWriteVNCRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yaml")
	original := ConfigurationFile
	ConfigurationFile = path
	t.Cleanup(func() { ConfigurationFile = original })

	legacy := "proto: http\nport:\n  http: 80\n  https: 443\njwt:\n  secretKey: keep-me\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy config: %s", err)
	}

	conf, err := Read()
	if err != nil {
		t.Fatalf("read: %s", err)
	}
	if conf.VNC.Enabled {
		t.Fatalf("vnc enabled = true, want false")
	}
	if conf.VNC.Port != 5900 {
		t.Fatalf("vnc port = %d, want 5900", conf.VNC.Port)
	}

	conf.VNC.Enabled = true
	if err := Write(conf); err != nil {
		t.Fatalf("write: %s", err)
	}

	reread, err := Read()
	if err != nil {
		t.Fatalf("re-read: %s", err)
	}
	if !reread.VNC.Enabled {
		t.Fatalf("vnc enabled = false, want true")
	}
	if reread.VNC.Port != 5900 {
		t.Fatalf("vnc port = %d, want 5900", reread.VNC.Port)
	}
	if reread.Proto != "http" || reread.Port.Http != 80 || reread.Port.Https != 443 {
		t.Fatalf("unrelated settings lost: %+v", reread)
	}
	if reread.JWT.SecretKey != "keep-me" {
		t.Fatalf("jwt secret = %q, want %q", reread.JWT.SecretKey, "keep-me")
	}
}
