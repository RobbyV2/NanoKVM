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
	mu      sync.Mutex
	suspend int
	applied int
}

func (f *fakeGadgetObserver) Suspend() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.suspend++
}

func (f *fakeGadgetObserver) Applied(context.Context, Profile, Plan) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applied++
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
			t.Fatalf("vendor 5.10 has no writable UVC function_name attribute: %+v", op)
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
			t.Fatalf("vendor 5.10 has no writable UAC2 function_name attribute: %+v", op)
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
