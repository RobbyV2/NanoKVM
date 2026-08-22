package tunnel

import (
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

func useTestBinDirs(t *testing.T) (string, string) {
	t.Helper()
	bins := t.TempDir()
	seeds := t.TempDir()
	oldBin, oldSeed := binDir, seedDir
	binDir, seedDir = bins, seeds
	t.Cleanup(func() { binDir, seedDir = oldBin, oldSeed })
	return bins, seeds
}

func gzipBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{name: "blank", input: "   \t ", want: nil},
		{name: "bare words", input: "client --foreground wss://example.org", want: []string{"client", "--foreground", "wss://example.org"}},
		{name: "collapsed spaces", input: "  a   b  ", want: []string{"a", "b"}},
		{name: "single quotes", input: `'hello world'`, want: []string{"hello world"}},
		{name: "double quotes", input: `"hello world"`, want: []string{"hello world"}},
		{name: "empty single quotes", input: `''`, want: []string{""}},
		{name: "quote inside single", input: `'it"s'`, want: []string{`it"s`}},
		{name: "quote inside double", input: `"it's"`, want: []string{"it's"}},
		{name: "escaped space", input: `a\ b`, want: []string{"a b"}},
		{name: "escaped quote", input: `a\'b`, want: []string{"a'b"}},
		{name: "escape inside double", input: `"a\"b"`, want: []string{`a"b`}},
		{name: "literal backslash inside double", input: `"a\nb"`, want: []string{`a\nb`}},
		{name: "adjacent quotes", input: `--flag="value with spaces"`, want: []string{"--flag=value with spaces"}},
		{name: "concatenated quotes", input: `'a'"b"c`, want: []string{"abc"}},
		{name: "unbalanced single", input: `'abc`, wantErr: true},
		{name: "unbalanced double", input: `"abc`, wantErr: true},
		{name: "unbalanced trailing", input: `a b'`, wantErr: true},
		{name: "trailing backslash", input: `a \`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("tokenize(%q) = %+v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("tokenize(%q): %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("tokenize(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "value", want: `'value'`},
		{name: "spaces", input: "a b", want: `'a b'`},
		{name: "empty", input: "", want: `''`},
		{name: "single quote", input: "it's", want: `'it'\''s'`},
		{name: "quote only", input: "'", want: `''\'''`},
		{name: "injection attempt", input: `'; rm -rf /; echo '`, want: `''\''; rm -rf /; echo '\'''`},
		{name: "double quote", input: `say "hi"`, want: `'say "hi"'`},
		{name: "dollar", input: "$HOME", want: `'$HOME'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Fatalf("shellQuote(%q) = %s, want %s", tt.input, got, tt.want)
			}
			if readBack := shellRead(t, "printf %s "+got); readBack != tt.input {
				t.Fatalf("shell read back %q, want %q", readBack, tt.input)
			}
		})
	}
}

func TestRenderWrapperQuotesEnvValue(t *testing.T) {
	cfg := Config{Env: map[string]string{"FOO": "it's"}}

	content, err := renderWrapper(proto.TunnelWstunnel, cfg, "/etc/kvm/bin/wstunnel")
	if err != nil {
		t.Fatalf("render wrapper: %v", err)
	}

	const want = `export FOO='it'\''s'`
	if !hasLine(content, want) {
		t.Fatalf("wrapper = %q, want line %s", content, want)
	}
	if got := shellRead(t, want+`; printf %s "$FOO"`); got != "it's" {
		t.Fatalf("shell read back %q, want %q", got, "it's")
	}
}

func TestRenderWrapperSpec(t *testing.T) {
	tests := []struct {
		name           string
		tunnel         proto.TunnelName
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "newt injects env and args",
			tunnel: proto.TunnelNewt,
			wantContains: []string{
				`export NEWT_SYSTEM_SUBSTRATE='CONTAINER'`,
				`export GOGC='50'`,
				`'--health-file' '/tmp/newt.health'`,
				`'--config-file' '/etc/kvm/newt-client.json'`,
				`exec '/etc/kvm/bin/newt'`,
				`>>'/tmp/newt.log' 2>&1`,
			},
		},
		{
			name:   "wstunnel injects nothing",
			tunnel: proto.TunnelWstunnel,
			wantContains: []string{
				`exec '/etc/kvm/bin/wstunnel' '--foreground'`,
				`>>'/tmp/wstunnel.log' 2>&1`,
			},
			wantNotContain: []string{
				"NEWT_SYSTEM_SUBSTRATE",
				"GOGC",
				"GOMEMLIMIT",
				"--health-file",
				"--config-file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Args: "--foreground", Env: map[string]string{"USER_KEY": "user value"}}
			binary := filepath.Join("/etc/kvm/bin", string(tt.tunnel))

			content, err := renderWrapper(tt.tunnel, cfg, binary)
			if err != nil {
				t.Fatalf("render wrapper: %v", err)
			}
			if !strings.HasPrefix(content, "#!/bin/sh\n") {
				t.Fatalf("wrapper missing shebang: %q", content)
			}
			if !hasLine(content, `export USER_KEY='user value'`) {
				t.Fatalf("wrapper missing user env: %q", content)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Fatalf("wrapper = %q, want %q", content, want)
				}
			}
			for _, unwanted := range tt.wantNotContain {
				if strings.Contains(content, unwanted) {
					t.Fatalf("wrapper = %q, unexpected %q", content, unwanted)
				}
			}
		})
	}
}

