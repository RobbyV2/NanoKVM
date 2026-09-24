package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Go port of the chaice extension's llm.js: turns -> provider-native request,
// response JSON -> answer. No network I/O.

const thinkingStep = 512

type Image struct {
	Mime string
	B64  string
}

type Attachment struct {
	Name        string
	Kind        string
	Mime        string
	AudioFormat string
	B64         string
	Text        string
}

type Turn struct {
	Role        string
	Text        string
	Images      []Image
	Attachments []Attachment
}

type Request struct {
	URL     string
	Headers map[string]string
	Body    []byte
}

type fileType struct {
	Kind        string
	Mime        string
	AudioFormat string
}

var fileTypes = map[string]fileType{
	"png": {"image", "image/png", ""}, "jpg": {"image", "image/jpeg", ""},
	"jpeg": {"image", "image/jpeg", ""}, "webp": {"image", "image/webp", ""},
	"gif": {"image", "image/gif", ""}, "heic": {"image", "image/heic", ""},
	"heif": {"image", "image/heif", ""},
	"pdf":  {"pdf", "application/pdf", ""},
	"mp3":  {"audio", "audio/mpeg", "mp3"}, "wav": {"audio", "audio/wav", "wav"},
	"ogg": {"audio", "audio/ogg", "ogg"}, "flac": {"audio", "audio/flac", "flac"},
	"aac": {"audio", "audio/aac", "aac"}, "m4a": {"audio", "audio/mp4", "m4a"},
	"aiff": {"audio", "audio/aiff", "aiff"},
	"mp4":  {"video", "video/mp4", ""}, "mov": {"video", "video/quicktime", ""},
	"webm": {"video", "video/webm", ""},
	"txt":  {"text", "text/plain", ""}, "md": {"text", "text/markdown", ""},
	"markdown": {"text", "text/markdown", ""}, "csv": {"text", "text/csv", ""},
	"tsv": {"text", "text/tab-separated-values", ""}, "json": {"text", "application/json", ""},
	"xml": {"text", "application/xml", ""}, "html": {"text", "text/html", ""},
	"htm": {"text", "text/html", ""}, "yaml": {"text", "text/yaml", ""},
	"yml": {"text", "text/yaml", ""}, "log": {"text", "text/plain", ""},
	"ini": {"text", "text/plain", ""}, "conf": {"text", "text/plain", ""},
}

var extPattern = regexp.MustCompile(`\.([^./\\]+)$`)

func classifyFile(name string) fileType {
	if m := extPattern.FindStringSubmatch(name); m != nil {
		if ft, ok := fileTypes[strings.ToLower(m[1])]; ok {
			return ft
		}
	}
	return fileType{Kind: "text", Mime: "text/plain"}
}

func formatTextAttachments(files []Attachment) string {
	var b strings.Builder
	b.WriteString("\n\nThe following text files were attached:\n")
	for _, f := range files {
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", f.Name, f.Text)
	}
	return b.String()
}

func rawBase64(s string) string {
	return strings.TrimPrefix(s, "data:image/png;base64,")
}

