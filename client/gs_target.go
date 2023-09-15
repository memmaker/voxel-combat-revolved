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
	selectedUnit        *Unit
	selectedAction      game.TargetAction
	trajectoryPositions []mgl32.Vec3
	targetBlock         voxel.Int3
}

func (g *GameStateAction) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateAction) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
	if g.engine.lastHitInfo.Hit && g.engine.lastHitInfo.InsideMap {
		if _, isThrow := g.selectedAction.(*game.ActionThrow); isThrow {
			g.updateThrowTrajectory(g.engine.lastHitInfo.CollisionWorldPosition)
		}
	}

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
	if g.engine.actionbar.HandleKeyEvent(key) {
		return
	}

	if key == glfw.KeyTab {
		g.engine.SwitchToNextUnit(g.selectedUnit)
	} else {
		g.IsoMovementState.OnKeyPressed(key)
	}
}

func (g *GameStateAction) Init(bool) {
	//println(fmt.Sprintf("[GameStateAction] Entered for %s with action %s", g.selectedUnit.GetName(), g.selectedAction.GetName()))
	validTargets := g.selectedAction.GetValidTargets()
	g.engine.groundSelector.Hide()
	g.engine.lines.Clear()
	//println(fmt.Sprintf("[GameStateAction] Valid targets: %d", len(validTargets)))
	if len(validTargets) > 0 {
		g.engine.highlights.SetFlat(voxel.HighlightTarget, validTargets, mgl32.Vec3{0.0, 1.0, 0.0})
	}
}

func (g *GameStateAction) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateAction] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	if len(g.trajectoryPositions) > 0 {
		g.engine.lines.Clear()
		g.engine.SpawnThrownObject(g.trajectoryPositions, func() {
			g.engine.smoker.AddPoisonCloud(g.targetBlock, 5, 1)
		})
		g.engine.PopState()
	}
}

func (g *GameStateAction) updateThrowTrajectory(targetPos mgl32.Vec3) {
	gravity := 9.8
	color := ColorTechTeal
	sourcePos := g.selectedUnit.GetEyePosition()
	maxVelocity := g.selectedUnit.Definition.CoreStats.ThrowVelocity
	g.trajectoryPositions = util.CalculateTrajectory(sourcePos, targetPos, maxVelocity, gravity)
	g.targetBlock = voxel.PositionToGridInt3(targetPos)
	g.engine.lines.Clear()
	if len(g.trajectoryPositions) > 0 {
		g.engine.lines.SetColor(color)
		g.engine.lines.AddPathLine(g.trajectoryPositions)
		g.engine.lines.UpdateVerticesAndShow()
	}
}
