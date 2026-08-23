package tunnel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

func callGetMemory(t *testing.T, name proto.TunnelName) proto.GetTunnelMemoryRsp {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/extensions/tunnel/"+string(name)+"/memory", nil)

	NewService(name).GetMemoryLimit(c)

	var rsp struct {
		Code int                      `json:"code"`
		Msg  string                   `json:"msg"`
		Data proto.GetTunnelMemoryRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode memory response %q: %v", recorder.Body.String(), err)
	}
	if rsp.Code != 0 {
		t.Fatalf("get memory reported %d %q", rsp.Code, rsp.Msg)
	}
	return rsp.Data
}

func callSetMemory(t *testing.T, name proto.TunnelName, enabled bool) proto.Response {
	t.Helper()

	body, err := json.Marshal(map[string]bool{"enabled": enabled})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/extensions/tunnel/"+string(name)+"/memory", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	NewService(name).SetMemoryLimit(c)

	var rsp proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode memory response %q: %v", recorder.Body.String(), err)
	}
	return rsp
}

// GOMEMLIMIT with no unit suffix is a byte count, so the plain integer the file
// holds exports a 75 byte heap and pins the collector at its 50% CPU ceiling
// for as long as newt runs. S98tailscaled appends MiB; the wrapper has to too.
func TestRenderWrapperExportsMemLimitInMiB(t *testing.T) {
	useTestConfigDir(t)
	if err := setMemLimit(proto.TunnelNewt, 75); err != nil {
		t.Fatal(err)
	}

	content, err := renderWrapper(proto.TunnelNewt, Config{}, "/etc/kvm/bin/newt")
	if err != nil {
		t.Fatalf("render wrapper: %v", err)
	}

	if !hasLine(content, `export GOMEMLIMIT='75MiB'`) {
		t.Fatalf("wrapper = %q, want line export GOMEMLIMIT='75MiB'", content)
	}
	if hasLine(content, `export GOMEMLIMIT='75'`) {
		t.Fatalf("wrapper exports GOMEMLIMIT as a byte count: %q", content)
	}
}

func TestRenderWrapperWithoutMemLimitFile(t *testing.T) {
	useTestConfigDir(t)

	content, err := renderWrapper(proto.TunnelNewt, Config{}, "/etc/kvm/bin/newt")
	if err != nil {
		t.Fatalf("render wrapper: %v", err)
	}

	if strings.Contains(content, "GOMEMLIMIT") {
		t.Fatalf("wrapper = %q, want no GOMEMLIMIT with no limit file", content)
	}
	if !hasLine(content, `export GOGC='50'`) {
		t.Fatalf("wrapper = %q, want GOGC untouched by the limit", content)
	}
}

func TestMemoryLimitRoundTrip(t *testing.T) {
	dir := useTestConfigDir(t)

	if data := callGetMemory(t, proto.TunnelNewt); data.Enabled || !data.Supported || data.Limit != 75 {
		t.Fatalf("get memory = %+v, want supported and disabled at 75", data)
	}

	if rsp := callSetMemory(t, proto.TunnelNewt, true); rsp.Code != 0 {
		t.Fatalf("set memory reported %d %q", rsp.Code, rsp.Msg)
	}

	stored, err := os.ReadFile(filepath.Join(dir, "newt.GOMEMLIMIT"))
	if err != nil {
		t.Fatalf("read stored limit: %v", err)
	}
	if strings.TrimSpace(string(stored)) != "75" {
		t.Fatalf("stored limit = %q, want 75", stored)
	}

	if data := callGetMemory(t, proto.TunnelNewt); !data.Enabled || data.Limit != 75 {
		t.Fatalf("get memory = %+v, want enabled at 75", data)
	}

	if rsp := callSetMemory(t, proto.TunnelNewt, false); rsp.Code != 0 {
		t.Fatalf("unset memory reported %d %q", rsp.Code, rsp.Msg)
	}
	if _, err := os.Stat(filepath.Join(dir, "newt.GOMEMLIMIT")); !os.IsNotExist(err) {
		t.Fatalf("limit file survives disabling: %v", err)
	}
	if data := callGetMemory(t, proto.TunnelNewt); data.Enabled {
		t.Fatalf("get memory = %+v, want disabled", data)
	}
}

// The shared /etc/kvm/GOMEMLIMIT is tailscaled's and the NanoKVM server's own
// runtime limit. Writing newt's limit must move neither.
func TestSetMemoryLimitLeavesTheSharedLimitAlone(t *testing.T) {
	dir := useTestConfigDir(t)

	shared := filepath.Join(dir, "GOMEMLIMIT")
	if err := os.WriteFile(shared, []byte("512"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := debug.SetMemoryLimit(-1)

	if rsp := callSetMemory(t, proto.TunnelNewt, true); rsp.Code != 0 {
		t.Fatalf("set memory reported %d %q", rsp.Code, rsp.Msg)
	}

	if got, err := os.ReadFile(shared); err != nil || string(got) != "512" {
		t.Fatalf("shared limit = %q err = %v, want 512 untouched", got, err)
	}
	if after := debug.SetMemoryLimit(-1); after != before {
		t.Fatalf("server runtime limit moved from %d to %d", before, after)
	}
	if utils.IsGoMemLimitExist() {
		t.Fatalf("newt's limit was written to %s", utils.GoMemLimitFile)
	}

	content, err := renderWrapper(proto.TunnelWstunnel, Config{}, "/etc/kvm/bin/wstunnel")
	if err != nil {
		t.Fatalf("render wrapper: %v", err)
	}
	if strings.Contains(content, "GOMEMLIMIT") {
		t.Fatalf("wstunnel wrapper = %q, want no GOMEMLIMIT", content)
	}
}

// wstunnel is Rust: no garbage collector, nothing GOMEMLIMIT can reach.
func TestMemoryLimitUnsupportedForWstunnel(t *testing.T) {
	dir := useTestConfigDir(t)

	data := callGetMemory(t, proto.TunnelWstunnel)
	if data.Supported || data.Enabled || data.Limit != 0 {
		t.Fatalf("get memory = %+v, want unsupported", data)
	}

	if rsp := callSetMemory(t, proto.TunnelWstunnel, true); rsp.Code == 0 {
		t.Fatal("set memory accepted a limit for a service with no Go runtime")
	}
	if _, err := os.Stat(filepath.Join(dir, "wstunnel.GOMEMLIMIT")); !os.IsNotExist(err) {
		t.Fatalf("limit file written for wstunnel: %v", err)
	}
}
