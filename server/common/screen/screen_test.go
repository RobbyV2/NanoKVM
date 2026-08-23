package screen

import "testing"

// Every entry must be a size a client can be told to allocate. A zero-sized
// entry would make "is this a real resolution?" answer yes for the native
// sentinel, which is exactly what let a 0x0 framebuffer reach RFB clients.
func TestResolutionMapOnlyHoldsRealFramebufferSizes(t *testing.T) {
	if width, ok := ResolutionMap[ResolutionNative]; ok {
		t.Fatalf("ResolutionMap accepts the native sentinel as a resolution: %d -> %d", ResolutionNative, width)
	}

	for height, width := range ResolutionMap {
		if width == 0 || height == 0 {
			t.Fatalf("ResolutionMap holds a zero-sized entry: %d -> %d", height, width)
		}
	}
}

func TestCheckScreenKeepsNativeAndRejectsUnknownResolutions(t *testing.T) {
	screen := Get()
	width, height := screen.Width, screen.Height
	t.Cleanup(func() { screen.Width, screen.Height = width, height })

	Set("resolution", int(ResolutionNative))
	Check()
	if screen.Width != ResolutionNative || screen.Height != ResolutionNative {
		t.Fatalf("native resolution became %dx%d, want the source's own size", screen.Width, screen.Height)
	}

	screen.Width, screen.Height = 2560, 1440
	Check()
	if screen.Width != 1920 || screen.Height != 1080 {
		t.Fatalf("unsupported resolution became %dx%d, want 1920x1080", screen.Width, screen.Height)
	}

	Set("resolution", 720)
	Check()
	if screen.Width != 1280 || screen.Height != 720 {
		t.Fatalf("supported resolution became %dx%d, want 1280x720", screen.Width, screen.Height)
	}
}
