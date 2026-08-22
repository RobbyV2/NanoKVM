package hid

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/inputcontrol"
	"NanoKVM-Server/service/presentation"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	ModeNormal  = presentation.ModeNormal
	ModeHidOnly = presentation.ModeHIDOnly

	ModeNormalScript  = "/kvmapp/system/init.d/S03usbdev"
	ModeHidOnlyScript = "/kvmapp/system/init.d/S03usbhid"

	USBDevScript = "/etc/init.d/S03usbdev"
)

var (
	managerOnce sync.Once
	manager     *presentation.Manager
)

// hid imports presentation and never the other way round, so the HID quiesce
// bracket the manager wraps every gadget mutation in is injected from here.
func Manager() *presentation.Manager {
	managerOnce.Do(func() {
		manager = presentation.GetManager()
		manager.SetHID(GetHid())
	})
	return manager
}

func (s *Service) GetHidMode(c *gin.Context) {
	var rsp proto.Response

	mode, err := GetMode()
	if err != nil {
		rsp.ErrRsp(c, -1, "get HID mode failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetHidModeRsp{
		Mode: mode,
	})
	log.Debugf("get hid mode: %s", mode)
}

func (s *Service) GetKeyboardLedStatus(c *gin.Context) {
	var rsp proto.Response
	status := GetKeyboardLedStatus()
	updatedAt := ""
	if !status.UpdatedAt.IsZero() {
		updatedAt = status.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}

	rsp.OkRspWithData(c, &proto.GetKeyboardLedStatusRsp{
		NumLock:    status.NumLock,
		CapsLock:   status.CapsLock,
		ScrollLock: status.ScrollLock,
		Known:      status.Known,
		UpdatedAt:  updatedAt,
	})
}

func (s *Service) SetHidMode(c *gin.Context) {
	var req proto.SetHidModeReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if req.Mode != ModeNormal && req.Mode != ModeHidOnly {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	mode, err := GetMode()
	if err != nil {
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}
	if req.Mode == mode {
		rsp.OkRsp(c)
		return
	}

	srcScript := ModeNormalScript
	if req.Mode == ModeHidOnly {
		srcScript = ModeHidOnlyScript
	}

	// The init script is still the boot-time configurator and system_init.cpp
	// restores its kvmapp copy on every update, so the mode is staged on disk
	// first and applied live second.
	if err := copyModeFile(srcScript); err != nil {
		rsp.ErrRsp(c, -3, "operation failed")
		return
	}
	if err := Manager().SetMode(c.Request.Context(), req.Mode); err != nil {
		log.Errorf("failed to apply hid mode %s: %s", req.Mode, err)
	}

	rsp.OkRsp(c)

	// Reboot stays: the profiles' report_desc differ, report_desc is -EBUSY once linked, and R1.1 forbids unlinking hid.*.
	log.Println("reboot system...")
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("reboot").Run()
}

func (s *Service) ResetHid(c *gin.Context) {
	var rsp proto.Response

	manual := s.newManualSession()
	defer manual.Close()
	reservation, err := manual.Reserve(c.Request.Context(), inputcontrol.ManualRelativeMouse, false, nil)
	if err != nil {
		log.Errorf("failed to acquire manual control for HID reset: %v", err)
		rsp.ErrRsp(c, -1, "HID control is busy")
		return
	}
	err = manual.Execute(ResetUSBPHY)
	reservation.Complete(err == nil)
	if err != nil {
		log.Errorf("failed to reset hid: %v", err)
		rsp.ErrRsp(c, -1, "failed to reset hid")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("reset hid success")
}

func (s *Service) RecoverUSB(c *gin.Context) {
	var rsp proto.Response

	if err := ResetUSBPHY(); err != nil {
		log.Errorf("failed to recover usb: %v", err)
		rsp.ErrRsp(c, -1, "failed to recover usb")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("recover usb success")
}

func ResetUSBPHY() error {
	return Manager().ResetPHY(context.Background())
}

func copyModeFile(srcScript string) error {
	srcFile, err := os.Open(srcScript)
	if err != nil {
		log.Errorf("failed to open %s: %s", srcScript, err)
		return err
	}
	defer func() {
		_ = srcFile.Close()
	}()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		log.Errorf("failed to get %s info: %s", srcScript, err)
		return err
	}

	tmpFile, err := os.CreateTemp("/etc/init.d/", ".S03usbdev-")
	if err != nil {
		log.Errorf("failed to create temp %s: %s", USBDevScript, err)
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	log.Debugf("create temporary file: %s", tmpPath)

	if err := tmpFile.Chmod(srcInfo.Mode()); err != nil {
		_ = tmpFile.Close()
		log.Errorf("failed to set %s mode: %s", tmpPath, err)
		return err
	}

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		_ = tmpFile.Close()
		log.Errorf("failed to copy %s: %s", srcScript, err)
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		log.Errorf("failed to sync %s: %s", tmpPath, err)
		return err
	}

	if err := tmpFile.Close(); err != nil {
		log.Errorf("failed to close %s: %s", tmpPath, err)
		return err
	}

	if err := os.Rename(tmpPath, USBDevScript); err != nil {
		log.Errorf("failed to rename %s: %s", tmpPath, err)
		return err
	}

	log.Debugf("copy %s to %s successful", srcScript, USBDevScript)
	return nil
}

func GetMode() (string, error) {
	mode, err := Manager().Mode()
	if err != nil {
		log.Errorf("failed to resolve hid mode: %s", err)
		return "", err
	}
	return mode, nil
}
