package presentation

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
)

type fakeGadgetObserver struct {
	mu       sync.Mutex
	suspend  int
	applied  int
	profiles []string
}

func (f *fakeGadgetObserver) Suspend() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.suspend++
}

func (f *fakeGadgetObserver) Applied(_ context.Context, profile Profile, _ Plan) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applied++
	f.profiles = append(f.profiles, profile.Name)
	return nil
}

func testCamera(instance string, packet uint16) Function {
	return Function{
		Kind:     FunctionUVC,
		Instance: instance,
		Video: &VideoFunction{
			FunctionName:       "NanoKVM Camera " + instance,
			StreamingMaxPacket: packet,
			StreamingMaxBurst:  0,
			StreamingInterval:  1,
			Formats: []VideoFormat{{
				Codec: "mjpeg",
				Frames: []VideoFrame{
					{Width: 1280, Height: 720, Intervals: []uint32{333333, 666666}},
					{Width: 640, Height: 480, Intervals: []uint32{333333, 666666}},
				},
			}},
		},
	}
}

func testMicrophone(instance string) Function {
	return Function{
		Kind:     FunctionUAC2,
		Instance: instance,
		Audio: &AudioFunction{
			FunctionName:  "NanoKVM Microphone " + instance,
			PChannelMask:  1,
			PSampleRate:   48000,
			PSampleSize:   2,
			CChannelMask:  0,
			CSampleRate:   48000,
			CSampleSize:   2,
			RequestNumber: 4,
		},
	}
}

func mediaProfile(functions ...Function) Profile {
	p := standardProfile()
	p.Name = "media-test"
	p.BuiltIn = false
	p.Functions = append(p.Functions, functions...)
	return p
}

func TestMediaFunctionsPreserveHIDOrder(t *testing.T) {
	plan, err := Compile(mediaProfile(testCamera("cam0", 768), testMicrophone("mic0")), staticV1)
	if err != nil {
		t.Fatal(err)
	}

	var links []string
	for _, op := range plan.Ops {
		if op.Kind == OpSymlink && strings.HasPrefix(op.Path, configPrefix+"/") {
			links = append(links, strings.TrimPrefix(op.Path, configPrefix+"/"))
		}
	}
	want := []string{"hid.GS0", "hid.GS1", "hid.GS2", "uvc.cam0", "uac2.mic0"}
	if !slices.Equal(links, want) {
		t.Fatalf("link order = %v, want %v", links, want)
	}
}

