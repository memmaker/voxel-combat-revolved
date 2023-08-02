package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type Action interface {
	GetValidTargets(unit UnitCore) []voxel.Int3
	Execute(unit UnitCore, target voxel.Int3)
	GetName() string
	IsValidTarget(unit UnitCore, target voxel.Int3) bool
}
