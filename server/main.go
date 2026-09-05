package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/config"
	"NanoKVM-Server/logger"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/router"
	"NanoKVM-Server/service/bootslot"
	"NanoKVM-Server/service/bridge"
	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/service/passthrough"
	"NanoKVM-Server/service/picoclaw"
	"NanoKVM-Server/service/presentation"
	"NanoKVM-Server/service/startup"
	"NanoKVM-Server/service/vm"
	"NanoKVM-Server/service/vm/jiggler"
	"NanoKVM-Server/service/vnc"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
)

const (
	confirmTimeout  = 150 * time.Second
	confirmInterval = 2 * time.Second

	// What each step of the way out may take before the process leaves without
	// it. The media pipeline's own waits are two seconds for a worker to leave
	// its loop and five for a close(2) on the video node to return
	// (media/CLAUDE.md), so its budget is those plus a margin; the vision
	// library's deinit closes the capture pipeline in C with no bound of its
	// own, so it gets a few seconds and no more. The sum stays under the init
	// script's twenty-second wait on SIGINT, so TERM and KILL never have to.
	shutdownMediaBudget  = 8 * time.Second
	shutdownUSBBudget    = 2 * time.Second
	shutdownVisionBudget = 3 * time.Second
)

func main() {
	initialize()
	defer shutdown()

	run()
}

// Nothing here may keep run() from binding: on a device with no console the web
// UI is the only way back in, so every step below is bounded and isolated and a
// failure only degrades that one subsystem. startup.Report carries the result.
func initialize() {
	logger.Init()

	startup.Run("picoclaw token", 5*time.Second, config.EnsurePicoclawInternalToken)
	startup.Run("usb passthrough", 15*time.Second, func() error { return passthrough.GetManager().Recover() })
	startup.Run("screen", 5*time.Second, func() error {
		_ = common.GetScreen()
		return nil
	})
	startup.Run("hdmi", 15*time.Second, func() error {
		vm.DisableHdmiCapture()
		time.Sleep(10 * time.Millisecond)
		if !utils.IsHdmiDisabled() {
			vm.EnableHdmiCapture()
		}
		vm.SetHdmiViewerCount(0)
		return nil
	})
	startup.Run("mouse jiggler", 5*time.Second, func() error {
		jiggler.GetJiggler().Run()
		return nil
	})

	// The first signal starts the way out and the second ends it. Notify takes
	// the default action away from every signal it names, so a TERM that lands
	// while the first signal's shutdown is still running used to sit unread in
	// the channel and the process outlived it until KILL; now it exits at once.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigChan
		log.Printf("received %v: shutting down", sig)
		go func() {
			sig := <-sigChan
			log.Printf("received %v during shutdown: exiting now", sig)
			os.Exit(1)
		}()

		shutdown()
		os.Exit(0)
	}()
}

func run() {
	conf := config.GetInstance()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// Ahead of the routes, so the bridge's third verification gate sees every
	// request a client makes over the uplink and not only the bridge ones.
	r.Use(bridge.RecordListener(bridge.Witness()))
	if conf.Authentication == "disable" {
		r.Use(cors.AllowAll())
	}

	router.Init(r)
	startVNC(conf)

	httpAddr := utils.ListenAddr(conf.Host, strconv.Itoa(conf.Port.Http))
	loopbackHTTPAddr := utils.ListenAddr("127.0.0.1", strconv.Itoa(conf.Port.Http))
	needsLoopbackHTTP := utils.NeedsDedicatedLoopbackListener(conf.Host)

	if conf.Proto == "https" {
		httpsPortStr := strconv.Itoa(conf.Port.Https)

		go func() {
			err := r.RunTLS(utils.ListenAddr(conf.Host, httpsPortStr), conf.Cert.Crt, conf.Cert.Key)
			if err != nil {
				panic("start https server failed")
			}
		}()

		if needsLoopbackHTTP {
			go func() {
				if err := middleware.ListenAndServeLoopbackHTTPRedirect(
					loopbackHTTPAddr,
					httpsPortStr,
					r,
					router.LoopbackHTTPAllowedPaths()...,
				); err != nil {
					panic("start loopback http server failed")
				}
			}()
		}

		go confirmKernel("https", conf.Port.Https)
		if err := middleware.ListenAndServeLoopbackHTTPRedirect(
			httpAddr,
			httpsPortStr,
			r,
			router.LoopbackHTTPAllowedPaths()...,
		); err != nil {
			panic("start http server failed")
		}
	} else {
		if needsLoopbackHTTP {
			go func() {
				if err := r.Run(loopbackHTTPAddr); err != nil {
					panic("start loopback http server failed")
				}
			}()
		}

		listener, err := net.Listen("tcp", httpAddr)
		if err != nil {
			panic("start http server failed")
		}
		go confirmKernel("http", conf.Port.Http)
		if err := r.RunListener(listener); err != nil {
			panic("start http server failed")
		}
	}
}

