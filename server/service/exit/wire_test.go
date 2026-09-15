package exit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

// The real factory: every component is the real type, the counter reaches
// the mux and the front door only, and the door reports "connecting" for a
// Mode A upgrade whose HELLO is still pending.
func TestNewComponentsWiresTheRealDataplane(t *testing.T) {
	slot := MustSlot("0")
	counter := NewCounter()
	c := newComponents(slot, componentDeps{
		Policy:    func() Policy { return Policy{AllowPrivate: true} },
		Bytes:     counter,
		Upstreams: func() []netip.Addr { return nil },
		Connected: func() bool { return false },
	})

	door, ok := c.Door.(*wiredDoor)
	if !ok {
		t.Fatalf("Door is %T", c.Door)
	}
	mux, ok := c.Mux.(*Mux)
	if !ok {
		t.Fatalf("Mux is %T", c.Mux)
	}
	proxy, ok := c.Proxy.(*WSProxy)
	if !ok {
		t.Fatalf("Proxy is %T", c.Proxy)
	}
	if _, ok := c.DNS.(*Forwarder); !ok {
		t.Fatalf("DNS is %T", c.DNS)
	}
	if _, ok := c.Probe.(*Prober); !ok {
		t.Fatalf("Probe is %T", c.Probe)
	}
	if door.mux != mux {
		t.Fatal("the door does not consult the slot's own mux")
	}
	if mux.hooks.Bytes != ByteCounter(counter) || door.FrontDoor.bytes != ByteCounter(counter) {
		t.Fatal("the counter did not reach the mux and the front door")
	}
	if proxy.bytes != nil {
		t.Fatal("the WSProxy got a counter: Mode B would be counted twice")
	}

	// Connecting: an accepted upgrade with no HELLO yet.
	srv := httptest.NewServer(http.HandlerFunc(mux.ServeNative))
	t.Cleanup(srv.Close)
	if state, _, _ := c.Door.Attached(); state != proto.ExitDisconnected {
		t.Fatalf("idle state = %s", state)
	}
	f := NewFakeExit()
	f.NoHello = true
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := f.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/exit/0/native", nil); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "connecting", func() bool {
		state, _, _ := c.Door.Attached()
		return state == proto.ExitConnecting
	})
	// The HELLO timeout ends the attempt and the door is idle again.
	waitFor(t, "disconnected after the HELLO timeout", func() bool {
		state, _, _ := c.Door.Attached()
		return state == proto.ExitDisconnected
	})
	f.Close()
}
