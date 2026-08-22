package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/authn"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/edid"
	"NanoKVM-Server/service/vm"
)

func vmRouter(r *gin.Engine) {
	service := vm.NewService()
	// A flash and the capture daemon both drive the LT6911 at 0x2b on /dev/i2c-4.
	edidService := edid.NewService(edid.CaptureGuard{
		Disable: vm.DisableHdmiCapture,
		Enable:  vm.EnableHdmiCapture,
	})

	api := r.Group("/api").Use(middleware.CheckToken())
	admin := r.Group("/api").Use(
		middleware.CheckToken(),
		middleware.RequireRole(authn.RoleAdmin),
	)

	api.GET("/vm/info", service.GetInfo)         // get device information
	api.GET("/vm/hardware", service.GetHardware) // get hardware version

	api.POST("/vm/gpio", service.SetGpio)     // update gpio
	api.GET("/vm/gpio", service.GetGpio)      // get gpio
	api.POST("/vm/screen", service.SetScreen) // update screen

	admin.GET("/vm/terminal", service.Terminal) // web terminal

	admin.GET("/vm/script", service.GetScripts)           // get script
	admin.POST("/vm/script/upload", service.UploadScript) // upload script
	admin.POST("/vm/script/run", service.RunScript)       // run script
	admin.DELETE("/vm/script", service.DeleteScript)      // delete script

	admin.GET("/vm/device/virtual", service.GetVirtualDevice)     // get virtual device
	admin.POST("/vm/device/virtual", service.UpdateVirtualDevice) // update virtual device

	admin.GET("/vm/memory/limit", service.GetMemoryLimit)  // get memory limit
	admin.POST("/vm/memory/limit", service.SetMemoryLimit) // set memory limit

	admin.GET("/vm/oled", service.GetOLED)  // get OLED configuration
	admin.POST("/vm/oled", service.SetOLED) // set OLED configuration

	// Only supported by PCIe version
	api.GET("/vm/hdmi", service.GetHdmiState)           // get HDMI state
	api.POST("/vm/hdmi/reset", service.ResetHdmi)       // reset hdmi
	admin.POST("/vm/hdmi/enable", service.EnableHdmi)   // enable hdmi
	admin.POST("/vm/hdmi/disable", service.DisableHdmi) // disable hdmi
	admin.POST("/vm/hdmi/timeout", service.SetHdmiIdleTimeout)

	admin.GET("/vm/edid", edidService.GetEdid)               // last verified flash, decoded, plus preflight
	admin.GET("/vm/edid/profiles", edidService.GetProfiles)  // shipped profile library
	admin.POST("/vm/edid/decode", edidService.DecodeEdid)    // decode an upload, no side effects
	admin.POST("/vm/edid/apply", edidService.ApplyEdid)      // validate, lock, flash, archive
	admin.POST("/vm/edid/restore", edidService.RestoreEdid)  // re-flash the factory image or a backup
	admin.GET("/vm/edid/download", edidService.DownloadEdid) // the active bytes
	admin.GET("/vm/edid/backup", edidService.DownloadBackup) // a named history entry

	admin.GET("/vm/ssh", service.GetSSHState)         // get SSH state
	admin.POST("/vm/ssh/enable", service.EnableSSH)   // enable SSH
	admin.POST("/vm/ssh/disable", service.DisableSSH) // disable SSH

	admin.GET("/vm/swap", service.GetSwap)  // get swap file size
	admin.POST("/vm/swap", service.SetSwap) // set swap file size

	admin.GET("/vm/mouse-jiggler", service.GetMouseJiggler)   // get mouse jiggler
	admin.POST("/vm/mouse-jiggler/", service.SetMouseJiggler) // set mouse jiggler

	api.GET("/vm/hostname", service.GetHostname)    // Get Hostname
	admin.POST("/vm/hostname", service.SetHostname) // Set Hostname

	api.GET("/vm/web-title", service.GetWebTitle)    // Get web title
	admin.POST("/vm/web-title", service.SetWebTitle) // Set web title

	admin.GET("/vm/mdns", service.GetMdnsState)         // get mDNS state
	admin.POST("/vm/mdns/enable", service.EnableMdns)   // enable mDNS
	admin.POST("/vm/mdns/disable", service.DisableMdns) // disable mDNS

	admin.POST("/vm/tls", service.SetTls) // enable/disable TLS

	admin.GET("/vm/autostart", service.GetAutostart)              // get autostart list
	admin.GET("/vm/autostart/:name", service.GetAutostartContent) // get autostart content
	admin.DELETE("/vm/autostart/:name", service.DeleteAutostart)  // delete autostart script
	admin.POST("/vm/autostart/:name", service.UploadAutostart)    // upload autostart script

	admin.POST("/vm/system/reboot", service.Reboot) // reboot system
}
