package presentation

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func testSpeaker(instance string) Function {
	return Function{
		Kind:     FunctionUAC2,
		Instance: instance,
		Audio: &AudioFunction{
			FunctionName:  "Speaker " + instance,
			PChannelMask:  0,
			PSampleRate:   48000,
			PSampleSize:   2,
			CChannelMask:  1,
			CSampleRate:   48000,
			CSampleSize:   2,
			RequestNumber: 4,
		},
	}
}

// A speaker is the c_chmask direction of f_uac2: EPOUT_EN(opts) is
// (opts)->c_chmask != 0, so the host writes and the gadget reads.
func TestSpeakerFunctionValidates(t *testing.T) {
	profile := mediaProfile(testSpeaker("spk0"))
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want a valid speaker", err)
	}
}

func TestSpeakerCostsAnOUTEndpointNotAnIN(t *testing.T) {
	used, err := AccountEndpoints([]Function{testMicrophone("mic0"), testSpeaker("spk0")}, staticV1)
	if err != nil {
		t.Fatal(err)
	}
	if used.In != 1 || used.Out != 1 {
		t.Fatalf("microphone + speaker = %+v, want {In:1 Out:1}", used)
	}
}

func TestSpeakerSeatsNoINFIFO(t *testing.T) {
	assigned, err := SeatFIFOs([]Function{testMicrophone("mic0"), testSpeaker("spk0")}, staticV1)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(assigned["uac2.mic0"], []int{128}) {
		t.Fatalf("uac2.mic0 fifos = %v, want [128]", assigned["uac2.mic0"])
	}
	if got := assigned["uac2.spk0"]; len(got) != 0 {
		t.Fatalf("uac2.spk0 fifos = %v, want none: a speaker has no IN endpoint to feed", got)
	}
}

// The whole point of the OUT direction: a speaker joins a profile that already
// spends every IN endpoint, and nothing else in it is disturbed.
func TestSpeakerFitsBesideEveryOtherFunction(t *testing.T) {
	table := staticV1.clone()
	caps := table.Functions[FunctionUVC]
	caps.Attributes = map[string]bool{UVCAttrInterruptEP: true, UVCAttrFunctionName: true}
	table.Functions[FunctionUVC] = caps

	camera := testCamera("cam0", 768)
	camera.Video.InterruptEndpoint = ptr(false)

	profile := standardProfile()
	profile.Name = "everything"
	profile.BuiltIn = false
	if err := SetHIDLayout(&profile, [][]HIDRole{{HIDRoleKeyboard, HIDRoleRelative, HIDRoleAbsolute}}); err != nil {
		t.Fatal(err)
	}
	profile.Functions = append(profile.Functions,
		Function{Kind: FunctionNCM, Instance: "usb0", Net: &NetFunction{CompatibleID: "WINNCM"}},
		Function{Kind: FunctionMassStorage, Instance: "disk0", Storage: &StorageFunction{Removable: true, File: "/dev/mmcblk0p3"}},
		camera,
		testMicrophone("mic0"),
		testSpeaker("spk0"),
	)
	profile.Normalize()

	withoutSpeaker := profile
	withoutSpeaker.Functions = slices.Clone(profile.Functions[:len(profile.Functions)-1])
	before, err := AccountEndpoints(withoutSpeaker.Functions, table)
	if err != nil {
		t.Fatalf("the profile without a speaker must already compile: %v", err)
	}

	after, err := AccountEndpoints(profile.Functions, table)
	if err != nil {
		t.Fatalf("adding a speaker broke the profile: %v", err)
	}
	if after.In != before.In {
		t.Fatalf("a speaker took %d IN endpoints from the rest of the profile", after.In-before.In)
	}
	if after.Out != before.Out+1 {
		t.Fatalf("speaker OUT cost = %d, want 1", after.Out-before.Out)
	}
	if _, err := Compile(profile, table); err != nil {
		t.Fatalf("Compile() = %v, want a plan the kernel can bind", err)
	}
}

