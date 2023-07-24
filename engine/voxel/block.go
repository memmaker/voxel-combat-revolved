package voxel

type Block struct {
	kind         int32
	textureIndex byte
	health       int32
}

func (b *Block) IsAir() bool {
	return b.kind == EMPTY
}

func (b *Block) GetTextureIndexForSide(side FaceType) byte {
	return b.textureIndex
}
