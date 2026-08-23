package sources

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStoreRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "sources.json")
	store := NewStore(path)
	if err := store.Save(testSlots); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	validated, _ := validateSlots(testSlots)
	if !reflect.DeepEqual(loaded, validated) {
		t.Fatalf("loaded=%+v want=%+v", loaded, validated)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestStoreMissingIsEmpty(t *testing.T) {
	loaded, err := NewStore(filepath.Join(t.TempDir(), "missing.json")).Load()
	if err != nil || len(loaded) != 0 {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}
}

func TestStoreRejectsUnknownAndOversizedContent(t *testing.T) {
	for name, content := range map[string]string{
		"unknown":   `{"schema_version":1,"slots":[],"extra":true}`,
		"trailing":  `{"schema_version":1,"slots":[]} {}`,
		"oversized": strings.Repeat(" ", maxConfigBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sources.json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewStore(path).Load(); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestSlotsRequireKindMatchedContiguousIDs(t *testing.T) {
	for _, slots := range [][]Slot{
		{{ID: "uvc.cam0", Kind: KindMicrophone, Label: "Mic"}},
		{{ID: "uvc.cam2", Kind: KindCamera, Label: "Camera"}},
		{{ID: "uvc.cam0", Kind: KindCamera, Label: "Camera"}, {ID: "uvc.cam0", Kind: KindCamera, Label: "Other"}},
	} {
		if _, err := validateSlots(slots); err == nil {
			t.Fatalf("invalid slots accepted: %+v", slots)
		}
	}
}
