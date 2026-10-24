package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/utils"
)

// The emulated USB disk is the LUN of the gadget's mass storage function, and
// the gadget belongs to the presentation manager (service/presentation): every
// medium change here is a presentation.Manager.SetLUN, which runs under the
// gadget lock with HID quiesced and the media pipeline stood down, releases the
// medium before the flags the kernel refuses to change under an open file, and
// re-enumerates the gadget once at the end. Nothing in this package writes to
// configfs.
const (
	// The SCSI inquiry product field holds 16 bytes.
	inquiryProductSize = 16
)

// ErrMediumLocked reports that the target machine is still using the medium.
// It is a normal condition, not a failure: the user has to eject the disk
// there before the web UI can take it back, or force the eject.
var ErrMediumLocked = presentation.ErrMediumLocked

type diskState string

const (
	diskStateNone        diskState = "none"        // no medium attached
	diskStateImage       diskState = "image"       // an image file is attached
	diskStatePartition   diskState = "partition"   // the whole data partition is attached
	diskStateUnsupported diskState = "unsupported" // no mass storage function (virtual disk off, or HID-only)
)

// diskStatus is what the gadget exposes to the target machine right now. The
// backing file is the single source of truth: both the UI and the file manager
// read it instead of keeping their own idea of what is mounted.
type diskStatus struct {
	State    diskState
	Backing  string
	ReadOnly bool
	CDROM    bool
	HidOnly  bool
}

func readDiskStatus() (diskStatus, error) {
	mode, err := hid.GetMode()
	hidOnly := err == nil && mode == hid.ModeHidOnly

	// Switching the virtual disk off takes the function out of the config, and
	// what is not linked does not reach the target machine whatever its LUN
	// still holds, so a mount would look like it worked and change nothing.
	lun, linked, err := hid.Manager().LUN()
	if err != nil {
		return diskStatus{}, err
	}
	if !linked {
		return diskStatus{State: diskStateUnsupported, HidOnly: hidOnly}, nil
	}

	status := diskStatus{ReadOnly: lun.ReadOnly, CDROM: lun.CDROM, HidOnly: hidOnly}
	switch {
	case lun.File == "":
		status.State = diskStateNone
	case lun.File == partitionDevice:
		status.State = diskStatePartition
		status.Backing = lun.File
	default:
		status.State = diskStateImage
		status.Backing = lun.File
		// A running gadget may still report an alias used before this process
		// started. Normalize it too, so every operation compares the same
		// exFAT directory entry rather than just the same string.
		if canonical, err := utils.CanonicalMediaPath(lun.File); err == nil {
			status.Backing = canonical
		}
	}

	return status, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// setMedium hands the gadget a medium, or none for an empty backing file. A
// CD-ROM is always read-only. force takes the medium from a target machine
// that locked it, which re-enumerates the gadget to clear that lock.
func setMedium(backing string, cdrom, ro, force bool) error {
	if cdrom {
		ro = true
	}

	return hid.Manager().SetLUN(context.Background(), presentation.LUN{
		File:     backing,
		ReadOnly: ro,
		CDROM:    cdrom,
		Product:  mediumProduct(backing),
	}, force)
}

// mediumProduct is the SCSI product string of the medium: an image is shown
// under its file name, and an empty name leaves the presentation manager to
// name the medium after its mode. The USB product string stays the profile's:
// it names the whole composite device, keyboard and camera included.
func mediumProduct(backing string) string {
	if backing == "" || backing == partitionDevice {
		return ""
	}
	return cutName(imageName(backing), inquiryProductSize)
}

// imageName strips the extension from the file name of a medium.
func imageName(path string) string {
	name := filepath.Base(path)
	if extension := filepath.Ext(name); extension != "" {
		name = strings.TrimSuffix(name, extension)
	}

	return name
}

// cutName cuts name to size bytes without leaving a broken UTF-8 rune behind:
// the inquiry field is byte limited.
func cutName(name string, size int) string {
	if len(name) <= size {
		return name
	}

	name = name[:size]
	for !utf8.ValidString(name) {
		name = name[:len(name)-1]
	}

	return name
}

// partitionGuard keeps this side off the data partition while the target
// machine writes it. The presentation manager consults it under the gadget
// lock, so it is ordered against every change to what the disk serves: a
// medium change from here, a profile apply that creates the disk, and the
// rollback of either.
type partitionGuard struct{}

// ServePartition runs before the gadget serves the raw partition writable. A
// direct write in flight would be cut off, so it refuses; otherwise /data goes
// read-only first, and a remount the kernel refuses because a writer still
// holds a file open (a writable loop mount of an image included) refuses the
// change too. The media API's own checks for images the file manager holds run
// under mediumOperationMu before it asks for the partition (attachMedium,
// LockVirtualDiskTransition); repeating them here would also refuse the
// rollback that puts back a disk which was serving the partition all along.
func (partitionGuard) ServePartition() error {
	if partitionWriteRefs.Load() > 0 {
		return errMediumBusy
	}
	return SetDataWritable(false)
}

// Settled hands /data back to this side once nothing serves the partition to
// the target machine writable, and takes it away from this side again when
// something does (the boot scripts serve it before the server is up).
func (partitionGuard) Settled(servesPartition bool) {
	if err := SetDataWritable(!servesPartition); err != nil {
		if servesPartition {
			log.Errorf("the target machine owns the data partition but %s stays writable: %s", mediaDirectory, err)
			return
		}
		log.Warnf("make %s writable failed: %s", mediaDirectory, err)
	}
}

// GuardDataPartition wires the data partition's ownership into the gadget. It
// has to run before the gadget's first bind, so the first answer this side
// gets is about the gadget the target machine enumerates.
func GuardDataPartition(manager *presentation.Manager) {
	manager.SetDiskGuard(partitionGuard{})
}
