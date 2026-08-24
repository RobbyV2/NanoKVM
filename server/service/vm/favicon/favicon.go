// Package favicon stores and serves the browser-tab icon for the web UI.
//
// It is a subpackage rather than part of service/vm because service/vm links
// libkvm.so through cgo and so cannot build a test binary for the host
// architecture; nothing here needs the device at all.
package favicon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Registered for their DecodeConfig side effect: an upload is only accepted
	// once the decoder agrees with the magic bytes about what it is.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	// A favicon that renders at 16-64px is a few KiB; even a multi-resolution
	// .ico carrying a 256px frame stays under 150KiB. 256KiB leaves room for
	// that without letting an upload matter on a 256MB board: the file is held
	// in memory while it is validated, so the transient cost is bounded here.
	MaxSize = 256 << 10

	// An unreachable host has to fail while the operator is still looking at
	// the settings panel, not minutes later.
	fetchTimeout   = 15 * time.Second
	fetchRedirects = 5

	// Rendered pixels, not file size. A solid-colour 4096px PNG compresses to
	// a few KiB and would sail past the byte cap, then cost width*height*4
	// bytes to decode on a board that cannot spare them.
	maxDimension = 1024

	// Never sniffed, never scripted. Applied to every favicon response rather
	// than only the SVG one, so the header set does not depend on what is
	// stored.
	securityPolicy = "default-src 'none'; style-src 'unsafe-inline'; sandbox"

	// Sources, in the order resolve consults them. Reported to the UI so the
	// panel never claims one icon is live while another is being rendered.
	SourceCustom  = "custom"
	SourceBoot    = "boot"
	SourceDefault = "default"
)

var (
	ErrInvalidURL      = errors.New("favicon: url must be http or https")
	ErrFetch           = errors.New("favicon: download failed")
	ErrTooLarge        = errors.New("favicon: image too large")
	ErrUnsupportedType = errors.New("favicon: unsupported image")
	ErrMissing         = errors.New("favicon: no icon available")
)

// A stored icon is named by its type: favicon.png, favicon.ico, favicon.svg.
// The extension *is* the content-type record, which is why there is no sidecar
// metadata file that could fall out of sync with the bytes.
type iconKind struct {
	ext         string
	contentType string
}

var kinds = []iconKind{
	{ext: "ico", contentType: "image/x-icon"},
	{ext: "png", contentType: "image/png"},
	{ext: "svg", contentType: "image/svg+xml"},
	{ext: "gif", contentType: "image/gif"},
	{ext: "jpg", contentType: "image/jpeg"},
}

func kindByExt(ext string) iconKind {
	for _, candidate := range kinds {
		if candidate.ext == ext {
			return candidate
		}
	}
	return iconKind{}
}

// iconStore owns the three places an icon can come from. It is a value rather
// than a singleton behind an interface because the only thing tests need to
// vary is where the paths point.
type iconStore struct {
	// Writable at runtime and preserved across reboots, alongside web-title.
	dir string
	// The stock override channel: an .ico dropped on the boot partition.
	bootLogo string
	// The icon shipped inside the served web root. Read, never written: on a
	// device where the boot swap has never run this file is the pristine
	// factory icon and no copy of it exists anywhere else to restore from.
	shipped func() string
}

var icons = iconStore{
	dir:      "/etc/kvm",
	bootLogo: "/boot/logo.ico",
	shipped:  shippedPath,
}

func shippedPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(execPath), "web", "sipeed.ico")
}

func (s iconStore) path(k iconKind) string {
	return filepath.Join(s.dir, "favicon."+k.ext)
}

// custom returns the stored override, if one exists.
func (s iconStore) custom() ([]byte, iconKind, bool) {
	for _, k := range kinds {
		data, err := os.ReadFile(s.path(k))
		if err == nil && len(data) > 0 {
			return data, k, true
		}
	}
	return nil, iconKind{}, false
}

// save validates data and replaces whatever override was stored. Every existing
// favicon.* is removed before the new one lands, so a crash mid-write leaves no
// icon at all — the default renders — rather than an icon served as the wrong
// type.
func (s iconStore) save(data []byte) (iconKind, error) {
	k, err := detectKind(data)
	if err != nil {
		return iconKind{}, err
	}

	if err = os.MkdirAll(s.dir, 0o755); err != nil {
		return iconKind{}, err
	}

	tmp, err := os.CreateTemp(s.dir, ".favicon-*")
	if err != nil {
		return iconKind{}, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return iconKind{}, err
	}
	if err = tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return iconKind{}, err
	}
	if err = tmp.Close(); err != nil {
		return iconKind{}, err
	}

	if err = s.clear(); err != nil {
		return iconKind{}, err
	}
	if err = os.Rename(tmpName, s.path(k)); err != nil {
		return iconKind{}, err
	}

	return k, nil
}

