package vm

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/vnc"
)

func (s *Service) GetVNC(c *gin.Context) {
	var rsp proto.Response

	conf, err := config.Read()
	if err != nil {
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetVNCRsp{
		Enabled: conf.VNC.Enabled,
		Port:    conf.VNC.Port,
	})
}

func (s *Service) SetVNC(c *gin.Context) {
	var req proto.SetVNCReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	conf, err := config.Read()
	if err != nil {
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}

	if req.Enabled {
		if err := vnc.GetManager().Start(); err != nil {
			log.Errorf("failed to start VNC server: %s", err)
			rsp.ErrRsp(c, -3, "operation failed")
			return
		}
	} else {
		vnc.GetManager().Stop()
	}

	conf.VNC.Enabled = req.Enabled
	if err := config.Write(conf); err != nil {
		rsp.ErrRsp(c, -4, "operation failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set VNC enabled: %t", req.Enabled)
}
