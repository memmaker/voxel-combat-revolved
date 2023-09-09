package client

import (
    "github.com/go-gl/gl/v4.1-core/gl"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/voxel"
)

// pivot here?
// https://www.labri.fr/perso/nrougier/python-opengl/code/chapter-09/linestrip-3d.py

type Line struct {
    vertices  *glhf.VertexSlice[glhf.GlFloat]
    lineParts []mgl32.Vec3
    isHidden  bool
    length    float32
}

func (l *Line) GetPointCount() int {
    return len(l.lineParts)
}

func (l *Line) Draw(shader *glhf.Shader) {
    if l.isHidden || l.vertices == nil {
        return
    }
    shader.SetUniformAttr(ShaderMultiPurpose, l.length)
    l.vertices.Begin()
    l.vertices.Draw()
    l.vertices.End()
}
func (l *Line) UpdateVerticesAndShow(shader *glhf.Shader) {
    pointsInLineCount := 0
    pointsInLineCount += l.GetPointCount()
    vertices := l.toDefaultShaderVertices()
    l.vertices = glhf.MakeVertexSlice(shader, pointsInLineCount*2, pointsInLineCount*2) //, l.createIndices(pointsInLineCount))
    l.vertices.SetPrimitiveType(gl.TRIANGLE_STRIP)
    l.vertices.Begin()
    l.vertices.SetVertexData(vertices)
    l.vertices.End()
    l.isHidden = false
}

func (l *Line) createIndices(length int) []uint32 {
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

/*
	Mapping to default shader:

{Name: "position", Type: glhf.Vec3},  -> vec3 position   //current point on line
{Name: "texCoord", Type: glhf.Vec2}, -> vec2 distance & direction  //a sign, -1 or 1
{Name: "vertexColor", Type: glhf.Vec3}, -> vec3 previous   //previous point on line
{Name: "normal", Type: glhf.Vec3}, -> vec3 next       //next point on line
*/
func (l *Line) toDefaultShaderVertices() []glhf.GlFloat {
    vertices := l.toLineVertices()
    var shaderVertices []glhf.GlFloat
    for _, vertex := range vertices {
        shaderVertices = append(shaderVertices,
            // current
            glhf.GlFloat(vertex.current.X()), glhf.GlFloat(vertex.current.Y()), glhf.GlFloat(vertex.current.Z()),
            // distance & direction
            glhf.GlFloat(vertex.uv.X()), glhf.GlFloat(vertex.uv.Y()),
            // previous
            glhf.GlFloat(vertex.previous.X()), glhf.GlFloat(vertex.previous.Y()), glhf.GlFloat(vertex.previous.Z()),
            // next
            glhf.GlFloat(vertex.next.X()), glhf.GlFloat(vertex.next.Y()), glhf.GlFloat(vertex.next.Z()),
        )
    }
    return shaderVertices
}
func (l *Line) toLineVertices() []LineVertex {
    // adapted from: https://www.labri.fr/perso/nrougier/python-opengl/#d-lines
    //each vertex has the following attribs:
    // vec3 position   //current point on line
    // vec3 previous   //previous point on line
    // vec3 next       //next point on line
    // vec2 uv 		   // distance and a sign, -1 or 1
    cumulativeLength := float32(0)
    vertices := make([]LineVertex, 0)
    var current, previous, next mgl32.Vec3
    for index := 0; index < len(l.lineParts); index++ {
        current = l.lineParts[index]

        if index == 0 {
            previous = current
        } else {
            previous = l.lineParts[index-1]
            cumulativeLength += previous.Sub(current).Len()
        }

        if index == len(l.lineParts)-1 {
            next = current
        } else {
            next = l.lineParts[index+1]
        }

        currentVertex := NewLineVertex(current, previous, next, cumulativeLength)
        vertices = append(vertices, currentVertex)
    }
    /*
       vertices = append(vertices, vertices[len(vertices)-1]) // double the last vertex
       // and the first
       vertices = append([]LineVertex{vertices[0]}, vertices...)
    */

    duplicated := make([]LineVertex, 0)
    for _, vertex := range vertices {
        duplicated = append(duplicated, vertex)
        duplicated = append(duplicated, vertex.MirroredDirection())
    }

    // swap: V[0], V[-1] = V[1], V[-2]
    //duplicated[0], duplicated[len(duplicated)-1] = duplicated[1], duplicated[len(duplicated)-2]

    l.length = cumulativeLength

    return duplicated
}

type LineDrawer struct {
    shader   *glhf.Shader
    lines    []*Line
    isHidden bool
}

func NewLineDrawer(shader *glhf.Shader) *LineDrawer {
    return &LineDrawer{
        shader: shader,
    }
}
func (l *LineDrawer) AddSimpleLine(start, end mgl32.Vec3) {
    l.lines = append(l.lines, &Line{
        lineParts: []mgl32.Vec3{
            start,
            end,
        },
    })
}

func (l *LineDrawer) AddPathLine(wayPoints []mgl32.Vec3) {
    l.lines = append(l.lines, &Line{
        lineParts: wayPoints,
    })
}

func (l *LineDrawer) AddLine(line *Line) {
    l.lines = append(l.lines, line)
}
func (l *LineDrawer) Draw() {
    if l.isHidden {
        return
    }
    for _, line := range l.lines {
        line.Draw(l.shader)
    }
}
func (l *LineDrawer) UpdateVerticesAndShow() {
    for _, line := range l.lines {
        line.UpdateVerticesAndShow(l.shader)
    }
    l.isHidden = false
}

type LineVertex struct {
    current  mgl32.Vec3
    previous mgl32.Vec3
    next     mgl32.Vec3
    uv       mgl32.Vec2 // U is the cumulative length of the line up to this point, V is the direction of the line vertex (-1, 1) inside/outside
}

func (v LineVertex) MirroredDirection() LineVertex {
    return LineVertex{
        current:  v.current,
        previous: v.previous,
        next:     v.next,
        uv:       mgl32.Vec2{v.uv.X(), -v.uv.Y()},
    }
}

func NewLineVertex(current mgl32.Vec3, previous mgl32.Vec3, next mgl32.Vec3, distance float32) LineVertex {
    return LineVertex{
        current:  current,
        previous: previous,
        next:     next,
        uv:       mgl32.Vec2{distance, 1},
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

func (l *LineDrawer) Clear() {
    l.lines = make([]*Line, 0)
    l.isHidden = true
}

func (l *LineDrawer) Hide() {
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
