package assistant

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func testEngine(t *testing.T, s *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(s).Register(r.Group("/api/assistant"))
	return r
}

func call(t *testing.T, r *gin.Engine, method, path, body string) (int, map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	code, _ := out["code"].(float64)
	return int(code), out, w
}

func TestConfigEndpointsMaskSecrets(t *testing.T) {
	useTestConfig(t)
	s := testService(t, "http://unused", &fakeCapturer{})
	s.loadConfig = loadConfig
	r := testEngine(t, s)

	code, out, _ := call(t, r, "POST", "/api/assistant/config", `{"enabled":true,"orApiKey":"sk-secret"}`)
	if code != 0 || strings.Contains(mustJSON(out), "sk-secret") || out["data"].(map[string]any)["hasOrApiKey"] != true {
		t.Fatalf("post %v", out)
	}
	code, out, _ = call(t, r, "POST", "/api/assistant/config", `{"provider":"nope"}`)
	if code != codeError {
		t.Fatalf("invalid provider accepted: %v", out)
	}
	_, out, _ = call(t, r, "GET", "/api/assistant/config", "")
	if strings.Contains(mustJSON(out), "sk-secret") || out["data"].(map[string]any)["enabled"] != true {
		t.Fatalf("get %v", out)
	}
}

func TestAskEndpointCodes(t *testing.T) {
	or := &orServer{replies: []string{"", "A"}}
	s := testService(t, or.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	r := testEngine(t, s)
	if code, out, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`); code != codeNoAnswer {
		t.Fatalf("no answer: %v", out)
	}
	code, out, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`)
	if code != 0 || out["data"].(map[string]any)["answer"] != "A" {
		t.Fatalf("answer: %v", out)
	}
	cfg := defaultConfig()
	s.loadConfig = func() (Config, error) { return cfg, nil }
	if code, _, _ := call(t, r, "POST", "/api/assistant/ask", `{"kind":"mcq"}`); code != codeDisabled {
		t.Fatalf("disabled code %d", code)
	}
}

func TestContextScreenshotAndAttachmentEndpoints(t *testing.T) {
	s := testService(t, "http://unused", &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	r := testEngine(t, s)
	if code, out, _ := call(t, r, "POST", "/api/assistant/context", `{}`); code != 0 || out["data"].(map[string]any)["count"] != 1.0 {
		t.Fatalf("add context %v", out)
	}
	if _, out, _ := call(t, r, "DELETE", "/api/assistant/context", ""); out["data"].(map[string]any)["count"] != 0.0 {
		t.Fatalf("clear %v", out)
	}
	_, _, w := call(t, r, "GET", "/api/assistant/screenshot", "")
	if w.Header().Get("Content-Type") != "image/jpeg" || w.Body.Len() == 0 {
		t.Fatalf("screenshot %v", w.Header())
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "notes.txt")
	fw.Write([]byte("hi"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/assistant/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_, out, _ := call(t, r, "GET", "/api/assistant/attachments", "")
	if list := out["data"].([]any); len(list) != 1 {
		t.Fatalf("list %v", out)
	}
	call(t, r, "DELETE", "/api/assistant/attachments?name=notes.txt", "")
	_, out, _ = call(t, r, "GET", "/api/assistant/attachments", "")
	if list := out["data"].([]any); len(list) != 0 {
		t.Fatalf("after delete %v", out)
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
