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
}

func (g *GameStateAction) OnKeyPressed(key glfw.Key) {

}

func (g *GameStateAction) Init() {
    // get valid targets for action
    g.validTargets = g.selectedAction.GetValidTargets(g.selectedUnit)
    // highlight valid targets
    g.engine.HighlightBlocks(g.validTargets)
}

func (g *GameStateAction) OnZoomIn(deltaTime float64) {
    g.engine.camera.ZoomIn(deltaTime)
}

func (g *GameStateAction) OnZoomOut(deltaTime float64) {
    g.engine.camera.ZoomOut(deltaTime)
}

func (g *GameStateAction) OnUpperRightAction() {
    g.engine.camera.RotateRight()
}

func (g *GameStateAction) OnUpperLeftAction() {
    g.engine.camera.RotateLeft()
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
    println(fmt.Sprintf("Clicked at %0.2f, %0.2f", x, y))
    // project point from screen space to camera space
    rayStart, rayEnd := g.engine.camera.GetPickingRayFromScreenPosition(x, y)
    hitInfo := g.engine.RayCast(rayStart, rayEnd)
    if hitInfo != nil && hitInfo.Hit {
        // check if target is valid
        g.selectedAction.Execute(g.selectedUnit, hitInfo.PreviousGridPosition)
        /*
        for _, target := range g.validTargets {
            if target == hitInfo.PreviousGridPosition {
                g.selectedAction.Execute(g.selectedUnit, target)
            }
        }
         */
    }
}


func (g *GameStateAction) OnDirectionKeys(movementVector [2]int, elapsed float64) {
    g.engine.camera.ChangePosition(movementVector, float32(elapsed))
}

func (g *GameStateAction) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {

}
