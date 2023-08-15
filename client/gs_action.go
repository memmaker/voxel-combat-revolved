package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type GameStateAction struct {
	IsoMovementState
	selectedUnit   *Unit
	selectedAction game.TargetAction
}

func (g *GameStateAction) OnKeyPressed(key glfw.Key) {
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.engine.SwitchToNextUnit(g.selectedUnit)
		}
	}
}

func (g *GameStateAction) Init(bool) {
	println(fmt.Sprintf("[GameStateAction] Entered for %s with action %s", g.selectedUnit.GetName(), g.selectedAction.GetName()))
	validTargets := g.selectedAction.GetValidTargets(g.selectedUnit)
	println(fmt.Sprintf("[GameStateAction] Valid targets: %d", len(validTargets)))
	if len(validTargets) > 0 {
		g.engine.GetVoxelMap().SetHighlights(validTargets, 12)
	}
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateAction] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	groundBlock := g.engine.groundSelector.GetBlockPosition()
	println(fmt.Sprintf("[GameStateAction] Block %s", groundBlock.ToString()))
	if g.selectedUnit.CanAct() && g.selectedAction.IsValidTarget(g.selectedUnit, groundBlock) {
		println(fmt.Sprintf("[GameStateAction] Target %s is VALID, sending to server.", groundBlock.ToString()))
		util.MustSend(g.engine.server.TargetedUnitAction(g.selectedUnit.UnitID(), g.selectedAction.GetName(), groundBlock))
		g.engine.PopState()
		g.engine.GetVoxelMap().ClearHighlights()
	}
}
