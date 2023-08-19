package game

import "github.com/memmaker/battleground/engine/voxel"

type Engine interface {
	GetVisibleUnits(instance uint64) []*UnitInstance
	GetVoxelMap() *voxel.Map
}
