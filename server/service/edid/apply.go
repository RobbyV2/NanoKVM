package edid

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

const (
	ToolPath    = "/kvmapp/system/tool/nanokvm_update_edid"
	FactoryPath = "/kvmapp/system/tool/E21_NanoKVM.bin"

	chipVersionPath = "/etc/kvm/hdmi_version"
	productPath     = "/etc/kvm/hw"

	// The tool resets this twice on pcie through system() calls it never checks,
	// and the export happens in kvm_vision.cpp, so a missing node is a no-op.
	pcieResetGPIO = "/sys/class/gpio/gpio451/value"

	// The whole stdin the child ever gets. The reader must then hit EOF: a
	// stream that stays open with a wrong first character makes the tool loop
	// on "Invalid input" for the whole timeout.
	confirmation = "y\n"

	applyTimeout = 90 * time.Second

	// sizeof(chip_version_str) in the tool, which compares the line truncated.
	fgetsBuffer = 32
)

// Chip is the value /etc/kvm/hdmi_version selects.
type Chip string

const (
	ChipLT6911C   Chip = "LT6911C"
	ChipLT6911UXC Chip = "LT6911UXC"
	ChipLT6911D   Chip = "LT6911D"
)

// Product is the value /etc/kvm/hw selects.
type Product string

const (
	ProductCubeA Product = "CUBE_A"
	ProductCubeB Product = "CUBE_B"
	ProductPCIeA Product = "PCIE_A"
)

// Success is exit 0 *and* the verification line: the tool exits 0 on paths
// that never compared the version page.
const (
	verifiedMarker = "EDID data verified successfully"
	writtenMarker  = "Written EDID data:"
	readMarker     = "Read EDID data:"
)

// Order is significant: the mismatch row must win over anything else in a dump
// that happens to contain a matching word.
type stderrRow struct {
	marker    string
	state     proto.EdidState
	retryable bool
}

var stderrRows = []stderrRow{
	{"EDID data mismatch after write/read cycle", proto.EdidStateNeedsRecovery, false},

	{"Unsupported chip version", proto.EdidStateChipRefused, false},
	{"Clean Error", proto.EdidStateChipRefused, false},
	{"Failed to read LT6911D version data", proto.EdidStateChipRefused, false},

	{"Failed to open the i2c bus", proto.EdidStateBusContention, true},
	{"Failed to acquire bus access", proto.EdidStateBusContention, true},

	{"Chip Version Error:", proto.EdidStatePreflight, false},
	{"Product Version Error:", proto.EdidStatePreflight, false},
	{"Please upgrade to the latest system", proto.EdidStatePreflight, false},
	{"Failed to read chip version", proto.EdidStatePreflight, false},
	{"Failed to read product version", proto.EdidStatePreflight, false},

	{"EDID data is invalid", proto.EdidStateInvalidInput, false},
	{"EDID data length is not", proto.EdidStateInvalidInput, false},
	{"EDID header is invalid", proto.EdidStateInvalidInput, false},
	{"Checksum for", proto.EdidStateInvalidInput, false},
}

var (
	// applyMu covers this process, the store's lockfile covers the rest.
	applyMu sync.Mutex

	// Every external dependency is a var so the package tests without a device.
	toolPath        = ToolPath
	factoryPath     = FactoryPath
	chipVersionFile = chipVersionPath
	productFile     = productPath
	resetGPIOFile   = pcieResetGPIO
)

// The capture daemon polls the same LT6911 at 0x2b on /dev/i2c-4, and a
// detection read landing between an erase and its status poll corrupts the
// program sequence. Injected rather than imported from service/vm, which pulls
// in the cgo capture library this package tests without.
type CaptureGuard struct {
	Disable func()
	Enable  func()
}

// Capture is not restarted when the operator had persistently disabled HDMI,
// since that would silently undo their setting.
func (g CaptureGuard) hold() func() {
	if g.Disable != nil {
		g.Disable()
	}

	return func() {
		if g.Enable == nil || utils.IsHdmiDisabled() {
			return
		}
		g.Enable()
	}
}

// Deliberately not config.GetHwVersion: that helper defaults unknown /etc/kvm/hw
// content to alpha, where the tool fails, so it would flash an unidentified board.
type PreflightResult struct {
	Chip               Chip
	Product            Product
	ChipRaw            string
	ProductRaw         string
	Supported          bool
	RequiresPowerCycle bool
	ToolAvailable      bool
	Reason             string
}

