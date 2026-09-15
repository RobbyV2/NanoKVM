package exit

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
)

// WstunnelVersion is the release pinned into every Mode B command (D16). The
// hashes are the upstream release's checksums.txt for v10.7.1, copied here so
// the command the operator pastes verifies the download before running it.
const WstunnelVersion = "10.7.1"

const wstunnelReleaseBase = "https://github.com/erebe/wstunnel/releases/download/v" + WstunnelVersion + "/"

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
type Origin struct {
	Scheme string // http or https
	Host   string // host[:port]
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
// platforms from the request origin (D14, D15, D16).
func Commands(slot Slot, cfg Config, origin Origin) proto.GetExitCommandsRsp {
	var cert Certificate
	if origin.TLS() {
		if c, err := readCertificate(); err == nil {
			cert = c
		}
	}
	return renderCommands(slot, cfg, origin, cert)
}

func renderCommands(slot Slot, cfg Config, origin Origin, cert Certificate) proto.GetExitCommandsRsp {
	auth := "Authorization: Bearer " + cfg.Token
	base := origin.base(slot)

	insecure := ""
	if origin.TLS() {
		insecure = "k"
	}
	shFetch := fmt.Sprintf("curl -fsSL%s -H '%s' %s/client.sh | sh", insecure, auth, base)

	psFetch := fmt.Sprintf("irm -Headers @{Authorization='Bearer %s'} %s/client.ps1 | iex", cfg.Token, base)
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
	client := fmt.Sprintf("client -P %s -H '%s' -R socks5://%s%s %s://%s",
		slot.ClientPathPrefix(), auth, slot.WstunnelReverseAddr(), verify, origin.WSScheme(), origin.Host)

	wstunnel := []proto.ExitCommand{
		{Platform: "windows", Shell: "powershell", Command: wstunnelWindows(slot, cfg, origin, verify), Notes: wstunnelNote},
		{Platform: "macos", Shell: "bash", Command: wstunnelUnix("darwin", "shasum -a 256 -c -", client), Notes: wstunnelNote},
		{Platform: "linux", Shell: "bash", Command: wstunnelUnix("linux", "sha256sum -c -", client), Notes: wstunnelNote},
	}

	return proto.GetExitCommandsRsp{
		Scheme:          origin.Scheme,
		Host:            origin.Host,
		Fingerprint:     cert.Fingerprint,
		WstunnelVersion: WstunnelVersion,
		Native:          native,
		Wstunnel:        wstunnel,
	}
}

// wstunnelUnix downloads the pinned release for the machine's own architecture,
// verifies it and runs the client in the foreground. One line so it pastes.
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
		"exec ./wstunnel " + client,
	}, "; ")
}

func wstunnelWindows(slot Slot, cfg Config, origin Origin, verify string) string {
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
		fmt.Sprintf(`& (Join-Path $d 'wstunnel.exe') client -P %s -H 'Authorization: Bearer %s' -R socks5://%s%s %s://%s`,
			slot.ClientPathPrefix(), cfg.Token, slot.WstunnelReverseAddr(), verify, origin.WSScheme(), origin.Host),
	}, "; ")
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
func RenderClient(name string, slot Slot, cfg Config, origin Origin, fingerprint string) ([]byte, bool) {
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
		phHost, origin.Host,
		phSlot, slot.ID,
		phToken, cfg.Token,
		phFingerprint, fingerprint,
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
