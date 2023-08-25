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
	color         mgl32.Vec4

	size              float32
	currentThickness  float32
	originalThickness float32
	camera            *util.FPSCamera

	isHidden bool
}

func NewCrosshair(shader *glhf.Shader, cam *util.FPSCamera) *Crosshair {

	scale := [3]float32{1, 1, 1}
	// adjust currentScale to match screen aspect ratio
	aspectRatio := cam.GetAspectRatio() // eg, 600 / 800 = 0.75
	scale[0] = aspectRatio
	c := &Crosshair{
		shader:            shader,
		camera:            cam,
		screenWidth:       cam.GetScreenWidth(),
		screenHeight:      cam.GetScreenHeight(),
		translation:       [3]float32{0, 0, -(cam.GetNearPlaneDist() + 0.01)},
		quatRotation:      mgl32.QuatIdent(),
		currentScale:      scale,
		originalScale:     scale,
		originalThickness: 0.02,
		currentThickness:  0.02,
		color:             ColorTechTeal,
		size:              1.0,
	}
	c.Init(shader)
	//c.SetSize(0.75)
	return c
}

func (c *Crosshair) SetHidden(hidden bool) {
	c.isHidden = hidden
}

func (c *Crosshair) IsHidden() bool {
	return c.isHidden
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
	c.shader.SetUniformAttr(ShaderProjectionViewMatrix, c.camera.GetProjectionMatrix())
	c.shader.SetUniformAttr(ShaderModelMatrix, c.localMatrix())
	c.shader.SetUniformAttr(ShaderDrawMode, ShaderDrawCircle)

	c.shader.SetUniformAttr(ShaderDrawColor, c.color)
	c.shader.SetUniformAttr(ShaderThickness, c.thickness())

	c.vertices.Begin()
	c.vertices.Draw()
	c.vertices.End()
}

func (c *Crosshair) SetSize(size float64) {
	c.size = mgl32.Clamp(float32(size), 0.05, 1.0)
	// size should be 1.0 at fov of 45 degrees
	c.currentScale = [3]float32{c.size, c.size, c.size}
	c.currentThickness = c.originalThickness / c.size
}
func (c *Crosshair) SetColor(color mgl32.Vec4) {
	c.color = color
}

func (c *Crosshair) SetThickness(thickness float64) {
	c.originalThickness = float32(thickness)
	c.currentThickness = c.originalThickness / c.size
}

func (c *Crosshair) GetNearPlaneQuad() []glhf.GlFloat {
	cam := c.camera
	proj := cam.GetProjectionMatrix()
	projInverted := proj.Inv()
	topLeft := c.transformVertex(-0.5/cam.GetAspectRatio(), 0.5, cam, projInverted)
	topRight := c.transformVertex(0.5/cam.GetAspectRatio(), 0.5, cam, projInverted)
	bottomRight := c.transformVertex(0.5/cam.GetAspectRatio(), -0.5, cam, projInverted)
	bottomLeft := c.transformVertex(-0.5/cam.GetAspectRatio(), -0.5, cam, projInverted)
	/*
		{Name: "position", Type: glhf.Vec3},
		{Name: "texCoord", Type: glhf.Vec2},
		{Name: "vertexColor", Type: glhf.Vec3},
		{Name: "normal", Type: glhf.Vec3},
	*/
	return []glhf.GlFloat{
		// positions (x,y,z=0)          // texture coords // color (1,1,1) // normal (0,0,1)
		glhf.GlFloat(topLeft.X()), glhf.GlFloat(topLeft.Y()), 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0, 0, 1.0, // top left
		glhf.GlFloat(topRight.X()), glhf.GlFloat(topRight.Y()), 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 0, 0, 1.0, // top right
		glhf.GlFloat(bottomRight.X()), glhf.GlFloat(bottomRight.Y()), 0.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0, 0, 1.0, // bottom right
		glhf.GlFloat(bottomLeft.X()), glhf.GlFloat(bottomLeft.Y()), 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0, 0, 1.0, // bottom left
	}
}

func (c *Crosshair) Init(shader *glhf.Shader) {
	vertices := glhf.MakeIndexedVertexSlice(shader, 4, 4, []uint32{
		// first triangle
		1, // top right
		0, // top left
		2, // bottom right
		// second triangle
		2, // bottom right
		0, // top left
		3, // bottom left

	})
	vertices.Begin()
	vertices.SetVertexData(c.GetNearPlaneQuad())
	vertices.End()
	c.vertices = vertices
}

func (c *Crosshair) transformVertex(x, y float32, cam *util.FPSCamera, projViewInverted mgl32.Mat4) mgl32.Vec3 {
	normalizedNearPos := mgl32.Vec4{x, y, cam.GetNearPlaneDist(), 1}
	// project point from camera space to world space
	nearWorldPos := projViewInverted.Mul4x1(normalizedNearPos)
	// perspective divide
	correctedNearWorldPos := nearWorldPos.Vec3().Mul(1 / nearWorldPos.W())
	return correctedNearWorldPos
}

func (c *Crosshair) thickness() float32 {
	fov := c.camera.GetFOV()
	fovFactor := fov / 45.0
	return c.currentThickness * fovFactor
}
