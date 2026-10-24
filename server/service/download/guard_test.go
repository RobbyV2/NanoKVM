package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestPublicAddressRefusesWhatOnlyTheDeviceReaches(t *testing.T) {
	for addr, want := range map[string]bool{
		"1.1.1.1":           true,
		"2606:4700::1111":   true,
		"127.0.0.1":         false,
		"::1":               false,
		"10.163.245.1":      false,
		"192.168.8.1":       false,
		"172.16.0.5":        false,
		"169.254.169.254":   false,
		"fe80::1":           false,
		"fc00::1":           false,
		"100.100.100.100":   false,
		"198.18.0.1":        false,
		"0.0.0.0":           false,
		"224.0.0.1":         false,
		"::ffff:127.0.0.1":  false,
		"::ffff:93.184.1.1": true,
		"0.1.2.3":           false,
		"250.1.2.3":         false,
		"::7f00:1":          false,
		"64:ff9b::7f00:1":   false,
		"64:ff9b::c0a8:801": false,
		"64:ff9b::101:101":  true,
		"64:ff9b:1::1":      false,
		"2001:0:4136::1":    false,
		"2002:7f00:1::1":    false,
	} {
		if got := publicAddress(netip.MustParseAddr(addr)); got != want {
			t.Errorf("publicAddress(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestCheckPublicHostRefusesInternalLiterals(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "192.168.8.210", "::1", "localhost"} {
		if err := checkPublicHost(context.Background(), host); !errors.Is(err, errInternalDestination) {
			t.Errorf("checkPublicHost(%q) = %v, want the internal-destination refusal", host, err)
		}
	}
}

// The connect-time check is what a redirect or a re-resolved name meets.
func TestPublicOnlyClientRefusesALoopbackServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := publicOnlyClient().Get(server.URL)
	if !errors.Is(err, errInternalDestination) {
		t.Fatalf("GET %s = %v, want the internal-destination refusal", server.URL, err)
	}
	resp, err := http.DefaultClient.Get(server.URL)
	if err != nil {
		t.Fatalf("the administrator's client cannot reach the test server: %v", err)
	}
	resp.Body.Close()
}
