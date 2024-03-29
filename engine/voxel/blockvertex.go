package voxel

import "github.com/memmaker/battleground/engine/glhf"

type FaceType int32

const (
	East FaceType = iota
	West
	Top
	Bottom
	South
	North
)

var NorthDir = Int3{0, 0, -1}
var SouthDir = Int3{0, 0, 1}
var EastDir = Int3{1, 0, 0}
var WestDir = Int3{-1, 0, 0}
var Up = Int3{0, 1, 0}
var Down = Int3{0, -1, 0}

type ChunkMesh interface {
	AppendQuad(tr, br, bl, tl Int3, normal FaceType, textureIndex byte, extraBits byte)
	Reset()
	Draw()
	UploadMeshToGPU(shader *glhf.Shader)
	TriangleCount() int
	MergeBuffer(buffer ChunkMesh)
}

type BlockFactory struct {
	KnownBlocks    map[string]*Block
	UnknownBlocks  map[string]bool
	TextureIndices map[string]byte
}

func NewBlockFactory(indices map[string]byte) *BlockFactory {
	return &BlockFactory{
		KnownBlocks: map[string]*Block{
			"air": NewAirBlock(),
		},
		UnknownBlocks:  map[string]bool{},
		TextureIndices: indices,
	}
}

func (f *BlockFactory) GetBlockByName(name string) *Block {
	if block, exists := f.KnownBlocks[name]; exists {
		return block
	} else if textureIndex, textureExists := f.TextureIndices[name]; textureExists {
		f.KnownBlocks[name] = NewBlock(textureIndex)
		return f.KnownBlocks[name]
	} else {
		f.UnknownBlocks[name] = true
		return NewBlock(0)
	}
}

func NewBlock(id byte) *Block {
	return &Block{
		ID: id,
	}
}
func NewAirBlock() *Block {
	return &Block{
		ID: 0,
	}
}

// CalculateCornerUVsTerrain returns the UV coordinates for the corners of a tile in a texture atlas
// in this order: top left, top right, bottom right, bottom left
func CalculateCornerUVsTerrain(tileSize, atlasWidth, atlasHeight int, textureIndex int) [4][2]float32 {
	// we need the width height of the whole atlas in pixels
	// we need the width height of each tile in pixels
	// we need the texture textureIndex

	result := [4][2]float32{}

	tileCountX := atlasWidth / tileSize
	//tileCountY := atlasHeight / tileSize

	tileX := textureIndex % tileCountX
	tileY := textureIndex / tileCountX

	// top left
	result[0][0] = float32(tileX*tileSize) / float32(atlasWidth)
	result[0][1] = float32(tileY*tileSize) / float32(atlasHeight)

	// top right
	result[1][0] = float32((tileX+1)*tileSize) / float32(atlasWidth)
	result[1][1] = float32(tileY*tileSize) / float32(atlasHeight)

	// bottom right
	result[2][0] = float32((tileX+1)*tileSize) / float32(atlasWidth)
	result[2][1] = float32((tileY+1)*tileSize) / float32(atlasHeight)

	// bottom left
	result[3][0] = float32(tileX*tileSize) / float32(atlasWidth)
	result[3][1] = float32((tileY+1)*tileSize) / float32(atlasHeight)

	return result
}
