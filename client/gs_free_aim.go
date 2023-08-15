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
	} else if key == glfw.KeyM {
		g.engine.fpsCamera.ChangeFOV(1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	} else if key == glfw.KeyN {
		g.engine.fpsCamera.ChangeFOV(-1, g.selectedUnit.GetWeapon().GetMinFOVForZoom())
	} else if key == glfw.KeyJ {
		g.engine.fpsCamera.ResetFOV()
	}
}

func (g *GameStateFreeAim) Init(bool) {
	println(fmt.Sprintf("[GameStateFreeAim] Entered for %s", g.selectedUnit.GetName()))
	g.engine.SwitchToFirstPerson(g.selectedUnit)
	g.engine.fpsCamera.ResetFOV()
}

func (g *GameStateFreeAim) OnUpperRightAction() {
	g.engine.isoCamera.RotateRight()
}

func (g *GameStateFreeAim) OnUpperLeftAction() {
	g.engine.isoCamera.RotateLeft()
}

func (g *GameStateFreeAim) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateFreeAim] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	groundBlock := g.engine.groundSelector.GetBlockPosition()
	println(fmt.Sprintf("[GameStateFreeAim] Block %s", groundBlock.ToString()))
	if g.selectedUnit.CanAct() {
		// check if target is valid

		camPos := g.engine.fpsCamera.GetPosition()
		camRotX, camRotY := g.engine.fpsCamera.GetRotation()
		//sourceOfProjectile := g.selectedUnit.GetEyePosition()

		//destination := sourceOffset.Add(velocity)
		//g.engine.SpawnProjectile(sourceOffset, velocity)
		println(fmt.Sprintf("[GameStateFreeAim] Sending action %s: (%0.2f, %0.2f, %0.2f) (%0.2f, %0.2f)", g.selectedAction.GetName(), camPos.X(), camPos.Y(), camPos.Z(), camRotX, camRotY))
		util.MustSend(g.engine.server.FreeAimAction(g.selectedUnit.UnitID(), g.selectedAction.GetName(), camPos, camRotX, camRotY))
	} else {
		println("[GameStateFreeAim] Unit cannot act")
		g.engine.Print("Unit cannot act")
	}
}

func (g *GameStateFreeAim) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	g.engine.isoCamera.ChangePosition(movementVector, float32(elapsed))
	g.engine.UpdateMousePicking(g.lastMouseX, g.lastMouseY)
}

func (g *GameStateFreeAim) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.lastMouseX = newX
	g.lastMouseY = newY
	dx := newX - oldX
	dy := newY - oldY
	g.engine.fpsCamera.ChangeAngles(float32(dx), float32(dy))
}