func TestSpeakerCompileEnablesTheOUTEndpointOnly(t *testing.T) {
	plan, err := Compile(mediaProfile(testSpeaker("spk0")), staticV1)
	if err != nil {
		t.Fatal(err)
	}
	writes := map[string]string{}
	for _, op := range plan.Ops {
		if strings.HasPrefix(op.Path, "functions/uac2.spk0/") && op.Kind == OpWrite {
			writes[op.Path] = string(op.Data)
		}
	}
	for path, want := range map[string]string{
		"functions/uac2.spk0/p_chmask": "0\n",
		"functions/uac2.spk0/c_chmask": "1\n",
		"functions/uac2.spk0/c_srate":  "48000\n",
		"functions/uac2.spk0/c_ssize":  "2\n",
	} {
		if got := writes[path]; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestAudioDirectionMustMatchTheInstanceName(t *testing.T) {
	tests := []struct {
		name     string
		function Function
		want     string
	}{
		{"microphone with USB OUT", func() Function { f := testMicrophone("mic0"); f.Audio.CChannelMask = 1; return f }(), "microphone"},
		{"speaker with USB IN", func() Function { f := testSpeaker("spk0"); f.Audio.PChannelMask = 1; return f }(), "speaker"},
		{"silent function", func() Function { f := testSpeaker("spk0"); f.Audio.CChannelMask = 0; return f }(), "no USB endpoint"},
		{"speaker stereo", func() Function { f := testSpeaker("spk0"); f.Audio.CChannelMask = 3; return f }(), "speaker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := mediaProfile(tt.function)
			err := profile.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want a refusal naming %q", err, tt.want)
			}
		})
	}
}

func TestSpeakerAndMicrophoneIndicesAreCountedApart(t *testing.T) {
	paired := mediaProfile(testMicrophone("mic0"), testSpeaker("spk0"))
	if err := paired.Validate(); err != nil {
		t.Fatalf("mic0 + spk0 = %v, want valid", err)
	}
	gap := mediaProfile(testMicrophone("mic0"), testSpeaker("spk1"))
	if err := gap.Validate(); err == nil || !strings.Contains(err.Error(), "contiguous") {
		t.Fatalf("spk1 without spk0 = %v, want a contiguity refusal", err)
	}
}

func TestSetMediaSlotsBuildsSpeakers(t *testing.T) {
	manager, _ := newTestManager(t)
	manager.caps = staticV1
	if err := manager.SetMediaSlots(context.Background(), nil, []string{"Microphone 1"}, []string{"Speaker 1"}); err != nil {
		t.Fatal(err)
	}
	profile, err := manager.store.LoadProfile(ProfileCurrent)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, function := range profile.Functions {
		if function.Kind != FunctionUAC2 || function.Instance != "spk0" {
			continue
		}
		found = true
		if function.Audio.CChannelMask != 1 || function.Audio.PChannelMask != 0 {
			t.Fatalf("speaker masks = p:%d c:%d, want p:0 c:1", function.Audio.PChannelMask, function.Audio.CChannelMask)
		}
		if function.Audio.FunctionName != "Speaker 1" {
			t.Fatalf("speaker label = %q", function.Audio.FunctionName)
		}
	}
	if !found {
		t.Fatal("no uac2.spk0 function was built")
	}
	if _, err := Compile(profile, staticV1); err != nil && errors.Is(err, ErrEndpointBudget) {
		t.Fatalf("speaker slot broke the budget: %v", err)
	}
}

// A missed isochronous OUT still completes its request, and u_audio copies
// that request's buffer into the ALSA ring a second time - the target host's
// audio comes back carrying an exact repeat of the block from req_number
// milliseconds earlier. More requests in flight means fewer misses: 17.8% of
// blocks repeated at 4, 2.3% at 8, and none at all at 32.
func TestSpeakerAcceptsMoreRequestsInFlight(t *testing.T) {
	for _, number := range []uint8{2, 4, 8, 16, 32} {
		function := testSpeaker("spk0")
		function.Audio.RequestNumber = number
		profile := mediaProfile(function)
		if err := profile.Validate(); err != nil {
			t.Errorf("req_number %d: Validate() = %v, want accepted", number, err)
		}
	}
	for _, number := range []uint8{0, 1, 33} {
		function := testSpeaker("spk0")
		function.Audio.RequestNumber = number
		profile := mediaProfile(function)
		if err := profile.Validate(); err == nil {
			t.Errorf("req_number %d: Validate() = nil, want rejected", number)
		}
	}
}

// The default has to carry the measured value, or every freshly created
// profile reintroduces the repeats.
func TestDefaultAudioKeepsEnoughRequestsInFlight(t *testing.T) {
	for _, function := range []Function{defaultMicrophone(0, "Microphone 1", true), defaultSpeaker(0, "Speaker 1", true)} {
		if function.Audio.RequestNumber < 32 {
			t.Errorf("%s default req_number = %d, want at least 32", function.Instance, function.Audio.RequestNumber)
		}
	}
}
