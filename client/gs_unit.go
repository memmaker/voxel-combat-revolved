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
}

func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeySpace && g.selectedUnit.CanAct() {
		g.engine.SwitchToAction(g.selectedUnit, game.NewActionMove(g.engine.voxelMap))
	} else if key == glfw.KeyF {
		g.engine.SwitchToAction(g.selectedUnit, game.NewActionShot(g.engine))
	} else if key == glfw.KeyEnter {
		g.engine.SwitchToFreeAim(g.selectedUnit, game.NewActionShot(g.engine))
	} else if key == glfw.KeyTab {
		g.nextUnit()
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
		println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
		footPos := util.ToGrid(g.selectedUnit.GetFootPosition())
		g.engine.SwitchToGroundSelector()
		g.engine.unitSelector.SetPosition(footPos)
		if !g.noCameraMovement {
			g.engine.isoCamera.CenterOn(footPos.Add(mgl32.Vec3{0.5, 0, 0.5}))
		}
		g.engine.actionbar.SetActions([]gui.ActionItem{
			{
				Name:         "Move",
				TextureIndex: 0,
				Execute: func() {
					if !g.selectedUnit.CanAct() {
						println("[GameStateUnit] Unit cannot act anymore.")
						return
					}
					g.engine.SwitchToAction(g.selectedUnit, game.NewActionMove(g.engine.voxelMap))
				},
				Hotkey: glfw.KeySpace,
			},
			{
				Name:         "Fire",
				TextureIndex: 1,
				Execute: func() {
					if !g.selectedUnit.CanAct() {
						println("[GameStateUnit] Unit cannot act anymore.")
						return
					}
					g.engine.SwitchToAction(g.selectedUnit, game.NewActionShot(g.engine))
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
					g.engine.SwitchToFreeAim(g.selectedUnit, game.NewActionShot(g.engine))
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
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateUnit] Screen clicked at (%0.1f, %0.1f)", x, y))
	// project point from screen space to isoCamera space
	rayStart, rayEnd := g.engine.isoCamera.GetPickingRayFromScreenPosition(x, y)
	hitInfo := g.engine.RayCastGround(rayStart, rayEnd)

	if hitInfo.HitUnit() {
		unitHit := hitInfo.UnitHit.(*game.UnitInstance)
		if unitHit != g.selectedUnit.UnitInstance && unitHit.CanAct() && g.engine.IsUnitOwnedByClient(unitHit.UnitID()) {
			g.selectedUnit = g.engine.GetUnit(unitHit.UnitID())
			println(fmt.Sprintf("[GameStateUnit] Selected unit at %s", g.selectedUnit.GetBlockPosition().ToString()))
			g.Init(false)
		}
	}
}
