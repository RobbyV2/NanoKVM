package exit

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func TestOriginOf(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://kvm.local:8080/api", nil)
	if o := OriginOf(r); o.Scheme != "http" || o.Host != "kvm.local:8080" || o.WSScheme() != "ws" {
		t.Fatalf("plain origin = %+v", o)
	}
	r.Header.Set("X-Forwarded-Proto", "https, http")
	if o := OriginOf(r); o.Scheme != "https" || o.WSScheme() != "wss" {
		t.Fatalf("forwarded origin = %+v", o)
	}
	r.Header.Set("X-Forwarded-Proto", "gopher")
	if o := OriginOf(r); o.Scheme != "http" {
		t.Fatalf("garbage forwarded scheme accepted: %+v", o)
	}
}

func selfSignedPEM(t *testing.T, issuer, subject string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		Issuer:       pkix.Name{CommonName: issuer},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	parent := &x509.Certificate{Subject: pkix.Name{CommonName: issuer}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestCertificateFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.crt")
	if err := os.WriteFile(path, selfSignedPEM(t, "nanokvm", "nanokvm"), 0o644); err != nil {
		t.Fatal(err)
	}
	cert, err := certificateFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.Fingerprint) != 64 || strings.ToLower(cert.Fingerprint) != cert.Fingerprint {
		t.Fatalf("fingerprint = %q", cert.Fingerprint)
	}
	if !cert.SelfSigned {
		t.Fatal("issuer == subject not reported as self-signed")
	}

	if err := os.WriteFile(path, selfSignedPEM(t, "Some CA", "nanokvm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cert, err := certificateFrom(path); err != nil || cert.SelfSigned {
		t.Fatalf("CA-issued certificate = %+v, %v", cert, err)
	}
	if _, err := certificateFrom(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing certificate returned no error")
	}
}

// allCommands is every one-liner a response carries, both modes and the
// latest-fetch set.
func allCommands(rsp proto.GetExitCommandsRsp) []proto.ExitCommand {
	all := append([]proto.ExitCommand{}, rsp.Native...)
	all = append(all, rsp.Wstunnel...)
	return append(all, rsp.WstunnelLatest...)
}

// Snapshot of every platform for both schemes and both certificate shapes.
// Regenerate with UPDATE_GOLDEN=1 after a deliberate change to the commands.
func TestCommandsSnapshot(t *testing.T) {
	slot := MustSlot("0")
	cfg := testConfig(slot)
	fp := "ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12"

	cases := []struct {
		name   string
		origin Origin
		cert   Certificate
	}{
		{"http", Origin{Scheme: "http", Host: "10.12.34.1"}, Certificate{}},
		{"https-selfsigned", Origin{Scheme: "https", Host: "kvm.example.net"}, Certificate{Fingerprint: fp, SelfSigned: true}},
		{"https-ca", Origin{Scheme: "https", Host: "kvm.example.net:8443"}, Certificate{Fingerprint: fp}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rsp := renderCommands(slot, cfg, tc.origin, tc.cert)
			if len(rsp.Native) != 3 || len(rsp.Wstunnel) != 3 {
				t.Fatalf("expected three platforms per mode, got %d/%d", len(rsp.Native), len(rsp.Wstunnel))
			}
			if rsp.WstunnelVersion != WstunnelVersion || rsp.Scheme != tc.origin.Scheme || rsp.Host != tc.origin.Host || rsp.WstunnelRepo != wstunnelRepoURL {
				t.Fatalf("header = %+v", rsp)
			}
			for _, cmd := range allCommands(rsp) {
				if !strings.Contains(cmd.Command, cfg.Token) {
					t.Fatalf("%s %s command lacks the token: %s", cmd.Platform, cmd.Shell, cmd.Command)
				}
				if strings.Contains(cmd.Command, "/exit/0/"+cfg.Token) || strings.Contains(cmd.Command, "?token=") {
					t.Fatalf("token in the path: %s", cmd.Command)
				}
				if strings.Contains(cmd.Command, "\n") {
					t.Fatalf("command is not one line: %q", cmd.Command)
				}
				// The release tarballs store the binary as 0644, so every
				// Unix command that extracts it must chmod it before exec.
				if cmd.Shell == "bash" && strings.Contains(cmd.Command, "tar -xzf wstunnel.tgz wstunnel") {
					chmod := strings.Index(cmd.Command, "chmod +x wstunnel")
					exec := strings.Index(cmd.Command, "exec ./wstunnel")
					if chmod < 0 || exec < 0 || chmod > exec {
						t.Fatalf("%s %s command does not chmod wstunnel before exec: %s", cmd.Platform, cmd.Shell, cmd.Command)
					}
				}
			}
			for _, cmd := range rsp.Wstunnel {
				if !strings.Contains(cmd.Command, "v10.7.1/wstunnel_10.7.1_") {
					t.Fatalf("wstunnel command is not pinned: %s", cmd.Command)
				}
				verify := strings.Contains(cmd.Command, "--tls-verify-certificate")
				wantVerify := tc.origin.TLS() && !tc.cert.SelfSigned
				if verify != wantVerify {
					t.Fatalf("%s verify=%v want %v: %s", tc.name, verify, wantVerify, cmd.Command)
				}
			}
			for _, cmd := range rsp.Native {
				hasK := strings.Contains(cmd.Command, "-fsSLk") || strings.Contains(cmd.Command, "ServerCertificateValidationCallback")
				if hasK != tc.origin.TLS() {
					t.Fatalf("%s %s insecure-fetch=%v want %v: %s", tc.name, cmd.Platform, hasK, tc.origin.TLS(), cmd.Command)
				}
			}
			for key, sum := range wstunnelSHA256 {
				if len(sum) != 64 {
					t.Fatalf("sha256 for %s has length %d", key, len(sum))
				}
			}

			// The latest-fetch set: windows, macos, linux in the pinned
			// order, nothing pinned in them, the same kvm url and verify flag
			// as the pinned command of the same platform, and the releases
			// page as fallback. The two Unix commands differ only in the
			// asset platform and the checksum tool.
			latest := rsp.WstunnelLatest
			if len(latest) != 3 ||
				latest[0].Platform != "windows" || latest[0].Shell != "powershell" ||
				latest[1].Platform != "macos" || latest[1].Shell != "bash" ||
				latest[2].Platform != "linux" || latest[2].Shell != "bash" {
				t.Fatalf("latest = %+v", latest)
			}
			if !strings.Contains(latest[0].Command, wstunnelLatestAPI) || !strings.Contains(latest[1].Command, "releases/latest") || !strings.Contains(latest[2].Command, "releases/latest") {
				t.Fatalf("latest commands do not resolve the release: %+v", latest)
			}
			if !strings.Contains(latest[1].Command, "_darwin_${a}.tar.gz") || !strings.Contains(latest[1].Command, "| shasum -a 256 -c - ||") {
				t.Fatalf("latest macos command does not fetch the darwin asset and check it with shasum: %s", latest[1].Command)
			}
			if !strings.Contains(latest[2].Command, "_linux_${a}.tar.gz") || !strings.Contains(latest[2].Command, "| sha256sum -c - ||") {
				t.Fatalf("latest linux command does not fetch the linux asset and check it with sha256sum: %s", latest[2].Command)
			}
			for _, cmd := range latest {
				if strings.Contains(cmd.Command, "v10.7.1/") {
					t.Fatalf("latest %s command carries the pin: %s", cmd.Platform, cmd.Command)
				}
				for key, sum := range wstunnelSHA256 {
					if strings.Contains(cmd.Command, sum) {
						t.Fatalf("latest %s command carries the %s hash: %s", cmd.Platform, key, cmd.Command)
					}
				}
				if !strings.Contains(cmd.Command, "github.com/erebe/wstunnel/releases") {
					t.Fatalf("latest %s command lacks the fallback link: %s", cmd.Platform, cmd.Command)
				}
				if !strings.Contains(cmd.Command, "'"+tc.origin.WSScheme()+"://"+tc.origin.Host+"'") {
					t.Fatalf("latest %s command does not single-quote the kvm url: %s", cmd.Platform, cmd.Command)
				}
				var pinned proto.ExitCommand
				for _, p := range rsp.Wstunnel {
					if p.Platform == cmd.Platform {
						pinned = p
					}
				}
				if strings.Contains(cmd.Command, "--tls-verify-certificate") != strings.Contains(pinned.Command, "--tls-verify-certificate") {
					t.Fatalf("latest %s verify flag differs from the pinned one:\n%s\n%s", cmd.Platform, cmd.Command, pinned.Command)
				}
			}

			got, err := json.MarshalIndent(rsp, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			golden := filepath.Join("testdata", "commands", tc.name+".json")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden: %v (run with UPDATE_GOLDEN=1 to create it)", err)
			}
			if string(want) != string(got) {
				t.Fatalf("commands differ from %s:\n%s", golden, got)
			}
		})
	}
}

