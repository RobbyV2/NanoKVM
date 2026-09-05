package utils

import (
	"errors"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// listenTCP is net.ListenTCP without the standard library's listen path: the
// same socket options net sets (SO_REUSEADDR, and a dual-stack IPv6 socket for
// an unspecified address), our own backlog read, then net.FileListener over
// the bound descriptor.
func listenTCP(laddr *net.TCPAddr) (net.Listener, error) {
	family, sa := sockaddrFor(laddr)
	fd, err := unix.Socket(family, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if errors.Is(err, unix.EAFNOSUPPORT) && family == unix.AF_INET6 && laddr.IP == nil {
		// No IPv6 in this kernel: an unspecified address is then IPv4 only,
		// which is the choice net.Listen makes too.
		family = unix.AF_INET
		sa = &unix.SockaddrInet4{Port: laddr.Port}
		fd, err = unix.Socket(family, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	}
	if err != nil {
		return nil, listenError(laddr, "socket", err)
	}
	closeOnError := func(op string, err error) (net.Listener, error) {
		unix.Close(fd)
		return nil, listenError(laddr, op, err)
	}
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return closeOnError("setsockopt SO_REUSEADDR", err)
	}
	if family == unix.AF_INET6 {
		// "tcp" rather than "tcp6": an IPv6 socket also takes IPv4 connections.
		if err := unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, 0); err != nil {
			return closeOnError("setsockopt IPV6_V6ONLY", err)
		}
	}
	if err := unix.Bind(fd, sa); err != nil {
		return closeOnError("bind", err)
	}
	if err := unix.Listen(fd, listenBacklog()); err != nil {
		return closeOnError("listen", err)
	}
	// FileListener dups the descriptor, so ours closes here either way.
	file := os.NewFile(uintptr(fd), "listen "+laddr.String())
	defer file.Close()
	listener, err := net.FileListener(file)
	if err != nil {
		return nil, listenError(laddr, "file listener", err)
	}
	return listener, nil
}

// sockaddrFor picks the family the way net does for "tcp": an IPv4 address is
// AF_INET, anything else, the unspecified address included, is AF_INET6.
func sockaddrFor(laddr *net.TCPAddr) (int, unix.Sockaddr) {
	if ip4 := laddr.IP.To4(); ip4 != nil {
		sa := &unix.SockaddrInet4{Port: laddr.Port}
		copy(sa.Addr[:], ip4)
		return unix.AF_INET, sa
	}
	sa := &unix.SockaddrInet6{Port: laddr.Port}
	if laddr.IP != nil {
		copy(sa.Addr[:], laddr.IP.To16())
	}
	if laddr.Zone != "" {
		if iface, err := net.InterfaceByName(laddr.Zone); err == nil {
			sa.ZoneId = uint32(iface.Index)
		}
	}
	return unix.AF_INET6, sa
}
