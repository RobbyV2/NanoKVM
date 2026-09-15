package tunnel

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

var (
	binDir  = "/etc/kvm/bin"
	seedDir = "/kvmapp/tunnels"
	logDir  = "/tmp"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type envInject struct {
	Key   string
	Value string
}

type tunnelSpec struct {
	Env        []envInject
	Args       []string
	SeededEnv  []string
	HealthFile string
	MemLimit   int64
}

var specs = map[proto.TunnelName]tunnelSpec{
	proto.TunnelWstunnel: {},
	proto.TunnelNewt: {
		Env: []envInject{
			{Key: "NEWT_SYSTEM_SUBSTRATE", Value: "CONTAINER"},
			{Key: "GOGC", Value: "50"},
		},
		Args: []string{
			"--health-file", "/tmp/newt.health",
			"--config-file", "/etc/kvm/newt-client.json",
		},
		SeededEnv:  []string{"PANGOLIN_ENDPOINT", "NEWT_ID", "NEWT_SECRET", "NEWT_PROVISIONING_KEY"},
		HealthFile: "/tmp/newt.health",
		MemLimit:   75,
	},
}

func specOf(name proto.TunnelName) (tunnelSpec, bool) {
	spec, ok := specs[name]
	return spec, ok
}

func wrapperPath(name proto.TunnelName) string {
	return filepath.Join(configDir, string(name)+".cmd")
}

func binaryFile(name proto.TunnelName) string {
	return filepath.Join(binDir, string(name))
}

func seedPath(name proto.TunnelName) string {
	return filepath.Join(seedDir, string(name)+".gz")
}

func customMarkerPath(name proto.TunnelName) string {
	return filepath.Join(binDir, "."+string(name)+".custom")
}

func tokenize(s string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	started := false

	const (
		stateBare = iota
		stateSingle
		stateDouble
	)
	state := stateBare

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		char := runes[i]

		switch state {
		case stateBare:
			switch char {
			case ' ', '\t', '\n', '\r':
				if started {
					tokens = append(tokens, current.String())
					current.Reset()
					started = false
				}
			case '\'':
				state = stateSingle
				started = true
			case '"':
				state = stateDouble
				started = true
			case '\\':
				if i+1 >= len(runes) {
					return nil, errors.New("trailing backslash in arguments")
				}
				i++
				current.WriteRune(runes[i])
				started = true
			default:
				current.WriteRune(char)
				started = true
			}

		case stateSingle:
			if char == '\'' {
				state = stateBare
				continue
			}
			current.WriteRune(char)

		case stateDouble:
			switch char {
			case '"':
				state = stateBare
			case '\\':
				if i+1 >= len(runes) {
					return nil, errors.New("trailing backslash in arguments")
				}
				next := runes[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					current.WriteRune(next)
					i++
					continue
				}
				current.WriteRune(char)
			default:
				current.WriteRune(char)
			}
		}
	}

	if state != stateBare {
		return nil, errors.New("unbalanced quote in arguments")
	}
	if started {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}

func isValidEnvKey(key string) bool {
	return envKeyPattern.MatchString(key)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func extractSeed(name proto.TunnelName) error {
	return extractSeedFrom(seedPath(name), binaryFile(name))
}

func extractSeedFrom(seed, target string) error {
	source, err := os.Open(seed)
	if err != nil {
		return fmt.Errorf("open tunnel seed: %w", err)
	}
	defer func() { _ = source.Close() }()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("read tunnel seed: %w", err)
	}
	defer func() { _ = reader.Close() }()

	file, err := utils.NewAtomicFile(target, 0o755)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("extract tunnel seed: %w", err)
	}
	return file.Commit()
}

func binaryPath(name proto.TunnelName) (string, error) {
	return EnsureBinary(string(name))
}

// EnsureBinary returns /etc/kvm/bin/<name>, extracting it from the seed in
// /kvmapp/tunnels when it is missing or when the seed has changed since the
// last extraction. The exit package's hev daemon has its seed elsewhere and
// goes through EnsureBinaryFrom.
func EnsureBinary(name string) (string, error) {
	return EnsureBinaryFrom(name, seedPath(proto.TunnelName(name)))
}

