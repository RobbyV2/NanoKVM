package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/bridge"
	"NanoKVM-Server/service/network"
)

func networkRouter(r *gin.Engine) {
	service := network.NewService()
	bridgeService := bridge.NewService()

	r.POST("/api/network/wifi", service.ConnectWifiNoAuth)    // connect Wi-Fi without auth (only available in ap mode)
	r.POST("/api/network/wifi/verify", service.VerifyApLogin) // verify ap login

	power := r.Group("/api").Use(middleware.CheckToken(), middleware.RequirePermission(authn.PermPower))
	network := r.Group("/api").Use(middleware.CheckToken(), middleware.RequirePermission(authn.PermNetwork))
	admin := r.Group("/api").Use(middleware.CheckToken(), middleware.RequireRole(authn.RoleAdmin))

	power.POST("/network/wol", service.WakeOnLAN) // wake on lan
	power.GET("/network/wol/mac", service.GetMac) // get mac list

	power.DELETE("/network/wol/mac", service.DeleteMac)     // delete mac
	power.POST("/network/wol/mac/name", service.SetMacName) // set mac name

	network.GET("/network/wifi", service.GetWifi)                    // get Wi-Fi information
	network.POST("/network/wifi/connect", service.ConnectWifi)       // connect Wi-Fi
	network.POST("/network/wifi/disconnect", service.DisconnectWifi) // disconnect Wi-Fi
	network.GET("/network/dns", service.GetDNS)                      // get DNS configuration
	network.POST("/network/dns", service.SetDNS)                     // set DNS configuration

	// The bridge puts the gadget NIC and the uplink on one segment and can cut
	// this device off its own network; the permission catalog has no grant for
	// it, so it stays with the administrator role.
	admin.GET("/network/bridge", bridgeService.GetBridge)            // bridge state, ports, armed marker, last apply
	admin.POST("/network/bridge", bridgeService.SetBridge)           // enable or disable the bridge
	admin.POST("/network/bridge/revert", bridgeService.RevertBridge) // force-restore the snapshot
}
