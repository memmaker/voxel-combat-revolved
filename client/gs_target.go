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

func (g *GameStateAction) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
	if _, isThrow := g.selectedAction.(*game.ActionThrow); isThrow {
		g.updateThrowTrajectory()
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
	if !g.engine.actionbar.HandleKeyEvent(key) {
		if key == glfw.KeyTab {
			g.engine.SwitchToNextUnit(g.selectedUnit)
		} else {
			g.IsoMovementState.OnKeyPressed(key)
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

func (g *GameStateAction) updateThrowTrajectory() {
	gravity := 9.8
	color := ColorTechTeal

	isReachable := false
	sourcePos := g.selectedUnit.GetEyePosition()
	targetPos := g.engine.groundSelector.GetBlockPosition().ToBlockCenterVec3()

	trajectoryPositions := g.calcTrajectory(sourcePos, targetPos, gravity)

	g.engine.lines.Clear()
	if len(trajectoryPositions) > 0 {
		if !isReachable {
			color = ColorNegativeRed
		}
		g.engine.lines.SetColor(color)
		g.engine.lines.AddPathLine(trajectoryPositions)
		g.engine.lines.UpdateVerticesAndShow()
	}
}

func (g *GameStateAction) calcTrajectory(sourcePos, dest mgl32.Vec3, gravity float64) []mgl32.Vec3 {
	destRelatedToOrigin := dest.Sub(sourcePos) // translate by sourcePos
	dest2D := mgl32.Vec3{destRelatedToOrigin.X(), 0, destRelatedToOrigin.Z()}
	xAxis := mgl32.Vec3{1.0, 0.0, 0.0}
	rotationForAxisAlign := mgl32.QuatBetweenVectors(dest2D, xAxis).Mat4()
	rotatedDest := rotationForAxisAlign.Mul4x1(destRelatedToOrigin.Vec4(1)).Vec3()

	twoDeeX := rotatedDest.X()
	twoDeeY := rotatedDest.Y()

	gravity = 9.8
	velocity := 10.0
	aimAngle := util.AngleOfLaunch(mgl32.Vec2{twoDeeX, twoDeeY}, velocity, gravity)
	timeToTarget := util.TimeOfFlight(float64(twoDeeX), aimAngle, velocity)
	var trajectory2D []mgl32.Vec2
	for i := 0; i <= 10; i++ {
		percent := float64(i) / 10.0
		time := percent * timeToTarget
		currentX, currentY := util.TrajectoryXY(time, aimAngle, velocity, gravity)
		trajectory2D = append(trajectory2D, mgl32.Vec2{float32(currentX), float32(currentY)})
	}

	var trajectory3D []mgl32.Vec3
	inverseRotation := rotationForAxisAlign.Inv()
	for _, point := range trajectory2D {
		rotatedPoint := inverseRotation.Mul4x1(mgl32.Vec4{point.X(), point.Y(), 0.0, 1.0}).Vec3()
		translatedPoint := rotatedPoint.Add(sourcePos)
		trajectory3D = append(trajectory3D, translatedPoint)
	}
	// now we need to solve a projectile motion from
	// (0,0) to (twoDeeX, twoDeeY)

	return trajectory3D
}
