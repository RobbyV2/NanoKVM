package exit

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
)

// WstunnelVersion is the release pinned into every Mode B command (D16). The
// hashes are the upstream release's checksums.txt for v10.7.1, copied here so
// the command the operator pastes verifies the download before running it.
const WstunnelVersion = "10.7.1"

const wstunnelReleaseBase = "https://github.com/erebe/wstunnel/releases/download/v" + WstunnelVersion + "/"

// The upstream repository, which the latest-fetch commands resolve at run
// time and which the UI links as the manual fallback.
const (
	wstunnelRepoURL     = "https://github.com/erebe/wstunnel"
	wstunnelReleasesURL = wstunnelRepoURL + "/releases"
	wstunnelLatestURL   = wstunnelReleasesURL + "/latest"
	wstunnelLatestAPI   = "https://api.github.com/repos/erebe/wstunnel/releases/latest"
)

var wstunnelSHA256 = map[string]string{
	"linux_amd64":   "fa842ed53fbb14b1c69cd98829f9895d7f8a6b0d562c57c1175851a52cea9ea2",
	"linux_arm64":   "99f9506d01d1b4073254609600ec5056dab8dc58aec75c32f6eb0508335a8fd2",
	"darwin_amd64":  "ac234c60d461532d81fb60e610787bbc2ce48bcd095f57eb8f86d1a3914e344c",
	"darwin_arm64":  "2c1f427fd651a74c8844a0ed352c13d667da7e08056b790549917d6952ec69ac",
	"windows_amd64": "deb3c8b8d9fecf5428f7e0caabbc13aeb3b1edbfba49e1724a7a065c027bd0f2",
	"windows_arm64": "1d642b29aba0e05c5564adf1dfe2e4494cd10b62331490471bdf6d6548085c96",
}

// WstunnelAssetURL is the pinned download for one platform_arch key.
func WstunnelAssetURL(platform, arch string) string {
	return fmt.Sprintf("%swstunnel_%s_%s_%s.tar.gz", wstunnelReleaseBase, WstunnelVersion, platform, arch)
}

// Origin is what the operator reached the UI through, read from the request
// (D15): the scheme from TLS state or X-Forwarded-Proto, the host verbatim.
// Nothing is rendered from a Host that fails ValidHost.
type Origin struct {
	Scheme string // http or https
	Host   string // host[:port]
}

// ErrBadHost is the reason Commands gives when the request's Host header is
// not something the templates may carry (D15). The value itself is not
// echoed: it is exactly the string that was refused for containing things a
// shell would interpret.
var ErrBadHost = errors.New("request Host header is not a plain host[:port]")

// ValidHost is the gate every templated host passes (D15). Go's net/http
// admits almost any byte in Host, and the host lands inside sh, Perl, Python
// and PowerShell string literals and on curl and wstunnel command lines, so
// the rule is the RFC 3986 authority and nothing looser: a reg-name or IPv4
// literal made of letters, digits, '-', '.' and '_', or an IPv6 literal in
// brackets (no zone), followed by an optional ':' and a decimal port in
// 1..65535. Letters are accepted in either case; hostnames are
// case-insensitive and browsers lower them before sending. No character a
// shell, a quote or a URL parser gives meaning to gets through, so the
// per-language escaping applied on top is defence in depth, never the fence.
func ValidHost(host string) bool {
	if host == "" || len(host) > 255 {
		return false
	}
	name, port := host, ""
	if strings.HasPrefix(host, "[") {
		end := strings.IndexByte(host, ']')
		if end < 0 {
			return false
		}
		literal := host[1:end]
		if !strings.Contains(literal, ":") || net.ParseIP(literal) == nil {
			return false
		}
		rest := host[end+1:]
		if rest == "" {
			return true
		}
		return rest[0] == ':' && validPort(rest[1:])
	}
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		name, port = host[:i], host[i+1:]
		if !validPort(port) {
			return false
		}
	}
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		alpha := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		digit := c >= '0' && c <= '9'
		if !alpha && !digit && c != '-' && c != '.' && c != '_' {
			return false
		}
	}
	return true
}

