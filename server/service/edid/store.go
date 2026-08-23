package edid

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

// last-applied.bin, last-applied.json, history/<ts>-<sha8>.bin and .lock, all
// 0600. Nothing is ever removed from history/: on hardware with no read
// primitive an archived file is the only rollback target that exists.
const StoreDir = "/etc/kvm/edid"

const (
	activeBinName  = "last-applied.bin"
	activeJSONName = "last-applied.json"
	pendingName    = "pending.json"
	historyDirName = "history"
	lockName       = ".lock"

	// A new one of these is exactly what the operator is being asked to
	// produce, and nothing short of a reboot produces one.
	bootIDPath = "/proc/sys/kernel/random/boot_id"

	dirMode  os.FileMode = 0o755
	fileMode os.FileMode = 0o600

	backupTimeLayout = "20060102T150405Z"
)

// Never escalated into a flash outcome, because nothing was spawned.
var ErrLocked = errors.New("edid: another apply is in progress")

var ErrBackupNotFound = errors.New("edid: backup not found")

// <ts>-<sha8>, and the traversal guard: an id that does not match never reaches
// filepath.Join.
var backupPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{8}$`)

var (
	// storeDir is a var so tests can point the store at t.TempDir().
	storeMu  sync.Mutex
	storeDir = StoreDir

	processAlive = defaultProcessAlive
	bootIDFile   = bootIDPath
)

// last-applied.json. The decoded fields keep the file legible from a serial
// console and let History list an entry without re-decoding it.
type Record struct {
	SHA256        string    `json:"sha256"`
	Source        string    `json:"source"`
	AppliedAt     time.Time `json:"appliedAt"`
	Manufacturer  string    `json:"manufacturer"`
	Model         string    `json:"model"`
	PreferredMode string    `json:"preferredMode"`
	Extensions    uint8     `json:"extensions"`
	Audio         bool      `json:"audio"`
}

// pending.json. On alpha and beta the chip reloads its EDID region only out of
// reset, so between a flash and the power cycle the device presents the old
// EDID and the record says otherwise. The marker is what the operator sees
// after a page reload or a service restart, both of which the apply response
// alone does not survive.
//
// Boot is the clearing signal because the tool has no read mode: nothing on
// this device can ask the chip what it holds. A power cycle necessarily gives
// the SoC a new boot_id, so the marker cannot outlive the event it waits for,
// and a service restart, which does not reset the chip, keeps the same one and
// leaves the marker armed. A warm reboot clears it a power cycle early; that is
// the price of the only observable that moves when the chip does, and it beats
// a notice nothing can ever retire. A board where boot_id cannot be read keeps
// the marker until the next apply, which is the safe direction to be wrong in.
type Pending struct {
	SHA256    string          `json:"sha256"`
	Source    string          `json:"source"`
	State     proto.EdidState `json:"state"`
	AppliedAt time.Time       `json:"appliedAt"`
	Boot      string          `json:"boot"`
}

type Backup struct {
	ID        string
	SHA256    string
	AppliedAt time.Time
	Size      int
}

type Store struct {
	dir string
}

func NewStore() *Store {
	return &Store{dir: storeDir}
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) activePath() string {
	return filepath.Join(s.dir, activeBinName)
}

func (s *Store) recordPath() string {
	return filepath.Join(s.dir, activeJSONName)
}

func (s *Store) pendingPath() string {
	return filepath.Join(s.dir, pendingName)
}

func (s *Store) historyDir() string {
	return filepath.Join(s.dir, historyDirName)
}

// Absent bytes are not an error: a device NanoKVM has never flashed reports
// unverified rather than failing.
func (s *Store) LoadActive() ([]byte, *Record, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	data, err := os.ReadFile(s.activePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", s.activePath(), err)
	}

	record, err := s.loadRecord()
	if err != nil {
		return data, nil, err
	}
	return data, record, nil
}

// A missing or unparseable record is nil with no error: the bytes are the
// source of truth and the record only describes them.
func (s *Store) loadRecord() (*Record, error) {
	raw, err := os.ReadFile(s.recordPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", s.recordPath(), err)
	}

	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, nil
	}
	return &record, nil
}

// Callers must not reach here on anything but a verified write: the archive
// exists to name bytes the chip accepted.
func (s *Store) Archive(data []byte, source string, decoded *EDID) (Record, error) {
	if len(data) != Size {
		return Record{}, fmt.Errorf("edid: archive %d bytes, want %d", len(data), Size)
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	if err := os.MkdirAll(s.historyDir(), dirMode); err != nil {
		return Record{}, fmt.Errorf("create %s: %w", s.historyDir(), err)
	}

	if err := s.archivePrevious(); err != nil {
		return Record{}, err
	}

	record := newRecord(data, source, decoded, time.Now().UTC())
	if err := utils.WriteFileAtomic(s.activePath(), data, fileMode); err != nil {
		return Record{}, fmt.Errorf("write %s: %w", s.activePath(), err)
	}

	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Record{}, fmt.Errorf("encode edid record: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := utils.WriteFileAtomic(s.recordPath(), encoded, fileMode); err != nil {
		return Record{}, fmt.Errorf("write %s: %w", s.recordPath(), err)
	}
	return record, nil
}

// Copies rather than moves, so the active file stays readable for the whole
// operation. An existing entry is left alone, so re-flashing identical bytes
// inside one second does not rewrite history.
func (s *Store) archivePrevious() error {
	previous, err := os.ReadFile(s.activePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", s.activePath(), err)
	}
	if len(previous) == 0 {
		return nil
	}

	id := backupIDFor(previous, s.previousTimestamp())
	path := filepath.Join(s.historyDir(), id+".bin")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := utils.WriteFileAtomic(path, previous, fileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Dates the outgoing entry by its own record, then mtime, then now, so a
// history id always carries a time that means something.
func (s *Store) previousTimestamp() time.Time {
	if record, err := s.loadRecord(); err == nil && record != nil && !record.AppliedAt.IsZero() {
		return record.AppliedAt.UTC()
	}
	if info, err := os.Stat(s.activePath()); err == nil {
		return info.ModTime().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) ArmPending(pending Pending) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", s.dir, err)
	}

	pending.AppliedAt = pending.AppliedAt.UTC()
	pending.Boot = bootID()

	encoded, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return fmt.Errorf("encode edid pending: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := utils.WriteFileAtomic(s.pendingPath(), encoded, fileMode); err != nil {
		return fmt.Errorf("write %s: %w", s.pendingPath(), err)
	}
	return nil
}

// Clears the marker on the first read from a different boot, so the power cycle
// retires it whether or not anyone was watching when it happened.
func (s *Store) Pending() (*Pending, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	raw, err := os.ReadFile(s.pendingPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", s.pendingPath(), err)
	}

	var pending Pending
	if err := json.Unmarshal(raw, &pending); err != nil {
		return nil, s.clearPending()
	}
	if pending.Boot != bootID() {
		return nil, s.clearPending()
	}
	return &pending, nil
}

func (s *Store) clearPending() error {
	if err := os.Remove(s.pendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", s.pendingPath(), err)
	}
	return nil
}

func bootID() string {
	data, err := os.ReadFile(bootIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Newest first. The id format sorts lexicographically in timestamp order.
func (s *Store) History() ([]Backup, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	entries, err := os.ReadDir(s.historyDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", s.historyDir(), err)
	}

	backups := make([]Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".bin")
		if id == entry.Name() || !backupPattern.MatchString(id) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.historyDir(), entry.Name()))
		if err != nil {
			continue
		}
		stamp, err := time.Parse(backupTimeLayout, id[:strings.IndexByte(id, '-')])
		if err != nil {
			stamp = time.Time{}
		}
		backups = append(backups, Backup{
			ID:        id,
			SHA256:    digest(data),
			AppliedAt: stamp.UTC(),
			Size:      len(data),
		})
	}

	sort.Slice(backups, func(i, j int) bool { return backups[i].ID > backups[j].ID })
	return backups, nil
}

// The id is matched against backupPattern before it is joined onto a path, so a
// caller cannot walk out of the history directory.
func (s *Store) ReadBackup(id string) ([]byte, error) {
	if !backupPattern.MatchString(id) {
		return nil, fmt.Errorf("%w: %q", ErrBackupNotFound, id)
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.historyDir(), id+".bin"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrBackupNotFound, id)
		}
		return nil, fmt.Errorf("read backup %s: %w", id, err)
	}
	return data, nil
}

// Two interleaved program sequences corrupt the EDID region outright, and the
// in-process mutex dies with the process, so the guard is on disk. A lockfile
// whose pid no longer exists is broken exactly once.
func (s *Store) Lock() (func(), error) {
	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return nil, fmt.Errorf("create %s: %w", s.dir, err)
	}

	path := filepath.Join(s.dir, lockName)
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
		if err == nil {
			_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
			if err := file.Close(); err != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("write %s: %w", path, err)
			}
			var once sync.Once
			return func() { once.Do(func() { _ = os.Remove(path) }) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create %s: %w", path, err)
		}
		if !breakStaleLock(path) {
			return nil, ErrLocked
		}
	}
	return nil, ErrLocked
}

// An unreadable or unparseable lockfile is stale: nothing can ever release it.
func breakStaleLock(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return errors.Is(err, os.ErrNotExist)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err == nil && pid > 0 && processAlive(pid) {
		return false
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// /proc/<pid>, falling back to signal 0 where /proc is not mounted.
func defaultProcessAlive(pid int) bool {
	if _, err := os.Stat("/proc"); err == nil {
		_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
		return err == nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func newRecord(data []byte, source string, decoded *EDID, at time.Time) Record {
	record := Record{
		SHA256:    digest(data),
		Source:    source,
		AppliedAt: at.UTC(),
	}
	if decoded == nil {
		return record
	}

	record.Manufacturer = decoded.Manufacturer
	record.Model = decoded.Name()
	record.Extensions = decoded.Extensions
	record.Audio = HasAudio(decoded)
	if timing := decoded.PreferredTiming(); timing != nil {
		record.PreferredMode = timing.Mode()
	}
	return record
}

// The CTA basic audio flag, or a short audio descriptor block.
func HasAudio(decoded *EDID) bool {
	if decoded == nil || decoded.CTA == nil {
		return false
	}
	if decoded.CTA.BasicAudio {
		return true
	}
	for _, block := range decoded.CTA.Blocks {
		if block.Tag == CTAAudio {
			return true
		}
	}
	return false
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func backupIDFor(data []byte, at time.Time) string {
	return at.UTC().Format(backupTimeLayout) + "-" + digest(data)[:8]
}
