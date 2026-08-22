package bridge

import (
	"context"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	manager *Manager
}

// The witness targets the listener the server actually serves. Nothing calls
// its Record, so the inbound gate always takes the self-connect fallback until
// the HTTP middleware is wired into it.
func NewService() *Service {
	conf := config.GetInstance()

	scheme, port := "http", conf.Port.Http
	if conf.Proto == "https" {
		scheme, port = "https", conf.Port.Https
	}

	return &Service{manager: New(Config{Liveness: NewListenerWitness(scheme, port)})}
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
