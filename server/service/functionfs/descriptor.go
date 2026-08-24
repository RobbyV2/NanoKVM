package functionfs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"NanoKVM-Server/service/presentation"
)

const (
	MaxDescriptorBytes = 64 << 10
	MaxControlBytes    = 4 << 10
	MaxTransferBytes   = 64 << 10
	MaxInterfaces      = 16
	MaxEndpoints       = 7
)

var (
	ErrMalformed    = errors.New("functionfs: malformed descriptors")
	ErrUnsupported  = errors.New("functionfs: unsupported device")
	ErrProtected    = errors.New("functionfs: protected device class")
	ErrAmbiguous    = errors.New("functionfs: ambiguous descriptor layout")
	ErrEndpointSize = errors.New("functionfs: endpoint transfer is too large")
)

type Fetcher interface {
	Descriptor(descriptorType uint8, index uint8, recipient uint16, limit int) ([]byte, error)
}

type Image struct {
	Device         []byte
	Configuration  []byte
	BOS            []byte
	Strings        map[uint8]string
	HIDReports     map[uint8][]byte
	HIDDescriptors map[uint8][]byte
	Interfaces     map[uint8]uint8
	Endpoints      map[uint8]uint8
	Alternates     map[uint8]uint8
	EndpointOwners map[uint8]uint8
	Function       presentation.FunctionFS
	Descriptors    []byte
	StringTable    []byte
}

type descriptor struct {
	offset          int
	data            []byte
	interfaceNumber uint8
	alternate       uint8
	class           uint8
	subClass        uint8
	hasInterface    bool
}

var protectedClasses = map[uint8]string{
	0x01: "audio",
	0x08: "mass storage",
	0x09: "hub",
	0x0b: "smart card",
	0x0d: "content security",
	0x10: "audio/video",
	0xe0: "wireless controller",
}

var supportedClasses = map[uint8]bool{
	0x02: true,
	0x03: true,
	0x07: true,
	0x0a: true,
	0x0e: true,
	0xff: true,
}

func Import(raw []byte, fetcher Fetcher, caps presentation.CapabilityTable) (Image, error) {
	device, config, descriptors, err := splitDescriptors(raw)
	if err != nil {
		return Image{}, err
	}
	if label, protected := protectedClasses[device[4]]; protected {
		return Image{}, fmt.Errorf("%w: device is %s class", ErrProtected, label)
	}
	// EFh/02h/01h is the Interface Association Descriptor marker. It binds no
	// device-level driver and only tells the host to read the associations this
	// already parses, which is what a composite camera declares. Every other
	// device-level class binds a driver to the whole device.
	if device[4] != 0 && !(device[4] == 0xef && device[5] == 0x02 && device[6] == 0x01) {
		return Image{}, fmt.Errorf("%w: device-level class 0x%02x/0x%02x/0x%02x needs Exact mode", ErrUnsupported, device[4], device[5], device[6])
	}

	image := Image{
		Device:         slices.Clone(device),
		Configuration:  slices.Clone(config),
		Strings:        make(map[uint8]string),
		HIDReports:     make(map[uint8][]byte),
		HIDDescriptors: make(map[uint8][]byte),
		Interfaces:     make(map[uint8]uint8),
		Endpoints:      make(map[uint8]uint8),
		Alternates:     make(map[uint8]uint8),
		EndpointOwners: make(map[uint8]uint8),
	}
	if err := image.importBOS(fetcher); err != nil {
		return Image{}, err
	}
	if err := image.importStrings(fetcher, descriptors); err != nil {
		return Image{}, err
	}
	if err := image.compile(descriptors, fetcher, caps); err != nil {
		return Image{}, err
	}
	if _, err := presentation.AccountEndpoints(hybridFunctions(image.Function), caps); err != nil {
		return Image{}, err
	}
	presented := presentedDescriptors(descriptors, image.Alternates)
	image.Descriptors = descriptorBlock(presented, image.Interfaces, image.Endpoints, image.Strings)
	image.StringTable = stringBlock(image.Strings, presented)
	return image, nil
}

