package presentation

import (
	"errors"
	"strings"
	"testing"
)

func containsFunctionKind(functions []Function, kind FunctionKind) bool {
	for _, function := range functions {
		if function.Kind == kind {
			return true
		}
	}
	return false
}

func fullFlagExpansion(p Profile) []Function {
	functions := append([]Function(nil), p.Functions...)

	if !containsFunctionKind(functions, FunctionNCM) && !containsFunctionKind(functions, FunctionRNDIS) {
		functions = append(functions, Function{Kind: FunctionRNDIS, Instance: "usb0"})
	}
	if !containsFunctionKind(functions, FunctionMassStorage) {
		functions = append(functions, Function{Kind: FunctionMassStorage, Instance: "disk0"})
	}
	return functions
}

func TestStaticV0IsTheShippingBudget(t *testing.T) {
	if staticV0.Source != SourceStaticV0 {
		t.Fatalf("source = %q, want %q", staticV0.Source, SourceStaticV0)
	}
	if staticV0.MaxInEndpoints != 6 || staticV0.MaxOutEndpoints != 5 {
		t.Fatalf("budget = %d IN %d OUT, want 6 IN 5 OUT", staticV0.MaxInEndpoints, staticV0.MaxOutEndpoints)
	}

	for _, kind := range []FunctionKind{FunctionHID, FunctionNCM, FunctionRNDIS, FunctionMassStorage} {
		caps, ok := staticV0.Functions[kind]
		if !ok || !caps.Available {
			t.Fatalf("%s caps = %+v present = %t, want available", kind, caps, ok)
		}
	}
}

func TestBuiltInProfilesFitStaticV0(t *testing.T) {
	tests := []struct {
		name      string
		functions []Function
		want      EndpointUse
	}{
		{
			name:      "standard at full flag expansion",
			functions: fullFlagExpansion(standardProfile()),
			want:      EndpointUse{In: 6, Out: 5},
		},
		{
			name:      "hid-only",
			functions: hidOnlyProfile().Functions,
			want:      EndpointUse{In: 3, Out: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, err := AccountEndpoints(tt.functions, staticV0)
			if err != nil {
				t.Fatalf("account endpoints: %v", err)
			}
			if used != tt.want {
				t.Fatalf("used = %+v, want %+v", used, tt.want)
			}
		})
	}
}

func TestStandardAtMaximumHasZeroHeadroom(t *testing.T) {
	used, err := AccountEndpoints(fullFlagExpansion(standardProfile()), staticV0)
	if err != nil {
		t.Fatalf("account endpoints: %v", err)
	}
	if headroom := used.Headroom(staticV0); headroom != (EndpointUse{}) {
		t.Fatalf("headroom = %+v, want zero", headroom)
	}
}

func TestAccountEndpointsRefusesOverflow(t *testing.T) {
	functions := append(fullFlagExpansion(standardProfile()), Function{Kind: FunctionHID, Instance: "GS3"})

	_, err := AccountEndpoints(functions, staticV0)
	if !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("err = %v, want ErrEndpointBudget", err)
	}
	if !strings.Contains(err.Error(), "hid.GS3") {
		t.Fatalf("err = %v, want the overflowing function named", err)
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v0") {
		t.Fatalf("err = %v, want the capability source carried", err)
	}
}

func TestAccountEndpointsRefusesUnavailableFunction(t *testing.T) {
	table := staticV0.clone()
	table.Functions[FunctionNCM] = FunctionCaps{Available: false, InEPs: 2, OutEPs: 1}

	_, err := AccountEndpoints([]Function{{Kind: FunctionNCM, Instance: "usb0"}}, table)
	if !errors.Is(err, ErrFunctionUnavailable) {
		t.Fatalf("err = %v, want ErrFunctionUnavailable", err)
	}
	if !strings.Contains(err.Error(), "ncm.usb0 rejected by capability table static-v0") {
		t.Fatalf("err = %v, want the function and source named", err)
	}
}

func TestAccountEndpointsRefusesUnknownFunction(t *testing.T) {
	table := staticV0.clone()
	delete(table.Functions, FunctionMassStorage)

	_, err := AccountEndpoints([]Function{{Kind: FunctionMassStorage, Instance: "disk0"}}, table)
	if !errors.Is(err, ErrUnknownFunction) {
		t.Fatalf("err = %v, want ErrUnknownFunction", err)
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v0") {
		t.Fatalf("err = %v, want the capability source carried", err)
	}
}

func TestProbeNeverProbesHID(t *testing.T) {
	for _, kind := range probeKinds {
		if kind == FunctionHID {
			t.Fatal("probeKinds includes hid, which would consume a /dev/hidgN minor")
		}
	}
}

func TestWithAvailabilityKeepsBudget(t *testing.T) {
	merged := staticV0.withAvailability(map[FunctionKind]bool{FunctionNCM: false})

	if merged.Source != SourceProbeV1 {
		t.Fatalf("source = %q, want %q", merged.Source, SourceProbeV1)
	}
	if merged.MaxInEndpoints != staticV0.MaxInEndpoints || merged.MaxOutEndpoints != staticV0.MaxOutEndpoints {
		t.Fatalf("budget = %d IN %d OUT, want the static-v0 budget", merged.MaxInEndpoints, merged.MaxOutEndpoints)
	}
	if merged.Functions[FunctionNCM].Available {
		t.Fatal("ncm availability = true, want the probe result")
	}
	if !staticV0.Functions[FunctionNCM].Available {
		t.Fatal("staticV0 was mutated by the merge")
	}
}
