package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type ServerAction interface {
	IsValid() (bool, string)
	Execute(mb *game.MessageBuffer)
	IsTurnEnding() bool
}

func GetServerActionForUnit(g *game.GameInstance, actionMessage game.UnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch typedMsg := actionMessage.(type) {
	case game.TargetedUnitActionMessage:
		return GetTargetedAction(g, typedMsg, unit)
	case game.FreeAimActionMessage:
		return GetFreeAimAction(g, typedMsg, unit)
	}
	return nil
}

func GetTargetedAction(g *game.GameInstance, targetAction game.TargetedUnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch targetAction.Action {
	case "Move":
		return NewServerActionMove(g, game.NewActionMove(g.GetVoxelMap()), unit, targetAction.Target)
	case "Shot":
		camera := util.NewFPSCamera(unit.GetEyePosition(), 100, 100)
		targetUnit := g.GetVoxelMap().GetMapObjectAt(targetAction.Target).(*game.UnitInstance)
		if targetUnit != nil {
			camera.FPSLookAt(targetUnit.GetCenterOfMassPosition())
		}
		return NewServerActionFreeShot(g, unit, camera)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", targetAction.Action))
	return nil
}

func GetFreeAimAction(g *game.GameInstance, msg game.FreeAimActionMessage, unit *game.UnitInstance) ServerAction {
	switch msg.Action {
	case "Shot":
		camera := util.NewFPSCamera(msg.CamPos, 100, 100)
		camera.Reposition(msg.CamPos, msg.CamRotX, msg.CamRotY)
		return NewServerActionFreeShot(g, unit, camera)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", msg.Action))
	return nil
}