// clear drops the override and restores whatever the device rendered before it.
// It never touches the boot logo or the shipped icon.
func (s iconStore) clear() error {
	for _, k := range kinds {
		if err := os.Remove(s.path(k)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s iconStore) hasBootLogo() bool {
	info, err := os.Stat(s.bootLogo)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

// resolve returns the bytes the browser should render and which of the three
// sources produced them. An explicit setting made in the UI outranks a file
// dropped on the boot partition: the operator who just clicked is more current
// than whoever imaged the card. The UI is told which source won, so an
// overridden boot logo is visible rather than silently ignored.
func (s iconStore) resolve() (data []byte, contentType string, source string, err error) {
	if stored, k, ok := s.custom(); ok {
		return stored, k.contentType, SourceCustom, nil
	}

	if s.hasBootLogo() {
		if stored, readErr := os.ReadFile(s.bootLogo); readErr == nil && len(stored) > 0 {
			return stored, contentTypeOf(stored, "image/x-icon"), SourceBoot, nil
		}
	}

	if path := s.shipped(); path != "" {
		if stored, readErr := os.ReadFile(path); readErr == nil && len(stored) > 0 {
			return stored, contentTypeOf(stored, "image/x-icon"), SourceDefault, nil
		}
	}

	return nil, "", "", ErrMissing
}

// contentTypeOf labels bytes that were never validated on the way in — the boot
// logo and the shipped icon. Unrecognised content keeps the caller's guess
// rather than failing, because those two files are the last-resort fallbacks
// and refusing to serve them would leave the tab with no icon at all.
func contentTypeOf(data []byte, fallback string) string {
	if k, err := detectKind(data); err == nil {
		return k.contentType
	}
	return fallback
}

// detectKind identifies data from its own bytes. The filename and the
// client-supplied MIME type are never consulted: both are chosen by whoever is
// uploading.
func detectKind(data []byte) (iconKind, error) {
	if len(data) == 0 {
		return iconKind{}, ErrUnsupportedType
	}

	if isICO(data) {
		if err := validateICO(data); err != nil {
			return iconKind{}, err
		}
		return kindByExt("ico"), nil
	}

	if looksLikeXML(data) {
		if err := validateSVG(data); err != nil {
			return iconKind{}, err
		}
		return kindByExt("svg"), nil
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return iconKind{}, ErrUnsupportedType
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maxDimension || config.Height > maxDimension {
		return iconKind{}, ErrUnsupportedType
	}

	switch format {
	case "png":
		return kindByExt("png"), nil
	case "gif":
		return kindByExt("gif"), nil
	case "jpeg":
		return kindByExt("jpg"), nil
	default:
		return iconKind{}, ErrUnsupportedType
	}
}

func isICO(data []byte) bool {
	// ICONDIR: reserved=0, type=1. Type 2 is a cursor, which is not an icon.
	return len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00
}

// validateICO walks the directory rather than trusting the magic number, so
// four familiar bytes glued to arbitrary content are rejected.
func validateICO(data []byte) error {
	const dirEntry = 16

	if len(data) < 6 {
		return ErrUnsupportedType
	}

	count := int(le16(data[4:6]))
	if count < 1 || count > 64 {
		return ErrUnsupportedType
	}

	headerLen := 6 + count*dirEntry
	if len(data) < headerLen {
		return ErrUnsupportedType
	}

	for i := 0; i < count; i++ {
		entry := data[6+i*dirEntry : 6+(i+1)*dirEntry]
		size := int(le32(entry[8:12]))
		offset := int(le32(entry[12:16]))
		if size <= 0 || offset < headerLen {
			return ErrUnsupportedType
		}
		if offset > len(data) || size > len(data)-offset {
			return ErrUnsupportedType
		}
	}

	return nil
}

func le16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func looksLikeXML(data []byte) bool {
	// The BOM is stripped by codepoint: a UTF-8 signature ahead of "<?xml" is
	// legal and common in files exported by Windows tooling.
	trimmed := bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed = bytes.TrimLeft(trimmed, " \t\r\n")
	return len(trimmed) > 0 && trimmed[0] == '<'
}

// Elements that can execute, navigate, or animate an attribute into doing
// either. A favicon needs none of them, so the whole class is refused rather
// than sanitised: a stored SVG is served from the device's own origin and is
// reachable by URL, which makes anything that survives this check same-origin
// executable content.
var forbiddenElements = map[string]struct{}{
	"script":           {},
	"foreignobject":    {},
	"iframe":           {},
	"embed":            {},
	"object":           {},
	"handler":          {},
	"animate":          {},
	"animatemotion":    {},
	"animatetransform": {},
	"set":              {},
	"audio":            {},
	"video":            {},
}

// Attributes whose value is a URL, or an animation endpoint that can become
// one.
var urlAttributes = map[string]struct{}{
	"href":       {},
	"src":        {},
	"from":       {},
	"to":         {},
	"values":     {},
	"by":         {},
	"data":       {},
	"action":     {},
	"formaction": {},
}

// validateSVG parses the document and refuses anything that could run. Go's XML
// decoder does not resolve external entities, and a DOCTYPE carrying an ENTITY
// declaration is rejected outright, which closes both XXE and the billion-laughs
// expansion that a 256MB board cannot absorb.
func validateSVG(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true

	seenRoot := false
	inStyle := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ErrUnsupportedType
		}

		switch node := token.(type) {
		case xml.Directive:
			if strings.Contains(strings.ToUpper(string(node)), "ENTITY") {
				return ErrUnsupportedType
			}

		case xml.ProcInst:
			// <?xml ...?> is a declaration; anything else is a processing hook.
			if node.Target != "xml" {
				return ErrUnsupportedType
			}

		case xml.StartElement:
			name := strings.ToLower(node.Name.Local)
			if !seenRoot {
				if name != "svg" {
					return ErrUnsupportedType
				}
				seenRoot = true
			}
			if _, bad := forbiddenElements[name]; bad {
				return ErrUnsupportedType
			}
			if name == "style" {
				inStyle = true
			}
			if err = validateAttributes(node.Attr); err != nil {
				return err
			}

		case xml.EndElement:
			if strings.ToLower(node.Name.Local) == "style" {
				inStyle = false
			}

		case xml.CharData:
			if inStyle && hasActiveContent(string(node)) {
				return ErrUnsupportedType
			}
		}
	}

	if !seenRoot {
		return ErrUnsupportedType
	}
	return nil
}

func validateAttributes(attrs []xml.Attr) error {
	for _, attr := range attrs {
		name := strings.ToLower(attr.Name.Local)

		// onload, onclick, and every other event handler, in any namespace.
		if strings.HasPrefix(name, "on") {
			return ErrUnsupportedType
		}

		if name == "style" && hasActiveContent(attr.Value) {
			return ErrUnsupportedType
		}

		if _, isURL := urlAttributes[name]; !isURL {
			continue
		}
		if err := checkReference(attr.Value); err != nil {
			return err
		}
	}
	return nil
}

// checkReference allows only same-document fragments and inline image data. An
// absolute reference would make the icon phone home from every browser that
// renders it, on a device whose whole point is living on a private network.
func checkReference(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{
		"data:image/png;base64,",
		"data:image/jpeg;base64,",
		"data:image/gif;base64,",
	} {
		if strings.HasPrefix(lower, prefix) {
			return nil
		}
	}

	// Animation endpoints are frequently plain numbers or colours, which are
	// not references at all.
	if !strings.ContainsAny(trimmed, ":/") {
		return nil
	}

	return ErrUnsupportedType
}

func hasActiveContent(value string) bool {
	// Whitespace is stripped first so "java\nscript:" and "url( http://" do not
	// slip past a substring check.
	lower := strings.Join(strings.Fields(strings.ToLower(value)), "")
	for _, needle := range []string{
		"javascript:",
		"expression(",
		"@import",
		"url(http",
		"url(//",
		"url('http",
		"url(\"http",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

// download pulls an icon onto the device. Resolving the URL here rather than
// storing the string and letting each browser fetch it is what makes the URL
// mode work at all: the UI is served over plain HTTP, so an https:// icon URL
// handed to the browser is blocked as mixed content on some clients and not
// others, and a URL only the operator's laptop can reach renders nothing for
// anybody else. Downloading once means every browser gets identical bytes from
// the same origin, and an unreachable URL fails loudly, here, with a reason.
func download(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return nil, ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidURL
	}

	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= fetchRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("unsupported redirect scheme")
			}
			return nil
		},
	}

	request, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, ErrInvalidURL
	}
	request.Header.Set("Accept", "image/*")
	request.Header.Set("User-Agent", "NanoKVM")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFetch, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: server returned %s", ErrFetch, response.Status)
	}
	if response.ContentLength > MaxSize {
		return nil, ErrTooLarge
	}

	// One byte past the cap distinguishes "exactly at the limit" from "the
	// remote is streaming us a filesystem".
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFetch, err)
	}
	if len(data) > MaxSize {
		return nil, ErrTooLarge
	}
	if len(data) == 0 {
		return nil, ErrUnsupportedType
	}

	return data, nil
}

