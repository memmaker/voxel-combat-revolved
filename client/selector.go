package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

func NewBlockSelector(shader *glhf.Shader) *util.LineMesh {
	blockSelector := util.NewLineMesh(shader, [][2]mgl32.Vec3{
		// we need to draw 12 lines, each line has 2 points, should be a wireframe cube
		// bottom
		{mgl32.Vec3{0, 0, 0}, mgl32.Vec3{1, 0, 0}},
		{mgl32.Vec3{1, 0, 0}, mgl32.Vec3{1, 0, 1}},
		{mgl32.Vec3{1, 0, 1}, mgl32.Vec3{0, 0, 1}},
		{mgl32.Vec3{0, 0, 1}, mgl32.Vec3{0, 0, 0}},
		// top
		{mgl32.Vec3{0, 1, 0}, mgl32.Vec3{1, 1, 0}},
		{mgl32.Vec3{1, 1, 0}, mgl32.Vec3{1, 1, 1}},
		{mgl32.Vec3{1, 1, 1}, mgl32.Vec3{0, 1, 1}},
		{mgl32.Vec3{0, 1, 1}, mgl32.Vec3{0, 1, 0}},

		// sides
		{mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0}},
		{mgl32.Vec3{1, 0, 0}, mgl32.Vec3{1, 1, 0}},
		{mgl32.Vec3{1, 0, 1}, mgl32.Vec3{1, 1, 1}},
		{mgl32.Vec3{0, 0, 1}, mgl32.Vec3{0, 1, 1}},
	})

	return blockSelector
}

type PositionDrawable interface {
	SetPosition(pos mgl32.Vec3)
	Draw()
	SetSize(scaleFactor float64)
}

type GroundSelector struct {
	*util.Transform
	mesh   *util.CompoundMesh
	shader *glhf.Shader
	hide   bool
}

func (g *GroundSelector) SetSize(scaleFactor float64) {
	println("Nonsense")
}

func (g *GroundSelector) SetPosition(pos mgl32.Vec3) {
	offset := mgl32.Vec3{0.5, 0.025, 0.5}
	g.Transform.SetPosition(pos.Add(offset))
	g.hide = false
}

func (g *GroundSelector) Hide() {
	g.hide = true
}

func (g *GroundSelector) Draw() {
	if g.hide {
		return
	}
	g.mesh.Draw(g.shader, ShaderModelMatrix)
}

func NewGroundSelector(mesh *util.CompoundMesh, shader *glhf.Shader) *GroundSelector {
	groundSelector := &GroundSelector{
		Transform: util.NewDefaultTransform("Ground Selector"),
		mesh:      mesh,
		shader:    shader,
		hide:      true,
	}
	mesh.ConvertVertexData(shader)
	mesh.RootNode.SetParent(groundSelector)
	return groundSelector
}
