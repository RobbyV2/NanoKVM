package assistant

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type fixtureCase struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Settings Config `json:"settings"`
	Thinking *bool  `json:"thinking"`
	Turns    []struct {
		Role        string   `json:"role"`
		Text        string   `json:"text"`
		Images      []string `json:"images"`
		Attachments []struct {
			Name string `json:"name"`
			B64  string `json:"b64"`
			Text string `json:"text"`
		} `json:"attachments"`
	} `json:"turns"`
}

type fixture struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

func TestBuildConversationMatchesLLMJS(t *testing.T) {
	var cases []fixtureCase
	var fixtures map[string]fixture
	mustReadJSON(t, "testdata/cases.json", &cases)
	mustReadJSON(t, "testdata/fixtures.json", &fixtures)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var turns []Turn
			for _, ct := range c.Turns {
				turn := Turn{Role: ct.Role, Text: ct.Text}
				for _, b64 := range ct.Images {
					turn.Images = append(turn.Images, Image{Mime: "image/png", B64: b64})
				}
				for _, a := range ct.Attachments {
					ft := classifyFile(a.Name)
					att := Attachment{Name: a.Name, Kind: ft.Kind, Mime: ft.Mime, AudioFormat: ft.AudioFormat, B64: a.B64}
					if ft.Kind == "text" {
						att.Text = a.Text
					}
					turn.Attachments = append(turn.Attachments, att)
				}
				turns = append(turns, turn)
			}
			got := BuildConversation(c.Provider, c.Settings, turns, c.Thinking)
			want := fixtures[c.Name]
			if got.URL != want.URL {
				t.Fatalf("url\n got %s\nwant %s", got.URL, want.URL)
			}
			if !reflect.DeepEqual(got.Headers, want.Headers) {
				t.Fatalf("headers\n got %v\nwant %v", got.Headers, want.Headers)
			}
			var body any
			if err := json.Unmarshal(got.Body, &body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, want.Body) {
				t.Fatalf("body\n got %s\nwant %v", got.Body, want.Body)
			}
		})
	}
}

func TestScreenshotPartsCarryTheirMime(t *testing.T) {
	turns := []Turn{{Role: "user", Text: "Q", Images: []Image{{Mime: "image/jpeg", B64: "eA=="}}}}
	or := BuildConversation("openrouter", defaultConfig(), turns, nil)
	if !strings.Contains(string(or.Body), `"data:image/jpeg;base64,eA=="`) {
		t.Fatalf("openrouter body %s", or.Body)
	}
	g := BuildConversation("gemini", defaultConfig(), turns, nil)
	if !strings.Contains(string(g.Body), `"mime_type":"image/jpeg"`) {
		t.Fatalf("gemini body %s", g.Body)
	}
}

func TestParseResponse(t *testing.T) {
	cases := []struct {
		provider, data, want string
		ok                   bool
	}{
		{"openrouter", `{"choices":[{"message":{"content":"B"}}]}`, "B", true},
		{"openrouter", `{"choices":[]}`, "", false},
		{"openrouter", `{"choices":[{"message":{"content":null}}]}`, "", false},
		{"gemini", `{"candidates":[{"content":{"parts":[{"text":"C"}]}}]}`, "C", true},
		{"gemini", `{"candidates":[{"content":{"parts":[]}}]}`, "", false},
		{"gemini", `"not an object"`, "", false},
	}
	for _, c := range cases {
		got, ok := ParseResponse(c.provider, json.RawMessage(c.data))
		if got != c.want || ok != c.ok {
			t.Fatalf("%s %s: got %q %v", c.provider, c.data, got, ok)
		}
	}
}

func mustReadJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}
