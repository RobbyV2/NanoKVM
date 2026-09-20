package main

import "net/netip"

// policy is the destination policy of D21, the same one the server applies at
// OPEN and S94exit renders into the EXIT<n> chain. It is enforced here too so
// a kvm that asked for a denied destination is refused at the exit as well,
// rather than relying on the other end to have asked correctly.
type policy struct {
	// allowPrivate admits RFC 1918, CGNAT and ULA destinations, meaning this
	// machine's own LAN. Off unless the slot says otherwise.
	allowPrivate bool
}

var (
	alwaysDenied4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("169.254.0.0/16"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("224.0.0.0/3"),
	}
	privateDenied4 = []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
	}
	alwaysDenied6 = []netip.Prefix{
		netip.MustParsePrefix("::/128"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fe80::/10"),
		netip.MustParsePrefix("ff00::/8"),
	}
	privateDenied6 = []netip.Prefix{
		netip.MustParsePrefix("fc00::/7"),
	}
)

// allow reports whether dst may be opened. IPv4-mapped IPv6 is judged as IPv4.
func (p policy) allow(dst netip.Addr) bool {
	if !dst.IsValid() {
		return false
	}
	dst = dst.Unmap()
	if dst.Is4() {
		return !p.deny(dst, alwaysDenied4, privateDenied4)
	}
	return !p.deny(dst, alwaysDenied6, privateDenied6)
}

func (p policy) deny(a netip.Addr, always, private []netip.Prefix) bool {
	for _, pfx := range always {
		if pfx.Contains(a) {
			return true
		}
	}
	if p.allowPrivate {
		return false
	}
	for _, pfx := range private {
		if pfx.Contains(a) {
			return true
		}
	}
	return false
}
