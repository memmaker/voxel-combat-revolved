package game

import (
    "fmt"
    "github.com/go-gl/glfw/v3.3/glfw"
    "github.com/memmaker/battleground/engine/util"
)

type GameStateUnit struct {
    engine *BattleGame
    selectedUnit *Unit
}

func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
    if key == glfw.KeySpace {
        g.engine.SwitchToAction(g.selectedUnit, ActionMove{engine: g.engine})
    }
}

func (g *GameStateUnit) Init() {
    println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
    g.engine.blockSelector.SetPosition(util.ToGrid(g.selectedUnit.GetFootPosition()))
}

func (g *GameStateUnit) OnZoomIn(deltaTime float64) {
    g.engine.camera.ZoomIn(deltaTime)
}

func (g *GameStateUnit) OnZoomOut(deltaTime float64) {
    g.engine.camera.ZoomOut(deltaTime)
}

func (g *GameStateUnit) OnUpperRightAction() {
    g.engine.camera.RotateRight()
}

func (g *GameStateUnit) OnUpperLeftAction() {
    g.engine.camera.RotateLeft()
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
    println(fmt.Sprintf("Clicked at %0.2f, %0.2f", x, y))
    // project point from screen space to camera space
    rayStart, rayEnd := g.engine.camera.GetPickingRayFromScreenPosition(x, y)
    hitInfo := g.engine.RayCast(rayStart, rayEnd)
    if hitInfo != nil && hitInfo.UnitHit != nil {
        g.selectedUnit = hitInfo.UnitHit
        println(fmt.Sprintf("Selected unit at %0.2f, %0.2f, %0.2f", g.selectedUnit.GetPosition().X(), g.selectedUnit.GetPosition().Y(), g.selectedUnit.GetPosition().Z()))
        g.engine.blockSelector.SetPosition(util.ToGrid(g.selectedUnit.GetFootPosition()))
    }
}


func (g *GameStateUnit) OnDirectionKeys(movementVector [2]int, elapsed float64) {
    g.engine.camera.ChangePosition(movementVector, float32(elapsed))
}

func (g *GameStateUnit) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {

}