func splitDescriptors(raw []byte) ([]byte, []byte, []descriptor, error) {
	if len(raw) < 27 || len(raw) > MaxDescriptorBytes {
		return nil, nil, nil, fmt.Errorf("%w: descriptor stream size %d", ErrMalformed, len(raw))
	}
	if raw[0] != 18 || raw[1] != 1 || raw[17] != 1 {
		return nil, nil, nil, fmt.Errorf("%w: Hybrid requires one 18-byte device descriptor and one configuration", ErrAmbiguous)
	}
	total := int(binary.LittleEndian.Uint16(raw[20:22]))
	if raw[18] != 9 || raw[19] != 2 || total < 9 || 18+total != len(raw) {
		return nil, nil, nil, fmt.Errorf("%w: configuration length %d, stream %d", ErrMalformed, total, len(raw))
	}
	config := raw[18:]
	if config[5] == 0 {
		return nil, nil, nil, fmt.Errorf("%w: configuration value is zero", ErrMalformed)
	}

	var descriptors []descriptor
	current := -1
	var alternate, class, subClass uint8
	for offset := 9; offset < len(config); {
		length := int(config[offset])
		if length < 2 || offset+length > len(config) {
			return nil, nil, nil, fmt.Errorf("%w: descriptor at %d has length %d", ErrMalformed, offset, length)
		}
		data := config[offset : offset+length]
		if data[1] == 4 {
			if length < 9 {
				return nil, nil, nil, fmt.Errorf("%w: short interface at %d", ErrMalformed, offset)
			}
			current, alternate, class, subClass = int(data[2]), data[3], data[5], data[6]
		}
		item := descriptor{offset: offset, data: slices.Clone(data)}
		if current >= 0 {
			item.interfaceNumber = uint8(current)
			item.alternate, item.class, item.subClass = alternate, class, subClass
			item.hasInterface = true
		}
		descriptors = append(descriptors, item)
		offset += length
	}
	return raw[:18], config, descriptors, nil
}

func (image *Image) importBOS(fetcher Fetcher) error {
	if binary.LittleEndian.Uint16(image.Device[2:4]) < 0x0201 {
		return nil
	}
	header, err := fetcher.Descriptor(15, 0, 0, 5)
	if err != nil {
		return fmt.Errorf("%w: read BOS header: %v", ErrMalformed, err)
	}
	if len(header) != 5 || header[0] != 5 || header[1] != 15 {
		return fmt.Errorf("%w: BOS header", ErrMalformed)
	}
	total := int(binary.LittleEndian.Uint16(header[2:4]))
	if total < 5 || total > MaxControlBytes {
		return fmt.Errorf("%w: BOS size %d", ErrMalformed, total)
	}
	bos, err := fetcher.Descriptor(15, 0, 0, total)
	if err != nil || len(bos) != total {
		return fmt.Errorf("%w: read BOS: %v", ErrMalformed, err)
	}
	set := presentation.DescriptorSet{BOS: bos}
	if err := set.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	image.BOS = slices.Clone(bos)
	if bos[4] != 0 {
		return fmt.Errorf("%w: device-level BOS capabilities need Exact mode", ErrUnsupported)
	}
	return nil
}

func (image *Image) importStrings(fetcher Fetcher, descriptors []descriptor) error {
	indices := map[uint8]bool{
		image.Device[14]:       true,
		image.Device[15]:       true,
		image.Device[16]:       true,
		image.Configuration[6]: true,
	}
	for _, item := range descriptors {
		switch item.data[1] {
		case 4:
			indices[item.data[8]] = true
		case 11:
			if len(item.data) >= 8 {
				indices[item.data[7]] = true
			}
		case 0x24:
			if len(item.data) >= 4 && item.data[2] == 0x0f {
				indices[item.data[3]] = true
			}
		}
	}
	delete(indices, 0)
	if len(indices) == 0 {
		return nil
	}
	languages, err := fetcher.Descriptor(3, 0, 0, 255)
	if err != nil || len(languages) < 4 || int(languages[0]) != len(languages) || languages[1] != 3 || len(languages)%2 != 0 {
		return fmt.Errorf("%w: string language table", ErrMalformed)
	}
	language := binary.LittleEndian.Uint16(languages[2:4])
	for index := range indices {
		raw, err := fetcher.Descriptor(3, index, language, 255)
		if err != nil {
			return fmt.Errorf("%w: string %d: %v", ErrMalformed, index, err)
		}
		value, err := decodeUSBString(raw)
		if err != nil {
			return fmt.Errorf("%w: string %d: %v", ErrMalformed, index, err)
		}
		image.Strings[index] = value
	}
	return nil
}

