package exit

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/service/extensions/tunnel"
	"NanoKVM-Server/utils"
)

// installScripts refreshes S94exit and S30rndis in /etc/init.d from the
// shipped copies (D18). Both are copied rather than only restored when
// missing: an update ships new scripts in /kvmapp that nothing else installs
// until system_init runs on the next app update. An identical file is left
// alone so the flash sees no write.
func installScripts() error {
	var firstErr error
	for _, name := range []string{S94Script, S30Script} {
		if err := installScript(name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func installScript(name string) error {
	seed, err := os.ReadFile(filepath.Join(InitSeedDir, name))
	if err != nil {
		return fmt.Errorf("read shipped %s: %w", name, err)
	}
	target := filepath.Join(InitDir, name)
	if current, err := os.ReadFile(target); err == nil && bytes.Equal(current, seed) {
		if info, statErr := os.Stat(target); statErr == nil && info.Mode()&0o111 != 0 {
			return nil
		}
	}
	return utils.WriteFileAtomic(target, seed, 0o755)
}

// ensureBinary extracts a daemon from its seed when the binary is missing or
// the seed changed, unless an operator marked it custom. The wstunnel seed is
// the tunnel package's own; hev's lives in this package's SeedDir. A variable
// so tests need no seeds.
var ensureBinary = func(name string) (string, error) {
	switch name {
	case HevBinary:
		return tunnel.EnsureBinaryFrom(name, filepath.Join(SeedDir, name+".gz"))
	case WstunnelBinary:
		return tunnel.EnsureBinary(name)
	default:
		return "", fmt.Errorf("unknown exit binary %q", name)
	}
}

// nexitAssets maps the token-gated path to the gzipped seed that answers it.
// The names carry the architecture because the command that fetches this runs
// before anything here can ask the machine what it is: cmd picks the branch
// from PROCESSOR_ARCHITECTURE and asks for the matching name.
var nexitAssets = map[string]string{
	"nexit-windows-amd64.exe": "nexit-windows-amd64.exe.gz",
	"nexit-windows-arm64.exe": "nexit-windows-arm64.exe.gz",
}

// serveNexit streams one seed out decompressed. It is never extracted to disk:
// the unit's /tmp is a small tmpfs and this binary is for the operator's
// machine, so the only copy that has to exist is the one being sent.
func serveNexit(w http.ResponseWriter, r *http.Request, seed string) {
	f, err := os.Open(filepath.Join(SeedDir, seed))
	if err != nil {
		log.Errorf("exit: open %s: %s", seed, err)
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		log.Errorf("exit: %s is not gzip: %s", seed, err)
		http.NotFound(w, r)
		return
	}
	defer zr.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="nexit.exe"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, zr); err != nil {
		log.Warnf("exit: serving %s stopped: %s", seed, err)
	}
}

// nexitConfigName is the token-gated path of the slot's nexit.json, the file
// nexit.exe reads when it is started with no arguments.
const nexitConfigName = "nexit.json"

// serveNexitConfig sends a rendered nexit.json as a download. It carries the
// slot's token, so like nexit.exe it is never cached.
func serveNexitConfig(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nexitConfigName+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
