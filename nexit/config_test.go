package main

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

func fakeEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func fakeUserDir(dir string, err error) func() (string, error) {
	return func() (string, error) { return dir, err }
}

func TestConfigCandidates(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		userDir string
		userErr error
		cwd     string
		exeDir  string
		goos    string
		want    []string
	}{
		{
			name:    "windows with ProgramData and AppData",
			env:     map[string]string{"ProgramData": `C:\ProgramData`},
			userDir: `C:\Users\bob\AppData\Roaming`,
			cwd:     `C:\Users\bob\Desktop`,
			exeDir:  `C:\Users\bob\Downloads`,
			goos:    "windows",
			want: []string{
				`C:\ProgramData\nexit\nexit.json`,
				`C:\Users\bob\AppData\Roaming\nexit\nexit.json`,
				`C:\Users\bob\Desktop\nexit.json`,
				`C:\Users\bob\Downloads\nexit.json`,
			},
		},
		{
			name:    "windows with an empty ProgramData skips the system file",
			env:     map[string]string{},
			userDir: `C:\Users\bob\AppData\Roaming`,
			cwd:     `C:\Users\bob\Desktop`,
			exeDir:  `C:\Users\bob\Downloads`,
			goos:    "windows",
			want: []string{
				`C:\Users\bob\AppData\Roaming\nexit\nexit.json`,
				`C:\Users\bob\Desktop\nexit.json`,
				`C:\Users\bob\Downloads\nexit.json`,
			},
		},
		{
			name:    "windows trailing separators and a shared cwd and exe dir",
			env:     map[string]string{"ProgramData": `C:\ProgramData\`},
			userDir: `C:\Users\bob\AppData\Roaming`,
			cwd:     `C:\Users\bob\Downloads`,
			exeDir:  `C:\Users\bob\Downloads`,
			goos:    "windows",
			want: []string{
				`C:\ProgramData\nexit\nexit.json`,
				`C:\Users\bob\AppData\Roaming\nexit\nexit.json`,
				`C:\Users\bob\Downloads\nexit.json`,
			},
		},
		{
			name:    "no user config dir and no cwd",
			env:     map[string]string{"ProgramData": `C:\ProgramData`},
			userErr: errors.New("no home"),
			exeDir:  `C:\tools`,
			goos:    "windows",
			want: []string{
				`C:\ProgramData\nexit\nexit.json`,
				`C:\tools\nexit.json`,
			},
		},
		{
			name:    "linux ignores ProgramData",
			env:     map[string]string{"ProgramData": `C:\ProgramData`},
			userDir: "/home/bob/.config",
			cwd:     "/home/bob/work",
			exeDir:  "/opt/nexit",
			goos:    "linux",
			want: []string{
				"/etc/nexit/nexit.json",
				"/home/bob/.config/nexit/nexit.json",
				"/home/bob/work/nexit.json",
				"/opt/nexit/nexit.json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configCandidates(fakeEnv(tt.env), fakeUserDir(tt.userDir, tt.userErr), tt.cwd, tt.exeDir, tt.goos)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

const goodJSON = `{"address":"wss://kvm/exit/0","passcode":"p1"}`

func fakeFiles(files map[string]string, errs map[string]error) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if err, ok := errs[p]; ok {
			return nil, err
		}
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}
		return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
	}
}

func TestLoadConfig(t *testing.T) {
	paths := []string{"/sys/nexit.json", "/user/nexit.json", "/cwd/nexit.json", "/exe/nexit.json"}
	tru := true
	tests := []struct {
		name     string
		files    map[string]string
		errs     map[string]error
		wantPath string
		wantCfg  fileConfig
		wantErr  []string // substrings the error must contain
	}{
		{
			name: "system file wins when all exist",
			files: map[string]string{
				"/sys/nexit.json":  `{"address":"sys","passcode":"a"}`,
				"/user/nexit.json": `{"address":"user","passcode":"b"}`,
				"/cwd/nexit.json":  `{"address":"cwd","passcode":"c"}`,
				"/exe/nexit.json":  `{"address":"exe","passcode":"d"}`,
			},
			wantPath: "/sys/nexit.json",
			wantCfg:  fileConfig{Address: "sys", Passcode: "a"},
		},
		{
			name:     "missing files fall through to the next",
			files:    map[string]string{"/exe/nexit.json": `{"address":"exe","passcode":"d","insecure":true,"allowPrivate":false}`},
			wantPath: "/exe/nexit.json",
			wantCfg:  fileConfig{Address: "exe", Passcode: "d", Insecure: &tru, AllowPrivate: new(bool)},
		},
		{
			name:     "a leading UTF-8 BOM is accepted",
			files:    map[string]string{"/user/nexit.json": "\xef\xbb\xbf" + goodJSON},
			wantPath: "/user/nexit.json",
			wantCfg:  fileConfig{Address: "wss://kvm/exit/0", Passcode: "p1"},
		},
		{
			name:    "bad JSON names the path",
			files:   map[string]string{"/user/nexit.json": `{"address":`},
			wantErr: []string{"/user/nexit.json"},
		},
		{
			name:    "trailing data names the path",
			files:   map[string]string{"/user/nexit.json": goodJSON + `{}`},
			wantErr: []string{"/user/nexit.json"},
		},
		{
			name:    "unknown key names the path and the key",
			files:   map[string]string{"/cwd/nexit.json": `{"address":"a","passcode":"b","insecur":true}`},
			wantErr: []string{"/cwd/nexit.json", "insecur"},
		},
		{
			name:    "missing passcode names the path",
			files:   map[string]string{"/cwd/nexit.json": `{"address":"a"}`},
			wantErr: []string{"/cwd/nexit.json", "passcode"},
		},
		{
			name:    "missing address names the path",
			files:   map[string]string{"/cwd/nexit.json": `{"passcode":"b"}`},
			wantErr: []string{"/cwd/nexit.json", "address"},
		},
		{
			name:    "a read error other than not-found names the path",
			errs:    map[string]error{"/user/nexit.json": fs.ErrPermission},
			files:   map[string]string{"/cwd/nexit.json": goodJSON},
			wantErr: []string{"/user/nexit.json", "permission"},
		},
		{
			name:    "not found lists every searched path",
			wantErr: append([]string{"no nexit.json"}, paths...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, path, err := loadConfig(paths, fakeFiles(tt.files, tt.errs))
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("want an error, got config from %s", path)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("error %q does not mention %q", err, s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if path != tt.wantPath {
				t.Errorf("path %q, want %q", path, tt.wantPath)
			}
			if !reflect.DeepEqual(cfg, tt.wantCfg) {
				t.Errorf("cfg %+v, want %+v", cfg, tt.wantCfg)
			}
		})
	}
}

func TestMergeFlags(t *testing.T) {
	tru, fal := true, false
	base := fileConfig{Address: "wss://kvm/exit/1", Passcode: "file", Insecure: &tru, AllowPrivate: &fal}
	tests := []struct {
		name string
		cfg  fileConfig
		set  map[string]string
		want settings
	}{
		{
			name: "file values when no flag is set",
			cfg:  base,
			want: settings{address: "wss://kvm/exit/1", passcode: "file", insecure: true, allowPrivate: false},
		},
		{
			name: "absent bools default to false",
			cfg:  fileConfig{Address: "a", Passcode: "p"},
			want: settings{address: "a", passcode: "p"},
		},
		{
			name: "explicit flags override the file",
			cfg:  base,
			set:  map[string]string{"passcode": "flag", "insecure": "false", "allow-private": "true"},
			want: settings{address: "wss://kvm/exit/1", passcode: "flag", insecure: false, allowPrivate: true},
		},
		{
			name: "only the set flag changes",
			cfg:  base,
			set:  map[string]string{"allow-private": "true", "version": "false"},
			want: settings{address: "wss://kvm/exit/1", passcode: "file", insecure: true, allowPrivate: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeFlags(tt.cfg, tt.set)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	if _, err := mergeFlags(base, map[string]string{"insecure": "maybe"}); err == nil {
		t.Error("want an error for a non-bool insecure value")
	}
}
