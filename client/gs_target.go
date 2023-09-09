package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateAction struct {
	IsoMovementState
	selectedUnit   *Unit
	selectedAction game.TargetAction
}

func (g *GameStateAction) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateAction) OnServerMessage(msgType string, json string) {
	switch msgType {
	case "RangedAttack":
		var msg game.VisualRangedAttack
		if util.FromJson(json, &msg) {
			if msg.Attacker == g.selectedUnit.UnitID() && !g.engine.cameraIsFirstPerson {
				g.engine.SwitchToUnitNoCameraMovement(g.selectedUnit)
			}
		}
	}
}

func (g *GameStateAction) OnKeyPressed(key glfw.Key) {
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.engine.SwitchToNextUnit(g.selectedUnit)
		}
	}
}

func (g *GameStateAction) Init(bool) {
	//println(fmt.Sprintf("[GameStateAction] Entered for %s with action %s", g.selectedUnit.GetName(), g.selectedAction.GetName()))
	validTargets := g.selectedAction.GetValidTargets()
	g.engine.lines.Clear()
	//println(fmt.Sprintf("[GameStateAction] Valid targets: %d", len(validTargets)))
	if len(validTargets) > 0 {
		g.engine.highlights.SetFlat(voxel.HighlightTarget, validTargets, mgl32.Vec3{0.0, 1.0, 0.0})
	}
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateAction] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	groundBlock := g.engine.groundSelector.GetBlockPosition()
	println(fmt.Sprintf("[GameStateAction] Block %s", groundBlock.ToString()))
	if g.selectedUnit.CanAct() && g.selectedAction.IsValidTarget(groundBlock) {
		println(fmt.Sprintf("[GameStateAction] Targets %s is VALID, sending to server.", groundBlock.ToString()))
		util.MustSend(g.engine.server.TargetedUnitAction(g.selectedUnit.UnitID(), g.selectedAction.GetName(), []voxel.Int3{groundBlock}))
	}
}
