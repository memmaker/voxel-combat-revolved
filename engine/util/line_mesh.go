package util

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
)

type LineMesh struct {
	vertices *glhf.VertexSlice[glhf.GlFloat]
	pos      mgl32.Vec3
	shader   *glhf.Shader
}

func (m *LineMesh) SetSize(scaleFactor float64) {
	panic("implement me")
}

func (m *LineMesh) SetBlockPosition(pos voxel.Int3) {
	m.pos = pos.ToVec3()
}
func (m *LineMesh) Draw() {
	m.shader.SetUniformAttr(2, m.GetMatrix())
	m.vertices.Begin()
	m.vertices.Draw()
	m.vertices.End()
}

func NewLineMesh(shader *glhf.Shader, lines [][2]mgl32.Vec3) *LineMesh {
	var flatLines []glhf.GlFloat
	for _, line := range lines {
		flatLines = append(flatLines, glhf.GlFloat(line[0].X()), glhf.GlFloat(line[0].Y()), glhf.GlFloat(line[0].Z()))
		flatLines = append(flatLines, glhf.GlFloat(line[1].X()), glhf.GlFloat(line[1].Y()), glhf.GlFloat(line[1].Z()))
	}
	var vertices *glhf.VertexSlice[glhf.GlFloat]
	vertices = glhf.MakeVertexSlice(shader, len(lines)*2, len(lines)*2)
	vertices.SetPrimitiveType(gl.LINES)
	vertices.Begin()
	vertices.SetVertexData(flatLines)
	vertices.End()

	return &LineMesh{
		vertices: vertices,
		shader:   shader,
	}
}
func (m *LineMesh) GetMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(m.pos.X(), m.pos.Y(), m.pos.Z())
}

func (m *LineMesh) GetBlockPosition() voxel.Int3 {
	return voxel.PositionToGridInt3(m.pos)
}
