package voxel

import (
    "github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

// idea:
// decals are basically the same as my highlights, just being able to be attached to any face

type Highlight int

const (
	HighlightMove Highlight = iota
	HighlightTarget
	HighlightOverwatch
	HighlightEditor
)

type Highlights struct {
	flatVertices  *glhf.VertexSlice[glhf.GlFloat]
	fancyVertices *glhf.VertexSlice[glhf.GlFloat]
	fancies       map[Highlight]map[Int3]mgl32.Vec3
	flats         map[Highlight]map[Int3]mgl32.Vec3
	shader        *glhf.Shader
	isHidden      bool
}

func NewHighlights(shader *glhf.Shader) *Highlights {
	return &Highlights{
		shader:  shader,
		flats:   make(map[Highlight]map[Int3]mgl32.Vec3),
		fancies: make(map[Highlight]map[Int3]mgl32.Vec3),
	}
}

func (h *Highlights) AddFlat(name Highlight, position []Int3, color mgl32.Vec3) {
	if _, ok := h.flats[name]; !ok {
		h.flats[name] = make(map[Int3]mgl32.Vec3)
	}
	for _, p := range position {
		h.flats[name][p] = color
	}
}

func (h *Highlights) AddFancy(name Highlight, position []Int3, color mgl32.Vec3) {
	if _, ok := h.fancies[name]; !ok {
		h.fancies[name] = make(map[Int3]mgl32.Vec3)
	}
	for _, p := range position {
		h.fancies[name][p] = color
	}
}

func (h *Highlights) SetFlat(name Highlight, positions []Int3, color mgl32.Vec3) {
	if _, ok := h.flats[name]; !ok {
		h.flats[name] = make(map[Int3]mgl32.Vec3)
	} else {
		clear(h.flats[name])
	}
	for _, p := range positions {
		h.flats[name][p] = color
	}
	h.ShowAsFlat(name)
}

func (h *Highlights) ClearFlat(name Highlight) {
	if _, ok := h.flats[name]; !ok {
		return
	}
	clear(h.flats[name])
}
func (h *Highlights) ClearFancy(name Highlight) {
	if _, ok := h.fancies[name]; !ok {
		return
	}
	clear(h.fancies[name])
}
func (h *Highlights) ClearAll() {
	for _, namedHighlight := range h.flats {
		clear(namedHighlight)
	}
	clear(h.flats)

	for _, namedHighlight := range h.fancies {
		clear(namedHighlight)
	}
	clear(h.fancies)
}

func (h *Highlights) Draw(uniformForDrawMode int, fancyQuadDrawMode int32) {
	if h.isHidden {
		return
	}
	if h.flatVertices != nil && h.flatVertices.Len() > 0 {
		h.flatVertices.Begin()
		h.flatVertices.Draw()
		h.flatVertices.End()
	}
	if h.fancyVertices != nil && h.fancyVertices.Len() > 0 {
		h.shader.SetUniformAttr(uniformForDrawMode, fancyQuadDrawMode)
		gl.Disable(gl.CULL_FACE)
		h.fancyVertices.Begin()
		h.fancyVertices.Draw()
		h.fancyVertices.End()
		gl.Enable(gl.CULL_FACE)
	}
}
func (h *Highlights) Hide() {
	h.isHidden = true
}
func (h *Highlights) ShowAsFlat(category Highlight) {
	highlights, ok := h.flats[category]
	if !ok {
		return
	}
	h.isHidden = false

	allVertices := make([]glhf.GlFloat, 0)
	allIndices := make([]uint32, 0)
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal

	for position, color := range highlights {
		allVertices, allIndices = h.appendQuad(allVertices, allIndices, position, color, floatsPerVertex)
	}

	vertexCount := len(allVertices) / floatsPerVertex

	h.flatVertices = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, allIndices)
	h.flatVertices.Begin()
	h.flatVertices.SetVertexData(allVertices)
	h.flatVertices.End()
}