func validPort(port string) bool {
	if port == "" || len(port) > 5 {
		return false
	}
	for i := 0; i < len(port); i++ {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

// shQuote is a POSIX sh single-quoted word: nothing inside is special, and an
// embedded quote closes the word, escapes itself and reopens it.
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// psQuote is a PowerShell single-quoted string, whose only escape is a
// doubled quote.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// escapeInDoubleQuotes backslash-escapes the characters a language gives
// meaning to inside its double-quoted string literal.
func escapeInDoubleQuotes(special string) func(string) string {
	return func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if strings.ContainsRune(special, r) {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		return b.String()
	}
}

// clientEscape escapes a templated value for the string literal each client
// declares its parameters in (clients/README.md): client.sh, client.pl and
// client.py double-quote them, client.ps1 single-quotes them. Its keys are
// also the set of clients the token-gated surface serves.
var clientEscape = map[string]func(string) string{
	"client.sh":  escapeInDoubleQuotes("\\\"$`"),
	"client.pl":  escapeInDoubleQuotes("\\\"$@"),
	"client.py":  escapeInDoubleQuotes("\\\""),
	"client.ps1": func(s string) string { return strings.ReplaceAll(s, "'", "''") },
}

// OriginOf applies the same scheme rule as middleware.CheckWebSocketOrigin.
func OriginOf(r *http.Request) Origin {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		forwarded = strings.ToLower(forwarded)
		if forwarded == "http" || forwarded == "https" {
			scheme = forwarded
		}
	}
	return Origin{Scheme: scheme, Host: r.Host}
}

func (o Origin) TLS() bool { return o.Scheme == "https" }

// WSScheme is the WebSocket scheme matching the HTTP one.
func (o Origin) WSScheme() string {
	if o.TLS() {
		return "wss"
	}
	return "ws"
}

func (o Origin) base(slot Slot) string {
	return fmt.Sprintf("%s://%s/exit/%s", o.Scheme, o.Host, slot.ID)
}

// Certificate is what the commands need to know about the serving certificate:
// its SHA-256 for the native clients to pin (D14) and whether it is
// self-signed, which decides whether wstunnel may be told to verify it (D16).
type Certificate struct {
	Fingerprint string // lowercase hex, "" when unknown
	SelfSigned  bool
}

// readCertificate reads the configured certificate file. A variable so tests
// and http-only deployments never touch the config.
var readCertificate = func() (Certificate, error) {
	return certificateFrom(config.GetInstance().Cert.Crt)
}

func certificateFrom(path string) (Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Certificate{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return Certificate{}, errors.New("certificate file holds no PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Certificate{}, fmt.Errorf("parse certificate: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	return Certificate{
		Fingerprint: hex.EncodeToString(sum[:]),
		SelfSigned:  cert.Issuer.String() == cert.Subject.String(),
	}, nil
}

// Commands renders the ready-to-paste commands for both modes and the three
// platforms from the request origin (D14, D15, D16). A Host that fails
// ValidHost renders nothing and is reported as ErrBadHost.
func Commands(slot Slot, cfg Config, origin Origin) (proto.GetExitCommandsRsp, error) {
	if !ValidHost(origin.Host) {
		return proto.GetExitCommandsRsp{}, ErrBadHost
	}
	var cert Certificate
	if origin.TLS() {
		if c, err := readCertificate(); err == nil {
			cert = c
		}
	}
	return renderCommands(slot, cfg, origin, cert), nil
}

// renderCommands assumes origin.Host passed ValidHost. Every value it puts on
// a command line is still quoted for the shell that will read it (sh single
// quotes, PowerShell single quotes), so the quoting is the second fence.
func renderCommands(slot Slot, cfg Config, origin Origin, cert Certificate) proto.GetExitCommandsRsp {
	auth := "Authorization: Bearer " + cfg.Token
	base := origin.base(slot)

	insecure := ""
	if origin.TLS() {
		insecure = "k"
	}
	shFetch := fmt.Sprintf("curl -fsSL%s -H %s %s | sh", insecure, shQuote(auth), shQuote(base+"/client.sh"))

	psFetch := fmt.Sprintf("irm -Headers @{Authorization=%s} %s | iex", psQuote("Bearer "+cfg.Token), psQuote(base+"/client.ps1"))
	if origin.TLS() {
		psFetch = "[Net.ServicePointManager]::ServerCertificateValidationCallback={$true}; " + psFetch
	}

	nativeNote := "The script pins the NanoKVM certificate by its SHA-256 fingerprint before trusting a byte."
	if !origin.TLS() {
		nativeNote = "Plain http: the token and every byte between the exit and the NanoKVM travel in cleartext."
	}

	native := []proto.ExitCommand{
		{Platform: "windows", Shell: "powershell", Command: psFetch, Notes: nativeNote},
		{Platform: "macos", Shell: "bash", Command: shFetch, Notes: nativeNote},
		{Platform: "linux", Shell: "bash", Command: shFetch, Notes: nativeNote},
	}

	verify := ""
	wstunnelNote := "wstunnel has no fingerprint pin; with the device's self-signed certificate its transport toward the NanoKVM is unauthenticated."
	if origin.TLS() && cert.Fingerprint != "" && !cert.SelfSigned {
		verify = " --tls-verify-certificate"
		wstunnelNote = "The installed certificate is CA-signed, so wstunnel verifies it."
	}
	if !origin.TLS() {
		wstunnelNote = "Plain http: the token and every byte between the exit and the NanoKVM travel in cleartext."
	}
	// The client arguments are the same for every wstunnel command; only the
	// string quoting follows the shell.
	clientArgs := func(quote func(string) string) string {
		return fmt.Sprintf("client -P %s -H %s -R socks5://%s%s %s",
			slot.ClientPathPrefix(), quote(auth), slot.WstunnelReverseAddr(), verify, quote(origin.WSScheme()+"://"+origin.Host))
	}
	client := clientArgs(shQuote)
	psClient := clientArgs(psQuote)

	wstunnel := []proto.ExitCommand{
		{Platform: "windows", Shell: "powershell", Command: wstunnelWindows(psClient), Notes: wstunnelNote},
		{Platform: "macos", Shell: "bash", Command: wstunnelUnix("darwin", "shasum -a 256 -c -", client), Notes: wstunnelNote},
		{Platform: "linux", Shell: "bash", Command: wstunnelUnix("linux", "sha256sum -c -", client), Notes: wstunnelNote},
	}

	latestNote := "Resolves the newest wstunnel release at run time and verifies the download against that release's own checksums.txt, which proves same-source integrity rather than the pinned hash. " +
		"If the fetch fails, download it by hand from " + wstunnelReleasesURL + "."
	latest := []proto.ExitCommand{
		{Platform: "windows", Shell: "powershell", Command: wstunnelLatestWindows(psClient), Notes: latestNote},
		{Platform: "macos", Shell: "bash", Command: wstunnelLatestUnix("darwin", "shasum -a 256 -c -", client), Notes: latestNote},
		{Platform: "linux", Shell: "bash", Command: wstunnelLatestUnix("linux", "sha256sum -c -", client), Notes: latestNote},
	}

	return proto.GetExitCommandsRsp{
		Scheme:          origin.Scheme,
		Host:            origin.Host,
		Fingerprint:     cert.Fingerprint,
		WstunnelVersion: WstunnelVersion,
		WstunnelRepo:    wstunnelRepoURL,
		Native:          native,
		Wstunnel:        wstunnel,
		WstunnelLatest:  latest,
	}
}

// wstunnelUnix downloads the pinned release for the machine's own architecture,
// verifies it, extracts the binary, sets its execute bit and runs the client in
// the foreground. One line so it pastes. The chmod is load-bearing: the release
// tarballs store wstunnel as 0644, so without it exec fails with Permission
// denied after a clean download and checksum.
func wstunnelUnix(platform, checker, client string) string {
	arm := "aarch64|arm64"
	return strings.Join([]string{
		"set -e",
		`d=$(mktemp -d)`,
		`cd "$d"`,
		fmt.Sprintf(`case "$(uname -m)" in x86_64) a=amd64 s=%s;; %s) a=arm64 s=%s;; *) echo "unsupported architecture: $(uname -m)" >&2; exit 1;; esac`,
			wstunnelSHA256[platform+"_amd64"], arm, wstunnelSHA256[platform+"_arm64"]),
		fmt.Sprintf(`curl -fsSLo wstunnel.tgz "%swstunnel_%s_%s_${a}.tar.gz"`, wstunnelReleaseBase, WstunnelVersion, platform),
		fmt.Sprintf(`echo "$s  wstunnel.tgz" | %s`, checker),
		"tar -xzf wstunnel.tgz wstunnel",
		"chmod +x wstunnel",
		"exec ./wstunnel " + client,
	}, "; ")
}

func wstunnelWindows(client string) string {
	return strings.Join([]string{
		fmt.Sprintf(`$v='%s'`, WstunnelVersion),
		`$a=if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {'arm64'} else {'amd64'}`,
		fmt.Sprintf(`$h=@{amd64='%s';arm64='%s'}[$a]`, wstunnelSHA256["windows_amd64"], wstunnelSHA256["windows_arm64"]),
		`$d=Join-Path $env:TEMP "wstunnel-$v"`,
		`New-Item -Force -ItemType Directory $d | Out-Null`,
		`$f=Join-Path $d 'wstunnel.tar.gz'`,
		fmt.Sprintf(`Invoke-WebRequest -UseBasicParsing "%swstunnel_%s_windows_$a.tar.gz" -OutFile $f`, wstunnelReleaseBase, WstunnelVersion),
		`if ((Get-FileHash $f -Algorithm SHA256).Hash.ToLower() -ne $h) { throw 'wstunnel checksum mismatch' }`,
		`tar -xzf $f -C $d`,
		`& (Join-Path $d 'wstunnel.exe') ` + client,
	}, "; ")
}

// wstunnelLatestUnix resolves the newest release from the releases/latest
// redirect, downloads that tag's asset for the platform (linux or darwin) and
// checks it against the same tag's checksums.txt with the given checker
// (sha256sum on Linux, shasum -a 256 on macOS, which has no sha256sum):
// same-source integrity, weaker than the pin. POSIX sh, so no ERR trap; every
// network step falls through to fail with the releases page. The grep, sed
// and mktemp calls are written to the BSD subset so the one command runs on
// both. The tarball ships the binary without the execute bit, so the extract
// step also sets it, and either failing reports as extract.
func wstunnelLatestUnix(platform, checker, client string) string {
	download := wstunnelReleasesURL + "/download/v$v/"
	return strings.Join([]string{
		"set -e",
		fmt.Sprintf(`fail() { echo "could not fetch the latest wstunnel ($1); download it from %s" >&2; exit 1; }`, wstunnelReleasesURL),
		`d=$(mktemp -d)`,
		`cd "$d"`,
		`case "$(uname -m)" in x86_64) a=amd64;; aarch64|arm64) a=arm64;; *) echo "unsupported architecture: $(uname -m)" >&2; exit 1;; esac`,
		fmt.Sprintf(`v=$(curl -fsSLo /dev/null -w '%%{url_effective}' %s) || fail resolve`, wstunnelLatestURL),
		`v=${v##*/v}`,
		fmt.Sprintf(`f="wstunnel_${v}_%s_${a}.tar.gz"`, platform),
		fmt.Sprintf(`curl -fsSLo wstunnel.tgz "%s$f" || fail download`, download),
		fmt.Sprintf(`curl -fsSL "%schecksums.txt" | grep " $f$" | sed 's/  .*/  wstunnel.tgz/' | %s || fail checksum`, download, checker),
		"tar -xzf wstunnel.tgz wstunnel && chmod +x wstunnel || fail extract",
		"exec ./wstunnel " + client,
	}, "; ")
}

// wstunnelLatestWindows is the same for PowerShell 5.1 and 7: the tag comes
// from the releases API (Invoke-RestMethod parses the JSON on both), the hash
// from that tag's checksums.txt, and any failure rethrows with the releases
// page. -ne compares case-insensitively, so Get-FileHash's upper case matches.
func wstunnelLatestWindows(client string) string {
	download := wstunnelReleasesURL + "/download/v$v/"
	steps := strings.Join([]string{
		fmt.Sprintf(`$v=(Invoke-RestMethod -UseBasicParsing '%s').tag_name.TrimStart('v')`, wstunnelLatestAPI),
		`$a=if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {'arm64'} else {'amd64'}`,
		`$n="wstunnel_${v}_windows_$a.tar.gz"`,
		`$d=Join-Path $env:TEMP 'wstunnel-latest'`,
		`New-Item -Force -ItemType Directory $d | Out-Null`,
		`$f=Join-Path $d 'wstunnel.tar.gz'`,
		fmt.Sprintf(`Invoke-WebRequest -UseBasicParsing "%s$n" -OutFile $f`, download),
		`$s=Join-Path $d 'checksums.txt'`,
		fmt.Sprintf(`Invoke-WebRequest -UseBasicParsing "%schecksums.txt" -OutFile $s`, download),
		`$h=(Select-String -SimpleMatch -Pattern "  $n" -Path $s | Select-Object -First 1).Line`,
		`if (-not $h -or $h.Split(' ')[0] -ne (Get-FileHash $f -Algorithm SHA256).Hash) { throw 'wstunnel checksum mismatch' }`,
		`tar -xzf $f -C $d`,
		`& (Join-Path $d 'wstunnel.exe') ` + client,
	}, "; ")
	return fmt.Sprintf(`try { %s } catch { throw "could not fetch the latest wstunnel: $_ Download it from %s" }`, steps, wstunnelReleasesURL)
}

// Client template placeholders (plan: W3 <-> W1 contract).
const (
	phScheme       = "__SCHEME__"
	phHost         = "__HOST__"
	phSlot         = "__SLOT__"
	phToken        = "__TOKEN__"
	phFingerprint  = "__FINGERPRINT__"
	phAllowPrivate = "__ALLOW_PRIVATE__"
)

// RenderClient fills one embedded client's placeholders. client.sh fetches, so
// it gets the http scheme; the three clients open the socket and get ws/wss.
// It returns false for an unknown client and for a Host that fails ValidHost,
// and the values it templates are escaped for the client's string literal.
func RenderClient(name string, slot Slot, cfg Config, origin Origin, fingerprint string) ([]byte, bool) {
	escape, known := clientEscape[name]
	if !known || !ValidHost(origin.Host) {
		return nil, false
	}
	body, err := clientFiles.ReadFile("clients/" + name)
	if err != nil {
		return nil, false
	}
	scheme := origin.WSScheme()
	if name == "client.sh" {
		scheme = origin.Scheme
	}
	allow := "0"
	if cfg.AllowPrivate {
		allow = "1"
	}
	out := strings.NewReplacer(
		phScheme, scheme,
		phHost, escape(origin.Host),
		phSlot, slot.ID,
		phToken, escape(cfg.Token),
		phFingerprint, escape(fingerprint),
		phAllowPrivate, allow,
	).Replace(string(body))
	return []byte(out), true
}

// clientNames are the scripts the token-gated surface serves.
var clientNames = map[string]bool{
	"client.sh":  true,
	"client.ps1": true,
	"client.pl":  true,
	"client.py":  true,
}
