package bridge

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/audit"
	"NanoKVM-Server/service/presentation"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	manager *Manager
}

var (
	witnessOnce sync.Once
	witness     *ListenerWitness
)

// One witness per process. The middleware that records and the transaction that
// reads have to be the same object: two would leave gate three consulting a map
// nothing ever writes to. The scheme and port target the listener the server
// actually serves, which is what the self-connect fallback dials.
func Witness() *ListenerWitness {
	witnessOnce.Do(func() {
		conf := config.GetInstance()

		scheme, port := "http", conf.Port.Http
		if conf.Proto == "https" {
			scheme, port = "https", conf.Port.Https
		}
		witness = NewListenerWitness(scheme, port)
	})
	return witness
}

// RecordListener is the strong form of gate three. http.LocalAddrContextKey
// carries the address the connection was accepted on, so a request from a real
// client is proof that the management plane answered at that address over the
// wire, which is the one thing a gateway ping and a self-connect cannot show.
func RecordListener(w *ListenerWitness) gin.HandlerFunc {
	return func(c *gin.Context) {
		if addr, ok := c.Request.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
			w.Record(addr.String())
		}
		c.Next()
	}
}

func NewService() *Service {
	return newService(Witness())
}

// The wiring NewService performs once it has resolved the listener. The config
// lookup is the only part of it a test cannot run, so the gadget half lives
// here: without it Gadget is nil, enable step 13 never runs, and a transparent
// Layer-2 bridge comes up with one port.
//
// The import points bridge at presentation and never the other way, since the
// presentation manager knows nothing about a bridge.
func newService(live Liveness) *Service {
	service := &Service{manager: New(Config{
		Liveness: live,
		Gadget:   presentation.GetManager(),
	})}

	// Before any route is registered, so the first GET already reports what the
	// boot check found. S29bridge leaves its record here for exactly this read.
	ctx, cancel := context.WithTimeout(context.Background(), restoreWindow)
	defer cancel()

	if recovered, err := service.manager.RecoverPending(ctx); err != nil {
		log.Errorf("bridge: boot recovery failed: %s", err)
	} else if recovered {
		log.Warnf("bridge: adopted the outcome of an apply that never finished")
	}
	return service
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

	apply, action := s.manager.Disable, "bridge.disable"
	if req.Enabled {
		apply, action = s.manager.Enable, "bridge.enable"
	}

	result, err := apply(ctx)
	// A rolled-back or failed apply comes back without an error, because the
	// caller is told what the device settled on rather than that a call failed.
	outcome := err
	if outcome == nil && result.State != proto.BridgeEnabled && result.State != proto.BridgeDisabled {
		outcome = errors.New(result.Message)
	}
	principal, _ := middleware.CurrentPrincipal(c)
	audit.Record(principal, action, string(result.State), outcome)
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

	principal, _ := middleware.CurrentPrincipal(c)
	err := s.manager.Revert(ctx)
	audit.Record(principal, "bridge.revert", "", err)
	if err != nil {
		log.Errorf("bridge: revert failed: %s", err)
		rsp.ErrRsp(c, -1, err.Error())
		return
	}

	rsp.OkRsp(c)
}