func (h *Highlights) ShowAsFancy(category Highlight) {
	highlights, ok := h.fancies[category]
	if !ok {
		return
	}

	h.isHidden = false

	allVertices := make([]glhf.GlFloat, 0)
	allIndices := make([]uint32, 0)
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal

	for position, color := range highlights {
		allVertices, allIndices = h.appendFancyQuads(allVertices, allIndices, position, color, floatsPerVertex)
	}

	vertexCount := len(allVertices) / floatsPerVertex

	h.fancyVertices = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, allIndices)
	h.fancyVertices.Begin()
	h.fancyVertices.SetVertexData(allVertices)
	h.fancyVertices.End()
}
func (h *Highlights) appendFancyQuads(allVertices []glhf.GlFloat, allIndices []uint32, position Int3, color mgl32.Vec3, stride int) ([]glhf.GlFloat, []uint32) {
	// we need 4 quads that 1 unit wide and 2 units in height
	// we want one quad at the border of each side of the block
	// pointing up
	fancyHeight := float32(2)
	inset := float32(0.05)
	topLeftGround := mgl32.Vec3{float32(position.X) + inset, float32(position.Y), float32(position.Z) + inset}
	topRightGround := mgl32.Vec3{float32(position.X+1) - inset, float32(position.Y), float32(position.Z) + inset}
	bottomRightGround := mgl32.Vec3{float32(position.X+1) - inset, float32(position.Y), float32(position.Z+1) - inset}
	bottomLeftGround := mgl32.Vec3{float32(position.X) + inset, float32(position.Y), float32(position.Z+1) - inset}

	topLeftUpper := topLeftGround.Add(mgl32.Vec3{0, fancyHeight, 0})
	topRightUpper := topRightGround.Add(mgl32.Vec3{0, fancyHeight, 0})
	bottomRightUpper := bottomRightGround.Add(mgl32.Vec3{0, fancyHeight, 0})
	bottomLeftUpper := bottomLeftGround.Add(mgl32.Vec3{0, fancyHeight, 0})

	// west side border (needs tlg, blg, tlu, blu)
	westNormal := mgl32.Vec3{-1, 0, 0}
	eastNormal := mgl32.Vec3{1, 0, 0}
	northNormal := mgl32.Vec3{0, 0, -1}
	southNormal := mgl32.Vec3{0, 0, 1}
	verts := []glhf.GlFloat{ //  going clockwise, starting from top-left
		// west side
		//tlu -> Top-Left
		glhf.GlFloat(topLeftUpper.X()), glhf.GlFloat(topLeftUpper.Y()), glhf.GlFloat(topLeftUpper.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(westNormal.X()), glhf.GlFloat(westNormal.Y()), glhf.GlFloat(westNormal.Z()),

		//blu -> Top-Right
		glhf.GlFloat(bottomLeftUpper.X()), glhf.GlFloat(bottomLeftUpper.Y()), glhf.GlFloat(bottomLeftUpper.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(westNormal.X()), glhf.GlFloat(westNormal.Y()), glhf.GlFloat(westNormal.Z()),

		//blg -> Bottom-Right
		glhf.GlFloat(bottomLeftGround.X()), glhf.GlFloat(bottomLeftGround.Y()), glhf.GlFloat(bottomLeftGround.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(westNormal.X()), glhf.GlFloat(westNormal.Y()), glhf.GlFloat(westNormal.Z()),

		//tlg -> Bottom-Left
		glhf.GlFloat(topLeftGround.X()), glhf.GlFloat(topLeftGround.Y()), glhf.GlFloat(topLeftGround.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(westNormal.X()), glhf.GlFloat(westNormal.Y()), glhf.GlFloat(westNormal.Z()),

		// east side
		// bru -> Top-Left
		glhf.GlFloat(bottomRightUpper.X()), glhf.GlFloat(bottomRightUpper.Y()), glhf.GlFloat(bottomRightUpper.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(eastNormal.X()), glhf.GlFloat(eastNormal.Y()), glhf.GlFloat(eastNormal.Z()),

		// tru -> Top-Right
		glhf.GlFloat(topRightUpper.X()), glhf.GlFloat(topRightUpper.Y()), glhf.GlFloat(topRightUpper.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(eastNormal.X()), glhf.GlFloat(eastNormal.Y()), glhf.GlFloat(eastNormal.Z()),

		// trg -> Bottom-Right
		glhf.GlFloat(topRightGround.X()), glhf.GlFloat(topRightGround.Y()), glhf.GlFloat(topRightGround.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(eastNormal.X()), glhf.GlFloat(eastNormal.Y()), glhf.GlFloat(eastNormal.Z()),

		// brg -> Bottom-Left
		glhf.GlFloat(bottomRightGround.X()), glhf.GlFloat(bottomRightGround.Y()), glhf.GlFloat(bottomRightGround.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(eastNormal.X()), glhf.GlFloat(eastNormal.Y()), glhf.GlFloat(eastNormal.Z()),

		// south side
		// blu -> Top-Left
		glhf.GlFloat(bottomLeftUpper.X()), glhf.GlFloat(bottomLeftUpper.Y()), glhf.GlFloat(bottomLeftUpper.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(southNormal.X()), glhf.GlFloat(southNormal.Y()), glhf.GlFloat(southNormal.Z()),
		// bru -> Top-Right
		glhf.GlFloat(bottomRightUpper.X()), glhf.GlFloat(bottomRightUpper.Y()), glhf.GlFloat(bottomRightUpper.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(southNormal.X()), glhf.GlFloat(southNormal.Y()), glhf.GlFloat(southNormal.Z()),
		// brg -> Bottom-Right
		glhf.GlFloat(bottomRightGround.X()), glhf.GlFloat(bottomRightGround.Y()), glhf.GlFloat(bottomRightGround.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(southNormal.X()), glhf.GlFloat(southNormal.Y()), glhf.GlFloat(southNormal.Z()),
		// blg -> Bottom-Left
		glhf.GlFloat(bottomLeftGround.X()), glhf.GlFloat(bottomLeftGround.Y()), glhf.GlFloat(bottomLeftGround.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(southNormal.X()), glhf.GlFloat(southNormal.Y()), glhf.GlFloat(southNormal.Z()),

		// north side
		// tru -> Top-Left
		glhf.GlFloat(topRightUpper.X()), glhf.GlFloat(topRightUpper.Y()), glhf.GlFloat(topRightUpper.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(northNormal.X()), glhf.GlFloat(northNormal.Y()), glhf.GlFloat(northNormal.Z()),
		// tlu -> Top-Right
		glhf.GlFloat(topLeftUpper.X()), glhf.GlFloat(topLeftUpper.Y()), glhf.GlFloat(topLeftUpper.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(northNormal.X()), glhf.GlFloat(northNormal.Y()), glhf.GlFloat(northNormal.Z()),

		// tlg -> Bottom-Right
		glhf.GlFloat(topLeftGround.X()), glhf.GlFloat(topLeftGround.Y()), glhf.GlFloat(topLeftGround.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(northNormal.X()), glhf.GlFloat(northNormal.Y()), glhf.GlFloat(northNormal.Z()),

		// trg -> Bottom-Left
		glhf.GlFloat(topRightGround.X()), glhf.GlFloat(topRightGround.Y()), glhf.GlFloat(topRightGround.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(northNormal.X()), glhf.GlFloat(northNormal.Y()), glhf.GlFloat(northNormal.Z()),
	}
	startIndex := uint32(len(allVertices) / stride)
	indices := []uint32{ // defined in CCW order
		// first triangle
		startIndex + 1, // Top-Right
		startIndex + 0, // Top-Left
		startIndex + 2, // Bottom-Right

		// second triangle
		startIndex + 2, // Bottom-Right
		startIndex + 0, // Top-Left
		startIndex + 3, // Bottom-Left

		// second quad
		startIndex + 5, // Top-Right
		startIndex + 4, // Top-Left
		startIndex + 6, // Bottom-Right

		// second triangle
		startIndex + 6, // Bottom-Right
		startIndex + 4, // Top-Left
		startIndex + 7, // Bottom-Left

		// third quad
		startIndex + 9,  // Top-Right
		startIndex + 8,  // Top-Left
		startIndex + 10, // Bottom-Right

		// second triangle
		startIndex + 10, // Bottom-Right
		startIndex + 8,  // Top-Left
		startIndex + 11, // Bottom-Left

		// fourth quad
		startIndex + 13, // Top-Right
		startIndex + 12, // Top-Left
		startIndex + 14, // Bottom-Right

		// second triangle
		startIndex + 14, // Bottom-Right
		startIndex + 12, // Top-Left
		startIndex + 15, // Bottom-Left
	}

	allVertices = append(allVertices, verts...)
	allIndices = append(allIndices, indices...)

	return allVertices, allIndices
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

func (h *Highlights) GetTintColor() mgl32.Vec4 {
	return mgl32.Vec4{1.0, 1.0, 1.0, 0.3}
}

func (h *Highlights) ClearAndUpdateFlat(category Highlight) {
	h.ClearFlat(category)
	h.updateFlats()
}

func (h *Highlights) ClearAndUpdateFancy(category Highlight) {
	h.ClearFancy(category)
	h.updateFancies()
}

func (h *Highlights) updateAll() {
	h.updateFlats()
	h.updateFancies()
}

func (h *Highlights) updateFancies() {
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal

	fancyVerts := make([]glhf.GlFloat, 0)
	fancyIndices := make([]uint32, 0)

	for _, highlights := range h.fancies {
		for position, color := range highlights {
			fancyVerts, fancyIndices = h.appendFancyQuads(fancyVerts, fancyIndices, position, color, floatsPerVertex)
		}
	}

	vertexCount := len(fancyVerts) / floatsPerVertex

	h.fancyVertices = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, fancyIndices)
	h.fancyVertices.Begin()
	h.fancyVertices.SetVertexData(fancyVerts)
	h.fancyVertices.End()
}

func (h *Highlights) updateFlats() {
	flatVerts := make([]glhf.GlFloat, 0)
	flatIndices := make([]uint32, 0)
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal
	for _, highlights := range h.flats {
		for position, color := range highlights {
			flatVerts, flatIndices = h.appendQuad(flatVerts, flatIndices, position, color, floatsPerVertex)
		}
	}
	vertexCount := len(flatVerts) / floatsPerVertex

	h.flatVertices = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, flatIndices)
	h.flatVertices.Begin()
	h.flatVertices.SetVertexData(flatVerts)
	h.flatVertices.End()
}
