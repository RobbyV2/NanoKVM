package exit

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/utils"
)

// testAddrs is where tests register the ephemeral addresses that stand in
// for a slot's fixed loopback ports. The package vars are set once, in
// TestMain, to closures reading this table, so no test ever reassigns a var
// that a goroutine of the previous test might still read.
var testAddrs = struct {
	sync.Mutex
	socks    map[string]string
	reverse  map[string]string
	wsServer map[string]string
}{
	socks:    map[string]string{},
	reverse:  map[string]string{},
	wsServer: map[string]string{},
}

func setTestSocksAddr(slot Slot, addr string) {
	testAddrs.Lock()
	defer testAddrs.Unlock()
	testAddrs.socks[slot.ID] = addr
}

func setTestReverseAddr(slot Slot, addr string) {
	testAddrs.Lock()
	defer testAddrs.Unlock()
	testAddrs.reverse[slot.ID] = addr
}

func setTestWSServerAddr(slot Slot, addr string) {
	testAddrs.Lock()
	defer testAddrs.Unlock()
	testAddrs.wsServer[slot.ID] = addr
}

func lookupTestAddr(m map[string]string, slot Slot) string {
	testAddrs.Lock()
	defer testAddrs.Unlock()
	if a, ok := m[slot.ID]; ok {
		return a
	}
	return "127.0.0.1:0"
}

// listenLog records every address the package binds, so TestMain can prove
// nothing new listens on 0.0.0.0 (spec, "Processes per slot").
var listenLog = struct {
	sync.Mutex
	addrs []string
}{}

func recordListen(addr string) {
	listenLog.Lock()
	defer listenLog.Unlock()
	listenLog.addrs = append(listenLog.addrs, addr)
}

// TestMain shortens every protocol timer once for the whole binary and
// installs the address indirections. Setting them per test would race with
// goroutines of the previous test that still read them.
func TestMain(m *testing.M) {
	helloTimeout = 300 * time.Millisecond
	openTimeout = 500 * time.Millisecond
	pingInterval = 100 * time.Millisecond
	pingMisses = 2
	udpIdleTimeout = 400 * time.Millisecond
	probeInterval = 100 * time.Millisecond
	probeTimeout = 500 * time.Millisecond
	probeInitialDelay = 20 * time.Millisecond
	dnsQueryBudget = time.Second
	reverseProbeInterval = 20 * time.Millisecond
	relayDialTimeout = time.Second

	socksListenAddr = func(Slot) string { return "127.0.0.1:0" }
	socksDialAddr = func(s Slot) string { return lookupTestAddr(testAddrs.socks, s) }
	wstunnelReverseAddr = func(s Slot) string { return lookupTestAddr(testAddrs.reverse, s) }
	wstunnelServerAddr = func(s Slot) string { return lookupTestAddr(testAddrs.wsServer, s) }
	listenTCP = func(addr string) (net.Listener, error) {
		ln, err := utils.Listen(addr)
		if err == nil {
			recordListen(ln.Addr().String())
		}
		return ln, err
	}
	listenPacket = func(network, addr string) (net.PacketConn, error) {
		pc, err := net.ListenPacket(network, addr)
		if err == nil {
			recordListen(pc.LocalAddr().String())
		}
		return pc, err
	}

	code := m.Run()

	listenLog.Lock()
	addrs := listenLog.addrs
	listenLog.Unlock()
	if len(addrs) == 0 {
		fmt.Fprintln(os.Stderr, "no listener was ever recorded; the wildcard check proved nothing")
		if code == 0 {
			code = 1
		}
	}
	for _, a := range addrs {
		ap, err := netip.ParseAddrPort(a)
		if err != nil || ap.Addr().IsUnspecified() {
			fmt.Fprintf(os.Stderr, "listener bound to a wildcard address: %s\n", a)
			code = 1
		}
	}
	os.Exit(code)
}
