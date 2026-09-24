package assistant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testTransport() *Transport {
	return &Transport{Client: &http.Client{}, AttemptTimeout: 2 * time.Second, RetryGap: time.Millisecond, MaxAttempts: 2}
}

func provider(t *testing.T, status int, body string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSendDirect(t *testing.T) {
	p := provider(t, 200, `{"a":1}`)
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL, Body: []byte("{}")}, "", "")
	if err != nil || !r.OK || string(r.Data) != `{"a":1}` || r.Route != "direct (no proxy configured)" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendNonJSONBody(t *testing.T) {
	p := provider(t, 502, "bad gateway")
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL}, "", "")
	if err != nil || r.OK || r.Status != 502 || r.BodyText != "bad gateway" || r.Data != nil {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendViaRelayUnwraps(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Target-URL") != "https://up/x" || r.Header.Get("X-Proxy-Pass") != "pw" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("relay headers %v", r.Header)
		}
		w.Header().Set("X-Relay-Wrap", "1")
		io.WriteString(w, `{"_relayWrap":1,"pad":"   ","status":200,"headers":{},"body":{"ok":true}}`)
	}))
	defer relay.Close()
	req := Request{URL: "https://up/x", Headers: map[string]string{"Authorization": "Bearer k"}}
	r, err := testTransport().Send(context.Background(), req, " "+relay.URL+" ", "pw")
	if err != nil || !r.OK || string(r.Data) != `{"ok":true}` || r.Route != "proxy "+relay.URL+" (attempt 1/2)" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendRelayStringBody(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Relay-Wrap", "1")
		io.WriteString(w, `{"_relayWrap":1,"pad":"","status":500,"body":"upstream down"}`)
	}))
	defer relay.Close()
	r, err := testTransport().Send(context.Background(), Request{URL: "https://up"}, relay.URL, "")
	if err != nil || r.OK || r.Status != 500 || r.BodyText != "upstream down" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendRelayAuthFailureIsReturnedNotRetried(t *testing.T) {
	calls := 0
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(401)
		io.WriteString(w, `{"error":{"message":"invalid proxy password"}}`)
	}))
	defer relay.Close()
	r, err := testTransport().Send(context.Background(), Request{URL: "https://up"}, relay.URL, "x")
	if err != nil || r.Status != 401 || calls != 1 {
		t.Fatalf("r=%+v err=%v calls=%d", r, err, calls)
	}
}

func TestSendFallsBackToDirect(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	p := provider(t, 200, `{"b":2}`)
	r, err := testTransport().Send(context.Background(), Request{URL: p.URL}, deadURL, "")
	want := "direct (FALLBACK after 2 failed proxy attempts to " + deadURL + ")"
	if err != nil || string(r.Data) != `{"b":2}` || r.Route != want {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendAttemptTimeoutCountsAsFailure(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer slow.Close()
	p := provider(t, 200, `{}`)
	tr := testTransport()
	tr.AttemptTimeout = 20 * time.Millisecond
	r, err := tr.Send(context.Background(), Request{URL: p.URL}, slow.URL, "")
	if err != nil || !strings.HasPrefix(r.Route, "direct (FALLBACK") {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendHonoursContextCancel(t *testing.T) {
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer hang.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := testTransport().Send(ctx, Request{URL: hang.URL}, "", "")
	if err == nil || time.Since(start) > time.Second {
		t.Fatalf("err=%v after %s", err, time.Since(start))
	}
}

func TestDirectFailureDoesNotLeakKey(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	u := dead.URL + "/models/m:generateContent?key=SECRETKEY"
	dead.Close()
	_, err := testTransport().Send(context.Background(), Request{URL: u}, "", "")
	if err == nil || strings.Contains(err.Error(), "SECRETKEY") {
		t.Fatalf("err=%v", err)
	}
}
