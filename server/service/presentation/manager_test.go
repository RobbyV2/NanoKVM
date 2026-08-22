package presentation

import (
	"context"
	"testing"
)

// NIC is the presentation half of the bridge's step 13. Without it the bridge's
// Gadget is nil in production, step 13 never runs, and br0 comes up with eth0 as
// its only port.
func TestNICReportsTheGadgetInterfaceOnlyWhenOneIsLinked(t *testing.T) {
	tests := []struct {
		name   string
		linked string
		want   string
	}{
		{name: "ncm linked", linked: "ncm.usb0", want: GadgetNIC},
		{name: "rndis linked", linked: "rndis.usb0", want: GadgetNIC},
		{name: "no network function", linked: "", want: ""},
		// The function exists but is not in configs/c.1, so the kernel never
		// bound it and there is no usb0 to enslave.
		{name: "present but unlinked", linked: "-", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, ops := newTestManager(t)

			switch test.linked {
			case "":
			case "-":
				seed(t, ops, functionsDir+"/ncm.usb0/dev_addr")
			default:
				seed(t, ops, functionsDir+"/"+test.linked+"/dev_addr")
				seed(t, ops, configPrefix+"/"+test.linked+"/dev_addr")
			}

			nic, err := manager.NIC(context.Background())
			if err != nil {
				t.Fatalf("NIC: %v", err)
			}
			if nic != test.want {
				t.Fatalf("NIC = %q, want %q", nic, test.want)
			}
		})
	}
}

// Which of the two the gadget presents is read back from the linkage, so the
// bridge panel and the Settings, Device selector agree with the gadget rather
// than with a /boot sentinel.
func TestNetworkProtocolNamesTheLinkedFunction(t *testing.T) {
	tests := []struct {
		name   string
		linked string
		want   string
	}{
		{name: "ncm", linked: "ncm.usb0", want: string(FunctionNCM)},
		{name: "rndis", linked: "rndis.usb0", want: string(FunctionRNDIS)},
		{name: "none", linked: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, ops := newTestManager(t)
			if test.linked != "" {
				seed(t, ops, functionsDir+"/"+test.linked+"/dev_addr")
				seed(t, ops, configPrefix+"/"+test.linked+"/dev_addr")
			}

			protocol, err := manager.NetworkProtocol(context.Background())
			if err != nil {
				t.Fatalf("NetworkProtocol: %v", err)
			}
			if protocol != test.want {
				t.Fatalf("NetworkProtocol = %q, want %q", protocol, test.want)
			}
		})
	}
}

func TestNICPropagatesAnUnreadableGadget(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.ops = nil

	if _, err := manager.NIC(context.Background()); err == nil {
		t.Fatal("NIC on a manager with no gadget returned no error")
	}
}

func seed(t *testing.T, ops *RecordOps, rel string) {
	t.Helper()
	if err := ops.Seed(rel, []byte("48:da:35:6e:11:22\n")); err != nil {
		t.Fatalf("seed %s: %v", rel, err)
	}
}
