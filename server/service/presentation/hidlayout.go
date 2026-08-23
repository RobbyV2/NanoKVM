package presentation

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

type HIDRole string

const (
	HIDRoleKeyboard HIDRole = "keyboard"
	HIDRoleRelative HIDRole = "relative"
	HIDRoleAbsolute HIDRole = "absolute"
)

var HIDRoles = [...]HIDRole{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}

var (
	ErrUnknownHIDRole = errors.New("unknown hid role")
	ErrHIDLayout      = errors.New("invalid hid layout")
)

// One /dev/hidgN write target for one role. ReportID is 0 when the role owns
// its interface alone, in which case reports carry no prefix byte and the
// descriptor is the byte-pinned single-role one the device has always shipped.
type HIDRoute struct {
	Role     HIDRole `json:"role"`
	Path     string  `json:"path"`
	ReportID uint8   `json:"report_id"`
	Length   int     `json:"length"`
}

var hidRolePayload = map[HIDRole]uint16{HIDRoleKeyboard: 8, HIDRoleRelative: 4, HIDRoleAbsolute: 6}

var hidRoleProtocol = map[HIDRole]uint8{HIDRoleKeyboard: 1, HIDRoleRelative: 2, HIDRoleAbsolute: 2}

func hidRoleDescriptor(role HIDRole, subClass uint8) ([]byte, error) {
	boot := subClass == 1
	switch role {
	case HIDRoleKeyboard:
		if boot {
			return descKeyboardHIDOnly, nil
		}
		return descKeyboardStandard, nil
	case HIDRoleRelative:
		return descMouseRelative, nil
	case HIDRoleAbsolute:
		if boot {
			return descPointerHIDOnly, nil
		}
		return descPointerStandard, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownHIDRole, role)
}

func ParseHIDRole(name string) (HIDRole, error) {
	for _, role := range HIDRoles {
		if string(role) == name {
			return role, nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownHIDRole, name)
}

// A layout is the groups in link order; group i becomes hid.GS<i> and every
// role in it shares that interface's single interrupt IN endpoint. Instances
// have to stay a prefix of GS0,GS1,GS2 because f_hid hands out the /dev/hidgN
// minor at mkdir in creation order and nothing at runtime reads it back.
func ValidateHIDLayout(groups [][]HIDRole) error {
	if len(groups) == 0 {
		return fmt.Errorf("%w: no hid interfaces", ErrHIDLayout)
	}
	if len(groups) > len(hidInstances) {
		return fmt.Errorf("%w: %d interfaces, at most %d", ErrHIDLayout, len(groups), len(hidInstances))
	}
	seen := make(map[HIDRole]bool, len(HIDRoles))
	for _, group := range groups {
		if len(group) == 0 {
			return fmt.Errorf("%w: empty interface", ErrHIDLayout)
		}
		for _, role := range group {
			if _, err := ParseHIDRole(string(role)); err != nil {
				return fmt.Errorf("%w: %s", ErrHIDLayout, err)
			}
			if seen[role] {
				return fmt.Errorf("%w: role %q appears twice", ErrHIDLayout, role)
			}
			seen[role] = true
		}
	}
	return nil
}

// The report ID is a global item that stays in force until the next one, so
// concatenating single-role collections that each open with their own ID is a
// well-formed composite descriptor and every role keeps its byte layout behind
// a one-byte prefix.
func composeHIDReport(group []HIDRole, subClass uint8) ([]byte, uint16, error) {
	if len(group) == 1 {
		desc, err := hidRoleDescriptor(group[0], subClass)
		if err != nil {
			return nil, 0, err
		}
		return bytes.Clone(desc), hidRolePayload[group[0]], nil
	}

	var composed []byte
	var payload uint16
	for index, role := range group {
		desc, err := hidRoleDescriptor(role, subClass)
		if err != nil {
			return nil, 0, err
		}
		tagged, err := insertReportID(desc, uint8(index+1))
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", role, err)
		}
		composed = append(composed, tagged...)
		payload = max(payload, hidRolePayload[role])
	}
	return composed, payload + 1, nil
}

