package etxt

import (
    "image"
)
import "golang.org/x/image/math/fixed"

type TargetImage = *TextMesh

type GlyphMask = *image.Rectangle

var Atlas *TextAtlas = NewTextAtlas(32)

// our Atlas is just the raw texture data needed
// we want one glyph next to the other and our height is fixed by the font size
// so just one very long row of glyphs
// the Atlas height would be fixed and the width would be variable
// we pre-calculate the height and expand the width as needed
// width and height need to be multiples of 4

// this would create the cached bitmap version of the glyph
// we want to write this bitmap to our Atlas here
// the returned glyphmask should encode the glyph's bounds and position in the Atlas
func convertAlphaImageToGlyphMask(i *image.Alpha) GlyphMask {
    if i == nil {
        return nil
    }
    return Atlas.AddGlyph(i)
}

// The default glyph drawing function used in renderers. Do not confuse with
// the main [Renderer.Draw]() function. DefaultDrawFunc is a low level function,
// rarely necessary except when paired with [Renderer.Traverse]*() operations.
func (self *Renderer) DefaultDrawFunc(dot fixed.Point26_6, mask GlyphMask, _ GlyphIndex) {
    if mask == nil {
        return
    } // spaces and empty glyphs will be nil

    drawPosX := fixed26_6ToFloat64(dot.X)
    drawPosY := fixed26_6ToFloat64(dot.Y)

    self.target.AddGlyphFromAtlas(drawPosX, drawPosY, image.Rect(mask.Min.X, mask.Min.Y, mask.Max.X, mask.Max.Y))
}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
    return float64(x>>6) + float64(x&((1<<6)-1))/float64(1<<6)
}
