package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionThrow struct {
	engine *GameInstance
	unit   *UnitInstance
	valid  map[voxel.Int3]bool
}

func (a *ActionThrow) IsTurnEnding() bool {
	return true
}

func NewActionThrow(engine *GameInstance, unit *UnitInstance) *ActionThrow {
	a := &ActionThrow{
		engine: engine,
		unit:   unit,
		valid:  make(map[voxel.Int3]bool),
	}
	a.updateValidTargets()
	return a
}

func (a *ActionThrow) IsValidTarget(target voxel.Int3) bool {
	value, exists := a.valid[target]
	return exists && value
}

func (a *ActionThrow) GetName() string {
	return "Throw"
}

func (a *ActionThrow) GetValidTargets() []voxel.Int3 {
	result := make([]voxel.Int3, 0, len(a.valid))
	for pos := range a.valid {
		result = append(result, pos)
	}
	return result
}

func (a *ActionThrow) updateValidTargets() {
	for _, otherUnit := range a.engine.GetVisibleEnemyUnits(a.unit.UnitID()) {
		a.valid[otherUnit.GetBlockPosition()] = true
	}
}
