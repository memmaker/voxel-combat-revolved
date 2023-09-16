package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionShot struct {
	engine           *game.GameInstance
	unit             *game.UnitInstance
	createRay        func() (mgl32.Vec3, mgl32.Vec3)
	lastAimDirection mgl32.Vec3
	totalAPCost      int
	accuracyModifier float64
	damageModifier   float64
}

func (a *ServerActionShot) GetUnit() *game.UnitInstance {
	return a.unit
}

func (a *ServerActionShot) GetAccuracyModifier() float64 {
	return a.accuracyModifier
}

func (a *ServerActionShot) SetAPCost(newCost int) {
	a.totalAPCost = newCost
}

func (a *ServerActionShot) IsTurnEnding() bool {
	return a.engine.GetRules().IsRangedAttackTurnEnding
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
	camera := util.NewFPSCamera(camPos, 100, 100, 0)
	rayCalls := 0
	// todo: add anti-cheat validation to camPos and angles here..
	s := &ServerActionShot{
		engine:           g,
		unit:             unit,
		totalAPCost:      int(unit.GetWeapon().Definition.BaseAPForShot) + 1,
		accuracyModifier: 1.0,
		damageModifier:   1.0,
	}
	s.createRay = func() (mgl32.Vec3, mgl32.Vec3) {
		targetAngle := targetAngles[rayCalls]

		camera.Reposition(camPos, targetAngle[0], targetAngle[1])
		util.LogServerUnitDebug(fmt.Sprintf("[ServerActionShot] %s(%d) fires a shot from (%0.2f, %0.2f, %0.2f) in direction %0.2f, %0.2f", unit.GetName(), unit.UnitID(), camPos.X(), camPos.Y(), camPos.Z(), targetAngle[0], targetAngle[1]))
		s.lastAimDirection = camera.GetForward()

		startRay, endRay := camera.GetRandomRayInCircleFrustum(s.finalShotAccuracy())
		direction := endRay.Sub(startRay).Normalize()

		rayCalls = rayCalls + 1%len(targetAngles)
		return startRay, direction
	}
	return s
}

func NewServerActionPerfectShot(g *game.GameInstance, unit *game.UnitInstance, rayStart, rayEnd mgl32.Vec3) *ServerActionShot {
	//camera := util.NewFPSCamera(camPos, 100, 100, 0)
	println(fmt.Sprintf("[ServerActionShot] %s(%d) fires a PERFECT shot from (%0.2f, %0.2f, %0.2f) to (%0.2f, %0.2f, %0.2f)", unit.GetName(), unit.UnitID(), rayStart.X(), rayStart.Y(), rayStart.Z(), rayEnd.X(), rayEnd.Y(), rayEnd.Z()))
	// todo: add anti-cheat validation to camPos and angles here..
	s := &ServerActionShot{
		engine:           g,
		unit:             unit,
		totalAPCost:      int(unit.GetWeapon().Definition.BaseAPForShot) + 1,
		accuracyModifier: 1.0,
		damageModifier:   1.0,
	}
	aimDirection := rayEnd.Sub(rayStart).Normalize()
	s.createRay = func() (mgl32.Vec3, mgl32.Vec3) {
		s.lastAimDirection = aimDirection
		return rayStart, aimDirection
	}
	return s
}
func NewServerActionSnapShot(g *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) *ServerActionShot {
	camera := util.NewFPSCamera(unit.GetEyePosition(), 100, 100, 0)
	targetsInWorld := make([]mgl32.Vec3, len(targets))
	for i, target := range targets {
		if g.GetVoxelMap().IsOccupied(target) {
			targetsInWorld[i] = g.GetVoxelMap().GetMapObjectAt(target).(*game.UnitInstance).GetCenterOfMassPosition()
		} else {
			targetsInWorld[i] = target.ToBlockCenterVec3D()
		}
	}

	rayCalls := 0

	s := &ServerActionShot{
		engine:           g,
		unit:             unit,
		totalAPCost:      int(unit.GetWeapon().Definition.BaseAPForShot),
		accuracyModifier: 1.0,
		damageModifier:   1.0,
	}
	s.createRay = func() (mgl32.Vec3, mgl32.Vec3) {
		targetLocation := targetsInWorld[rayCalls]
		camera.SetLookTarget(targetLocation)
		s.lastAimDirection = camera.GetForward()

		startRay, endRay := camera.GetRandomRayInCircleFrustum(s.finalShotAccuracy())
		direction := endRay.Sub(startRay).Normalize()

		rayCalls = rayCalls + 1%len(targets)
		return startRay, direction
	}
	return s
}

