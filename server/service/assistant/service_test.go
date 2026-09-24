package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeCapturer struct {
	jpeg  []byte
	calls int
}

func (f *fakeCapturer) Capture(context.Context) (Frame, error) {
	f.calls++
	return Frame{JPEG: f.jpeg, Width: 200, Height: 100}, nil
}

type orServer struct {
	bodies  []map[string]any
	replies []string // one per call; "" means {"choices":[]}
	status  int
}

func (o *orServer) start(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		o.bodies = append(o.bodies, body)
		if o.status != 0 {
			w.WriteHeader(o.status)
			io.WriteString(w, `{"error":{"message":"bad key"}}`)
			return
		}
		reply := o.replies[len(o.bodies)-1]
		if reply == "" {
			io.WriteString(w, `{"choices":[]}`)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": reply}}}})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func testService(t *testing.T, providerURL string, cap *fakeCapturer) *Service {
	cfg := defaultConfig()
	cfg.Enabled = true
	cfg.ORBaseURL = providerURL
	cfg.ORModel = "om"
	cfg.ORAPIKey = "sk"
	p, _ := parsePrompts([]byte("[universal]\nprompt = \"U\"\n[mcq]\nprompt = \"M\"\n[frq]\nprompt = \"F\"\n[custom]\nprompt = \"C\"\n"))
	return &Service{
		capture:     cap,
		transport:   &Transport{Client: &http.Client{}, AttemptTimeout: 5 * time.Second, RetryGap: time.Millisecond, MaxAttempts: 2},
		contexts:    &Contexts{},
		attachments: NewAttachmentStore(filepath.Join(t.TempDir(), "att")),
		prompts:     func() (map[string]any, error) { return p, nil },
		loadConfig:  func() (Config, error) { return cfg, nil },
	}
}

func userContent(t *testing.T, body map[string]any, i int) []any {
	t.Helper()
	msgs := body["messages"].([]any)
	return msgs[i].(map[string]any)["content"].([]any)
}

func partText(p any) string { return p.(map[string]any)["text"].(string) }
func partURL(p any) string {
	return p.(map[string]any)["image_url"].(map[string]any)["url"].(string)
}

func TestAskMCQSendsContextsThenScreenshot(t *testing.T) {
	or := &orServer{replies: []string{"B"}}
	cap := &fakeCapturer{jpeg: testJPEG(t, 200, 100)}
	s := testService(t, or.start(t), cap)
	s.contexts.Add(Image{Mime: "image/jpeg", B64: "Y3R4"})

	res, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"})
	if err != nil || res.Answer != "B" || res.Route != "direct (no proxy configured)" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	content := userContent(t, or.bodies[0], 0)
	if len(content) != 3 || partText(content[0]) != "U\n\nM" || partURL(content[1]) != "data:image/jpeg;base64,Y3R4" ||
		!strings.HasPrefix(partURL(content[2]), "data:image/jpeg;base64,") {
		t.Fatalf("content %v", content)
	}
}

func TestAskFRQWithCropAndAttachmentsUsesTwoTurns(t *testing.T) {
	or := &orServer{replies: []string{"  ", "answer"}}
	cap := &fakeCapturer{jpeg: testJPEG(t, 200, 100)}
	s := testService(t, or.start(t), cap)
	cfg, _ := s.loadConfig()
	cfg.ORReasoning = true
	s.loadConfig = func() (Config, error) { return cfg, nil }
	s.attachments.Put("notes.txt", strings.NewReader("hello"))

	res, err := s.Ask(context.Background(), AskRequest{Kind: "frq", Crop: &Crop{X: 0, Y: 0, W: 0.5, H: 0.5}})
	if err != nil || res.Answer != "answer" || len(or.bodies) != 2 {
		t.Fatalf("res=%+v err=%v calls=%d", res, err, len(or.bodies))
	}
	if _, ok := or.bodies[0]["reasoning"]; ok {
		t.Fatal("anchor turn must not reason")
	}
	if partText(userContent(t, or.bodies[0], 0)[0]) != ackPrompt {
		t.Fatalf("anchor text %v", userContent(t, or.bodies[0], 0)[0])
	}
	msgs := or.bodies[1]["messages"].([]any)
	if len(msgs) != 3 || msgs[1].(map[string]any)["content"] != "OK" {
		t.Fatalf("history %v", msgs)
	}
	want := "U\n\nF\n\nNow answer the question above." + "\n\nThe following text files were attached:\n\n--- notes.txt ---\nhello\n"
	if got := partText(userContent(t, or.bodies[1], 2)[0]); got != want {
		t.Fatalf("turn 2 text %q", got)
	}
	if _, ok := or.bodies[1]["reasoning"]; !ok {
		t.Fatal("answer turn must reason")
	}
}

func TestAskCustomUsesTextAndContextsOnly(t *testing.T) {
	or := &orServer{replies: []string{"reply"}}
	cap := &fakeCapturer{}
	s := testService(t, or.start(t), cap)
	s.contexts.Add(Image{Mime: "image/jpeg", B64: "Y3R4"})
	res, err := s.Ask(context.Background(), AskRequest{Kind: "custom", Text: "  why?  "})
	if err != nil || res.Answer != "reply" || cap.calls != 0 {
		t.Fatalf("res=%+v err=%v captures=%d", res, err, cap.calls)
	}
	content := userContent(t, or.bodies[0], 0)
	if partText(content[0]) != "why?\n\nU\n\nC" || len(content) != 2 {
		t.Fatalf("content %v", content)
	}
}

func TestAskOutcomes(t *testing.T) {
	or := &orServer{replies: []string{""}}
	s := testService(t, or.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("empty choices: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "custom", Text: "  "}); !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("empty custom: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "essay"}); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("kind: %v", err)
	}
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq", Crop: &Crop{X: 0.5, Y: 0.5}}); !errors.Is(err, ErrEmptyCrop) {
		t.Fatalf("empty crop: %v", err)
	}

	bad := &orServer{status: 401}
	s = testService(t, bad.start(t), &fakeCapturer{jpeg: testJPEG(t, 8, 8)})
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); err == nil || err.Error() != "bad key" {
		t.Fatalf("provider error: %v", err)
	}

	cfg := defaultConfig()
	s.loadConfig = func() (Config, error) { return cfg, nil }
	if _, err := s.Ask(context.Background(), AskRequest{Kind: "mcq"}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled: %v", err)
	}
}

func TestAddContextAndReasoningPersist(t *testing.T) {
	useTestConfig(t)
	updateConfig(func(c Config) (Config, error) { c.Enabled = true; return c, nil })
	s := testService(t, "http://unused", &fakeCapturer{jpeg: testJPEG(t, 200, 100)})
	s.loadConfig = loadConfig
	if n, err := s.AddContext(context.Background(), &Crop{X: 0, Y: 0, W: 1, H: 1}); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	res, err := s.AdjustReasoning(true)
	if err != nil || res.Label != "Reasoning: low" || !res.Config.ORReasoning {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if cfg, _ := loadConfig(); !cfg.ORReasoning {
		t.Fatal("reasoning change not persisted")
	}
}
