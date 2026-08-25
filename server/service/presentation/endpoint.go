package presentation

import (
	"errors"
	"fmt"
	"math/bits"
	"slices"
	"strings"
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

		if function.Kind == FunctionUVC && function.Video != nil && !function.Video.interruptEndpoint() {
			if !caps.Attributes[UVCAttrInterruptEP] {
				return used, fmt.Errorf("%w: %s declines the control interrupt endpoint, which needs %s on the uvc function, rejected by capability table %s",
					ErrFunctionUnavailable, name, UVCAttrInterruptEP, table.Source)
			}
			caps.InEPs--
		}
		if function.Kind == FunctionUVC && function.Video != nil && function.Video.HostName != nil && !caps.Attributes[UVCAttrFunctionName] {
			return used, fmt.Errorf("%w: %s names itself to the host, which needs a writable %s on the uvc function, rejected by capability table %s",
				ErrFunctionUnavailable, name, UVCAttrFunctionName, table.Source)
		}
		if function.Kind == FunctionUAC2 && function.Audio != nil && function.Audio.HostName != nil && !caps.Attributes[UAC2AttrFunctionName] {
			return used, fmt.Errorf("%w: %s names itself to the host, which needs a writable %s on the uac2 function, rejected by capability table %s",
				ErrFunctionUnavailable, name, UAC2AttrFunctionName, table.Source)
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
		used = used.add(caps)
		if used.In > table.MaxInEndpoints {
			return used, fmt.Errorf("%w: %s needs %d IN endpoints, %d of %d in use, rejected by capability table %s%s",
				ErrEndpointBudget, name, caps.InEPs, used.In, table.MaxInEndpoints, table.Source,
				budgetAdvice(functions, function, table, used.In-table.MaxInEndpoints))
		}
		if used.Out > table.MaxOutEndpoints {
			return used, fmt.Errorf("%w: %s needs %d OUT endpoints, %d of %d in use, rejected by capability table %s",
				ErrEndpointBudget, name, caps.OutEPs, used.Out, table.MaxOutEndpoints, table.Source)
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

// Six IN endpoints is a hard silicon ceiling, so a refusal can never be answered
// with more endpoints and is only useful if it says what is spending them and
// which trade would fit. Only the trades this plan can actually make are named,
// and each is worth what removing it would really return.
func budgetAdvice(functions []Function, refused Function, table CapabilityTable, short int) string {
	var held []string
	var hid, net, disk int

	for _, function := range functions {
		if function.Kind == refused.Kind && function.Instance == refused.Instance {
			break
		}
		cost := table.Functions[function.Kind].InEPs
		if function.Kind == FunctionUVC && function.Video != nil && !function.Video.interruptEndpoint() {
			cost--
		}
		if cost <= 0 {
			continue
		}
		held = append(held, fmt.Sprintf("%s %d", functionName(function), cost))
		switch {
		case function.Kind == FunctionHID:
			hid++
		case function.Kind.isNet():
			net += cost
		case function.Kind == FunctionMassStorage:
			disk += cost
		}
	}
	if len(held) == 0 {
		return ""
	}

	var frees []string
	if hid > 1 {
		frees = append(frees, fmt.Sprintf("putting the %d HID interfaces on one frees %d", hid, hid-1))
	}
	if net > 0 {
		frees = append(frees, fmt.Sprintf("turning off USB networking frees %d", net))
	}
	if disk > 0 {
		frees = append(frees, fmt.Sprintf("turning off the virtual disk frees %d", disk))
	}

	advice := "; in use by " + strings.Join(held, ", ")
	if len(frees) == 0 {
		return advice
	}
	return fmt.Sprintf("%s; %d more needed, and %s", advice, short, strings.Join(frees, ", "))
}

func inPackets(function Function, caps FunctionCaps) []int {
	switch function.Kind {
	case FunctionHID:
		if function.HID != nil {
			return []int{int(function.HID.ReportLength)}
		}
	case FunctionUVC:
		if function.Video != nil {
			// A high-bandwidth isochronous endpoint moves maxburst+1 packets per
			// microframe and the controller needs a FIFO deep enough for all of
			// them, so the burst has to be costed here or the seating passes a
			// depth the kernel then refuses at ep_enable.
			streaming := int(function.Video.StreamingMaxPacket) * (int(function.Video.StreamingMaxBurst) + 1)
			if !function.Video.interruptEndpoint() {
				return []int{streaming}
			}
			return []int{16, streaming}
		}
	case FunctionUAC2:
		if function.Audio != nil {
			channels := bits.OnesCount32(function.Audio.PChannelMask)
			packet := channels * int(function.Audio.PSampleSize) * int((function.Audio.PSampleRate+999)/1000)
			return []int{packet}
		}
	case FunctionFFS:
		if function.FFS != nil {
			packets := make([]int, 0, len(function.FFS.Endpoints))
			for _, endpoint := range function.FFS.Endpoints {
				if endpoint.Address&0x80 != 0 {
					packets = append(packets, int(endpoint.MaxPacket)*(int(endpoint.Mult)+1))
				}
			}
			return packets
		}
	}
	return slices.Clone(caps.INPackets)
}