func decodeUSBString(raw []byte) (string, error) {
	if len(raw) < 2 || int(raw[0]) != len(raw) || raw[1] != 3 || len(raw)%2 != 0 {
		return "", errors.New("invalid USB string descriptor")
	}
	units := make([]uint16, 0, (len(raw)-2)/2)
	for offset := 2; offset < len(raw); offset += 2 {
		units = append(units, binary.LittleEndian.Uint16(raw[offset:offset+2]))
	}
	for index := 0; index < len(units); index++ {
		unit := units[index]
		if unit < 0xd800 || unit > 0xdfff {
			continue
		}
		if unit > 0xdbff || index+1 == len(units) || units[index+1] < 0xdc00 || units[index+1] > 0xdfff {
			return "", errors.New("invalid UTF-16 surrogate")
		}
		index++
	}
	value := string(utf16.Decode(units))
	if !utf8.ValidString(value) || len(value) > 126 || slices.Contains([]byte(value), byte(0)) {
		return "", errors.New("invalid USB string")
	}
	return value, nil
}

type alternateSetting struct {
	endpoints int
	widestIn  int
	widestOut int
}

// dwc2_hsotg_ep_enable compares maxpacket*mc in bytes against the depth field of
// DPTXFSIZN in words, so the widest IN packet the controller can ever seat is the
// deepest dedicated FIFO the capability table reports.
func isochronousCeiling(caps presentation.CapabilityTable) int {
	ceiling := 0
	for _, words := range caps.InFIFOWords {
		if words > ceiling {
			ceiling = words
		}
	}
	if ceiling == 0 || ceiling > 3*1024 {
		ceiling = 3 * 1024
	}
	return ceiling
}

