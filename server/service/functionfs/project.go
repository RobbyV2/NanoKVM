package functionfs

import (
	"encoding/binary"
	"fmt"
	"slices"
)

func Project(device, configuration []byte, selected []uint8) ([]byte, error) {
	if len(device) != 18 || device[0] != 18 || device[1] != 1 {
		return nil, fmt.Errorf("%w: device descriptor", ErrMalformed)
	}
	if len(configuration) < 9 || configuration[0] != 9 || configuration[1] != 2 || int(binary.LittleEndian.Uint16(configuration[2:4])) != len(configuration) {
		return nil, fmt.Errorf("%w: configuration descriptor", ErrMalformed)
	}
	if len(selected) == 0 || len(selected) > MaxInterfaces {
		return nil, fmt.Errorf("%w: selected interface count %d", ErrMalformed, len(selected))
	}
	wanted := make(map[uint8]bool, len(selected))
	for _, number := range selected {
		if wanted[number] {
			return nil, fmt.Errorf("%w: duplicate interface %d", ErrAmbiguous, number)
		}
		wanted[number] = true
	}

	type item struct {
		data        []byte
		owner       uint8
		hasOwner    bool
		association []uint8
	}
	items := []item{{data: slices.Clone(configuration[:9])}}
	found := make(map[uint8]bool)
	var current uint8
	hasCurrent := false
	for offset := 9; offset < len(configuration); {
		length := int(configuration[offset])
		if length < 2 || offset+length > len(configuration) {
			return nil, fmt.Errorf("%w: descriptor at %d", ErrMalformed, offset)
		}
		data := slices.Clone(configuration[offset : offset+length])
		entry := item{data: data, owner: current, hasOwner: hasCurrent}
		switch data[1] {
		case 4:
			if len(data) < 9 {
				return nil, fmt.Errorf("%w: interface descriptor", ErrMalformed)
			}
			current, hasCurrent = data[2], true
			entry.owner, entry.hasOwner = current, true
			if data[3] == 0 {
				found[current] = true
			} else if wanted[current] {
				return nil, fmt.Errorf("%w: interface %d has alternate setting %d", ErrAmbiguous, current, data[3])
			}
			label, protected := protectedClasses[data[5]]
			if data[5] == 0x03 {
				label, protected = "human interface", true
			}
			if protected && wanted[current] {
				return nil, fmt.Errorf("%w: interface %d is %s class", ErrProtected, current, label)
			}
		case 5:
			if len(data) < 7 || !hasCurrent {
				return nil, fmt.Errorf("%w: endpoint without interface", ErrMalformed)
			}
			if data[3]&3 == 1 && wanted[current] {
				return nil, fmt.Errorf("%w: endpoint 0x%02x", ErrIsochronous, data[2])
			}
		case 11:
			if len(data) < 8 || data[3] == 0 {
				return nil, fmt.Errorf("%w: interface association", ErrMalformed)
			}
			first, count := int(data[2]), int(data[3])
			if first+count > 256 {
				return nil, fmt.Errorf("%w: interface association range", ErrMalformed)
			}
			entry.hasOwner = false
			for n := first; n < first+count; n++ {
				entry.association = append(entry.association, uint8(n))
			}
		}
		items = append(items, entry)
		offset += length
	}
	for number := range wanted {
		if !found[number] {
			return nil, fmt.Errorf("%w: interface %d is missing", ErrMalformed, number)
		}
	}

	projected := slices.Clone(configuration[:9])
	for _, entry := range items[1:] {
		keep := !entry.hasOwner
		if len(entry.association) != 0 {
			keep = wanted[entry.association[0]]
			for _, number := range entry.association[1:] {
				if wanted[number] != keep {
					return nil, fmt.Errorf("%w: partial interface association", ErrAmbiguous)
				}
			}
		} else if entry.hasOwner {
			keep = wanted[entry.owner]
		}
		if keep {
			projected = append(projected, entry.data...)
		}
	}
	projected[4] = uint8(len(wanted))
	binary.LittleEndian.PutUint16(projected[2:4], uint16(len(projected)))
	runtimeDevice := slices.Clone(device)
	runtimeDevice[17] = 1
	return append(runtimeDevice, projected...), nil
}
