package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

// for snap

type ServerActionShot struct {
	engine     *GameInstance
	unit       *game.UnitInstance
	origin     mgl32.Vec3
	velocity   mgl32.Vec3
	isCheating bool
}

func (a *ServerActionShot) IsValid() (bool, string) {
	return !a.isCheating, "Cheating detected"
}
func NewServerActionSnapShot(engine *GameInstance, unit *game.UnitInstance, target voxel.Int3) *ServerActionShot {
	otherUnit := engine.voxelMap.GetMapObjectAt(target).(*game.UnitInstance)
	sourceOfProjectile := unit.GetEyePosition()
	directionVector := otherUnit.GetPosition().Sub(sourceOfProjectile).Normalize()
	sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))
	velocity := directionVector.Mul(10)
	s := &ServerActionShot{
		engine:   engine,
		unit:     unit,
		origin:   sourceOffset,
		velocity: velocity,
	}
	s.checkForCheating()
	return s
}
func NewServerActionFreeShot(engine *GameInstance, unit *game.UnitInstance, origin mgl32.Vec3, velocity mgl32.Vec3) *ServerActionShot {

	s := &ServerActionShot{
		engine:   engine,
		unit:     unit,
		origin:   origin,
		velocity: velocity,
	}
	s.checkForCheating()
	return s
}
func (a *ServerActionShot) Execute(mb *game.MessageBuffer) {
	currentPos := voxel.ToGridInt3(a.unit.GetFootPosition())
	println(fmt.Sprintf("[ServerActionShot] %s(%d) fires a shot from %s. dir.: %v", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString(), a.velocity))
	//a.engine.SpawnProjectile(sourceOffset, velocity)
	rayHitInfo := a.engine.RayCastFreeAim(a.origin, a.origin.Add(a.velocity.Normalize().Mul(100)), a.unit)
	unitHidID := int64(-1)
	if rayHitInfo.UnitHit != nil {
		unitHidID = int64(rayHitInfo.UnitHit.UnitID())
		println(fmt.Sprintf("[ServerActionShot] Unit was HIT %s(%d) -> %s", rayHitInfo.UnitHit.GetName(), unitHidID, rayHitInfo.BodyPart))
		// TODO: Apply damage on server.. eg. kill and remove unit from map
	} else {
		if rayHitInfo.Hit {
			println(fmt.Sprintf("[ServerActionShot] MISS -> World Collision at %s", rayHitInfo.HitInfo3D.PreviousGridPosition.ToString()))
		} else {
			println(fmt.Sprintf("[ServerActionShot] MISS -> No Collision"))
		}
	}

	mb.AddMessageForAll(game.VisualProjectileFired{
		Origin:      a.origin,
		Destination: rayHitInfo.HitInfo3D.CollisionWorldPosition,
		Velocity:    a.velocity,
		UnitHit:     unitHidID,
		BodyPart:    rayHitInfo.BodyPart,
		Damage:      1,
		IsLethal:    true,
	})
}

func (a *ServerActionShot) checkForCheating() {
	// check distance of origin and unit eye position
	dist := a.origin.Sub(a.unit.GetEyePosition()).Len()
	if dist > 1.0 {
		a.isCheating = true
	}
	// check if velocity is not too high
}
