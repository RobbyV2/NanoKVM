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
		Network: snapshot.HasNetwork(),
		Disk:    snapshot.HasDisk(),
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

	toggle(&profile, req.Device, !deviceOn(snapshot, req.Device))
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
// D5: an off/on round trip loses NCM, because removing the entry also removes
// the /boot/usb.ncm mirror and nothing on disk remembers the choice.
func toggle(profile *presentation.Profile, device string, on bool) {
	functions := slices.DeleteFunc(slices.Clone(profile.Functions), func(f presentation.Function) bool {
		return deviceOf(f.Kind) == device
	})

	switch {
	case device == deviceNetwork && on:
		functions = append([]presentation.Function{presentation.NetworkFunction(presentation.FunctionRNDIS)}, functions...)
	case device == deviceDisk && on:
		functions = append(functions, presentation.DiskFunction(""))
	}

	profile.Functions = functions
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
