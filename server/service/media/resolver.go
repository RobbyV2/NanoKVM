package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrNodeNotFound             = errors.New("media node not found")
	ErrNodeIdentityAmbiguous    = errors.New("media node identity is ambiguous")
	ErrAudioIdentityUnavailable = errors.New("uac2 function identity is unavailable")
)

type SysfsResolver struct {
	sysRoot string
	devRoot string
}

func NewSysfsResolver(sysRoot, devRoot string) *SysfsResolver {
	if sysRoot == "" {
		sysRoot = "/sys"
	}
	if devRoot == "" {
		devRoot = "/dev"
	}
	return &SysfsResolver{sysRoot: sysRoot, devRoot: devRoot}
}

func (r *SysfsResolver) ResolveVideo(function string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(r.sysRoot, "class/video4linux/video*"))
	if err != nil {
		return "", err
	}
	var matches []string
	named := false
	for _, entry := range entries {
		identity, err := os.ReadFile(filepath.Join(entry, "function_name"))
		if err != nil {
			continue
		}
		named = true
		if strings.TrimSuffix(string(identity), "\n") != function {
			continue
		}
		matches = append(matches, filepath.Join(r.devRoot, filepath.Base(entry)))
	}
	if named {
		return uniqueNode(function, matches, ErrNodeNotFound)
	}
	// This kernel carries no function_name backport, so the only identity a
	// gadget video node has left is the controller it hangs off: the node is
	// named for the UDC (4340000.usb on this board) and lives under the same
	// platform device. That separates gadget nodes from capture nodes but not
	// two cameras from each other, so a second gadget node is reported as
	// ambiguous rather than guessed at from minor order.
	nodes, err := r.GadgetVideoNodes()
	if err != nil {
		return "", err
	}
	return uniqueNode(function, nodes, ErrNodeNotFound)
}

// GadgetVideoNodes is every video node the gadget owns. A node exists only
// while its UVC function is linked and bound, so this is also the set of
// functions currently holding the controller deactivated.
func (r *SysfsResolver) GadgetVideoNodes() ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(r.sysRoot, "class/video4linux/video*"))
	if err != nil {
		return nil, err
	}
	udcs, err := filepath.Glob(filepath.Join(r.sysRoot, "class/udc/*"))
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(udcs))
	var roots []string
	for _, udc := range udcs {
		names[filepath.Base(udc)] = true
		// /sys/class/udc/4340000.usb -> ../../devices/platform/4340000.usb/udc/4340000.usb,
		// and uvc parents its video device on the gadget, which is a sibling of
		// that udc directory under the same platform device.
		if target, err := filepath.EvalSymlinks(udc); err == nil {
			roots = append(roots, filepath.Dir(filepath.Dir(target)))
		}
	}
	var nodes []string
	for _, entry := range entries {
		if !gadgetVideoNode(entry, names, roots) {
			continue
		}
		nodes = append(nodes, filepath.Join(r.devRoot, filepath.Base(entry)))
	}
	sort.Strings(nodes)
	return nodes, nil
}

func gadgetVideoNode(entry string, udcNames map[string]bool, udcRoots []string) bool {
	if _, err := os.Stat(filepath.Join(entry, "function_name")); err == nil {
		return true
	}
	name, err := os.ReadFile(filepath.Join(entry, "name"))
	if err == nil && udcNames[strings.TrimSpace(string(name))] {
		return true
	}
	target, err := filepath.EvalSymlinks(entry)
	if err != nil {
		return false
	}
	for _, root := range udcRoots {
		if root != "" && root != string(filepath.Separator) && strings.HasPrefix(target, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (r *SysfsResolver) ResolveAudio(function string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(r.sysRoot, "class/sound/card*"))
	if err != nil {
		return "", err
	}
	var matches []string
	identitySeen := false
	for _, entry := range entries {
		identity, err := os.ReadFile(filepath.Join(entry, "function_name"))
		if err != nil {
			continue
		}
		identitySeen = true
		if strings.TrimSuffix(string(identity), "\n") != function {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(entry), "card"))
		if err == nil {
			matches = append(matches, fmt.Sprintf("hw:%d,0", index))
		}
	}
	if !identitySeen {
		return "", fmt.Errorf("%w: vendor 5.10 exposes no per-function ALSA sysfs attribute for %s", ErrAudioIdentityUnavailable, function)
	}
	return uniqueNode(function, matches, ErrNodeNotFound)
}

func uniqueNode(function string, matches []string, missing error) (string, error) {
	sort.Strings(matches)
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s", missing, function)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %s resolves to %v", ErrNodeIdentityAmbiguous, function, matches)
	}
}
