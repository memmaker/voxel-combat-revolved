package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

// for snap

type ServerActionShot struct {
	engine    *GameInstance
	unit      *game.UnitInstance
	createRay func() (mgl32.Vec3, mgl32.Vec3)
}

func (a *ServerActionShot) IsTurnEnding() bool {
	return true
}

func (a *ServerActionShot) IsValid() (bool, string) {
	// check if weapon is ready
	if !a.unit.Weapon.IsReady() {
		return false, "Weapon is not ready"
	}

	return true, ""
}
func NewServerActionSnapShot(engine *GameInstance, unit *game.UnitInstance, target voxel.Int3) *ServerActionShot {
	s := &ServerActionShot{
		engine: engine,
		unit:   unit,
		createRay: func() (mgl32.Vec3, mgl32.Vec3) {
			otherUnit := engine.voxelMap.GetMapObjectAt(target).(*game.UnitInstance)
			sourceOfProjectile := unit.GetEyePosition()
			directionVector := otherUnit.GetCenterOfMassPosition().Sub(sourceOfProjectile).Normalize()
			sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))
			return sourceOffset, directionVector
		},
	}
	return s
}
func NewServerActionFreeShot(engine *GameInstance, unit *game.UnitInstance, cam *util.FPSCamera) *ServerActionShot {
	s := &ServerActionShot{
		engine: engine,
		unit:   unit,
		createRay: func() (mgl32.Vec3, mgl32.Vec3) {
			startRay, endRay := cam.GetRandomRayInCircleFrustum(unit.GetFreeAimAccuracy())
			direction := endRay.Sub(startRay).Normalize()
			return startRay, direction
		},
	}
	return s
}
func (a *ServerActionShot) Execute(mb *game.MessageBuffer) {
	currentPos := voxel.ToGridInt3(a.unit.GetFootPosition())
	println(fmt.Sprintf("[ServerActionShot] %s(%d) fires a shot from %s.", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString()))

	var projectiles []game.VisualProjectile
	numberOfProjectiles := a.unit.Weapon.Definition.BulletsPerShot

	for i := 0; i < numberOfProjectiles; i++ {
		projectiles = append(projectiles, a.simulateOneProjectile())
	}

	ammoCost := 1
	a.unit.Weapon.ConsumeAmmo(ammoCost)

	mb.AddMessageForAll(game.VisualRangedAttack{
		Projectiles: projectiles,
		WeaponType:  a.unit.Weapon.Definition.WeaponType,
		AmmoCost:    ammoCost,
	})
}

func (a *ServerActionShot) simulateOneProjectile() game.VisualProjectile {
	projectileDamage := a.unit.Weapon.Definition.BaseDamagePerBullet
	lethal := false
	origin, direction := a.createRay()
	endOfRay := origin.Add(direction.Mul(float32(a.unit.Weapon.Definition.MaxRange)))

	rayHitInfo := a.engine.RayCastFreeAim(origin, endOfRay, a.unit)
	unitHidID := int64(-1)
	if rayHitInfo.UnitHit != nil {
		unitHidID = int64(rayHitInfo.UnitHit.UnitID())
		println(fmt.Sprintf("[ServerActionShot] Unit was HIT %s(%d) -> %s", rayHitInfo.UnitHit.GetName(), unitHidID, rayHitInfo.BodyPart))
		hitUnit := rayHitInfo.UnitHit.(*game.UnitInstance)
		lethal = hitUnit.ApplyDamage(projectileDamage, rayHitInfo.BodyPart)
		if lethal {
			a.engine.Kill(a.unit, rayHitInfo.UnitHit.(*game.UnitInstance))
		}
	} else {
		if rayHitInfo.Hit {
			println(fmt.Sprintf("[ServerActionShot] MISS -> World Collision at %s", rayHitInfo.HitInfo3D.PreviousGridPosition.ToString()))
		} else {
			println(fmt.Sprintf("[ServerActionShot] MISS -> No Collision"))
		}
	}

	projectile := game.VisualProjectile{
		Origin:      origin,
		Velocity:    direction.Mul(10),
		Destination: rayHitInfo.HitInfo3D.CollisionWorldPosition,
		UnitHit:     unitHidID,
		BodyPart:    rayHitInfo.BodyPart,
		Damage:      projectileDamage,
		IsLethal:    lethal,
	}
	return projectile
}
