package voxel

type Block struct {
	ID       byte
	occupant MapObject
}

func (b *Block) IsAir() bool {
	return b.ID == EMPTY
}

func (b *Block) RemoveUnit(unit MapObject) {
	if b.occupant.UnitID() == unit.UnitID() {
		b.occupant = nil
	}
}

func (b *Block) AddUnit(unit MapObject) {
	b.occupant = unit
}

func (b *Block) IsOccupied() bool {
	if b == nil {
		return false
	}
	var nilMapObject MapObject
	return b.occupant != nil && b.occupant != nilMapObject
}

func (b *Block) GetOccupant() MapObject {
	return b.occupant
}
