package edid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

// buildMismatchStderr reproduces what the tool writes on a failed readback:
// the marker line, the written buffer as "%02X " sixteen per line, a blank
// line, the read buffer, and finally main's own line.
func buildMismatchStderr(written, read []byte) string {
	var out strings.Builder
	out.WriteString("EDID data mismatch after write/read cycle\n")
	out.WriteString("Written EDID data:\n")
	writeDump(&out, written)
	out.WriteString("\nRead EDID data:\n")
	writeDump(&out, read)
	out.WriteString("Failed to configure LT6911 EDID\n")
	return out.String()
}

func writeDump(out *strings.Builder, data []byte) {
	for i, b := range data {
		fmt.Fprintf(out, "%02X ", b)
		if (i+1)%16 == 0 {
			out.WriteString("\n")
		}
	}
}

func expandHex(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, " ")
}

func TestClassifyCoversEveryDocumentedMessage(t *testing.T) {
	tests := []struct {
		name      string
		result    Result
		state     proto.EdidState
		retryable bool
		verified  bool
	}{
		{
			name:   "mismatch",
			result: Result{ExitCode: 1, Stderr: "EDID data mismatch after write/read cycle\nFailed to configure LT6911 EDID\n"},
			state:  proto.EdidStateNeedsRecovery,
		},
		{
			name:   "unsupported chip",
			result: Result{ExitCode: 1, Stderr: "Unsupported chip version\nFailed to write EDID data to LT6911UXC\n"},
			state:  proto.EdidStateChipRefused,
		},
		{
			name:   "erase verify",
			result: Result{ExitCode: 1, Stderr: "Clean Error\n"},
			state:  proto.EdidStateChipRefused,
		},
		{
			name:   "lt6911d version page",
			result: Result{ExitCode: 1, Stderr: "Failed to read LT6911D version data\n"},
			state:  proto.EdidStateChipRefused,
		},
		{
			name:      "bus open",
			result:    Result{ExitCode: 1, Stderr: "Failed to open the i2c bus: No such file or directory\n"},
			state:     proto.EdidStateBusContention,
			retryable: true,
		},
		{
			name:      "bus claim",
			result:    Result{ExitCode: 1, Stderr: "Failed to acquire bus access and/or talk to slave: Device or resource busy\n"},
			state:     proto.EdidStateBusContention,
			retryable: true,
		},
		{
			name:   "chip version ue",
			result: Result{ExitCode: 1, Stderr: "Chip Version Error: UE version's edid can't be updated\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "chip version unknown",
			result: Result{ExitCode: 1, Stderr: "Chip Version Error: Unknown version\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "product version unknown",
			result: Result{ExitCode: 1, Stderr: "Product Version Error: Unknown version\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "version file missing",
			result: Result{ExitCode: 1, Stderr: "Please upgrade to the latest system\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "chip version unreadable",
			result: Result{ExitCode: 1, Stderr: "Failed to read chip version\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "product version unreadable",
			result: Result{ExitCode: 1, Stderr: "Failed to read product version\n"},
			state:  proto.EdidStatePreflight,
		},
		{
			name:   "check_edid rejected",
			result: Result{ExitCode: 1, Stderr: "EDID data is invalid\n"},
			state:  proto.EdidStateInvalidInput,
		},
		{
			name:   "wrong length",
			result: Result{ExitCode: 1, Stderr: "EDID data length is not 256 bytes\nEDID data is invalid\n"},
			state:  proto.EdidStateInvalidInput,
		},
		{
			name:   "bad header",
			result: Result{ExitCode: 1, Stderr: "EDID header is invalid\nEDID data is invalid\n"},
			state:  proto.EdidStateInvalidInput,
		},
		{
			name:   "first block checksum",
			result: Result{ExitCode: 1, Stderr: "Checksum for first 128 bytes is incorrect\n"},
			state:  proto.EdidStateInvalidInput,
		},
		{
			name:   "second block checksum",
			result: Result{ExitCode: 1, Stderr: "Checksum for second 128 bytes is incorrect\n"},
			state:  proto.EdidStateInvalidInput,
		},
		{
			name:     "verified",
			result:   Result{ExitCode: 0, Stdout: "Writing EDID....\nEDID write completed\nEDID data verified successfully\n"},
			state:    proto.EdidStateSuccess,
			verified: true,
		},
		{
			name:   "exit zero without the verification line",
			result: Result{ExitCode: 0, Stdout: "Writing EDID....\nEDID write completed\n"},
			state:  proto.EdidStateUnverified,
		},
		{
			name:   "unrecognised failure",
			result: Result{ExitCode: 1, Stderr: "Failed to write to the i2c bus: Remote I/O error\n"},
			state:  proto.EdidStateGeneric,
		},
		{
			name:   "timeout",
			result: Result{ExitCode: -1, TimedOut: true},
			state:  proto.EdidStateTimeout,
		},
		{
			name:   "spawn failure",
			result: Result{ExitCode: -1, SpawnErr: errors.New("fork/exec: permission denied")},
			state:  proto.EdidStateGeneric,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outcome := Classify(test.result)

			if outcome.State != test.state {
				t.Fatalf("state %q, want %q", outcome.State, test.state)
			}
			if outcome.Retryable != test.retryable {
				t.Fatalf("retryable %v, want %v", outcome.Retryable, test.retryable)
			}
			if outcome.Verified != test.verified {
				t.Fatalf("verified %v, want %v", outcome.Verified, test.verified)
			}
			if outcome.Message == "" {
				t.Fatal("outcome carries no message")
			}
		})
	}
}

// A mismatch is the one outcome nobody can diagnose any other way, so the two
// dumps have to survive the trip out of stderr byte for byte.
func TestClassifyRecoversBothHexDumps(t *testing.T) {
	written := fixture(t)
	read := bytes.Clone(written)
	read[100] ^= 0xFF
	read[200] ^= 0x0F

	outcome := Classify(Result{ExitCode: 1, Stderr: buildMismatchStderr(written, read)})

	if outcome.State != proto.EdidStateNeedsRecovery {
		t.Fatalf("state %q, want %q", outcome.State, proto.EdidStateNeedsRecovery)
	}
	if outcome.Retryable {
		t.Fatal("a mismatch must never be reported as retryable")
	}
	if got, want := outcome.WrittenHex, expandHex(written); got != want {
		t.Fatalf("written dump did not survive:\n got %s\nwant %s", got, want)
	}
	if got, want := outcome.ReadHex, expandHex(read); got != want {
		t.Fatalf("read dump did not survive:\n got %s\nwant %s", got, want)
	}
}

// Only bus contention is auto-retried, and only once. Everything else that
// leaves the region in an unknown state stays put.
func TestOnlyBusContentionIsRetryable(t *testing.T) {
	for _, state := range []proto.EdidState{
		proto.EdidStateNeedsRecovery, proto.EdidStateTimeout, proto.EdidStateGeneric,
		proto.EdidStatePreflight, proto.EdidStateInvalidInput, proto.EdidStateChipRefused,
	} {
		for _, row := range stderrRows {
			if row.state == state && row.retryable {
				t.Fatalf("stderr row %q marks %s retryable", row.marker, state)
			}
		}
	}
}

func swapString(t *testing.T, target *string, value string) {
	t.Helper()
	old := *target
	*target = value
	t.Cleanup(func() { *target = old })
}

func writeTemp(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// usePreflightFiles points the preflight at a temporary /etc/kvm, a fake tool
// and a fake gpio node, and returns the directory so a case can delete one.
func usePreflightFiles(t *testing.T, chip, product string) string {
	t.Helper()
	dir := t.TempDir()

	swapString(t, &chipVersionFile, writeTemp(t, filepath.Join(dir, "hdmi_version"), chip))
	swapString(t, &productFile, writeTemp(t, filepath.Join(dir, "hw"), product))
	swapString(t, &toolPath, writeTemp(t, filepath.Join(dir, "nanokvm_update_edid"), ""))
	swapString(t, &resetGPIOFile, writeTemp(t, filepath.Join(dir, "gpio451"), "0\n"))
	return dir
}

func TestPreflightMatchesTheToolsStrcmpTable(t *testing.T) {
	tests := []struct {
		name       string
		chip       string
		product    string
		chipWant   Chip
		product2   Product
		supported  bool
		powerCycle bool
		reason     string
	}{
		{name: "lt6911c on alpha", chip: "c\n", product: "alpha\n", chipWant: ChipLT6911C, product2: ProductCubeA, supported: true, powerCycle: true},
		{name: "lt6911uxc on beta", chip: "ux\n", product: "beta\n", chipWant: ChipLT6911UXC, product2: ProductCubeB, supported: true, powerCycle: true},
		{name: "lt6911d on pcie", chip: "d\n", product: "pcie\n", chipWant: ChipLT6911D, product2: ProductPCIeA, supported: true},
		{name: "no trailing newline", chip: "c", product: "pcie", chipWant: ChipLT6911C, product2: ProductPCIeA, supported: true},

		{name: "ue is refused", chip: "ue\n", product: "pcie\n", reason: "Chip Version Error: UE version's edid can't be updated"},
		{name: "unknown chip", chip: "zz\n", product: "pcie\n", reason: "Chip Version Error: Unknown version"},
		{name: "crlf is not a match", chip: "c\r\n", product: "pcie\n", reason: "Chip Version Error: Unknown version"},
		{name: "trailing space is not a match", chip: "c \n", product: "pcie\n", reason: "Chip Version Error: Unknown version"},
		{name: "empty chip file", chip: "", product: "pcie\n", reason: "Failed to read chip version"},

		{name: "unknown product", chip: "c\n", product: "gamma\n", reason: "Product Version Error: Unknown version"},
		{name: "empty product file", chip: "c\n", product: "", reason: "Failed to read product version"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			usePreflightFiles(t, test.chip, test.product)

			result := Preflight()

			if result.Supported != test.supported {
				t.Fatalf("supported %v, want %v (reason %q)", result.Supported, test.supported, result.Reason)
			}
			if test.supported {
				if result.Chip != test.chipWant {
					t.Fatalf("chip %q, want %q", result.Chip, test.chipWant)
				}
				if result.Product != test.product2 {
					t.Fatalf("product %q, want %q", result.Product, test.product2)
				}
				if result.RequiresPowerCycle != test.powerCycle {
					t.Fatalf("requiresPowerCycle %v, want %v", result.RequiresPowerCycle, test.powerCycle)
				}
				return
			}
			if !strings.HasPrefix(result.Reason, test.reason) {
				t.Fatalf("reason %q, want prefix %q", result.Reason, test.reason)
			}
		})
	}
}

// config.GetHwVersion() defaults unknown /etc/kvm/hw content to alpha, where
// the tool fails. The preflight must not inherit that, or an apply would prompt
// and flash on a board nobody has identified.
func TestPreflightDoesNotDefaultUnknownHardwareToAlpha(t *testing.T) {
	usePreflightFiles(t, "c\n", "something-nobody-shipped\n")

	result := Preflight()

	if result.Supported {
		t.Fatal("unknown /etc/kvm/hw content was accepted")
	}
	if result.Product == ProductCubeA {
		t.Fatal("unknown /etc/kvm/hw content was resolved to alpha")
	}
}

func TestPreflightMissingVersionFile(t *testing.T) {
	dir := usePreflightFiles(t, "c\n", "pcie\n")
	if err := os.Remove(filepath.Join(dir, "hdmi_version")); err != nil {
		t.Fatalf("remove version file: %v", err)
	}

	result := Preflight()

	if result.Supported {
		t.Fatal("a missing /etc/kvm/hdmi_version was accepted")
	}
	if result.Reason != "Please upgrade to the latest system" {
		t.Fatalf("reason %q, want the tool's own message", result.Reason)
	}
}

func TestPreflightMissingTool(t *testing.T) {
	dir := usePreflightFiles(t, "c\n", "pcie\n")
	if err := os.Remove(filepath.Join(dir, "nanokvm_update_edid")); err != nil {
		t.Fatalf("remove tool: %v", err)
	}

	result := Preflight()

	if result.Supported || result.ToolAvailable {
		t.Fatal("preflight passed with no tool on disk")
	}
}

type fakeFlasher struct {
	results []Result
	calls   int
	paths   []string
}

func (f *fakeFlasher) Flash(_ context.Context, blobPath string) Result {
	f.paths = append(f.paths, blobPath)

	index := f.calls
	if index >= len(f.results) {
		index = len(f.results) - 1
	}
	f.calls++
	return f.results[index]
}

type captureLog struct {
	disabled int
	enabled  int
}

// useApplier wires a fake flasher, a temporary store and a counted capture
// guard, so an apply can be driven end to end with no device.
func useApplier(t *testing.T, flasher Flasher) (*Applier, *captureLog) {
	t.Helper()
	usePreflightFiles(t, "c\n", "alpha\n")
	swapString(t, &storeDir, t.TempDir())

	counted := &captureLog{}
	guard := CaptureGuard{
		Disable: func() { counted.disabled++ },
		Enable:  func() { counted.enabled++ },
	}

	return &Applier{store: NewStore(), flasher: flasher, capture: guard}, counted
}

func verifiedRun() Result {
	return Result{ExitCode: 0, Stdout: "EDID write completed\nEDID data verified successfully\n"}
}

func TestApplyArchivesOnlyAfterTheReadbackIsConfirmed(t *testing.T) {
	blob := fixture(t)
	flasher := &fakeFlasher{results: []Result{verifiedRun()}}
	applier, capture := useApplier(t, flasher)

	outcome, err := applier.Apply(context.Background(), blob, "upload")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if outcome.State != proto.EdidStateSuccess || !outcome.Verified {
		t.Fatalf("state %q verified %v, want success", outcome.State, outcome.Verified)
	}
	if !outcome.RequiresPowerCycle {
		t.Fatal("a verified flash on alpha must require a power cycle")
	}
	if outcome.Record == nil || outcome.Record.Source != "upload" {
		t.Fatalf("record %+v, want one naming the source", outcome.Record)
	}
	if capture.disabled != 1 || capture.enabled != 1 {
		t.Fatalf("capture guard taken %d times and released %d, want 1 and 1", capture.disabled, capture.enabled)
	}

	archived, record, err := applier.store.LoadActive()
	if err != nil {
		t.Fatalf("load active: %v", err)
	}
	if !bytes.Equal(archived, blob) {
		t.Fatal("archived bytes are not the bytes that were flashed")
	}
	if record == nil || record.PreferredMode != "1920x1080p60" {
		t.Fatalf("record %+v, want the decoded summary", record)
	}

	// The staged blob is the child's only argument and must not outlive the run.
	if len(flasher.paths) != 1 {
		t.Fatalf("%d spawns, want 1", len(flasher.paths))
	}
	if _, err := os.Stat(flasher.paths[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged blob %s survived the apply", flasher.paths[0])
	}
}

func TestApplyDoesNotArchiveAMismatch(t *testing.T) {
	blob := fixture(t)
	read := bytes.Clone(blob)
	read[64] ^= 0xFF

	flasher := &fakeFlasher{results: []Result{{ExitCode: 1, Stderr: buildMismatchStderr(blob, read)}}}
	applier, _ := useApplier(t, flasher)

	outcome, err := applier.Apply(context.Background(), blob, "upload")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if outcome.State != proto.EdidStateNeedsRecovery {
		t.Fatalf("state %q, want needs_recovery", outcome.State)
	}
	if outcome.Retryable {
		t.Fatal("a mismatch was reported as retryable")
	}
	if flasher.calls != 1 {
		t.Fatalf("%d spawns, want exactly 1: a mismatch must never be re-attempted", flasher.calls)
	}
	if outcome.WrittenHex == "" || outcome.ReadHex == "" {
		t.Fatal("the two dumps did not reach the outcome")
	}
	if !outcome.RequiresPowerCycle {
		t.Fatal("a half-written region on alpha needs a power cycle before anything else")
	}

	data, _, err := applier.store.LoadActive()
	if err != nil {
		t.Fatalf("load active: %v", err)
	}
	if data != nil {
		t.Fatal("a failed apply archived bytes the chip never accepted")
	}
}

func TestApplyRetriesBusContentionExactlyOnce(t *testing.T) {
	blob := fixture(t)
	flasher := &fakeFlasher{results: []Result{
		{ExitCode: 1, Stderr: "Failed to acquire bus access and/or talk to slave: Device or resource busy\n"},
		verifiedRun(),
	}}
	applier, capture := useApplier(t, flasher)

	outcome, err := applier.Apply(context.Background(), blob, "upload")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if flasher.calls != 2 {
		t.Fatalf("%d spawns, want 2", flasher.calls)
	}
	if outcome.State != proto.EdidStateSuccess {
		t.Fatalf("state %q, want success on the retry", outcome.State)
	}
	// The retry re-takes the capture guard rather than reusing a stale one.
	if capture.disabled != 2 || capture.enabled != 2 {
		t.Fatalf("capture guard taken %d times and released %d, want 2 and 2", capture.disabled, capture.enabled)
	}
}

func TestApplyGivesUpAfterOneRetry(t *testing.T) {
	blob := fixture(t)
	contention := Result{ExitCode: 1, Stderr: "Failed to open the i2c bus: Device or resource busy\n"}
	flasher := &fakeFlasher{results: []Result{contention}}
	applier, _ := useApplier(t, flasher)

	outcome, err := applier.Apply(context.Background(), blob, "upload")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if flasher.calls != 2 {
		t.Fatalf("%d spawns, want 2", flasher.calls)
	}
	if outcome.Retryable {
		t.Fatal("the outcome still invites a retry after one was already spent")
	}
}

func TestApplyRejectsBeforeSpawning(t *testing.T) {
	tests := []struct {
		name  string
		chip  string
		blob  func(t *testing.T) []byte
		state proto.EdidState
	}{
		{
			name:  "refused chip version",
			chip:  "ue\n",
			blob:  fixture,
			state: proto.EdidStatePreflight,
		},
		{
			name:  "unknown chip version",
			chip:  "ux \n",
			blob:  fixture,
			state: proto.EdidStatePreflight,
		},
		{
			name: "oversized blob the tool would have truncated",
			chip: "c\n",
			blob: func(t *testing.T) []byte {
				return append(fixture(t), 0x00)
			},
			state: proto.EdidStateInvalidInput,
		},
		{
			name: "broken checksum",
			chip: "c\n",
			blob: func(t *testing.T) []byte {
				blob := fixture(t)
				blob[127]++
				return blob
			},
			state: proto.EdidStateInvalidInput,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blob := test.blob(t)
			flasher := &fakeFlasher{results: []Result{verifiedRun()}}
			applier, capture := useApplier(t, flasher)
			swapString(t, &chipVersionFile, writeTemp(t, filepath.Join(t.TempDir(), "hdmi_version"), test.chip))

			outcome, err := applier.Apply(context.Background(), blob, "upload")
			if err != nil {
				t.Fatalf("apply: %v", err)
			}

			if outcome.State != test.state {
				t.Fatalf("state %q, want %q (%s)", outcome.State, test.state, outcome.Message)
			}
			if flasher.calls != 0 {
				t.Fatal("the tool was spawned for a blob or a board that was already refused")
			}
			if capture.disabled != 0 {
				t.Fatal("capture was stopped for an apply that never ran")
			}
			if outcome.RequiresPowerCycle {
				t.Fatal("nothing was written, so nothing needs a power cycle")
			}
		})
	}
}

// The tool's reset on pcie is a pair of system() calls whose return values it
// ignores, so an unexported gpio451 makes it silently not happen.
func TestApplyRefusesPcieWithoutTheResetGPIO(t *testing.T) {
	flasher := &fakeFlasher{results: []Result{verifiedRun()}}
	applier, _ := useApplier(t, flasher)

	dir := t.TempDir()
	swapString(t, &productFile, writeTemp(t, filepath.Join(dir, "hw"), "pcie\n"))
	swapString(t, &resetGPIOFile, filepath.Join(dir, "absent"))

	outcome, err := applier.Apply(context.Background(), fixture(t), "upload")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if outcome.State != proto.EdidStatePreflight {
		t.Fatalf("state %q, want preflight", outcome.State)
	}
	if flasher.calls != 0 {
		t.Fatal("the tool was spawned with no reset gpio exported")
	}
}

func TestApplyRefusesWhileTheLockfileIsHeld(t *testing.T) {
	flasher := &fakeFlasher{results: []Result{verifiedRun()}}
	applier, _ := useApplier(t, flasher)

	unlock, err := applier.store.Lock()
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	defer unlock()

	// The lockfile names this process, so it is not stale and must not break.
	if _, err := applier.Apply(context.Background(), fixture(t), "upload"); !errors.Is(err, ErrLocked) {
		t.Fatalf("apply error %v, want %v", err, ErrLocked)
	}
	if flasher.calls != 0 {
		t.Fatal("two flashers reached the chip at once")
	}
}

// The spawn is behind the Flasher interface everywhere else, but the stdin
// handshake is the one part of it worth proving: a bare fgets on a pipe, one
// "y\n", then EOF.
func TestToolFlasherFeedsTheConfirmationAndClosesStdin(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no shell available")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "fake-tool")
	body := "#!/bin/sh\n" +
		"read answer\n" +
		"echo \"arg=$1 answer=$answer\"\n" +
		"if read extra; then echo \"stdin stayed open\" >&2; exit 1; fi\n" +
		"echo 'EDID data verified successfully'\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake tool: %v", err)
	}

	result := toolFlasher{path: script}.Flash(context.Background(), "/tmp/edid.bin")

	if result.ExitCode != 0 {
		t.Fatalf("exit %d, stderr %q", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "arg=/tmp/edid.bin answer=y") {
		t.Fatalf("stdout %q, want the path argument and the confirmation", result.Stdout)
	}
	if Classify(result).State != proto.EdidStateSuccess {
		t.Fatal("a verified run did not classify as success")
	}
}
