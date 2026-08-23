package presentation

import "strings"

// A preset is a device identity and nothing else. It carries the four fields
// the USB ID Repository publishes for a device - vendor, product and the two
// registered names - because those are the only parts of somebody else's device
// that can be stated without having read the device. A full descriptor tree
// cannot be written from a public registry, and this repository has no capture
// tool yet, so no shipped preset carries one and none of them pretends to: the
// Profile schema already holds a captured tree in Descriptors, and a preset
// that grows one later needs no schema change, only the capture that produced
// it and a Provenance saying so.
//
// Manufacturer and Product are the registry's own strings rather than the
// strings a real unit reports, which are not knowable from the registry. Source
// names the entry so the difference stays checkable.
type Preset struct {
	ID           string `json:"id"`
	VendorID     string `json:"vendor_id"`
	ProductID    string `json:"product_id"`
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	Source       string `json:"source"`
}

var presets = [...]Preset{
	{
		ID:           "nanokvm",
		VendorID:     "0x3346",
		ProductID:    "0x1009",
		Manufacturer: "sipeed",
		Product:      "NanoKVM",
		Source:       "kvmapp/system/init.d/S03usbdev",
	},
	{
		ID:           "logitech-unifying-receiver",
		VendorID:     "0x046d",
		ProductID:    "0xc52b",
		Manufacturer: "Logitech, Inc.",
		Product:      "Unifying Receiver",
		Source:       "usb.ids 046d:c52b",
	},
	{
		ID:           "dell-kb216",
		VendorID:     "0x413c",
		ProductID:    "0x2113",
		Manufacturer: "Dell Computer Corp.",
		Product:      "KB216 Wired Keyboard",
		Source:       "usb.ids 413c:2113",
	},
	{
		ID:           "sandisk-cruzer-blade",
		VendorID:     "0x0781",
		ProductID:    "0x5567",
		Manufacturer: "SanDisk Corp.",
		Product:      "Cruzer Blade",
		Source:       "usb.ids 0781:5567",
	},
	{
		ID:           "linux-composite-gadget",
		VendorID:     "0x1d6b",
		ProductID:    "0x0104",
		Manufacturer: "Linux Foundation",
		Product:      "Multifunction Composite Gadget",
		Source:       "usb.ids 1d6b:0104",
	},
	{
		ID:           "linux-hid-gadget",
		VendorID:     "0x0525",
		ProductID:    "0xa4ac",
		Manufacturer: "Netchip Technology, Inc.",
		Product:      "Linux-USB HID Gadget",
		Source:       "usb.ids 0525:a4ac",
	},
	{
		ID:           "linux-storage-gadget",
		VendorID:     "0x0525",
		ProductID:    "0xa4a5",
		Manufacturer: "Netchip Technology, Inc.",
		Product:      "Linux-USB File-backed Storage Gadget",
		Source:       "usb.ids 0525:a4a5",
	},
}

func Presets() []Preset {
	shipped := make([]Preset, len(presets))
	copy(shipped, presets[:])
	return shipped
}

func PresetByID(id string) (Preset, bool) {
	for _, preset := range presets {
		if preset.ID == id {
			return preset, true
		}
	}
	return Preset{}, false
}

// What keeps a profile from claiming a preset it has since been edited away
// from: Normalize demotes the provenance the moment the four fields stop
// agreeing with the entry it names.
func (p Preset) matches(device Device) bool {
	return strings.EqualFold(p.VendorID, device.VendorID) &&
		strings.EqualFold(p.ProductID, device.ProductID) &&
		p.Manufacturer == device.Manufacturer &&
		p.Product == device.Product
}