// EnsureBinaryFrom is EnsureBinary with an explicit seed. The staleness record
// is /etc/kvm/bin/.<name>.seed, the sha256 of the seed the binary was extracted
// from: an update that ships a new seed re-extracts once, and a device that
// already has the matching binary never touches the flash. An operator's
// uploaded binary, marked by .<name>.custom, is never replaced (D18).
func EnsureBinaryFrom(name, seed string) (string, error) {
	if name == "" || name != filepath.Base(name) {
		return "", fmt.Errorf("invalid binary name %q", name)
	}
	target := filepath.Join(binDir, name)

	info, err := os.Stat(target)
	present := err == nil && !info.IsDir()
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("stat tunnel binary: %w", err)
	case err == nil && info.IsDir():
		return "", fmt.Errorf("tunnel binary %s is a directory", target)
	}

	if present {
		if _, err := os.Stat(filepath.Join(binDir, "."+name+".custom")); err == nil {
			return target, nil
		}
	}

	hash, err := fileSHA256(seed)
	if err != nil {
		if present {
			// No seed to compare against: the binary that exists is the one
			// there is, and nothing here can improve on it.
			return target, nil
		}
		return "", fmt.Errorf("open tunnel seed: %w", err)
	}

	recordPath := filepath.Join(binDir, "."+name+".seed")
	if present {
		if record, err := os.ReadFile(recordPath); err == nil && strings.TrimSpace(string(record)) == hash {
			return target, nil
		}
	}

	if err := extractSeedFrom(seed, target); err != nil {
		return "", err
	}
	if err := utils.WriteFileAtomic(recordPath, []byte(hash+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("record tunnel seed hash: %w", err)
	}
	return target, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func isCustom(name proto.TunnelName) bool {
	if _, err := os.Stat(binaryFile(name)); err != nil {
		return false
	}
	_, err := os.Stat(customMarkerPath(name))
	return err == nil
}

func setCustom(name proto.TunnelName, custom bool) error {
	path := customMarkerPath(name)
	if !custom {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("clear custom tunnel binary marker: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create tunnel binary directory: %w", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return fmt.Errorf("write custom tunnel binary marker: %w", err)
	}
	return nil
}

func renderWrapper(name proto.TunnelName, cfg Config, binary string) (string, error) {
	spec, ok := specOf(name)
	if !ok {
		return "", fmt.Errorf("unknown tunnel %s", name)
	}

	args, err := tokenize(cfg.Args)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(cfg.Env))
	for key := range cfg.Env {
		if !isValidEnvKey(key) {
			return "", fmt.Errorf("invalid environment variable name %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString("#!/bin/sh\n")
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("export %s=%s\n", key, shellQuote(cfg.Env[key])))
	}
	for _, entry := range spec.Env {
		builder.WriteString(fmt.Sprintf("export %s=%s\n", entry.Key, shellQuote(entry.Value)))
	}
	if limit, ok := memLimit(name); ok {
		builder.WriteString(fmt.Sprintf("export GOMEMLIMIT=%s\n", shellQuote(fmt.Sprintf("%dMiB", limit))))
	}

	command := []string{shellQuote(binary)}
	for _, arg := range spec.Args {
		command = append(command, shellQuote(arg))
	}
	for _, arg := range args {
		command = append(command, shellQuote(arg))
	}

	builder.WriteString(fmt.Sprintf("exec %s >>%s 2>&1\n", strings.Join(command, " "), shellQuote(logPath(name))))

	return builder.String(), nil
}

func writeWrapper(name proto.TunnelName, cfg Config) error {
	binary, err := binaryPath(name)
	if err != nil {
		return err
	}

	content, err := renderWrapper(name, cfg, binary)
	if err != nil {
		return err
	}

	file, err := utils.NewAtomicFile(wrapperPath(name), 0o700)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := io.WriteString(file, content); err != nil {
		return fmt.Errorf("write tunnel wrapper: %w", err)
	}
	return file.Commit()
}