// encodeURIComponent matches JavaScript's: everything but A-Z a-z 0-9 -_.!~*'() is escaped.
func encodeURIComponent(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-_.!~*'()", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func marshalBody(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// --- Gemini ---

type geminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       *string       `json:"text,omitempty"`
	InlineData *geminiInline `json:"inline_data,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiGenerationConfig struct {
	ThinkingConfig geminiThinkingConfig `json:"thinkingConfig"`
}

type geminiBody struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

func geminiParts(t Turn) []geminiPart {
	parts := []geminiPart{}
	if t.Text != "" {
		text := t.Text
		parts = append(parts, geminiPart{Text: &text})
	}
	for _, img := range t.Images {
		parts = append(parts, geminiPart{InlineData: &geminiInline{MimeType: img.Mime, Data: rawBase64(img.B64)}})
	}
	for _, a := range t.Attachments {
		mime := a.Mime
		if a.Kind == "text" {
			mime = "text/plain"
		}
		parts = append(parts, geminiPart{InlineData: &geminiInline{MimeType: mime, Data: a.B64}})
	}
	return parts
}

func buildGemini(cfg Config, turns []Turn, thinking *bool) Request {
	base := strings.TrimRight(cfg.GeminiBaseURL, "/")
	url := base + "/models/" + cfg.GeminiModel + ":generateContent?key=" + encodeURIComponent(cfg.GeminiAPIKey)

	contents := make([]geminiContent, 0, len(turns))
	for _, t := range turns {
		role := "user"
		if t.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: geminiParts(t)})
	}
	body := geminiBody{Contents: contents}
	want := cfg.GeminiThinking
	if thinking != nil {
		want = *thinking
	}
	if want {
		budget := cfg.ThinkingBudget
		if budget == 0 {
			budget = thinkingStep
		}
		body.GenerationConfig = &geminiGenerationConfig{ThinkingConfig: geminiThinkingConfig{ThinkingBudget: budget}}
	}
	return Request{URL: url, Headers: map[string]string{"Content-Type": "application/json"}, Body: marshalBody(body)}
}

// --- OpenRouter ---

type orURL struct {
	URL string `json:"url"`
}

type orFile struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type orAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type orPart struct {
	Type       string   `json:"type"`
	Text       *string  `json:"text,omitempty"`
	ImageURL   *orURL   `json:"image_url,omitempty"`
	File       *orFile  `json:"file,omitempty"`
	InputAudio *orAudio `json:"input_audio,omitempty"`
	VideoURL   *orURL   `json:"video_url,omitempty"`
}

type orMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type orReasoning struct {
	Effort string `json:"effort"`
}

type orBody struct {
	Model     string       `json:"model"`
	Messages  []orMessage  `json:"messages"`
	Reasoning *orReasoning `json:"reasoning,omitempty"`
}

func openRouterContent(t Turn) []orPart {
	var textFiles, media []Attachment
	for _, a := range t.Attachments {
		if a.Kind == "text" {
			textFiles = append(textFiles, a)
		} else {
			media = append(media, a)
		}
	}
	text := t.Text
	if len(textFiles) > 0 {
		text += formatTextAttachments(textFiles)
	}
	content := []orPart{{Type: "text", Text: &text}}
	for _, img := range t.Images {
		content = append(content, orPart{Type: "image_url", ImageURL: &orURL{URL: "data:" + img.Mime + ";base64," + rawBase64(img.B64)}})
	}
	for _, a := range media {
		switch a.Kind {
		case "image":
			content = append(content, orPart{Type: "image_url", ImageURL: &orURL{URL: "data:" + a.Mime + ";base64," + a.B64}})
		case "pdf":
			content = append(content, orPart{Type: "file", File: &orFile{Filename: a.Name, FileData: "data:application/pdf;base64," + a.B64}})
		case "audio":
			format := a.AudioFormat
			if format == "" {
				format = "mp3"
			}
			content = append(content, orPart{Type: "input_audio", InputAudio: &orAudio{Data: a.B64, Format: format}})
		case "video":
			content = append(content, orPart{Type: "video_url", VideoURL: &orURL{URL: "data:" + a.Mime + ";base64," + a.B64}})
		}
	}
	return content
}

func buildOpenRouter(cfg Config, turns []Turn, thinking *bool) Request {
	base := cfg.ORBaseURL
	if base == "" {
		base = defaultORBaseURL
	}
	url := strings.TrimRight(base, "/") + "/chat/completions"

	messages := make([]orMessage, 0, len(turns))
	for _, t := range turns {
		if t.Role == "assistant" {
			messages = append(messages, orMessage{Role: "assistant", Content: t.Text})
		} else {
			messages = append(messages, orMessage{Role: "user", Content: openRouterContent(t)})
		}
	}
	body := orBody{Model: cfg.ORModel, Messages: messages}
	want := cfg.ORReasoning
	if thinking != nil {
		want = *thinking
	}
	if want {
		effort := cfg.ORReasoningEffort
		if effort == "" {
			effort = "low"
		}
		body.Reasoning = &orReasoning{Effort: effort}
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + cfg.ORAPIKey,
		"HTTP-Referer":  "https://github.com/chaice",
		"X-Title":       "ChAIce",
	}
	return Request{URL: url, Headers: headers, Body: marshalBody(body)}
}

// BuildConversation: provider "openrouter", anything else is Gemini (llm.js default).
func BuildConversation(provider string, cfg Config, turns []Turn, thinking *bool) Request {
	if provider == "openrouter" {
		return buildOpenRouter(cfg, turns, thinking)
	}
	return buildGemini(cfg, turns, thinking)
}

func ParseResponse(provider string, data json.RawMessage) (string, bool) {
	if provider == "openrouter" {
		var r struct {
			Choices []struct {
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(data, &r) != nil || len(r.Choices) == 0 {
			return "", false
		}
		s, ok := r.Choices[0].Message.Content.(string)
		return s, ok
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text any `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal(data, &r) != nil || len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", false
	}
	s, ok := r.Candidates[0].Content.Parts[0].Text.(string)
	return s, ok
}
