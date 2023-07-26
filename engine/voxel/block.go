package voxel

type Block struct {
	ID           byte
	occupant    MapObject
}

func (b *Block) IsAir() bool {
	return b.ID == EMPTY
}

func (b *Block) GetTextureIndexForSide(side FaceType) byte {
	return b.ID - 1
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