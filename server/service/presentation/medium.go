package presentation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
)

// ErrMediumLocked is the kernel refusing to release a medium the host has
// locked (prevent_medium_removal, which Windows sets while the disk is
// mounted). It is a normal condition: the host has to eject the disk, or the
// caller forces the change, which takes the controller off the bus first.
var ErrMediumLocked = errors.New("the target machine is using the medium")

// LUN is what the disk function serves. It is runtime state rather than
// profile state: service/storage owns it through SetLUN, and an apply that
// keeps the disk linked leaves file, ro and cdrom as they are (see
// keepsRuntimeMedium). File is the backing file: DefaultDiskFile is the data
// partition and an empty File is no medium at all.
type LUN struct {
	File     string `json:"file"`
	ReadOnly bool   `json:"ro"`
	CDROM    bool   `json:"cdrom"`

	// Product names the medium in the SCSI inquiry data, cut to the field's 16
	// bytes; empty names it after the mode. An inquiry_string the active profile
	// chose for itself wins over both. It is written, never read back.
	Product string `json:"product,omitempty"`
}

// The SCSI inquiry fields are fixed byte widths: vendor 8, product 16 and
// revision 4. Padding by bytes rather than by runes keeps a product name with
// multibyte characters from pushing the revision out of the 28 bytes the
// kernel keeps.
const (
	inquiryVendor      = "NanoKVM"
	inquiryRevision    = "0520"
	inquiryProductSize = 16

	inquiryProductDisk  = "USB Mass Storage"
	inquiryProductCDROM = "USB CD/DVD-ROM"
)

func inquiryString(product string) string {
	product = cutUTF8(product, inquiryProductSize)
	return fmt.Sprintf("%-8s", inquiryVendor) + product +
		strings.Repeat(" ", inquiryProductSize-len(product)) + inquiryRevision
}

