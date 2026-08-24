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

// A faithful copy of the shapes on the board: /sys/class/udc/<udc> and
// /sys/class/video4linux/videoN are symlinks into /sys/devices, and the uvc
// gadget parents its video device on the gadget, a sibling of the udc
// directory under the same platform device.
func writeGadgetNode(t *testing.T, root, node, name string) {
	t.Helper()
	device := filepath.Join(root, "devices/platform/4340000.usb/gadget/video4linux", node)
	if err := os.MkdirAll(device, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(device, "name"), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkClass(t, root, "video4linux", node, device)
}

func writeCaptureNode(t *testing.T, root, node, name string) {
	t.Helper()
	device := filepath.Join(root, "devices/platform/vi/video4linux", node)
	if err := os.MkdirAll(device, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(device, "name"), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkClass(t, root, "video4linux", node, device)
}

func writeUDC(t *testing.T, root, udc string) {
	t.Helper()
	device := filepath.Join(root, "devices/platform", udc, "udc", udc)
	if err := os.MkdirAll(device, 0o755); err != nil {
		t.Fatal(err)
	}
	linkClass(t, root, "udc", udc, device)
}

func linkClass(t *testing.T, root, class, node, device string) {
	t.Helper()
	dir := filepath.Join(root, "class", class)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(device, filepath.Join(dir, node)); err != nil {
		t.Fatal(err)
	}
}

// This kernel publishes no function_name, so the node has to be identified by
// the controller it belongs to. Only one camera is linked, so the answer is
// unambiguous without inferring anything from minor order.
func TestResolverFindsTheGadgetNodeWithoutFunctionName(t *testing.T) {
	root := t.TempDir()
	writeUDC(t, root, "4340000.usb")
	writeGadgetNode(t, root, "video0", "4340000.usb")
	writeCaptureNode(t, root, "video1", "cvi-vi")
	resolver := NewSysfsResolver(root, "/dev")

	video, err := resolver.ResolveVideo("uvc.cam0")
	if err != nil || video != "/dev/video0" {
		t.Fatalf("video = %q %v, want /dev/video0", video, err)
	}
	nodes, err := resolver.GadgetVideoNodes()
	if err != nil || len(nodes) != 1 || nodes[0] != "/dev/video0" {
		t.Fatalf("gadget nodes = %v %v, want [/dev/video0]", nodes, err)
	}
}

// The device tree is the identity when the name is not: a gadget node named
// anything at all still hangs off the same platform device as the UDC.
func TestResolverFindsTheGadgetNodeByItsController(t *testing.T) {
	root := t.TempDir()
	writeUDC(t, root, "4340000.usb")
	writeGadgetNode(t, root, "video0", "uvc-gadget")
	writeCaptureNode(t, root, "video1", "cvi-vi")
	resolver := NewSysfsResolver(root, "/dev")

	video, err := resolver.ResolveVideo("uvc.cam0")
	if err != nil || video != "/dev/video0" {
		t.Fatalf("video = %q %v, want /dev/video0", video, err)
	}
}

// Two cameras and no function_name is a case this kernel gives no honest answer
// to. The slots fail closed; the nodes are still listed so both can be held and
// the gadget stays on the bus.
func TestResolverRefusesToGuessBetweenTwoGadgetNodes(t *testing.T) {
	root := t.TempDir()
	writeUDC(t, root, "4340000.usb")
	writeGadgetNode(t, root, "video0", "4340000.usb")
	writeGadgetNode(t, root, "video1", "4340000.usb")
	resolver := NewSysfsResolver(root, "/dev")

	if _, err := resolver.ResolveVideo("uvc.cam0"); !errors.Is(err, ErrNodeIdentityAmbiguous) {
		t.Fatalf("err = %v, want ErrNodeIdentityAmbiguous", err)
	}
	nodes, err := resolver.GadgetVideoNodes()
	if err != nil || len(nodes) != 2 {
		t.Fatalf("gadget nodes = %v %v, want both nodes listed so both can be held", nodes, err)
	}
}

// A kernel that does publish function_name keeps the strict answer: the
// fallback must not soften an identity the kernel is willing to state.
func TestResolverPrefersFunctionNameWhereTheKernelHasIt(t *testing.T) {
	root := t.TempDir()
	writeUDC(t, root, "4340000.usb")
	writeIdentity(t, root, "video4linux", "video0", "uvc.cam0")
	writeIdentity(t, root, "video4linux", "video1", "uvc.cam1")
	resolver := NewSysfsResolver(root, "/dev")

	video, err := resolver.ResolveVideo("uvc.cam1")
	if err != nil || video != "/dev/video1" {
		t.Fatalf("video = %q %v, want /dev/video1", video, err)
	}
	if _, err := resolver.ResolveVideo("uvc.cam2"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("err = %v, want ErrNodeNotFound", err)
	}
	nodes, err := resolver.GadgetVideoNodes()
	if err != nil || len(nodes) != 2 {
		t.Fatalf("gadget nodes = %v %v, want both function_name nodes", nodes, err)
	}
}
