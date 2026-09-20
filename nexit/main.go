// nexit is the NanoKVM exit client. It holds one WebSocket open to a NanoKVM
// and originates every TCP connection and UDP datagram the machine behind that
// NanoKVM asks for, so that machine's internet egress happens here.
//
// It exists for hosts with no usable scripting host, where client.ps1 and
// client.sh cannot run. Configuration is passed in, never compiled in: the
// binary is identical for every device and every slot, so its hash is stable
// and an antivirus verdict or an allow-list entry keeps applying to it.
//
//	nexit.exe wss://nanokvm.local/exit/0 --passcode f8n9yze8
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("nexit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	passcode := fs.String("passcode", "", "slot passcode from the NanoKVM's Exit panel (required)")
	allowPrivate := fs.Bool("allow-private", false, "also serve private destinations, meaning this machine's own LAN")
	insecure := fs.Bool("insecure", false, "do not verify the NanoKVM's TLS certificate (needed for its self-signed default)")
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.Usage = func() { usage(fs) }

	// flag stops at the first non-flag argument, so parse in rounds and let
	// the address sit anywhere on the line: people write the address first.
	address := ""
	args := os.Args[1:]
	for {
		if err := fs.Parse(args); err != nil {
			return 2
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		if address != "" {
			logf("expected one NanoKVM address, got %q and %q", address, rest[0])
			usage(fs)
			return 2
		}
		address, args = rest[0], rest[1:]
	}
	if *showVersion {
		fmt.Printf("nexit %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return 0
	}
	if address == "" {
		usage(fs)
		return 2
	}

	endpoint, err := parseEndpoint(address)
	if err != nil {
		logf("%s", err)
		return 2
	}
	if *passcode == "" {
		logf("a passcode is required; the Exit panel shows it")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logf("nexit %s for %s (allowPrivate=%t, verifyTLS=%t)",
		version, endpoint.display, *allowPrivate, endpoint.tls && !*insecure)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+*passcode)
	header.Set("User-Agent", "nexit/1 ("+runtime.GOOS+")")

	host, _ := os.Hostname()
	pol := policy{allowPrivate: *allowPrivate}

	backoff := time.Second
	for {
		s := &session{pol: pol, hostname: host, os: runtime.GOOS}
		err := s.dial(ctx, endpoint.ws, header, endpoint.tls && *insecure)
		if err == nil {
			logf("connected: streamWindow=%d connWindow=%d maxStreams=%d",
				s.welcome.streamWindow, s.welcome.connWindow, s.welcome.maxStreams)
			backoff = time.Second // a session that reached WELCOME resets it
			werr := s.wait()
			if ctx.Err() != nil {
				logf("stopping")
				return 0
			}
			logf("session ended: %s", reason(werr))
		} else {
			if ctx.Err() != nil {
				logf("stopping")
				return 0
			}
			logf("connect failed: %s", reason(err))
		}

		logf("reconnecting in %s", backoff.Round(time.Second))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			logf("stopping")
			return 0
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

type endpoint struct {
	ws      string // ws:// or wss:// url of the slot's native route
	display string
	tls     bool
}

// parseEndpoint accepts the address the Exit panel shows, with or without the
// scheme and with or without the /exit/<slot> path, and returns the WebSocket
// url of that slot's native route.
func parseEndpoint(raw string) (endpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return endpoint{}, errors.New("no NanoKVM address given")
	}
	if !strings.Contains(raw, "://") {
		raw = "wss://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return endpoint{}, fmt.Errorf("bad address %q: %w", raw, err)
	}

	secure := true
	switch u.Scheme {
	case "wss", "https":
		u.Scheme = "wss"
	case "ws", "http":
		u.Scheme = "ws"
		secure = false
	default:
		return endpoint{}, fmt.Errorf("address scheme %q is not one of ws, wss, http, https", u.Scheme)
	}
	if u.Host == "" {
		return endpoint{}, fmt.Errorf("address %q names no host", raw)
	}

	// Accept "host", "host/exit/0" and "host/exit/0/native" alike.
	path := strings.Trim(u.Path, "/")
	switch {
	case path == "":
		path = "exit/0/native"
	case strings.HasSuffix(path, "/native"):
	default:
		path += "/native"
	}
	if !strings.HasPrefix(path, "exit/") {
		return endpoint{}, fmt.Errorf("address %q does not name an exit slot, expected .../exit/<slot>", raw)
	}
	u.Path = "/" + path

	display := u.Scheme + "://" + u.Host + "/" + strings.TrimSuffix(path, "/native")
	return endpoint{ws: u.String(), display: display, tls: secure}, nil
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `nexit %s: NanoKVM exit client

The machine running this becomes the internet egress for whatever is plugged
into the NanoKVM's USB port. It keeps reconnecting until you stop it with
Ctrl+C.

usage:
  nexit <nanokvm-address> --passcode <passcode> [flags]

examples:
  nexit wss://nanokvm.local/exit/0 --passcode f8n9yze8
  nexit https://192.168.1.50/exit/0 --passcode f8n9yze8 --insecure

flags:
`, version)
	fs.PrintDefaults()
}

// reason trims the noise off a transport error so the log stays readable.
func reason(err error) string {
	if err == nil {
		return "closed"
	}
	msg := err.Error()
	if i := strings.Index(msg, "websocket: close "); i >= 0 {
		return msg[i+len("websocket: "):]
	}
	return msg
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s nexit: %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func insecureTLS() *tls.Config {
	// The NanoKVM's default certificate is self-signed, so no CA store can
	// vouch for it. The passcode in the Authorization header is what
	// authenticates the session, exactly as it is for wstunnel against the
	// same host.
	return &tls.Config{InsecureSkipVerify: true} //nolint:gosec // documented above
}

// hasDefaultRoute reports whether this machine has an IPv4 default route,
// which HELLO's flag bit 0 tells the NanoKVM. No packet is sent: connecting a
// UDP socket only does the route lookup.
func hasDefaultRoute() bool {
	c, err := net.Dial("udp4", "1.1.1.1:53")
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}
