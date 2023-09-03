package glhf

import (
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Texture is an OpenGL texture.
type Texture struct {
	tex           binder
	width, height int
	smooth        bool
    itemSizeX     int
    itemSizeY     int
    itemPaddingX  int
    itemPaddingY  int
    itemMarginX   int
    itemMarginY   int
}

func NewSolidColorTexture(color [3]uint8) *Texture {
	// we want to create a 4x4 texture with a solid color
	// we'll use the color passed in

	pixels := make([]uint8, 4*4*4)
	for i := 0; i < 4*4; i++ {
		pixels[i*4] = color[0]
		pixels[i*4+1] = color[1]
		pixels[i*4+2] = color[2]
		pixels[i*4+3] = 255
	}

	var texture *Texture
	texture = NewTexture(4, 4, false, pixels)
	return texture
}

// NewTexture creates a new texture with the specified width and height with some initial
// pixel values. The pixels must be a sequence of RGBA values (one byte per component).
func NewTexture(width, height int, smooth bool, pixels []uint8) *Texture {
	tex := &Texture{
		tex: binder{
			restoreLoc: gl.TEXTURE_BINDING_2D,
			bindFunc: func(obj uint32) {
				gl.BindTexture(gl.TEXTURE_2D, obj)
			},
		},
		width:  width,
		height: height,
	}

	gl.GenTextures(1, &tex.tex.obj)

	tex.Begin()
	defer tex.End()

	// initial data
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(width),
		int32(height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(pixels),
	)
	/*
		borderColor := mgl32.Vec4{0, 0, 0, 0}
		gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	*/
	tex.SetSmooth(smooth)
	tex.SetWrapToRepeat()
	runtime.SetFinalizer(tex, (*Texture).delete)

	return tex
}

func (t *Texture) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteTextures(1, &t.tex.obj)
	})
}

// ID returns the OpenGL ID of this Texture.
func (t *Texture) ID() uint32 {
	return t.tex.obj
}

// Width returns the width of the Texture in pixels.
func (t *Texture) Width() int {
	return t.width
}

// Height returns the height of the Texture in pixels.
func (t *Texture) Height() int {
	return t.height
}

// SetPixels sets the content of a sub-region of the Texture. Pixels must be an RGBA byte sequence.
func (t *Texture) SetPixels(x, y, w, h int, pixels []uint8) {
	if len(pixels) != w*h*4 {
		panic("set pixels: wrong number of pixels")
	}
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		int32(x),
		int32(y),
		int32(w),
		int32(h),
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(pixels),
	)
}

// Pixels returns the content of a sub-region of the Texture as an RGBA byte sequence.
func (t *Texture) Pixels(x, y, w, h int) []uint8 {
	pixels := make([]uint8, t.width*t.height*4)
	gl.GetTexImage(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(pixels),
	)
	subPixels := make([]uint8, w*h*4)
	for i := 0; i < h; i++ {
		row := pixels[(i+y)*t.width*4+x*4 : (i+y)*t.width*4+(x+w)*4]
		subRow := subPixels[i*w*4 : (i+1)*w*4]
		copy(subRow, row)
	}
	return subPixels
}

// SetSmooth sets whether the Texture should be drawn "smoothly" or "pixely".
//
// It affects how the Texture is drawn when zoomed. Smooth interpolates between the neighbour
// pixels, while pixely always chooses the nearest pixel.
func (t *Texture) SetSmooth(smooth bool) {
	t.smooth = smooth
	if smooth {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	} else {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	}
}

func (t *Texture) SetWrapToRepeat() {
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
}

// Smooth returns whether the Texture is set to be drawn "smooth" or "pixely".
func (t *Texture) Smooth() bool {
	return t.smooth
}

// Begin binds the Texture. This is necessary before using the Texture.
func (t *Texture) Begin() {
	t.tex.bind()
}

// End unbinds the Texture and restores the previous one.
func (t *Texture) End() {
	t.tex.restore()
}

func (t *Texture) GetUV(index uint16) (GlFloat, GlFloat, GlFloat, GlFloat) {
    width := t.itemSizeX
    height := t.itemSizeY

	xCount := int(t.width / width)

    leftBorder := t.itemMarginX + ((int(index) % xCount) * (width + t.itemPaddingX))
    topBorder := t.itemMarginY + ((int(index) / xCount) * (height + t.itemPaddingY))

    topLeftU := GlFloat(leftBorder) / GlFloat(t.width)
    topLeftV := GlFloat(topBorder) / GlFloat(t.height)

    bottomRightU := GlFloat(leftBorder+width) / GlFloat(t.width)
    bottomRightV := GlFloat(topBorder+height) / GlFloat(t.height)

	return topLeftU, topLeftV, bottomRightU, bottomRightV
}

func (t *Texture) SetAtlasItemSize(width int, height int) {
    t.itemSizeX = width
    t.itemSizeY = height
}

func (t *Texture) SetPaddingBetweenItems(x, y int) {
    t.itemPaddingX = x
    t.itemPaddingY = y
}

func (t *Texture) GetAtlasItemSize() (int, int) {
    return t.itemSizeX, t.itemSizeY
}

func (t *Texture) SetAtlasMargin(x int, y int) {
    t.itemMarginX = x
    t.itemMarginY = y
}