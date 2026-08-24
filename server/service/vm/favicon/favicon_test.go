package favicon

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
)

// useStore points the package store at scratch paths. The store is a
// package var, so these tests deliberately do not run in parallel.
func useStore(t *testing.T) (dir string, bootLogo string, shipped string) {
	t.Helper()

	root := t.TempDir()
	dir = filepath.Join(root, "etc-kvm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bootLogo = filepath.Join(root, "boot", "logo.ico")
	shipped = filepath.Join(root, "web", "sipeed.ico")
	if err := os.MkdirAll(filepath.Dir(shipped), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shipped, icoBytes(t, 16), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := icons
	icons = iconStore{
		dir:      dir,
		bootLogo: bootLogo,
		shipped:  func() string { return shipped },
	}
	t.Cleanup(func() { icons = previous })

	return dir, bootLogo, shipped
}

func pngBytes(t *testing.T, size int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 0x40, A: 0xff})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func gifBytes(t *testing.T, size int) []byte {
	t.Helper()

	img := image.NewPaletted(image.Rect(0, 0, size, size), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// icoBytes wraps a PNG in a single-entry ICONDIR, which is how every icon
// authored since Vista is actually laid out.
func icoBytes(t *testing.T, size int) []byte {
	t.Helper()

	payload := pngBytes(t, size)
	const headerLen = 6 + 16

	out := make([]byte, 0, headerLen+len(payload))
	out = append(out, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00) // reserved, type=icon, count=1
	out = append(out, byte(size), byte(size), 0x00, 0x00) // w, h, palette, reserved
	out = append(out, 0x01, 0x00, 0x20, 0x00)             // planes=1, bpp=32
	out = append(out,
		byte(len(payload)), byte(len(payload)>>8), byte(len(payload)>>16), byte(len(payload)>>24))
	out = append(out, headerLen, 0x00, 0x00, 0x00)
	return append(out, payload...)
}

const plainSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">` +
	`<rect width="16" height="16" fill="#123456"/><circle cx="8" cy="8" r="4" fill="#fff"/></svg>`

func multipartUpload(t *testing.T, filename string, content []byte) (string, *bytes.Buffer) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), &body
}

func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	service := NewService()

	router := gin.New()
	router.GET("/api/vm/favicon", service.Get)
	router.GET("/api/vm/favicon/state", service.GetState)
	router.POST("/api/vm/favicon", service.Set)
	router.POST("/api/vm/favicon/upload", service.Upload)
	return router
}

