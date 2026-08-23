package router

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/picoclaw"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Init(r *gin.Engine) {
	web(r)
	server(r)
	log.Debugf("router init done")
}

func web(r *gin.Engine) {
	execPath, err := os.Executable()
	if err != nil {
		panic("invalid executable path")
	}

	execDir := filepath.Dir(execPath)
	webPath := fmt.Sprintf("%s/web", execDir)

	r.Use(static.Serve("/", static.LocalFile(webPath, true)))
}

func server(r *gin.Engine) {
	control := controlmode.GetManager()
	picoclawService := picoclaw.NewService(control)
	sourceService := sources.NewService()
	mediaManager := media.NewManager(sourceService.Registry())
	sourceService.SetIngress(mediaManager)
	presentationManager := presentation.GetManager()
	sourceService.SetSlotManager(presentationManager)
	presentationManager.SetObserver(mediaManager)
	if err := presentationManager.RefreshObserver(context.Background()); err != nil {
		log.Debugf("media gadget unavailable: %s", err)
	}

	authRouter(r)
	applicationRouter(r)
	vmRouter(r)
	streamRouter(r)
	storageRouter(r)
	networkRouter(r)
	presentationRouter(r)
	hidRouter(r)
	controlRouter(r, control, picoclawService)
	mcpRouter(r, control, picoclawService)
	picoclawRouter(r, picoclawService)
	wsRouter(r)
	sourcesRouter(r, sourceService)
	downloadRouter(r)
	extensionsRouter(r)
}

func LoopbackHTTPAllowedPaths() []string {
	paths := PicoclawLoopbackHTTPAllowedPaths()
	paths = append(paths, HIDLoopbackHTTPAllowedPaths()...)
	return paths
}
