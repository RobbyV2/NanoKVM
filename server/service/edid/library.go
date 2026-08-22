package edid

import (
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrProfileNotFound = errors.New("edid: profile not found")
	ErrBlobSize        = errors.New("edid: unsupported blob size")
)

type Profile struct {
	SHA256        [32]byte
	Manufacturer  string
	Model         string
	PreferredMode string
	Source        string
	Data          []byte
}

func (p Profile) ID() string {
	return hex.EncodeToString(p.SHA256[:])
}

func Profiles() []Profile {
	shipped := make([]Profile, len(profiles))
	copy(shipped, profiles)
	return shipped
}

func ProfileByID(id string) (Profile, error) {
	for _, profile := range profiles {
		if profile.ID() == id {
			return profile, nil
		}
	}
	return Profile{}, fmt.Errorf("%w: %s", ErrProfileNotFound, id)
}

func Normalize(data []byte) ([]byte, error) {
	switch {
	case len(data) == Size:
	case len(data) == BlockSize && data[extensionCount] == 0:
	case len(data) == BlockSize:
		return nil, fmt.Errorf("%w: %d bytes declaring %d extension blocks", ErrBlobSize, BlockSize, data[extensionCount])
	default:
		return nil, fmt.Errorf("%w: %d bytes", ErrBlobSize, len(data))
	}

	blob := make([]byte, Size)
	copy(blob, data)
	return blob, nil
}
