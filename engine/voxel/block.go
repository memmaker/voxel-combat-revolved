package voxel

type Block struct {
	ID       byte
	occupant MapObject
	lightLevel byte
}

const EMPTYBLOCK = 0

func (b *Block) SetTorchLight(lightLevel byte) {
	b.lightLevel = (b.lightLevel & 0xF0) | lightLevel
}

func (b *Block) GetTorchLight() byte {
	if b == nil {
		return 0
	}
	return b.lightLevel & 0xF
}
func (b *Block) SetSunLight(lightLevel byte) {
	b.lightLevel = (b.lightLevel & 0xF) | (lightLevel << 4)
}

func (b *Block) GetSunLight() byte {
	if b == nil {
		return 0
	}
	return b.lightLevel >> 4
}

func (b *Block) IsAir() bool {
	return b.ID == EMPTYBLOCK
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
