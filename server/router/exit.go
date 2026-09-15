package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/exit"

	"github.com/gin-gonic/gin"
)

// exitRouter registers both halves of the exit tunnel's HTTP surface: the
// admin API under /api/extensions/exit behind the JWT and the admin role, and
// the token-gated group under /exit that the exit device itself talks to,
// authenticated by the slot's bearer token alone (exit-tunnel design, D10).
// GetManager has already initialised the manager by the time this runs: the
// post-attach hook in server() reaches it first.
func exitRouter(r *gin.Engine) {
	service := exit.NewService(exit.GetManager())

	admin := r.Group("/api/extensions/exit").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)
	admin.GET("/slots", service.GetSlots)                          // every slot with status
	admin.GET("/:slot/status", service.GetStatus)                  // one slot's status
	admin.GET("/:slot/config", service.GetConfig)                  // editable config, without the token
	admin.POST("/:slot/config", service.SetConfig)                 // mode, dns, mtu, allowPrivate, pinPeer
	admin.POST("/:slot/enable", service.Enable)                    // the enable transaction
	admin.POST("/:slot/disable", service.Disable)                  // the disable transaction
	admin.POST("/:slot/token/regenerate", service.RegenerateToken) // new token, sessions dropped
	admin.POST("/:slot/disconnect", service.Disconnect)            // drop the active exit
	admin.GET("/:slot/commands", service.GetCommands)              // templated commands, both modes
	admin.GET("/:slot/logs", service.GetLogs)                      // hev and wstunnel log tails

	// Outside /api, no JWT. Both patterns route to the gate so a bare slot
	// path is a 404 rather than gin's trailing-slash redirect, and every
	// rejection is byte-identical to an unknown route.
	r.GET("/exit/:slot", service.Gate)
	r.GET("/exit/:slot/*rest", service.Gate)
}
