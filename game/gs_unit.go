package game

import (
    "fmt"
    "github.com/go-gl/mathgl/mgl32"
)

type Engine interface {

}

type GameStateUnit struct {
    engine *BattleGame
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
    println(fmt.Sprintf("Clicked at %0.2f, %0.2f", x, y))
    // project point from screen space to camera space
    proj := g.engine.camera.GetProjectionMatrix()
    view := g.engine.camera.GetViewMatrix()
    projViewInverted := proj.Mul4(view).Inv()
    // normalize x and y to -1..1
    normalizedX := ((float32(x)/float32(g.engine.WindowWidth))*2 - 1)
    normalizedY := ((float32(y)/float32(g.engine.WindowHeight))*2 - 1) * -1
    normalizedNearPos := mgl32.Vec4{normalizedX, normalizedY, g.engine.camera.GetNearPlaneDist(), 1}
    normalizedFarPos := mgl32.Vec4{normalizedX, normalizedY, 100, 1}

    println(fmt.Sprintf("Normalized mouse pos: %v", normalizedNearPos))
    // project point from camera space to world space
    nearWorldPos := projViewInverted.Mul4x1(normalizedNearPos)
    farWorldPos := projViewInverted.Mul4x1(normalizedFarPos)
    // perspective divide
    nearPosCorrected := nearWorldPos.Vec3().Mul(1 / nearWorldPos.W())
    farPosCorrected := farWorldPos.Vec3().Mul(1 / farWorldPos.W())
    println(fmt.Sprintf("Ray start: %v", nearPosCorrected))
    dir := nearPosCorrected.Sub(farPosCorrected).Normalize()
    rayEnd := nearPosCorrected.Add(dir.Mul(100))
    println(fmt.Sprintf("Ray end: %v", rayEnd))
    g.engine.updateSelectedBlock(nearPosCorrected, rayEnd)
}

func (g *GameStateUnit) OnDirectionKeys(movementVector [2]int, elapsed float64) {
    g.engine.camera.ChangePosition(movementVector, float32(elapsed))
}

func (g *GameStateUnit) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {

}
