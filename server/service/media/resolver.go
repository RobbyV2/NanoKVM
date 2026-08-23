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
	for _, entry := range entries {
		identity, err := os.ReadFile(filepath.Join(entry, "function_name"))
		if err != nil || strings.TrimSuffix(string(identity), "\n") != function {
			continue
		}
		matches = append(matches, filepath.Join(r.devRoot, filepath.Base(entry)))
	}
	return uniqueNode(function, matches, ErrNodeNotFound)
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
