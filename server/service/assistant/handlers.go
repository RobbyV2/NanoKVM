package assistant

import (
	"errors"
	"net/http"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	codeError    = -1
	codeDisabled = -2
	codeNoAnswer = -3
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(g gin.IRoutes) {
	g.GET("/config", h.GetConfig)
	g.POST("/config", h.SetConfig)
	g.POST("/ask", h.Ask)
	g.GET("/context", h.ContextCount)
	g.POST("/context", h.AddContext)
	g.DELETE("/context", h.ClearContexts)
	g.GET("/screenshot", h.Screenshot)
	g.POST("/reasoning", h.Reasoning)
	g.GET("/attachments", h.ListAttachments)
	g.POST("/attachments", h.UploadAttachment)
	g.DELETE("/attachments", h.DeleteAttachment)
}

func respondErr(c *gin.Context, err error, data any) {
	var rsp proto.Response
	rsp.Data = data
	switch {
	case errors.Is(err, ErrDisabled):
		rsp.ErrRsp(c, codeDisabled, err.Error())
	case errors.Is(err, ErrNoAnswer):
		rsp.ErrRsp(c, codeNoAnswer, err.Error())
	default:
		rsp.ErrRsp(c, codeError, err.Error())
	}
}

func (h *Handler) GetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	cfg, err := h.svc.loadConfig()
	if err != nil {
		log.Errorf("assistant: load config: %v", err)
		rsp.ErrRsp(c, codeError, "get assistant config failed")
		return
	}
	rsp.OkRspWithData(c, publicConfig(cfg))
}

func (h *Handler) SetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	var u ConfigUpdate
	if err := c.ShouldBindJSON(&u); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	cfg, err := updateConfig(func(cfg Config) (Config, error) { return applyUpdate(cfg, u) })
	if err != nil {
		if !errors.Is(err, errInvalidConfig) {
			log.Errorf("assistant: save config: %v", err)
		}
		rsp.ErrRsp(c, codeError, err.Error())
		return
	}
	rsp.OkRspWithData(c, publicConfig(cfg))
}

func (h *Handler) Ask(c *gin.Context) {
	var rsp proto.Response
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	res, err := h.svc.Ask(c.Request.Context(), req)
	if err != nil {
		log.Warnf("assistant: %s ask: %v (route: %s)", req.Kind, err, res.Route)
		respondErr(c, err, res)
		return
	}
	rsp.OkRspWithData(c, res)
}

type contextRequest struct {
	Crop *Crop `json:"crop"`
}

func (h *Handler) AddContext(c *gin.Context) {
	var rsp proto.Response
	var req contextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	n, err := h.svc.AddContext(c.Request.Context(), req.Crop)
	if err != nil {
		respondErr(c, err, gin.H{"count": n})
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": n})
}

func (h *Handler) ClearContexts(c *gin.Context) {
	var rsp proto.Response
	if err := h.svc.ClearContexts(); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": 0})
}

func (h *Handler) ContextCount(c *gin.Context) {
	var rsp proto.Response
	n, err := h.svc.ContextCount()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, gin.H{"count": n})
}

func (h *Handler) Screenshot(c *gin.Context) {
	data, err := h.svc.Screenshot(c.Request.Context())
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/jpeg", data)
}

type reasoningRequest struct {
	Direction string `json:"direction"`
}

func (h *Handler) Reasoning(c *gin.Context) {
	var rsp proto.Response
	var req reasoningRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.Direction != "up" && req.Direction != "down") {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	res, err := h.svc.AdjustReasoning(req.Direction == "up")
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, res)
}

func (h *Handler) ListAttachments(c *gin.Context) {
	var rsp proto.Response
	list, err := h.svc.attachments.List()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRspWithData(c, list)
}

func (h *Handler) UploadAttachment(c *gin.Context) {
	var rsp proto.Response
	file, err := c.FormFile("file")
	if err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	f, err := file.Open()
	if err != nil {
		respondErr(c, err, nil)
		return
	}
	defer f.Close()
	if err := h.svc.attachments.Put(file.Filename, f); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRsp(c)
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	var rsp proto.Response
	if err := h.svc.attachments.Delete(c.Query("name")); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRsp(c)
}
