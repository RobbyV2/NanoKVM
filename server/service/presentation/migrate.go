package presentation

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProfileCurrent = "current"

	sentinelNCM        = "usb.ncm"
	sentinelRNDIS      = "usb.rndis0"
	sentinelDisk       = "usb.disk0"
	sentinelDiskRO     = "usb.disk0.ro"
	sentinelBIOS       = "BIOS"
	sentinelNoWakeup   = "usb.notwakeup"
	sentinelDisableHID = "disable_hid"
	sentinelVendorID   = "usb.vid"
	sentinelProductID  = "usb.pid"

	DefaultDiskFile = "/dev/mmcblk0p3"

	osDescVendorCode   = "0xCD"
	osDescQwSign       = "MSFT100"
	macDevPrefix       = "48:da:35:6e"
	macHostPrefix      = "48:da:35:6d"
	compatibleNCM      = "WINNCM"
	compatibleRNDIS    = "RNDIS"
	subCompatibleRNDIS = "5162001"
)

var (
	bootDir     = "/boot"
	baseUIDPath = "/sys/class/cvi-base/base_uid"
)

type sentinels struct {
	ncm          bool
	rndis        bool
	disk         bool
	diskReadOnly bool
	bios         bool
	noWakeup     bool
	disableHID   bool
	diskFile     string
	vendorID     string
	productID    string
}

func Migrate(store *Store, ops Ops) error {
	active, err := store.Active()
	if err != nil {
		return fmt.Errorf("read active profile: %w", err)
	}
	if active != "" {
		return nil
	}

	profile := derivedProfile()
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("migrate %s: %w", profile.Name, err)
	}
	if err := store.SaveProfile(profile); err != nil {
		return fmt.Errorf("migrate %s: %w", profile.Name, err)
	}
	if hidOnlyGadget(ops) {
		return store.SetActive(ProfileHIDOnly)
	}
	return store.SetActive(profile.Name)
}

func derivedProfile() Profile {
	flags := readSentinels()

	profile := standardProfile()
	profile.Name = ProfileCurrent
	profile.BuiltIn = false
	profile.Device.VendorID, profile.Device.ProductID = flags.vendorID, flags.productID

	var functions []Function
	if kind, ok := flags.netKind(); ok {
		functions = append(functions, NetworkFunction(kind))
		profile.OSDesc = MSOSDesc()
	}
	if !flags.disableHID {
		for _, function := range profile.Functions {
			function.HID.WakeupOnWrite = !flags.noWakeup
			if flags.bios {
				function.HID.SubClass = 1
			}
			functions = append(functions, function)
		}
	}
	if flags.disk {
		disk := DiskFunction(flags.diskFile)
		disk.Storage.ReadOnly = flags.diskReadOnly
		functions = append(functions, disk)
	}

	profile.Functions = functions
	profile.Normalize()
	return profile
}

// D5, S03usbdev:53,61: ncm has absolute priority over rndis.
func (s sentinels) netKind() (FunctionKind, bool) {
	switch {
	case s.ncm:
		return FunctionNCM, true
	case s.rndis:
		return FunctionRNDIS, true
	}
	return "", false
}

func NetworkFunction(kind FunctionKind) Function {
	dev, host := gadgetMACs()

	net := &NetFunction{DevAddr: dev, HostAddr: host, CompatibleID: compatibleNCM}
	if kind == FunctionRNDIS {
		net.SubClass, net.Protocol = ptr[uint8](0x01), ptr[uint8](0x03)
		net.CompatibleID, net.SubCompatibleID = compatibleRNDIS, subCompatibleRNDIS
	}
	return Function{Kind: kind, Instance: netInstance, Net: net}
}

func DiskFunction(file string) Function {
	if file == "" {
		file = DefaultDiskFile
	}
	return Function{Kind: FunctionMassStorage, Instance: diskInstance, Storage: &StorageFunction{
		Removable:     true,
		InquiryString: InquiryString,
		File:          file,
	}}
}

func MSOSDesc() *OSDesc {
	return &OSDesc{VendorCode: osDescVendorCode, QwSign: osDescQwSign}
}

// H13: sixteen bits of entropy off the chip UID, and the script's [ -n ] guards
// drop both addresses when base_uid is unreadable, which hands the gadget a
// fresh random MAC on every bind.
func gadgetMACs() (*string, *string) {
	data, err := os.ReadFile(baseUIDPath)
	if err != nil {
		return nil, nil
	}

	sum := sha512.Sum512(data)
	uid := hex.EncodeToString(sum[:])[:4]
	return ptr(macDevPrefix + ":" + uid[:2] + ":" + uid[2:]), ptr(macHostPrefix + ":" + uid[:2] + ":" + uid[2:])
}

func hidOnlyGadget(ops Ops) bool {
	if ops == nil {
		return false
	}

	data, err := ops.ReadFile(attrBCDDevice)
	if err != nil {
		return false
	}
	mode, err := modeFromBCDDevice(strings.TrimSpace(string(data)))
	return err == nil && mode == ModeHIDOnly
}

func readSentinels() sentinels {
	device := standardProfile().Device
	flags := sentinels{
		ncm:          sentinelExists(sentinelNCM),
		rndis:        sentinelExists(sentinelRNDIS),
		disk:         sentinelExists(sentinelDisk),
		diskReadOnly: sentinelExists(sentinelDiskRO),
		bios:         sentinelExists(sentinelBIOS),
		noWakeup:     sentinelExists(sentinelNoWakeup),
		disableHID:   sentinelExists(sentinelDisableHID),
		diskFile:     DefaultDiskFile,
		vendorID:     device.VendorID,
		productID:    device.ProductID,
	}

	for name, field := range map[string]*string{
		sentinelDisk:      &flags.diskFile,
		sentinelVendorID:  &flags.vendorID,
		sentinelProductID: &flags.productID,
	} {
		if content := sentinelValue(name); content != "" {
			*field = content
		}
	}
	return flags
}

func sentinelExists(name string) bool {
	_, err := os.Stat(filepath.Join(bootDir, name))
	return err == nil
}

func sentinelValue(name string) string {
	data, err := os.ReadFile(filepath.Join(bootDir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// D4: the init script is still the boot-time configurator and system_init.cpp
// may restore an old copy of it over /etc/init.d, so every apply rewrites the
// sentinels it reads. hid-only is a script swap rather than a sentinel state,
// so it leaves the user's network and disk choice on disk untouched.
func mirrorSentinels(p Profile) error {
	if p.Name == ProfileHIDOnly {
		return nil
	}

	contents := map[string]*string{sentinelNCM: nil, sentinelRNDIS: nil, sentinelDisk: nil}
	for _, function := range p.Functions {
		switch function.Kind {
		case FunctionNCM:
			contents[sentinelNCM] = ptr("")
		case FunctionRNDIS:
			contents[sentinelRNDIS] = ptr("")
		case FunctionMassStorage:
			content := ""
			if function.Storage.File != DefaultDiskFile {
				content = function.Storage.File + "\n"
			}
			contents[sentinelDisk] = &content
		}
	}

	for name, content := range contents {
		path := filepath.Join(bootDir, name)
		if content == nil {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove sentinel %s: %w", path, err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte(*content), 0o644); err != nil {
			return fmt.Errorf("write sentinel %s: %w", path, err)
		}
	}
	return nil
}
