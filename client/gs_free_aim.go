package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateFreeAim struct {
	engine         *BattleGame
	selectedUnit   *Unit
	validTargets   []voxel.Int3
	lastMouseX     float64
	lastMouseY     float64
	selectedAction game.TargetAction
}

func (g *GameStateFreeAim) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		g.engine.isoCamera.ZoomOut(deltaTime, yoff)
	} else {
		g.engine.isoCamera.ZoomIn(deltaTime, -yoff)
	}
}

func (g *GameStateFreeAim) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyEscape {
		g.engine.SwitchToIsoCamera()
		g.engine.PopState()
	}
}

func (g *GameStateFreeAim) Init(bool) {
	println(fmt.Sprintf("[GameStateFreeAim] Entered for %s", g.selectedUnit.GetName()))
	g.engine.SwitchToFirstPerson(g.selectedUnit.GetEyePosition())
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
		sourceOfProjectile := g.selectedUnit.GetEyePosition()
		directionVector := g.engine.fpsCamera.GetFront().Normalize()
		velocity := directionVector.Mul(10)
		sourceOffset := sourceOfProjectile.Add(directionVector.Mul(0.5))

		g.engine.SpawnProjectile(sourceOffset, velocity)
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