// Mirrors the tool's strcmp table verbatim, including a CRLF or trailing space
// yielding "Unknown version".
func Preflight() PreflightResult {
	result := PreflightResult{ToolAvailable: toolExists()}

	chipRaw, ok, reason := readVersionFile(chipVersionFile, "Failed to read chip version")
	if !ok {
		result.Reason = reason
		return result
	}
	result.ChipRaw = chipRaw

	productRaw, ok, reason := readVersionFile(productFile, "Failed to read product version")
	if !ok {
		result.Reason = reason
		return result
	}
	result.ProductRaw = productRaw

	switch chipRaw {
	case "c":
		result.Chip = ChipLT6911C
	case "ux":
		result.Chip = ChipLT6911UXC
	case "d":
		result.Chip = ChipLT6911D
	case "ue":
		result.Reason = "Chip Version Error: UE version's edid can't be updated"
		return result
	default:
		result.Reason = fmt.Sprintf("Chip Version Error: Unknown version %q", chipRaw)
		return result
	}

	switch productRaw {
	case "alpha":
		result.Product, result.RequiresPowerCycle = ProductCubeA, true
	case "beta":
		result.Product, result.RequiresPowerCycle = ProductCubeB, true
	case "pcie":
		result.Product = ProductPCIeA
	default:
		result.Reason = fmt.Sprintf("Product Version Error: Unknown version %q", productRaw)
		return result
	}

	if !result.ToolAvailable {
		result.Reason = fmt.Sprintf("%s is missing, upgrade to the latest system", toolPath)
		return result
	}

	result.Supported = true
	return result
}

// fopen plus one fgets plus strcspn(s, "\n"). Unopenable and empty are
// different messages in the tool.
func readVersionFile(path string, emptyReason string) (string, bool, string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false, "Please upgrade to the latest system"
	}
	if len(raw) == 0 {
		return "", false, emptyReason
	}
	return firstLine(raw), true, ""
}

func firstLine(raw []byte) string {
	if len(raw) > fgetsBuffer-1 {
		raw = raw[:fgetsBuffer-1]
	}
	if i := bytes.IndexByte(raw, '\n'); i >= 0 {
		raw = raw[:i]
	}
	return string(raw)
}

func toolExists() bool {
	info, err := os.Stat(toolPath)
	return err == nil && !info.IsDir()
}

// Stdout and stderr are separate because the tool splits progress from
// diagnostics.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
	SpawnErr error
}

type Flasher interface {
	Flash(ctx context.Context, blobPath string) Result
}

type toolFlasher struct {
	path string
}

// A closed stdin reads EOF, which the tool treats as a decline, and the prompt
// runs ahead of the argc check, so the path argument is never omitted.
func (f toolFlasher) Flash(ctx context.Context, blobPath string) Result {
	cmd := exec.CommandContext(ctx, f.path, blobPath)
	cmd.Stdin = strings.NewReader(confirmation)

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}

	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
	default:
		result.ExitCode = -1
		result.SpawnErr = err
	}

	if ctx.Err() != nil {
		result.TimedOut = true
	}
	return result
}

type Outcome struct {
	State              proto.EdidState
	Verified           bool
	Retryable          bool
	RequiresPowerCycle bool
	Message            string
	WrittenHex         string
	ReadHex            string
	Record             *Record
}

// From stderr, never from the exit code: every failure path returns 1, so a
// rejected blob and a half-written flash region are the same integer.
func Classify(result Result) Outcome {
	if result.TimedOut {
		return Outcome{
			State:   proto.EdidStateTimeout,
			Message: fmt.Sprintf("the tool did not finish within %s; the flash region is in an unknown state", applyTimeout),
		}
	}
	if result.SpawnErr != nil {
		return Outcome{State: proto.EdidStateGeneric, Message: result.SpawnErr.Error()}
	}

	for _, row := range stderrRows {
		if !strings.Contains(result.Stderr, row.marker) {
			continue
		}

		outcome := Outcome{
			State:     row.state,
			Retryable: row.retryable,
			Message:   matchedLine(result.Stderr, row.marker),
		}
		if row.state == proto.EdidStateNeedsRecovery {
			outcome.WrittenHex, outcome.ReadHex = parseDumps(result.Stderr)
		}
		return outcome
	}

	if result.ExitCode == 0 {
		if strings.Contains(result.Stdout, verifiedMarker) {
			return Outcome{State: proto.EdidStateSuccess, Verified: true, Message: verifiedMarker}
		}
		return Outcome{
			State:   proto.EdidStateUnverified,
			Message: "the tool exited 0 without confirming the readback",
		}
	}

	return Outcome{State: proto.EdidStateGeneric, Message: lastLine(result.Stderr)}
}

// The whole line, so the UI shows the tool's own words.
func matchedLine(stderr, marker string) string {
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, marker) {
			return strings.TrimSpace(line)
		}
	}
	return marker
}

func lastLine(stderr string) string {
	lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return "the tool failed without writing to stderr"
}

// The two dumps a mismatch prints are the only diagnostic that exists on
// hardware with no read primitive, so they reach the API intact.
func parseDumps(stderr string) (written string, read string) {
	return hexSection(stderr, writtenMarker, readMarker), hexSection(stderr, readMarker, "")
}

func hexSection(stderr, start, end string) string {
	i := strings.Index(stderr, start)
	if i < 0 {
		return ""
	}
	section := stderr[i+len(start):]
	if end != "" {
		if j := strings.Index(section, end); j >= 0 {
			section = section[:j]
		}
	}

	fields := strings.Fields(section)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) != 2 {
			continue
		}
		if _, err := hex.DecodeString(field); err != nil {
			continue
		}
		out = append(out, strings.ToUpper(field))
	}
	return strings.Join(out, " ")
}

