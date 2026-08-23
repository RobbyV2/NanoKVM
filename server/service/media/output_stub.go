//go:build !linux || !cgo

package media

import "errors"

type platformFactory struct{}

func (platformFactory) Open(SlotSpec, string) (Output, error) {
	return nil, errors.New("media gadget output requires linux and cgo")
}
