package presentation

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

func hidFunctionsOf(profile Profile) []Function {
	var functions []Function
	for _, function := range profile.Functions {
		if function.Kind == FunctionHID {
			functions = append(functions, function)
		}
	}
	return functions
}

func netFunction() Function {
	return Function{Kind: FunctionNCM, Instance: "usb0", Net: &NetFunction{
		DevAddr: ptr("42:32:5b:74:a1:6c"), HostAddr: ptr("42:32:5b:74:a1:6d"), CompatibleID: "NCM",
	}}
}

func layoutProfile(groups [][]HIDRole, extra ...Function) (Profile, error) {
	profile := standardProfile()
	profile.Name = "layout-test"
	profile.BuiltIn = false
	profile.Functions = append([]Function{netFunction()}, profile.Functions...)
	profile.Functions = append(profile.Functions, extra...)
	if err := SetHIDLayout(&profile, groups); err != nil {
		return profile, err
	}
	return profile, nil
}

// The traces are byte-comparisons against a recorded shell script, so a layout
// of one role per interface has to keep emitting the descriptors that script
// wrote. Anything else is a change to the gadget the host enumerates.
func TestSingleRoleGroupsKeepTheShippedDescriptors(t *testing.T) {
	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative}, {HIDRoleAbsolute}})
	if err != nil {
		t.Fatal(err)
	}
	want := standardProfile()
	for i, function := range hidFunctionsOf(profile) {
		expected := want.Functions[i].HID
		if !bytes.Equal(function.HID.ReportDesc, expected.ReportDesc) {
			t.Fatalf("hid %d report_desc = % x, want % x", i, function.HID.ReportDesc, expected.ReportDesc)
		}
		if function.HID.ReportLength != expected.ReportLength || function.HID.Protocol != expected.Protocol {
			t.Fatalf("hid %d = length %d protocol %d, want length %d protocol %d",
				i, function.HID.ReportLength, function.HID.Protocol, expected.ReportLength, expected.Protocol)
		}
	}
}

func TestCompositeReportCarriesReportIDsAndOneExtraByte(t *testing.T) {
	desc, length, err := composeHIDReport([]HIDRole{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if length != 9 {
		t.Fatalf("report length = %d, want 9 (one prefix byte plus the 8 byte keyboard report)", length)
	}
	for id := byte(1); id <= 3; id++ {
		if !bytes.Contains(desc, []byte{0x85, id}) {
			t.Fatalf("descriptor is missing report id %d: % x", id, desc)
		}
	}
	if err := validateHIDReportDescriptor(desc); err != nil {
		t.Fatalf("composite descriptor is malformed: %v", err)
	}
	measured, err := reportLength(desc)
	if err != nil {
		t.Fatal(err)
	}
	if measured != length {
		t.Fatalf("reportLength(desc) = %d, want %d", measured, length)
	}
}

// A composite cannot answer a boot-protocol Get_Report, so it must not claim to.
func TestCompositeInterfaceDropsTheBootProtocolClaim(t *testing.T) {
	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}})
	if err != nil {
		t.Fatal(err)
	}
	functions := hidFunctionsOf(profile)
	if len(functions) != 1 {
		t.Fatalf("hid functions = %d, want 1", len(functions))
	}
	if functions[0].HID.Protocol != 0 || functions[0].HID.SubClass != 0 {
		t.Fatalf("protocol = %d subclass = %d, want 0 and 0", functions[0].HID.Protocol, functions[0].HID.SubClass)
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("composite profile rejected: %v", err)
	}
}

// The whole point of the layout: six IN endpoints is silicon, and camera, mic
// and NIC take five of them between them.
func TestCameraMicAndNICFitOnlyWithTheHIDRolesShared(t *testing.T) {
	media := []Function{testCamera("cam0", 768), testMicrophone("mic0")}

	separate, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative}, {HIDRoleAbsolute}}, media...)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Compile(separate, staticV1)
	if !errors.Is(err, ErrEndpointBudget) || !strings.Contains(err.Error(), "uvc.cam0") {
		t.Fatalf("three separate hid interfaces: err = %v, want an endpoint refusal naming uvc.cam0", err)
	}

	shared, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}}, media...)
	if err != nil {
		t.Fatal(err)
	}
	used, err := AccountEndpoints(shared.Functions, staticV1)
	if err != nil {
		t.Fatalf("one shared hid interface: %v", err)
	}
	if used.In != staticV1.MaxInEndpoints {
		t.Fatalf("IN endpoints = %d, want exactly %d", used.In, staticV1.MaxInEndpoints)
	}
	if _, err := Compile(shared, staticV1); err != nil {
		t.Fatalf("compile: %v", err)
	}
}

// Dropping a role returns exactly one endpoint and no more, which is why the
// full HID set alongside camera and microphone needs the composition rather
// than just switching something off.
func TestDroppingARoleReturnsOneEndpointAndIsStillOneShort(t *testing.T) {
	media := []Function{testCamera("cam0", 768), testMicrophone("mic0")}
	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative}}, media...)
	if err != nil {
		t.Fatal(err)
	}
	used, err := AccountEndpoints(profile.Functions, staticV1)
	if !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("err = %v, want an endpoint refusal", err)
	}
	if used.In != staticV1.MaxInEndpoints+1 {
		t.Fatalf("IN endpoints = %d, want %d", used.In, staticV1.MaxInEndpoints+1)
	}

	// Without the camera it fits with an endpoint to spare.
	profile, err = layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative}}, testMicrophone("mic0"))
	if err != nil {
		t.Fatal(err)
	}
	if used, err = AccountEndpoints(profile.Functions, staticV1); err != nil {
		t.Fatal(err)
	}
	if used.In != 5 {
		t.Fatalf("IN endpoints = %d, want 5", used.In)
	}
}

