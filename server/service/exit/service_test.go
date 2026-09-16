package exit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

// gateEngine registers the token-gated group the way router/exit.go does.
func gateEngine(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/exit/:slot", svc.Gate)
	r.GET("/exit/:slot/*rest", svc.Gate)
	return r
}

func perform(r *gin.Engine, target, token, remote string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if remote != "" {
		req.RemoteAddr = remote
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func stripDate(h http.Header) http.Header {
	out := h.Clone()
	out.Del("Date")
	return out
}

func TestGateRejectionsAreGinsDefault404(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	svc := NewService(h.mgr)
	r := gateEngine(svc)
	if err := h.mgr.Enable(context.Background(), MustSlot("0")); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := LoadConfig(MustSlot("0"))

	reference := perform(r, "/nowhere/at/all", "", "203.0.113.7:1")
	if reference.Code != http.StatusNotFound {
		t.Fatalf("unknown route = %d", reference.Code)
	}

	cases := map[string]struct {
		target, token, remote string
	}{
		"no token":         {"/exit/0/native", "", "203.0.113.7:2"},
		"wrong token":      {"/exit/0/native", "k7m2p9vx", "203.0.113.7:3"},
		"unknown slot":     {"/exit/7/native", cfg.Token, "203.0.113.7:4"},
		"invalid slot":     {"/exit/zz/native", cfg.Token, "203.0.113.7:5"},
		"bare slot":        {"/exit/0", cfg.Token, "203.0.113.7:6"},
		"unknown rest":     {"/exit/0/whatever", cfg.Token, "203.0.113.7:7"},
		"events in mode A": {"/exit/0/events", cfg.Token, "203.0.113.7:8"},
		"missing client":   {"/exit/0/client.nope", cfg.Token, "203.0.113.7:9"},
		"lowercase scheme": {"/exit/0/native", "", "203.0.113.7:10"},
		"trailing slash":   {"/exit/0/", cfg.Token, "203.0.113.7:11"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := perform(r, tc.target, tc.token, tc.remote)
			if rec.Code != reference.Code {
				t.Fatalf("status %d, want %d", rec.Code, reference.Code)
			}
			if rec.Body.String() != reference.Body.String() {
				t.Fatalf("body %q, want %q", rec.Body.String(), reference.Body.String())
			}
			got, want := stripDate(rec.Header()), stripDate(reference.Header())
			if len(got) != len(want) {
				t.Fatalf("headers %v, want %v", got, want)
			}
			for k, v := range want {
				if strings.Join(got[k], ",") != strings.Join(v, ",") {
					t.Fatalf("header %s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
	if h.mux.served != 0 || h.proxy.served != 0 {
		t.Fatal("a rejected request reached the dataplane")
	}
}

func TestGateAdmitsTheTokenAndLocksASource(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	svc := NewService(h.mgr)
	r := gateEngine(svc)
	slot := MustSlot("0")
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := LoadConfig(slot)

	if rec := perform(r, "/exit/0/native", cfg.Token, "203.0.113.7:1"); rec.Code != http.StatusSwitchingProtocols || h.mux.served != 1 {
		t.Fatalf("native upgrade = %d served=%d", rec.Code, h.mux.served)
	}

	// Enough wrong tokens from one source lock it, even with the right token.
	for i := 0; i < limiterFreeAttempts+1; i++ {
		perform(r, "/exit/0/native", "wrongtok", "198.51.100.1:5")
	}
	if rec := perform(r, "/exit/0/native", cfg.Token, "198.51.100.1:6"); rec.Code != http.StatusNotFound {
		t.Fatalf("locked source admitted: %d", rec.Code)
	}
	// Another source is unaffected.
	if rec := perform(r, "/exit/0/native", cfg.Token, "203.0.113.8:1"); rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("unrelated source refused: %d", rec.Code)
	}

	// Mode B routes everything but the scripts to the proxy.
	mode := proto.ExitModeWstunnel
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{Mode: &mode}); err != nil {
		t.Fatal(err)
	}
	if rec := perform(r, "/exit/0/events", cfg.Token, "203.0.113.9:1"); rec.Code != http.StatusSwitchingProtocols || h.proxy.served != 1 {
		t.Fatalf("mode B events = %d served=%d", rec.Code, h.proxy.served)
	}
	if rec := perform(r, "/exit/0/native", cfg.Token, "203.0.113.9:2"); rec.Code != http.StatusNotFound {
		t.Fatalf("native served in mode B: %d", rec.Code)
	}

	// A disabled slot answers 404 to a correct token.
	if err := h.mgr.Disable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	if rec := perform(r, "/exit/0/events", cfg.Token, "203.0.113.10:1"); rec.Code != http.StatusNotFound {
		t.Fatalf("disabled slot served: %d", rec.Code)
	}
}

func TestAdminHandlersEnvelope(t *testing.T) {
	h := newHarness(t)
	h.mgr.Init()
	svc := NewService(h.mgr)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/extensions/exit")
	api.GET("/slots", svc.GetSlots)
	api.GET("/:slot/status", svc.GetStatus)
	api.GET("/:slot/config", svc.GetConfig)
	api.POST("/:slot/config", svc.SetConfig)
	api.POST("/:slot/enable", svc.Enable)
	api.POST("/:slot/disable", svc.Disable)
	api.POST("/:slot/token/regenerate", svc.RegenerateToken)
	api.POST("/:slot/disconnect", svc.Disconnect)
	api.GET("/:slot/commands", svc.GetCommands)
	api.GET("/:slot/logs", svc.GetLogs)

	call := func(method, target, body string) (int, proto.Response) {
		var reader *strings.Reader
		if body != "" {
			reader = strings.NewReader(body)
		} else {
			reader = strings.NewReader("")
		}
		req := httptest.NewRequest(method, target, reader)
		req.Host = "kvm.local"
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var rsp proto.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &rsp); err != nil {
			t.Fatalf("%s %s: body %q is not the envelope: %v", method, target, rec.Body.String(), err)
		}
		return rec.Code, rsp
	}

	if code, rsp := call(http.MethodGet, "/api/extensions/exit/slots", ""); code != 200 || rsp.Code != 0 {
		t.Fatalf("slots = %d %+v", code, rsp)
	}
	if _, rsp := call(http.MethodGet, "/api/extensions/exit/0/status", ""); rsp.Code != 0 {
		t.Fatalf("status = %+v", rsp)
	}
	if _, rsp := call(http.MethodGet, "/api/extensions/exit/9/status", ""); rsp.Code == 0 {
		t.Fatal("unknown slot status succeeded")
	}
	if _, rsp := call(http.MethodGet, "/api/extensions/exit/x/status", ""); rsp.Code == 0 {
		t.Fatal("invalid slot status succeeded")
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/config", `{"mode":"bogus"}`); rsp.Code == 0 {
		t.Fatal("invalid mode accepted")
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/config", `{"mtu":100}`); rsp.Code == 0 {
		t.Fatal("mtu below the floor accepted")
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/config", `{"dns":["1.1.1.1","not an ip"]}`); rsp.Code == 0 {
		t.Fatal("invalid dns accepted")
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/config", `{"dns":["9.9.9.9"],"allowPrivate":true}`); rsp.Code != 0 {
		t.Fatalf("valid config refused: %+v", rsp)
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/enable", ""); rsp.Code != 0 {
		t.Fatalf("enable = %+v", rsp)
	}
	_, rsp := call(http.MethodGet, "/api/extensions/exit/0/commands", "")
	if rsp.Code != 0 {
		t.Fatalf("commands = %+v", rsp)
	}
	data, _ := json.Marshal(rsp.Data)
	var commands proto.GetExitCommandsRsp
	if err := json.Unmarshal(data, &commands); err != nil {
		t.Fatal(err)
	}
	if commands.Scheme != "http" || commands.Host != "kvm.local" || len(commands.Native) != 3 {
		t.Fatalf("commands = %+v", commands)
	}
	_, rsp = call(http.MethodGet, "/api/extensions/exit/0/config", "")
	data, _ = json.Marshal(rsp.Data)
	if strings.Contains(string(data), `"token"`) {
		t.Fatalf("config leaks the token: %s", data)
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/token/regenerate", ""); rsp.Code != 0 {
		t.Fatalf("regenerate = %+v", rsp)
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/disconnect", ""); rsp.Code != 0 {
		t.Fatalf("disconnect = %+v", rsp)
	}
	if _, rsp := call(http.MethodGet, "/api/extensions/exit/0/logs", ""); rsp.Code != 0 {
		t.Fatalf("logs = %+v", rsp)
	}
	if _, rsp := call(http.MethodPost, "/api/extensions/exit/0/disable", ""); rsp.Code != 0 {
		t.Fatalf("disable = %+v", rsp)
	}
}

// Every rejection on the token-gated surface, whoever produces it (the
// handler, the real Mux, the real WSProxy), must be the response gin writes
// for an unknown route: same status, same body, same headers minus Date. This
// runs over a real listener so Content-Length and the server's own headers
// are part of the comparison.
func TestRejectionPathsAreOneGin404(t *testing.T) {
	h := newHarness(t)
	slot := MustSlot("0")
	// Real mux and proxy, fake door/dns/probe (the dns forwarder would need
	// the gadget address to bind on).
	var mux *Mux
	var pxy *WSProxy
	h.mgr.deps.Components = func(slot Slot, d componentDeps) components {
		mux = NewMux(slot, MuxHooks{OnSession: d.OnSession, OnClose: d.OnClose, Policy: d.Policy, Bytes: d.Bytes})
		pxy = NewWSProxy(slot, nil, d.OnPeerChange)
		return components{Door: h.door, Mux: mux, Proxy: pxy, DNS: h.dns, Probe: h.probe}
	}
	h.mgr.Init()
	svc := NewService(h.mgr)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Tests play several exits from one machine: the peer address comes from
	// a header when present.
	r.Use(func(c *gin.Context) {
		if remote := c.GetHeader("X-Test-Remote"); remote != "" {
			c.Request.RemoteAddr = remote
		}
	})
	r.GET("/exit/:slot", svc.Gate)
	r.GET("/exit/:slot/*rest", svc.Gate)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	if err := h.mgr.Enable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := LoadConfig(slot)

	get := func(t *testing.T, path, token, remote string, upgrade bool) (*http.Response, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if remote != "" {
			req.Header.Set("X-Test-Remote", remote)
		}
		if upgrade {
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
		}
		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer rsp.Body.Close()
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return rsp, string(body)
	}

	reference, refBody := get(t, "/nowhere/at/all", "", "203.0.113.1:1", false)
	if reference.StatusCode != http.StatusNotFound || refBody != default404Body {
		t.Fatalf("unknown route = %d %q", reference.StatusCode, refBody)
	}
	same := func(t *testing.T, path, token, remote string, upgrade bool) {
		t.Helper()
		rsp, body := get(t, path, token, remote, upgrade)
		if rsp.StatusCode != reference.StatusCode {
			t.Fatalf("status %d, want %d", rsp.StatusCode, reference.StatusCode)
		}
		if body != refBody {
			t.Fatalf("body %q, want %q", body, refBody)
		}
		got, want := stripDate(rsp.Header), stripDate(reference.Header)
		if len(got) != len(want) {
			t.Fatalf("headers %v, want %v", got, want)
		}
		for k, v := range want {
			if strings.Join(got[k], ",") != strings.Join(v, ",") {
				t.Fatalf("header %s = %v, want %v", k, got[k], v)
			}
		}
		if rsp.ContentLength != reference.ContentLength {
			t.Fatalf("content length %d, want %d", rsp.ContentLength, reference.ContentLength)
		}
	}

	// Handler rejections.
	for name, tc := range map[string]struct{ path, token, remote string }{
		"no token":     {"/exit/0/native", "", "203.0.113.2:1"},
		"wrong token":  {"/exit/0/native", "k7m2p9vx", "203.0.113.3:1"},
		"unknown slot": {"/exit/7/native", cfg.Token, "203.0.113.4:1"},
		"bare slot":    {"/exit/0", cfg.Token, "203.0.113.5:1"},
		"mode B path":  {"/exit/0/events", cfg.Token, "203.0.113.6:1"},
	} {
		t.Run(name, func(t *testing.T) { same(t, tc.path, tc.token, tc.remote, false) })
	}
	t.Run("locked source", func(t *testing.T) {
		for i := 0; i < limiterFreeAttempts+1; i++ {
			get(t, "/exit/0/native", "wrongtok", "198.51.100.1:5", false)
		}
		same(t, "/exit/0/native", cfg.Token, "198.51.100.1:6", false)
	})

	// Mux pin-peer refusal: one exit attached, another host knocks.
	t.Run("mux pinned peer", func(t *testing.T) {
		pin := true
		if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{PinPeer: &pin}); err != nil {
			t.Fatal(err)
		}
		f := NewFakeExit()
		hdr := http.Header{}
		hdr.Set("Authorization", "Bearer "+cfg.Token)
		hdr.Set("X-Test-Remote", "198.51.100.10:1000")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := f.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/exit/0/native", hdr); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(f.Close)
		waitFor(t, "session attached", func() bool { return mux.Current() != nil })
		same(t, "/exit/0/native", cfg.Token, "198.51.100.11:1000", true)
		if mux.Current() == nil {
			t.Fatal("the refused knock disturbed the pinned session")
		}
	})

	// WSProxy rejections in Mode B.
	mode := proto.ExitModeWstunnel
	if err := h.mgr.SetConfig(context.Background(), slot, proto.SetExitConfigReq{Mode: &mode}); err != nil {
		t.Fatal(err)
	}
	t.Run("proxy non-upgrade", func(t *testing.T) { same(t, "/exit/0/events", cfg.Token, "203.0.113.7:1", false) })
	t.Run("proxy other path", func(t *testing.T) { same(t, "/exit/0/whatever", cfg.Token, "203.0.113.8:1", true) })
	t.Run("proxy pinned peer", func(t *testing.T) {
		// The first upgrade pins its host even though wstunnel is not there
		// to answer it (the proxy reports that as 502, not 404).
		if rsp, _ := get(t, "/exit/0/events", cfg.Token, "198.51.100.20:1000", true); rsp.StatusCode == http.StatusNotFound {
			t.Fatal("the pinning upgrade itself was refused")
		}
		same(t, "/exit/0/events", cfg.Token, "198.51.100.21:1000", true)
	})
	t.Run("native in mode B", func(t *testing.T) { same(t, "/exit/0/native", cfg.Token, "203.0.113.9:1", true) })

	// Disabled slot, right token.
	if err := h.mgr.Disable(context.Background(), slot); err != nil {
		t.Fatal(err)
	}
	t.Run("disabled slot", func(t *testing.T) { same(t, "/exit/0/events", cfg.Token, "203.0.113.10:1", true) })
}

// SEC-1: an on-disk token that this package never issued (hand-edited,
// restored, or simply missing from the JSON) must never turn the gate into an
// open door for a request that carries no token at all.
func TestGateNeverAdmitsAnEmptyHeaderAgainstAnEmptyToken(t *testing.T) {
	h := newHarness(t)
	if err := os.MkdirAll(ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corrupt := `{"slot":"0","enabled":true,"pending":false,"mode":"native","token":"","nic":"gadget","mtu":1280}` + "\n"
	if err := os.WriteFile(filepath.Join(ConfigDir, "0.json"), []byte(corrupt), 0o600); err != nil {
		t.Fatal(err)
	}
	h.mgr.Init()
	svc := NewService(h.mgr)
	r := gateEngine(svc)

	// The load repaired the file: a fresh token, and the slot forced off so the
	// operator re-enables it with the new token in hand.
	cfg, ok, err := LoadConfig(MustSlot("0"))
	if err != nil || !ok {
		t.Fatalf("slot 0 config: ok=%v err=%v", ok, err)
	}
	if !ValidToken(cfg.Token) || cfg.Enabled {
		t.Fatalf("corrupt token not repaired: %+v", cfg)
	}
	if rec := perform(r, "/exit/0/native", "", "203.0.113.7:2"); rec.Code != http.StatusNotFound || h.mux.served != 0 {
		t.Fatalf("empty header admitted after load: %d served=%d", rec.Code, h.mux.served)
	}

	// Belt and braces: even with an empty token sitting in memory on an enabled
	// slot with live components, the gate itself refuses an empty header.
	if err := h.mgr.Enable(context.Background(), MustSlot("0")); err != nil {
		t.Fatal(err)
	}
	s, _ := h.mgr.get(MustSlot("0"))
	h.mgr.mu.Lock()
	s.cfg.Token = ""
	h.mgr.mu.Unlock()
	if rec := perform(r, "/exit/0/native", "", "203.0.113.7:3"); rec.Code != http.StatusNotFound || h.mux.served != 0 {
		t.Fatalf("empty header admitted against an empty token: %d served=%d", rec.Code, h.mux.served)
	}
	// And the dummy the gate compares against is not itself a valid token.
	if rec := perform(r, "/exit/0/native", strings.Repeat("x", TokenLength), "203.0.113.7:4"); rec.Code != http.StatusNotFound || h.mux.served != 0 {
		t.Fatalf("placeholder token admitted: %d served=%d", rec.Code, h.mux.served)
	}
}
