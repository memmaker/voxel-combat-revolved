package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

func (g *GameInstance) ExecuteOnServer(unit *game.UnitInstance, target voxel.Int3) ([]string, []any) {
	currentPos := voxel.ToGridInt3(unit.GetFootPosition())
	mapObjectAt := g.voxelMap.GetMapObjectAt(target)
	if mapObjectAt == nil {
		println(fmt.Sprintf("[ActionShot] ERR -> Attacking %s: from %s to %s (no target)", unit.GetName(), currentPos.ToString(), target.ToString()))
		return nil, nil
	}
	otherUnit := mapObjectAt.(*game.UnitInstance)
	println(fmt.Sprintf("[ActionShot] Attacking %s: from %s to %s", unit.GetName(), currentPos.ToString(), target.ToString()))
	sourceOfProjectile := unit.GetEyePosition()
	directionVector := otherUnit.GetPosition().Sub(sourceOfProjectile).Normalize()
	velocity := directionVector.Mul(10)

	sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))

	//a.engine.SpawnProjectile(sourceOffset, velocity)
	return []string{"ProjectileFired"}, []any{game.VisualProjectileFired{
		SourcePosition: sourceOffset,
		Velocity:       velocity,
	}}
}