func TestRenderWrapperRejectsBadArgs(t *testing.T) {
	if _, err := renderWrapper(proto.TunnelNewt, Config{Args: `'unbalanced`}, "/etc/kvm/bin/newt"); err == nil {
		t.Fatal("expected tokenize error")
	}
	if _, err := renderWrapper(proto.TunnelName("bogus"), Config{}, "/etc/kvm/bin/bogus"); err == nil {
		t.Fatal("expected unknown tunnel error")
	}
}

func TestRenderWrapperRejectsBadEnvKeys(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "statement separator", key: "A=1; id; B", wantErr: true},
		{name: "newline", key: "FOO\nid\nX", wantErr: true},
		{name: "backticks", key: "FOO`id`", wantErr: true},
		{name: "command substitution", key: "FOO$(id)", wantErr: true},
		{name: "space", key: "FOO BAR", wantErr: true},
		{name: "equals", key: "FOO=BAR", wantErr: true},
		{name: "empty", key: "", wantErr: true},
		{name: "plain", key: "FOO"},
		{name: "leading underscore", key: "_FOO"},
		{name: "digits and underscores", key: "FOO_BAR_1"},
		{name: "seeded key", key: "PANGOLIN_ENDPOINT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Env: map[string]string{tt.key: "value"}}

			content, err := renderWrapper(proto.TunnelWstunnel, cfg, "/etc/kvm/bin/wstunnel")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("renderWrapper(%q) = %q, want error", tt.key, content)
				}
				return
			}
			if err != nil {
				t.Fatalf("renderWrapper(%q): %v", tt.key, err)
			}
			if want := "export " + tt.key + "='value'"; !hasLine(content, want) {
				t.Fatalf("wrapper = %q, want line %s", content, want)
			}
		})
	}
}

