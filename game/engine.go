package game

import "github.com/memmaker/battleground/engine/voxel"

type Engine interface {
	GetVisibleUnits(instance UnitCore) []UnitCore
	GetVoxelMap() *voxel.Map
}
