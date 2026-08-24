package presentation

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
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

func TestStaticV1IsTheShippingBudget(t *testing.T) {
	if staticV1.Source != SourceStaticV1 {
		t.Fatalf("source = %q, want %q", staticV1.Source, SourceStaticV1)
	}
	if staticV1.MaxInEndpoints != 6 || staticV1.MaxOutEndpoints != 5 {
		t.Fatalf("budget = %d IN %d OUT, want 6 IN 5 OUT", staticV1.MaxInEndpoints, staticV1.MaxOutEndpoints)
	}
	if len(staticV1.InFIFOWords) != staticV1.MaxInEndpoints {
		t.Fatalf("in fifos = %v, want one per IN endpoint", staticV1.InFIFOWords)
	}

	for _, kind := range []FunctionKind{FunctionHID, FunctionNCM, FunctionRNDIS, FunctionMassStorage, FunctionFFS, FunctionUVC, FunctionUAC2} {
		caps, ok := staticV1.Functions[kind]
		if !ok || !caps.Available {
			t.Fatalf("%s caps = %+v present = %t, want available", kind, caps, ok)
		}
	}
	for kind, want := range map[FunctionKind][]int{
		FunctionNCM:         {512, 16},
		FunctionRNDIS:       {512, 16},
		FunctionMassStorage: {512},
		FunctionUVC:         {16, 768},
		FunctionUAC2:        {96},
	} {
		if got := staticV1.Functions[kind].INPackets; !slices.Equal(got, want) {
			t.Fatalf("%s in packets = %v, want %v", kind, got, want)
		}
	}
}

func TestLoadCapabilitiesShipsStaticV1(t *testing.T) {
	useTestPresentationDir(t)

	shipped := LoadCapabilities()
	if shipped.Source != SourceStaticV1 && shipped.Source != SourceProbeV1 {
		t.Fatalf("source = %q, want a static-v1 derived table", shipped.Source)
	}
	if shipped.MaxInEndpoints != staticV1.MaxInEndpoints || shipped.MaxOutEndpoints != staticV1.MaxOutEndpoints {
		t.Fatalf("budget = %d IN %d OUT, want the static-v1 budget", shipped.MaxInEndpoints, shipped.MaxOutEndpoints)
	}
	if !slices.Equal(shipped.InFIFOWords, staticV1.InFIFOWords) {
		t.Fatalf("in fifos = %v, want %v", shipped.InFIFOWords, staticV1.InFIFOWords)
	}
	for kind, caps := range staticV1.Functions {
		got, ok := shipped.Functions[kind]
		if !ok || !slices.Equal(got.INPackets, caps.INPackets) {
			t.Fatalf("%s in packets = %v present = %t, want %v", kind, got.INPackets, ok, caps.INPackets)
		}
	}
}

func TestBuiltInProfilesFitStaticV1(t *testing.T) {
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
			used, err := AccountEndpoints(tt.functions, staticV1)
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
	used, err := AccountEndpoints(fullFlagExpansion(standardProfile()), staticV1)
	if err != nil {
		t.Fatalf("account endpoints: %v", err)
	}
	if headroom := used.Headroom(staticV1); headroom != (EndpointUse{}) {
		t.Fatalf("headroom = %+v, want zero", headroom)
	}
}

func TestAccountEndpointsRefusesOverflow(t *testing.T) {
	functions := append(fullFlagExpansion(standardProfile()), Function{Kind: FunctionHID, Instance: "GS3"})

	_, err := AccountEndpoints(functions, staticV1)
	if !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("err = %v, want ErrEndpointBudget", err)
	}
	if !strings.Contains(err.Error(), "hid.GS3") {
		t.Fatalf("err = %v, want the overflowing function named", err)
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v1") {
		t.Fatalf("err = %v, want the capability source carried", err)
	}
}

