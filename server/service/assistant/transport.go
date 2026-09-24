package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Port of background.js performApiRequest: direct when no proxy; otherwise the
// chaice relay, 2 attempts 5 s apart, then direct. Thinking calls can take
// minutes, so each attempt gets 300 s to produce response headers.

type Result struct {
	OK       bool
	Status   int
	Data     json.RawMessage
	BodyText string
	Route    string
}

type Transport struct {
	Client         *http.Client
	AttemptTimeout time.Duration
	RetryGap       time.Duration
	MaxAttempts    int
}

func NewTransport() *Transport {
	return &Transport{Client: &http.Client{}, AttemptTimeout: 300 * time.Second, RetryGap: 5 * time.Second, MaxAttempts: 2}
}

func (t *Transport) Send(ctx context.Context, req Request, proxyURL, proxyPass string) (Result, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		r, err := t.direct(ctx, req)
		r.Route = "direct (no proxy configured)"
		return r, err
	}

	for attempt := 1; attempt <= t.MaxAttempts; attempt++ {
		r, err := t.viaProxy(ctx, req, proxyURL, proxyPass)
		if err == nil {
			r.Route = fmt.Sprintf("proxy %s (attempt %d/%d)", proxyURL, attempt, t.MaxAttempts)
			return r, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		log.Warnf("assistant: proxy attempt %d/%d failed: %v", attempt, t.MaxAttempts, err)
		if attempt < t.MaxAttempts {
			select {
			case <-time.After(t.RetryGap):
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		}
	}
	log.Warnf("assistant: proxy unreachable after %d attempts; falling back to direct", t.MaxAttempts)
	r, err := t.direct(ctx, req)
	r.Route = fmt.Sprintf("direct (FALLBACK after %d failed proxy attempts to %s)", t.MaxAttempts, proxyURL)
	return r, err
}

// post returns once headers arrive; the attempt timeout stops there, as
// fetchWithTimeout's does. cancel must be called after the body is read.
func (t *Transport) post(ctx context.Context, target string, headers map[string]string, body []byte) (*http.Response, context.CancelFunc, error) {
	attemptCtx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(t.AttemptTimeout, cancel)
	hr, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		timer.Stop()
		cancel()
		return nil, nil, redact(err)
	}
	for k, v := range headers {
		hr.Header.Set(k, v)
	}
	resp, err := t.Client.Do(hr)
	timer.Stop()
	if err != nil {
		cancel()
		return nil, nil, redact(err)
	}
	return resp, cancel, nil
}

// A *url.Error prints the URL, and a Gemini URL carries the API key.
func redact(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s request failed: %w", ue.Op, ue.Err)
	}
	return err
}

func (t *Transport) direct(ctx context.Context, req Request) (Result, error) {
	resp, cancel, err := t.post(ctx, req.URL, req.Headers, req.Body)
	if err != nil {
		return Result{}, err
	}
	defer cancel()
	defer resp.Body.Close()
	return readResponse(resp)
}

func (t *Transport) viaProxy(ctx context.Context, req Request, proxyURL, proxyPass string) (Result, error) {
	headers := make(map[string]string, len(req.Headers)+2)
	for k, v := range req.Headers {
		headers[k] = v
	}
	headers["X-Target-URL"] = req.URL
	if proxyPass != "" {
		headers["X-Proxy-Pass"] = proxyPass
	}
	resp, cancel, err := t.post(ctx, proxyURL, headers, req.Body)
	if err != nil {
		return Result{}, err
	}
	defer cancel()
	defer resp.Body.Close()

	if resp.Header.Get("X-Relay-Wrap") != "1" {
		return readResponse(resp)
	}
	var wrap struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return Result{}, fmt.Errorf("relay wrap: %w", err)
	}
	r := Result{OK: wrap.Status >= 200 && wrap.Status < 300, Status: wrap.Status}
	body := bytes.TrimSpace(wrap.Body)
	switch {
	case len(body) > 0 && (body[0] == '{' || body[0] == '['):
		r.Data = json.RawMessage(body)
	case len(body) > 0 && body[0] == '"':
		_ = json.Unmarshal(body, &r.BodyText)
	}
	return r, nil
}

func readResponse(resp *http.Response) (Result, error) {
	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, redact(err)
	}
	r := Result{OK: resp.StatusCode >= 200 && resp.StatusCode < 300, Status: resp.StatusCode}
	trimmed := bytes.TrimSpace(text)
	if len(trimmed) > 0 && json.Valid(trimmed) && !bytes.Equal(trimmed, []byte("null")) {
		r.Data = json.RawMessage(trimmed)
	} else if len(text) > 0 {
		r.BodyText = string(text)
	}
	return r, nil
}
