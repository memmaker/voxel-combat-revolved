package voxel

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

// idea:
// decals are basically the same as my highlights, just being able to be attached to any face

type Decal struct {
	Side          FaceType
	WorldPosition mgl32.Vec3
}

// TODO: we should add support for frustum culling here (also to the particles..)
type Decals struct {
	texture      *glhf.Texture // for now just one texture for all decals
	flatVertices *glhf.VertexSlice[glhf.GlFloat]
	allDecals    map[Int3][]Decal
	shader       *glhf.Shader
	isHidden     bool
}

func NewDecals(shader *glhf.Shader) *Decals {
	return &Decals{
		shader:    shader,
		allDecals: make(map[Int3][]Decal),
	}
}

/*
func (h *Decals) ShowAsFlat(category Highlight) {
	h.isHidden = false

	allVertices := make([]glhf.GlFloat, 0)
	allIndices := make([]uint32, 0)
	floatsPerVertex := h.shader.VertexFormat().Size() / 4 // = 11 -> 3 for position, 2 for texture coords, 3 for color, 3 for normal

	for position, decals := range h.allDecals {
		for _, decal := range decals {
			allVertices, allIndices = h.appendQuad(allVertices, allIndices, position, decal.Side, floatsPerVertex)
		}
	}

	vertexCount := len(allVertices) / floatsPerVertex

	h.flatVertices = glhf.MakeIndexedVertexSlice(h.shader, vertexCount, vertexCount, allIndices)
	h.flatVertices.Begin()
	h.flatVertices.SetVertexData(allVertices)
	h.flatVertices.End()
}

*/

func (h *Decals) appendQuad(allVertices []glhf.GlFloat, allIndices []uint32, position Int3, normal mgl32.Vec3, floatsPerVertex int) ([]glhf.GlFloat, []uint32) {
	// we need a quad that is 1x1 in size and parallel to the xz plane
	unitTopQuadNormal := mgl32.Vec3{0, 1, 0}

	color := mgl32.Vec3{1.0, 1.0, 1.0}

	rotation := mgl32.QuatBetweenVectors(unitTopQuadNormal, normal).Mat4()

	transl := mgl32.Translate3D(float32(position.X)+0.5, float32(position.Y)-0.5, float32(position.Z)+0.5)

	planeOffset := float32(0.05)
	unitTopQuad := []mgl32.Vec3{
		// top-left
		mgl32.Vec3{-0.5, 0.5 + planeOffset, -0.5},
		// top-right
		mgl32.Vec3{0.5, 0.5 + planeOffset, -0.5},
		// bottom-right
		mgl32.Vec3{0.5, 0.5 + planeOffset, 0.5},
		// bottom-left
		mgl32.Vec3{-0.5, 0.5 + planeOffset, 0.5},
	}
	for i := 0; i < len(unitTopQuad); i++ {
		rotated := rotation.Mul4x1(unitTopQuad[i].Vec4(1))
		translated := transl.Mul4x1(rotated)
		unitTopQuad[i] = translated.Vec3()
	}

	topLeft := unitTopQuad[0]
	topRight := unitTopQuad[1]
	bottomRight := unitTopQuad[2]
	bottomLeft := unitTopQuad[3]

	startVertexIndex := uint32(len(allVertices) / floatsPerVertex)
	allVertices = append(allVertices,
		[]glhf.GlFloat{
			// positions (x,y,z=0)   // texture coords (0..1,0..1) // color (1,1,1) // normal (0,0,1)
			//tl
			glhf.GlFloat(topLeft.X()), glhf.GlFloat(topLeft.Y()), glhf.GlFloat(topLeft.Z()), 0.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//tr
			glhf.GlFloat(topRight.X()), glhf.GlFloat(topRight.Y()), glhf.GlFloat(topRight.Z()), 1.0, 0.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//br
			glhf.GlFloat(bottomRight.X()), glhf.GlFloat(bottomRight.Y()), glhf.GlFloat(bottomRight.Z()), 1.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),

			//bl
			glhf.GlFloat(bottomLeft.X()), glhf.GlFloat(bottomLeft.Y()), glhf.GlFloat(bottomLeft.Z()), 0.0, 1.0, glhf.GlFloat(color.X()), glhf.GlFloat(color.Y()), glhf.GlFloat(color.Z()), glhf.GlFloat(normal.X()), glhf.GlFloat(normal.Y()), glhf.GlFloat(normal.Z()),
		}...)
	allIndices = append(allIndices, []uint32{
		// first triangle
		startVertexIndex + 1, // top right
		startVertexIndex + 0, // top left
		startVertexIndex + 2, // bottom right
		// second triangle
		startVertexIndex + 2, // bottom right
		startVertexIndex + 0, // top left
		startVertexIndex + 3, // bottom left
	}...)

	return allVertices, allIndices
}

func (h *Decals) GetTintColor() mgl32.Vec4 {
	return mgl32.Vec4{1.0, 1.0, 1.0, 0.3}
}

func (h *Decals) Draw(uniformForDrawMode int, fancyQuadDrawMode int32) {
	if h.isHidden {
		return
	}
	if h.flatVertices != nil && h.flatVertices.Len() > 0 {
		h.texture.Begin()
		h.flatVertices.Begin()
		h.flatVertices.Draw()
		h.flatVertices.End()
		h.texture.End()
	}
}
func (h *Decals) Hide() {
	h.isHidden = true
}