// FunctionFS creates one endpoint file per endpoint descriptor and names it after
// the address, so two alternate settings that both carry an endpoint collide on the
// same name. MAX_ALT_SETTINGS is 2 either way. A camera that exposes six streaming
// alternates is therefore presented as its zero-bandwidth alternate 0 plus the one
// widest streaming alternate the controller can seat.
func selectAlternates(descriptors []descriptor, caps presentation.CapabilityTable) (map[uint8]uint8, error) {
	settings := make(map[uint8]map[uint8]*alternateSetting)
	for _, item := range descriptors {
		switch item.data[1] {
		case 4:
			byAlt := settings[item.data[2]]
			if byAlt == nil {
				byAlt = make(map[uint8]*alternateSetting)
				settings[item.data[2]] = byAlt
			}
			if _, exists := byAlt[item.data[3]]; exists {
				return nil, fmt.Errorf("%w: interface %d repeats alternate setting %d", ErrAmbiguous, item.data[2], item.data[3])
			}
			byAlt[item.data[3]] = &alternateSetting{endpoints: int(item.data[4])}
		case 5:
			if !item.hasInterface || len(item.data) < 7 {
				continue
			}
			setting := settings[item.interfaceNumber][item.alternate]
			if setting == nil {
				return nil, fmt.Errorf("%w: endpoint precedes an interface", ErrMalformed)
			}
			raw := binary.LittleEndian.Uint16(item.data[4:6])
			size := int(raw&0x07ff) * (int(raw>>11&3) + 1)
			if item.data[2]&0x80 != 0 {
				setting.widestIn = max(setting.widestIn, size)
			} else {
				setting.widestOut = max(setting.widestOut, size)
			}
		}
	}

	ceiling := isochronousCeiling(caps)
	chosen := make(map[uint8]uint8)
	for _, number := range slices.Sorted(maps.Keys(settings)) {
		byAlt := settings[number]
		if len(byAlt) == 1 {
			continue
		}
		zero, ok := byAlt[0]
		if !ok {
			return nil, fmt.Errorf("%w: interface %d has no alternate setting 0", ErrMalformed, number)
		}
		if zero.endpoints != 0 {
			return nil, fmt.Errorf("%w: interface %d alternate setting 0 declares %d endpoints, Hybrid presents a zero-bandwidth alternate 0 and one streaming alternate", ErrAmbiguous, number, zero.endpoints)
		}
		best, widest, found := uint8(0), -1, false
		var offered []int
		for _, alt := range slices.Sorted(maps.Keys(byAlt)) {
			if alt == 0 {
				continue
			}
			setting := byAlt[alt]
			offered = append(offered, setting.widestIn)
			if setting.widestIn > ceiling || setting.widestOut > 3*1024 {
				continue
			}
			if setting.widestIn > widest {
				best, widest, found = alt, setting.widestIn, true
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: interface %d offers IN packets %v, none of which fits the %d byte controller ceiling", ErrEndpointSize, number, offered, ceiling)
		}
		chosen[number] = best
	}
	return chosen, nil
}

func presentedDescriptors(descriptors []descriptor, alternates map[uint8]uint8) []descriptor {
	presented := make([]descriptor, 0, len(descriptors))
	for _, item := range descriptors {
		if item.hasInterface && item.alternate != 0 && item.alternate != alternates[item.interfaceNumber] {
			continue
		}
		presented = append(presented, item)
	}
	return presented
}

func (image *Image) compile(all []descriptor, fetcher Fetcher, caps presentation.CapabilityTable) error {
	alternates, err := selectAlternates(all, caps)
	if err != nil {
		return err
	}
	image.Alternates = alternates
	descriptors := presentedDescriptors(all, alternates)

	interfaces := make(map[uint8]descriptor)
	endpointOwner := make(map[uint8]uint8)
	endpointCount := make(map[uint8]int)
	inEndpoints := make(map[uint8]int)
	outEndpoints := make(map[uint8]int)
	dataInterfaces := make(map[uint8]bool)
	cdcReferences := make(map[uint8]uint8)
	cdcOwnerReferences := make(map[uint8]bool)
	cdcFeatures := make(map[uint8]uint8)
	declared := make(map[uint8]int)
	var associations [][2]uint8
	nextIn, nextOut := uint8(1), uint8(1)
	ceiling := isochronousCeiling(caps)

	for _, item := range descriptors {
		data := item.data
		switch data[1] {
		case 4:
			if data[3] == alternates[data[2]] {
				declared[data[2]] = int(data[4])
			}
			if data[3] != 0 {
				if base := interfaces[data[2]]; base.data == nil || !slices.Equal(base.data[5:8], data[5:8]) {
					return fmt.Errorf("%w: interface %d alternate setting %d changes class 0x%02x/0x%02x/0x%02x", ErrAmbiguous, data[2], data[3], data[5], data[6], data[7])
				}
				continue
			}
			if _, exists := interfaces[data[2]]; exists {
				return fmt.Errorf("%w: duplicate interface %d", ErrAmbiguous, data[2])
			}
			if label, protected := protectedClasses[data[5]]; protected {
				return fmt.Errorf("%w: interface %d is %s class", ErrProtected, data[2], label)
			}
			if !supportedClasses[data[5]] {
				return fmt.Errorf("%w: interface %d class 0x%02x", ErrUnsupported, data[2], data[5])
			}
			if data[5] == 0x02 && (data[6] != 0x02 || data[7] > 1) {
				return fmt.Errorf("%w: interface %d is not CDC ACM", ErrUnsupported, data[2])
			}
			if data[5] == 0x03 && (data[6] > 1 || data[7] > 2) {
				return fmt.Errorf("%w: interface %d HID subclass or protocol", ErrUnsupported, data[2])
			}
			if data[5] == 0x07 && (data[6] != 1 || data[7] < 1 || data[7] > 3) {
				return fmt.Errorf("%w: interface %d printer protocol", ErrUnsupported, data[2])
			}
			if data[5] == 0x0a {
				if data[6] != 0 || data[7] != 0 {
					return fmt.Errorf("%w: interface %d CDC data subclass or protocol", ErrUnsupported, data[2])
				}
				dataInterfaces[data[2]] = true
			}
			if data[5] == 0x0e && (data[6] < 1 || data[6] > 2 || data[7] > 1) {
				return fmt.Errorf("%w: interface %d video subclass %d protocol %d", ErrUnsupported, data[2], data[6], data[7])
			}
			interfaces[data[2]] = item
			image.Interfaces[data[2]] = uint8(len(image.Interfaces))
		case 5:
			if !item.hasInterface {
				return fmt.Errorf("%w: endpoint precedes an interface", ErrMalformed)
			}
			if len(data) < 7 {
				return fmt.Errorf("%w: short endpoint at %d", ErrMalformed, item.offset)
			}
			kind := data[3] & 3
			if kind != 1 && kind != 2 && kind != 3 {
				return fmt.Errorf("%w: endpoint 0x%02x type %d", ErrUnsupported, data[2], kind)
			}
			// Usage type 0 is a data endpoint. Feedback and implicit-feedback
			// endpoints come with a companion the relay has no pairing for.
			if kind == 1 && data[3]&0x30 != 0 {
				return fmt.Errorf("%w: isochronous endpoint 0x%02x has usage type %d, only a data endpoint is relayed", ErrUnsupported, data[2], data[3]>>4&3)
			}
			class, subClass := interfaces[item.interfaceNumber].data[5], interfaces[item.interfaceNumber].data[6]
			if class == 0x03 && kind != 3 || class == 0x07 && kind != 2 || class == 0x02 && kind != 3 || class == 0x0a && kind != 2 ||
				class == 0x0e && subClass == 1 && kind != 3 || class == 0x0e && subClass == 2 && kind == 3 {
				return fmt.Errorf("%w: interface %d class 0x%02x endpoint type %d", ErrUnsupported, item.interfaceNumber, class, kind)
			}
			if _, exists := endpointOwner[data[2]]; exists {
				return fmt.Errorf("%w: endpoint 0x%02x is reused", ErrAmbiguous, data[2])
			}
			rawPacket := binary.LittleEndian.Uint16(data[4:6])
			if rawPacket&0xe000 != 0 {
				return fmt.Errorf("%w: endpoint 0x%02x sets reserved wMaxPacketSize bits 0x%04x", ErrMalformed, data[2], rawPacket)
			}
			mult := uint8(rawPacket >> 11 & 3)
			if mult == 3 {
				return fmt.Errorf("%w: endpoint 0x%02x asks for 4 transactions per microframe", ErrEndpointSize, data[2])
			}
			if mult != 0 && kind != 1 && kind != 3 {
				return fmt.Errorf("%w: endpoint 0x%02x type %d asks for %d additional transactions per microframe", ErrEndpointSize, data[2], kind, mult)
			}
			packet := rawPacket & 0x07ff
			if packet == 0 || kind == 2 && packet > 512 || kind == 3 && packet > 1024 || kind == 1 && packet > 1024 {
				return fmt.Errorf("%w: endpoint 0x%02x packet %d", ErrEndpointSize, data[2], packet)
			}
			// An isochronous endpoint that lives only at alternate setting 0
			// asks every host for bandwidth in the default configuration and
			// gives the relay no SET_INTERFACE to start a stream on.
			if kind == 1 && alternates[item.interfaceNumber] == 0 {
				return fmt.Errorf("%w: isochronous endpoint 0x%02x is on interface %d alternate setting 0, which reserves no bandwidth and never starts a stream", ErrUnsupported, data[2], item.interfaceNumber)
			}
			if kind == 1 && data[2]&0x80 != 0 && int(packet)*(int(mult)+1) > ceiling {
				return fmt.Errorf("%w: isochronous endpoint 0x%02x asks for %d bytes per microframe, the controller seats at most %d", ErrEndpointSize, data[2], int(packet)*(int(mult)+1), ceiling)
			}
			if (kind == 3 || kind == 1) && (data[6] == 0 || data[6] > 16) {
				return fmt.Errorf("%w: endpoint 0x%02x interval %d", ErrEndpointSize, data[2], data[6])
			}
			mapped := nextOut
			if data[2]&0x80 != 0 {
				mapped, nextIn = 0x80|nextIn, nextIn+1
			} else {
				nextOut++
			}
			image.Endpoints[data[2]] = mapped
			image.EndpointOwners[mapped] = item.interfaceNumber
			endpointOwner[data[2]] = item.interfaceNumber
			endpointCount[item.interfaceNumber]++
			if data[2]&0x80 != 0 {
				inEndpoints[item.interfaceNumber]++
			} else {
				outEndpoints[item.interfaceNumber]++
			}
			transfer := presentation.EndpointBulk
			interval := uint8(0)
			switch kind {
			case 1:
				transfer, interval = presentation.EndpointIsochronous, data[6]
			case 3:
				transfer, interval = presentation.EndpointInterrupt, data[6]
			}
			image.Function.Endpoints = append(image.Function.Endpoints, presentation.FunctionFSEndpoint{
				SourceAddress: data[2], Address: mapped, Transfer: transfer, MaxPacket: packet, Interval: interval, Mult: mult,
			})
		case 11:
			if len(data) < 8 || data[3] == 0 {
				return fmt.Errorf("%w: interface association at %d", ErrMalformed, item.offset)
			}
			associations = append(associations, [2]uint8{data[2], data[3]})
		case 0x21:
			if !item.hasInterface {
				return fmt.Errorf("%w: HID descriptor precedes an interface", ErrMalformed)
			}
			iface := interfaces[item.interfaceNumber]
			if len(data) < 9 || iface.data == nil || iface.data[5] != 3 || data[5] != 1 || data[6] != 0x22 {
				return fmt.Errorf("%w: HID descriptor on interface %d", ErrMalformed, item.interfaceNumber)
			}
			length := int(binary.LittleEndian.Uint16(data[7:9]))
			if length == 0 || length > MaxControlBytes {
				return fmt.Errorf("%w: HID report length %d", ErrMalformed, length)
			}
			report, err := fetcher.Descriptor(0x22, 0, uint16(item.interfaceNumber), length)
			if err != nil || len(report) != length {
				return fmt.Errorf("%w: HID report on interface %d: %v", ErrMalformed, item.interfaceNumber, err)
			}
			set := presentation.DescriptorSet{HIDReports: map[string][]byte{strconv.Itoa(int(item.interfaceNumber)): report}}
			if err := set.Validate(); err != nil {
				return fmt.Errorf("%w: %v", ErrMalformed, err)
			}
			mapped := image.Interfaces[item.interfaceNumber]
			image.HIDReports[mapped] = slices.Clone(report)
			image.HIDDescriptors[mapped] = slices.Clone(data)
		case 0x24:
			if !item.hasInterface || interfaces[item.interfaceNumber].data == nil {
				return fmt.Errorf("%w: class descriptor precedes an interface", ErrAmbiguous)
			}
			switch interfaces[item.interfaceNumber].data[5] {
			case 0x02:
				if err := validateCDC(data, item.interfaceNumber, cdcReferences, cdcOwnerReferences, cdcFeatures); err != nil {
					return err
				}
			case 0x0e:
				if err := validateVideoClass(data, item.subClass, item.interfaceNumber); err != nil {
					return err
				}
			default:
				return fmt.Errorf("%w: CDC descriptor outside ACM interface", ErrAmbiguous)
			}
		case 0x25:
			if !item.hasInterface || interfaces[item.interfaceNumber].data == nil || interfaces[item.interfaceNumber].data[5] != 0x0e || len(data) < 3 {
				return fmt.Errorf("%w: descriptor type 0x%02x at %d", ErrUnsupported, data[1], item.offset)
			}
		case 0x30:
			return fmt.Errorf("%w: descriptor type 0x%02x at %d", ErrUnsupported, data[1], item.offset)
		default:
			return fmt.Errorf("%w: descriptor type 0x%02x at %d", ErrUnsupported, data[1], item.offset)
		}
	}

	if len(interfaces) == 0 || len(interfaces) > MaxInterfaces || len(interfaces) != int(image.Configuration[4]) {
		return fmt.Errorf("%w: %d interfaces, header says %d", ErrAmbiguous, len(interfaces), image.Configuration[4])
	}
	if len(image.Function.Endpoints) == 0 || len(image.Function.Endpoints) > MaxEndpoints {
		return fmt.Errorf("%w: %d endpoints", ErrAmbiguous, len(image.Function.Endpoints))
	}
	for number, iface := range interfaces {
		if endpointCount[number] != declared[number] {
			return fmt.Errorf("%w: interface %d declares %d endpoints, has %d", ErrMalformed, number, declared[number], endpointCount[number])
		}
		switch iface.data[5] {
		case 0x02:
			if inEndpoints[number] > 1 || outEndpoints[number] != 0 || cdcFeatures[number]&0x05 != 0x05 || !cdcOwnerReferences[number] {
				return fmt.Errorf("%w: CDC ACM interface %d layout", ErrAmbiguous, number)
			}
		case 0x03:
			if _, ok := image.HIDReports[image.Interfaces[number]]; !ok {
				return fmt.Errorf("%w: interface %d has no HID report", ErrMalformed, number)
			}
			if inEndpoints[number] != 1 || outEndpoints[number] > 1 {
				return fmt.Errorf("%w: HID interface %d endpoint layout", ErrAmbiguous, number)
			}
		case 0x07:
			wantIn := 0
			if iface.data[7] != 1 {
				wantIn = 1
			}
			if inEndpoints[number] != wantIn || outEndpoints[number] != 1 {
				return fmt.Errorf("%w: printer interface %d endpoint layout", ErrAmbiguous, number)
			}
		case 0x0a:
			if inEndpoints[number] != 1 || outEndpoints[number] != 1 {
				return fmt.Errorf("%w: CDC data interface %d endpoint layout", ErrAmbiguous, number)
			}
		case 0x0e:
			if iface.data[6] == 1 && (inEndpoints[number] > 1 || outEndpoints[number] != 0) {
				return fmt.Errorf("%w: video control interface %d endpoint layout", ErrAmbiguous, number)
			}
			if iface.data[6] == 2 && inEndpoints[number]+outEndpoints[number] > 1 {
				return fmt.Errorf("%w: video streaming interface %d endpoint layout", ErrAmbiguous, number)
			}
		}
	}
	for number := range dataInterfaces {
		if _, ok := cdcReferences[number]; !ok {
			return fmt.Errorf("%w: CDC data interface %d has no ACM reference", ErrAmbiguous, number)
		}
	}
	for number := range cdcReferences {
		iface, ok := interfaces[number]
		if !ok || iface.data[5] != 0x0a {
			return fmt.Errorf("%w: CDC reference to interface %d", ErrAmbiguous, number)
		}
	}
	associated := make(map[uint8]bool)
	for _, association := range associations {
		first, count := association[0], association[1]
		end := int(first) + int(count)
		if end > 256 {
			return fmt.Errorf("%w: interface association range", ErrAmbiguous)
		}
		for value := int(first); value < end; value++ {
			number := uint8(value)
			if _, ok := interfaces[number]; !ok || associated[number] {
				return fmt.Errorf("%w: interface association member %d", ErrAmbiguous, number)
			}
			associated[number] = true
		}
		firstInterface := interfaces[first].data
		association := associationDescriptor(descriptors, first)
		// USB Video names the function SC_VIDEO_INTERFACE_COLLECTION while the
		// first interface of the collection is SC_VIDEOCONTROL, so a video
		// function is the one association whose subclass legitimately differs
		// from the interface it starts at.
		wantSubClass := firstInterface[6]
		if association[4] == 0x0e {
			wantSubClass = 0x03
		}
		if association[4] != firstInterface[5] || association[5] != wantSubClass || association[6] != firstInterface[7] {
			return fmt.Errorf("%w: association at interface %d is 0x%02x/0x%02x/0x%02x, interface %d is 0x%02x/0x%02x/0x%02x",
				ErrAmbiguous, first, association[4], association[5], association[6], first, firstInterface[5], firstInterface[6], firstInterface[7])
		}
	}
	image.Function.Interfaces = uint8(len(interfaces))
	return image.Function.Validate()
}

func validateCDC(data []byte, owner uint8, references map[uint8]uint8, ownerReferences map[uint8]bool, features map[uint8]uint8) error {
	if len(data) < 3 {
		return fmt.Errorf("%w: short CDC descriptor", ErrMalformed)
	}
	feature := uint8(0)
	switch data[2] {
	case 0x00:
		if len(data) != 5 {
			return fmt.Errorf("%w: CDC header length", ErrMalformed)
		}
		feature = 0x01
	case 0x02:
		if len(data) != 4 {
			return fmt.Errorf("%w: CDC ACM length", ErrMalformed)
		}
		feature = 0x04
	case 0x01:
		if len(data) != 5 {
			return fmt.Errorf("%w: CDC call management length", ErrMalformed)
		}
		feature = 0x02
		if err := addCDCReference(owner, data[4], references, ownerReferences); err != nil {
			return err
		}
	case 0x06:
		if len(data) < 5 || data[3] != owner {
			return fmt.Errorf("%w: CDC union master", ErrAmbiguous)
		}
		feature = 0x08
		for _, number := range data[4:] {
			if err := addCDCReference(owner, number, references, ownerReferences); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: CDC subtype 0x%02x", ErrUnsupported, data[2])
	}
	if features[owner]&feature != 0 {
		return fmt.Errorf("%w: duplicate CDC subtype 0x%02x", ErrAmbiguous, data[2])
	}
	features[owner] |= feature
	return nil
}

func validateVideoClass(data []byte, subClass uint8, number uint8) error {
	if len(data) < 3 {
		return fmt.Errorf("%w: short video class descriptor on interface %d", ErrMalformed, number)
	}
	if subClass == 1 && data[2] == 0x01 {
		if len(data) < 12 || len(data) != 12+int(data[11]) {
			return fmt.Errorf("%w: video control header on interface %d is %d bytes for %d collected interfaces", ErrMalformed, number, len(data), data[11])
		}
	}
	if subClass == 2 && (data[2] == 0x01 || data[2] == 0x02) && len(data) < 7 {
		return fmt.Errorf("%w: video streaming header on interface %d is %d bytes", ErrMalformed, number, len(data))
	}
	return nil
}

func addCDCReference(owner uint8, target uint8, references map[uint8]uint8, ownerReferences map[uint8]bool) error {
	if previous, exists := references[target]; exists && previous != owner {
		return fmt.Errorf("%w: CDC data interface %d has multiple owners", ErrAmbiguous, target)
	}
	references[target] = owner
	ownerReferences[owner] = true
	return nil
}

func associationDescriptor(descriptors []descriptor, first uint8) []byte {
	for _, item := range descriptors {
		if item.data[1] == 11 && item.data[2] == first {
			return item.data
		}
	}
	return nil
}

func hybridFunctions(function presentation.FunctionFS) []presentation.Function {
	return []presentation.Function{
		{Kind: presentation.FunctionHID, Instance: "GS0", HID: &presentation.HIDFunction{Protocol: 1, ReportLength: 8, DevNodeIndex: 0}},
		{Kind: presentation.FunctionHID, Instance: "GS1", HID: &presentation.HIDFunction{Protocol: 2, ReportLength: 4, DevNodeIndex: 1}},
		{Kind: presentation.FunctionFFS, Instance: "hybrid", FFS: &function},
	}
}

func descriptorBlock(descriptors []descriptor, interfaces map[uint8]uint8, endpoints map[uint8]uint8, strings map[uint8]string) []byte {
	hs := rewriteDescriptors(descriptors, interfaces, endpoints, stringOrder(strings, descriptors), false)
	fs := rewriteDescriptors(descriptors, interfaces, endpoints, stringOrder(strings, descriptors), true)
	count := uint32(len(descriptors))
	length := 20 + len(fs) + len(hs)
	out := make([]byte, 20, length)
	binary.LittleEndian.PutUint32(out[0:4], 3)
	binary.LittleEndian.PutUint32(out[4:8], uint32(length))
	binary.LittleEndian.PutUint32(out[8:12], 1|2|16)
	binary.LittleEndian.PutUint32(out[12:16], count)
	binary.LittleEndian.PutUint32(out[16:20], count)
	out = append(out, fs...)
	out = append(out, hs...)
	return out
}

func rewriteDescriptors(descriptors []descriptor, interfaces map[uint8]uint8, endpoints map[uint8]uint8, strings map[uint8]uint8, fullSpeed bool) []byte {
	var out []byte
	for _, item := range descriptors {
		data := slices.Clone(item.data)
		switch data[1] {
		case 4:
			data[2], data[8] = interfaces[data[2]], strings[data[8]]
			if data[3] != 0 {
				data[3] = 1
			}
		case 5:
			data[2] = endpoints[data[2]]
			// Full speed has no high-bandwidth transactions, so the mult bits go
			// with the clamp rather than surviving into the FS descriptor.
			if fullSpeed {
				packet := binary.LittleEndian.Uint16(data[4:6]) & 0x07ff
				limit := uint16(64)
				if data[3]&3 == 1 {
					limit = 1023
				}
				if packet > limit {
					packet = limit
				}
				binary.LittleEndian.PutUint16(data[4:6], packet)
			}
		case 11:
			data[2], data[7] = interfaces[data[2]], strings[data[7]]
		case 0x24:
			if item.class == 0x0e {
				if item.subClass == 1 && data[2] == 0x01 && len(data) >= 12 {
					for i := 12; i < len(data); i++ {
						data[i] = interfaces[data[i]]
					}
				}
				if item.subClass == 2 && (data[2] == 0x01 || data[2] == 0x02) && len(data) >= 7 && data[6] != 0 {
					data[6] = endpoints[data[6]]
				}
				break
			}
			switch data[2] {
			case 0x01:
				data[4] = interfaces[data[4]]
			case 0x06:
				for i := 3; i < len(data); i++ {
					data[i] = interfaces[data[i]]
				}
			}
		}
		out = append(out, data...)
	}
	return out
}

func stringOrder(values map[uint8]string, descriptors []descriptor) map[uint8]uint8 {
	order := make(map[uint8]uint8)
	add := func(index uint8) {
		if index != 0 && order[index] == 0 {
			order[index] = uint8(len(order) + 1)
		}
	}
	for _, item := range descriptors {
		switch item.data[1] {
		case 4:
			add(item.data[8])
		case 11:
			add(item.data[7])
		}
	}
	return order
}

func stringBlock(values map[uint8]string, descriptors []descriptor) []byte {
	order := stringOrder(values, descriptors)
	// __ffs_data_got_strings rejects a language that carries no strings, so a
	// source device with every string index zero needs a header with both
	// counts zero and no language at all. Declaring 0x0409 anyway is EINVAL on
	// the ep0 write and takes the whole session down with it.
	if len(order) == 0 {
		out := make([]byte, 16)
		binary.LittleEndian.PutUint32(out[0:4], 2)
		binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)))
		return out
	}
	length := 18
	for source := range order {
		length += len(values[source]) + 1
	}
	out := make([]byte, 18, length)
	binary.LittleEndian.PutUint32(out[0:4], 2)
	binary.LittleEndian.PutUint32(out[4:8], uint32(length))
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(order)))
	binary.LittleEndian.PutUint32(out[12:16], 1)
	binary.LittleEndian.PutUint16(out[16:18], 0x0409)
	byTarget := make([]uint8, len(order)+1)
	for source, target := range order {
		byTarget[target] = source
	}
	for target := 1; target < len(byTarget); target++ {
		out = append(out, values[byTarget[target]]...)
		out = append(out, 0)
	}
	return out
}
