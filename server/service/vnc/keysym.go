package vnc

const (
	modLeftControl  byte = 0x01
	modLeftShift    byte = 0x02
	modLeftAlt      byte = 0x04
	modLeftGui      byte = 0x08
	modRightControl byte = 0x10
	modRightShift   byte = 0x20
	modRightAlt     byte = 0x40
	modRightGui     byte = 0x80
)

var modifierKeysyms = map[uint32]byte{
	0xffe1: modLeftShift,
	0xffe2: modRightShift,
	0xffe3: modLeftControl,
	0xffe4: modRightControl,
	0xffe9: modLeftAlt,
	0xffea: modRightAlt,
	0xffe7: modLeftGui,
	0xffe8: modRightGui,
	0xffeb: modLeftGui,
	0xffec: modRightGui,
}

var unshiftedASCII = map[uint32]byte{
	' ': 0x2c, '-': 0x2d, '=': 0x2e, '[': 0x2f, ']': 0x30, '\\': 0x31,
	';': 0x33, '\'': 0x34, '`': 0x35, ',': 0x36, '.': 0x37, '/': 0x38,
}

var shiftedASCII = map[uint32]byte{
	'!': 0x1e, '@': 0x1f, '#': 0x20, '$': 0x21, '%': 0x22, '^': 0x23,
	'&': 0x24, '*': 0x25, '(': 0x26, ')': 0x27,
	'_': 0x2d, '+': 0x2e, '{': 0x2f, '}': 0x30, '|': 0x31,
	':': 0x33, '"': 0x34, '~': 0x35, '<': 0x36, '>': 0x37, '?': 0x38,
}

var specialKeysyms = map[uint32]byte{
	0xff08: 0x2a, // BackSpace
	0xff09: 0x2b, // Tab
	0xff0d: 0x28, // Return
	0xff8d: 0x58, // KP_Enter
	0xff1b: 0x29, // Escape
	0xff63: 0x49, // Insert
	0xffff: 0x4c, // Delete
	0xff50: 0x4a, // Home
	0xff57: 0x4d, // End
	0xff55: 0x4b, // Page_Up
	0xff56: 0x4e, // Page_Down
	0xff51: 0x50, // Left
	0xff52: 0x52, // Up
	0xff53: 0x4f, // Right
	0xff54: 0x51, // Down
	0xff61: 0x46, // Print
	0xff14: 0x47, // Scroll_Lock
	0xff13: 0x48, // Pause
	0xffe5: 0x39, // Caps_Lock
	0xff7f: 0x53, // Num_Lock
	0xff67: 0x65, // Menu
	0xffaa: 0x55, // KP_Multiply
	0xffab: 0x57, // KP_Add
	0xffad: 0x56, // KP_Subtract
	0xffae: 0x63, // KP_Decimal
	0xffaf: 0x54, // KP_Divide
	0xffb0: 0x62, // KP_0
	0xffb1: 0x59,
	0xffb2: 0x5a,
	0xffb3: 0x5b,
	0xffb4: 0x5c,
	0xffb5: 0x5d,
	0xffb6: 0x5e,
	0xffb7: 0x5f,
	0xffb8: 0x60,
	0xffb9: 0x61, // KP_9
}

// keysymToUsage maps an X11 keysym to a USB HID usage on the keyboard page and
// reports whether the key needs shift to produce the requested symbol.
func keysymToUsage(keysym uint32) (byte, bool, bool) {
	if keysym >= 'a' && keysym <= 'z' {
		return byte(keysym-'a') + 0x04, false, true
	}
	if keysym >= 'A' && keysym <= 'Z' {
		return byte(keysym-'A') + 0x04, true, true
	}
	if keysym >= '1' && keysym <= '9' {
		return byte(keysym-'1') + 0x1e, false, true
	}
	if keysym == '0' {
		return 0x27, false, true
	}
	if keysym >= 0xffbe && keysym <= 0xffc9 {
		return byte(keysym-0xffbe) + 0x3a, false, true
	}
	if usage, ok := unshiftedASCII[keysym]; ok {
		return usage, false, true
	}
	if usage, ok := shiftedASCII[keysym]; ok {
		return usage, true, true
	}
	if usage, ok := specialKeysyms[keysym]; ok {
		return usage, false, true
	}
	return 0, false, false
}
