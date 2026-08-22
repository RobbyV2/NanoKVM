package router

import (
	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/extensions/tailscale"
	"NanoKVM-Server/service/extensions/tunnel"

	"github.com/gin-gonic/gin"
)

func extensionsRouter(r *gin.Engine) {
	api := r.Group("/api/extensions").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)

	ts := tailscale.NewService()

	api.POST("/tailscale/install", ts.Install)     // install tailscale
	api.POST("/tailscale/uninstall", ts.Uninstall) // uninstall tailscale
	api.GET("/tailscale/status", ts.GetStatus)     // get tailscale status
	api.POST("/tailscale/up", ts.Up)               // run tailscale up
	api.POST("/tailscale/down", ts.Down)           // run tailscale down
	api.POST("/tailscale/login", ts.Login)         // tailscale login
	api.POST("/tailscale/logout", ts.Logout)       // tailscale logout
	api.POST("/tailscale/start", ts.Start)         // tailscale start
	api.POST("/tailscale/stop", ts.Stop)           // tailscale stop
	api.POST("/tailscale/restart", ts.Restart)     // tailscale restart

	names := []proto.TunnelName{proto.TunnelWstunnel, proto.TunnelNewt}
	for _, name := range names {
		tn := tunnel.NewService(name)
		group := "/tunnel/" + string(name)

		api.GET(group+"/status", tn.GetStatus)       // get tunnel status
		api.GET(group+"/config", tn.GetConfig)       // get tunnel config
		api.POST(group+"/config", tn.SetConfig)      // update tunnel config
		api.POST(group+"/start", tn.Start)           // start tunnel
		api.POST(group+"/stop", tn.Stop)             // stop tunnel
		api.POST(group+"/restart", tn.Restart)       // restart tunnel
		api.GET(group+"/logs", tn.GetLogs)           // get tunnel logs
		api.POST(group+"/binary", tn.UploadBinary)   // upload a custom tunnel binary
		api.DELETE(group+"/binary", tn.DeleteBinary) // remove the custom tunnel binary
	}

	tunnel.StartWatchdog(names)
}