func TestRenderClientTemplating(t *testing.T) {
	slot := MustSlot("0")
	cfg := testConfig(slot)
	origin := Origin{Scheme: "https", Host: "kvm.example.net"}
	if _, ok := RenderClient("client.nope", slot, cfg, origin, ""); ok {
		t.Fatal("unknown client rendered")
	}
	// Every placeholder must be gone after templating (client.sh keeps a
	// literal `__*)` guard that refuses to run untemplated, so the check is by
	// placeholder name). client.sh fetches and gets the http scheme; the three
	// clients open the socket and get the ws scheme (plan, W3 <-> W1).
	placeholders := []string{phScheme, phHost, phSlot, phToken, phFingerprint, phAllowPrivate}
	for name := range clientNames {
		for _, o := range []Origin{origin, {Scheme: "http", Host: "10.1.2.1"}} {
			body, ok := RenderClient(name, slot, cfg, o, "ab12")
			if !ok {
				t.Fatalf("%s is not embedded", name)
			}
			text := string(body)
			for _, ph := range placeholders {
				if strings.Contains(text, ph) {
					t.Fatalf("%s (%s) still carries %s", name, o.Scheme, ph)
				}
			}
			if !strings.Contains(text, cfg.Token) || !strings.Contains(text, o.Host) {
				t.Fatalf("%s lacks the token or host", name)
			}
			want := o.WSScheme()
			if name == "client.sh" {
				want = o.Scheme
			}
			if !strings.Contains(text, `"`+want+`"`) && !strings.Contains(text, `'`+want+`'`) {
				t.Fatalf("%s (%s) was not templated with scheme %q", name, o.Scheme, want)
			}
		}
	}
}

