package etxt

import (
    "github.com/memmaker/battleground/engine/glhf"
    "image"
    "image/color"
    "image/png"
    "os"
)

type TextAtlas struct {
    pixels         *image.NRGBA // RGBA - 1 byte per channel, 4 channels per pixel -> 4 bytes per pixel
    currentYOffset uint64
    isDirty        bool
}

type AtlasPosition struct {
    yOffset uint64
    bounds  image.Rectangle
}

func NewTextAtlas(fontSize int) *TextAtlas {
    if fontSize%4 != 0 {
        fontSize += 4 - (fontSize % 4)
    }
    return &TextAtlas{
        pixels: image.NewNRGBA(image.Rect(0, 0, fontSize, fontSize)),
    }
}

func (a *TextAtlas) AddGlyph(glyph *image.Alpha) *image.Rectangle {
    glyphWidth := glyph.Rect.Dx()
    glyphHeight := glyph.Rect.Dy()
    atlasWidth := a.pixels.Bounds().Dx()
    atlasHeight := a.pixels.Bounds().Dy()
    oldData := a.pixels.Pix
    if glyphWidth > atlasWidth {
        println("WARNING: Glyph width is greater than Atlas width")
    }
    newHeight := a.currentYOffset + uint64(glyphHeight)
    if newHeight > uint64(atlasHeight) {
        if newHeight%4 != 0 {
            newHeight += 4 - (newHeight % 4)
        }
        a.pixels = image.NewNRGBA(image.Rect(0, 0, atlasWidth, int(newHeight)))
        copy(a.pixels.Pix, oldData)
        println("WARNING: Atlas height exceeded, resizing")
    }

    for y := 0; y < glyphHeight; y++ {
        for x := 0; x < atlasWidth; x++ {
            //value := uint8(255)
            pixColor := color.NRGBA{R: 255, G: 255, B: 255, A: 0}
            if x < glyphWidth {
                originalAlpha := glyph.Pix[y*glyph.Stride+x]
                invertedAlpha := 255 - originalAlpha
                pixColor = color.NRGBA{R: invertedAlpha, G: invertedAlpha, B: invertedAlpha, A: originalAlpha}
            }

            a.pixels.Set(x, int(a.currentYOffset)+y, pixColor)
        }
    }
    result := image.Rect(0, int(a.currentYOffset), glyphWidth, int(a.currentYOffset)+glyphHeight)
    a.currentYOffset = newHeight
    a.isDirty = true
    return &result
}

func imageToPng(a image.Image, filename string) {
    f, err := os.Create(filename)
    if err != nil {
        panic(err)
    }
    defer f.Close()
    err = png.Encode(f, a)
    if err != nil {
        panic(err)
    }
}

func (a *TextAtlas) Bounds() image.Rectangle {
    return a.pixels.Bounds()
}

func (a *TextAtlas) ToTexture() *glhf.Texture {
    return glhf.NewTexture(a.pixels.Bounds().Dx(), a.pixels.Bounds().Dy(), false, a.pixels.Pix)
}
