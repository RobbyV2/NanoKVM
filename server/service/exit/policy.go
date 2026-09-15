package exit

import "net/netip"

// Policy is the destination policy of D21, applied by the front door at
// CONNECT/UDP send, by the mux at OPEN, mirrored in the native clients, and
// rendered into the EXIT<n> chain as REJECT rules by S94exit.
type Policy struct {
	// AllowPrivate admits RFC 1918, CGNAT and ULA destinations (the exit
	// device's own LAN). Default false.
	AllowPrivate bool
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

// Allow reports whether dst may be opened through the exit. IPv4-mapped IPv6
// addresses are judged as IPv4.
func (p Policy) Allow(dst netip.Addr) bool {
	if !dst.IsValid() {
		return false
	}
	dst = dst.Unmap()
	if dst.Is4() {
		return !p.deny(dst, alwaysDenied4, privateDenied4)
	}
	return !p.deny(dst, alwaysDenied6, privateDenied6)
}

func (p Policy) deny(a netip.Addr, always, private []netip.Prefix) bool {
	for _, pfx := range always {
		if pfx.Contains(a) {
			return true
		}
	}
	if p.AllowPrivate {
		return false
	}
	for _, pfx := range private {
		if pfx.Contains(a) {
			return true
		}
	}
	return false
}

// RejectPrefixes4 lists the IPv4 prefixes S94exit must REJECT in the EXIT<n>
// chain for this policy, in rule order.
func (p Policy) RejectPrefixes4() []string {
	out := make([]string, 0, len(alwaysDenied4)+len(privateDenied4))
	for _, pfx := range alwaysDenied4 {
		out = append(out, pfx.String())
	}
	if !p.AllowPrivate {
		for _, pfx := range privateDenied4 {
			out = append(out, pfx.String())
		}
	}
	return out
}
