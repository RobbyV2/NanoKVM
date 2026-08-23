package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	GadgetRoot = "/sys/kernel/config/usb_gadget/g0"

	udcAttr       = "UDC"
	attrBCDDevice = "bcdDevice"

	configPrefix = "configs/c.1"
	dwc2Device   = "4340000.usb"
	emptyUDCName = "\n"

	phyPollInterval = 100 * time.Millisecond
	phyPollTimeout  = 10 * time.Second
)

var (
	probeGadgetDir = filepath.Join(filepath.Dir(GadgetRoot), "g_probe")

	udcDir      = "/sys/class/udc"
	otgRolePath = "/proc/cviusb/otg_role"
	dwc2Bind    = "/sys/bus/platform/drivers/dwc2/bind"
	dwc2Unbind  = "/sys/bus/platform/drivers/dwc2/unbind"
)

var (
	ErrInvalidPath = errors.New("invalid gadget path")
	ErrUDCCount    = errors.New("expected exactly one udc")
	ErrUDCName     = errors.New("invalid udc name")
)

var rootSegments = map[string]bool{
	"functions": true,
	"configs":   true,
	"strings":   true,
	"os_desc":   true,
	udcAttr:     true,
}

var deviceAttrs = map[string]bool{
	"idVendor":        true,
	"idProduct":       true,
	"bcdUSB":          true,
	attrBCDDevice:     true,
	"bDeviceClass":    true,
	"bDeviceSubClass": true,
	"bDeviceProtocol": true,
	"bMaxPacketSize0": true,
}

func validateRel(rel string) error {
	if rel == "" || strings.HasPrefix(rel, "/") {
		return fmt.Errorf("%w: %q", ErrInvalidPath, rel)
	}

	segments := strings.Split(rel, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: %q", ErrInvalidPath, rel)
		}
	}

	if head := segments[0]; !rootSegments[head] && !deviceAttrs[head] {
		return fmt.Errorf("%w: unknown root segment %q in %q", ErrInvalidPath, head, rel)
	}
	return nil
}

func validateRemove(rel string) error {
	if err := validateRel(rel); err != nil {
		return err
	}

	if rel == osDescDir+"/"+configName {
		return nil
	}
	if validUVCLink(rel) {
		return nil
	}

	segments := strings.Split(rel, "/")
	if len(segments) != 3 || segments[0]+"/"+segments[1] != configPrefix {
		return fmt.Errorf("%w: remove is limited to %s and %s symlinks, got %q", ErrInvalidPath, configPrefix, osDescDir, rel)
	}
	return nil
}

func validUVCLink(rel string) bool {
	segments := strings.Split(rel, "/")
	if len(segments) != 6 || segments[0] != functionsDir || !strings.HasPrefix(segments[1], string(FunctionUVC)+".") || !cameraPattern.MatchString(strings.TrimPrefix(segments[1], string(FunctionUVC)+".")) {
		return false
	}
	if segments[2] == "control" && segments[3] == "class" && (segments[4] == "fs" || segments[4] == "ss") && segments[5] == "h" {
		return true
	}
	if segments[2] == "streaming" && segments[3] == "class" && (segments[4] == "fs" || segments[4] == "hs" || segments[4] == "ss") && segments[5] == "h" {
		return true
	}
	return segments[2] == "streaming" && segments[3] == "header" && segments[4] == "h" && segments[5] == "m"
}

func validateRmdir(rel string) error {
	if err := validateRel(rel); err != nil {
		return err
	}
	segments := strings.Split(rel, "/")
	if len(segments) < 5 || len(segments) > 6 || segments[0] != functionsDir || !strings.HasPrefix(segments[1], string(FunctionUVC)+".") || !cameraPattern.MatchString(strings.TrimPrefix(segments[1], string(FunctionUVC)+".")) {
		return fmt.Errorf("%w: rmdir is limited to uvc descriptor groups, got %q", ErrInvalidPath, rel)
	}
	if len(segments) == 5 {
		valid := (segments[2] == "control" && segments[3] == "header" && segments[4] == "h") ||
			(segments[2] == "streaming" && segments[3] == "header" && segments[4] == "h") ||
			(segments[2] == "streaming" && segments[3] == "mjpeg" && segments[4] == "m")
		if valid {
			return nil
		}
	}
	if len(segments) == 6 && segments[2] == "streaming" && segments[3] == "mjpeg" && segments[4] == "m" {
		for _, size := range [...]string{"1280x720", "640x480", "320x240", "160x120"} {
			if segments[5] == size {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: rmdir is limited to uvc descriptor groups, got %q", ErrInvalidPath, rel)
}

type ConfigFSOps struct {
	root string
	dir  *os.File
}

func NewConfigFSOps(root string) (*ConfigFSOps, error) {
	if err := os.Mkdir(root, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("create gadget %s: %w", root, err)
	}

	dir, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf("open gadget %s: %w", root, err)
	}
	return &ConfigFSOps{root: root, dir: dir}, nil
}

func (o *ConfigFSOps) Close() error {
	return o.dir.Close()
}

func (o *ConfigFSOps) fd() int {
	return int(o.dir.Fd())
}

func (o *ConfigFSOps) Mkdir(rel string) error {
	if err := validateRel(rel); err != nil {
		return err
	}
	if err := unix.Mkdirat(o.fd(), rel, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
		return fmt.Errorf("mkdir %s: %w", rel, err)
	}
	return nil
}

func (o *ConfigFSOps) WriteFile(rel string, data []byte) error {
	if err := validateRel(rel); err != nil {
		return err
	}

	fd, err := unix.Openat(o.fd(), rel, unix.O_WRONLY|unix.O_TRUNC|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", rel, err)
	}
	file := os.NewFile(uintptr(fd), rel)

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write %s: %w", rel, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", rel, err)
	}
	return nil
}

