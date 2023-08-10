package etxt

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"image"
	"image/color"
	"log"
)

type OpenGLTextRenderer struct {
	shader       *glhf.Shader
	fontFilePath string
	renderer     *Renderer
}

func NewOpenGLTextRenderer(textShader *glhf.Shader) *OpenGLTextRenderer {
	const TextSizePx = 32
	fontFilePath := "assets/fonts/Ac437_EagleSpCGA_Alt2-2y.ttf"
	font, _, err := ParseFontFrom(fontFilePath)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("[OpenGLTextRenderer] Font loaded: %s\n", fontName)
	renderer := NewStdRenderer()
	cache := NewDefaultCache(1024 * 1024 * 1024) // 1GB cache
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(TextSizePx)
	renderer.SetFont(font)
	renderer.SetAlign(Top, Left)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // black

	return &OpenGLTextRenderer{
		renderer: renderer,
		shader:   textShader,
	}
}

func (r *OpenGLTextRenderer) SetSizePx(size int) {
	r.renderer.SetSizePx(size)
}
func (r *OpenGLTextRenderer) SetColor(color color.RGBA) {
	r.renderer.SetColor(color)
}
func (r *OpenGLTextRenderer) SetAlign(vert VertAlign, horiz HorzAlign) {
	r.renderer.SetAlign(vert, horiz)
}
func (r *OpenGLTextRenderer) DrawText(text string) *TextMesh {
	textMesh := NewTextMesh(r.shader)
	r.renderer.SetTarget(textMesh) // we want to set something as target, that can receive the vertex and uv positions
	r.renderer.SetColor(color.RGBA{255, 255, 255, 255})
	r.renderer.Draw(text, 0, 0)
	textMesh.SetTexture(Atlas.ToTexture())
	textMesh.PrepareForRender()
	return textMesh
}

type TextMesh struct {
	vertices *glhf.VertexSlice[glhf.GlFloat]
	pos      mgl32.Vec3
	shader   *glhf.Shader
	texture  *glhf.Texture
	glyphs   []DrawInfo
}
type DrawInfo struct {
	drawPosX, drawPosY float64
	glyphBounds        image.Rectangle
}

func (t *TextMesh) SetPosition(pos mgl32.Vec3) {
	t.pos = pos
}
func (t *TextMesh) AddGlyphFromAtlas(drawPosX, drawPosY float64, glyphBounds image.Rectangle) {
	// drawPosY is bottom of glyph -> convert to top of glyph
	drawPosY -= float64(glyphBounds.Dy())
	t.glyphs = append(t.glyphs, DrawInfo{
		drawPosX:    drawPosX,
		drawPosY:    drawPosY,
		glyphBounds: glyphBounds,
	})
}

func (t *TextMesh) Draw() {
	t.shader.SetUniformAttr(1, t.GetTransformMatrix())

	t.texture.Begin()

	t.vertices.Begin()
	t.vertices.Draw()
	t.vertices.End()

	t.texture.End()
}

func NewTextMesh(shader *glhf.Shader) *TextMesh {
	return &TextMesh{
		shader:   shader,
		vertices: glhf.MakeVertexSlice(shader, 0, 0),
	}
}
func (t *TextMesh) PrepareForRender() {
	if t.texture == nil {
		println("TextMesh: texture is nil")
		return
	}
	tWidth := glhf.GlFloat(t.texture.Width())
	tHeight := glhf.GlFloat(t.texture.Height())
	var rawVertices []glhf.GlFloat
	for _, glyph := range t.glyphs {
		gWidth := glhf.GlFloat(glyph.glyphBounds.Dx())
		gHeight := glhf.GlFloat(glyph.glyphBounds.Dy())

		// calculate uv coordinates
		leftU := glhf.GlFloat(glyph.glyphBounds.Min.X) / tWidth
		rightU := glhf.GlFloat(glyph.glyphBounds.Max.X) / tWidth

		topV := glhf.GlFloat(glyph.glyphBounds.Min.Y) / tHeight
		bottomV := glhf.GlFloat(glyph.glyphBounds.Max.Y) / tHeight

		rawVertices = append(rawVertices, []glhf.GlFloat{
			// first triangle
			// Top-left
			glhf.GlFloat(glyph.drawPosX), glhf.GlFloat(glyph.drawPosY), leftU, topV,
			// Bottom-left
			glhf.GlFloat(glyph.drawPosX), glhf.GlFloat(glyph.drawPosY) + gHeight, leftU, bottomV,
			// Bottom-right
			glhf.GlFloat(glyph.drawPosX) + gWidth, glhf.GlFloat(glyph.drawPosY) + gHeight, rightU, bottomV,

			// second triangle
			// Top-left
			glhf.GlFloat(glyph.drawPosX), glhf.GlFloat(glyph.drawPosY), leftU, topV,
			// Bottom-right
			glhf.GlFloat(glyph.drawPosX) + gWidth, glhf.GlFloat(glyph.drawPosY) + gHeight, rightU, bottomV,
			// Top-right
			glhf.GlFloat(glyph.drawPosX) + gWidth, glhf.GlFloat(glyph.drawPosY), rightU, topV,
		}...)
	}
	t.SetVertices(rawVertices)
}
func (t *TextMesh) SetTexture(text *glhf.Texture) {
	t.texture = text
}

func (t *TextMesh) SetVertices(rawVertices []glhf.GlFloat) {
	vertices := glhf.MakeVertexSlice(t.shader, len(rawVertices)/4, len(rawVertices)/4)
	vertices.Begin()
	vertices.SetVertexData(rawVertices)
	vertices.End()
	t.vertices = vertices
}

func (t *TextMesh) GetTransformMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(t.pos.X(), t.pos.Y(), t.pos.Z())
}
