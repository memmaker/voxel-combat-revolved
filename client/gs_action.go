package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateAction struct {
	IsoMovementState
	selectedUnit   *Unit
	selectedAction game.TargetAction
	validTargets   []voxel.Int3
}

func (g *GameStateAction) OnKeyPressed(key glfw.Key) {

}

func (g *GameStateAction) Init(bool) {
	println(fmt.Sprintf("[GameStateAction] Entered for %s with action %s", g.selectedUnit.GetName(), g.selectedAction.GetName()))
	g.validTargets = g.selectedAction.GetValidTargets(g.selectedUnit)
	println(fmt.Sprintf("[GameStateAction] Valid targets: %d", len(g.validTargets)))
	if len(g.validTargets) > 0 {
		g.engine.voxelMap.SetHighlights(g.validTargets, 12)
	}
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateAction] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	groundBlock := g.engine.groundSelector.GetBlockPosition()
	println(fmt.Sprintf("[GameStateAction] Block %s", groundBlock.ToString()))
	if g.selectedUnit.CanAct() {
		// check if target is valid
		for _, target := range g.validTargets {
			if target == groundBlock {
				println(fmt.Sprintf("[GameStateAction] Target %s is VALID, sending to server.", target.ToString()))
				util.MustSend(g.engine.server.TargetedUnitAction(g.selectedUnit.ID, g.selectedAction.GetName(), target))
				g.engine.PopState()
			}
		}
	}
}