func (o *ConfigFSOps) ReadFile(rel string) ([]byte, error) {
	if err := validateRel(rel); err != nil {
		return nil, err
	}

	fd, err := unix.Openat(o.fd(), rel, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", rel, err)
	}
	file := os.NewFile(uintptr(fd), rel)
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	return data, nil
}

func (o *ConfigFSOps) Symlink(target, linkRel string) error {
	if err := validateRel(linkRel); err != nil {
		return err
	}
	if !filepath.IsAbs(target) {
		if err := validateRel(target); err != nil {
			return err
		}
		target = filepath.Join(o.root, target)
	}
	if err := unix.Symlinkat(target, o.fd(), linkRel); err != nil && !errors.Is(err, unix.EEXIST) {
		return fmt.Errorf("symlink %s -> %s: %w", linkRel, target, err)
	}
	return nil
}

func (o *ConfigFSOps) Remove(rel string) error {
	if err := validateRemove(rel); err != nil {
		return err
	}

	var stat unix.Stat_t
	if err := unix.Fstatat(o.fd(), rel, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", rel, err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFLNK {
		return fmt.Errorf("%w: %s is not a symlink", ErrInvalidPath, rel)
	}

	if err := unix.Unlinkat(o.fd(), rel, 0); err != nil && !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("unlink %s: %w", rel, err)
	}
	return nil
}

func (o *ConfigFSOps) RemoveDir(rel string) error {
	if err := validateRmdir(rel); err != nil {
		return err
	}
	if err := unix.Unlinkat(o.fd(), rel, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
		return fmt.Errorf("rmdir %s: %w", rel, err)
	}
	return nil
}

func (o *ConfigFSOps) ListUDC() ([]string, error) {
	entries, err := os.ReadDir(udcDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", udcDir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 1 {
		return nil, fmt.Errorf("%w: %s holds %d entries %v", ErrUDCCount, udcDir, len(names), names)
	}
	return names, nil
}

func (o *ConfigFSOps) BindUDC(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: an empty name is an unbind, not a bind", ErrUDCName)
	}
	return o.WriteFile(udcAttr, []byte(name+"\n"))
}

func (o *ConfigFSOps) UnbindUDC() error {
	return o.WriteFile(udcAttr, []byte(emptyUDCName))
}

func (o *ConfigFSOps) SetOTGRole(role string) error {
	if err := os.WriteFile(otgRolePath, []byte(role+"\n"), 0o644); err != nil {
		return fmt.Errorf("set otg role %s: %w", role, err)
	}
	return nil
}

func (o *ConfigFSOps) ResetPHY(ctx context.Context) error {
	if err := os.WriteFile(dwc2Unbind, []byte(dwc2Device), 0o644); err != nil {
		return fmt.Errorf("unbind %s: %w", dwc2Device, err)
	}
	if err := o.pollUDCCount(ctx, 0); err != nil {
		return err
	}
	if err := os.WriteFile(dwc2Bind, []byte(dwc2Device), 0o644); err != nil {
		return fmt.Errorf("bind %s: %w", dwc2Device, err)
	}
	return o.pollUDCCount(ctx, 1)
}

func (o *ConfigFSOps) pollUDCCount(ctx context.Context, want int) error {
	ticker := time.NewTicker(phyPollInterval)
	defer ticker.Stop()
	deadline := time.After(phyPollTimeout)

	for {
		if entries, err := os.ReadDir(udcDir); err == nil && len(entries) == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %d udc in %s: %w", want, udcDir, ctx.Err())
		case <-deadline:
			return fmt.Errorf("%w: %s did not settle to %d entries", ErrUDCCount, udcDir, want)
		case <-ticker.C:
		}
	}
}

// The scratch gadget is a second root that Ops, which holds a dirfd on g0,
// cannot reach, so its mkdir lives here rather than behind the interface. It is
// never bound and never carries a hid.* function, because f_hid would consume a
// /dev/hidgN minor and shift the numbering hid/hid.go:29-32 depends on.
func probeAvailability() (map[FunctionKind]FunctionProbe, error) {
	if err := os.Mkdir(probeGadgetDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("create probe gadget %s: %w", probeGadgetDir, err)
	}
	defer func() { _ = os.Remove(probeGadgetDir) }()

	available := make(map[FunctionKind]FunctionProbe, len(probeKinds))
	for _, kind := range probeKinds {
		dir := filepath.Join(probeGadgetDir, functionsDir, string(kind)+".probe")
		err := os.Mkdir(dir, 0o755)
		probed := FunctionProbe{Available: err == nil || errors.Is(err, os.ErrExist)}
		if probed.Available {
			for _, name := range probeAttributes[kind] {
				if probed.Attributes == nil {
					probed.Attributes = make(map[string]bool, len(probeAttributes[kind]))
				}
				_, statErr := os.Stat(filepath.Join(dir, name))
				probed.Attributes[name] = statErr == nil
			}
			_ = os.Remove(dir)
		}
		available[kind] = probed
	}
	return available, nil
}