func TestEmbeddedClientsAreExactlyTheFour(t *testing.T) {
	entries, err := clientFiles.ReadDir("clients")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	want := []string{"client.pl", "client.ps1", "client.py", "client.sh"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("embedded %v, want %v (README.md and testdata must stay out of the binary)", names, want)
	}
	for _, name := range want {
		if !clientNames[name] || clientEscape[name] == nil {
			t.Fatalf("%s is embedded but not served or has no escaper", name)
		}
	}
	if len(clientNames) != len(want) || len(clientEscape) != len(want) {
		t.Fatalf("served %d, escaped %d, embedded %d", len(clientNames), len(clientEscape), len(want))
	}
}

// The quoting is the second fence behind ValidHost: even a value that could
// never pass it is inert in every language the templates and one-liners are
// written in.
func TestTemplatedValuesAreQuotedPerLanguage(t *testing.T) {
	hostile := "a'b\"c$(id)`id`\\d@e"
	if got, want := shQuote(hostile), `'a'\''b"c$(id)`+"`id`"+`\d@e'`; got != want {
		t.Errorf("shQuote = %s, want %s", got, want)
	}
	if got, want := psQuote(hostile), "'a''b\"c$(id)`id`\\d@e'"; got != want {
		t.Errorf("psQuote = %s, want %s", got, want)
	}
	want := map[string]string{
		"client.sh":  "a'b\\\"c\\$(id)\\`id\\`\\\\d@e",
		"client.pl":  "a'b\\\"c\\$(id)`id`\\\\d\\@e",
		"client.py":  "a'b\\\"c$(id)`id`\\\\d@e",
		"client.ps1": "a''b\"c$(id)`id`\\d@e",
	}
	for name, esc := range clientEscape {
		if got := esc(hostile); got != want[name] {
			t.Errorf("%s escape = %s, want %s", name, got, want[name])
		}
	}

	// The one-liners never carry a bare url: the validator does not admit a
	// space or a quote, so the quoting is only ever visible as the quotes
	// themselves, and the UI's readdressing treats a quote as a url boundary.
	rsp := renderCommands(MustSlot("0"), testConfig(MustSlot("0")), Origin{Scheme: "https", Host: "kvm.example.net:8443"}, Certificate{})
	for _, cmd := range allCommands(rsp) {
		if !strings.Contains(cmd.Command, "'https://kvm.example.net:8443/exit/0/client.") && !strings.Contains(cmd.Command, "'wss://kvm.example.net:8443'") {
			t.Errorf("%s %s does not quote its url: %s", cmd.Platform, cmd.Shell, cmd.Command)
		}
	}
}

