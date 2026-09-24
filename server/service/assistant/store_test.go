package assistant

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContexts(t *testing.T) {
	var c Contexts
	if n, _ := c.Add(Image{Mime: "image/jpeg", B64: "YQ=="}); n != 1 {
		t.Fatalf("n=%d", n)
	}
	snap := c.Snapshot()
	c.Add(Image{Mime: "image/jpeg", B64: "Yg=="})
	if len(snap) != 1 || c.Count() != 2 {
		t.Fatalf("snapshot aliased or count wrong")
	}
	c.Clear()
	if c.Count() != 0 {
		t.Fatal("not cleared")
	}
	if _, err := c.Add(Image{B64: strings.Repeat("a", maxContextBytes+1)}); !errors.Is(err, ErrContextsFull) {
		t.Fatalf("err=%v", err)
	}
}

func TestAttachmentStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "att")
	s := NewAttachmentStore(dir)
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("missing dir: %v %v", list, err)
	}
	for _, bad := range []string{"", ".", "..", "../x", "a/b", ".hidden", "index.json"} {
		if err := s.Put(bad, strings.NewReader("x")); !errors.Is(err, ErrAttachmentName) {
			t.Fatalf("name %q: err=%v", bad, err)
		}
	}
	for _, bad := range []string{"", "..", "../x", "a/b", "a\\b", "/etc/passwd", "x\x00y"} {
		if err := s.Delete(bad); !errors.Is(err, ErrAttachmentName) {
			t.Fatalf("delete %q: err=%v", bad, err)
		}
	}
	for _, bad := range []string{"a\\b", "/etc/passwd", "x\x00y"} {
		if err := s.Put(bad, strings.NewReader("x")); !errors.Is(err, ErrAttachmentName) {
			t.Fatalf("name %q: err=%v", bad, err)
		}
	}
	s.Put("b.txt", strings.NewReader("\uFEFFhello"))
	s.Put("a.pdf", strings.NewReader("%PDF"))
	s.Put("bin.txt", bytes.NewReader([]byte{'a', 0, 'b'}))
	list, _ := s.List()
	if len(list) != 3 || list[0].Name != "a.pdf" || list[1].Name != "b.txt" {
		t.Fatalf("list %+v", list)
	}
	loaded := s.Load()
	// The NUL-containing text file is skipped, the BOM is stripped (TextDecoder).
	if len(loaded) != 2 || loaded[0].Kind != "pdf" || loaded[1].Text != "hello" || loaded[1].B64 == "" {
		t.Fatalf("loaded %+v", loaded)
	}
	info, _ := os.Stat(filepath.Join(dir, "b.txt"))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
	if err := s.Put("big.bin", bytes.NewReader(make([]byte, maxAttachmentBytes))); !errors.Is(err, ErrAttachmentsFull) {
		t.Fatalf("cap err=%v", err)
	}
	if err := s.Delete("b.txt"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.List(); len(list) != 2 {
		t.Fatalf("after delete %+v", list)
	}
}
