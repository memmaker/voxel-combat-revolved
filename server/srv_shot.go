package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

// for snap

type ServerActionShot struct {
	engine   *GameInstance
	unit     *game.UnitInstance
	origin   mgl32.Vec3
	velocity mgl32.Vec3
}

func (a *ServerActionShot) IsValid() bool {
	return true
}
func NewServerActionSnapShot(engine *GameInstance, unit *game.UnitInstance, target voxel.Int3) *ServerActionShot {
	otherUnit := engine.voxelMap.GetMapObjectAt(target).(*game.UnitInstance)
	sourceOfProjectile := unit.GetEyePosition()
	directionVector := otherUnit.GetPosition().Sub(sourceOfProjectile).Normalize()
	sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))
	velocity := directionVector.Mul(10)
	return &ServerActionShot{
		engine:   engine,
		unit:     unit,
		origin:   sourceOffset,
		velocity: velocity,
	}
}
func NewServerActionFreeShot(engine *GameInstance, unit *game.UnitInstance, origin mgl32.Vec3, velocity mgl32.Vec3) *ServerActionShot {
	return &ServerActionShot{
		engine:   engine,
		unit:     unit,
		origin:   origin,
		velocity: velocity,
	}
}
func (a *ServerActionShot) Execute(mb *game.MessageBuffer) {
	currentPos := voxel.ToGridInt3(a.unit.GetFootPosition())
	println(fmt.Sprintf("[ServerActionShot] Attacking %s: from %s. Dir.: %v", a.unit.GetName(), currentPos.ToString(), a.velocity))
	//a.engine.SpawnProjectile(sourceOffset, velocity)
	rayHitInfo := a.engine.RayCastFreeAim(a.origin, a.origin.Add(a.velocity.Normalize().Mul(100)), a.unit)
	unitHidID := int64(-1)
	if rayHitInfo.UnitHit != nil {
		unitHidID = int64(rayHitInfo.UnitHit.UnitID())
	}
	mb.AddMessageForAll(game.VisualProjectileFired{
		Origin:      a.origin,
		Destination: rayHitInfo.HitInfo3D.CollisionWorldPosition,
		Velocity:    a.velocity,
		UnitHit:     unitHidID,
		BodyPart:    rayHitInfo.BodyPart,
	})
}