// versionOf is the cache key the UI appends to the icon URL. It is derived from
// the bytes, so setting an icon back to a previous one reuses the earlier
// version and the browser's cached copy is correct rather than merely stale.
func versionOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// Get serves the icon itself. Deliberately unauthenticated: the browser asks
// for the favicon while painting the login page, before any token exists, and
// an icon is not a secret.
func (s *Service) Get(c *gin.Context) {
	data, contentType, _, err := icons.resolve()
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	etag := `"` + versionOf(data) + `"`

	// no-cache revalidates on every load instead of caching for a session, so
	// a change made on one machine reaches every other browser on its next
	// page load. The ETag keeps that revalidation a 304 rather than a refetch.
	c.Header("Cache-Control", "no-cache")
	c.Header("ETag", etag)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", securityPolicy)

	if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, etag) {
		c.Status(http.StatusNotModified)
		return
	}

	c.Data(http.StatusOK, contentType, data)
}

// GetState describes what is live, for the settings panel.
func (s *Service) GetState(c *gin.Context) {
	writeState(c)
}

// Set points the device at a URL, or resets to the default when the URL is
// empty. Empty means reset for the same reason it does for the web title: the
// field beside it commits on blur, and clearing it is how you undo it.
func (s *Service) Set(c *gin.Context) {
	var req proto.SetFaviconReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if strings.TrimSpace(req.Url) == "" {
		if err := icons.clear(); err != nil {
			rsp.ErrRsp(c, -7, "reset failed")
			return
		}
		writeState(c)
		log.Debugf("favicon reset to default")
		return
	}

	data, err := download(req.Url)
	if err != nil {
		rsp.ErrRsp(c, errorCode(err), err.Error())
		return
	}

	if _, err = icons.save(data); err != nil {
		rsp.ErrRsp(c, errorCode(err), err.Error())
		return
	}

	writeState(c)
	log.Debugf("set favicon from url, %d bytes", len(data))
}

