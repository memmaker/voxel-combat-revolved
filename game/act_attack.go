package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionAttack struct {
	engine *BattleGame
}

func (a *ActionAttack) GetName() string {
	return "Attack"
}

func (a *ActionAttack) GetValidTargets(unit *Unit) []voxel.Int3 {
	valid := make([]voxel.Int3, 0)
	for _, otherUnit := range a.engine.GetVisibleUnits(unit) {
		valid = append(valid, voxel.ToGridInt3(otherUnit.GetFootPosition()))
	}
	return valid
}

func (a *ActionAttack) Execute(unit *Unit, target voxel.Int3) {
	currentPos := voxel.ToGridInt3(unit.GetFootPosition())
	mapObjectAt := a.engine.voxelMap.GetMapObjectAt(target)
	if mapObjectAt == nil {
		println(fmt.Sprintf("[ActionAttack] ERR -> Attacking %s: from %s to %s (no target)", unit.GetName(), currentPos.ToString(), target.ToString()))
		return
	}
	otherUnit := mapObjectAt.(*Unit)
	println(fmt.Sprintf("[ActionAttack] Attacking %s: from %s to %s", unit.GetName(), currentPos.ToString(), target.ToString()))
	sourceOfProjectile := unit.GetEyePosition()
	directionVector := otherUnit.GetPosition().Sub(sourceOfProjectile).Normalize()
	velocity := directionVector.Mul(10)

	sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))

	a.engine.SpawnProjectile(sourceOffset, velocity)
}
