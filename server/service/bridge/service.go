package bridge

import (
	"context"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/presentation"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	manager *Manager
}

// The witness targets the listener the server actually serves.
func NewService() *Service {
	conf := config.GetInstance()

	scheme, port := "http", conf.Port.Http
	if conf.Proto == "https" {
		scheme, port = "https", conf.Port.Https
	}

	return newService(NewListenerWitness(scheme, port))
}

// The wiring NewService performs once it has resolved the listener. The config
// lookup is the only part of it a test cannot run, so the gadget half lives
// here: without it Gadget is nil, enable step 13 never runs, and a transparent
// Layer-2 bridge comes up with one port.
//
// The import points bridge at presentation and never the other way, since the
// presentation manager knows nothing about a bridge.
func newService(live Liveness) *Service {
	return &Service{manager: New(Config{
		Liveness: live,
		Gadget:   presentation.GetManager(),
	})}
}

func (s *Service) GetBridge(c *gin.Context) {
	var rsp proto.Response

	status, err := s.manager.Status(c.Request.Context())
	if err != nil {
		log.Errorf("bridge: read status failed: %s", err)
		rsp.ErrRsp(c, -1, "read bridge status failed")
		return
	}

	rsp.OkRspWithData(c, &status)
}

func (s *Service) SetBridge(c *gin.Context) {
	var req proto.SetBridgeReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	// WithoutCancel because moving the management address cuts this very
	// request, and a transaction that inherits the cancellation abandons a
	// half-applied device to the watchdog rather than finishing or rolling back.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), s.manager.window)
	defer cancel()

	apply := s.manager.Disable
	if req.Enabled {
		apply = s.manager.Enable
	}

	result, err := apply(ctx)
	if err != nil {
		log.Errorf("bridge: apply failed: %s", err)
		rsp.ErrRsp(c, -2, result.Message)
		return
	}

	rsp.OkRspWithData(c, &result)
	log.Debugf("set bridge: enabled=%t state=%s uplink=%s", req.Enabled, result.State, result.Uplink)
}

func (s *Service) RevertBridge(c *gin.Context) {
	var rsp proto.Response

	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), restoreWindow)
	defer cancel()

	if err := s.manager.Revert(ctx); err != nil {
		log.Errorf("bridge: revert failed: %s", err)
		rsp.ErrRsp(c, -1, err.Error())
		return
	}

	rsp.OkRsp(c)
}
