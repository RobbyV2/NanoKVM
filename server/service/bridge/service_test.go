package bridge

import "testing"

// The gap this covers is not a wrong value but an absent one: with Gadget nil
// the twelve steps that hold the management address all pass, the response says
// enabled, and usb0 is silently never enslaved.
func TestServiceWiresTheGadget(t *testing.T) {
	service := newService(&fakeLiveness{})

	if service.manager.gadget == nil {
		t.Fatal("bridge.NewService leaves Gadget nil, so enable step 13 never enslaves usb0")
	}
}
