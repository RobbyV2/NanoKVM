package router

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/picoclaw"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/sources"
	"NanoKVM-Server/service/startup"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const observerRefreshBudget = 10 * time.Second

// A pull-up cycle is two sysfs writes and a settle; anything longer than this
// means the controller is wedged, and boot should carry on regardless.
const reattachBudget = 5 * time.Second

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
	// Wire the HID quiescer here rather than leaving it to whichever handler
	// happens to reach hid.Manager() first. Until it is wired the manager never
	// pushes the report-ID routes, so a collapsed HID layout - one interface
	// carrying keyboard, mouse and pointer behind report IDs 1, 2 and 3 - sends
	// every report in the old prefix-free framing instead. The host discards
	// those as report 0, which is a keyboard that types nothing while the
	// gadget, the descriptor and the writes all look correct. It only worked
	// before when the UI happened to poll the virtual-device endpoint first.
	hid.Manager()
	startup.Fail("usb presentation", presentationManager.Err())
	startup.Run("media gadget", observerRefreshBudget, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), observerRefreshBudget)
		defer cancel()
		if err := presentationManager.RefreshObserver(ctx); err != nil {
			log.Debugf("media gadget unavailable: %s", err)
			return err
		}
		return nil
	})

	// The bind that put this gadget on the bus necessarily happened while the
	// board was still coming up, and a host that asked for a descriptor then
	// may have given up on the interfaces it had not started yet. Everything
	// above is done, so drop the pull-up and raise it: same gadget, same
	// descriptors, one clean enumeration against a device that can answer.
	startup.Run("usb reattach", reattachBudget, func() error {
		return presentationManager.Reattach()
	})

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
