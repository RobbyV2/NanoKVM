package audit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"NanoKVM-Server/middleware"

	log "github.com/sirupsen/logrus"
)

// One JSON object per line, and never more than two files of maxFileBytes: the
// trail is bounded before it is complete, because filling the card costs more
// than losing the oldest records.
const (
	FilePath = "/etc/kvm/audit.jsonl"

	maxFileBytes = 128 << 10
	maxFieldLen  = 200
)

var (
	mu sync.Mutex
	// path is a var so tests can point the trail at t.TempDir().
	path = FilePath
)

type entry struct {
	Time            string `json:"time"`
	Actor           string `json:"actor,omitempty"`
	Unauthenticated bool   `json:"unauthenticated,omitempty"`
	Action          string `json:"action"`
	Target          string `json:"target,omitempty"`
	OK              bool   `json:"ok"`
	Error           string `json:"error,omitempty"`
}

// Record appends one line for a gadget- or network-mutating operation. Nothing
// here identifies the client beyond the account it authenticated as: no
// address, no request, no payload.
func Record(principal middleware.Principal, action string, target string, err error) {
	record := entry{
		Time:   time.Now().UTC().Format(time.RFC3339),
		Action: action,
		Target: truncate(target),
		OK:     err == nil,
	}
	if principal.Username != "" && !principal.Unauthenticated {
		record.Actor = truncate(principal.Username)
	} else {
		record.Unauthenticated = true
	}
	if err != nil {
		record.Error = truncate(err.Error())
	}
	if writeErr := write(record); writeErr != nil {
		log.Errorf("audit: record %s failed: %s", action, writeErr)
	}
}

func write(record entry) error {
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if info, statErr := os.Stat(path); statErr == nil && info.Size()+int64(len(line)) > maxFileBytes {
		if err := os.Rename(path, path+".1"); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, err = file.Write(line)
	return errors.Join(err, file.Close())
}

func truncate(value string) string {
	if len(value) <= maxFieldLen {
		return value
	}
	return value[:maxFieldLen]
}
