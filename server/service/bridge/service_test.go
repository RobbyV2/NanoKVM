package bridge

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// The gap this covers is not a wrong value but an absent one: with Gadget nil
// the twelve steps that hold the management address all pass, the response says
// enabled, and usb0 is silently never enslaved.
func TestServiceWiresTheGadget(t *testing.T) {
	service := newService(&fakeLiveness{})

	if service.manager.gadget == nil {
		t.Fatal("bridge.NewService leaves Gadget nil, so enable step 13 never enslaves usb0")
	}
}

// Gate three's strong form. Without a middleware calling Record, Observed can
// never be true, every apply reports inboundWeak and the wire is never proven.
func TestRecordListenerSatisfiesTheStrongInboundGate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		record  bool
		observe bool
	}{
		{name: "middleware registered", record: true, observe: true},
		{name: "nobody watching", record: false, observe: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := NewListenerWitness("http", 80)

			engine := gin.New()
			if test.record {
				engine.Use(RecordListener(w))
			}
			engine.GET("/api/vm/info", func(c *gin.Context) { c.Status(http.StatusOK) })

			server := httptest.NewServer(engine)
			defer server.Close()

			since := time.Now()
			rsp, err := http.Get(server.URL + "/api/vm/info")
			if err != nil {
				t.Fatalf("GET %s: %v", server.URL, err)
			}
			_ = rsp.Body.Close()

			host, _, err := net.SplitHostPort(server.Listener.Addr().String())
			if err != nil {
				t.Fatalf("split %s: %v", server.Listener.Addr(), err)
			}
			if got := w.Observed(host, since); got != test.observe {
				t.Fatalf("Observed(%s) = %t, want %t", host, got, test.observe)
			}
		})
	}
}

// Observed is a claim about one address at or after one instant, and both
// halves are load-bearing: the address, because a request that arrived on the
// wlan0 AP proves nothing about the uplink under test; the instant, because a
// request from before the apply began arrived at the old configuration.
func TestObservedIsScopedToTheAddressAndTheApply(t *testing.T) {
	w := NewListenerWitness("http", 80)

	before := time.Now().Add(-time.Minute)
	w.Record("192.168.1.50:443")
	after := time.Now().Add(time.Minute)

	if !w.Observed("192.168.1.50", before) {
		t.Error("a request recorded after the apply began did not satisfy the gate")
	}
	if w.Observed("192.168.1.50", after) {
		t.Error("a request recorded before the apply began satisfied the gate")
	}
	if w.Observed("10.0.0.1", before) {
		t.Error("a request to another address satisfied the gate")
	}
}