func TestExtractSeed(t *testing.T) {
	payload := bytes.Repeat([]byte("nanokvm tunnel binary payload\n"), 256)
	valid := gzipBytes(t, payload)

	tests := []struct {
		name    string
		seed    []byte
		wantErr bool
	}{
		{name: "valid stream", seed: valid},
		{name: "interrupted stream", seed: valid[:len(valid)/2], wantErr: true},
		{name: "corrupt stream", seed: []byte("this is not gzip at all"), wantErr: true},
		{name: "empty stream", seed: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bins, seeds := useTestBinDirs(t)
			if err := os.WriteFile(filepath.Join(seeds, "newt.gz"), tt.seed, 0o644); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(bins, "newt")

			err := extractSeed(proto.TunnelNewt)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected extract error")
				}
				if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
					t.Fatalf("target exists after failure: %v", statErr)
				}
				entries, readErr := os.ReadDir(bins)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(entries) != 0 {
					t.Fatalf("bin dir not clean: %+v", entries)
				}
				return
			}
			if err != nil {
				t.Fatalf("extract seed: %v", err)
			}

			got, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("extracted %d bytes, want %d", len(got), len(payload))
			}
			info, statErr := os.Stat(target)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if info.Mode().Perm() != 0o755 {
				t.Fatalf("binary permissions = %o, want 755", info.Mode().Perm())
			}
		})
	}
}

func TestExtractSeedMissingFile(t *testing.T) {
	bins, _ := useTestBinDirs(t)

	if err := extractSeed(proto.TunnelNewt); err == nil {
		t.Fatal("expected missing seed error")
	}
	if _, err := os.Stat(filepath.Join(bins, "newt")); !os.IsNotExist(err) {
		t.Fatalf("target exists after failure: %v", err)
	}
}

func TestBinaryPath(t *testing.T) {
	bins, seeds := useTestBinDirs(t)
	payload := []byte("seeded binary")
	if err := os.WriteFile(filepath.Join(seeds, "wstunnel.gz"), gzipBytes(t, payload), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := binaryPath(proto.TunnelWstunnel)
	if err != nil {
		t.Fatalf("binary path: %v", err)
	}
	if path != filepath.Join(bins, "wstunnel") {
		t.Fatalf("path = %q", path)
	}
	if got, readErr := os.ReadFile(path); readErr != nil || !bytes.Equal(got, payload) {
		t.Fatalf("content = %q err = %v", got, readErr)
	}

	if err := os.WriteFile(path, []byte("uploaded"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err = binaryPath(proto.TunnelWstunnel)
	if err != nil {
		t.Fatalf("binary path: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "uploaded" {
		t.Fatalf("existing binary was replaced: %q", got)
	}
}

func TestIsCustom(t *testing.T) {
	bins, _ := useTestBinDirs(t)

	if isCustom(proto.TunnelNewt) {
		t.Fatal("custom without binary")
	}
	if err := os.WriteFile(filepath.Join(bins, "newt"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if isCustom(proto.TunnelNewt) {
		t.Fatal("seeded binary reported as custom")
	}
	if err := setCustom(proto.TunnelNewt, true); err != nil {
		t.Fatal(err)
	}
	if !isCustom(proto.TunnelNewt) {
		t.Fatal("uploaded binary not reported as custom")
	}
	if err := setCustom(proto.TunnelNewt, false); err != nil {
		t.Fatal(err)
	}
	if isCustom(proto.TunnelNewt) {
		t.Fatal("custom marker not cleared")
	}
}

func TestWriteWrapper(t *testing.T) {
	dir := useTestConfigDir(t)
	bins, _ := useTestBinDirs(t)
	if err := os.WriteFile(filepath.Join(bins, "newt"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeWrapper(proto.TunnelNewt, Config{Args: "--accept-clients"}); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	path := filepath.Join(dir, "newt.cmd")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("wrapper permissions = %o, want 700", info.Mode().Perm())
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), shellQuote(filepath.Join(bins, "newt"))) {
		t.Fatalf("wrapper = %q, want extracted binary path", content)
	}
}

func hasLine(content string, line string) bool {
	for _, candidate := range strings.Split(content, "\n") {
		if candidate == line {
			return true
		}
	}
	return false
}

func shellRead(t *testing.T, script string) string {
	t.Helper()
	output, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		t.Fatalf("run %q: %v", script, err)
	}
	return string(output)
}
