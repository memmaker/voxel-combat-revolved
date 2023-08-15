package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type GameStateUnit struct {
	IsoMovementState
	selectedUnit     *Unit
	noCameraMovement bool
	moveAction       *game.ActionMove
}

func NewGameStateUnit(engine *BattleClient, unit *Unit) *GameStateUnit {
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		selectedUnit: unit,
		moveAction:   game.NewActionMove(engine.GetVoxelMap()),
	}
}
func NewGameStateUnitNoCamMove(engine *BattleClient, unit *Unit) *GameStateUnit {
	return &GameStateUnit{
		IsoMovementState: IsoMovementState{
			engine: engine,
		},
		selectedUnit:     unit,
		moveAction:       game.NewActionMove(engine.GetVoxelMap()),
		noCameraMovement: true,
	}
}
func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.nextUnit()
		}
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
			validTargets := g.moveAction.GetValidTargets(g.selectedUnit)
			if len(validTargets) > 0 {
				g.engine.GetVoxelMap().SetHighlights(validTargets, 12)
			}
		}
		println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
		footPos := util.ToGrid(g.selectedUnit.GetFootPosition())
		g.engine.SwitchToGroundSelector()
		g.engine.unitSelector.SetPosition(footPos)

		if !g.noCameraMovement {
			g.engine.isoCamera.CenterOn(footPos.Add(mgl32.Vec3{0.5, 0, 0.5}))
		}

		g.setActionBar()
	}
}

func (g *GameStateUnit) setActionBar() {
	g.engine.actionbar.SetActions([]gui.ActionItem{
		{
			Name:         "Fire",
			TextureIndex: 1,
			Execute: func() {
				if !g.selectedUnit.CanAct() {
					println("[GameStateUnit] Unit cannot act anymore.")
					return
				}
				g.engine.SwitchToAction(g.selectedUnit, game.NewActionShot(g.engine.GameInstance))
			},
			Hotkey: glfw.KeyR,
		},
		{
			Name:         "Free Aim",
			TextureIndex: 2,
			Execute: func() {
				if !g.selectedUnit.CanAct() {
					println("[GameStateUnit] Unit cannot act anymore.")
					return
				}
				g.engine.SwitchToFreeAim(g.selectedUnit, game.NewActionShot(g.engine.GameInstance))
			},
			Hotkey: glfw.KeyF,
		},
		{
			Name:         "End Turn",
			TextureIndex: 3,
			Execute:      g.engine.EndTurn,
			Hotkey:       glfw.KeyF8,
		},
	})
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
	} else if g.moveAction.IsValidTarget(g.selectedUnit, groundBlockPos) && g.selectedUnit.CanAct() {
		util.MustSend(g.engine.server.TargetedUnitAction(g.selectedUnit.UnitID(), g.moveAction.GetName(), groundBlockPos))
	}
}
