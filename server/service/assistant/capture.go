package assistant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
)

var ErrEmptyCrop = errors.New("selection is empty")

type Frame struct {
	JPEG   []byte
	Width  int
	Height int
}

// Capturer is the device's screen capture; the router adapts the MCP backend
// so this package stays free of cgo.
type Capturer interface {
	Capture(ctx context.Context) (Frame, error)
}

// Crop is a selection as fractions of the video frame.
type Crop struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

func cropBounds(c Crop, w, h int) (image.Rectangle, error) {
	for _, v := range []float64{c.X, c.Y, c.W, c.H} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return image.Rectangle{}, ErrEmptyCrop
		}
	}
	if c.W <= 0 || c.H <= 0 {
		return image.Rectangle{}, ErrEmptyCrop
	}
	x0 := math.Floor(c.X * float64(w))
	y0 := math.Floor(c.Y * float64(h))
	x1 := math.Ceil((c.X + c.W) * float64(w))
	y1 := math.Ceil((c.Y + c.H) * float64(h))
	clamp := func(v float64, max int) int { return int(math.Max(0, math.Min(v, float64(max)))) }
	r := image.Rect(clamp(x0, w), clamp(y0, h), clamp(x1, w), clamp(y1, h))
	if r.Empty() {
		return image.Rectangle{}, ErrEmptyCrop
	}
	return r, nil
}

func cropJPEG(data []byte, c Crop) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode capture: %w", err)
	}
	b := img.Bounds()
	r, err := cropBounds(c, b.Dx(), b.Dy())
	if err != nil {
		return nil, err
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, errors.New("capture cannot be cropped")
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, sub.SubImage(r.Add(b.Min)), &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("encode crop: %w", err)
	}
	return out.Bytes(), nil
}