type Applier struct {
	store   *Store
	flasher Flasher
	capture CaptureGuard
}

func NewApplier(capture CaptureGuard) *Applier {
	return &Applier{store: NewStore(), flasher: toolFlasher{path: toolPath}, capture: capture}
}

// The returned error is for infrastructure failures only. A chip that refused
// is a successful call with a failing Outcome.
func (a *Applier) Apply(ctx context.Context, data []byte, source string) (Outcome, error) {
	blob, err := Normalize(data)
	if err != nil {
		return Outcome{State: proto.EdidStateInvalidInput, Message: err.Error()}, nil
	}

	// Stricter than the tool's check_edid, so nothing structurally nonsensical
	// reaches the flash.
	decoded, err := Decode(blob)
	if err != nil {
		return Outcome{State: proto.EdidStateInvalidInput, Message: err.Error()}, nil
	}

	pre := Preflight()
	if !pre.Supported {
		return Outcome{State: proto.EdidStatePreflight, Message: pre.Reason}, nil
	}
	if err := checkResetGPIO(pre); err != nil {
		return Outcome{State: proto.EdidStatePreflight, Message: err.Error()}, nil
	}

	applyMu.Lock()
	defer applyMu.Unlock()

	unlock, err := a.store.Lock()
	if err != nil {
		return Outcome{}, err
	}
	defer unlock()

	blobPath, cleanup, err := stageBlob(blob)
	if err != nil {
		return Outcome{}, err
	}
	defer cleanup()

	outcome := a.run(ctx, blobPath)

	// Only bus contention retries, where the region is untouched, and only
	// once. run() re-takes the capture guard rather than reusing a stale one.
	if outcome.State == proto.EdidStateBusContention {
		log.Warn("edid: i2c bus contention, retrying once after re-taking the capture guard")
		outcome = a.run(ctx, blobPath)
		outcome.Retryable = false
	}

	outcome.RequiresPowerCycle = pre.RequiresPowerCycle && touchedFlash(outcome.State)
	if outcome.RequiresPowerCycle {
		pending := Pending{SHA256: digest(blob), Source: source, State: outcome.State, AppliedAt: time.Now().UTC()}
		if err := a.store.ArmPending(pending); err != nil {
			log.Errorf("edid: recording the pending power cycle failed: %s", err)
		}
	}

	if outcome.Verified {
		record, err := a.store.Archive(blob, source, decoded)
		if err != nil {
			log.Errorf("edid: flash verified but the archive failed: %s", err)
			outcome.Message += fmt.Sprintf(" (the flash verified, but recording it failed: %s)", err)
		} else {
			outcome.Record = &record
		}
	}

	log.Infof("edid: apply from %s finished as %s on %s/%s", source, outcome.State, pre.Chip, pre.Product)
	return outcome, nil
}

// Holds the capture guard for the whole child lifetime, readback included, and
// releases it on every path including a timeout kill.
func (a *Applier) run(ctx context.Context, blobPath string) Outcome {
	defer a.capture.hold()()

	child, cancel := context.WithTimeout(ctx, applyTimeout)
	defer cancel()

	return Classify(a.flasher.Flash(child, blobPath))
}

// Whether the chip may have been written. The LT6911 reloads its EDID region
// only out of reset, which is what makes the power cycle necessary.
func touchedFlash(state proto.EdidState) bool {
	switch state {
	case proto.EdidStateSuccess, proto.EdidStateUnverified, proto.EdidStateNeedsRecovery,
		proto.EdidStateTimeout, proto.EdidStateGeneric:
		return true
	default:
		return false
	}
}

// A missing gpio451 export turns the tool's reset into a silent no-op, leaving
// the chip programmed without ever having been reset.
func checkResetGPIO(pre PreflightResult) error {
	if pre.Product != ProductPCIeA {
		return nil
	}
	if _, err := os.Stat(resetGPIOFile); err != nil {
		return fmt.Errorf("%s is not exported, so the tool's HDMI reset would silently do nothing", resetGPIOFile)
	}
	return nil
}

// The tool takes a path and silently truncates over 256 bytes, so what it opens
// must be exactly what was validated.
func stageBlob(blob []byte) (string, func(), error) {
	dir, err := os.MkdirTemp("", "nanokvm-edid-")
	if err != nil {
		return "", nil, fmt.Errorf("create staging directory: %w", err)
	}

	path := filepath.Join(dir, "edid.bin")
	if err := utils.WriteFileAtomic(path, blob, fileMode); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("stage edid blob: %w", err)
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}

// The shipped E21_NanoKVM.bin, the restore target of last resort.
func FactoryImage() ([]byte, error) {
	data, err := os.ReadFile(factoryPath)
	if err != nil {
		return nil, fmt.Errorf("read factory edid %s: %w", factoryPath, err)
	}
	return data, nil
}

func FactoryAvailable() bool {
	info, err := os.Stat(factoryPath)
	return err == nil && !info.IsDir()
}
