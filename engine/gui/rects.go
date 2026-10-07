package gui

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

// RectBatch draws solid colored screen-space rectangles in a single draw call.
// It works with the gui shader (position vec2, texCoord vec2) by sampling a small
// palette texture: every color in the palette is one texel.
type RectBatch struct {
	shader   *glhf.Shader
	palette  *glhf.Texture
	colors   int
	vertices *glhf.VertexSlice[glhf.GlFloat]
	capacity int
	data     []glhf.GlFloat
}

const maxRectsPerBatch = 2048

// NewRectBatch creates a batch that can draw rectangles in any of the given opaque colors.
// Colors are referenced by their index in the palette.
func NewRectBatch(shader *glhf.Shader, palette []mgl32.Vec3) *RectBatch {
	withAlpha := make([]mgl32.Vec4, len(palette))
	for i, color := range palette {
		withAlpha[i] = color.Vec4(1)
	}
	return NewRectBatchRGBA(shader, withAlpha)
}

// NewRectBatchRGBA is like NewRectBatch, but palette colors may be translucent.
func NewRectBatchRGBA(shader *glhf.Shader, palette []mgl32.Vec4) *RectBatch {
	pixels := make([]uint8, len(palette)*4)
	for i, color := range palette {
		pixels[i*4] = uint8(color.X() * 255)
		pixels[i*4+1] = uint8(color.Y() * 255)
		pixels[i*4+2] = uint8(color.Z() * 255)
		pixels[i*4+3] = uint8(color.W() * 255)
	}
	capacity := maxRectsPerBatch * 6
	return &RectBatch{
		shader:   shader,
		palette:  glhf.NewTexture(len(palette), 1, false, pixels),
		colors:   len(palette),
		vertices: glhf.MakeVertexSlice(shader, capacity, capacity),
		capacity: capacity,
		data:     make([]glhf.GlFloat, 0, capacity*4),
	}
}

// Clear removes all rectangles queued since the last draw.
func (r *RectBatch) Clear() {
	r.data = r.data[:0]
}

// Add queues a rectangle with its top-left corner at (x, y), in screen pixels.
func (r *RectBatch) Add(x, y, width, height float32, colorIndex int) {
	if len(r.data)/4+6 > r.capacity || width <= 0 || height <= 0 {
		return
	}
	u := glhf.GlFloat((float32(colorIndex) + 0.5) / float32(r.colors))
	v := glhf.GlFloat(0.5)
	left, top := glhf.GlFloat(x), glhf.GlFloat(y)
	right, bottom := glhf.GlFloat(x+width), glhf.GlFloat(y+height)
	r.data = append(r.data,
		left, top, u, v,
		left, bottom, u, v,
		right, bottom, u, v,
		left, top, u, v,
		right, bottom, u, v,
		right, top, u, v,
	)
}

// AddBar queues a background rectangle and a fill proportional to fraction (0..1).
func (r *RectBatch) AddBar(x, y, width, height float32, fraction float32, backgroundIndex, fillIndex int) {
	fraction = mgl32.Clamp(fraction, 0, 1)
	r.Add(x, y, width, height, backgroundIndex)
	r.Add(x+1, y+1, (width-2)*fraction, height-2, fillIndex)
}

// AddFrame queues the outline of a rectangle.
func (r *RectBatch) AddFrame(x, y, width, height, thickness float32, colorIndex int) {
	r.Add(x, y, width, thickness, colorIndex)
	r.Add(x, y+height-thickness, width, thickness, colorIndex)
	r.Add(x, y, thickness, height, colorIndex)
	r.Add(x+width-thickness, y, thickness, height, colorIndex)
}

// Draw renders the queued rectangles. The gui shader must be active.
func (r *RectBatch) Draw() {
	vertexCount := len(r.data) / 4
	if vertexCount == 0 {
		return
	}
	r.shader.SetUniformAttr(1, mgl32.Ident4())
	r.shader.SetUniformAttr(2, mgl32.Vec4{0, 0, 0, 0}) // no tint
	r.shader.SetUniformAttr(3, mgl32.Vec4{0, 0, 0, 0}) // no discard
	r.palette.Begin()
	r.vertices.Begin()
	r.vertices.Slice(0, vertexCount).SetVertexData(r.data)
	r.vertices.Slice(0, vertexCount).Draw()
	r.vertices.End()
	r.palette.End()
}
