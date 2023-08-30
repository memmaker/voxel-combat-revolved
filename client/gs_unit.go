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
	noCameraMovement bool
	moveAction       *game.ActionMove
	lastCursorPos    voxel.Int3
}

func (g *GameStateUnit) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateUnit) OnServerMessage(msgType string, json string) {

}

func NewGameStateUnit(engine *BattleClient, unit *Unit) *GameStateUnit {
	engine.SetSelectedUnit(unit)
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		moveAction: game.NewActionMove(engine.GetVoxelMap(), unit.UnitInstance),
	}
}
func NewGameStateUnitNoCamMove(engine *BattleClient, unit *Unit) *GameStateUnit {
	engine.SetSelectedUnit(unit)
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		moveAction:       game.NewActionMove(engine.GetVoxelMap(), unit.UnitInstance),
		noCameraMovement: false,
	}
}
func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.nextUnit()
			return
		}
	}
	if key == glfw.KeyF1 {
		g.engine.showDebugInfo = !g.engine.showDebugInfo
		if !g.engine.showDebugInfo {
			g.engine.textLabel = nil
			return
		} else {
			g.engine.timer.Reset()
			return
		}
	}
	if key == glfw.KeyF2 {
		g.engine.debugToggleWireFrame()
		return
	}
}

func (g *GameStateUnit) nextUnit() {
	nextUnit, exists := g.engine.GetNextUnit(g.engine.selectedUnit)
	if !exists {
		println("[GameStateUnit] No unit left to act.")
	} else {
		g.engine.SetSelectedUnit(nextUnit)
		g.Init(false)
	}
}

func (g *GameStateUnit) Init(wasPopped bool) {
	if !wasPopped {
		if g.engine.selectedUnit.CanMove() {
			g.moveAction = game.NewActionMove(g.engine.GetVoxelMap(), g.engine.selectedUnit.UnitInstance)
			validTargets := g.moveAction.GetValidTargets()
			if len(validTargets) > 0 {
				g.engine.SetHighlightsForMovement(g.moveAction, g.engine.selectedUnit, validTargets)
			}
		} else {
			g.engine.highlights.ClearAndUpdateFlat(voxel.HighlightMove)
		}

		//println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
		footPos := util.ToGrid(g.engine.selectedUnit.GetPosition())
		g.engine.SwitchToIsoCamera()

		if !g.noCameraMovement {
			startCam := g.engine.isoCamera.GetTransform()
			g.engine.isoCamera.SetLookTarget(footPos.Add(mgl32.Vec3{0.5, 0, 0.5}))
			endCam := g.engine.isoCamera
			g.engine.StartCameraLookAnimation(startCam, endCam, 0.5)
		}
	}
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
	groundBlockPos := g.engine.groundSelector.GetBlockPosition()
	if g.engine.GetVoxelMap().IsOccupied(groundBlockPos) {
		unitHit := g.engine.GetVoxelMap().GetMapObjectAt(groundBlockPos).(*game.UnitInstance)
		if unitHit != g.engine.selectedUnit.UnitInstance && unitHit.CanAct() && g.engine.IsUnitOwnedByClient(unitHit.UnitID()) {
			clickedUnit, _ := g.engine.GetClientUnit(unitHit.UnitID())
			g.engine.SetSelectedUnit(clickedUnit)
			println(fmt.Sprintf("[GameStateUnit] Selected unit at %s", g.engine.selectedUnit.GetBlockPosition().ToString()))
			g.Init(false)
		}
	} else if g.moveAction.IsValidTarget(groundBlockPos) {
		util.MustSend(g.engine.server.TargetedUnitAction(g.engine.selectedUnit.UnitID(), g.moveAction.GetName(), []voxel.Int3{groundBlockPos}))
	}
}

func (g *GameStateUnit) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
	cursorPos := g.engine.groundSelector.GetBlockPosition()
	if cursorPos == g.lastCursorPos {
		return
	}
	g.lastCursorPos = cursorPos
	g.updateLOSIndicator(cursorPos)
}

func (g *GameStateUnit) updateLOSIndicator(cursorPos voxel.Int3) {
	g.engine.lines.Clear()
	atLeastOneEnemyInSight := false
	currentUnit := g.engine.selectedUnit
	visibleEnemies := g.engine.GetAllVisibleEnemies(currentUnit.ControlledBy())
	observerPos := cursorPos.ToBlockCenterVec3().Add(currentUnit.GetEyeOffset())
	for enemy, _ := range visibleEnemies {
		if g.engine.CanSeeFrom(currentUnit.UnitInstance, enemy, observerPos) {
			enemyEyePos := enemy.GetEyePosition()
			g.engine.lines.AddSimpleLine(observerPos, enemyEyePos)
			atLeastOneEnemyInSight = true
		}
	}
	if atLeastOneEnemyInSight {
		g.engine.lines.UpdateVerticesAndShow()
	}
}
