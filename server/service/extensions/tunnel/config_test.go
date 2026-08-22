package tunnel

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"NanoKVM-Server/proto"
)

func useTestConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := configDir
	configDir = dir
	t.Cleanup(func() { configDir = old })
	return dir
}

func TestConfigRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		tunnel proto.TunnelName
		cfg    Config
	}{
		{
			name:   "empty",
			tunnel: proto.TunnelWstunnel,
			cfg:    Config{},
		},
		{
			name:   "args only",
			tunnel: proto.TunnelWstunnel,
			cfg:    Config{Args: "client -L tcp://0.0.0.0:8080 wss://example.org"},
		},
		{
			name:   "args and env",
			tunnel: proto.TunnelNewt,
			cfg: Config{
				Args: "--accept-clients",
				Env: map[string]string{
					"PANGOLIN_ENDPOINT": "https://example.org",
					"NEWT_SECRET":       "it's a secret",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := useTestConfigDir(t)

			if err := saveConfig(tt.tunnel, tt.cfg); err != nil {
				t.Fatalf("save config: %v", err)
			}

			path := filepath.Join(dir, string(tt.tunnel)+".json")
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat config: %v", err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
			}

			got, err := loadConfig(tt.tunnel)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if got.Args != tt.cfg.Args {
				t.Fatalf("args = %q, want %q", got.Args, tt.cfg.Args)
			}
			if len(got.Env) != len(tt.cfg.Env) || (len(tt.cfg.Env) > 0 && !reflect.DeepEqual(got.Env, tt.cfg.Env)) {
				t.Fatalf("env = %+v, want %+v", got.Env, tt.cfg.Env)
			}
		})
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	tests := []proto.TunnelName{proto.TunnelWstunnel, proto.TunnelNewt}

	for _, tunnel := range tests {
		t.Run(string(tunnel), func(t *testing.T) {
			useTestConfigDir(t)

			cfg, err := loadConfig(tunnel)
			if err != nil {
				t.Fatalf("load missing config: %v", err)
			}
			if !reflect.DeepEqual(cfg, Config{}) {
				t.Fatalf("cfg = %+v, want zero value", cfg)
			}
		})
	}
}

func TestLoadConfigRejectsCorruptJSON(t *testing.T) {
	dir := useTestConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "newt.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(proto.TunnelNewt); err == nil {
		t.Fatal("expected corrupt config error")
	}
}

func TestSaveConfigReplacesExisting(t *testing.T) {
	useTestConfigDir(t)

	if err := saveConfig(proto.TunnelNewt, Config{Args: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := saveConfig(proto.TunnelNewt, Config{Args: "second"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(proto.TunnelNewt)
	if err != nil || cfg.Args != "second" {
		t.Fatalf("cfg = %+v err = %v", cfg, err)
	}
}
