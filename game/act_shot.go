package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionShot struct {
	engine Engine
}

func NewActionShot(engine Engine) *ActionShot {
	return &ActionShot{
		engine: engine,
	}
}

func (a *ActionShot) IsValidTarget(unit UnitCore, target voxel.Int3) bool {
	return true
}

func (a *ActionShot) GetName() string {
	return "Shot"
}

func (a *ActionShot) GetValidTargets(unit UnitCore) []voxel.Int3 {
	valid := make([]voxel.Int3, 0)
	for _, otherUnit := range a.engine.GetVisibleUnits(unit.UnitID()) {
		valid = append(valid, voxel.ToGridInt3(otherUnit.GetFootPosition()))
	}
	return valid
}
