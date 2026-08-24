package presentation

import (
	"slices"
	"time"
)

const (
	netInstance  = "usb0"
	diskInstance = "disk0"
	lunDir       = "lun.0"

	diskFunctionName = string(FunctionMassStorage) + "." + diskInstance

	// What a gadget with no orphaned net function left over from an earlier
	// apply names its NIC, and nothing stronger than that: gether_setup asks
	// for "usb%d" and the kernel fills in the first free number, which is not
	// necessarily zero. Manager.NIC reads functions/<net-fn>/ifname for the
	// real answer and falls back to this only where the attribute is absent.
	GadgetNIC = netInstance
)

type Snapshot struct {
	Active string   `json:"active"`
	Mode   string   `json:"mode"`
	Linked []string `json:"linked"`

	// Which controller the gadget holds and what the host on the other end has
	// made of it. Every mutator already reads the UDC attribute to prove its
	// bind took; state and speed are what say whether a host is there at all.
	UDC UDCStatus `json:"udc"`

	// ResetPHY and the ffs recovery path take the controller away from a host
	// that has already enumerated it, which no rebind on this side puts back.
	PendingPowerCycle bool `json:"pending_power_cycle"`

	// A failed apply rolls back, so the linkage above is the one that was
	// already there and carries no trace of the attempt. Without this the only
	// record of it is an HTTP response the operator has already dismissed.
	LastError *ApplyFailure `json:"last_error,omitempty"`

	// What the active profile costs against the controller's endpoint budget,
	// and what is left of it. The compiler already accounts for this to reject
	// a profile that does not fit; reporting it is what lets a caller see how
	// close to the ceiling the gadget is before it asks for another function.
	Endpoints EndpointUse    `json:"endpoints"`
	Headroom  EndpointUse    `json:"headroom"`
	FIFOs     FIFOAssignment `json:"fifos,omitempty"`

	// How the gadget the kernel bound differs from the profile named above,
	// absent when it does not. It is re-derived on every read rather than
	// remembered, because S03usbdev rebuilds the gadget from scratch on every
	// boot and any flag written about it goes stale the moment it does.
	Diverged *Divergence `json:"diverged,omitempty"`
}

type UDCStatus struct {
	Name  string `json:"name"`
	Bound bool   `json:"bound"`
	State string `json:"state,omitempty"`
	Speed string `json:"speed,omitempty"`
}

type ApplyFailure struct {
	Profile string    `json:"profile"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// Linkage is read back through the function's own attribute rather than from a
// /boot sentinel, so a gadget the NCM branch built stops reporting the RNDIS
// toggle as on (S03usbdev:53, H10).
var functionProbeAttr = map[FunctionKind]string{
	FunctionHID:         "protocol",
	FunctionNCM:         "dev_addr",
	FunctionRNDIS:       "dev_addr",
	FunctionMassStorage: lunDir + "/file",
	FunctionUVC:         "streaming_maxpacket",
	FunctionUAC2:        "p_srate",
}

func knownFunctions() []Function {
	functions := []Function{
		{Kind: FunctionNCM, Instance: netInstance},
		{Kind: FunctionRNDIS, Instance: netInstance},
	}
	for _, instance := range hidInstances {
		functions = append(functions, Function{Kind: FunctionHID, Instance: instance})
	}
	return append(functions, Function{Kind: FunctionMassStorage, Instance: diskInstance})
}

func functionName(f Function) string {
	return string(f.Kind) + "." + f.Instance
}

// Only the linkage under configs/c.1 is probed. What exists under functions/*
// is a different question, and an add-only transaction never removes a function
// directory, so the answer is "everything ever built" rather than anything a
// caller can act on: probing it cost one configfs read per known function and
// told nobody anything.
func readSnapshot(ops Ops, extra []Function) Snapshot {
	var snapshot Snapshot

	seen := make(map[string]bool)
	for _, function := range append(knownFunctions(), extra...) {
		attr, ok := functionProbeAttr[function.Kind]
		name := functionName(function)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true

		if readable(ops, configPrefix+"/"+name+"/"+attr) {
			snapshot.Linked = append(snapshot.Linked, name)
		}
	}
	return snapshot
}

func readable(ops Ops, rel string) bool {
	_, err := ops.ReadFile(rel)
	return err == nil
}

func (s Snapshot) HasFunction(name string) bool {
	return slices.Contains(s.Linked, name)
}

func (s Snapshot) HasDisk() bool {
	return s.HasFunction(diskFunctionName)
}

func (s Snapshot) HasNetwork() bool {
	return s.NetworkKind() != ""
}

// The protocol the gadget is presenting to the attached host, empty when it is
// presenting none. It reads the linkage rather than a /boot sentinel, so a
// gadget the NCM branch built reports ncm rather than the rndis the sentinel
// pair implies (S03usbdev:53, H10).
func (s Snapshot) NetworkKind() FunctionKind {
	for _, kind := range NetworkKinds {
		if s.HasFunction(functionName(Function{Kind: kind, Instance: netInstance})) {
			return kind
		}
	}
	return ""
}