func startVNC(conf *config.Config) {
	vnc.GetManager().Configure(func() *vnc.Server {
		return &vnc.Server{
			Addr:      utils.ListenAddr(conf.Host, strconv.Itoa(conf.VNC.Port)),
			AllowNone: conf.Authentication == "disable",
			Screen: func() (uint16, uint16, uint16, int) {
				screen := common.GetScreen()
				common.CheckScreen()
				return screen.Width, screen.Height, screen.Quality, screen.FPS
			},
			ReadJPEG: common.GetKvmVision().ReadMjpeg,
			AllowInput: func(mode controlmode.Mode) bool {
				return mode != controlmode.ModePicoclaw || !picoclaw.GetSessionLock().BlocksManualInput()
			},
			ViewerCount: func(count int, version uint64) {
				vm.UpdateHdmiViewerSnapshot("vnc", count, version)
			},
		}
	})

	if !conf.VNC.Enabled {
		return
	}

	if err := vnc.GetManager().Start(); err != nil {
		log.Printf("start vnc server failed: %v", err)
	}
}

// shutdown gives back what the host can see before the process dies, in order
// and on a budget per step. The camera goes first: with a stream open the host
// is polling the video node, and closing it here ends the stream the way a
// STREAMOFF does instead of leaving the node to fall with the process. Then
// the gadget leaves the bus, so the host sees a disconnect now rather than a
// device that answers nothing until the next server has rebound it. The
// vision library's deinit is last because it is the step with no bound of its
// own; if it does not return, the result says so and the process exits anyway,
// which is what the init script's SIGKILL used to do twenty seconds later.
func shutdown() {
	results := startup.Stop(
		startup.Step{Name: "media pipeline", Budget: shutdownMediaBudget, Run: func() error {
			if manager := presentation.Current(); manager != nil {
				return manager.SuspendMedia()
			}
			return nil
		}},
		startup.Step{Name: "usb passthrough", Budget: shutdownUSBBudget, Run: passthrough.GetManager().Close},
		// After passthrough, whose session may hold the controller on loan and
		// gives it back when it stops; before vision, so the host sees the
		// gadget go while the process is still certainly able to say so.
		startup.Step{Name: "usb gadget", Budget: shutdownUSBBudget, Run: func() error {
			if manager := presentation.Current(); manager != nil {
				return manager.Detach()
			}
			return nil
		}},
		startup.Step{Name: "kvm vision", Budget: shutdownVisionBudget, Run: func() error {
			common.GetKvmVision().Close()
			return nil
		}},
	)
	for _, result := range results {
		log.Printf("shutdown: %s", result)
	}
}

// confirmKernel commits a trial kernel once the device is genuinely usable.
// Reaching userspace is not enough: a kernel that boots with no working NIC
// must still roll back, so this waits for the UI to answer on the loopback and
// for a routable address to exist.
func confirmKernel(scheme string, port int) {
	err := bootslot.Default().ConfirmWhenReady(bootslot.Ready{
		Serving:  func() bool { return serving(scheme, port) },
		Routable: bootslot.Routable,
		Timeout:  confirmTimeout,
		Interval: confirmInterval,
	})
	if err != nil {
		log.Printf("commit trial kernel: %v", err)
	}
}

// The scheme has to follow the listener. Probing http:// against a TLS port
// gets a handshake error rather than a page, so with HTTPS enabled this never
// reported the UI as serving, the trial never committed, and every kernel
// update rolled itself back after the guard's deadline - on a device that was
// running the new kernel perfectly well. The certificate is the device's own
// and is usually self-signed, so this one probe does not verify it: it is
// asking "is my own UI up", over the loopback, not trusting a peer.
func serving(scheme string, port int) bool {
	client := http.Client{Timeout: 5 * time.Second}
	if scheme == "https" {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	rsp, err := client.Get(scheme + "://127.0.0.1:" + strconv.Itoa(port) + "/")
	if err != nil {
		return false
	}
	defer rsp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(rsp.Body, 64<<10))
	return err == nil && bytes.Contains(body, []byte("<title>NanoKVM</title>"))
}
