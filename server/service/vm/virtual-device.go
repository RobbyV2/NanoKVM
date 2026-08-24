package vm

import (
	"fmt"
	"slices"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/presentation"
)

const (
	deviceNetwork = "network"
	deviceDisk    = "disk"
)

func (s *Service) GetVirtualDevice(c *gin.Context) {
	var rsp proto.Response

	snapshot, err := hid.Manager().Snapshot()
	if err != nil {
		log.Errorf("get virtual device failed: %s", err)
	}

	rsp.OkRspWithData(c, &proto.GetVirtualDeviceRsp{
		Network:   snapshot.HasNetwork(),
		Disk:      snapshot.HasDisk(),
		Protocol:  string(snapshot.NetworkKind()),
		Endpoints: proto.EndpointUsage{In: snapshot.Endpoints.In, Out: snapshot.Endpoints.Out},
		Headroom:  proto.EndpointUsage{In: snapshot.Headroom.In, Out: snapshot.Headroom.Out},
	})
	log.Debugf("get virtual device success")
}

func (s *Service) UpdateVirtualDevice(c *gin.Context) {
	var req proto.UpdateVirtualDeviceReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid argument")
		return
	}
	if req.Device != deviceNetwork && req.Device != deviceDisk {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	// A protocol only means anything for the network function, and only the two
	// the gadget layer can build are accepted, so a selector can never ask for
	// one the compiler has no case for.
	var protocol presentation.FunctionKind
	if req.Protocol != "" {
		kind, err := presentation.ParseNetworkKind(req.Protocol)
		if err != nil || req.Device != deviceNetwork {
			log.Errorf("update virtual device %s failed: protocol %q", req.Device, req.Protocol)
			rsp.ErrRsp(c, -2, "invalid arguments")
			return
		}
		protocol = kind
	}

	manager := hid.Manager()

	snapshot, err := manager.Snapshot()
	if err != nil {
		log.Errorf("update virtual device %s failed: %s", req.Device, err)
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}
	if snapshot.Mode == presentation.ModeHIDOnly {
		log.Errorf("update virtual device %s failed: hid-only mode has no %s function", req.Device, req.Device)
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}

	profile, err := editableProfile(snapshot.Active)
	if err != nil {
		log.Errorf("update virtual device %s failed: %s", req.Device, err)
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}

	if protocol != "" {
		setNetwork(&profile, protocol)
	} else {
		toggle(&profile, req.Device, !deviceOn(snapshot, req.Device))
	}
	if err := manager.ApplyProfile(c.Request.Context(), profile); err != nil {
		log.Errorf("update virtual device %s failed: %s", req.Device, err)
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}

	applied, err := manager.Snapshot()
	if err != nil {
		log.Errorf("re-read virtual device %s failed: %s", req.Device, err)
	}

	rsp.OkRspWithData(c, &proto.UpdateVirtualDeviceRsp{
		On: deviceOn(applied, req.Device),
	})
	log.Debugf("update virtual device %s success", req.Device)
}

// A built-in is reconstructed from code on every load, so an edited one would
// lose its added functions on the next load and the toggle would be a no-op.
func editableProfile(active string) (presentation.Profile, error) {
	if active == "" {
		active = presentation.ProfileStandard
	}

	profile, err := presentation.NewStore().LoadProfile(active)
	if err != nil {
		return profile, fmt.Errorf("load profile %s: %w", active, err)
	}
	if profile.Name == "" {
		return profile, fmt.Errorf("profile %s not found", active)
	}
	if profile.BuiltIn {
		profile.Name, profile.BuiltIn = presentation.ProfileCurrent, false
	}
	return profile, nil
}

// The link order {ncm|rndis}, hid.GS0-2, mass_storage.disk0 fixes
// bInterfaceNumber assignment, so the net function goes first and the disk last.
// Turning the network back on keeps whichever protocol the profile last named,
// so an off/on round trip no longer silently demotes an NCM gadget to RNDIS.
func toggle(profile *presentation.Profile, device string, on bool) {
	kind := profileNetworkKind(*profile)

	functions := slices.DeleteFunc(slices.Clone(profile.Functions), func(f presentation.Function) bool {
		return deviceOf(f.Kind) == device
	})

	switch {
	case device == deviceNetwork && on:
		functions = append([]presentation.Function{presentation.NetworkFunction(kind)}, functions...)
	case device == deviceDisk && on:
		functions = append(functions, presentation.DiskFunction(""))
	}

	profile.Functions = functions
}

// Replaces the net function rather than toggling it, so selecting a protocol on
// a gadget that already presents one is a switch and not an unmount.
func setNetwork(profile *presentation.Profile, kind presentation.FunctionKind) {
	functions := slices.DeleteFunc(slices.Clone(profile.Functions), func(f presentation.Function) bool {
		return deviceOf(f.Kind) == deviceNetwork
	})
	profile.Functions = append([]presentation.Function{presentation.NetworkFunction(kind)}, functions...)
}

// NCM is the fallback for a profile that never named a protocol. RNDIS was the
// obvious choice while it was what the shipping toggle built, but Windows has
// been withdrawing the driver and on 11 the adapter now installs and then
// carries nothing: measured on a live target, the gadget NIC took 0 packets
// from the host over an RNDIS link and passed ordinary traffic over an NCM one
// with nothing else changed. NCM is native on Windows 10 1809 and later, macOS
// and Linux, so it is the choice that works when no one has expressed a
// preference. A profile that names RNDIS still gets RNDIS - this is only the
// default, and older Windows still needs it.
func profileNetworkKind(profile presentation.Profile) presentation.FunctionKind {
	for _, f := range profile.Functions {
		if deviceOf(f.Kind) == deviceNetwork {
			return f.Kind
		}
	}
	return presentation.FunctionNCM
}

func deviceOn(snapshot presentation.Snapshot, device string) bool {
	if device == deviceNetwork {
		return snapshot.HasNetwork()
	}
	return snapshot.HasDisk()
}

func deviceOf(kind presentation.FunctionKind) string {
	switch kind {
	case presentation.FunctionNCM, presentation.FunctionRNDIS:
		return deviceNetwork
	case presentation.FunctionMassStorage:
		return deviceDisk
	default:
		return ""
	}
}