func doUpload(t *testing.T, router *gin.Engine, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	contentType, body := multipartUpload(t, filename, content)
	request := httptest.NewRequest(http.MethodPost, "/api/vm/favicon/upload", body)
	request.Header.Set("Content-Type", contentType)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func doSetURL(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()

	form := url.Values{}
	form.Set("url", target)
	request := httptest.NewRequest(http.MethodPost, "/api/vm/favicon", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) envelope {
	t.Helper()

	var out envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %s", recorder.Body.String(), err)
	}
	return out
}

func decodeState(t *testing.T, recorder *httptest.ResponseRecorder) (source string, version string, bootLogo bool) {
	t.Helper()

	rsp := decodeEnvelope(t, recorder)
	if rsp.Code != 0 {
		t.Fatalf("code = %d, msg = %q", rsp.Code, rsp.Msg)
	}

	var state struct {
		Source   string `json:"source"`
		Version  string `json:"version"`
		BootLogo bool   `json:"bootLogo"`
	}
	if err := json.Unmarshal(rsp.Data, &state); err != nil {
		t.Fatal(err)
	}
	return state.Source, state.Version, state.BootLogo
}

func fetchIcon(t *testing.T, router *gin.Engine, ifNoneMatch string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/vm/favicon", nil)
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestUploadRoundTripsThroughToTheServedIcon(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	content := pngBytes(t, 32)
	source, version, _ := decodeState(t, doUpload(t, router, "icon.png", content))
	if source != SourceCustom {
		t.Fatalf("source = %q, want %q", source, SourceCustom)
	}
	if version == "" {
		t.Fatal("upload returned no version, so the UI has no cache key to bust")
	}

	stored, err := os.ReadFile(filepath.Join(dir, "favicon.png"))
	if err != nil {
		t.Fatalf("stored file: %s", err)
	}
	if !bytes.Equal(stored, content) {
		t.Fatal("stored bytes differ from the uploaded bytes")
	}

	served := fetchIcon(t, router, "")
	if served.Code != http.StatusOK {
		t.Fatalf("GET status = %d", served.Code)
	}
	if got := served.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q, want image/png", got)
	}
	if !bytes.Equal(served.Body.Bytes(), content) {
		t.Fatal("served bytes differ from the uploaded bytes")
	}
}

func TestUploadStoresByDetectedTypeNotByFilename(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	// A GIF wearing an .svg name. Trusting the name would store executable
	// content under favicon.svg and serve it as image/svg+xml.
	decodeState(t, doUpload(t, router, "totally-a.svg", gifBytes(t, 16)))

	if _, err := os.Stat(filepath.Join(dir, "favicon.gif")); err != nil {
		t.Fatalf("expected favicon.gif: %s", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "favicon.svg")); !os.IsNotExist(err) {
		t.Fatal("the client-supplied extension was trusted")
	}

	if got := fetchIcon(t, router, "").Header().Get("Content-Type"); got != "image/gif" {
		t.Fatalf("content-type = %q, want image/gif", got)
	}
}

func TestUploadRejectsAnythingOverTheCap(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	oversize := bytes.Repeat([]byte{0x00}, MaxSize+1)
	copy(oversize, pngBytes(t, 16))

	rsp := decodeEnvelope(t, doUpload(t, router, "huge.png", oversize))
	if rsp.Code != -4 {
		t.Fatalf("code = %d, msg = %q, want -4", rsp.Code, rsp.Msg)
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "favicon.*")); len(entries) != 0 {
		t.Fatalf("oversize upload was stored: %v", entries)
	}
}

func TestUploadRejectsNonImageContent(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	cases := map[string][]byte{
		"plain text":        []byte("this is not an image, it is a sentence"),
		"elf header":        {0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00},
		"truncated png":     pngBytes(t, 16)[:12],
		"ico magic only":    {0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0xff, 0xff},
		"html masquerading": []byte("<html><body><script>alert(1)</script></body></html>"),
		"empty":             {},
		"zip archive":       {'P', 'K', 0x03, 0x04, 0x00, 0x00},
	}

	for name, content := range cases {
		rsp := decodeEnvelope(t, doUpload(t, router, "icon.png", content))
		if rsp.Code == 0 {
			t.Fatalf("%s: accepted as an image", name)
		}
		if entries, _ := filepath.Glob(filepath.Join(dir, "favicon.*")); len(entries) != 0 {
			t.Fatalf("%s: stored anyway: %v", name, entries)
		}
	}
}

func TestEmptyUrlResetsToTheDefaultIcon(t *testing.T) {
	dir, _, shipped := useStore(t)
	router := testRouter()

	decodeState(t, doUpload(t, router, "icon.png", pngBytes(t, 16)))
	if _, err := os.Stat(filepath.Join(dir, "favicon.png")); err != nil {
		t.Fatal(err)
	}

	source, _, _ := decodeState(t, doSetURL(t, router, ""))
	if source != SourceDefault {
		t.Fatalf("source after reset = %q, want %q", source, SourceDefault)
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "favicon.*")); len(entries) != 0 {
		t.Fatalf("reset left files behind: %v", entries)
	}

	// The reset must hand back the shipped icon untouched. On a device where
	// the boot swap never ran, this file is the only copy of it that exists.
	want, err := os.ReadFile(shipped)
	if err != nil {
		t.Fatal(err)
	}
	served := fetchIcon(t, router, "")
	if !bytes.Equal(served.Body.Bytes(), want) {
		t.Fatal("reset did not restore the shipped icon")
	}
}

