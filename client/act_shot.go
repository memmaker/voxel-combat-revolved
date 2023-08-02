package client

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ActionShot struct {
	engine *BattleGame
}

func (a *ActionShot) IsValidTarget(unit game.UnitCore, target voxel.Int3) bool {
	return true
}

func (a *ActionShot) GetName() string {
	return "Shot"
}

func (a *ActionShot) GetValidTargets(unit game.UnitCore) []voxel.Int3 {
	valid := make([]voxel.Int3, 0)
	for _, otherUnit := range a.engine.GetVisibleUnits(unit.(*Unit)) {
		valid = append(valid, voxel.ToGridInt3(otherUnit.GetFootPosition()))
	}
	return valid
}

func (a *ActionShot) Execute(unit game.UnitCore, target voxel.Int3) {
	currentPos := voxel.ToGridInt3(unit.GetFootPosition())
	mapObjectAt := a.engine.voxelMap.GetMapObjectAt(target)
	if mapObjectAt == nil {
		println(fmt.Sprintf("[ActionShot] ERR -> Attacking %s: from %s to %s (no target)", unit.GetName(), currentPos.ToString(), target.ToString()))
		return
	}
	otherUnit := mapObjectAt.(game.UnitCore)
	println(fmt.Sprintf("[ActionShot] Attacking %s: from %s to %s", unit.GetName(), currentPos.ToString(), target.ToString()))
	sourceOfProjectile := unit.GetEyePosition()
	directionVector := otherUnit.GetPosition().Sub(sourceOfProjectile).Normalize()
	velocity := directionVector.Mul(10)

	sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))

	a.engine.SpawnProjectile(sourceOffset, velocity)
}
