package presentation

import (
	"slices"
	"strings"
)

const (
	netInstance  = "usb0"
	diskInstance = "disk0"
	lunDir       = "lun.0"

	diskFunctionName = string(FunctionMassStorage) + "." + diskInstance

	// f_ncm and f_rndis both take the interface name from the function
	// instance, so the gadget NIC is usb0 on this device and this package is
	// what knows it.
	GadgetNIC = netInstance
)

type Snapshot struct {
	Active        string   `json:"active"`
	LastKnownGood string   `json:"last_known_good"`
	Mode          string   `json:"mode"`
	Capabilities  string   `json:"capabilities"`
	UDC           string   `json:"udc"`
	Bound         bool     `json:"bound"`
	Present       []string `json:"present"`
	Linked        []string `json:"linked"`
}

// Linkage is read back through the function's own attribute rather than from a
// /boot sentinel, so a gadget the NCM branch built stops reporting the RNDIS
// toggle as on (S03usbdev:53, H10).
var functionProbeAttr = map[FunctionKind]string{
	FunctionHID:         "protocol",
	FunctionNCM:         "dev_addr",
	FunctionRNDIS:       "dev_addr",
	FunctionMassStorage: lunDir + "/file",
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

func readSnapshot(ops Ops, extra []Function) Snapshot {
	var snapshot Snapshot
	if data, err := ops.ReadFile(udcAttr); err == nil {
		snapshot.UDC = strings.TrimSpace(string(data))
		snapshot.Bound = snapshot.UDC != ""
	}

	seen := make(map[string]bool)
	for _, function := range append(knownFunctions(), extra...) {
		attr, ok := functionProbeAttr[function.Kind]
		name := functionName(function)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true

		if readable(ops, functionsDir+"/"+name+"/"+attr) {
			snapshot.Present = append(snapshot.Present, name)
		}
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
