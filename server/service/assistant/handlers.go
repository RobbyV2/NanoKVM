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
	g.GET("/prompts", h.GetPrompts)
	g.POST("/prompts", h.SavePrompts)
	g.DELETE("/prompts", h.ResetPrompts)
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

// uploadBodyLimit caps the raw multipart body. The 1 MB slack covers
// multipart framing; PutStream enforces the exact total while streaming the
// file part straight to flash (never buffered in memory or /tmp).
var uploadBodyLimit int64 = maxAttachmentBytes + 1<<20

func (h *Handler) UploadAttachment(c *gin.Context) {
	var rsp proto.Response
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, uploadBodyLimit)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				rsp.ErrRsp(c, codeError, ErrAttachmentsFull.Error())
				return
			}
			rsp.ErrRsp(c, codeError, "invalid arguments")
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			part.Close()
			continue
		}
		err = h.svc.attachments.PutStream(part.FileName(), part)
		part.Close()
		if err != nil {
			respondErr(c, err, nil)
			return
		}
		rsp.OkRsp(c)
		return
	}
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	var rsp proto.Response
	if err := h.svc.attachments.Delete(c.Query("name")); err != nil {
		respondErr(c, err, nil)
		return
	}
	rsp.OkRsp(c)
}

func respondPrompts(c *gin.Context, set PromptSet, err error, action string) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	if err != nil {
		if errors.Is(err, errInvalidConfig) {
			rsp.ErrRsp(c, codeError, err.Error())
			return
		}
		log.Errorf("assistant: %s prompts: %v", action, err)
		rsp.ErrRsp(c, codeError, action+" assistant prompts failed")
		return
	}
	rsp.OkRspWithData(c, set)
}

func (h *Handler) GetPrompts(c *gin.Context) {
	set, err := loadPromptSet()
	respondPrompts(c, set, err, "get")
}

func (h *Handler) SavePrompts(c *gin.Context) {
	var req promptsFile
	if err := c.ShouldBindJSON(&req); err != nil {
		var rsp proto.Response
		rsp.ErrRsp(c, codeError, "invalid arguments")
		return
	}
	set, err := savePrompts(req.Entries)
	respondPrompts(c, set, err, "save")
}

func (h *Handler) ResetPrompts(c *gin.Context) {
	set, err := resetPrompts()
	respondPrompts(c, set, err, "reset")
}