func TestAccountEndpointsRefusesUnavailableFunction(t *testing.T) {
	table := staticV1.clone()
	table.Functions[FunctionNCM] = FunctionCaps{Available: false, InEPs: 2, OutEPs: 1}

	_, err := AccountEndpoints([]Function{{Kind: FunctionNCM, Instance: "usb0"}}, table)
	if !errors.Is(err, ErrFunctionUnavailable) {
		t.Fatalf("err = %v, want ErrFunctionUnavailable", err)
	}
	if !strings.Contains(err.Error(), "ncm.usb0 rejected by capability table static-v1") {
		t.Fatalf("err = %v, want the function and source named", err)
	}
}

func TestAccountEndpointsRefusesUnknownFunction(t *testing.T) {
	table := staticV1.clone()
	delete(table.Functions, FunctionMassStorage)

	_, err := AccountEndpoints([]Function{{Kind: FunctionMassStorage, Instance: "disk0"}}, table)
	if !errors.Is(err, ErrUnknownFunction) {
		t.Fatalf("err = %v, want ErrUnknownFunction", err)
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v1") {
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
	merged := staticV1.withAvailability(map[FunctionKind]FunctionProbe{FunctionNCM: {Available: false}})

	if merged.Source != SourceProbeV1 {
		t.Fatalf("source = %q, want %q", merged.Source, SourceProbeV1)
	}
	if merged.MaxInEndpoints != staticV1.MaxInEndpoints || merged.MaxOutEndpoints != staticV1.MaxOutEndpoints {
		t.Fatalf("budget = %d IN %d OUT, want the static-v1 budget", merged.MaxInEndpoints, merged.MaxOutEndpoints)
	}
	if merged.Functions[FunctionNCM].Available {
		t.Fatal("ncm availability = true, want the probe result")
	}
	if !staticV1.Functions[FunctionNCM].Available {
		t.Fatal("staticV1 was mutated by the merge")
	}
}

func TestMediaCapabilityNeedsFunctionsAndFIFOMap(t *testing.T) {
	if !staticV1.supportsMedia() {
		t.Fatal("media capability table was rejected")
	}

	tests := []struct {
		name  string
		table CapabilityTable
	}{
		{"UVC", withoutFunction(FunctionUVC)},
		{"UAC2", withoutFunction(FunctionUAC2)},
		{"FunctionFS", withoutFunction(FunctionFFS)},
		{"a FIFO map", func() CapabilityTable { table := staticV1.clone(); table.InFIFOWords = nil; return table }()},
		{"an answer for the uvc naming attribute", withoutAttribute(FunctionUVC, UVCAttrFunctionName)},
		{"an answer for the uac2 naming attribute", withoutAttribute(FunctionUAC2, UAC2AttrFunctionName)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.table.supportsMedia() {
				t.Fatalf("capability table without %s was accepted", tt.name)
			}
		})
	}
}

func TestCompileRefusesAProfileThatOverflowsTheShippingFIFOs(t *testing.T) {
	endpoints := make([]FunctionFSEndpoint, 0, 5)
	for i := 0; i < 5; i++ {
		address := uint8(0x81 + i)
		endpoints = append(endpoints, FunctionFSEndpoint{SourceAddress: address, Address: address, Transfer: EndpointInterrupt, MaxPacket: 1024, Interval: 1})
	}
	profile := standardProfile()
	profile.Name = "fifo-overflow"
	profile.BuiltIn = false
	profile.Functions = []Function{{Kind: FunctionFFS, Instance: "hybrid", FFS: &FunctionFS{Interfaces: 1, Endpoints: endpoints}}}

	_, err := Compile(profile, staticV1)
	if !errors.Is(err, ErrFIFOBudget) {
		t.Fatalf("err = %v, want ErrFIFOBudget", err)
	}
	if !strings.Contains(err.Error(), "ffs.hybrid") {
		t.Fatalf("err = %v, want the overflowing function named", err)
	}
	if !strings.Contains(err.Error(), "rejected by capability table static-v1") {
		t.Fatalf("err = %v, want the capability source carried", err)
	}
}

func withoutFunction(kind FunctionKind) CapabilityTable {
	table := staticV1.clone()
	delete(table.Functions, kind)
	return table
}