func TestResetNeverWritesToTheShippedIcon(t *testing.T) {
	_, _, shipped := useStore(t)
	router := testRouter()

	before, err := os.ReadFile(shipped)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(shipped)
	if err != nil {
		t.Fatal(err)
	}

	doUpload(t, router, "icon.png", pngBytes(t, 16))
	doSetURL(t, router, "")
	doUpload(t, router, "icon.svg", []byte(plainSVG))
	doSetURL(t, router, "")

	after, err := os.ReadFile(shipped)
	if err != nil {
		t.Fatalf("the shipped icon is gone: %s", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the shipped icon was modified")
	}
	afterInfo, err := os.Stat(shipped)
	if err != nil {
		t.Fatal(err)
	}
	if beforeInfo.ModTime() != afterInfo.ModTime() {
		t.Fatal("the shipped icon was rewritten")
	}
}

func TestCustomIconOutranksTheBootLogoAndTheStateSaysSo(t *testing.T) {
	_, bootLogo, _ := useStore(t)
	router := testRouter()

	if err := os.MkdirAll(filepath.Dir(bootLogo), 0o755); err != nil {
		t.Fatal(err)
	}
	bootBytes := icoBytes(t, 32)
	if err := os.WriteFile(bootLogo, bootBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/vm/favicon/state", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	source, _, hasBoot := decodeState(t, recorder)
	if source != SourceBoot || !hasBoot {
		t.Fatalf("source = %q, bootLogo = %v, want boot/true", source, hasBoot)
	}
	if !bytes.Equal(fetchIcon(t, router, "").Body.Bytes(), bootBytes) {
		t.Fatal("the boot logo was not served")
	}

	custom := pngBytes(t, 16)
	source, _, hasBoot = decodeState(t, doUpload(t, router, "icon.png", custom))
	if source != SourceCustom {
		t.Fatalf("source = %q, want custom", source)
	}
	if !hasBoot {
		t.Fatal("bootLogo flag dropped, so the UI cannot say what is being overridden")
	}
	if !bytes.Equal(fetchIcon(t, router, "").Body.Bytes(), custom) {
		t.Fatal("the custom icon lost to the boot logo")
	}

	// Clearing it must hand the boot logo back rather than the shipped icon.
	source, _, _ = decodeState(t, doSetURL(t, router, ""))
	if source != SourceBoot {
		t.Fatalf("source after reset = %q, want boot", source)
	}
}

func TestSavingADifferentTypeReplacesThePreviousIcon(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	doUpload(t, router, "a.png", pngBytes(t, 16))
	doUpload(t, router, "b.svg", []byte(plainSVG))

	entries, _ := filepath.Glob(filepath.Join(dir, "favicon.*"))
	if len(entries) != 1 || filepath.Base(entries[0]) != "favicon.svg" {
		t.Fatalf("stored icons = %v, want exactly favicon.svg", entries)
	}
	if got := fetchIcon(t, router, "").Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Fatalf("content-type = %q, want image/svg+xml", got)
	}
}

func TestServedIconRevalidatesAndAnswersNotModified(t *testing.T) {
	useStore(t)
	router := testRouter()

	doUpload(t, router, "icon.png", pngBytes(t, 16))

	first := fetchIcon(t, router, "")
	if got := first.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache-control = %q, want no-cache: a cached icon makes the change invisible", got)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag, so every revalidation refetches the whole icon")
	}
	if got := first.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff = %q", got)
	}
	if got := first.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("no CSP on a same-origin image response")
	}

	repeat := fetchIcon(t, router, etag)
	if repeat.Code != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want 304", repeat.Code)
	}

	// A different icon must invalidate the old validator.
	doUpload(t, router, "icon2.png", pngBytes(t, 24))
	changed := fetchIcon(t, router, etag)
	if changed.Code != http.StatusOK {
		t.Fatalf("status after change = %d, want 200", changed.Code)
	}
	if changed.Header().Get("ETag") == etag {
		t.Fatal("ETag did not change with the icon")
	}
}

