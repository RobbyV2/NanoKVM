package main

import (
	"log"
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
	"NanoKVM-Server/service/bridge"
	"NanoKVM-Server/service/controlmode"
	"NanoKVM-Server/service/passthrough"
	"NanoKVM-Server/service/picoclaw"
	"NanoKVM-Server/service/vm"
	"NanoKVM-Server/service/vm/jiggler"
	"NanoKVM-Server/service/vnc"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
)

func main() {
	initialize()
	defer dispose()

	run()
}

func initialize() {
	if err := config.EnsurePicoclawInternalToken(); err != nil {
		log.Fatalf("failed to initialize picoclaw internal token: %v", err)
	}

	logger.Init()
	if err := passthrough.GetManager().Recover(); err != nil {
		log.Printf("recover usb passthrough: %v", err)
	}

	// init screen parameters
	_ = common.GetScreen()

	// init HDMI
	vm.DisableHdmiCapture()
	time.Sleep(10 * time.Millisecond)
	if !utils.IsHdmiDisabled() {
		vm.EnableHdmiCapture()
	}
	vm.SetHdmiViewerCount(0)

	// run mouse jiggler
	jiggler.GetJiggler().Run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigChan
		log.Printf("\nReceived signal: %v\n", sig)

		dispose()
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

		if err := r.Run(httpAddr); err != nil {
			panic("start http server failed")
		}
	}
}

func startVNC(conf *config.Config) {
	if !conf.VNC.Enabled {
		return
	}

	server := &vnc.Server{
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

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("vnc server stopped: %v", err)
		}
	}()
}

func dispose() {
	if err := passthrough.GetManager().Close(); err != nil {
		log.Printf("stop usb passthrough: %v", err)
	}
	common.GetKvmVision().Close()
}
