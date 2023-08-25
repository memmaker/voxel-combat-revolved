package voxel

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

// idea:
// have a "manager" for all the highlights we need.
// it basically records positions and colors and then renders them as one mesh with quads

type Highlights struct {
	vertexData        *glhf.VertexSlice[glhf.GlFloat]
	colorMap          map[Int3]mgl32.Vec3
	namedHighlightMap map[string]map[Int3]mgl32.Vec3
	shader            *glhf.Shader
	isHidden          bool
}

func NewHighlights(shader *glhf.Shader) *Highlights {
	return &Highlights{
		shader:   shader,
		colorMap: make(map[Int3]mgl32.Vec3),
	}
}

func (h *Highlights) Add(position Int3, color mgl32.Vec3) {
	h.colorMap[position] = color
}

func (h *Highlights) AddMulti(position []Int3, color mgl32.Vec3) {
	for _, p := range position {
		h.Add(p, color)
	}
}
func (h *Highlights) SetNamedMultiAndUpdate(name string, positions []Int3, color mgl32.Vec3) {
	if h.namedHighlightMap == nil {
		h.namedHighlightMap = make(map[string]map[Int3]mgl32.Vec3)
	}
	if _, ok := h.namedHighlightMap[name]; !ok {
		h.namedHighlightMap[name] = make(map[Int3]mgl32.Vec3)
	} else {
		clear(h.namedHighlightMap[name])
	}
	for _, p := range positions {
		h.namedHighlightMap[name][p] = color
	}
	h.UpdateVertexData()
}
func (h *Highlights) SetMultiAndUpdate(position []Int3, color mgl32.Vec3) {
	h.Clear()
	h.AddMulti(position, color)
	h.UpdateVertexData()
}

func (h *Highlights) UnSet(position Int3) {
	delete(h.colorMap, position)
}

func (h *Highlights) Clear() {
	h.colorMap = make(map[Int3]mgl32.Vec3)
}

func (h *Highlights) Draw() {
	if h.isHidden || (len(h.colorMap) == 0 && len(h.namedHighlightMap) == 0) {
		return
	}

	h.vertexData.Begin()
	h.vertexData.Draw()
	h.vertexData.End()
}
func (h *Highlights) Hide() {
	h.isHidden = true
}
func (h *Highlights) UpdateVertexData() {
	allVertices := make([]glhf.GlFloat, 0)
	allIndices := make([]uint32, 0)
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal
	for position, color := range h.colorMap {
		allVertices, allIndices = h.appendQuad(allVertices, allIndices, position, color, floatsPerVertex)
	}
	for _, namedHighlight := range h.namedHighlightMap {
		for position, color := range namedHighlight {
			allVertices, allIndices = h.appendQuad(allVertices, allIndices, position, color, floatsPerVertex)
		}
	}

	vertexCount := len(allVertices) / floatsPerVertex

	h.vertexData = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, allIndices)
	h.vertexData.Begin()
	h.vertexData.SetVertexData(allVertices)
	h.vertexData.End()
}

func (h *Highlights) appendQuad(allVertices []glhf.GlFloat, allIndices []uint32, position Int3, color mgl32.Vec3, stride int) ([]glhf.GlFloat, []uint32) {
	// we need a quad that is 1x1 in size and parallel to the xz plane
	topLeft := mgl32.Vec3{float32(position.X), float32(position.Y), float32(position.Z)}
	topRight := mgl32.Vec3{float32(position.X + 1), float32(position.Y), float32(position.Z)}
	bottomRight := mgl32.Vec3{float32(position.X + 1), float32(position.Y), float32(position.Z + 1)}
	bottomLeft := mgl32.Vec3{float32(position.X), float32(position.Y), float32(position.Z + 1)}
	normal := mgl32.Vec3{0, 1, 0}
	yOffset := float32(0.05)
	startIndex := uint32(len(allVertices) / stride)
	allVertices = append(allVertices,
		[]glhf.GlFloat{
			// positions (x,y,z=0)          // texture coords (0..1,0..1) // color (1,1,1) // normal (0,0,1)
			//tl
			glhf.GlFloat(topLeft.X()), glhf.GlFloat(topLeft.Y() + yOffset), glhf.GlFloat(topLeft.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//tr
			glhf.GlFloat(topRight.X()), glhf.GlFloat(topRight.Y() + yOffset), glhf.GlFloat(topRight.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//br
			glhf.GlFloat(bottomRight.X()), glhf.GlFloat(bottomRight.Y() + yOffset), glhf.GlFloat(bottomRight.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//bl
			glhf.GlFloat(bottomLeft.X()), glhf.GlFloat(bottomLeft.Y() + yOffset), glhf.GlFloat(bottomLeft.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),
		}...)
	allIndices = append(allIndices, []uint32{
		// first triangle
		startIndex + 1, // top right
		startIndex + 0, // top left
		startIndex + 2, // bottom right
		// second triangle
		startIndex + 2, // bottom right
		startIndex + 0, // top left
		startIndex + 3, // bottom left
	}...)

	return allVertices, allIndices
}

func (h *Highlights) GetTransparentColor() mgl32.Vec4 {
	return mgl32.Vec4{1.0, 1.0, 1.0, 0.7}
}
