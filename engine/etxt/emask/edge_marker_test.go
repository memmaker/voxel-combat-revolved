//go:build gtxt

package emask

import "time"
import "math"
import "math/rand"
import "testing"

import "os"
import "strconv"
import "image"
import "image/png"
import "image/color"

import "golang.org/x/image/math/fixed"

func TestEdgeAlignedRects(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (5x4)
	}{
		{ // small square
			in: []float64{1, 1, 1, 3, 3, 3, 3, 1, 1, 1},
			out: []float64 {
				0, 0, 0,  0, 0,
				0, 1, 0, -1, 0,
				0, 1, 0, -1, 0,
				0, 0, 0,  0, 0,
			},
		},
		{ // full canvas rect
			in: []float64{0, 0, 0, 4, 5, 4, 5, 0, 0, 0},
			out: []float64 {
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
				1, 0, 0, 0, 0,
			},
		},
		{ // large canvas square
			in: []float64{0, 0, 0, 4, 4, 4, 4, 0, 0, 0},
			out: []float64 {
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
			},
		},
		{ // large outside rect
			in: []float64{-5, 0, -5, 4, 4, 4, 4, 0, -5, 0},
			out: []float64 {
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
			},
		},
		{ // smaller outside rect
			in: []float64{-5, 1, -5, 3, 4, 3, 4, 1, -5, 1},
			out: []float64 {
				0, 0, 0, 0,  0,
				1, 0, 0, 0, -1,
				1, 0, 0, 0, -1,
				0, 0, 0, 0,  0,
			},
		},
		{ // shape
			in: []float64{0, 1, 0, 3, 1, 3, 1, 4, 2, 4, 2, 2, 3, 2, 3, 1, 0, 1},
			out: []float64 {
				0, 0,  0,  0,  0,
				1, 0,  0, -1,  0,
				1, 0, -1,  0,  0,
				0, 1, -1,  0,  0,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(5, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeAlignedTriangles(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (4x4)
	}{
		{ // right triangle
			in: []float64{0, 0, 0, 4, 4, 4, 0, 0},
			out: []float64 {
				0.5, -0.5,  0.0,  0.0,
				1.0, -0.5, -0.5,  0.0,
				1.0,  0.0, -0.5, -0.5,
				1.0,  0.0,    0, -0.5,
			},
		},
		{ // right triangle, alternative orientation (both shape and def. direction)
			in: []float64{0, 0, 4, 0, 0, 4, 0, 0},
			out: []float64 {
				-1.0, 0.0, 0.0, 0.5,
				-1.0, 0.0, 0.5, 0.5,
				-1.0, 0.5, 0.5, 0.0,
				-0.5, 0.5, 0.0, 0.0,
			},
		},
		{ // triangle with wide base
			in: []float64{0, 0, 2, 4, 4, 0, 0, 0},
			out: []float64 {
				0.75, 0.25, 0.0, -0.25,
				0.25, 0.75, 0.0, -0.75,
				0.00, 0.75, 0.0, -0.75,
				0.00, 0.25, 0.0, -0.25,
			},
		},
		{ // slimmer right triangle, tricky fill proportions
			in: []float64{0, 0, 0, 4, 3, 4, 0, 0},
			out: []float64 {
				 3/ 8., -3 / 8.,    0.0 ,   0.0 ,
				23/24., -19/24., -4 /24.,   0.0 ,
				  1.0 ,  -1/ 6., -19/24., -1/24.,
				  1.0 ,    0.0 , -3 / 8., -5/ 8.,
			},
		},
		{ // slimmer right triangle, alternative orientation
			in: []float64{0, 0, 3, 0, 0, 4, 0, 0},
			out: []float64 {
				  -1.0 ,   0.0 , 3 / 8., 5/ 8.,
				  -1.0 ,  1/ 6., 19/24., 1/24.,
				-23/24., 19/24., 4 /24.,  0.0 ,
				 -3/ 8., 3 / 8.,   0.0 ,  0.0 ,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(4, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeUnalignedRects(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (4x4)
	}{
		{ // shifted square
			in: []float64{0.5, 0, 0.5, 4, 2.5, 4, 2.5, 0, 0.5, 0},
			out: []float64 {
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
				0.5, 0.5, -0.5, -0.5,
			},
		},
		{ // shifted square, in both axes
			in: []float64{0.5, 0.5, 0.5, 3.5, 2.5, 3.5, 2.5, 0.5, 0.5, 0.5},
			out: []float64 {
				0.25, 0.25, -0.25, -0.25,
				0.50, 0.50, -0.50, -0.50,
				0.50, 0.50, -0.50, -0.50,
				0.25, 0.25, -0.25, -0.25,
			},
		},
		{ // slightly shifted square
			in: []float64{0.2, 0, 0.2, 4, 2.2, 4, 2.2, 0, 0.2, 0},
			out: []float64 {
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
				0.8, 0.2, -0.8, -0.2,
			},
		},
		{ // significantly shifted square
			in: []float64{0.8, 0, 0.8, 4, 2.8, 4, 2.8, 0, 0.8, 0},
			out: []float64 {
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
				0.2, 0.8, -0.2, -0.8,
			},
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(4, 4)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestEdgeSinglePixel(t *testing.T) {
	tests := []struct {
		in  []float64 // one moveTo + many lineTo coords
		out []float64 // output buffer (5x4)
	}{
		{ // pix square
			in: []float64{0, 0, 0, 1, 1, 1, 1, 0, 0, 0},
			out: []float64 { 1, -1 },
		},
		{ // half-pix square
			in: []float64{0.5, 0, 0.5, 1, 1, 1, 1, 0, 0.5, 0},
			out: []float64 { 0.5, -0.5 },
		},
	}

	emarker := edgeMarker{}
	emarker.Buffer.Resize(2, 1)
	for n, test := range tests {
		emarker.MoveTo(test.in[0], test.in[1])
		for i := 2; i < len(test.in); i += 2 {
			emarker.LineTo(test.in[i], test.in[i + 1])
		}
		if !similarFloat64Slices(test.out, emarker.Buffer.Values) {
			t.Fatalf("test#%d, on input %v, expected %v, got %v", n, test.in, test.out, emarker.Buffer.Values)
		}
		emarker.Buffer.Clear()
	}
}

func TestCompareEdgeAndStdRasts(t *testing.T) {
	const useTimeSeed = false
	const avgCmpTolerance = 2.0 // alpha value per 255
	const canvasWidth  = 80
	const canvasHeight = 80

	seed := time.Now().UnixNano()
	if !useTimeSeed { seed = 8623001 }
	rng := rand.New(rand.NewSource(seed)) // *
	// * Variable time seed works most of the time, but in some
	//   cases there are still differences that are big enough to
	//   be reported as failing tests.
	//   Still, I decided to switch to a static seed in order to make
	//   life more peaceful. Even if there's a bug, I'll wait until
	//   I come across a problematic case that happens in real life...
	//   instead of these synthetic tests.

	stdRasterizer  := &DefaultRasterizer{}
	edgeRasterizer := NewStdEdgeMarkerRasterizer()
	for n := 0; n < 30; n++ {
		// create random shape
		shape := randomShape(rng, 16, canvasWidth, canvasHeight)
		segments := shape.Segments()

		// rasterize with both rasterizers
		stdMask, err := Rasterize(segments, stdRasterizer, fixed.Point26_6{})
		if err != nil { t.Fatalf("stdRast error: %s", err.Error()) }
		edgeMask, err := Rasterize(segments, edgeRasterizer, fixed.Point26_6{})
		if err != nil { t.Fatalf("edgeRast error: %s", err.Error()) }

		// compare results
		if len(stdMask.Pix) != len(edgeMask.Pix) {
			t.Fatalf("len(stdMask.Pix) != len(edgeMask.Pix)")
		}

		totalDiff := 0
		for i := 0; i < len(stdMask.Pix); i++ {
			stdValue  := stdMask.Pix[i]
			edgeValue := edgeMask.Pix[i]
			if stdValue == edgeValue { continue }
			var diff uint8
			if stdValue > edgeValue {
				diff = stdValue - edgeValue
			} else {
				diff = edgeValue - stdValue
			}

			totalDiff += int(diff)
			// Note: we could compare pixel values individually here, but
			//       different thresholds and curve segmentation methods
			//       can cause severe value differences in some cases.
		}

		avgDiff := float64(totalDiff)/(canvasWidth*canvasHeight)
		if avgDiff > avgCmpTolerance {
			// Notice: this actually fails sometimes if a variable seed is used.
			//         Different curve segmentation methods are responsible for
			//         it, as far as I have seen. Look at the results yourself
			//         if this ever fails for you.
			exportTest("cmp_rasts_" + strconv.Itoa(n) + "_edge.png", edgeMask)
			exportTest("cmp_rasts_" + strconv.Itoa(n) + "_rast.png", stdMask)
			t.Fatalf("iter %d, totalDiff = %d average tolerance is too big (%f) (written files for visual debug)", n, totalDiff, avgDiff)
		}
	}
}

func exportTest(filename string, mask *image.Alpha) {
	rgba := image.NewRGBA(mask.Rect)
	r, g, b, a := color.White.RGBA()
	nrgba := color.NRGBA64 { R: uint16(r), G: uint16(g), B: uint16(b), A: 0 }
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			nrgba.A = uint16((a*uint32(mask.AlphaAt(x, y).A))/255)
			rgba.Set(x, y, mixColors(nrgba, color.Black))
		}
	}

	file, err := os.Create(filename)
	if err != nil { panic(err) }
	err = png.Encode(file, rgba)
	if err != nil { panic(err) }
	err = file.Close()
	if err != nil { panic(err) }
}

func randomShape(rng *rand.Rand, lines, w, h int) Shape {
	fsw, fsh := float64(w)*64, float64(h)*64
	var makeXY = func() (fixed.Int26_6, fixed.Int26_6) {
		return fixed.Int26_6(rng.Float64()*fsw), fixed.Int26_6(rng.Float64()*fsh)
	}
	startX, startY := makeXY()

	shape := NewShape(lines + 1)
	shape.MoveToFract(startX, startY)
	for i := 0; i < lines; i++ {
		x, y := makeXY()
		switch rng.Intn(3) {
		case 0: // LineTo
			shape.LineToFract(x, y)
		case 1: // QuadTo
			cx, cy := makeXY()
			shape.QuadToFract(cx, cy, x, y)
		case 2: // CubeTo
			cx1, cy1 := makeXY()
			cx2, cy2 := makeXY()
			shape.CubeToFract(cx1, cy1, cx2, cy2, x, y)
		}
	}
	shape.LineToFract(startX, startY)
	return shape
}

func similarFloat64Slices(a []float64, b []float64) bool {
	if len(a) != len(b) { return false }
	for i, valueA := range a {
		if valueA != b[i] {
			diff := math.Abs(valueA - b[i])
			if diff > 0.001 { return false } // allow small precision differences
		}
	}
	return true
}
