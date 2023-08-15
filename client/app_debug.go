package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
)

func (a *BattleClient) updateDebugInfo() {
	if !a.showDebugInfo {
		return
	}
	//camPos := a.isoCamera.GetPosition()

	selectedBlockString := "Block: none"
	unitPosString := "Unit: none"
	if a.lastHitInfo != nil {
		cursorPos := a.groundSelector.GetBlockPosition()
		selectedBlockString = fmt.Sprintf("Cursor: %d, %d, %d", cursorPos.X, cursorPos.Y, cursorPos.Z)
		if a.lastHitInfo.UnitHit != nil {
			unit := a.lastHitInfo.UnitHit.(*Unit)
			unitPos := unit.GetBlockPosition()
			unitPosString = fmt.Sprintf("Unit(%d): %s @ %d, %d, %d", unit.UnitID(), unit.GetName(), unitPos.X, unitPos.Y, unitPos.Z)
		}

	}

	//animString := a.allUnits[0].model.GetAnimationDebugString()

	//timerString := a.timer.String()
	debugInfo := fmt.Sprintf("%s\n%s", selectedBlockString, unitPosString)
	//debugInfo := fmt.Sprintf("\n%s\n%s", unitPosString, animString)
	a.Print(debugInfo)
}

func (a *BattleClient) placeDebugLine(startEnd [2]mgl32.Vec3) {
	a.debugObjects = a.debugObjects[:0]
	//camDirection := a.player.cam.GetFront()
	var lines [][2]mgl32.Vec3
	lines = [][2]mgl32.Vec3{
		startEnd,
	}
	rayLine := util.NewLineMesh(a.lineShader, lines)
	a.debugObjects = append(a.debugObjects, rayLine)
}
