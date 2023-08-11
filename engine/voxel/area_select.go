package voxel

import "github.com/memmaker/battleground/engine/glhf"

type AreaOfEffect interface {
	GetValidTargets(origin Int3) []Int3
	GetAffectedPositions(target Int3) []Int3
}
type HighlightMesh struct {
	ChunkMesh
}

func NewHighlightMesh(hlShader *glhf.Shader, chunkRelativeBlockPositions []Int3, textureIndex byte) *HighlightMesh {
	h := &HighlightMesh{
		ChunkMesh: NewMeshBuffer(),
	}
	for _, blockPos := range chunkRelativeBlockPositions {
		topRight := blockPos                         // min x & z
		bottomRight := blockPos.Add(Int3{Z: 1})      // min x, max z
		bottomLeft := blockPos.Add(Int3{X: 1, Z: 1}) // max x & z
		topLeft := blockPos.Add(Int3{X: 1})          // max x, min z
		h.AppendQuad(topRight, bottomRight, bottomLeft, topLeft, YP, textureIndex, [4]uint8{1, 1, 1, 1})
	}
	h.FlushMesh(hlShader)
	return h
}

func (h *HighlightMesh) Draw() {
	h.ChunkMesh.Draw()
}