// Upload stores an icon chosen from the operator's own machine.
func (s *Service) Upload(c *gin.Context) {
	var rsp proto.Response

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		rsp.ErrRsp(c, -1, "bad request")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Checked before reading as well as after: the header is a hint, the
	// LimitReader is the enforcement.
	if header.Size > MaxSize {
		rsp.ErrRsp(c, -4, ErrTooLarge.Error())
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxSize+1))
	if err != nil {
		rsp.ErrRsp(c, -1, "read failed")
		return
	}
	if len(data) > MaxSize {
		rsp.ErrRsp(c, -4, ErrTooLarge.Error())
		return
	}

	if _, err = icons.save(data); err != nil {
		rsp.ErrRsp(c, errorCode(err), err.Error())
		return
	}

	writeState(c)
	log.Debugf("uploaded favicon %s, %d bytes", header.Filename, len(data))
}

func writeState(c *gin.Context) {
	var rsp proto.Response

	data, contentType, source, err := icons.resolve()
	if err != nil {
		// No icon anywhere is not an error the panel can act on; it renders
		// "default" and the tab keeps the browser's own placeholder.
		rsp.OkRspWithData(c, &proto.GetFaviconRsp{
			Source:   SourceDefault,
			BootLogo: icons.hasBootLogo(),
		})
		return
	}

	rsp.OkRspWithData(c, &proto.GetFaviconRsp{
		Source:      source,
		ContentType: contentType,
		Size:        len(data),
		Version:     versionOf(data),
		BootLogo:    icons.hasBootLogo(),
	})
}

// Stable codes, because the browser turns them into translated messages. A
// failure the operator can act on must not arrive as an untranslated Go string.
func errorCode(err error) int {
	switch {
	case errors.Is(err, ErrInvalidURL):
		return -2
	case errors.Is(err, ErrFetch):
		return -3
	case errors.Is(err, ErrTooLarge):
		return -4
	case errors.Is(err, ErrUnsupportedType):
		return -5
	default:
		return -6
	}
}
