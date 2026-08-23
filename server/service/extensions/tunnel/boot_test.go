package tunnel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"NanoKVM-Server/proto"
)

// The seed script every enable copies from, and the same file an installer
// would copy over /etc/init.d.
func seedInitScript(t *testing.T, name proto.TunnelName, marker string) {
	t.Helper()

	script := "#!/bin/sh\necho \"$1\" >> " + marker + "\n"
	if err := os.WriteFile(initSeedPath(name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installFromSeed(t *testing.T, name proto.TunnelName) {
	t.Helper()

	content, err := os.ReadFile(initSeedPath(name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initScriptPath(name), content, 0o755); err != nil {
		t.Fatal(err)
	}
}

func callLifecycle(t *testing.T, name proto.TunnelName, action string) proto.Response {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/extensions/tunnel/"+string(name)+"/"+action, nil)

	service := NewService(name)
	if action == "stop" {
		service.Stop(c)
	} else {
		service.lifecycle(c, action)
	}

	var rsp proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode %s response %q: %v", action, recorder.Body.String(), err)
	}
	if rsp.Code != 0 {
		t.Fatalf("%s reported %d %q", action, rsp.Code, rsp.Msg)
	}
	return rsp
}

func storedIntent(t *testing.T, name proto.TunnelName) *bool {
	t.Helper()

	cfg, err := loadConfig(name)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Enabled
}

func useBootTestDirs(t *testing.T, name proto.TunnelName) string {
	t.Helper()

	useTestConfigDir(t)
	useTestInitDirs(t)
	useTestProcDirs(t)
	bins, _ := useTestBinDirs(t)
	if err := os.WriteFile(filepath.Join(bins, string(name)), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(t.TempDir(), "actions")
	seedInitScript(t, name, marker)
	return marker
}

func TestStartAndStopRecordBootIntent(t *testing.T) {
	useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	if intent := storedIntent(t, proto.TunnelNewt); intent == nil || !*intent {
		t.Fatalf("boot intent after start = %v, want true", intent)
	}
	if !intendedEnabled(proto.TunnelNewt) {
		t.Fatal("started service does not start at boot")
	}

	callLifecycle(t, proto.TunnelNewt, "stop")
	if intent := storedIntent(t, proto.TunnelNewt); intent == nil || *intent {
		t.Fatalf("boot intent after stop = %v, want false", intent)
	}
	if intendedEnabled(proto.TunnelNewt) {
		t.Fatal("stopped service still starts at boot")
	}
}

func TestRestartKeepsBootIntent(t *testing.T) {
	useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	callLifecycle(t, proto.TunnelNewt, "restart")

	if intent := storedIntent(t, proto.TunnelNewt); intent == nil || !*intent {
		t.Fatalf("boot intent after restart = %v, want true", intent)
	}
}

// package.sh ships /kvmapp wholesale and installers here copy init scripts out
// of it. A service the user stopped must not come back because an update put
// its script back in /etc/init.d.
func TestUpdateCannotReviveAStoppedService(t *testing.T) {
	marker := useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	callLifecycle(t, proto.TunnelNewt, "stop")

	installFromSeed(t, proto.TunnelNewt)
	Reconcile([]proto.TunnelName{proto.TunnelNewt})

	if isEnabled(proto.TunnelNewt) {
		t.Fatal("an update re-enabled a service the user stopped")
	}
	if intendedEnabled(proto.TunnelNewt) {
		t.Fatal("a stopped service reports that it starts at boot")
	}

	actions, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("init script never ran: %v", err)
	}
	if want := "start\nstop\nstop\n"; string(actions) != want {
		t.Fatalf("init script actions = %q, want %q", actions, want)
	}
}

// The watchdog is what would otherwise restart it 30 seconds later.
func TestWatchdogLeavesAStoppedServiceAlone(t *testing.T) {
	marker := useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	callLifecycle(t, proto.TunnelNewt, "stop")
	installFromSeed(t, proto.TunnelNewt)

	revive([]proto.TunnelName{proto.TunnelNewt})

	actions, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if want := "start\nstop\n"; string(actions) != want {
		t.Fatalf("init script actions = %q, want %q: the watchdog restarted a stopped service", actions, want)
	}
}

// A device upgrading from a build with no recorded intent has only the init
// script to go on, and its running tunnel must not be disabled by the upgrade.
func TestReconcileAdoptsAnAlreadyEnabledService(t *testing.T) {
	useBootTestDirs(t, proto.TunnelNewt)

	if err := saveConfig(proto.TunnelNewt, Config{Args: "--foreground"}); err != nil {
		t.Fatal(err)
	}
	installFromSeed(t, proto.TunnelNewt)

	Reconcile([]proto.TunnelName{proto.TunnelNewt})

	if !isEnabled(proto.TunnelNewt) {
		t.Fatal("an upgrade disabled a service that was starting at boot")
	}
	if intent := storedIntent(t, proto.TunnelNewt); intent == nil || !*intent {
		t.Fatalf("adopted boot intent = %v, want true", intent)
	}
}

// An update ships a new S97<name> in /kvmapp and nothing else installs it, so
// an enabled service would keep running the copy it was enabled with.
func TestReconcileRefreshesTheInitScriptFromTheSeed(t *testing.T) {
	marker := useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	seedInitScript(t, proto.TunnelNewt, marker+".updated")

	Reconcile([]proto.TunnelName{proto.TunnelNewt})

	installed, err := os.ReadFile(initScriptPath(proto.TunnelNewt))
	if err != nil {
		t.Fatal(err)
	}
	seeded, err := os.ReadFile(initSeedPath(proto.TunnelNewt))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(seeded) {
		t.Fatalf("installed script = %q, want the updated seed %q", installed, seeded)
	}
}

func TestReconcileRestoresAClobberedInitScript(t *testing.T) {
	useBootTestDirs(t, proto.TunnelNewt)

	callLifecycle(t, proto.TunnelNewt, "start")
	if err := os.Remove(initScriptPath(proto.TunnelNewt)); err != nil {
		t.Fatal(err)
	}

	Reconcile([]proto.TunnelName{proto.TunnelNewt})

	if !isEnabled(proto.TunnelNewt) {
		t.Fatal("an enabled service was left with no init script")
	}
}
