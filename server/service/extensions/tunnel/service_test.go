package tunnel

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"NanoKVM-Server/proto"
)

func useTestInitDirs(t *testing.T) (string, string) {
	t.Helper()
	scripts := t.TempDir()
	seeds := t.TempDir()
	oldInit, oldSeed := initDir, initSeedDir
	initDir, initSeedDir = scripts, seeds
	t.Cleanup(func() { initDir, initSeedDir = oldInit, oldSeed })
	return scripts, seeds
}

// pidOf believes a pid file only when /proc says the process behind it is the
// tunnel, so a fake running service needs both halves.
func useTestProcDirs(t *testing.T) (string, string) {
	t.Helper()
	pids := t.TempDir()
	procs := t.TempDir()
	oldPid, oldProc := pidDir, procDir
	pidDir, procDir = pids, procs
	t.Cleanup(func() { pidDir, procDir = oldPid, oldProc })
	return pids, procs
}

func fakeRunning(t *testing.T, name proto.TunnelName, pid int) {
	t.Helper()

	if err := os.WriteFile(pidPath(name), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(procDir, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(string(name)+"\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func callStop(t *testing.T, name proto.TunnelName) proto.Response {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/extensions/"+string(name)+"/stop", nil)

	NewService(name).Stop(c)

	var rsp proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode stop response %q: %v", recorder.Body.String(), err)
	}
	return rsp
}

// An action with no script to run touched nothing, and every caller used to be
// told it succeeded.
func TestRunInitScriptRefusesToRunNothing(t *testing.T) {
	useTestInitDirs(t)

	for _, action := range []string{"start", "stop", "restart"} {
		t.Run(action, func(t *testing.T) {
			err := runInitScript(proto.TunnelWstunnel, action)
			if !errors.Is(err, ErrNoInitScript) {
				t.Fatalf("%s returned %v, want ErrNoInitScript", action, err)
			}
		})
	}
}

// The regression: no script to stop with, a process still running, and Stop
// answering code 0 so the panel reported the tunnel down while it was up.
func TestStopFailsWhenItCannotStopARunningService(t *testing.T) {
	useTestInitDirs(t)
	useTestProcDirs(t)
	fakeRunning(t, proto.TunnelWstunnel, 4242)

	rsp := callStop(t, proto.TunnelWstunnel)
	if rsp.Code == 0 {
		t.Fatal("stop reported success for a running service it never stopped")
	}
}

// The other half: nothing running and no script is genuinely stopped, and a
// panel that cannot stop an already-stopped service would be worse than the lie.
func TestStopSucceedsWhenNothingIsRunning(t *testing.T) {
	useTestInitDirs(t)
	useTestProcDirs(t)

	rsp := callStop(t, proto.TunnelWstunnel)
	if rsp.Code != 0 {
		t.Fatalf("stop reported %d %q for a service that is not running", rsp.Code, rsp.Msg)
	}
}

// With a script present the stop runs it, and its failure is still the answer.
func TestStopRunsTheInitScript(t *testing.T) {
	scripts, _ := useTestInitDirs(t)
	useTestProcDirs(t)
	fakeRunning(t, proto.TunnelWstunnel, 4243)

	marker := filepath.Join(t.TempDir(), "stopped")
	script := "#!/bin/sh\necho \"$1\" > " + marker + "\n"
	if err := os.WriteFile(filepath.Join(scripts, "S97wstunnel"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := callStop(t, proto.TunnelWstunnel)
	if rsp.Code != 0 {
		t.Fatalf("stop reported %d %q", rsp.Code, rsp.Msg)
	}

	action, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("init script did not run: %v", err)
	}
	if got := string(action); got != "stop\n" {
		t.Fatalf("init script ran with %q, want stop", got)
	}
}

// The panel no longer carries its own copy of newt's variable names, so the
// seeded keys reaching an unconfigured service is what puts the form on screen.
func TestMaskEnvSeedsTheKeysASpecExpects(t *testing.T) {
	entries := maskEnv(proto.TunnelNewt, nil)

	want := specs[proto.TunnelNewt].SeededEnv
	if len(entries) != len(want) {
		t.Fatalf("%d entries, want %d", len(entries), len(want))
	}

	for i, key := range want {
		entry := entries[i]
		switch {
		case entry.Key != key:
			t.Fatalf("entry %d is %q, want %q", i, entry.Key, key)
		case entry.Configured:
			t.Fatalf("%s reported as configured with no config", key)
		case entry.Value != "":
			t.Fatalf("%s carried a value with no config", key)
		}
	}

	// wstunnel seeds nothing, so its form starts empty rather than with newt's
	// keys on it.
	if got := maskEnv(proto.TunnelWstunnel, nil); len(got) != 0 {
		t.Fatalf("wstunnel seeded %d entries, want none", len(got))
	}
}

// A secret never leaves the device, but the panel still has to show that one is
// set, or a saved secret looks unset and gets retyped.
func TestMaskEnvHidesSecretValues(t *testing.T) {
	entries := maskEnv(proto.TunnelNewt, map[string]string{
		"NEWT_ID":     "abc",
		"NEWT_SECRET": "shhh",
	})

	for _, entry := range entries {
		switch entry.Key {
		case "NEWT_ID":
			if entry.Value != "abc" || !entry.Configured {
				t.Fatalf("NEWT_ID = %+v, want the value carried", entry)
			}
		case "NEWT_SECRET":
			if entry.Value != "" || !entry.Configured || !entry.Secret {
				t.Fatalf("NEWT_SECRET = %+v, want masked but configured", entry)
			}
		}
	}
}
