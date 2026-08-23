package screen

import "sync"

type Screen struct {
	Width   uint16
	Height  uint16
	FPS     int
	Quality uint16
	BitRate uint16
	GOP     uint8
}

var (
	screen     *Screen
	screenOnce sync.Once
)

// ResolutionNative asks the encoder for whatever the HDMI source is currently
// delivering. It is not a framebuffer size, so it is not in ResolutionMap.
const ResolutionNative uint16 = 0

// ResolutionMap height to width
var ResolutionMap = map[uint16]uint16{
	1080: 1920,
	720:  1280,
	600:  800,
	480:  640,
}

var QualityMap = map[uint16]bool{
	100: true,
	80:  true,
	60:  true,
	50:  true,
}

var BitRateMap = map[uint16]bool{
	5000: true,
	3000: true,
	2000: true,
	1000: true,
}

func Get() *Screen {
	screenOnce.Do(func() {
		screen = &Screen{
			Width:   ResolutionNative,
			Height:  ResolutionNative,
			Quality: 80,
			FPS:     30,
			BitRate: 3000,
			GOP:     30,
		}
	})

	return screen
}

func Set(key string, value int) {
	switch key {
	case "resolution":
		height := uint16(value)
		if height == ResolutionNative {
			screen.Width = ResolutionNative
			screen.Height = ResolutionNative
			return
		}
		if width, ok := ResolutionMap[height]; ok {
			screen.Width = width
			screen.Height = height
		}

	case "quality":
		if value > 100 {
			screen.BitRate = uint16(value)
		} else {
			screen.Quality = uint16(value)
		}

	case "fps":
		screen.FPS = validateFPS(value)

	case "gop":
		screen.GOP = uint8(value)
	}
}

func Check() {
	if screen.Height != ResolutionNative {
		if _, ok := ResolutionMap[screen.Height]; !ok {
			screen.Width = 1920
			screen.Height = 1080
		}
	}

	if _, ok := QualityMap[screen.Quality]; !ok {
		screen.Quality = 80
	}

	if _, ok := BitRateMap[screen.BitRate]; !ok {
		screen.BitRate = 3000
	}
}

func validateFPS(fps int) int {
	if fps > 60 {
		return 60
	}
	if fps < 10 {
		return 10
	}

	return fps
}
