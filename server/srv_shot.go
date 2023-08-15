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
	engine       *game.GameInstance
	unit         *game.UnitInstance
	createRay    func() (mgl32.Vec3, mgl32.Vec3)
	aimDirection mgl32.Vec3
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
func NewServerActionFreeShot(g *game.GameInstance, unit *game.UnitInstance, camPos mgl32.Vec3, camRotX, camRotY float32) *ServerActionShot {
	camera := util.NewFPSCamera(camPos, 100, 100)
	camera.Reposition(camPos, camRotX, camRotY)
	return newServerActionShot(g, unit, camera)
}
func NewServerActionSnapShot(g *game.GameInstance, unit *game.UnitInstance, target voxel.Int3) ServerAction {
	camera := util.NewFPSCamera(unit.GetEyePosition(), 100, 100)
	if !g.GetVoxelMap().IsOccupied(target) {
		return NewInvalidServerAction(fmt.Sprintf("SnapShot target %s is not occupied", target.ToString()))
	}
	targetUnit := g.GetVoxelMap().GetMapObjectAt(target).(*game.UnitInstance)
	if targetUnit != nil {
		camera.FPSLookAt(targetUnit.GetCenterOfMassPosition())
	}
	return newServerActionShot(g, unit, camera)
}
func newServerActionShot(engine *game.GameInstance, unit *game.UnitInstance, cam *util.FPSCamera) *ServerActionShot {
	s := &ServerActionShot{
		engine:       engine,
		unit:         unit,
		aimDirection: cam.GetFront(),
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

	for i := uint(0); i < numberOfProjectiles; i++ {
		projectiles = append(projectiles, a.simulateOneProjectile())
	}

	ammoCost := uint(1)
	a.unit.Weapon.ConsumeAmmo(ammoCost)

	newForward := util.DirectionToCardinalAim(a.aimDirection)
	a.unit.SetForward(newForward)

	mb.AddMessageForAll(game.VisualRangedAttack{
		Projectiles:  projectiles,
		WeaponType:   a.unit.Weapon.Definition.WeaponType,
		AmmoCost:     ammoCost,
		Attacker:     a.unit.UnitID(),
		AimDirection: newForward,
	})
}

func (a *ServerActionShot) simulateOneProjectile() game.VisualProjectile {
	projectileDamage := a.unit.Weapon.Definition.BaseDamagePerBullet
	lethal := false
	origin, direction := a.createRay()
	endOfRay := origin.Add(direction.Mul(float32(a.unit.Weapon.Definition.MaxRange)))

	rayHitInfo := a.engine.RayCastFreeAim(origin, endOfRay, a.unit)
	unitHidID := int64(-1)
	var hitUnit *game.UnitInstance = nil
	if rayHitInfo.UnitHit != nil && rayHitInfo.UnitHit != hitUnit {
		unitHidID = int64(rayHitInfo.UnitHit.UnitID())
		println(fmt.Sprintf("[ServerActionShot] Unit was HIT %s(%d) -> %s", rayHitInfo.UnitHit.GetName(), unitHidID, rayHitInfo.BodyPart))
		hitUnit = rayHitInfo.UnitHit.(*game.UnitInstance)
		lethal = a.engine.ApplyDamage(a.unit, hitUnit, projectileDamage, rayHitInfo.BodyPart)
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
