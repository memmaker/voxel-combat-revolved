package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionSnapShot struct {
	engine *GameInstance
	unit   *UnitInstance
	valid  map[voxel.Int3]bool
}

func (a *ActionSnapShot) IsTurnEnding() bool {
	return true
}

func NewActionShot(engine *GameInstance, unit *UnitInstance) *ActionSnapShot {
	a := &ActionSnapShot{
		engine: engine,
		unit:   unit,
		valid:  make(map[voxel.Int3]bool),
	}
	a.updateValidTargets()
	return a
}

func (a *ActionSnapShot) IsValidTarget(target voxel.Int3) bool {
	value, exists := a.valid[target]
	return exists && value
}

func (a *ActionSnapShot) GetName() string {
	return "Shot"
}

func (a *ActionSnapShot) GetValidTargets() []voxel.Int3 {
	result := make([]voxel.Int3, 0, len(a.valid))
	for pos := range a.valid {
		result = append(result, pos)
	}
	return result
}

func (a *ActionSnapShot) updateValidTargets() {
	for _, otherUnit := range a.engine.GetVisibleEnemyUnits(a.unit.UnitID()) {
		a.valid[otherUnit.GetBlockPosition()] = true
	}
}
