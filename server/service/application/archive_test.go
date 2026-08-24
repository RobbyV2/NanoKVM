package application

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveEntry struct {
	name     string
	typeFlag byte
	data     string
}

func writeTestArchive(t *testing.T, entries []archiveEntry) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "update.tar.gz")
	out, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(out)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Typeflag: entry.typeFlag, Mode: 0o755}
		if entry.typeFlag == tar.TypeReg || entry.typeFlag == tar.TypeRegA {
			header.Size = int64(len(entry.data))
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if entry.data != "" {
			if _, err := tarWriter.Write([]byte(entry.data)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestInspectArchiveRejectsUnsafeEntries(t *testing.T) {
	root := "nanokvm_1.2.3"
	for _, entry := range []archiveEntry{
		{name: "../escape", typeFlag: tar.TypeReg, data: "x"},
		{name: "/escape", typeFlag: tar.TypeReg, data: "x"},
		{name: root + "/link", typeFlag: tar.TypeSymlink},
	} {
		archive := writeTestArchive(t, []archiveEntry{{name: root, typeFlag: tar.TypeDir}, entry})
		if _, err := inspectUpdateArchive(archive, root); err == nil {
			t.Fatalf("unsafe entry %+v was accepted", entry)
		}
	}
}

func TestInspectAndExtractArchive(t *testing.T) {
	root := "nanokvm_1.2.3"
	archive := writeTestArchive(t, []archiveEntry{
		{name: root, typeFlag: tar.TypeDir},
		{name: root + "/version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
		{name: root + "/server/NanoKVM-Server", typeFlag: tar.TypeReg, data: "server"},
		{name: root + "/kvm_system/kvm_system", typeFlag: tar.TypeReg, data: "system"},
		{name: root + "/system/init.d/S95nanokvm", typeFlag: tar.TypeReg, data: "init"},
	})
	info, err := inspectUpdateArchive(archive, root)
	if err != nil {
		t.Fatal(err)
	}
	if info.expandedBytes != 22 { // data totals: 6 + 6 + 6 + 4
		t.Fatalf("unexpected expanded size %d", info.expandedBytes)
	}
	destination := t.TempDir()
	extracted, err := extractUpdateArchive(archive, destination, root)
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := validateExtractedPackage(extracted, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if kernel != nil {
		t.Fatalf("an application package reported a kernel payload %+v", kernel)
	}
}

func extractedKernelPackage(t *testing.T, entries []archiveEntry) (string, error) {
	t.Helper()
	root := "nanokvm_1.2.3"
	all := append([]archiveEntry{
		{name: root, typeFlag: tar.TypeDir},
		{name: root + "/version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
		{name: root + "/kernel", typeFlag: tar.TypeDir},
	}, entries...)
	archive := writeTestArchive(t, all)
	if _, err := inspectUpdateArchive(archive, root); err != nil {
		t.Fatal(err)
	}
	extracted, err := extractUpdateArchive(archive, t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := validateExtractedPackage(extracted, "1.2.3")
	if err != nil {
		return "", err
	}
	return kernel.version, nil
}

func TestKernelSubtreeIsAcceptedOnlyWithFDTMagic(t *testing.T) {
	root := "nanokvm_1.2.3"
	version, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/boot.itb", typeFlag: tar.TypeReg, data: "\xd0\x0d\xfe\xed\x00\x00\x10\x00"},
		{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
	})
	if err != nil {
		t.Fatalf("a valid kernel package was rejected: %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("kernel version = %q, want 1.2.3", version)
	}
}

// A kernel/ subtree whose payload is not a FIT image would be written over the
// trial slot and only discovered by a device that no longer boots.
func TestKernelSubtreeWithABadPayloadIsRejected(t *testing.T) {
	root := "nanokvm_1.2.3"
	for _, payload := range []string{
		"\x1f\x8b\x08\x00\x00\x00\x00\x00",
		"\xed\xfe\x0d\xd0\x00\x00\x10\x00",
		"\xd0\x0d\xfe",
		"",
	} {
		_, err := extractedKernelPackage(t, []archiveEntry{
			{name: root + "/kernel/boot.itb", typeFlag: tar.TypeReg, data: payload},
			{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
		})
		if err == nil {
			t.Errorf("boot.itb %q was accepted as a FIT image", payload)
		}
	}
}

func TestKernelSubtreeMustHoldARegularBootItb(t *testing.T) {
	root := "nanokvm_1.2.3"
	if _, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/boot.itb", typeFlag: tar.TypeDir},
		{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
	}); err == nil {
		t.Error("a directory named boot.itb was accepted")
	}
	if _, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
	}); err == nil {
		t.Error("a kernel subtree with no boot.itb was accepted")
	}
	if _, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/boot.itb", typeFlag: tar.TypeReg, data: "\xd0\x0d\xfe\xed"},
	}); err == nil {
		t.Error("a kernel subtree with no kernel.version was accepted")
	}
	if _, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/boot.itb", typeFlag: tar.TypeReg, data: "\xd0\x0d\xfe\xed"},
		{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "not-a-version\n"},
	}); err == nil {
		t.Error("a kernel subtree with an unparseable version was accepted")
	}
}

// A/B protects the kernel and not the application, so a rollback would
// otherwise leave the new application running on the old kernel.
func TestAPackageMayNotCarryBothAKernelAndAnApplication(t *testing.T) {
	root := "nanokvm_1.2.3"
	_, err := extractedKernelPackage(t, []archiveEntry{
		{name: root + "/kernel/boot.itb", typeFlag: tar.TypeReg, data: "\xd0\x0d\xfe\xed"},
		{name: root + "/kernel/kernel.version", typeFlag: tar.TypeReg, data: "1.2.3\n"},
		{name: root + "/server/NanoKVM-Server", typeFlag: tar.TypeReg, data: "server"},
		{name: root + "/kvm_system/kvm_system", typeFlag: tar.TypeReg, data: "system"},
		{name: root + "/system/init.d/S95nanokvm", typeFlag: tar.TypeReg, data: "init"},
	})
	if err == nil {
		t.Fatal("a package carrying both a kernel and an application was accepted")
	}
	if !strings.Contains(err.Error(), "its own package") {
		t.Errorf("unexpected rejection: %v", err)
	}
}

func TestManifestAndPackageMustAgreeAboutTheKernel(t *testing.T) {
	itb := filepath.Join(t.TempDir(), "boot.itb")
	if err := os.WriteFile(itb, []byte("\xd0\x0d\xfe\xed"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum512([]byte("\xd0\x0d\xfe\xed"))
	sum := base64.StdEncoding.EncodeToString(digest[:])
	kernel := &kernelPayload{itb: itb, version: "1.2.3"}

	if err := validateManifestKernel(&Latest{}, kernel); err == nil {
		t.Error("an undeclared kernel was installed")
	}
	if err := validateManifestKernel(&Latest{KernelVersion: "1.2.3", KernelSha512: sum}, nil); err == nil {
		t.Error("a manifest declaring a kernel accepted a package without one")
	}
	if err := validateManifestKernel(&Latest{KernelVersion: "1.2.4", KernelSha512: sum}, kernel); err == nil {
		t.Error("a kernel version mismatch was accepted")
	}
	if err := validateManifestKernel(&Latest{KernelVersion: "1.2.3", KernelSha512: "AAAA"}, kernel); err == nil {
		t.Error("a kernel sha512 mismatch was accepted")
	}
	if err := validateManifestKernel(&Latest{KernelVersion: "1.2.3", KernelSha512: sum}, kernel); err != nil {
		t.Errorf("a matching manifest and kernel were rejected: %v", err)
	}
}
