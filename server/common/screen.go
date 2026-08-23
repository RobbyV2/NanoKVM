package common

// The screen state is pure Go, but this package links libkvm.so for riscv64, so
// anything that lives here is untestable off-device. It lives in common/screen
// and is re-exported here so call sites keep the names they had.
import "NanoKVM-Server/common/screen"

type Screen = screen.Screen

const ResolutionNative = screen.ResolutionNative

var (
	ResolutionMap = screen.ResolutionMap
	QualityMap    = screen.QualityMap
	BitRateMap    = screen.BitRateMap
)

func GetScreen() *Screen {
	return screen.Get()
}

func SetScreen(key string, value int) {
	screen.Set(key, value)
}

func CheckScreen() {
	screen.Check()
}
