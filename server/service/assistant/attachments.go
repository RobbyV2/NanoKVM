package assistant

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

// Files uploaded in settings, sent with every ask exactly as the extension
// sends its bundled attachments/ folder.
const (
	AttachmentsDir     = "/etc/kvm/assistant/attachments"
	maxAttachmentBytes = 20 << 20
)

var (
	ErrAttachmentName  = errors.New("invalid attachment name")
	ErrAttachmentsFull = errors.New("attachments exceed 20 MB")
	skippedNames       = map[string]bool{".gitignore": true, "index.json": true, ".DS_Store": true}
)

type AttachmentInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type AttachmentStore struct {
	dir string
	mu  sync.Mutex
}

func NewAttachmentStore(dir string) *AttachmentStore {
	return &AttachmentStore{dir: dir}
}

func validAttachmentName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.HasPrefix(name, ".") &&
		!skippedNames[name] && !strings.ContainsAny(name, "/\\\x00")
}

func (s *AttachmentStore) List() ([]AttachmentInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *AttachmentStore) list() ([]AttachmentInfo, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return []AttachmentInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []AttachmentInfo{}
	for _, e := range entries {
		if !validAttachmentName(e.Name()) || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, AttachmentInfo{Name: e.Name(), Size: info.Size()})
	}
	return out, nil
}

func (s *AttachmentStore) Put(name string, r io.Reader) error {
	if !validAttachmentName(name) {
		return ErrAttachmentName
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.list()
	if err != nil {
		return err
	}
	var used int64
	for _, a := range existing {
		if a.Name != name {
			used += a.Size
		}
	}
	data, err := io.ReadAll(io.LimitReader(r, maxAttachmentBytes-used+1))
	if err != nil {
		return err
	}
	if used+int64(len(data)) > maxAttachmentBytes {
		return ErrAttachmentsFull
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return utils.WriteFileAtomic(filepath.Join(s.dir, name), data, 0o600)
}

func (s *AttachmentStore) Delete(name string) error {
	if !validAttachmentName(name) {
		return ErrAttachmentName
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(filepath.Join(s.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Load mirrors content.js loadAttachments: classify by extension, base64
// everything, decode text files and skip ones that look binary.
func (s *AttachmentStore) Load() []Attachment {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos, err := s.list()
	if err != nil {
		log.Warnf("assistant: list attachments: %v", err)
		return nil
	}
	var out []Attachment
	for _, info := range infos {
		data, err := os.ReadFile(filepath.Join(s.dir, info.Name))
		if err != nil {
			log.Warnf("assistant: attachment load failed: %s: %v", info.Name, err)
			continue
		}
		ft := classifyFile(info.Name)
		a := Attachment{Name: info.Name, Kind: ft.Kind, Mime: ft.Mime, AudioFormat: ft.AudioFormat, B64: base64.StdEncoding.EncodeToString(data)}
		if ft.Kind == "text" {
			if bytes.IndexByte(data, 0) >= 0 {
				log.Warnf("assistant: skipping %s: looks binary, not text", info.Name)
				continue
			}
			a.Text = strings.TrimPrefix(strings.ToValidUTF8(string(data), "\uFFFD"), "\uFEFF")
		}
		out = append(out, a)
	}
	return out
}