func cutUTF8(value string, size int) string {
	if len(value) <= size {
		return value
	}
	value = value[:size]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func (l LUN) inquiry() string {
	switch {
	case l.Product != "":
		return inquiryString(l.Product)
	case l.CDROM:
		return InquiryStringCDROM
	default:
		return InquiryString
	}
}

func (l LUN) servesPartition() bool {
	return l.File == DefaultDiskFile && !l.ReadOnly
}

// DiskGuard is the other owner of the data partition the disk serves by
// default: this device mounts the same filesystem at /data, and the two sides
// must never both write it. service/storage implements it. The dependency
// points from storage into this package, so it is injected the way the HID
// quiescer and the media observer are. Both methods run under the gadget lock,
// which is what orders them against every change to the LUN.
type DiskGuard interface {
	// ServePartition runs before any write that makes the LUN serve the data
	// partition writable. It returns nil only once this side has stopped
	// writing the partition; an error refuses the write.
	ServePartition() error
	// Settled runs at the end of every gadget-lock section with whether the
	// linked disk now serves the data partition writable, so this side takes
	// the partition back once nothing hands it to the host.
	Settled(servesPartition bool)
}

func (m *Manager) SetDiskGuard(guard DiskGuard) {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	m.disk = guard
}

func (m *Manager) diskGuard() DiskGuard {
	m.wireMu.Lock()
	defer m.wireMu.Unlock()
	return m.disk
}

func lunAttr(attr string) string {
	return functionsDir + "/" + diskFunctionName + "/" + lunDir + "/" + attr
}

func (m *Manager) diskLinked() bool {
	return readable(m.ops, configPrefix+"/"+diskFunctionName+"/"+lunDir+"/file")
}

// LUN reads what the disk serves. linked is false when no disk function is
// linked into the config, in which case nothing reaches the host whatever the
// function directory still holds.
func (m *Manager) LUN() (lun LUN, linked bool, err error) {
	if err := m.ready(); err != nil {
		return LUN{}, false, err
	}
	if !m.diskLinked() {
		return LUN{}, false, nil
	}
	lun, err = m.readLUN()
	return lun, err == nil, err
}

func (m *Manager) readLUN() (LUN, error) {
	var values [3]string
	for i, attr := range [...]string{"file", "ro", "cdrom"} {
		data, err := m.ops.ReadFile(lunAttr(attr))
		if err != nil {
			return LUN{}, fmt.Errorf("read %s: %w", lunAttr(attr), err)
		}
		values[i] = strings.TrimSpace(string(data))
	}
	return LUN{File: values[0], ReadOnly: values[1] == "1", CDROM: values[2] == "1"}, nil
}

// A LUN change ends in a rebind, and a rebind takes the camera's video node
// away with it: f_uvc unregisters its device on unbind and registers a new one
// on bind, the descriptor the media manager holds stays on the old one, and the
// new function keeps the whole gadget deactivated until something opens the new
// node. Left alone, mounting an image cost the operator HID, the NIC and the
// very disk being mounted until the next profile apply. So the media pipeline
// is stood down before and rebuilt after, exactly as Rebind does; the refresh
// runs on the refusal paths too, since a suspend is undone by nothing else.
//
// force takes the medium from a host that locked it: the kernel clears that
// lock when the function is reconfigured, so a refused release is retried with
// the controller off the bus, and the gadget is bound again whatever happens.
func (m *Manager) SetLUN(ctx context.Context, lun LUN, force bool) error {
	if err := m.ready(); err != nil {
		return err
	}
	_ = m.suspend(false)
	err := m.withGadgetLock(func() error { return m.setLUN(ctx, lun, force) })
	m.refreshObserver(context.Background())
	return err
}

// Ordering constraint 3: the LUN attributes carry no refcnt check, but ro and
// cdrom return -EBUSY while lun.0/file is open, so the medium is released
// first and every flag is written on every change. A flag left behind is how
// an image ends up presented as a CD-ROM, or the partition as read-only.
// S03usbdev rebuilds the LUN from the /boot sentinels on every boot, so this
// runtime state is deliberately not part of the profile (H7). The exception is
// an inquiry_string the active profile set for itself, which a mount would
// otherwise throw away every time the disk changed.
func (m *Manager) setLUN(ctx context.Context, lun LUN, force bool) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("set lun: %w", err)
	}
	if lun.CDROM {
		lun.ReadOnly = true
	}
	// Refused before anything is released, so a refusal leaves the host the
	// medium it had.
	if lun.servesPartition() {
		if err := m.guardPartition(); err != nil {
			return fmt.Errorf("set lun: %w", err)
		}
	}

	unbound := false
	err := m.releaseMedium()
	if errors.Is(err, ErrMediumLocked) && force {
		if err = m.unbindIfBound(); err != nil {
			return fmt.Errorf("set lun: unbind: %w", err)
		}
		unbound = true
		err = m.releaseMedium()
	}
	if err == nil {
		err = m.writeMedium(lun)
	}
	switch {
	case unbound && err != nil:
		// Leaving the controller off the bus would cost the host its keyboard
		// and mouse as well as the disk.
		return errors.Join(err, m.bind())
	case err != nil:
		return err
	case unbound:
		return m.bind()
	default:
		return m.rebind(ctx)
	}
}

// A zero-length write never reaches the kernel, so an empty line is what
// releases the medium.
func (m *Manager) releaseMedium() error {
	if err := m.ops.WriteFile(lunAttr("file"), []byte("\n")); err != nil {
		if errors.Is(err, syscall.EBUSY) {
			return fmt.Errorf("%w: %w", ErrMediumLocked, err)
		}
		return fmt.Errorf("release backing file: %w", err)
	}
	return nil
}

