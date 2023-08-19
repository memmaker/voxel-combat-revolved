package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionOverwatch struct {
	engine *GameInstance
}

func (a *ActionOverwatch) IsTurnEnding() bool {
	return true
}

func NewActionOverwatch(engine *GameInstance) *ActionOverwatch {
	return &ActionOverwatch{
		engine: engine,
	}
}

func (a *ActionOverwatch) IsValidTarget(unit *UnitInstance, target voxel.Int3) bool {
	return true
}

func (a *ActionOverwatch) GetName() string {
	return "Overwatch"
}

func (a *ActionOverwatch) GetValidTargets(unit *UnitInstance) []voxel.Int3 {
	footPosInt := unit.GetBlockPosition()
	origin := footPosInt.Add(voxel.Int3{Y: 2})
	radius := unit.GetWeapon().Definition.MaxRange
	if radius > 20 {
		radius = 20
	}
	return GetSphere(origin, radius, func(pos voxel.Int3) bool {
		placeable, _ := a.engine.GetVoxelMap().IsHumanoidPlaceable(pos)
		if !placeable {
			return false
		}
		canSeePos := a.engine.CanSeePos(unit, pos)
		if !canSeePos {
			return false
		}
		return true
	})
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
