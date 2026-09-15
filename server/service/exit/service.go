package exit

import (
	"net/http"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Service is the gin surface: the admin API under /api/extensions/exit (JWT,
// admin role, applied by the router) and the token-gated group under /exit.
type Service struct {
	manager *Manager
}

func NewService(manager *Manager) *Service {
	return &Service{manager: manager}
}

func (s *Service) slot(c *gin.Context) (Slot, bool) {
	slot, err := ParseSlot(c.Param("slot"))
	if err != nil {
		var rsp proto.Response
		rsp.ErrRsp(c, -1, "invalid slot")
		return Slot{}, false
	}
	return slot, true
}

func (s *Service) GetSlots(c *gin.Context) {
	var rsp proto.Response
	rsp.OkRspWithData(c, s.manager.Slots())
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	status, err := s.manager.Status(slot)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRspWithData(c, status)
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	cfg, err := s.manager.Config(slot)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRspWithData(c, cfg)
}

func (s *Service) SetConfig(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	var req proto.SetExitConfigReq
	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := s.manager.SetConfig(c.Request.Context(), slot, req); err != nil {
		log.Errorf("exit: slot %s set config: %s", slot.ID, err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) Enable(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	if err := s.manager.Enable(c.Request.Context(), slot); err != nil {
		log.Errorf("exit: slot %s enable: %s", slot.ID, err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	status, _ := s.manager.Status(slot)
	rsp.OkRspWithData(c, status)
}

func (s *Service) Disable(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	if err := s.manager.Disable(c.Request.Context(), slot); err != nil {
		log.Errorf("exit: slot %s disable: %s", slot.ID, err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	status, _ := s.manager.Status(slot)
	rsp.OkRspWithData(c, status)
}

func (s *Service) RegenerateToken(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	token, err := s.manager.RegenerateToken(c.Request.Context(), slot)
	if err != nil {
		log.Errorf("exit: slot %s regenerate token: %s", slot.ID, err)
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	rsp.OkRspWithData(c, gin.H{"token": token})
}

func (s *Service) Disconnect(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	if err := s.manager.Disconnect(slot); err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) GetCommands(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	commands, err := s.manager.Commands(slot, c.Request)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRspWithData(c, commands)
}

func (s *Service) GetLogs(c *gin.Context) {
	var rsp proto.Response
	slot, ok := s.slot(c)
	if !ok {
		return
	}
	logs, err := s.manager.Logs(slot)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRspWithData(c, logs)
}

// Gate serves /exit/:slot and /exit/:slot/*rest. Every rejection is the 404
// gin writes for an unknown route, byte for byte: text/plain, the default
// body, nothing else (D10).
func (s *Service) Gate(c *gin.Context) {
	if s.manager.Gate(c.Writer, c.Request, c.Param("slot"), c.Param("rest")) {
		return
	}
	NotFound(c)
}

// NotFound mirrors gin's serveError for StatusNotFound.
func NotFound(c *gin.Context) {
	c.Data(http.StatusNotFound, "text/plain", []byte(default404Body))
	c.Abort()
}

const default404Body = "404 page not found"
