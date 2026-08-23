package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/presentation"
)

const imageDirectory = "/data"

func (s *Service) GetImages(c *gin.Context) {
	var rsp proto.Response
	var images []string

	err := filepath.Walk(imageDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			name := strings.ToLower(info.Name())
			if strings.HasSuffix(name, ".iso") || strings.HasSuffix(name, ".img") {
				images = append(images, path)
			}
		}

		return nil
	})
	if err != nil {
		rsp.ErrRsp(c, -2, "get images failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetImagesRsp{
		Files: images,
	})
	log.Debugf("get images success, total %d", len(images))
}

func (s *Service) MountImage(c *gin.Context) {
	var req proto.MountImageReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	manager := hid.Manager()

	// The LUN attributes exist only while the disk function does, and writing
	// them blind is what surfaces an opaque -2 when it does not.
	snapshot, err := manager.Snapshot()
	if err != nil {
		log.Errorf("read usb gadget failed: %s", err)
		rsp.ErrRsp(c, -2, "read usb gadget failed")
		return
	}
	if !snapshot.HasDisk() {
		rsp.ErrRsp(c, -2, "usb disk is disabled")
		return
	}

	if err := manager.SetLUN(c.Request.Context(), presentation.LUN{File: req.File, CDROM: req.Cdrom}); err != nil {
		log.Errorf("mount image %s failed: %s", req.File, err)
		// Mounting an image rebinds the UDC, so a passthrough session refuses
		// it. "mount image failed" hides the one thing that would fix it.
		if errors.Is(err, presentation.ErrUDCLoaned) {
			rsp.ErrRsp(c, -2, err.Error())
			return
		}
		rsp.ErrRsp(c, -2, "mount image failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("mount image %s success", req.File)
}

func (s *Service) GetMountedImage(c *gin.Context) {
	var rsp proto.Response

	mode, err := hid.GetMode()
	if err != nil {
		rsp.ErrRsp(c, -2, "get HID mode failed")
		return
	}

	if mode == hid.ModeHidOnly {
		rsp.OkRspWithData(c, &proto.GetMountedImageRsp{
			File: "",
		})
		return
	}

	lun, err := hid.Manager().LUN()
	if err != nil {
		log.Errorf("read mounted image failed: %s", err)
		rsp.ErrRsp(c, -2, "read failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetMountedImageRsp{
		File: lun.File,
	})
}

func (s *Service) GetCdRom(c *gin.Context) {
	var rsp proto.Response

	lun, err := hid.Manager().LUN()
	if err != nil {
		log.Errorf("read cdrom flag failed: %s", err)
		rsp.ErrRsp(c, -1, "read failed")
		return
	}

	var cdrom int64
	if lun.CDROM {
		cdrom = 1
	}

	rsp.OkRspWithData(c, &proto.GetCdRomRsp{
		Cdrom: cdrom,
	})
}

func (s *Service) DeleteImage(c *gin.Context) {
	var req proto.DeleteImageReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	filename := strings.ToLower(req.File)
	validPrefix := strings.HasPrefix(filename, imageDirectory)
	validSuffix := strings.HasSuffix(filename, ".iso") || strings.HasSuffix(filename, ".img")

	if !validPrefix || !validSuffix {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if err := os.Remove(req.File); err != nil {
		rsp.ErrRsp(c, -3, "remove file failed")
		log.Errorf("failed to remove file %s: %s", req.File, err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("delete image %s success", req.File)
}