// SEC-3: the Host header is templated into sh, Perl, PowerShell and Python
// literals and into unquoted one-liners. Go's net/http admits ', $(, ;, & and
// more in Host, so the templates must refuse anything outside host[:port].
func TestRenderClientRefusesHostMetacharacters(t *testing.T) {
	slot := MustSlot("0")
	cfg := testConfig(slot)
	bad := []string{
		"kvm'x", "kvm$(id)", "kvm;rm", "kvm&x", "kvm<x", "k(v)m", "kvm*x", "kvm`id`",
		"kvm x", `kvm"x`, "kvm\\x", "", ":8443", "kvm:", "kvm:123456", "kvm:8a",
		"fd00::1", "[fd00::1", "[fd00::1]x", "[fd00::1]:", "[fe80::1%25eth0]", "kvm/x", "kvm?x=1", "kvm#x",
	}
	for _, host := range bad {
		if ValidHost(host) {
			t.Errorf("ValidHost(%q) = true", host)
		}
		for name := range clientNames {
			if body, ok := RenderClient(name, slot, cfg, Origin{Scheme: "http", Host: host}, ""); ok {
				t.Errorf("%s rendered for host %q:\n%s", name, host, body)
			}
		}
	}
	good := []string{"kvm.local", "10.1.2.1", "kvm.example.net:8443", "[fd00::1]:8443", "[fd00::1]", "localhost", "my_kvm", "KVM-1.lan:80"}
	for _, host := range good {
		if !ValidHost(host) {
			t.Errorf("ValidHost(%q) = false", host)
		}
		if _, ok := RenderClient("client.sh", slot, cfg, Origin{Scheme: "http", Host: host}, ""); !ok {
			t.Errorf("client.sh refused host %q", host)
		}
	}
}

func TestGateAndCommandsRefuseAMetacharacterHost(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := LoadConfig(slot)
	r := gateEngine(NewService(h.mgr))

	req := httptest.NewRequest(http.MethodGet, "/exit/0/client.sh", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.RemoteAddr = "203.0.113.7:1"
	req.Host = "kvm.local$(id)"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "$(id)") {
		t.Fatalf("script fetch with a metacharacter host: %d\n%s", rec.Code, rec.Body.String())
	}
	// The refusal is not a token failure, so the source is not counted.
	if h.mgr.limiter.Locked("203.0.113.7") {
		t.Fatal("a bad host counted against the source")
	}
	req.Host = "kvm.local"
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `HOST="kvm.local"`) {
		t.Fatalf("script fetch with a plain host: %d", rec.Code)
	}

	admin := httptest.NewRequest(http.MethodGet, "/api/extensions/exit/0/commands", nil)
	admin.Host = "kvm.local'x"
	if _, err := h.mgr.Commands(slot, admin); !errors.Is(err, ErrBadHost) {
		t.Fatalf("commands for a host with a quote in it: %v", err)
	}
	admin.Host = "kvm.local:8443"
	rsp, err := h.mgr.Commands(slot, admin)
	if err != nil {
		t.Fatal(err)
	}
	// The one-liners quote the url they carry, so the host is never bare in a
	// shell or PowerShell command line even after validation.
	for _, cmd := range allCommands(rsp) {
		if strings.Contains(cmd.Command, " http://kvm.local:8443") || strings.Contains(cmd.Command, " ws://kvm.local:8443") {
			t.Errorf("%s %s carries the url unquoted: %s", cmd.Platform, cmd.Shell, cmd.Command)
		}
	}
}
