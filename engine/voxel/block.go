package voxel

type Block struct {
	kind         int32
	textureIndex byte
	health       int32
	occupant    MapObject
}

func (b *Block) IsAir() bool {
	return b.kind == EMPTY
}

func (b *Block) GetTextureIndexForSide(side FaceType) byte {
	return b.textureIndex
}

func (b *Block) RemoveUnit(unit MapObject) {
	if b.occupant == unit {
		b.occupant = nil
	}
}

func (b *Block) AddUnit(unit MapObject) {
	b.occupant = unit
}

func (b *Block) IsOccupied() bool {
	return b.occupant != nil
}

func (b *Block) GetOccupant() MapObject {
	return b.occupant
}