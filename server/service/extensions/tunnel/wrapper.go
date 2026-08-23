package tunnel

import (
	"compress/gzip"
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
	source, err := os.Open(seedPath(name))
	if err != nil {
		return fmt.Errorf("open tunnel seed: %w", err)
	}
	defer func() { _ = source.Close() }()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("read tunnel seed: %w", err)
	}
	defer func() { _ = reader.Close() }()

	file, err := utils.NewAtomicFile(binaryFile(name), 0o755)
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
	target := binaryFile(name)

	info, err := os.Stat(target)
	switch {
	case err == nil && !info.IsDir():
		return target, nil
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("stat tunnel binary: %w", err)
	case err == nil:
		return "", fmt.Errorf("tunnel binary %s is a directory", target)
	}

	if err := extractSeed(name); err != nil {
		return "", err
	}
	return target, nil
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
