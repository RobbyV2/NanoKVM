package passthrough

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type Module string

const (
	ModuleUSBIPCore Module = "usbip_core"
	ModuleVHCI      Module = "vhci_hcd"
	ModuleRawGadget Module = "raw_gadget"
)

// usbip_core first: vhci_hcd links against it and there is no depmod on this
// image to work that dependency out. S00kmod is a fixed vendor list, so nothing
// loads these at boot and /dev/raw-gadget does not exist until it is asked for.
var sessionModules = []Module{ModuleUSBIPCore, ModuleVHCI, ModuleRawGadget}

var (
	sysModuleDir = "/sys/module"
	moduleRoots  = []string{"/mnt/system/ko", "/lib/modules"}
)

var ErrModuleMissing = errors.New("passthrough: kernel module not in this image")

type ModuleLoader interface {
	Load(modules ...Module) error
}

type Insmod struct{}

func (Insmod) Load(modules ...Module) error {
	for _, module := range modules {
		if err := module.load(); err != nil {
			return err
		}
	}
	return nil
}

func (m Module) Loaded() (bool, error) {
	_, err := os.Stat(filepath.Join(sysModuleDir, string(m)))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("stat module %s: %w", m, err)
	}
}

func (m Module) load() error {
	loaded, err := m.Loaded()
	if err != nil || loaded {
		return err
	}

	path, err := m.file()
	if err != nil {
		return err
	}
	if err := insmod(path); err != nil {
		return fmt.Errorf("load %s from %s: %w", m, path, err)
	}
	return nil
}

// The module is found by filename because there is no modules.dep to consult,
// and a module built with dashes in its filename reports underscores in
// /sys/module, so both spellings are candidates.
func (m Module) file() (string, error) {
	candidates := map[string]bool{
		string(m) + ".ko": true,
		strings.ReplaceAll(string(m), "_", "-") + ".ko": true,
	}

	for _, root := range moduleRoots {
		var found string
		walk := func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !candidates[entry.Name()] {
				return nil
			}
			found = path
			return fs.SkipAll
		}
		if err := filepath.WalkDir(root, walk); err != nil {
			return "", fmt.Errorf("search %s for %s: %w", root, m, err)
		}
		if found != "" {
			return found, nil
		}
	}
	return "", fmt.Errorf("%w: %s under %s", ErrModuleMissing, m, strings.Join(moduleRoots, " or "))
}

// finit_module rather than an exec of insmod: no argv, no busybox dependency,
// and EEXIST from a racing loader answers the same question the /sys/module
// check answers, so loading stays idempotent under concurrency.
var insmod = func(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	if err := unix.FinitModule(int(file.Fd()), "", 0); err != nil && !errors.Is(err, unix.EEXIST) {
		return err
	}
	return nil
}
