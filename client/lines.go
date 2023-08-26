package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
)

// pivot here?
// https://www.labri.fr/perso/nrougier/python-opengl/code/chapter-09/linestrip-3d.py

type Line struct {
	lineParts []mgl32.Vec3
}

func (l Line) GetPointCount() int {
	return len(l.lineParts)
}

type LineDrawer struct {
	vertices *glhf.VertexSlice[glhf.GlFloat]
	shader   *glhf.Shader
	lines    []Line
	isHidden bool
}

func NewLineDrawer(shader *glhf.Shader) *LineDrawer {
	return &LineDrawer{
		shader: shader,
	}
}
func (l *LineDrawer) AddSimpleLine(start, end mgl32.Vec3) {
	l.lines = append(l.lines, Line{
		lineParts: []mgl32.Vec3{
			start,
			end,
		},
	})
}

func (l *LineDrawer) Draw() {
	if l.isHidden || len(l.lines) == 0 || l.vertices == nil {
		return
	}
	l.shader.SetUniformAttr(ShaderModelMatrix, mgl32.Ident4())
	l.shader.SetUniformAttr(ShaderDrawMode, ShaderDrawLine)
	l.vertices.Begin()
	l.vertices.Draw()
	l.vertices.End()

}
func (l *LineDrawer) AddLine(line Line) {
	l.lines = append(l.lines, line)
}

func (l *LineDrawer) UpdateVerticesAndShow() {
	var vertices []glhf.GlFloat
	pointsInLineCount := 0
	for _, line := range l.lines {
		vertices = append(vertices, line.toDefaultShaderVertices()...)
		pointsInLineCount += line.GetPointCount()
	}
	l.vertices = glhf.MakeIndexedVertexSlice(l.shader, pointsInLineCount*2, pointsInLineCount*2, l.createIndices(pointsInLineCount))
	l.vertices.Begin()
	l.vertices.SetVertexData(vertices)
	l.vertices.End()
	l.isHidden = false
}

type LineVertex struct {
	position  mgl32.Vec3
	previous  mgl32.Vec3
	next      mgl32.Vec3
	direction float32
}

func (v LineVertex) MirroredDirection() LineVertex {
	return LineVertex{
		position:  v.position,
		previous:  v.previous,
		next:      v.next,
		direction: -v.direction,
	}
}

/*
	Mapping to default shader:

{Name: "position", Type: glhf.Vec3},  -> vec3 position   //current point on line
{Name: "texCoord", Type: glhf.Vec2}, X-Coordinate  -> vec3 direction  //a sign, -1 or 1
{Name: "vertexColor", Type: glhf.Vec3}, -> vec3 previous   //previous point on line
{Name: "normal", Type: glhf.Vec3}, -> vec3 next       //next point on line
*/
func (l Line) toDefaultShaderVertices() []glhf.GlFloat {
	vertices := l.toLineVertices()
	var shaderVertices []glhf.GlFloat
	for _, vertex := range vertices {
		shaderVertices = append(shaderVertices,
			// position
			glhf.GlFloat(vertex.position.X()), glhf.GlFloat(vertex.position.Y()), glhf.GlFloat(vertex.position.Z()),
			// direction
			glhf.GlFloat(vertex.direction), 0.0,
			// previous
			glhf.GlFloat(vertex.previous.X()), glhf.GlFloat(vertex.previous.Y()), glhf.GlFloat(vertex.previous.Z()),
			// next
			glhf.GlFloat(vertex.next.X()), glhf.GlFloat(vertex.next.Y()), glhf.GlFloat(vertex.next.Z()),
		)
	}
	return shaderVertices
}
func (l Line) toLineVertices() []LineVertex {
	// adapted from: https://mattdesl.svbtle.com/drawing-lines-is-hard
	//each vertex has the following attribs:
	// vec3 position   //current point on line
	// vec3 previous   //previous point on line
	// vec3 next       //next point on line
	// float direction //a sign, -1 or 1
	vertices := make([]LineVertex, 0)
	var current, previous, next mgl32.Vec3
	for index := 0; index < len(l.lineParts); index++ {
		current = l.lineParts[index]

		if index == 0 {
			previous = current
		} else {
			previous = l.lineParts[index-1]
		}

		if index == len(l.lineParts)-1 {
			next = current
		} else {
			next = l.lineParts[index+1]
		}

		currentVertex := NewLineVertex(current, previous, next)

		vertices = append(vertices, currentVertex)
		vertices = append(vertices, currentVertex.MirroredDirection())
	}
	return vertices
}

