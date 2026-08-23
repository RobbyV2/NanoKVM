package presentation

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownFunction     = errors.New("unknown function")
	ErrFunctionUnavailable = errors.New("function unavailable")
	ErrEndpointBudget      = errors.New("endpoint budget exceeded")
)

type EndpointUse struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

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
	return used, nil
}
