package presentation

import (
	"errors"
	"fmt"
	"math/bits"
	"slices"
)

var (
	ErrUnknownFunction     = errors.New("unknown function")
	ErrFunctionUnavailable = errors.New("function unavailable")
	ErrEndpointBudget      = errors.New("endpoint budget exceeded")
	ErrFIFOBudget          = errors.New("fifo budget exceeded")
)

type EndpointUse struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

type FIFOAssignment map[string][]int

func (u EndpointUse) add(caps FunctionCaps) EndpointUse {
	return EndpointUse{In: u.In + caps.InEPs, Out: u.Out + caps.OutEPs}
}

func (u EndpointUse) Headroom(table CapabilityTable) EndpointUse {
	return EndpointUse{In: table.MaxInEndpoints - u.In, Out: table.MaxOutEndpoints - u.Out}
}

func AccountEndpoints(functions []Function, table CapabilityTable) (EndpointUse, error) {
	var used EndpointUse

	for _, function := range functions {
		kind, instance := function.Kind, function.Instance
		name := string(kind) + "." + instance

		caps, ok := table.Functions[kind]
		if !ok {
			return used, fmt.Errorf("%w: %s rejected by capability table %s", ErrUnknownFunction, name, table.Source)
		}
		if !caps.Available {
			return used, fmt.Errorf("%w: %s rejected by capability table %s", ErrFunctionUnavailable, name, table.Source)
		}

		if function.Kind == FunctionFFS {
			caps.InEPs, caps.OutEPs = 0, 0
			for _, endpoint := range function.FFS.Endpoints {
				if endpoint.Address&0x80 != 0 {
					caps.InEPs++
				} else {
					caps.OutEPs++
				}
			}
		}
		before := used
		used = used.add(caps)
		if used.In > table.MaxInEndpoints {
			return used, fmt.Errorf("%w: %s needs %d IN endpoints, %d of %d in use, rejected by capability table %s",
				ErrEndpointBudget, name, caps.InEPs, used.In, table.MaxInEndpoints, table.Source)
		}
		if used.Out > table.MaxOutEndpoints {
			return used, fmt.Errorf("%w: %s needs %d OUT endpoints, %d of %d in use, rejected by capability table %s",
				ErrEndpointBudget, name, caps.OutEPs, used.Out, table.MaxOutEndpoints, table.Source)
		}
		if function.Kind == FunctionFFS && len(table.InFIFOWords) != 0 {
			in := before.In
			for _, endpoint := range function.FFS.Endpoints {
				if endpoint.Address&0x80 == 0 {
					continue
				}
				words := 0
				if in < len(table.InFIFOWords) {
					words = table.InFIFOWords[in]
				}
				if int(endpoint.MaxPacket) > words {
					return used, fmt.Errorf("%w: %s endpoint 0x%02x packet %d exceeds IN FIFO %d at slot %d, rejected by capability table %s",
						ErrEndpointBudget, name, endpoint.Address, endpoint.MaxPacket, words, in+1, table.Source)
				}
				in++
			}
		}
	}
	if _, err := SeatFIFOs(functions, table); err != nil {
		return used, err
	}
	return used, nil
}

func SeatFIFOs(functions []Function, table CapabilityTable) (FIFOAssignment, error) {
	assigned := make(FIFOAssignment)
	if len(table.InFIFOWords) == 0 {
		return assigned, nil
	}
	free := slices.Clone(table.InFIFOWords)
	slices.Sort(free)
	for _, function := range functions {
		name := functionName(function)
		for _, packet := range inPackets(function, table.Functions[function.Kind]) {
			words := (packet + 3) / 4
			index := -1
			for i, depth := range free {
				if depth >= words {
					index = i
					break
				}
			}
			if index < 0 {
				return assigned, fmt.Errorf("%w: %s needs a fifo of at least %d words for a %d byte IN packet, free fifos are %v, rejected by capability table %s",
					ErrFIFOBudget, name, words, packet, free, table.Source)
			}
			assigned[name] = append(assigned[name], free[index])
			free = append(free[:index], free[index+1:]...)
		}
	}
	return assigned, nil
}

func inPackets(function Function, caps FunctionCaps) []int {
	switch function.Kind {
	case FunctionHID:
		if function.HID != nil {
			return []int{int(function.HID.ReportLength)}
		}
	case FunctionUVC:
		if function.Video != nil {
			return []int{16, int(function.Video.StreamingMaxPacket)}
		}
	case FunctionUAC2:
		if function.Audio != nil {
			channels := bits.OnesCount32(function.Audio.PChannelMask)
			packet := channels * int(function.Audio.PSampleSize) * int((function.Audio.PSampleRate+999)/1000)
			return []int{packet}
		}
	}
	return slices.Clone(caps.INPackets)
}