func TestUVCCompileMatchesVendorConfigFS(t *testing.T) {
	plan, err := Compile(mediaProfile(testCamera("cam0", 768)), staticV1)
	if err != nil {
		t.Fatal(err)
	}

	writes := map[string]string{}
	links := map[string]bool{}
	var linkAt, lastAttribute int
	for i, op := range plan.Ops {
		if strings.HasPrefix(op.Path, "functions/uvc.cam0/") && op.Kind == OpWrite {
			writes[op.Path] = string(op.Data)
			lastAttribute = i
		}
		if op.Path == "configs/c.1/uvc.cam0" {
			linkAt = i
		}
		if op.Kind == OpSymlink {
			links[op.Path] = true
		}
		if strings.HasSuffix(op.Path, "/function_name") {
			t.Fatalf("an unnamed camera must leave f_uvc's own name alone: %+v", op)
		}
	}
	for path, want := range map[string]string{
		"functions/uvc.cam0/streaming_interval":                                   "1\n",
		"functions/uvc.cam0/streaming_maxpacket":                                  "768\n",
		"functions/uvc.cam0/streaming_maxburst":                                   "0\n",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720/wWidth":                    "1280\n",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720/wHeight":                   "720\n",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720/dwDefaultFrameInterval":    "333333\n",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720/dwFrameInterval":           "333333\n666666\n",
		"functions/uvc.cam0/streaming/mjpeg/m/1280x720/dwMaxVideoFrameBufferSize": "1843200\n",
	} {
		if got := writes[path]; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if linkAt <= lastAttribute {
		t.Fatalf("config link op %d precedes final attribute op %d", linkAt, lastAttribute)
	}
	if !links["functions/uvc.cam0/control/class/fs/h"] || links["functions/uvc.cam0/control/class/hs/h"] {
		t.Fatalf("control class links = %v, vendor 5.10 exposes fs and ss but not hs", links)
	}
	// From 5.15 uvc_function_bind copies the SuperSpeed descriptors whether or
	// not the UDC is one, and refuses to bind without these two links.
	for _, path := range []string{"functions/uvc.cam0/control/class/ss/h", "functions/uvc.cam0/streaming/class/ss/h"} {
		if !links[path] {
			t.Fatalf("%s is not linked, so f_uvc from 5.15 on cannot bind", path)
		}
	}
	firstCreate := slices.IndexFunc(plan.Ops, func(op Op) bool { return op.Kind == OpMkdir && op.Path == "functions/uvc.cam0" })
	lastCleanup := slices.IndexFunc(plan.Ops, func(op Op) bool { return op.Kind == OpRmdir && op.Path == "functions/uvc.cam0/control/header/h" })
	if firstCreate < 0 || lastCleanup < 0 || lastCleanup >= firstCreate {
		t.Fatalf("cleanup/create order = %d/%d", lastCleanup, firstCreate)
	}
}

func TestMediaLinkIsRefreshedWithoutTouchingHID(t *testing.T) {
	ops := NewRecordOps()
	manager := &Manager{ops: ops}
	plan, err := Compile(mediaProfile(testCamera("cam0", 768)), staticV1)
	if err != nil {
		t.Fatal(err)
	}
	before := Snapshot{Linked: []string{"hid.GS0", "uvc.cam0"}}
	if err := manager.unlinkStale(before, plan); err != nil {
		t.Fatal(err)
	}
	trace := ops.Trace()
	if len(trace) != 1 || trace[0].Kind != OpUnlink || trace[0].Path != "configs/c.1/uvc.cam0" {
		t.Fatalf("unlink trace = %+v", trace)
	}
}

func TestUAC2CompileBuildsHostMicrophoneDirection(t *testing.T) {
	plan, err := Compile(mediaProfile(testMicrophone("mic0")), staticV1)
	if err != nil {
		t.Fatal(err)
	}

	writes := map[string]string{}
	var linkAt, lastAttribute int
	for i, op := range plan.Ops {
		if strings.HasPrefix(op.Path, "functions/uac2.mic0/") && op.Kind == OpWrite {
			writes[op.Path] = string(op.Data)
			lastAttribute = i
		}
		if op.Path == "configs/c.1/uac2.mic0" {
			linkAt = i
		}
		if strings.HasSuffix(op.Path, "/function_name") {
			t.Fatalf("an unnamed microphone must leave f_uac2's own name alone: %+v", op)
		}
	}
	for path, want := range map[string]string{
		"functions/uac2.mic0/p_chmask":   "1\n",
		"functions/uac2.mic0/p_srate":    "48000\n",
		"functions/uac2.mic0/p_ssize":    "2\n",
		"functions/uac2.mic0/c_chmask":   "0\n",
		"functions/uac2.mic0/req_number": "4\n",
	} {
		if got := writes[path]; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if linkAt <= lastAttribute {
		t.Fatalf("config link op %d precedes final attribute op %d", linkAt, lastAttribute)
	}
}

func TestMediaProfilesRejectUnsupportedRuntimeFormats(t *testing.T) {
	tests := []struct {
		name     string
		function Function
		want     string
	}{
		{"interval order", func() Function {
			f := testCamera("cam0", 768)
			f.Video.Formats[0].Frames[0].Intervals = []uint32{666666, 333333}
			return f
		}(), "intervals are not ascending"},
		{"capture defaults", func() Function { f := testMicrophone("mic0"); f.Audio.CSampleRate = 0; return f }(), "disabled USB OUT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := mediaProfile(tt.function)
			err := profile.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestMediaEndpointCeilingsNameTheRejectedSlot(t *testing.T) {
	tests := []struct {
		name      string
		functions []Function
		wantErr   error
		wantName  string
	}{
		{"camera", []Function{testCamera("cam0", 768)}, nil, ""},
		{"camera microphone", []Function{testCamera("cam0", 768), testMicrophone("mic0")}, nil, ""},
		{"camera microphone storage", []Function{testCamera("cam0", 768), testMicrophone("mic0"), {Kind: FunctionMassStorage, Instance: "disk0", Storage: &StorageFunction{Removable: true, File: "/dev/mmcblk0p3"}}}, ErrEndpointBudget, "mass_storage.disk0"},
		{"two cameras", []Function{testCamera("cam0", 768), testCamera("cam1", 512)}, ErrEndpointBudget, "uvc.cam1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(mediaProfile(tt.functions...), staticV1)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), tt.wantName) {
				t.Fatalf("err = %v, want %v naming %s", err, tt.wantErr, tt.wantName)
			}
		})
	}
}

func TestFIFOSeatingIsBestFitAndNamesFailure(t *testing.T) {
	functions := []Function{testCamera("cam0", 768), testMicrophone("mic0")}
	assigned, err := SeatFIFOs(functions, staticV1)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(assigned["uvc.cam0"], []int{128, 384}) {
		t.Fatalf("uvc.cam0 fifos = %v, want [128 384]", assigned["uvc.cam0"])
	}
	if !slices.Equal(assigned["uac2.mic0"], []int{128}) {
		t.Fatalf("uac2.mic0 fifos = %v, want [128]", assigned["uac2.mic0"])
	}

	table := staticV1.clone()
	table.MaxInEndpoints = 7
	table.InFIFOWords = []int{192, 128, 128, 128, 128, 128, 64}
	_, err = AccountEndpoints([]Function{testCamera("cam0", 768), testCamera("cam1", 768)}, table)
	if !errors.Is(err, ErrFIFOBudget) || !strings.Contains(err.Error(), "uvc.cam1") {
		t.Fatalf("err = %v, want FIFO failure naming uvc.cam1", err)
	}
}

func TestMediaInstancesMayBeDeclaredUntilWholeProfileValidation(t *testing.T) {
	p := mediaProfile(
		testCamera("cam0", 768),
		testCamera("cam1", 512),
		testMicrophone("mic0"),
		testMicrophone("mic1"),
	)
	if err := p.Validate(); err != nil {
		t.Fatalf("schema rejected declarable slots: %v", err)
	}
	if _, err := Compile(p, staticV1); !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("compile err = %v, want endpoint refusal", err)
	}
}

func TestFunctionFSAndMediaCannotShareAProfile(t *testing.T) {
	profile := mediaProfile(testCamera("cam0", 768))
	profile.Functions = append(profile.Functions[:2], Function{Kind: FunctionFFS, Instance: "hybrid", FFS: pointer(testFunctionFS())}, testCamera("cam0", 768))
	if err := profile.Validate(); err == nil || !strings.Contains(err.Error(), "cannot coexist") {
		t.Fatalf("Validate() = %v, want coexistence refusal", err)
	}
}

func TestFunctionFSParticipatesInFIFOSeating(t *testing.T) {
	function := Function{Kind: FunctionFFS, Instance: "hybrid", FFS: pointer(testFunctionFS())}
	assigned, err := SeatFIFOs([]Function{function}, staticV1)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(assigned["ffs.hybrid"], []int{128}) {
		t.Fatalf("ffs.hybrid fifos = %v, want [128]", assigned["ffs.hybrid"])
	}
}

func TestFailedSurrenderRestoresMediaObserver(t *testing.T) {
	manager, ops := newTestManager(t)
	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatal(err)
	}
	observer := &fakeGadgetObserver{}
	manager.SetObserver(observer)
	ops.SetUDC()

	if _, err := manager.SurrenderUDC(); !errors.Is(err, ErrUDCCount) {
		t.Fatalf("surrender err = %v, want UDC count error", err)
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.suspend != 1 || observer.applied != 1 {
		t.Fatalf("observer suspend/applied = %d/%d, want 1/1", observer.suspend, observer.applied)
	}
}

func TestFunctionFSTransitionSuspendsAndRestoresMediaObserver(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.caps = staticV1
	profile := mediaProfile(testCamera("cam0", 768))
	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	observer := &fakeGadgetObserver{}
	manager.SetObserver(observer)

	state, err := manager.StartFunctionFS(context.Background(), testFunctionFS())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.StopFunctionFS(context.Background(), state.Token); err != nil {
		t.Fatal(err)
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.suspend != 2 || !slices.Equal(observer.profiles, []string{ProfileHybrid, profile.Name}) {
		t.Fatalf("observer suspend/profiles = %d/%v", observer.suspend, observer.profiles)
	}
}

func TestFunctionFSFailureRestoresMediaObserver(t *testing.T) {
	manager, ops := newTestManager(t)
	manager.caps = staticV1
	profile := mediaProfile(testCamera("cam0", 768))
	if err := manager.ApplyProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	observer := &fakeGadgetObserver{}
	manager.SetObserver(observer)
	ops.SetUDC()

	if _, err := manager.StartFunctionFS(context.Background(), testFunctionFS()); !errors.Is(err, ErrUDCCount) {
		t.Fatalf("start err = %v, want UDC count error", err)
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.suspend != 1 || !slices.Equal(observer.profiles, []string{profile.Name}) {
		t.Fatalf("observer suspend/profiles = %d/%v", observer.suspend, observer.profiles)
	}
}

func TestSetMediaSlotsPersistsACompiledProfile(t *testing.T) {
	manager, ops := newTestManager(t)
	manager.caps = staticV1
	observer := &fakeGadgetObserver{}
	manager.SetObserver(observer)

	if err := manager.SetMediaSlots(context.Background(), []string{"Desk Camera"}, []string{"Desk Microphone"}, nil); err != nil {
		t.Fatal(err)
	}
	active, err := manager.store.Active()
	if err != nil || active != ProfileCurrent {
		t.Fatalf("active = %q, err = %v", active, err)
	}
	profile, err := manager.store.LoadProfile(ProfileCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Functions) != 5 || profile.Functions[3].Video.FunctionName != "Desk Camera" || profile.Functions[4].Audio.FunctionName != "Desk Microphone" {
		t.Fatalf("media functions = %+v", profile.Functions)
	}
	links := ops.Links()
	if _, ok := links[configPrefix+"/uvc.cam0"]; !ok {
		t.Fatal("camera is not linked")
	}
	if _, ok := links[configPrefix+"/uac2.mic0"]; !ok {
		t.Fatal("microphone is not linked")
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if !slices.Equal(observer.profiles, []string{ProfileCurrent}) {
		t.Fatalf("observer profiles = %v", observer.profiles)
	}
}

func TestSetMediaSlotsRejectsImpossibleProfileWithoutChangingActive(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.caps = staticV1
	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetMediaSlots(context.Background(), []string{"Camera 1", "Camera 2"}, nil, nil); !errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("SetMediaSlots() = %v, want endpoint refusal", err)
	}
	active, err := manager.store.Active()
	if err != nil || active != ProfileStandard {
		t.Fatalf("active = %q, err = %v", active, err)
	}
}

func namingCaps() CapabilityTable {
	table := staticV1.clone()
	table.Source = SourceProbeV1
	table.Functions[FunctionUVC].Attributes[UVCAttrFunctionName] = true
	table.Functions[FunctionUAC2].Attributes[UAC2AttrFunctionName] = true
	return table
}

func named(function Function, name string) Function {
	if function.Video != nil {
		video := *function.Video
		video.HostName = &name
		function.Video = &video
	}
	if function.Audio != nil {
		audio := *function.Audio
		audio.HostName = &name
		function.Audio = &audio
	}
	return function
}

func TestMediaHostNamesAreWrittenBeforeTheLink(t *testing.T) {
	profile := mediaProfile(named(testCamera("cam0", 768), "Desk Camera"), named(testMicrophone("mic0"), "Desk Microphone"))
	plan, err := Compile(profile, namingCaps())
	if err != nil {
		t.Fatal(err)
	}

	writes := map[string]string{}
	lastAttribute := map[string]int{}
	linkAt := map[string]int{}
	for i, op := range plan.Ops {
		for _, function := range []string{"uvc.cam0", "uac2.mic0"} {
			if op.Kind == OpWrite && strings.HasPrefix(op.Path, functionsDir+"/"+function+"/") {
				writes[op.Path] = string(op.Data)
				lastAttribute[function] = i
			}
			if op.Path == configPrefix+"/"+function {
				linkAt[function] = i
			}
		}
	}
	for path, want := range map[string]string{
		"functions/uvc.cam0/function_name":  "Desk Camera\n",
		"functions/uac2.mic0/function_name": "Desk Microphone\n",
	} {
		if got := writes[path]; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	for _, function := range []string{"uvc.cam0", "uac2.mic0"} {
		if linkAt[function] <= lastAttribute[function] {
			t.Fatalf("%s link op %d precedes final attribute op %d, which the kernel refuses with -EBUSY",
				function, linkAt[function], lastAttribute[function])
		}
	}
	want := map[string]string{"uvc.cam0": "Desk Camera", "uac2.mic0": "Desk Microphone"}
	for function, name := range want {
		if plan.MediaNames[function] != name {
			t.Fatalf("plan media name for %s = %q, want %q", function, plan.MediaNames[function], name)
		}
	}
}

// A slot the operator has never named still reports the name its host reads,
// which is the kernel's, so the panel can show the change before it is saved.
func TestMediaNamesFallBackToTheKernelDefaults(t *testing.T) {
	plan, err := Compile(mediaProfile(testCamera("cam0", 768), testMicrophone("mic0")), namingCaps())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"uvc.cam0": uvcDefaultName, "uac2.mic0": uac2DefaultName}
	for function, name := range want {
		if plan.MediaNames[function] != name {
			t.Fatalf("plan media name for %s = %q, want %q", function, plan.MediaNames[function], name)
		}
	}
	if plan, err := Compile(mediaProfile(testCamera("cam0", 768)), staticV1); err != nil || plan.MediaNames != nil {
		t.Fatalf("media names on a kernel without the attribute = %v, err = %v", plan.MediaNames, err)
	}
}

func TestMediaHostNameIsRefusedWithoutTheKernelAttribute(t *testing.T) {
	for _, tt := range []struct {
		name     string
		function Function
	}{
		{"uvc.cam0", named(testCamera("cam0", 768), "Desk Camera")},
		{"uac2.mic0", named(testMicrophone("mic0"), "Desk Microphone")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(mediaProfile(tt.function), staticV1)
			if !errors.Is(err, ErrFunctionUnavailable) {
				t.Fatalf("Compile() = %v, want ErrFunctionUnavailable", err)
			}
			if !strings.Contains(err.Error(), tt.name) || !strings.Contains(err.Error(), "function_name") {
				t.Fatalf("err = %v, want the function and attribute named", err)
			}
		})
	}
}

func TestMediaHostNameIsBoundedLikeASlotLabel(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"81 bytes", strings.Repeat("n", 81)},
		{"control characters", "Desk\x01Camera"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Compile(mediaProfile(named(testCamera("cam0", 768), tt.value)), namingCaps()); err == nil {
				t.Fatal("host name was accepted")
			}
		})
	}
}

func TestSetMediaSlotsNamesTheHostOnlyWhereTheKernelCan(t *testing.T) {
	for _, tt := range []struct {
		name  string
		caps  CapabilityTable
		named bool
	}{
		{"probed kernel with the attribute", namingCaps(), true},
		{"kernel without it", staticV1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager, _ := newTestManager(t)
			manager.caps = tt.caps
			if err := manager.SetMediaSlots(context.Background(), []string{"Desk Camera"}, []string{"Desk Microphone"}, nil); err != nil {
				t.Fatal(err)
			}
			profile, err := manager.store.LoadProfile(ProfileCurrent)
			if err != nil {
				t.Fatal(err)
			}
			video, audio := profile.Functions[3].Video.HostName, profile.Functions[4].Audio.HostName
			if tt.named {
				if video == nil || *video != "Desk Camera" || audio == nil || *audio != "Desk Microphone" {
					t.Fatalf("host names = %v/%v, want the slot labels", video, audio)
				}
				return
			}
			if video != nil || audio != nil {
				t.Fatalf("host names = %v/%v, want nil on a kernel that cannot carry them", video, audio)
			}
		})
	}
}