func (m *Manager) writeMedium(lun LUN) error {
	inquiry := lun.inquiry()
	if chosen := m.store.ActiveInquiry(); chosen != "" {
		inquiry = chosen
	}
	for _, write := range []struct{ attr, value string }{
		{"ro", boolBit(lun.ReadOnly)},
		{"cdrom", boolBit(lun.CDROM)},
		{"inquiry_string", inquiry},
	} {
		if err := m.ops.WriteFile(lunAttr(write.attr), []byte(write.value)); err != nil {
			return fmt.Errorf("set %s: %w", lunAttr(write.attr), err)
		}
	}
	if lun.File == "" {
		return nil
	}
	if err := m.ops.WriteFile(lunAttr("file"), []byte(lun.File)); err != nil {
		if errors.Is(err, syscall.EBUSY) {
			return fmt.Errorf("%w: %w", ErrMediumLocked, err)
		}
		return fmt.Errorf("set %s: %w", lunAttr("file"), err)
	}
	return nil
}

func (m *Manager) guardPartition() error {
	guard := m.diskGuard()
	if guard == nil {
		return nil
	}
	return guard.ServePartition()
}

// A plan writes the data partition into the LUN only for a disk function it
// has just created (keepsRuntimeMedium drops the write for a linked one), and
// the compiler writes ro ahead of file, so the flag the write is about to serve
// under can be read back here.
func (m *Manager) guardPlanWrite(op Op) error {
	if op.Path != lunAttr("file") || string(bytes.TrimSpace(op.Data)) != DefaultDiskFile {
		return nil
	}
	if ro, err := m.ops.ReadFile(lunAttr("ro")); err == nil && strings.TrimSpace(string(ro)) == "1" {
		return nil
	}
	return m.guardPartition()
}

// Called with m.mu held, on the way out of every gadget-lock section.
func (m *Manager) settleDisk() {
	guard := m.diskGuard()
	if guard == nil || m.ops == nil {
		return
	}
	serves := false
	if m.diskLinked() {
		lun, err := m.readLUN()
		if err != nil {
			// Not knowing what the host is given is no reason to hand the
			// partition back to this side.
			log.Errorf("read the usb disk medium: %s", err)
			serves = true
		} else {
			serves = lun.servesPartition()
		}
	}
	guard.Settled(serves)
}

// keepsRuntimeMedium says whether an apply leaves the medium of the disk
// alone. A linked disk whose profile names the default backing file says
// nothing about the medium: what it serves is whatever service/storage last
// attached, and re-pointing it at the partition on every profile change would
// eject a mounted image, serve the partition under the image's read-only flag,
// or hand the partition to the host while this side still writes /data.
func keepsRuntimeMedium(linked map[string]bool, plan Plan) bool {
	if !linked[diskFunctionName] {
		return false
	}
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && op.Path == lunAttr("file") {
			return string(bytes.TrimSpace(op.Data)) == DefaultDiskFile
		}
	}
	return false
}

func isMediumAttr(path string) bool {
	return path == lunAttr("file") || path == lunAttr("ro") || path == lunAttr("cdrom")
}

// A linked disk that a profile re-points at a file of its own takes the new
// medium the way setLUN gives one. ro and cdrom are refused while a file is
// open, so when a flag still has to change, the medium is released first and
// the plan's ro, cdrom and file writes follow in plan order, the file included
// even where it was dropped as unchanged.
func releaseMediumFirst(ops []Op, plan Plan) []Op {
	flagChanges := false
	for _, op := range ops {
		if op.Kind == OpWrite && (op.Path == lunAttr("ro") || op.Path == lunAttr("cdrom")) {
			flagChanges = true
		}
	}
	if !flagChanges {
		return ops
	}

	var medium []Op
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && isMediumAttr(op.Path) {
			medium = append(medium, op)
		}
	}
	out := make([]Op, 0, len(ops)+len(medium)+1)
	inserted := false
	for _, op := range ops {
		if op.Kind == OpWrite && isMediumAttr(op.Path) {
			if !inserted {
				out = append(out, Op{Kind: OpWrite, Path: lunAttr("file"), Data: []byte("\n")})
				out = append(out, medium...)
				inserted = true
			}
			continue
		}
		out = append(out, op)
	}
	return out
}