func TestSetFaviconDownloadsTheUrlOntoTheDevice(t *testing.T) {
	dir, _, _ := useStore(t)
	router := testRouter()

	content := pngBytes(t, 48)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A server that lies about the type must not change the outcome.
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(content)
	}))
	defer origin.Close()

	source, _, _ := decodeState(t, doSetURL(t, router, origin.URL+"/icon"))
	if source != SourceCustom {
		t.Fatalf("source = %q, want custom", source)
	}

	stored, err := os.ReadFile(filepath.Join(dir, "favicon.png"))
	if err != nil {
		t.Fatalf("nothing was downloaded: %s", err)
	}
	if !bytes.Equal(stored, content) {
		t.Fatal("stored bytes differ from what the origin served")
	}

	// Serving from the device is the point: the browser never touches the
	// remote origin, so a URL only this device can reach still works.
	if !bytes.Equal(fetchIcon(t, router, "").Body.Bytes(), content) {
		t.Fatal("the downloaded icon is not what gets served")
	}
}

func TestSetFaviconSurfacesUrlFailuresInsteadOfFailingSilently(t *testing.T) {
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer notFound.Close()

	notAnImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!doctype html><title>a web page</title>"))
	}))
	defer notAnImage.Close()

	oversize := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte{0x41}, MaxSize+4096))
	}))
	defer oversize.Close()

	// Bound to a port nothing is listening on.
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := closed.URL
	closed.Close()

	cases := []struct {
		name string
		url  string
		code int
	}{
		{"not a url", "://nonsense", -2},
		{"no scheme", "example.com/icon.png", -2},
		{"file scheme", "file:///etc/shadow", -2},
		{"data scheme", "data:image/png;base64,AAAA", -2},
		{"unreachable host", deadURL + "/icon.png", -3},
		{"http error", notFound.URL + "/icon.png", -3},
		{"not an image", notAnImage.URL, -5},
		{"too large", oversize.URL, -4},
	}

	for _, testCase := range cases {
		dir, _, _ := useStore(t)
		router := testRouter()

		rsp := decodeEnvelope(t, doSetURL(t, router, testCase.url))
		if rsp.Code != testCase.code {
			t.Fatalf("%s: code = %d (%q), want %d", testCase.name, rsp.Code, rsp.Msg, testCase.code)
		}
		if rsp.Msg == "" {
			t.Fatalf("%s: no message for the operator", testCase.name)
		}
		if entries, _ := filepath.Glob(filepath.Join(dir, "favicon.*")); len(entries) != 0 {
			t.Fatalf("%s: a failed fetch stored something: %v", testCase.name, entries)
		}
	}
}

func TestSvgAcceptsAPlainIconAndRefusesActiveContent(t *testing.T) {
	if _, err := detectKind([]byte(plainSVG)); err != nil {
		t.Fatalf("a plain svg icon was rejected: %s", err)
	}

	header := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">`
	hostile := map[string]string{
		"script element":  header + `<script>fetch('/api/auth/account')</script></svg>`,
		"event handler":   header + `<rect width="16" height="16" onload="alert(1)"/></svg>`,
		"onclick":         header + `<rect width="16" height="16" onclick="alert(1)"/></svg>`,
		"foreign object":  header + `<foreignObject><body xmlns="http://www.w3.org/1999/xhtml"/></foreignObject></svg>`,
		"external use":    header + `<use href="https://example.com/evil.svg#x"/></svg>`,
		"external image":  header + `<image href="http://example.com/pixel.png"/></svg>`,
		"javascript href": header + `<a href="javascript:alert(1)"><rect width="8" height="8"/></a></svg>`,
		"style import":    header + `<style>@import url(http://example.com/x.css);</style></svg>`,
		"style attr js":   header + `<rect style="background:url(javascript:alert(1))"/></svg>`,
		"animate to js":   header + `<set attributeName="href" to="javascript:alert(1)"/></svg>`,
		"entity bomb":     `<!DOCTYPE svg [<!ENTITY lol "lolol">]>` + header + `<title>&lol;</title></svg>`,
		"not svg root":    `<html xmlns="http://www.w3.org/1999/xhtml"><body/></html>`,
		"malformed xml":   header + `<rect width="16"`,
	}

	for name, content := range hostile {
		if _, err := detectKind([]byte(content)); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}

	// A fragment reference and inline image data are the two legitimate cases
	// and must survive, or every design-tool export gets rejected.
	benign := []string{
		header + `<defs><rect id="r" width="8" height="8"/></defs><use href="#r"/></svg>`,
		header + `<image href="data:image/png;base64,iVBORw0KGgo="/></svg>`,
		`<?xml version="1.0" encoding="UTF-8"?>` + header + `<rect width="16" height="16"/></svg>`,
		"\xef\xbb\xbf" + header + `<rect width="16" height="16"/></svg>`,
	}
	for _, content := range benign {
		if _, err := detectKind([]byte(content)); err != nil {
			t.Fatalf("rejected a legitimate svg (%.40s): %s", content, err)
		}
	}
}

