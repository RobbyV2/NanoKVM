package functionfs

import (
	"encoding/binary"
	"errors"
	"fmt"
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
	ErrIsochronous  = errors.New("functionfs: isochronous endpoint")
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
	Function       presentation.FunctionFS
	Descriptors    []byte
	StringTable    []byte
}

type descriptor struct {
	offset          int
	data            []byte
	interfaceNumber uint8
	hasInterface    bool
}

var protectedClasses = map[uint8]string{
	0x01: "audio",
	0x08: "mass storage",
	0x09: "hub",
	0x0b: "smart card",
	0x0d: "content security",
	0x0e: "video",
	0x10: "audio/video",
	0xe0: "wireless controller",
}

var supportedClasses = map[uint8]bool{
	0x02: true,
	0x03: true,
	0x07: true,
	0x0a: true,
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
	if device[4] != 0 {
		return Image{}, fmt.Errorf("%w: device-level class 0x%02x needs Exact mode", ErrUnsupported, device[4])
	}

	image := Image{
		Device:         slices.Clone(device),
		Configuration:  slices.Clone(config),
		Strings:        make(map[uint8]string),
		HIDReports:     make(map[uint8][]byte),
		HIDDescriptors: make(map[uint8][]byte),
		Interfaces:     make(map[uint8]uint8),
		Endpoints:      make(map[uint8]uint8),
	}
	if err := image.importBOS(fetcher); err != nil {
		return Image{}, err
	}
	if err := image.importStrings(fetcher, descriptors); err != nil {
		return Image{}, err
	}
	if err := image.compile(descriptors, fetcher); err != nil {
		return Image{}, err
	}
	if _, err := presentation.AccountEndpoints(hybridFunctions(image.Function), caps); err != nil {
		return Image{}, err
	}
	image.Descriptors = descriptorBlock(image.Configuration, descriptors, image.Interfaces, image.Endpoints, image.Strings)
	image.StringTable = stringBlock(image.Strings, descriptors)
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
			current = int(data[2])
		}
		item := descriptor{offset: offset, data: slices.Clone(data)}
		if current >= 0 {
			item.interfaceNumber = uint8(current)
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

func (image *Image) compile(descriptors []descriptor, fetcher Fetcher) error {
	interfaces := make(map[uint8]descriptor)
	endpointOwner := make(map[uint8]uint8)
	endpointCount := make(map[uint8]int)
	inEndpoints := make(map[uint8]int)
	outEndpoints := make(map[uint8]int)
	dataInterfaces := make(map[uint8]bool)
	cdcReferences := make(map[uint8]uint8)
	cdcOwnerReferences := make(map[uint8]bool)
	cdcFeatures := make(map[uint8]uint8)
	var associations [][2]uint8
	nextIn, nextOut := uint8(1), uint8(1)

	for _, item := range descriptors {
		data := item.data
		switch data[1] {
		case 4:
			if data[3] != 0 {
				return fmt.Errorf("%w: interface %d has alternate setting %d", ErrAmbiguous, data[2], data[3])
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
			if kind == 1 {
				return fmt.Errorf("%w: endpoint 0x%02x", ErrIsochronous, data[2])
			}
			if kind != 2 && kind != 3 {
				return fmt.Errorf("%w: endpoint 0x%02x type %d", ErrUnsupported, data[2], kind)
			}
			class := interfaces[item.interfaceNumber].data[5]
			if class == 0x03 && kind != 3 || class == 0x07 && kind != 2 || class == 0x02 && kind != 3 || class == 0x0a && kind != 2 {
				return fmt.Errorf("%w: interface %d class 0x%02x endpoint type %d", ErrUnsupported, item.interfaceNumber, class, kind)
			}
			if _, exists := endpointOwner[data[2]]; exists {
				return fmt.Errorf("%w: endpoint 0x%02x is reused", ErrAmbiguous, data[2])
			}
			rawPacket := binary.LittleEndian.Uint16(data[4:6])
			if rawPacket&0xf800 != 0 {
				return fmt.Errorf("%w: endpoint 0x%02x uses high-bandwidth transactions", ErrEndpointSize, data[2])
			}
			packet := rawPacket & 0x07ff
			if packet == 0 || kind == 2 && packet > 512 || kind == 3 && packet > 1024 {
				return fmt.Errorf("%w: endpoint 0x%02x packet %d", ErrEndpointSize, data[2], packet)
			}
			if kind == 3 && (data[6] == 0 || data[6] > 16) {
				return fmt.Errorf("%w: endpoint 0x%02x interval %d", ErrEndpointSize, data[2], data[6])
			}
			mapped := nextOut
			if data[2]&0x80 != 0 {
				mapped, nextIn = 0x80|nextIn, nextIn+1
			} else {
				nextOut++
			}
			image.Endpoints[data[2]] = mapped
			endpointOwner[data[2]] = item.interfaceNumber
			endpointCount[item.interfaceNumber]++
			if data[2]&0x80 != 0 {
				inEndpoints[item.interfaceNumber]++
			} else {
				outEndpoints[item.interfaceNumber]++
			}
			transfer := presentation.EndpointBulk
			interval := uint8(0)
			if kind == 3 {
				transfer, interval = presentation.EndpointInterrupt, data[6]
			}
			image.Function.Endpoints = append(image.Function.Endpoints, presentation.FunctionFSEndpoint{
				SourceAddress: data[2], Address: mapped, Transfer: transfer, MaxPacket: packet, Interval: interval,
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
			if !item.hasInterface || interfaces[item.interfaceNumber].data[5] != 0x02 {
				return fmt.Errorf("%w: CDC descriptor outside ACM interface", ErrAmbiguous)
			}
			if err := validateCDC(data, item.interfaceNumber, cdcReferences, cdcOwnerReferences, cdcFeatures); err != nil {
				return err
			}
		case 0x25, 0x30:
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
		if endpointCount[number] != int(iface.data[4]) {
			return fmt.Errorf("%w: interface %d declares %d endpoints, has %d", ErrMalformed, number, iface.data[4], endpointCount[number])
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
		for offset := 4; offset <= 6; offset++ {
			if associationDescriptor(descriptors, first)[offset] != firstInterface[offset+1] {
				return fmt.Errorf("%w: interface association class mismatch", ErrAmbiguous)
			}
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

func descriptorBlock(_ []byte, descriptors []descriptor, interfaces map[uint8]uint8, endpoints map[uint8]uint8, strings map[uint8]string) []byte {
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
		case 5:
			data[2] = endpoints[data[2]]
			if fullSpeed {
				packet := binary.LittleEndian.Uint16(data[4:6]) & 0x07ff
				limit := uint16(64)
				if data[3]&3 == 3 && limit > packet {
					limit = packet
				}
				if packet > limit {
					binary.LittleEndian.PutUint16(data[4:6], limit)
				}
			}
		case 11:
			data[2], data[7] = interfaces[data[2]], strings[data[7]]
		case 0x24:
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
