package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type InvalidServerAction struct {
	Reason string
}

func (i InvalidServerAction) IsValid() (bool, string) {
	return false, i.Reason
}

func (i InvalidServerAction) Execute(mb *game.MessageBuffer) {
	println(fmt.Sprintf("[InvalidServerAction] ERR - Execute - %s", i.Reason))
}

func (i InvalidServerAction) IsTurnEnding() bool {
	return false
}

func NewInvalidServerAction(reason string) *InvalidServerAction {
	return &InvalidServerAction{
		Reason: reason,
	}
}

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
	return NewInvalidServerAction(fmt.Sprintf("Unknown action type %T", actionMessage))
}

func GetTargetedAction(g *game.GameInstance, targetAction game.TargetedUnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch targetAction.Action {
	case "Move":
		return NewServerActionMove(g, game.NewActionMove(g.GetVoxelMap()), unit, targetAction.Target)
	case "Shot":
		camera := util.NewFPSCamera(unit.GetEyePosition(), 100, 100)
		if !g.GetVoxelMap().IsOccupied(targetAction.Target) {
			return NewInvalidServerAction(fmt.Sprintf("SnapShot target %s is not occupied", targetAction.Target.ToString()))
		}
		targetUnit := g.GetVoxelMap().GetMapObjectAt(targetAction.Target).(*game.UnitInstance)
		if targetUnit != nil {
			camera.FPSLookAt(targetUnit.GetCenterOfMassPosition())
		}
		return NewServerActionFreeShot(g, unit, camera)
	}
	return NewInvalidServerAction(fmt.Sprintf("Unknown action %s", targetAction.Action))
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
