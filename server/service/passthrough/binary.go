package passthrough

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"

	"NanoKVM-Server/utils"
)

// Vars rather than consts so the tests can point both at a temp dir, the way
// tunnel/wrapper.go swaps its own bin and seed paths. Nothing reachable from a
// request writes either, so proxyBinary stays the whole allowlist.
var (
	proxyBinary = "/etc/kvm/bin/usb-proxy"
	proxySeed   = "/kvmapp/passthrough/usb-proxy.gz"
)

var ErrNoProxy = errors.New("passthrough: usb-proxy is neither installed nor seeded")

// The seed ships gzipped beside the tunnel seeds and is extracted on first use.
// Start calls this before it takes the UDC so an image without the seed costs
// nobody their keyboard.
func installProxy() (string, error) {
	info, err := os.Stat(proxyBinary)
	switch {
	case err == nil && !info.IsDir():
		return proxyBinary, nil
	case err == nil:
		return "", fmt.Errorf("%w: %s is a directory", ErrNoProxy, proxyBinary)
	case !errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("stat %s: %w", proxyBinary, err)
	}

	if err := extractProxy(); err != nil {
		return "", err
	}
	return proxyBinary, nil
}

func extractProxy() error {
	source, err := os.Open(proxySeed)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNoProxy, err)
	}
	defer func() { _ = source.Close() }()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("read %s: %w", proxySeed, err)
	}
	defer func() { _ = reader.Close() }()

	file, err := utils.NewAtomicFile(proxyBinary, 0o755)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("extract %s: %w", proxySeed, err)
	}
	return file.Commit()
}
