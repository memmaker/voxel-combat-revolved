package game

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionThrow struct {
	engine *GameInstance
	unit   *UnitInstance
	valid  map[voxel.Int3]bool
	item   *Item
}

func (a *ActionThrow) IsTurnEnding() bool {
	return true
}

func NewActionThrow(engine *GameInstance, unit *UnitInstance, item *Item) *ActionThrow {
	a := &ActionThrow{
		engine: engine,
		unit:   unit,
		item: item,
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

func (a *ActionThrow) GetTrajectory(target mgl32.Vec3) []mgl32.Vec3 {
    sourcePos := a.unit.GetEyePosition()
    maxVelocity := a.unit.Definition.CoreStats.ThrowVelocity
    gravity := 9.8
    return util.CalculateTrajectory(sourcePos, target, maxVelocity, gravity)
}

func (a *ActionThrow) GetItemName() string {
	return a.item.Definition.UniqueName
}

