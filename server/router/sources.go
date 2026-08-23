package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/sources"

	"github.com/gin-gonic/gin"
)

func sourcesRouter(r *gin.Engine) {
	service := sources.NewService()
	api := r.Group("/api").Use(middleware.CheckToken())
	admin := r.Group("/api").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)

	api.GET("/sources", service.Get)
	api.GET("/sources/events", service.Events)
	api.GET("/sources/ws", service.SourceSocket)
	api.DELETE("/sources/bindings/:sink", service.Release)

	admin.PUT("/sources/sinks", service.SetSinks)
	admin.DELETE("/sources/bindings", service.DisconnectAll)
}
