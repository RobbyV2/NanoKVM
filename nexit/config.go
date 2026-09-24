package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"
)

// configName is the file nexit looks for when it is started without an
// address, which is what a double-click does.
const configName = "nexit.json"

// configShape is shown to the user when no usable config file is found.
const configShape = `{
  "address": "wss://nanokvm.local/exit/0",
  "passcode": "f8n9yze8",
  "insecure": false,
  "allowPrivate": false
}`

// fileConfig is the content of nexit.json. The bools are pointers so a key
// that is absent can be told apart from one set to false.
type fileConfig struct {
	Address      string `json:"address"`
	Passcode     string `json:"passcode"`
	Insecure     *bool  `json:"insecure"`
	AllowPrivate *bool  `json:"allowPrivate"`
}

// settings is what a run needs, whether it came from the command line or
// from a config file.
type settings struct {
	address      string
	passcode     string
	insecure     bool
	allowPrivate bool
}

// configCandidates returns, in search order, the paths nexit.json may live
// at: the system-wide file, the user's file, the current directory and the
// directory of the executable. It takes its inputs as arguments so it stays
// pure; goos picks the path separator and the system location.
func configCandidates(env func(string) string, userConfigDir func() (string, error), cwd string, exeDir string, goos string) []string {
	sep := "/"
	if goos == "windows" {
		sep = `\`
	}
	join := func(dir string, parts ...string) string {
		dir = strings.TrimRight(dir, `/\`)
		if dir == "" {
			dir = sep // a root, trimmed to nothing
		} else {
			dir += sep
		}
		return dir + strings.Join(parts, sep)
	}

	var out []string
	add := func(p string) {
		for _, seen := range out {
			if seen == p {
				return
			}
		}
		out = append(out, p)
	}

	if goos == "windows" {
		if pd := env("ProgramData"); pd != "" {
			add(join(pd, "nexit", configName))
		}
	} else {
		add("/etc/nexit/" + configName)
	}
	if dir, err := userConfigDir(); err == nil && dir != "" {
		add(join(dir, "nexit", configName))
	}
	if cwd != "" {
		add(join(cwd, configName))
	}
	if exeDir != "" {
		add(join(exeDir, configName))
	}
	return out
}

// loadConfig reads the first of paths that exists. Files that do not exist
// are skipped; any other failure stops the search and names the file, so a
// broken file is never silently passed over for a later one.
func loadConfig(paths []string, readFile func(string) ([]byte, error)) (cfg fileConfig, path string, err error) {
	for _, p := range paths {
		data, err := readFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fileConfig{}, p, fmt.Errorf("reading %s: %w", p, err)
		}
		cfg, err := parseConfig(data)
		if err != nil {
			return fileConfig{}, p, fmt.Errorf("%s: %w", p, err)
		}
		return cfg, p, nil
	}
	return fileConfig{}, "", fmt.Errorf("no %s found; searched:\n  %s", configName, strings.Join(paths, "\n  "))
}

func parseConfig(data []byte) (fileConfig, error) {
	var cfg fileConfig
	// Windows editors such as Notepad may save the file with a UTF-8 BOM.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return fileConfig{}, fmt.Errorf("not valid %s: %w", configName, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fileConfig{}, fmt.Errorf("not valid %s: extra data after the JSON object", configName)
	}
	if strings.TrimSpace(cfg.Address) == "" {
		return fileConfig{}, errors.New(`"address" is missing or empty`)
	}
	if cfg.Passcode == "" {
		return fileConfig{}, errors.New(`"passcode" is missing or empty`)
	}
	return cfg, nil
}

// mergeFlags lays the flags that were given explicitly over the file's
// values. set maps a flag name to its value, as flag.FlagSet.Visit sees it.
func mergeFlags(cfg fileConfig, set map[string]string) (settings, error) {
	s := settings{address: cfg.Address, passcode: cfg.Passcode}
	if cfg.Insecure != nil {
		s.insecure = *cfg.Insecure
	}
	if cfg.AllowPrivate != nil {
		s.allowPrivate = *cfg.AllowPrivate
	}
	if v, ok := set["passcode"]; ok {
		s.passcode = v
	}
	for name, dst := range map[string]*bool{"insecure": &s.insecure, "allow-private": &s.allowPrivate} {
		v, ok := set[name]
		if !ok {
			continue
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			return settings{}, fmt.Errorf("--%s: %w", name, err)
		}
		*dst = b
	}
	return s, nil
}
