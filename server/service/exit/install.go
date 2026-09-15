package exit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

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