func NewLineVertex(current mgl32.Vec3, previous mgl32.Vec3, next mgl32.Vec3) LineVertex {
	return LineVertex{
		position:  current,
		previous:  previous,
		next:      next,
		direction: 1.0,
	}
}

/*
//counter-clockwise indices but prepared for duplicate vertices
module.exports.createIndices = function createIndices(length) {
  let indices = new Uint16Array(length * 6)
  let c = 0, index = 0
  for (let j=0; j<length; j++) {
    let i = index
    indices[c++] = i + 0
    indices[c++] = i + 1
    indices[c++] = i + 2
    indices[c++] = i + 2
    indices[c++] = i + 1
    indices[c++] = i + 3
    index += 2
  }
  return indices
}
*/

func (l *LineDrawer) createIndices(length int) []uint32 {
	indices := make([]uint32, length*6)
	c := 0
	index := 0
	for j := 0; j < length; j++ {
		i := index
		indices[c] = uint32(i + 0)
		c++
		indices[c] = uint32(i + 1)
		c++
		indices[c] = uint32(i + 2)
		c++
		indices[c] = uint32(i + 2)
		c++
		indices[c] = uint32(i + 1)
		c++
		indices[c] = uint32(i + 3)
		c++
		index += 2
	}
	return indices
}

func (l *LineDrawer) AddCubeAt(cubePos voxel.Int3) {
	cubeLine := NewOutlinedCube(cubePos, cubePos.Add(voxel.Int3{X: 1, Y: 1, Z: 1}))
	for _, line := range cubeLine {
		l.AddLine(line)
	}
}

func (l *LineDrawer) Clear() {
	l.lines = make([]Line, 0)
	l.isHidden = true
}

func NewOutlinedCube(min voxel.Int3, max voxel.Int3) []Line {
	// top
	topLeftFront := mgl32.Vec3{float32(min.X), float32(max.Y), float32(min.Z)}
	topRightFront := mgl32.Vec3{float32(max.X), float32(max.Y), float32(min.Z)}
	topLeftBack := mgl32.Vec3{float32(min.X), float32(max.Y), float32(max.Z)}
	topRightBack := mgl32.Vec3{float32(max.X), float32(max.Y), float32(max.Z)}

	// bottom
	bottomLeftFront := mgl32.Vec3{float32(min.X), float32(min.Y), float32(min.Z)}
	bottomRightFront := mgl32.Vec3{float32(max.X), float32(min.Y), float32(min.Z)}
	bottomLeftBack := mgl32.Vec3{float32(min.X), float32(min.Y), float32(max.Z)}
	bottomRightBack := mgl32.Vec3{float32(max.X), float32(min.Y), float32(max.Z)}

	// lines
	top := Line{lineParts: []mgl32.Vec3{topLeftFront, topRightFront, topRightBack, topLeftBack}}
	bottom := Line{lineParts: []mgl32.Vec3{bottomLeftFront, bottomRightFront, bottomRightBack, bottomLeftBack}}

	tl := Line{lineParts: []mgl32.Vec3{topLeftFront, bottomLeftFront}}
	tr := Line{lineParts: []mgl32.Vec3{topRightFront, bottomRightFront}}
	bl := Line{lineParts: []mgl32.Vec3{topLeftBack, bottomLeftBack}}
	br := Line{lineParts: []mgl32.Vec3{topRightBack, bottomRightBack}}

	return []Line{top, bottom, tl, tr, bl, br}
}
