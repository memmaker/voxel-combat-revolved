package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type GameStateUnit struct {
	IsoMovementState
	selectedUnit     *Unit
	noCameraMovement bool
	moveAction       *game.ActionMove
}

func (g *GameStateUnit) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateUnit) OnServerMessage(msgType string, json string) {

}

func NewGameStateUnit(engine *BattleClient, unit *Unit) *GameStateUnit {
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		selectedUnit: unit,
		moveAction:   game.NewActionMove(engine.GetVoxelMap(), unit.UnitInstance),
	}
}
func NewGameStateUnitNoCamMove(engine *BattleClient, unit *Unit) *GameStateUnit {
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		selectedUnit:     unit,
		moveAction:       game.NewActionMove(engine.GetVoxelMap(), unit.UnitInstance),
		noCameraMovement: false,
	}
}
func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.nextUnit()
		}
	}
	if key == glfw.KeyF1 {
		g.engine.showDebugInfo = !g.engine.showDebugInfo
		if !g.engine.showDebugInfo {
			g.engine.textLabel = nil
		} else {
			g.engine.timer.Reset()
		}
	}
	if key == glfw.KeyF2 {
		g.engine.debugToggleWireFrame()
	}
}

func (g *GameStateUnit) nextUnit() {
	nextUnit, exists := g.engine.GetNextUnit(g.selectedUnit)
	if !exists {
		println("[GameStateUnit] No unit left to act.")
	} else {
		g.selectedUnit = nextUnit
		g.Init(false)
	}
}

func (g *GameStateUnit) Init(wasPopped bool) {
	if !wasPopped {
		if g.selectedUnit.CanMove() {
			g.moveAction = game.NewActionMove(g.engine.GetVoxelMap(), g.selectedUnit.UnitInstance)
			validTargets := g.moveAction.GetValidTargets()
			if len(validTargets) > 0 {
				g.engine.GetVoxelMap().SetHighlights(validTargets)
			}
		}
		//println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
		footPos := util.ToGrid(g.selectedUnit.GetPosition())
		g.engine.SwitchToGroundSelector()
		g.engine.SwitchToIsoCamera()
		g.engine.unitSelector.SetPosition(footPos)

		if !g.noCameraMovement {
			startCam := g.engine.isoCamera.GetTransform()
			g.engine.isoCamera.CenterOn(footPos.Add(mgl32.Vec3{0.5, 0, 0.5}))
			endCam := g.engine.isoCamera.GetTransform()
			g.engine.StartCameraAnimation(startCam, endCam, 0.5)
		}

		g.engine.UpdateActionbarFor(g.selectedUnit)
	}
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
	groundBlockPos := g.engine.groundSelector.GetBlockPosition()
	if g.engine.GetVoxelMap().IsOccupied(groundBlockPos) {
		unitHit := g.engine.GetVoxelMap().GetMapObjectAt(groundBlockPos).(*game.UnitInstance)
		if unitHit != g.selectedUnit.UnitInstance && unitHit.CanAct() && g.engine.IsUnitOwnedByClient(unitHit.UnitID()) {
			g.selectedUnit, _ = g.engine.GetClientUnit(unitHit.UnitID())
			println(fmt.Sprintf("[GameStateUnit] Selected unit at %s", g.selectedUnit.GetBlockPosition().ToString()))
			g.Init(false)
		}
	} else if g.moveAction.IsValidTarget(groundBlockPos) && g.selectedUnit.CanAct() {
		util.MustSend(g.engine.server.TargetedUnitAction(g.selectedUnit.UnitID(), g.moveAction.GetName(), []voxel.Int3{groundBlockPos}))
	}
}