// LoadCapabilities runs before the HTTP listener binds, and its probe is a
// series of configfs mkdirs against a gadget the boot script already bound.
// A probe that never returns must not take the listener with it.
func TestLoadCapabilitiesAbandonsAStalledProbe(t *testing.T) {
	dir := t.TempDir()
	previousDir, previousProbe, previousBudget := presentationDir, probe, probeBudget
	t.Cleanup(func() { presentationDir, probe, probeBudget = previousDir, previousProbe, previousBudget })

	presentationDir = dir
	probeBudget = 50 * time.Millisecond
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	probe = func() (map[FunctionKind]FunctionProbe, error) {
		<-release
		return nil, nil
	}

	done := make(chan CapabilityTable, 1)
	go func() { done <- LoadCapabilities() }()

	select {
	case table := <-done:
		if table.Source != SourceStaticV1 {
			t.Fatalf("got source %q, want %q", table.Source, SourceStaticV1)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("LoadCapabilities never returned from a stalled probe")
	}
}

// The isochronous lane credits back a UVC control interrupt endpoint in
// AccountEndpoints, and the HID lane changes how many HID interfaces a profile
// carries. Both feed the same budget, and the two halves of the arithmetic only
// agree if every IN endpoint charged also gets a FIFO seated: a function that is
// charged an endpoint SeatFIFOs never seats compiles here and fails at bind.
func TestEveryChargedINEndpointIsSeatedAFIFO(t *testing.T) {
	table := staticV1.clone()
	table.Functions[FunctionUVC] = FunctionCaps{
		Available: true, InEPs: 2, OutEPs: 0, INPackets: []int{16, 768},
		Attributes: map[string]bool{UVCAttrInterruptEP: true},
	}
	declined := false

	cases := map[string]Function{
		"hid single role":    {Kind: FunctionHID, Instance: "GS0", HID: &HIDFunction{ReportLength: 8}},
		"hid composite":      {Kind: FunctionHID, Instance: "GS0", HID: &HIDFunction{ReportLength: 9}},
		"ncm":                {Kind: FunctionNCM, Instance: "usb0", Net: &NetFunction{}},
		"rndis":              {Kind: FunctionRNDIS, Instance: "usb0", Net: &NetFunction{}},
		"mass storage":       {Kind: FunctionMassStorage, Instance: "disk", Storage: &StorageFunction{}},
		"uvc with interrupt": {Kind: FunctionUVC, Instance: "cam0", Video: &VideoFunction{StreamingMaxPacket: 768}},
		"uvc declining the interrupt": {Kind: FunctionUVC, Instance: "cam0",
			Video: &VideoFunction{StreamingMaxPacket: 768, InterruptEndpoint: &declined}},
		"uac2": {Kind: FunctionUAC2, Instance: "mic0",
			Audio: &AudioFunction{PChannelMask: 3, PSampleRate: 48000, PSampleSize: 2}},
		"functionfs": {Kind: FunctionFFS, Instance: "hybrid", FFS: &FunctionFS{Endpoints: []FunctionFSEndpoint{
			{Address: 0x81, MaxPacket: 512}, {Address: 0x02, MaxPacket: 512}, {Address: 0x83, MaxPacket: 64},
		}}},
	}

	for name, function := range cases {
		t.Run(name, func(t *testing.T) {
			use, err := AccountEndpoints([]Function{function}, table)
			if err != nil {
				t.Fatalf("account: %s", err)
			}
			fifos, err := SeatFIFOs([]Function{function}, table)
			if err != nil {
				t.Fatalf("seat: %s", err)
			}
			seated := 0
			for _, assignment := range fifos {
				seated += len(assignment)
			}
			if seated != use.In {
				t.Fatalf("charged %d IN endpoints but seated %d FIFOs (%v)", use.In, seated, fifos)
			}
		})
	}
}

// Whichever lane produced the plan, the budget has to be the same budget. The
// layout editor decides how many HID interfaces are charged and the isochronous
// opt-in decides whether UVC costs one endpoint or two; the two are only
// consistent if Compile refuses exactly what AccountEndpoints refuses.
func TestHIDLayoutAndISOOptOutShareTheSixINBudget(t *testing.T) {
	table := staticV1.clone()
	table.Functions[FunctionUVC] = FunctionCaps{
		Available: true, InEPs: 2, OutEPs: 0, INPackets: []int{16, 768},
		Attributes: map[string]bool{UVCAttrInterruptEP: true},
	}

	layouts := [][][]HIDRole{
		{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}},
		{{HIDRoleKeyboard}, {HIDRoleRelative, HIDRoleAbsolute}},
		{{HIDRoleKeyboard}, {HIDRoleRelative}, {HIDRoleAbsolute}},
	}
	// Charged IN endpoints per layout is exactly the interface count, and the
	// camera costs two or one depending on the isochronous opt-in.
	want := map[string]int{
		"1 interface, interrupt kept":      1 + 2,
		"1 interface, interrupt declined":  1 + 1,
		"2 interfaces, interrupt kept":     2 + 2,
		"2 interfaces, interrupt declined": 2 + 1,
		"3 interfaces, interrupt kept":     3 + 2,
		"3 interfaces, interrupt declined": 3 + 1,
	}

	for _, groups := range layouts {
		for _, keepInterrupt := range []bool{true, false} {
			label := fmt.Sprintf("%d interface", len(groups))
			if len(groups) > 1 {
				label += "s"
			}
			if keepInterrupt {
				label += ", interrupt kept"
			} else {
				label += ", interrupt declined"
			}

			t.Run(label, func(t *testing.T) {
				profile := standardProfile()
				if err := SetHIDLayout(&profile, groups); err != nil {
					t.Fatalf("layout: %s", err)
				}
				video := defaultCamera(0, "Camera", false)
				if !keepInterrupt {
					declined := false
					video.Video.InterruptEndpoint = &declined
				}
				profile.Functions = append(profile.Functions, video)

				use, accountErr := AccountEndpoints(profile.Functions, table)
				plan, compileErr := Compile(profile, table)

				if (accountErr == nil) != (compileErr == nil) {
					t.Fatalf("account error %v disagrees with compile error %v", accountErr, compileErr)
				}
				if accountErr != nil {
					if !errors.Is(accountErr, ErrEndpointBudget) && !errors.Is(accountErr, ErrFIFOBudget) {
						t.Fatalf("unexpected refusal: %s", accountErr)
					}
					return
				}
				if use.In != want[label] {
					t.Fatalf("charged %d IN endpoints, want %d", use.In, want[label])
				}
				if use.In > table.MaxInEndpoints {
					t.Fatalf("accepted %d IN endpoints over the %d budget", use.In, table.MaxInEndpoints)
				}
				if plan.Endpoints != use {
					t.Fatalf("the compiled plan reports %+v but the accounting says %+v", plan.Endpoints, use)
				}
			})
		}
	}
}