// After the application collection opens and before its first main item, which
// is where a Report ID has to sit for every item that follows to inherit it.
func insertReportID(desc []byte, id uint8) ([]byte, error) {
	for offset := 0; offset < len(desc); {
		size := int(desc[offset] & 0x03)
		if size == 3 {
			size = 4
		}
		next := offset + 1 + size
		if next > len(desc) {
			return nil, fmt.Errorf("truncated item at offset %d", offset)
		}
		if desc[offset]&0xFC == 0xA0 {
			tagged := make([]byte, 0, len(desc)+2)
			tagged = append(tagged, desc[:next]...)
			tagged = append(tagged, 0x85, id)
			return append(tagged, desc[next:]...), nil
		}
		offset = next
	}
	return nil, errors.New("no application collection")
}

func hidLayoutFunctions(groups [][]HIDRole, subClass uint8) ([]Function, error) {
	if err := ValidateHIDLayout(groups); err != nil {
		return nil, err
	}
	functions := make([]Function, 0, len(groups))
	for index, group := range groups {
		desc, length, err := composeHIDReport(group, subClass)
		if err != nil {
			return nil, err
		}
		protocol, functionSubClass := hidRoleProtocol[group[0]], subClass
		if len(group) > 1 {
			// bInterfaceProtocol and the boot subclass promise a boot report
			// on this interface, and a report-ID composite cannot deliver one.
			protocol, functionSubClass = 0, 0
		}
		functions = append(functions, Function{
			Kind:     FunctionHID,
			Instance: hidInstances[index],
			HID: &HIDFunction{
				Protocol:      protocol,
				SubClass:      functionSubClass,
				ReportLength:  length,
				WakeupOnWrite: true,
				ReportDesc:    desc,
				Roles:         slices.Clone(group),
				DevNodeIndex:  index,
			},
		})
	}
	return functions, nil
}

func HIDLayout(functions []Function) [][]HIDRole {
	var groups [][]HIDRole
	for _, function := range functions {
		if function.Kind == FunctionHID && function.HID != nil {
			groups = append(groups, slices.Clone(function.HID.Roles))
		}
	}
	return groups
}

func HIDRoutes(functions []Function) []HIDRoute {
	var routes []HIDRoute
	for _, function := range functions {
		if function.Kind != FunctionHID || function.HID == nil {
			continue
		}
		path := fmt.Sprintf("/dev/hidg%d", function.HID.DevNodeIndex)
		for index, role := range function.HID.Roles {
			route := HIDRoute{Role: role, Path: path, Length: int(hidRolePayload[role])}
			if len(function.HID.Roles) > 1 {
				route.ReportID = uint8(index + 1)
			}
			routes = append(routes, route)
		}
	}
	return routes
}

// SetHIDLayout rebuilds a profile's HID functions in place, keeping their
// position in the link order because that is what fixes bInterfaceNumber.
func SetHIDLayout(profile *Profile, groups [][]HIDRole) error {
	// The descriptor variant follows the mode marker, not the surviving
	// function's subclass, which a composite interface has already given up.
	subClass := uint8(0)
	if profile.Device.BCDDevice != nil && *profile.Device.BCDDevice == BCDDeviceHIDOnly {
		subClass = 1
	}
	first := -1
	rest := make([]Function, 0, len(profile.Functions))
	for _, function := range profile.Functions {
		if function.Kind != FunctionHID {
			rest = append(rest, function)
			continue
		}
		if first < 0 {
			first = len(rest)
		}
	}
	replacement, err := hidLayoutFunctions(groups, subClass)
	if err != nil {
		return err
	}
	if first < 0 {
		first = len(rest)
	}
	functions := make([]Function, 0, len(rest)+len(replacement))
	functions = append(functions, rest[:first]...)
	functions = append(functions, replacement...)
	functions = append(functions, rest[first:]...)
	profile.Functions = functions
	// A built-in is code and cannot carry the operator's layout, so editing one
	// produces the same "current" profile a media slot change does.
	if profile.BuiltIn {
		profile.Name, profile.BuiltIn = ProfileCurrent, false
	}
	profile.Normalize()
	return nil
}

func boolAttr(on bool) string {
	if on {
		return "1"
	}
	return "0"
}

func (v *VideoFunction) interruptEndpoint() bool {
	return v.InterruptEndpoint == nil || *v.InterruptEndpoint
}
