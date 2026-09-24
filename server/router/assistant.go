package router

import (
	"context"
	"errors"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/assistant"
	mcpservice "NanoKVM-Server/service/mcp"
	"NanoKVM-Server/service/mcp/capture/kvm"

	"github.com/gin-gonic/gin"
)

// assistantCapturer adapts the MCP capture backend (cgo) to the assistant
// package, which stays cgo-free so its tests run anywhere.
type assistantCapturer struct {
	snapshotter mcpservice.Snapshotter
}

func (a assistantCapturer) Capture(ctx context.Context) (assistant.Frame, error) {
	snap, err := a.snapshotter.Capture(ctx, mcpservice.SnapshotRequest{})
	if err != nil {
		return assistant.Frame{}, err
	}
	if !snap.OK {
		return assistant.Frame{}, errors.New(snap.Message)
	}
	return assistant.Frame{JPEG: snap.JPEG, Width: snap.Width, Height: snap.Height}, nil
}

func assistantRouter(r *gin.Engine) {
	handler := assistant.NewHandler(assistant.NewService(assistantCapturer{snapshotter: kvmcapture.New()}))
	handler.Register(r.Group("/api/assistant").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	))
}