func (a *ServerActionShot) finalShotAccuracy() float64 {
	return a.engine.GetRules().GetShotAccuracy(a)
}

func (a *ServerActionShot) Execute(mb *game.MessageBuffer) {
	currentPos := a.unit.GetBlockPosition()
	println(fmt.Sprintf("[ServerActionShot] %s(%d) fires a shot from %s.", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString()))

	var projectiles []game.VisualProjectile
	numberOfProjectiles := a.unit.Weapon.Definition.BulletsPerShot

	for i := uint(0); i < numberOfProjectiles; i++ {
		projectile := a.simulateOneProjectile()
		projectiles = append(projectiles, projectile)
	}

	ammoCost := uint(1)
	costOfAPForShot := a.totalAPCost
	a.unit.ConsumeAP(costOfAPForShot)
	a.unit.Weapon.ConsumeAmmo(ammoCost)

	lastAimDir := a.lastAimDirection
	a.unit.SetForward(voxel.DirectionToGridInt3(lastAimDir))
	a.unit.UpdateMapPosition()

	mb.AddMessageForAll(game.VisualRangedAttack{
		Projectiles:       projectiles,
		WeaponType:        a.unit.Weapon.Definition.WeaponType,
		AmmoCost:          ammoCost,
		Attacker:          a.unit.UnitID(),
		APCostForAttacker: costOfAPForShot,
		AimDirection:      lastAimDir,
		IsTurnEnding:      a.IsTurnEnding(),
	})
}

func (a *ServerActionShot) simulateOneProjectile() game.VisualProjectile {
	projectileBaseDamage := a.unit.Weapon.Definition.BaseDamagePerBullet
	lethal := false
	origin, direction := a.createRay()
	endOfRay := origin.Add(direction.Mul(float32(a.unit.Weapon.Definition.MaxRange)))

	rayHitInfo := a.engine.RayCastFreeAim(origin, endOfRay, a.unit)
	unitHitID := int64(-1)
	var hitBlocks []voxel.Int3
	projectileDestination := rayHitInfo.HitInfo3D.CollisionWorldPosition

	if rayHitInfo.HitUnit() {
		unitHitID = int64(rayHitInfo.UnitHit.UnitID())
		util.LogServerUnitDebug(fmt.Sprintf("[ServerActionShot] Unit was HIT %s(%d) -> %s", rayHitInfo.UnitHit.GetName(), unitHitID, rayHitInfo.BodyPart))
		projectileBaseDamage, lethal = a.engine.HandleUnitHitWithProjectile(a.unit, a.damageModifier, rayHitInfo)
	} else {
		if rayHitInfo.Hit {
			blockPosHit := rayHitInfo.HitInfo3D.CollisionGridPosition
			blockDef := a.engine.GetBlockDefAt(blockPosHit)
			if blockDef.OnDamageReceived != nil {
				blockDef.OnDamageReceived(blockPosHit, projectileBaseDamage)
				hitBlocks = append(hitBlocks, blockPosHit)
				util.LogServerUnitDebug(fmt.Sprintf("[ServerActionShot] HIT -> Block with on damage effect %s at %s", blockDef.UniqueName, blockPosHit.ToString()))
			} else {
				util.LogServerUnitDebug(fmt.Sprintf("[ServerActionShot] MISS -> World Collision at %s hit %s", rayHitInfo.HitInfo3D.CollisionGridPosition.ToString(), blockDef.UniqueName))
			}
		} else {
			util.LogServerUnitDebug(fmt.Sprintf("[ServerActionShot] MISS -> No Collision, out of weapon range"))
			projectileDestination = endOfRay
		}
	}

	projectile := game.VisualProjectile{
		Origin:        origin,
		Velocity:      direction.Mul(2),
		Destination:   projectileDestination,
		UnitHit:       unitHitID,
		BodyPart:      rayHitInfo.BodyPart,
		Damage:        projectileBaseDamage,
		IsLethal:      lethal,
		BlocksHit:     hitBlocks,
		VisitedBlocks: rayHitInfo.VisitedBlocks,
	}
	return projectile
}

func (a *ServerActionShot) SetAccuracyModifier(factor float64) {
	a.accuracyModifier = factor
}

func (a *ServerActionShot) SetDamageModifier(factor float64) {
	a.damageModifier = factor
}
