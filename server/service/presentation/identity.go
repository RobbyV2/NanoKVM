package presentation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const (
	OriginBuiltIn  = "built-in"
	OriginPreset   = "preset"
	OriginMigrated = "migrated"
	OriginImported = "imported"
	OriginUser     = "user"

	// The vendor ID the board ships with. Anything else is somebody's.
	VendorSipeed = "0x3346"

	// The fleet-wide constant S03usbdev:32 wrote. It stays as the fallback for
	// a board whose UID cannot be read, because a serial that changes between
	// boots makes the host file a fresh device instance every time, which is a
	// worse failure than the collision it would be avoiding.
	fallbackSerial = "0123456789ABCDEF"

	serialDomain = "nanokvm-usb-serial:"
)

var provenanceOrigins = [...]string{OriginBuiltIn, OriginPreset, OriginMigrated, OriginImported, OriginUser}

// Where the identity in a profile came from, and whether it carries a
// descriptor tree captured off a real device. Descriptors is not independent
// state: Normalize derives it from Profile.Descriptors so the two can never
// disagree, and it rides in an exported manifest, where the assets have been
// split out of the profile, as the claim ImportPackage checks them against.
type Provenance struct {
	Origin      string `json:"origin"`
	Source      string `json:"source,omitempty"`
	Descriptors bool   `json:"descriptors"`
}

func (p Provenance) validate() error {
	for _, origin := range provenanceOrigins {
		if p.Origin == origin {
			return nil
		}
	}
	return fmt.Errorf("unknown origin %q", p.Origin)
}

// H13: the serial was the same sixteen characters on every NanoKVM ever
// shipped, so two of them on one host share a device-instance key. base_uid is
// the SoC's own identifier, which makes it per-device and makes it survive a
// firmware update precisely because nothing on the filesystem holds it. The
// attached host reads this string, so what goes out is a truncated hash under
// its own domain prefix and never the UID itself.
func DeviceSerial() string {
	data, err := os.ReadFile(baseUIDPath)
	if err != nil {
		return fallbackSerial
	}
	uid := bytes.TrimSpace(data)
	if len(uid) == 0 {
		return fallbackSerial
	}
	sum := sha256.Sum256(append([]byte(serialDomain), uid...))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// A captured identity is allowed, so this is not an error. It is only ever
// deliberate though, and presenting someone else's vendor ID by accident is the
// case worth naming, so the preview carries the fact next to the field.
func ForeignVendor(vendorID string) bool {
	return vendorID != "" && !strings.EqualFold(vendorID, VendorSipeed)
}
