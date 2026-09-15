package exit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
			if rsp.WstunnelVersion != WstunnelVersion || rsp.Scheme != tc.origin.Scheme || rsp.Host != tc.origin.Host {
				t.Fatalf("header = %+v", rsp)
			}
			for _, cmd := range append(append([]proto.ExitCommand{}, rsp.Native...), rsp.Wstunnel...) {
				if !strings.Contains(cmd.Command, cfg.Token) {
					t.Fatalf("%s %s command lacks the token: %s", cmd.Platform, cmd.Shell, cmd.Command)
				}
				if strings.Contains(cmd.Command, "/exit/0/"+cfg.Token) || strings.Contains(cmd.Command, "?token=") {
					t.Fatalf("token in the path: %s", cmd.Command)
				}
				if strings.Contains(cmd.Command, "\n") {
					t.Fatalf("command is not one line: %q", cmd.Command)
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
	// The clients land with the clients package; when present, every
	// placeholder must be gone after templating.
	for name := range clientNames {
		body, ok := RenderClient(name, slot, cfg, origin, "ab12")
		if !ok {
			continue
		}
		if strings.Contains(string(body), "__") {
			t.Fatalf("%s still carries a placeholder", name)
		}
		if !strings.Contains(string(body), cfg.Token) {
			t.Fatalf("%s lacks the token", name)
		}
	}
}
