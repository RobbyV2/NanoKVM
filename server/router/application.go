package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/application"

	"github.com/gin-gonic/gin"
)

func applicationRouter(r *gin.Engine) {
	service := application.NewService()
	api := r.Group("/api").Use(middleware.CheckToken(), middleware.RequirePermission(authn.PermUpdate))
	// Startup diagnostics and the A/B kernel slots have no grant of their own in
	// the permission catalog, so they stay with the administrator role.
	admin := r.Group("/api").Use(middleware.CheckToken(), middleware.RequireRole(authn.RoleAdmin))

	admin.GET("/application/startup", service.GetStartup)          // get startup subsystem status
	api.GET("/application/version", service.GetVersion)            // get application version
	api.POST("/application/update", service.Update)                // update application
	api.POST("/application/update/offline", service.OfflineUpdate) // update application offline

	admin.GET("/application/kernel", service.GetKernel)                // get A/B kernel slot state
	admin.POST("/application/kernel/dismiss", service.DismissRollback) // acknowledge a kernel rollback

	api.GET("/application/preview", service.GetPreview)  // get preview updates state
	api.POST("/application/preview", service.SetPreview) // set preview updates state

	api.GET("/application/update-server", service.GetUpdateServer)  // get custom update server
	api.POST("/application/update-server", service.SetUpdateServer) // set custom update server
}
