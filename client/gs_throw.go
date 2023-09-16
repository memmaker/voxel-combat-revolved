package client

import (
    "github.com/go-gl/glfw/v3.3/glfw"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/game"
)

type GameStateThrowTarget struct {
    IsoMovementState
    throwAction         *game.ActionThrow
    trajectoryPositions []mgl32.Vec3
    targetPos           mgl32.Vec3
}

func (g *GameStateThrowTarget) OnServerMessage(msgType string, json string) {

}

func (g *GameStateThrowTarget) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateThrowTarget) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
    g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
    if g.engine.lastHitInfo.Hit && g.engine.lastHitInfo.InsideMap {
        g.updateThrowTrajectory(g.engine.lastHitInfo.CollisionWorldPosition)
    }
}

func (g *GameStateThrowTarget) OnKeyPressed(key glfw.Key) {
    if g.engine.actionbar.HandleKeyEvent(key) {
        return
    }

    if key == glfw.KeyTab {
        g.engine.SwitchToNextUnit(g.engine.selectedUnit)
    } else {
        g.IsoMovementState.OnKeyPressed(key)
    }
}

func (g *GameStateThrowTarget) Init(bool) {
    g.engine.groundSelector.Hide()
    g.engine.lines.Clear()
}

func (g *GameStateThrowTarget) OnMouseClicked(x float64, y float64) {
    if len(g.trajectoryPositions) > 0 && g.engine.selectedUnit.CanAct() {
        g.engine.lines.Clear()
        // must send item
        util.MustSend(g.engine.server.ThrownUnitAction(g.engine.selectedUnit.UnitID(), g.throwAction.GetName(), g.throwAction.GetItemName(), []mgl32.Vec3{g.targetPos}))
        g.engine.PopState()
    }
}

func (g *GameStateThrowTarget) updateThrowTrajectory(targetPos mgl32.Vec3) {
    color := ColorTechTeal
    path := g.throwAction.GetTrajectory(targetPos)

    g.targetPos = targetPos
    g.trajectoryPositions = path

    g.engine.lines.Clear()
    if len(g.trajectoryPositions) > 0 {
        g.engine.lines.SetColor(color)
        g.engine.lines.AddPathLine(g.trajectoryPositions)
        g.engine.lines.UpdateVerticesAndShow()
    }
}
