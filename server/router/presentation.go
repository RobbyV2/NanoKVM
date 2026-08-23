package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/presentation"

	"github.com/gin-gonic/gin"
)

func presentationRouter(r *gin.Engine) {
	service := presentation.NewService()
	admin := r.Group("/api/presentation").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)

	admin.GET("/status", service.GetStatus)
	admin.PUT("/config/preview", service.PreviewProfile)
	admin.PUT("/config/apply", service.ApplyProfile)
	admin.POST("/rollback", service.RollbackProfile)

	admin.GET("/profiles", service.GetProfiles)
	admin.POST("/profiles", service.CreateProfile)
	admin.POST("/profiles/import", service.ImportProfile)
	admin.GET("/profiles/:id", service.GetProfile)
	admin.PUT("/profiles/:id", service.UpdateProfile)
	admin.DELETE("/profiles/:id", service.DeleteProfile)
	admin.POST("/profiles/:id/clone", service.CloneProfile)
	admin.POST("/profiles/:id/validate", service.ValidateProfile)
	admin.GET("/profiles/:id/export", service.ExportProfile)
}