func TestIcoStructureIsValidatedNotJustItsMagic(t *testing.T) {
	good := icoBytes(t, 16)
	kind, err := detectKind(good)
	if err != nil {
		t.Fatalf("a real ico was rejected: %s", err)
	}
	if kind.ext != "ico" {
		t.Fatalf("ext = %q, want ico", kind.ext)
	}

	truncated := good[:len(good)-40]
	if _, err = detectKind(truncated); err == nil {
		t.Fatal("an ico whose entry runs past the end of the file was accepted")
	}

	cursor := append([]byte(nil), good...)
	cursor[2] = 0x02 // type 2 is a cursor, not an icon
	if _, err = detectKind(cursor); err == nil {
		t.Fatal("a cursor was accepted as an icon")
	}

	zeroCount := append([]byte(nil), good...)
	zeroCount[4], zeroCount[5] = 0x00, 0x00
	if _, err = detectKind(zeroCount); err == nil {
		t.Fatal("an ico declaring no images was accepted")
	}

	overlapping := append([]byte(nil), good...)
	overlapping[6+12] = 0x00 // offset now points inside the directory
	if _, err = detectKind(overlapping); err == nil {
		t.Fatal("an ico whose payload overlaps its own header was accepted")
	}
}

func TestFaviconReadIsPublicAndWritesAreAdminOnly(t *testing.T) {
	useStore(t)
	gin.SetMode(gin.TestMode)

	store := authn.NewStore(filepath.Join(t.TempDir(), "pwd"))
	admin, ok, err := store.Authenticate("admin", "admin")
	if err != nil || !ok {
		t.Fatalf("default login: ok=%v err=%v", ok, err)
	}
	if err = store.Create("alice", "valid-password", authn.RoleUser); err != nil {
		t.Fatal(err)
	}
	alice, err := store.Get("alice")
	if err != nil {
		t.Fatal(err)
	}

	previous := authn.DefaultStore
	authn.DefaultStore = store
	t.Cleanup(func() { authn.DefaultStore = previous })

	adminToken, err := middleware.GenerateJWT(admin.Username, admin.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	userToken, err := middleware.GenerateJWT(alice.Username, alice.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}

	// The same split the real router builds.
	service := NewService()
	router := gin.New()
	router.GET("/api/vm/favicon", service.Get)
	api := router.Group("/api").Use(middleware.CheckToken())
	api.GET("/vm/favicon/state", service.GetState)
	adminGroup := router.Group("/api").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)
	adminGroup.POST("/vm/favicon", service.Set)
	adminGroup.POST("/vm/favicon/upload", service.Upload)

	send := func(method, path, token string, body *bytes.Buffer, contentType string) int {
		if body == nil {
			body = &bytes.Buffer{}
		}
		request := httptest.NewRequest(method, path, body)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		if token != "" {
			request.AddCookie(&http.Cookie{Name: middleware.CookieName, Value: token})
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder.Code
	}

	// The browser asks for the icon while painting the login page.
	if status := send(http.MethodGet, "/api/vm/favicon", "", nil, ""); status != http.StatusOK {
		t.Fatalf("anonymous icon read = %d, want 200", status)
	}

	if status := send(http.MethodGet, "/api/vm/favicon/state", "", nil, ""); status != http.StatusUnauthorized {
		t.Fatalf("anonymous state read = %d, want 401", status)
	}
	if status := send(http.MethodGet, "/api/vm/favicon/state", userToken, nil, ""); status != http.StatusOK {
		t.Fatalf("user state read = %d, want 200", status)
	}

	form := "application/x-www-form-urlencoded"
	if status := send(http.MethodPost, "/api/vm/favicon", "", bytes.NewBufferString("url="), form); status != http.StatusUnauthorized {
		t.Fatalf("anonymous write = %d, want 401", status)
	}
	if status := send(http.MethodPost, "/api/vm/favicon", userToken, bytes.NewBufferString("url="), form); status != http.StatusForbidden {
		t.Fatalf("non-admin write = %d, want 403", status)
	}
	if status := send(http.MethodPost, "/api/vm/favicon", adminToken, bytes.NewBufferString("url="), form); status != http.StatusOK {
		t.Fatalf("admin write = %d, want 200", status)
	}

	contentType, body := multipartUpload(t, "icon.png", pngBytes(t, 16))
	if status := send(http.MethodPost, "/api/vm/favicon/upload", "", body, contentType); status != http.StatusUnauthorized {
		t.Fatalf("anonymous upload = %d, want 401", status)
	}

	contentType, body = multipartUpload(t, "icon.png", pngBytes(t, 16))
	if status := send(http.MethodPost, "/api/vm/favicon/upload", userToken, body, contentType); status != http.StatusForbidden {
		t.Fatalf("non-admin upload = %d, want 403", status)
	}

	contentType, body = multipartUpload(t, "icon.png", pngBytes(t, 16))
	if status := send(http.MethodPost, "/api/vm/favicon/upload", adminToken, body, contentType); status != http.StatusOK {
		t.Fatalf("admin upload = %d, want 200", status)
	}
}

func TestOversizeImageDimensionsAreRefused(t *testing.T) {
	// A 4096x4096 PNG is only a few KiB when it is a solid colour, so the byte
	// cap alone would let it through and then hand the decoder 64MB of work on
	// a board with 256MB of RAM.
	huge := pngBytes(t, maxDimension+1)
	if len(huge) > MaxSize {
		t.Skipf("test image is %d bytes, the size cap would catch it first", len(huge))
	}
	if _, err := detectKind(huge); err == nil {
		t.Fatalf("a %dpx image was accepted", maxDimension+1)
	}
}

func TestVersionTracksTheBytes(t *testing.T) {
	first := versionOf(pngBytes(t, 16))
	again := versionOf(pngBytes(t, 16))
	other := versionOf(pngBytes(t, 24))

	if first != again {
		t.Fatal("identical bytes produced different versions")
	}
	if first == other {
		t.Fatal("different bytes produced the same version")
	}
	if len(first) != 16 {
		t.Fatalf("version %q is %d chars, want a 16-char digest prefix", first, len(first))
	}
}

func TestResolveFallsAllTheWayThroughToNothing(t *testing.T) {
	root := t.TempDir()
	previous := icons
	icons = iconStore{
		dir:      filepath.Join(root, "etc"),
		bootLogo: filepath.Join(root, "boot", "logo.ico"),
		shipped:  func() string { return filepath.Join(root, "web", "sipeed.ico") },
	}
	t.Cleanup(func() { icons = previous })

	if _, _, _, err := icons.resolve(); err == nil {
		t.Fatal("resolve invented an icon out of an empty filesystem")
	}

	router := testRouter()
	if status := fetchIcon(t, router, "").Code; status != http.StatusNotFound {
		t.Fatalf("status with no icon anywhere = %d, want 404", status)
	}

	// The panel must still render rather than erroring out.
	request := httptest.NewRequest(http.MethodGet, "/api/vm/favicon/state", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	source, _, _ := decodeState(t, recorder)
	if source != SourceDefault {
		t.Fatalf("source = %q, want %q", source, SourceDefault)
	}
}
