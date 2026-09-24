package assistant

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"testing"
)

func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCropBounds(t *testing.T) {
	r, err := cropBounds(Crop{X: 0.25, Y: 0.5, W: 0.5, H: 0.25}, 1920, 1080)
	if err != nil || r != image.Rect(480, 540, 1440, 810) {
		t.Fatalf("r=%v err=%v", r, err)
	}
	// Outside the frame is clamped.
	r, err = cropBounds(Crop{X: -0.1, Y: 0.9, W: 0.3, H: 0.5}, 100, 100)
	if err != nil || r != image.Rect(0, 90, 20, 100) {
		t.Fatalf("clamp r=%v err=%v", r, err)
	}
}

func TestCropBoundsEmpty(t *testing.T) {
	for _, c := range []Crop{{X: 0.5, Y: 0.5}, {X: 1.2, Y: 0, W: 0.5, H: 1}, {X: math.NaN(), W: 1, H: 1},
		{X: 1.2, W: -0.5, H: 1}, {X: 0.5, Y: 0, W: -0.2, H: 1}, {X: math.Inf(1), W: 1, H: 1}} {
		if _, err := cropBounds(c, 100, 100); !errors.Is(err, ErrEmptyCrop) {
			t.Fatalf("crop %+v: err=%v", c, err)
		}
	}
}

func TestCropJPEG(t *testing.T) {
	out, err := cropJPEG(testJPEG(t, 200, 100), Crop{X: 0.5, Y: 0, W: 0.5, H: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 100 || cfg.Height != 50 {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}
