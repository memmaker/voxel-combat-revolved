package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionOverwatch struct {
	engine *GameInstance
	valid  map[voxel.Int3]bool
	unit   *UnitInstance
}

func (a *ActionOverwatch) IsTurnEnding() bool {
	return true
}

func NewActionOverwatch(engine *GameInstance, unit *UnitInstance) *ActionOverwatch {
	a := &ActionOverwatch{
		engine: engine,
		valid:  make(map[voxel.Int3]bool),
		unit:   unit,
	}
	a.updateValidTargets()
	return a
}

func (a *ActionOverwatch) IsValidTarget(target voxel.Int3) bool {
	value, exists := a.valid[target]
	return exists && value
}

func (a *ActionOverwatch) GetName() string {
	return "Overwatch"
}

func (a *ActionOverwatch) GetValidTargets() []voxel.Int3 {
	result := make([]voxel.Int3, 0, len(a.valid))
	for pos := range a.valid {
		result = append(result, pos)
	}
	return result
}

func (a *ActionOverwatch) updateValidTargets() {
	footPosInt := a.unit.GetBlockPosition()
	origin := footPosInt.Add(voxel.Int3{Y: 2})

	radius := a.unit.GetWeapon().Definition.MaxRange

	if radius > a.engine.rules.MaxOverwatchRange {
		radius = a.engine.rules.MaxOverwatchRange
	}
	for _, pos := range GetSphere(origin, radius, func(pos voxel.Int3) bool {
		placeable, _ := a.engine.GetVoxelMap().IsHumanoidPlaceable(pos)
		if !placeable {
			return false
		}
		canSeePos := a.engine.CanSeePos(a.unit, pos)
		if !canSeePos {
			return false
		}
		return true
	}) {
		a.valid[pos] = true
	}
}

func GetSphere(origin voxel.Int3, radius uint, keep func(pos voxel.Int3) bool) []voxel.Int3 {
	var valid []voxel.Int3
	for x := -int(radius); x <= int(radius); x++ {
		for y := -int(radius); y <= int(radius); y++ {
			for z := -int(radius); z <= int(radius); z++ {
				pos := voxel.Int3{X: int32(x), Y: int32(y), Z: int32(z)}
				if pos.Length() <= int32(radius) {
					if keep(origin.Add(pos)) {
						valid = append(valid, origin.Add(pos))
					}
				}
			}
		}
	}
	return valid
}
