package exit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// loadedNexitFile mirrors nexit's fileConfig (nexit/config.go). Decoding with
// DisallowUnknownFields is what nexit does, so a key the server adds that
// nexit does not know fails here the way it would fail on the exit.
type loadedNexitFile struct {
	Address      string `json:"address"`
	Passcode     string `json:"passcode"`
	Insecure     *bool  `json:"insecure"`
	AllowPrivate *bool  `json:"allowPrivate"`
}

func decodeNexitFile(t *testing.T, body []byte) loadedNexitFile {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var f loadedNexitFile
	if err := dec.Decode(&f); err != nil {
		t.Fatalf("nexit.json does not load the way nexit loads it: %v\n%s", err, body)
	}
	if f.Insecure == nil || f.AllowPrivate == nil {
		t.Fatalf("nexit.json omits insecure or allowPrivate: %s", body)
	}
	return f
}

// The file carries exactly what the nexit one-liners carry: the address they
// pass, the passcode they pass and --insecure exactly when they add it.
func TestNexitConfigMatchesTheCommands(t *testing.T) {
	slot := MustSlot("0")
	fp := "ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12"
	cases := []struct {
		name         string
		origin       Origin
		cert         Certificate
		allowPrivate bool
		wantInsecure bool
	}{
		{"http", Origin{Scheme: "http", Host: "10.12.34.1"}, Certificate{}, false, false},
		{"https-selfsigned", Origin{Scheme: "https", Host: "kvm.example.net"}, Certificate{Fingerprint: fp, SelfSigned: true}, true, true},
		{"https-ca", Origin{Scheme: "https", Host: "kvm.example.net:8443"}, Certificate{Fingerprint: fp}, false, false},
		{"https-unknown-cert", Origin{Scheme: "https", Host: "kvm.local"}, Certificate{}, false, true},
	}
	addressIn := regexp.MustCompile(`"(https?://[^"]+/exit/0)" --passcode`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig(slot)
			cfg.AllowPrivate = tc.allowPrivate
			body, ok := NexitConfig(slot, cfg, tc.origin, tc.cert)
			if !ok {
				t.Fatal("NexitConfig refused a valid host")
			}
			f := decodeNexitFile(t, body)

			rsp := renderCommands(slot, cfg, tc.origin, tc.cert)
			var cmd string
			for _, c := range rsp.Nexit {
				if c.Shell == "cmd" {
					cmd = c.Command
				}
			}
			m := addressIn.FindStringSubmatch(cmd)
			if m == nil {
				t.Fatalf("no address in the nexit cmd command: %s", cmd)
			}
			if f.Address != m[1] {
				t.Fatalf("address = %q, the command runs nexit against %q", f.Address, m[1])
			}
			if f.Passcode != cfg.Token || !strings.Contains(cmd, `--passcode "`+cfg.Token+`"`) {
				t.Fatalf("passcode = %q, token %q, command %s", f.Passcode, cfg.Token, cmd)
			}
			for _, c := range rsp.Nexit {
				if strings.Contains(c.Command, "--insecure") != *f.Insecure {
					t.Fatalf("insecure = %v but the %s command reads %s", *f.Insecure, c.Shell, c.Command)
				}
			}
			if *f.Insecure != tc.wantInsecure {
				t.Fatalf("insecure = %v, want %v", *f.Insecure, tc.wantInsecure)
			}
			if *f.AllowPrivate != tc.allowPrivate {
				t.Fatalf("allowPrivate = %v, want %v", *f.AllowPrivate, tc.allowPrivate)
			}
		})
	}

	if _, ok := NexitConfig(slot, testConfig(slot), Origin{Scheme: "http", Host: "kvm.local$(id)"}, Certificate{}); ok {
		t.Fatal("NexitConfig rendered a metacharacter host")
	}
}

// nexit.json sits behind the same gate as nexit.exe: the token is required,
// a wrong one counts against the source, and the file downloads as nexit.json
// and is never cached.
func TestGateServesNexitJSON(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := LoadConfig(slot)
	r := gateEngine(NewService(h.mgr))

	get := func(token, remote, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/exit/0/nexit.json", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.RemoteAddr = remote
		req.Host = host
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := get("", "203.0.113.7:1", "kvm.local"); rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), cfg.Token) {
		t.Fatalf("no token: %d %s", rec.Code, rec.Body.String())
	}
	if rec := get("wrongtok", "203.0.113.7:2", "kvm.local"); rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), cfg.Token) {
		t.Fatalf("wrong token: %d %s", rec.Code, rec.Body.String())
	}

	rec := get(cfg.Token, "203.0.113.8:1", "kvm.local")
	if rec.Code != http.StatusOK {
		t.Fatalf("right token: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="nexit.json"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q", got)
	}
	f := decodeNexitFile(t, rec.Body.Bytes())
	if f.Address != "http://kvm.local/exit/0" || f.Passcode != cfg.Token || *f.Insecure || *f.AllowPrivate {
		t.Fatalf("body = %+v", f)
	}

	// A bad host renders nothing and is not a token failure.
	if rec := get(cfg.Token, "203.0.113.9:1", "kvm.local'x"); rec.Code != http.StatusNotFound {
		t.Fatalf("metacharacter host: %d", rec.Code)
	}
	if h.mgr.limiter.Locked("203.0.113.9") {
		t.Fatal("a bad host counted against the source")
	}

	// Wrong tokens on nexit.json count towards the lock like any other path.
	for i := 0; i < limiterFreeAttempts+1; i++ {
		get("wrongtok", "198.51.100.1:5", "kvm.local")
	}
	if rec := get(cfg.Token, "198.51.100.1:6", "kvm.local"); rec.Code != http.StatusNotFound {
		t.Fatalf("locked source got nexit.json: %d", rec.Code)
	}

	// A disabled slot serves nothing, the same as nexit.exe.
	if err := h.mgr.Disable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	if rec := get(cfg.Token, "203.0.113.10:1", "kvm.local"); rec.Code != http.StatusNotFound {
		t.Fatalf("disabled slot served nexit.json: %d", rec.Code)
	}
}
