package game

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/voxel"
)

type GameStateAction struct {
	engine         *BattleGame
	selectedUnit   *Unit
	selectedAction Action
	validTargets   []voxel.Int3
	lastMouseX     float64
	lastMouseY     float64
}

func (g *GameStateAction) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		g.engine.camera.ZoomOut(deltaTime, yoff)
	} else {
		g.engine.camera.ZoomIn(deltaTime, -yoff)
	}
}

func (g *GameStateAction) OnKeyPressed(key glfw.Key) {

}

func (g *GameStateAction) Init() {
	println(fmt.Sprintf("[GameStateAction] Entered for %s with action %s", g.selectedUnit.GetName(), g.selectedAction.GetName()))
	// get valid targets for action
	g.validTargets = g.selectedAction.GetValidTargets(g.selectedUnit)
	println(fmt.Sprintf("[GameStateAction] Valid targets: %d", len(g.validTargets)))
	for _, target := range g.validTargets {
		println(fmt.Sprintf(" --> %s", target.ToString()))
	}
	// highlight valid targets
	g.engine.voxelMap.SetHighlights(g.validTargets)
}

func (g *GameStateAction) OnUpperRightAction() {
	g.engine.camera.RotateRight()
}

func (g *GameStateAction) OnUpperLeftAction() {
	g.engine.camera.RotateLeft()
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateAction] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to camera space
	rayStart, rayEnd := g.engine.camera.GetPickingRayFromScreenPosition(x, y)
	hitInfo := g.engine.RayCast(rayStart, rayEnd)
	println(fmt.Sprintf("[GameStateAction] Block %s", hitInfo.PreviousGridPosition.ToString()))
	if hitInfo != nil && hitInfo.Hit {
		// check if target is valid
		for _, target := range g.validTargets {
			if target == hitInfo.PreviousGridPosition {
				println(fmt.Sprintf("[GameStateAction] Target %s is VALID", target.ToString()))
				g.selectedAction.Execute(g.selectedUnit, target)
				g.engine.voxelMap.ClearHighlights()
				g.engine.PopState()
				return
			}
		}
	}
}

func (g *GameStateAction) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	g.engine.camera.ChangePosition(movementVector, float32(elapsed))
	g.engine.UpdateMousePicking(g.lastMouseX, g.lastMouseY)
}

func (g *GameStateAction) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.lastMouseX = newX
	g.lastMouseY = newY
	g.engine.UpdateMousePicking(newX, newY)
}
