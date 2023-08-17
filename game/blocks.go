package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type BlockDefinition struct {
	BlockID                byte
	UniqueName             string
	TextureIndicesForFaces map[voxel.FaceType]byte
}

type BlockLibrary struct {
	blocks   map[byte]*BlockDefinition
	nameToId map[string]byte
}

// TODO: Use texture atlas data to fill the library
func NewBlockLibrary(blockNames []string, indexMap map[string]byte) *BlockLibrary {
	b := &BlockLibrary{
		nameToId: make(map[string]byte),
		blocks: map[byte]*BlockDefinition{
			0: &BlockDefinition{
				BlockID:                0,
				UniqueName:             "air",
				TextureIndicesForFaces: map[voxel.FaceType]byte{},
			},
		},
	}

	b.loadFromIndexMap(blockNames, indexMap)
	return b
}
func (b *BlockLibrary) LastBlockID() byte {
	return byte(len(b.blocks) - 1)
}
func (b *BlockLibrary) GetTextureIndexForFaces(block *voxel.Block, side voxel.FaceType) byte {
	if block == nil {
		return 0
	}
	if block.IsAir() {
		return 0
	}
	blockDefinition := b.blocks[block.ID]
	if blockDefinition == nil {
		return 0
	}
	return blockDefinition.TextureIndicesForFaces[side]
}

func (b *BlockLibrary) loadFromIndexMap(blockNames []string, indexMap map[string]byte) {
	if len(blockNames) > 254 {
		panic("Too many blocks")
	}
	for index, name := range blockNames {
		byteIndex := byte(index + 1) // block 0 is air
		b.AddBlockDefinition(byteIndex, name, getFaceMapForBlock(name, indexMap))
	}
}

func (b *BlockLibrary) AddBlockDefinition(blockID byte, name string, indexMap map[voxel.FaceType]byte) {
	if _, exists := b.blocks[blockID]; exists {
		panic("Block already exists")
	}
	b.blocks[blockID] = &BlockDefinition{
		BlockID:                blockID,
		UniqueName:             name,
		TextureIndicesForFaces: indexMap,
	}
	b.nameToId[name] = blockID
}

func (b *BlockLibrary) NewBlockFromName(name string) *voxel.Block {
	if blockID, exists := b.nameToId[name]; exists {
		return voxel.NewBlock(blockID)
	}
	println(fmt.Sprintf("[BlockLibrary] Unknown block name: %s", name))
	return voxel.NewBlock(0)
}

func (b *BlockLibrary) GetTextureIndexByName(name string) byte {
	if blockID, exists := b.nameToId[name]; exists {
		return b.blocks[blockID].TextureIndicesForFaces[voxel.Top]
	}
	return 0
}

func (b *BlockLibrary) GetBlockDefinition(blockID byte) *BlockDefinition {
	return b.blocks[blockID]
}
func getFaceMapForBlock(blockName string, indexMap map[string]byte) map[voxel.FaceType]byte {
	result := make(map[voxel.FaceType]byte)
	result[voxel.North] = util.MapFaceToTextureIndex(blockName, voxel.North, indexMap)
	result[voxel.South] = util.MapFaceToTextureIndex(blockName, voxel.South, indexMap)
	result[voxel.East] = util.MapFaceToTextureIndex(blockName, voxel.East, indexMap)
	result[voxel.West] = util.MapFaceToTextureIndex(blockName, voxel.West, indexMap)
	result[voxel.Top] = util.MapFaceToTextureIndex(blockName, voxel.Top, indexMap)
	result[voxel.Bottom] = util.MapFaceToTextureIndex(blockName, voxel.Bottom, indexMap)
	return result
}
