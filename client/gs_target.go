package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateBlockTarget struct {
	IsoMovementState
	selectedAction      game.TargetAction
	trajectoryPositions []mgl32.Vec3
	targetBlock         voxel.Int3
}

func (g *GameStateBlockTarget) OnMouseReleased(x float64, y float64) {

}
func (g *GameStateBlockTarget) OnServerMessage(msgType string, json string) {
	switch msgType {
	case "RangedAttack":
		var msg game.VisualRangedAttack
		if util.FromJson(json, &msg) {
            if msg.Attacker == g.engine.selectedUnit.UnitID() && !g.engine.cameraIsFirstPerson {
                g.engine.SwitchToUnitNoCameraMovement(g.engine.selectedUnit)
			}
		}
	}
}

func (g *GameStateBlockTarget) OnKeyPressed(key glfw.Key) {
	if g.engine.actionbar.HandleKeyEvent(key) {
		return
	}

	if key == glfw.KeyTab {
        g.engine.SwitchToNextUnit(g.engine.selectedUnit)
	} else {
		g.IsoMovementState.OnKeyPressed(key)
	}
}

func (g *GameStateBlockTarget) Init(bool) {
	validTargets := g.selectedAction.GetValidTargets()
	g.engine.groundSelector.Hide()
	g.engine.lines.Clear()

	if len(validTargets) > 0 {
		g.engine.highlights.SetFlat(voxel.HighlightTarget, validTargets, mgl32.Vec3{0.0, 1.0, 0.0})
	}
}

func (g *GameStateBlockTarget) OnMouseClicked(x float64, y float64) {
    println(fmt.Sprintf("[GameStateBlockTarget] Clicked at %0.2f, %0.2f", x, y))
    groundBlock := g.engine.groundSelector.GetBlockPosition()
    println(fmt.Sprintf("[GameStateBlockTarget] Block %s", groundBlock.ToString()))

    if g.engine.selectedUnit.CanAct() && g.selectedAction.IsValidTarget(groundBlock) {
        println(fmt.Sprintf("[GameStateBlockTarget] Target %s is VALID, sending to server.", groundBlock.ToString()))
        util.MustSend(g.engine.server.TargetedUnitAction(g.engine.selectedUnit.UnitID(), g.selectedAction.GetName(), []voxel.Int3{groundBlock}))
	}
}
