package client

import (
	"fmt"
	"github.com/memmaker/battleground/game"
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
			unit := a.lastHitInfo.UnitHit.(*game.UnitInstance)
			unitPos := unit.GetBlockPosition()
			unitPosString = fmt.Sprintf("Unit(%d): %s @ %d, %d, %d", unit.UnitID(), unit.GetName(), unitPos.X, unitPos.Y, unitPos.Z)
		}

	}

	//animString := a.allUnits[0].model.GetAnimationDebugString()

	//timerString := a.timer.String()
	camString := a.isoCamera.DebugString()
	debugInfo := fmt.Sprintf("%s\n%s\n%s", selectedBlockString, unitPosString, camString)

	//debugInfo := fmt.Sprintf("\n%s\n%s", unitPosString, animString)
	a.Print(debugInfo)
}