// Declining f_uvc's unused control interrupt endpoint is what buys a separate
// boot keyboard alongside the camera, and only a kernel carrying the option
// can honour it.
func TestUVCInterruptEndpointOptOutNeedsTheKernelOption(t *testing.T) {
	camera := testCamera("cam0", 768)
	camera.Video.InterruptEndpoint = ptr(false)
	media := []Function{camera, testMicrophone("mic0")}

	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative, HIDRoleAbsolute}}, media...)
	if err != nil {
		t.Fatal(err)
	}
	_, err = AccountEndpoints(profile.Functions, staticV1)
	if !errors.Is(err, ErrFunctionUnavailable) || !strings.Contains(err.Error(), UVCAttrInterruptEP) {
		t.Fatalf("err = %v, want a refusal naming %s", err, UVCAttrInterruptEP)
	}

	patched := staticV1.withAvailability(map[FunctionKind]FunctionProbe{
		FunctionUVC: {Available: true, Attributes: map[string]bool{UVCAttrInterruptEP: true}},
	})
	used, err := AccountEndpoints(profile.Functions, patched)
	if err != nil {
		t.Fatalf("patched kernel: %v", err)
	}
	if used.In != patched.MaxInEndpoints {
		t.Fatalf("IN endpoints = %d, want %d", used.In, patched.MaxInEndpoints)
	}
	if plan, err := Compile(profile, patched); err != nil {
		t.Fatal(err)
	} else if !planWrites(plan, "functions/uvc.cam0/"+UVCAttrInterruptEP, "0") {
		t.Fatalf("plan does not write %s", UVCAttrInterruptEP)
	}
}

func planWrites(plan Plan, path, data string) bool {
	for _, op := range plan.Ops {
		if op.Kind == OpWrite && strings.HasSuffix(op.Path, path) && strings.TrimSpace(string(op.Data)) == data {
			return true
		}
	}
	return false
}

func TestHIDRoutesCarryNodeAndReportID(t *testing.T) {
	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard}, {HIDRoleRelative, HIDRoleAbsolute}})
	if err != nil {
		t.Fatal(err)
	}
	routes := HIDRoutes(profile.Functions)
	want := []HIDRoute{
		{Role: HIDRoleKeyboard, Path: "/dev/hidg0", ReportID: 0, Length: 8},
		{Role: HIDRoleRelative, Path: "/dev/hidg1", ReportID: 1, Length: 4},
		{Role: HIDRoleAbsolute, Path: "/dev/hidg1", ReportID: 2, Length: 6},
	}
	if !slices.Equal(routes, want) {
		t.Fatalf("routes = %+v, want %+v", routes, want)
	}
}

// The instances have to stay a prefix of GS0,GS1,GS2 and the block has to stay
// where it was in the link order, because both fix what the host enumerates.
func TestLayoutKeepsInstancePrefixAndLinkOrder(t *testing.T) {
	profile, err := layoutProfile([][]HIDRole{{HIDRoleKeyboard, HIDRoleAbsolute}})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, function := range profile.Functions {
		names = append(names, functionName(function))
	}
	if !slices.Equal(names, []string{"ncm.usb0", "hid.GS0"}) {
		t.Fatalf("functions = %v, want [ncm.usb0 hid.GS0]", names)
	}
	if err := profile.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLayoutRefusesDuplicateAndEmptyGroups(t *testing.T) {
	for _, groups := range [][][]HIDRole{
		{},
		{{}},
		{{HIDRoleKeyboard}, {HIDRoleKeyboard}},
		{{HIDRoleKeyboard}, {HIDRoleRelative}, {HIDRoleAbsolute}, {HIDRoleKeyboard}},
		{{"trackball"}},
	} {
		if err := ValidateHIDLayout(groups); !errors.Is(err, ErrHIDLayout) {
			t.Fatalf("ValidateHIDLayout(%v) = %v, want ErrHIDLayout", groups, err)
		}
	}
}

// A profile stored before roles existed carries none, and its three functions
// still have to mean keyboard, relative mouse and absolute pointer.
func TestNormalizeBackfillsRolesOnStoredProfiles(t *testing.T) {
	profile := standardProfile()
	for i := range profile.Functions {
		profile.Functions[i].HID.Roles = nil
	}
	profile.Normalize()
	if got := HIDLayout(profile.Functions); !slices.Equal(HIDRoutes(profile.Functions), []HIDRoute{
		{Role: HIDRoleKeyboard, Path: "/dev/hidg0", Length: 8},
		{Role: HIDRoleRelative, Path: "/dev/hidg1", Length: 4},
		{Role: HIDRoleAbsolute, Path: "/dev/hidg2", Length: 6},
	}) {
		t.Fatalf("layout = %v", got)
	}
}
