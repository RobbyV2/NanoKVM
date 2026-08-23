package vnc

const maxPressedKeys = 6

type keyboardState struct {
	modifiers byte
	pressed   []uint32
	usages    map[uint32]byte
	shifted   map[uint32]bool
}

func newKeyboardState() *keyboardState {
	return &keyboardState{
		usages:  make(map[uint32]byte),
		shifted: make(map[uint32]bool),
	}
}

// key updates the tracked keyboard state and returns the report to send, or nil
// when the keysym is not mappable to a USB HID usage.
func (k *keyboardState) key(keysym uint32, down bool) []byte {
	if bit, ok := modifierKeysyms[keysym]; ok {
		if down {
			k.modifiers |= bit
		} else {
			k.modifiers &^= bit
		}
		return k.report()
	}

	usage, needsShift, ok := keysymToUsage(keysym)
	if !ok {
		return nil
	}

	if down {
		if _, held := k.usages[keysym]; !held {
			if len(k.pressed) >= maxPressedKeys {
				return nil
			}
			k.pressed = append(k.pressed, keysym)
		}
		k.usages[keysym] = usage
		k.shifted[keysym] = needsShift
	} else {
		delete(k.usages, keysym)
		delete(k.shifted, keysym)
		for index, held := range k.pressed {
			if held == keysym {
				k.pressed = append(k.pressed[:index], k.pressed[index+1:]...)
				break
			}
		}
	}

	return k.report()
}

func (k *keyboardState) report() []byte {
	report := make([]byte, 8)
	report[0] = k.modifiers

	index := 2
	for _, keysym := range k.pressed {
		if k.shifted[keysym] {
			report[0] |= modLeftShift
		}
		report[index] = k.usages[keysym]
		index++
	}

	return report
}

type pointerState struct {
	mask byte
	x    uint16
	y    uint16
}

const (
	rfbButtonLeft   = 1 << 0
	rfbButtonMiddle = 1 << 1
	rfbButtonRight  = 1 << 2
	rfbWheelUp      = 1 << 3
	rfbWheelDown    = 1 << 4
)

// pointer converts an RFB pointer event into an absolute HID mouse report.
// Framebuffer coordinates are scaled the same way the web client scales them.
func (p *pointerState) pointer(mask byte, x uint16, y uint16, width uint16, height uint16) []byte {
	var buttons byte
	if mask&rfbButtonLeft != 0 {
		buttons |= 0x01
	}
	if mask&rfbButtonRight != 0 {
		buttons |= 0x02
	}
	if mask&rfbButtonMiddle != 0 {
		buttons |= 0x04
	}

	var wheel byte
	if mask&rfbWheelUp != 0 && p.mask&rfbWheelUp == 0 {
		wheel = 1
	} else if mask&rfbWheelDown != 0 && p.mask&rfbWheelDown == 0 {
		wheel = 0xff
	}

	p.x = scaleCoordinate(x, width)
	p.y = scaleCoordinate(y, height)
	p.mask = mask

	return []byte{
		buttons,
		byte(p.x), byte(p.x >> 8),
		byte(p.y), byte(p.y >> 8),
		wheel,
	}
}

func scaleCoordinate(value uint16, size uint16) uint16 {
	if size <= 1 {
		return 1
	}
	if value >= size {
		value = size - 1
	}
	scaled := uint32(value)*0x7fff/uint32(size-1) + 1
	if scaled > 0x7fff {
		scaled = 0x7fff
	}
	return uint16(scaled)
}
