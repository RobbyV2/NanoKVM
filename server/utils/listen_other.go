//go:build !linux

package utils

import "net"

// Only Linux reads /proc/sys on listen, so everywhere else the standard
// library's own listener is the right one.
func listenTCP(laddr *net.TCPAddr) (net.Listener, error) {
	return net.ListenTCP("tcp", laddr)
}
