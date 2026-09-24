package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	ErrDisabled    = errors.New("assistant is disabled")
	ErrNoAnswer    = errors.New("no answer")
	ErrInvalidKind = errors.New("unknown ask kind")
)

// Verbatim from content.js askLLM.
const ackPrompt = `Above is the question and its context. Reply with only "OK" to confirm — ` +
	"do not answer yet. I will then send the remaining information and the " +
	"instructions, and you will answer."

type Service struct {
	capture     Capturer
	transport   *Transport
	contexts    *Contexts
	attachments *AttachmentStore
	prompts     func() (map[string]any, error)
	loadConfig  func() (Config, error)
}

func NewService(capture Capturer) *Service {
	return &Service{
		capture:     capture,
		transport:   NewTransport(),
		contexts:    &Contexts{},
		attachments: NewAttachmentStore(AttachmentsDir),
		prompts:     loadPrompts,
		loadConfig:  loadConfig,
	}
}

type AskRequest struct {
	Kind string `json:"kind"`
	Crop *Crop  `json:"crop"`
	Text string `json:"text"`
}

type AskResult struct {
	Answer string `json:"answer"`
	Route  string `json:"route"`
}

type ReasoningResult struct {
	Label   string       `json:"label"`
	Changed bool         `json:"changed"`
	Config  PublicConfig `json:"config"`
}

func (s *Service) enabledConfig() (Config, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return Config{}, err
	}
	if !cfg.Enabled {
		return Config{}, ErrDisabled
	}
	return cfg, nil
}

func (s *Service) captureJPEG(ctx context.Context, crop *Crop) ([]byte, error) {
	frame, err := s.capture.Capture(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}
	if crop == nil {
		return frame.JPEG, nil
	}
	return cropJPEG(frame.JPEG, *crop)
}

func jpegImage(data []byte) Image {
	return Image{Mime: "image/jpeg", B64: base64.StdEncoding.EncodeToString(data)}
}

// Ask runs content.js handleAnswer (mcq), handleFRQAnswer (frq) or
// handleCustomMessage (custom) up to the answer.
func (s *Service) Ask(ctx context.Context, req AskRequest) (AskResult, error) {
	cfg, err := s.enabledConfig()
	if err != nil {
		return AskResult{}, err
	}
	var questionText string
	var images []Image
	switch req.Kind {
	case "mcq", "frq":
		shot, err := s.captureJPEG(ctx, req.Crop)
		if err != nil {
			return AskResult{}, err
		}
		images = append(s.contexts.Snapshot(), jpegImage(shot))
	case "custom":
		if strings.TrimSpace(req.Text) == "" {
			return AskResult{}, ErrNoAnswer
		}
		questionText = req.Text
		images = s.contexts.Snapshot()
	default:
		return AskResult{}, ErrInvalidKind
	}
	p, err := s.prompts()
	if err != nil {
		return AskResult{}, err
	}
	answer, route, err := s.askLLM(ctx, cfg, questionText, images, getPrompt(p, req.Kind), s.attachments.Load())
	if err != nil {
		return AskResult{Route: route}, err
	}
	if answer == "" {
		return AskResult{Route: route}, ErrNoAnswer
	}
	return AskResult{Answer: answer, Route: route}, nil
}

// askLLM ports content.js askLLM: one turn, or with attachments a question
// anchor answered "OK" (thinking off) followed by the attachments and prompt.
func (s *Service) askLLM(ctx context.Context, cfg Config, questionText string, images []Image, answerPrompt string, atts []Attachment) (string, string, error) {
	provider := cfg.Provider
	qText := strings.TrimSpace(questionText)
	prompt := strings.TrimSpace(answerPrompt)
	lead := ""
	if qText != "" {
		lead = qText + "\n\n"
	}

	if len(atts) == 0 {
		turn := Turn{Role: "user", Text: lead + prompt, Images: images}
		data, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, []Turn{turn}, nil), "API")
		if err != nil {
			return "", route, err
		}
		answer, _ := ParseResponse(provider, data)
		return answer, route, nil
	}

	turn1 := Turn{Role: "user", Text: lead + ackPrompt, Images: images}
	off := false
	data1, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, []Turn{turn1}, &off), "Q-anchor")
	if err != nil {
		return "", route, err
	}
	ack, _ := ParseResponse(provider, data1)
	if strings.TrimSpace(ack) == "" {
		ack = "OK"
	}
	turn2 := Turn{Role: "user", Text: prompt + "\n\nNow answer the question above.", Attachments: atts}
	turns := []Turn{turn1, {Role: "assistant", Text: ack}, turn2}
	data2, route, err := s.apiSend(ctx, cfg, BuildConversation(provider, cfg, turns, nil), "API")
	if err != nil {
		return "", route, err
	}
	answer, _ := ParseResponse(provider, data2)
	return answer, route, nil
}

// apiSend ports content.js apiSend: any failure or non-2xx is an error whose
// text is the provider's error.message, else "HTTP n[: first 300 chars]".
func (s *Service) apiSend(ctx context.Context, cfg Config, req Request, label string) (json.RawMessage, string, error) {
	r, err := s.transport.Send(ctx, req, cfg.ProxyURL, cfg.ProxyPass)
	if err != nil {
		return nil, r.Route, err
	}
	log.Infof("assistant: %s route: %s", label, r.Route)
	if r.OK {
		return r.Data, r.Route, nil
	}
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if r.Data != nil && json.Unmarshal(r.Data, &e) == nil && e.Error.Message != "" {
		return nil, r.Route, errors.New(e.Error.Message)
	}
	if r.BodyText != "" {
		text := []rune(r.BodyText)
		if len(text) > 300 {
			text = text[:300]
		}
		return nil, r.Route, fmt.Errorf("HTTP %d: %s", r.Status, string(text))
	}
	return nil, r.Route, fmt.Errorf("HTTP %d", r.Status)
}

func (s *Service) AddContext(ctx context.Context, crop *Crop) (int, error) {
	if _, err := s.enabledConfig(); err != nil {
		return 0, err
	}
	shot, err := s.captureJPEG(ctx, crop)
	if err != nil {
		return s.contexts.Count(), err
	}
	return s.contexts.Add(jpegImage(shot))
}

func (s *Service) ClearContexts() error {
	if _, err := s.enabledConfig(); err != nil {
		return err
	}
	s.contexts.Clear()
	return nil
}

func (s *Service) ContextCount() (int, error) {
	if _, err := s.enabledConfig(); err != nil {
		return 0, err
	}
	return s.contexts.Count(), nil
}

func (s *Service) Screenshot(ctx context.Context) ([]byte, error) {
	if _, err := s.enabledConfig(); err != nil {
		return nil, err
	}
	return s.captureJPEG(ctx, nil)
}

func (s *Service) AdjustReasoning(up bool) (ReasoningResult, error) {
	var res ReasoningResult
	cfg, err := updateConfig(func(c Config) (Config, error) {
		if !c.Enabled {
			return c, ErrDisabled
		}
		next, label, changed := adjustReasoning(c, up)
		res.Label, res.Changed = label, changed
		return next, nil
	})
	if err != nil {
		return ReasoningResult{}, err
	}
	res.Config = publicConfig(cfg)
	return res, nil
}
