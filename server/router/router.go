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

// One bind, the HID reopen behind it (two seconds of retry at most), the media
// pipeline's node holds (five seconds each to settle) and a wait of three for
// a host that was configured before the start to come back. Anything longer is
// a controller that is wedged, and the listener comes up regardless.
const attachBudget = 15 * time.Second

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

	// The one bind of the start, and the last thing the start does to the
	// gadget. The profile was reconciled when the manager was built, with its
	// bind held back; the HID writers and the media observer are wired above;
	// so the gadget that goes on the bus here is finished, and the host that
	// asks for a descriptor gets an answer. Attach also holds the camera's
	// video node and builds the media workers, since the node exists only
	// once the function is bound. It used to be a bind at the reconcile and
	// a pull-up cycle here, which the host saw as the device coming and going
	// within a second at every start.
	startup.Run("usb attach", attachBudget, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), attachBudget)
		defer cancel()
		if err := presentationManager.Attach(ctx); err != nil {
			log.Debugf("usb attach: %s", err)
			return err
		}
		return nil
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
