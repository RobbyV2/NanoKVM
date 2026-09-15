package exit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
