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
	totalAPCost  int
}

func (a *ServerActionShot) SetAPCost(newCost int) {
	a.totalAPCost = newCost
}

func (a *ServerActionShot) IsTurnEnding() bool {
	return true
}

func (a *ServerActionShot) IsValid() (bool, string) {
	// check if weapon is ready
	if !a.unit.Weapon.IsReady() {
		return false, "Weapon is not ready"
	}

	if a.unit.GetIntegerAP() < a.totalAPCost {
		return false, fmt.Sprintf("Not enough AP for shot. Need %d, have %d", a.totalAPCost, a.unit.GetIntegerAP())
	}

	return true, ""
}
func NewServerActionFreeShot(g *game.GameInstance, unit *game.UnitInstance, camPos mgl32.Vec3, targetAngles [][2]float32) *ServerActionShot {
	camera := util.NewFPSCamera(camPos, 100, 100)
	rayCalls := 0

	s := &ServerActionShot{
		engine:       g,
		unit:         unit,
		totalAPCost:  int(unit.GetWeapon().Definition.BaseAPForShot) + 1,
		aimDirection: camera.GetFront(),
	}
	s.createRay = func() (mgl32.Vec3, mgl32.Vec3) {
		targetAngle := targetAngles[rayCalls]

		camera.Reposition(camPos, targetAngle[0], targetAngle[1])
		s.aimDirection = camera.GetFront()

		startRay, endRay := camera.GetRandomRayInCircleFrustum(unit.GetFreeAimAccuracy())
		direction := endRay.Sub(startRay).Normalize()

		rayCalls = rayCalls + 1%len(targetAngles)
		return startRay, direction
	}
	return s
}
func NewServerActionSnapShot(g *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) ServerAction {
	camera := util.NewFPSCamera(unit.GetEyePosition(), 100, 100)
	targetsInWorld := make([]mgl32.Vec3, len(targets))
	for i, target := range targets {
		if !g.GetVoxelMap().IsOccupied(target) {
			return NewInvalidServerAction(fmt.Sprintf("SnapShot target %s is not occupied", target.ToString()))
		} else {
			targetsInWorld[i] = g.GetVoxelMap().GetMapObjectAt(target).(*game.UnitInstance).GetCenterOfMassPosition()
		}
	}

	rayCalls := 0

	s := &ServerActionShot{
		engine:       g,
		unit:         unit,
		totalAPCost:  int(unit.GetWeapon().Definition.BaseAPForShot),
		aimDirection: camera.GetFront(),
	}
	s.createRay = func() (mgl32.Vec3, mgl32.Vec3) {
		targetLocation := targetsInWorld[rayCalls]
		camera.FPSLookAt(targetLocation)
		s.aimDirection = camera.GetFront()

		startRay, endRay := camera.GetRandomRayInCircleFrustum(unit.GetFreeAimAccuracy())
		direction := endRay.Sub(startRay).Normalize()

		rayCalls = rayCalls + 1%len(targets)
		return startRay, direction
	}
	return s
}

func (a *ServerActionShot) Execute(mb *game.MessageBuffer) {
	currentPos := a.unit.GetBlockPosition()
	println(fmt.Sprintf("[ServerActionShot] %s(%d) fires a shot from %s.", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString()))

	var projectiles []game.VisualProjectile
	numberOfProjectiles := a.unit.Weapon.Definition.BulletsPerShot

	for i := uint(0); i < numberOfProjectiles; i++ {
		projectiles = append(projectiles, a.simulateOneProjectile())
	}

	ammoCost := uint(1)
	costOfAPForShot := int(a.unit.Weapon.Definition.BaseAPForShot) + a.totalAPCost
	a.unit.ConsumeAP(costOfAPForShot)
	a.unit.Weapon.ConsumeAmmo(ammoCost)

	newForward := util.DirectionToCardinalAim(a.aimDirection)
	a.unit.SetForward2DCardinal(newForward)

	mb.AddMessageForAll(game.VisualRangedAttack{
		Projectiles:       projectiles,
		WeaponType:        a.unit.Weapon.Definition.WeaponType,
		AmmoCost:          ammoCost,
		Attacker:          a.unit.UnitID(),
		APCostForAttacker: costOfAPForShot,
		AimDirection:      newForward,
		IsTurnEnding:      a.IsTurnEnding(),
	})
}

func (a *ServerActionShot) simulateOneProjectile() game.VisualProjectile {
	projectileBaseDamage := a.unit.Weapon.Definition.BaseDamagePerBullet
	lethal := false
	origin, direction := a.createRay()
	endOfRay := origin.Add(direction.Mul(float32(a.unit.Weapon.Definition.MaxRange)))

	rayHitInfo := a.engine.RayCastFreeAim(origin, endOfRay, a.unit)
	unitHidID := int64(-1)
	var hitUnit *game.UnitInstance = nil
	var hitBlocks []voxel.Int3
	if rayHitInfo.UnitHit != nil && rayHitInfo.UnitHit != hitUnit {
		unitHidID = int64(rayHitInfo.UnitHit.UnitID())
		println(fmt.Sprintf("[ServerActionShot] Unit was HIT %s(%d) -> %s", rayHitInfo.UnitHit.GetName(), unitHidID, rayHitInfo.BodyPart))
		hitUnit = rayHitInfo.UnitHit.(*game.UnitInstance)
		projectileBaseDamage, lethal = a.engine.HandleUnitHitWithProjectile(a.unit, rayHitInfo)
	} else {
		if rayHitInfo.Hit {
			blockPosHit := rayHitInfo.HitInfo3D.CollisionGridPosition
			blockDef := a.engine.GetBlockDefAt(blockPosHit)
			if blockDef.OnDamageReceived != nil {
				blockDef.OnDamageReceived(blockPosHit, projectileBaseDamage)
				hitBlocks = append(hitBlocks, blockPosHit)
				println(fmt.Sprintf("[ServerActionShot] HIT -> Block with on damage effect %s at %s", blockDef.UniqueName, blockPosHit.ToString()))
			} else {
				println(fmt.Sprintf("[ServerActionShot] MISS -> World Collision at %s hit %s", rayHitInfo.HitInfo3D.CollisionGridPosition.ToString(), blockDef.UniqueName))
			}
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
		Damage:      projectileBaseDamage,
		IsLethal:    lethal,
		BlocksHit:   hitBlocks,
	}
	return projectile
}
