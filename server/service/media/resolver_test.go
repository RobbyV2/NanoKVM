package media

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeIdentity(t *testing.T, root, class, node, identity string) {
	t.Helper()
	dir := filepath.Join(root, "class", class, node)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "function_name"), []byte(identity+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolverUsesFunctionIdentityNotMinorOrder(t *testing.T) {
	root := t.TempDir()
	writeIdentity(t, root, "video4linux", "video2", "uvc.cam1")
	writeIdentity(t, root, "video4linux", "video9", "uvc.cam0")
	writeIdentity(t, root, "sound", "card7", "uac2.mic0")
	resolver := NewSysfsResolver(root, "/nodes")

	video, err := resolver.ResolveVideo("uvc.cam0")
	if err != nil || video != "/nodes/video9" {
		t.Fatalf("video = %q %v", video, err)
	}
	audio, err := resolver.ResolveAudio("uac2.mic0")
	if err != nil || audio != "hw:7,0" {
		t.Fatalf("audio = %q %v", audio, err)
	}
}

func TestResolverFailsClosedOnAmbiguousIdentity(t *testing.T) {
	root := t.TempDir()
	writeIdentity(t, root, "video4linux", "video2", "uvc.cam0")
	writeIdentity(t, root, "video4linux", "video9", "uvc.cam0")
	resolver := NewSysfsResolver(root, "")
	if _, err := resolver.ResolveVideo("uvc.cam0"); !errors.Is(err, ErrNodeIdentityAmbiguous) {
		t.Fatalf("err = %v, want ErrNodeIdentityAmbiguous", err)
	}
}

func TestResolverRequiresUAC2KernelIdentityAttribute(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "class/sound/card0"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolver := NewSysfsResolver(root, "")
	if _, err := resolver.ResolveAudio("uac2.mic0"); !errors.Is(err, ErrAudioIdentityUnavailable) {
		t.Fatalf("err = %v, want ErrAudioIdentityUnavailable", err)
	}
}

func TestResolverRequiresExactIdentity(t *testing.T) {
	root := t.TempDir()
	writeIdentity(t, root, "sound", "card2", " uac2.mic0 ")
	resolver := NewSysfsResolver(root, "")
	if _, err := resolver.ResolveAudio("uac2.mic0"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("err = %v, want exact identity mismatch", err)
	}
}
