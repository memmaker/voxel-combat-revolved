package game

import "github.com/memmaker/battleground/engine/voxel"

const PositionalTolerance = 0.08


type Engine interface {
	GetVisibleUnits(instance uint64) []*UnitInstance
	GetVoxelMap() *voxel.Map
}
