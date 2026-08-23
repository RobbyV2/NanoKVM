package presentation

import (
	"context"
	"errors"
	"testing"
)

func testFunctionFS() FunctionFS {
	return FunctionFS{Interfaces: 1, Endpoints: []FunctionFSEndpoint{
		{SourceAddress: 0x01, Address: 0x01, Transfer: EndpointBulk, MaxPacket: 64},
		{SourceAddress: 0x81, Address: 0x81, Transfer: EndpointBulk, MaxPacket: 512},
	}}
}

func TestFunctionFSProfileRetainsBootHID(t *testing.T) {
	profile := standardProfile()
	profile.Name = ProfileHybrid
	profile.BuiltIn = false
	profile.Functions = append(profile.Functions[:2], Function{Kind: FunctionFFS, Instance: "hybrid", FFS: pointer(testFunctionFS())})
	plan, err := Compile(profile, staticV0)
	if err != nil {
		t.Fatal(err)
	}
	links := map[string]bool{}
	for _, op := range plan.Ops {
		if op.Kind == OpSymlink {
			links[op.Path] = true
		}
	}
	for _, path := range []string{configPrefix + "/hid.GS0", configPrefix + "/hid.GS1", configPrefix + "/ffs.hybrid"} {
		if !links[path] {
			t.Fatalf("plan lacks %s", path)
		}
	}
	if links[configPrefix+"/hid.GS2"] {
		t.Fatal("Hybrid linked the absolute pointer")
	}
}

func TestFunctionFSTransientRestoresPersistentProfile(t *testing.T) {
	manager, ops := newTestManager(t)
	if err := manager.Apply(context.Background(), ProfileStandard); err != nil {
		t.Fatal(err)
	}
	state, err := manager.StartFunctionFS(context.Background(), testFunctionFS())
	if err != nil {
		t.Fatal(err)
	}
	active, err := manager.store.Active()
	if err != nil || active != ProfileStandard {
		t.Fatalf("transient changed active profile to %q: %v", active, err)
	}
	if _, ok := ops.Links()[configPrefix+"/ffs.hybrid"]; !ok {
		t.Fatal("FunctionFS link is absent")
	}
	if _, ok := ops.Links()[configPrefix+"/hid.GS2"]; ok {
		t.Fatal("Hybrid retained the absolute pointer")
	}
	if err := manager.Apply(context.Background(), ProfileHIDOnly); !errors.Is(err, ErrTransient) {
		t.Fatalf("concurrent apply returned %v", err)
	}
	if err := manager.StopFunctionFS(context.Background(), state.Token); err != nil {
		t.Fatal(err)
	}
	links := ops.Links()
	if _, ok := links[configPrefix+"/ffs.hybrid"]; ok {
		t.Fatal("FunctionFS link survived rollback")
	}
	if _, ok := links[configPrefix+"/hid.GS2"]; !ok {
		t.Fatal("absolute pointer was not restored")
	}
}

func TestFunctionFSRejectsWrongRollbackToken(t *testing.T) {
	manager, _ := newTestManager(t)
	state, err := manager.StartFunctionFS(context.Background(), testFunctionFS())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.StopFunctionFS(context.Background(), state.Token+1); !errors.Is(err, ErrTransient) {
		t.Fatalf("wrong token returned %v", err)
	}
	if err := manager.StopFunctionFS(context.Background(), state.Token); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionFSRecoveryRebindsAfterCancellation(t *testing.T) {
	manager, ops := newTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.RecoverFunctionFS(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("recovery returned %v", err)
	}
	if ops.Bound() != dwc2Device {
		t.Fatalf("recovery left UDC bound to %q", ops.Bound())
	}
}

func TestPersistentApplyRejectsFunctionFS(t *testing.T) {
	manager, _ := newTestManager(t)
	profile := standardProfile()
	profile.Name = ProfileHybrid
	profile.BuiltIn = false
	profile.Functions = append(profile.Functions[:2], Function{Kind: FunctionFFS, Instance: "hybrid", FFS: pointer(testFunctionFS())})
	if err := manager.ApplyProfile(context.Background(), profile); !errors.Is(err, ErrTransient) {
		t.Fatalf("persistent FunctionFS apply returned %v", err)
	}
}

func TestFunctionFSRollbackRemovesFailedLink(t *testing.T) {
	manager, ops := newTestManager(t)
	failed, err := manager.hybridProfile(testFunctionFS())
	if err != nil {
		t.Fatal(err)
	}
	recovery := recoveryPlan{profile: standardProfile()}
	recovery.plan, err = Compile(recovery.profile, staticV0)
	if err != nil {
		t.Fatal(err)
	}
	if err := ops.Symlink(functionsDir+"/ffs.hybrid", configPrefix+"/ffs.hybrid"); err != nil {
		t.Fatal(err)
	}
	if err := manager.restore(failed, recovery, dwc2Device); err != nil {
		t.Fatal(err)
	}
	if _, ok := ops.Links()[configPrefix+"/ffs.hybrid"]; ok {
		t.Fatal("failed FunctionFS link survived rollback")
	}
}

func pointer[T any](value T) *T { return &value }
