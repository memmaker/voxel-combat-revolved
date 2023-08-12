package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

type Crosshair struct {
	vertices     *glhf.VertexSlice[glhf.GlFloat]
	shader       *glhf.Shader
	screenWidth  int
	screenHeight int

	translation   [3]float32
	quatRotation  mgl32.Quat
	originalScale [3]float32
	currentScale  [3]float32
	color         mgl32.Vec3

	size              float32
	currentThickness  float32
	originalThickness float32
}

func NewCrosshair(shader *glhf.Shader, screenWidth, screenHeight int) *Crosshair {
	vertices := glhf.MakeIndexedVertexSlice(shader, 4, 4, []uint32{
		// first triangle
		0, // top left
		1, // top right
		2, // bottom right
		// second triangle
		0, // top left
		2, // bottom right
		3, // bottom left
	})
	vertices.Begin()
	vertices.SetVertexData([]glhf.GlFloat{
		// positions          // texture coords
		-0.5, 0.5, 0.0, 0.0, // top left
		0.5, 0.5, 1.0, 0.0, // top right
		0.5, -0.5, 1.0, 1.0, // bottom right
		-0.5, -0.5, 0.0, 1.0, // bottom left
	})
	vertices.End()
	scale := [3]float32{1, 1, 1}
	// adjust currentScale to match screen aspect ratio
	aspectRatio := float32(screenHeight) / float32(screenWidth) // eg, 600 / 800 = 0.75
	scale[0] = aspectRatio
	c := &Crosshair{
		shader:       shader,
		vertices:     vertices,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,

		translation:   [3]float32{0.5, 0.5, 0},
		quatRotation:  mgl32.QuatIdent(),
		currentScale:  scale,
		originalScale: scale,

		originalThickness: 0.02,
		currentThickness:  0.02,
		color:             mgl32.Vec3{float32(47) / float32(255), float32(214) / float32(255), float32(195) / float32(255)},
		size:              1.0,
	}
	//c.SetSize(0.75)
	return c
}

func (c *Crosshair) SetPosition(pos mgl32.Vec3) {
	c.translation = [3]float32{pos.X(), pos.Y(), pos.Z()}
}
func (c *Crosshair) localMatrix() mgl32.Mat4 {
	translation := mgl32.Translate3D(c.translation[0], c.translation[1], c.translation[2])
	quaternion := c.quatRotation.Mat4()
	scale := mgl32.Scale3D(c.currentScale[0], c.currentScale[1], c.currentScale[2])
	return translation.Mul4(quaternion).Mul4(scale)
}
func (c *Crosshair) Draw() {
	c.shader.SetUniformAttr(0, util.Get2DOrthographicProjectionMatrix())
	c.shader.SetUniformAttr(1, c.localMatrix())
	c.shader.SetUniformAttr(2, c.color)
	c.shader.SetUniformAttr(3, c.currentThickness)
	c.vertices.Begin()
	c.vertices.Draw()
	c.vertices.End()
}

func (c *Crosshair) SetSize(size float64) {
	c.size = float32(size)
	c.currentScale = [3]float32{c.originalScale[0] * c.size, c.originalScale[1] * c.size, c.originalScale[2] * c.size}
	c.currentThickness = c.originalThickness / c.size
}

func (c *Crosshair) SetColor(color mgl32.Vec3) {
	c.color = color
}

func (c *Crosshair) SetThickness(thickness float64) {
	c.originalThickness = float32(thickness)
	c.currentThickness = c.originalThickness / c.size
}
