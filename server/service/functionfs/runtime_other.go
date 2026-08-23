//go:build !linux

package functionfs

import (
	"errors"

	"NanoKVM-Server/service/presentation"
)

type Prepared struct {
	Image Image
	Relay *Relay
}

func Prepare(string, uint32, uint32, presentation.CapabilityTable) (*Prepared, error) {
	return nil, ErrUnsupported
}

func Cleanup() error {
	return errors.New("functionfs: cleanup is only available on Linux")
}
