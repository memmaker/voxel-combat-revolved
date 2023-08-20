package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateFreeAim struct {
	engine         *BattleClient
	selectedUnit   *Unit
	validTargets   []voxel.Int3
	lastMouseX     float64
	lastMouseY     float64
	selectedAction game.TargetAction
	lockedTarget   int
	visibleEnemies []*game.UnitInstance
}

func (g *GameStateFreeAim) GetUnit() *game.UnitInstance {
	return g.selectedUnit.UnitInstance
}

func (g *GameStateFreeAim) GetAccuracyModifier() float64 {
	return 1.0
}

func (g *GameStateFreeAim) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateFreeAim) OnServerMessage(msgType string, json string) {

}

func (g *GameStateFreeAim) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		g.engine.fpsCamera.ChangeFOV(1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	} else {
		g.engine.fpsCamera.ChangeFOV(-1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	}
}

func (g *GameStateFreeAim) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyEscape {
		g.engine.SwitchToIsoCamera()
		g.engine.PopState()
		g.engine.SwitchToUnitNoCameraMovement(g.selectedUnit)
	} else if key == glfw.KeyM {
		g.engine.fpsCamera.ChangeFOV(1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	} else if key == glfw.KeyN {
		g.engine.fpsCamera.ChangeFOV(-1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	} else if key == glfw.KeyJ {
		g.engine.fpsCamera.ResetFOV()
	} else if key == glfw.KeyTab {
		g.aimAtNextTarget()
	}
}
func (g *GameStateFreeAim) aimAtNextTarget() {
	if len(g.visibleEnemies) == 0 {
		g.engine.fpsCamera.FPSLookAt(g.selectedUnit.GetEyePosition().Add(g.selectedUnit.GetForward()))
		return
	}
	g.lockedTarget = (g.lockedTarget + 1) % len(g.visibleEnemies)
	targetUnit := g.visibleEnemies[g.lockedTarget]

	g.engine.fpsCamera.FPSLookAt(targetUnit.GetEyePosition())

	g.showTargetInfo(targetUnit, util.ZoneNone)
}

func (g *GameStateFreeAim) updateTargetInfo() {
	rayStart := g.engine.fpsCamera.GetPosition()
	rayEnd := g.engine.fpsCamera.GetPosition().Add(g.engine.fpsCamera.GetFront().Mul(100))
	hitInfo := g.engine.RayCastFreeAim(rayStart, rayEnd, g.selectedUnit.UnitInstance)
	if hitInfo.HitUnit() {
		hitUnit := hitInfo.UnitHit.(*game.UnitInstance)
		zone := hitInfo.BodyPart
		g.showTargetInfo(hitUnit, zone)
	} else {
		distanceToTarget := hitInfo.CollisionWorldPosition.Sub(rayStart).Len()
		g.engine.Print(fmt.Sprintf("Distance to target: %0.2f", distanceToTarget))
	}
}

func (g *GameStateFreeAim) showTargetInfo(targetUnit *game.UnitInstance, zone util.DamageZone) {
	weaponMaxRange := float32(g.selectedUnit.GetWeapon().Definition.MaxRange)
	weaponEffectiveRange := float32(g.selectedUnit.GetWeapon().Definition.EffectiveRange)
	distanceToTarget := g.selectedUnit.GetEyePosition().Sub(targetUnit.GetCenterOfMassPosition()).Len()

	description := targetUnit.GetEnemyDescription()

	description += fmt.Sprintf("\nZone: %s", zone)

	if distanceToTarget > weaponMaxRange {
		description += fmt.Sprintf("\nTarget is out of max range (%0.2f > %0.2f)", distanceToTarget, weaponMaxRange)
	} else if distanceToTarget > weaponEffectiveRange {
		description += fmt.Sprintf("\nTarget is out of effective range (%0.2f > %0.2f)", distanceToTarget, weaponEffectiveRange)
	} else {
		description += fmt.Sprintf("\nTarget is in effective range (%0.2f < %0.2f)", distanceToTarget, weaponEffectiveRange)
	}

	projectedMaxDamage := g.selectedUnit.GetWeapon().GetEstimatedDamage(distanceToTarget)
	health := targetUnit.Health
	bestCaseHealth := health - projectedMaxDamage

	description += fmt.Sprintf("\nMax Damage: %d > Enemy HP: %d", projectedMaxDamage, bestCaseHealth)

	g.engine.Print(description)
}
func (g *GameStateFreeAim) Init(bool) {
	println(fmt.Sprintf("[GameStateFreeAim] Entered for %s", g.selectedUnit.GetName()))

	accuracy := g.engine.GetRules().GetShotAccuracy(g) // TODO: needs to use the final shot accuracy from the server side action..

	g.engine.SwitchToFirstPerson(g.selectedUnit, accuracy)

	g.visibleEnemies = g.engine.GetVisibleEnemyUnits(g.selectedUnit.UnitID())

	g.aimAtNextTarget()
}

func (g *GameStateFreeAim) OnUpperRightAction() {

}

func (g *GameStateFreeAim) OnUpperLeftAction() {

}

func (g *GameStateFreeAim) OnMouseClicked(x float64, y float64) {
	if g.selectedUnit.CanAct() {
		camPos := g.engine.fpsCamera.GetPosition()
		camRotX, camRotY := g.engine.fpsCamera.GetRotation()
		println(fmt.Sprintf("[GameStateFreeAim] Sending action %s: (%0.2f, %0.2f, %0.2f) (%0.2f, %0.2f)", g.selectedAction.GetName(), camPos.X(), camPos.Y(), camPos.Z(), camRotX, camRotY))
		util.MustSend(g.engine.server.FreeAimAction(g.selectedUnit.UnitID(), g.selectedAction.GetName(), camPos, [][2]float32{{camRotX, camRotY}}))
	} else {
		println("[GameStateFreeAim] Unit cannot act")
		g.engine.Print("Unit cannot act")
	}
}

func (g *GameStateFreeAim) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	oldCamPos := g.engine.fpsCamera.GetPosition()
	oldPos := voxel.PositionToGridInt3(oldCamPos)
	g.engine.fpsCamera.ChangePosition(float32(elapsed), movementVector)
	newPos := voxel.PositionToGridInt3(g.engine.fpsCamera.GetPosition())

	if newPos != oldPos {
		g.engine.fpsCamera.SetPosition(oldCamPos)
	}
}

func (g *GameStateFreeAim) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.lastMouseX = newX
	g.lastMouseY = newY

	dx := newX - oldX
	dy := newY - oldY

	g.engine.fpsCamera.ChangeAngles(float32(dx), float32(dy))

	g.updateTargetInfo()
}
