package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/sources"
	"NanoKVM-Server/service/webusb"

	"github.com/gin-gonic/gin"
)

func sourcesRouter(r *gin.Engine, service *sources.Service) {
	service.SetUSBBackend(webusb.NewBackend())
	// Reading the slot state is part of watching the console. Binding a browser
	// camera, microphone, speaker or USB device into a slot reaches the target
	// machine the way keyboard and mouse input does, so it follows the control
	// grant; the source socket is the transport for exactly those bindings.
	view := r.Group("/api").Use(middleware.CheckToken(), middleware.RequirePermission(authn.PermConsoleView))
	view.GET("/sources", service.Get)
	view.GET("/sources/events", service.Events)

	control := r.Group("/api").Use(middleware.CheckToken(), middleware.RequirePermission(authn.PermConsoleControl))
	control.GET("/sources/ws", service.SourceSocket)
	control.POST("/sources/bindings", service.Claim)
	control.POST("/sources/bindings/:sink/resume", service.Resume)
	control.POST("/sources/bindings/:sink/takeover", service.Takeover)
	control.DELETE("/sources/bindings/:sink", service.Release)

	admin := r.Group("/api").Use(middleware.CheckToken(), middleware.RequireRole(authn.RoleAdmin))
	admin.PUT("/sources/sinks", service.SetSinks)
	admin.DELETE("/sources/bindings", service.DisconnectAll)
}