// A probe-v1 table written before the naming attributes were probed answers
// nothing about them, and a missing key is indistinguishable from a false one.
func withoutAttribute(kind FunctionKind, name string) CapabilityTable {
	table := staticV1.clone()
	table.Source = SourceProbeV1
	delete(table.Functions[kind].Attributes, name)
	return table
}

// Six IN endpoints cannot be widened, so the only useful refusal is one that
// names what is holding them and what trading it away would return.
func TestEndpointBudgetRefusalNamesTheHoldersAndTheTrades(t *testing.T) {
	functions := []Function{
		{Kind: FunctionRNDIS, Instance: "usb0", Net: &NetFunction{}},
		{Kind: FunctionHID, Instance: "GS0", HID: &HIDFunction{ReportLength: 8}},
		{Kind: FunctionHID, Instance: "GS1", HID: &HIDFunction{ReportLength: 4}},
		{Kind: FunctionHID, Instance: "GS2", HID: &HIDFunction{ReportLength: 6}},
		testCamera("cam0", 768),
	}

	_, err := AccountEndpoints(functions, staticV1)
	if !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("err = %v, want ErrEndpointBudget", err)
	}

	for _, want := range []string{
		"in use by rndis.usb0 2, hid.GS0 1, hid.GS1 1, hid.GS2 1",
		"1 more needed",
		"putting the 3 HID interfaces on one frees 2",
		"turning off USB networking frees 2",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %q, want it to contain %q", err.Error(), want)
		}
	}
}
