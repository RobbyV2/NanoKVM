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

		used = used.add(caps)
		if used.In > table.MaxInEndpoints {
			return used, fmt.Errorf("%w: %s needs %d IN endpoints, %d of %d in use, rejected by capability table %s",
				ErrEndpointBudget, name, caps.InEPs, used.In, table.MaxInEndpoints, table.Source)
		}
		if used.Out > table.MaxOutEndpoints {
			return used, fmt.Errorf("%w: %s needs %d OUT endpoints, %d of %d in use, rejected by capability table %s",
				ErrEndpointBudget, name, caps.OutEPs, used.Out, table.MaxOutEndpoints, table.Source)
		}
	}
	return used, nil
}
