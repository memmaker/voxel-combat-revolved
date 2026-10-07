//go:build !js

package util

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Development aid: when BT_SHOT_DIR is set, the current frame is saved as a PNG
// every BT_SHOT_EVERY seconds (default 5). Only the game's own framebuffer is captured.
type AutoScreenshot struct {
	dir       string
	every     float64
	sinceLast float64
	counter   int
}

func NewAutoScreenshot() *AutoScreenshot {
	dir := os.Getenv("BT_SHOT_DIR")
	if dir == "" {
		return nil
	}
	every := 5.0
	if v, err := strconv.ParseFloat(os.Getenv("BT_SHOT_EVERY"), 64); err == nil && v > 0 {
		every = v
	}
	_ = os.MkdirAll(dir, 0o755)
	return &AutoScreenshot{dir: dir, every: every}
}

func (s *AutoScreenshot) Update(elapsed float64, fbWidth, fbHeight int) {
	if s == nil {
		return
	}
	s.sinceLast += elapsed
	if s.sinceLast < s.every {
		return
	}
	s.sinceLast = 0
	s.counter++
	SaveFramebufferPNG(filepath.Join(s.dir, fmt.Sprintf("frame_%03d.png", s.counter)), fbWidth, fbHeight)
}

func SaveFramebufferPNG(filename string, width, height int) {
	pixels := make([]uint8, width*height*4)
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	rowSize := width * 4
	for y := 0; y < height; y++ { // OpenGL rows start at the bottom
		copy(img.Pix[y*rowSize:(y+1)*rowSize], pixels[(height-1-y)*rowSize:(height-y)*rowSize])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	file, err := os.Create(filename)
	if err != nil {
		println("[Screenshot] " + err.Error())
		return
	}
	defer file.Close()
	_ = png.Encode(file, img)
}
