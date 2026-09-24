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
//
// Started without an address, as a double-click does, it reads nexit.json
// instead; see configCandidates for where it looks.
package main

import (
	"bufio"
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
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	code := run()
	if code != 0 && len(os.Args) == 1 {
		// Started with no arguments, most likely by a double-click: keep
		// the console window open so the message can be read.
		waitForEnter()
	}
	os.Exit(code)
}

// waitForEnter blocks until Enter is pressed, but only when stdin is a
// console, so a redirected or detached run never hangs.
func waitForEnter() {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return
	}
	fmt.Fprint(os.Stderr, "\nPress Enter to exit.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
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

	st := settings{address: address, passcode: *passcode, insecure: *insecure, allowPrivate: *allowPrivate}
	configPath := ""
	if address == "" {
		set := map[string]string{}
		fs.Visit(func(f *flag.Flag) { set[f.Name] = f.Value.String() })
		paths := configCandidates(os.Getenv, os.UserConfigDir, workDir(), exeDir(), runtime.GOOS)
		cfg, path, err := loadConfig(paths, os.ReadFile)
		if err != nil {
			configHelp(err, path, paths)
			return 2
		}
		if st, err = mergeFlags(cfg, set); err != nil {
			logf("%s", err)
			return 2
		}
		configPath = path
		logf("config: %s", configPath)
	}

	endpoint, err := parseEndpoint(st.address)
	if err != nil {
		if configPath != "" {
			logf("%s: %s", configPath, err)
		} else {
			logf("%s", err)
		}
		return 2
	}
	if st.passcode == "" {
		logf("a passcode is required; the Exit panel shows it")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logf("nexit %s for %s (allowPrivate=%t, verifyTLS=%t)",
		version, endpoint.display, st.allowPrivate, endpoint.tls && !st.insecure)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+st.passcode)
	header.Set("User-Agent", "nexit/1 ("+runtime.GOOS+")")

	host, _ := os.Hostname()
	pol := policy{allowPrivate: st.allowPrivate}

	backoff := time.Second
	for {
		s := &session{pol: pol, hostname: host, os: runtime.GOOS}
		err := s.dial(ctx, endpoint.ws, header, endpoint.tls && st.insecure)
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
  nexit [flags]            (reads nexit.json, see below)

examples:
  nexit wss://nanokvm.local/exit/0 --passcode f8n9yze8
  nexit https://192.168.1.50/exit/0 --passcode f8n9yze8 --insecure

config file:
  Without an address, nexit reads the first nexit.json it finds in:
    %%ProgramData%%\nexit\nexit.json   (/etc/nexit/nexit.json elsewhere)
    %%APPDATA%%\nexit\nexit.json       (the user config dir elsewhere)
    the current directory
    the directory nexit.exe is in
  so nexit.json next to nexit.exe and a double-click is enough. Flags given
  on the command line override the file's values. The file looks like:
%s

flags:
`, version, indent(configShape, "    "))
	fs.PrintDefaults()
}

// configHelp explains a config failure to someone who likely just
// double-clicked nexit.exe: what went wrong, where nexit looked, and what the
// file should contain. path is the file at fault, or "" if none was found.
func configHelp(err error, path string, searched []string) {
	logf("%s", err)
	if path != "" {
		fmt.Fprintf(os.Stderr, "\nnexit searches, and uses the first file found:\n  %s\n",
			strings.Join(searched, "\n  "))
	}
	fmt.Fprintf(os.Stderr, `
Put a nexit.json next to nexit.exe (the NanoKVM's Exit panel offers one to
download). It looks like:
%s

Or give the address on the command line:
  nexit <nanokvm-address> --passcode <passcode> [--insecure]
`, indent(configShape, "  "))
}

func indent(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

// workDir and exeDir return "" when the OS cannot say, which drops that
// location from the config search rather than failing the run.
func workDir() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(p)
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
