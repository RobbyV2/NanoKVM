package download

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// The downloader fetches from the NanoKVM's own position on the network and
// stores the response where the library grants read it back. An administrator
// may fetch from anywhere, the LAN included; an account that only holds
// library.write must not reach what only the device can: its loopback
// listeners, the networks behind it, link-local addresses and the exit
// tunnels' range.
var errInternalDestination = errors.New("downloads from local or private addresses need an administrator")

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), // shared address space, tailnets
	netip.MustParsePrefix("198.18.0.0/15"), // the exit tunnels' addresses
}

func publicAddress(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() || addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsMulticast() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// checkPublicHost refuses a host that names or resolves to an internal address
// before the download starts, so the caller gets a message rather than a
// failed download. publicOnlyClient is what enforces it.
func checkPublicHost(ctx context.Context, host string) error {
	if addr, err := netip.ParseAddr(host); err == nil {
		if !publicAddress(addr) {
			return errInternalDestination
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}
	for _, addr := range addrs {
		if !publicAddress(addr) {
			return errInternalDestination
		}
	}
	return nil
}

// publicOnlyClient checks every address at connect time, which also covers
// redirects and a name that resolves differently once checked. No proxy is
// used: the check would then see the proxy instead of the destination.
func publicOnlyClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			addr, err := netip.ParseAddr(host)
			if err != nil || !publicAddress(addr) {
				return errInternalDestination
			}
			return nil
		},
	}
	transport.DialContext = dialer.DialContext
	return &http.Client{Transport: transport}
}
