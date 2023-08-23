package server

import (
	"fmt"
	"github.com/memmaker/battleground/game"
)

type InvalidServerAction struct {
	Reason string
}

func (i InvalidServerAction) SetAPCost(newCost int) {

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
	SetAPCost(newCost int)
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
		return NewServerActionMove(g, unit, targetAction.Targets)
	case "Shot":
		return NewServerActionSnapShot(g, unit, targetAction.Targets)
	case "Overwatch":
		return NewServerActionOverwatch(g, unit, targetAction.Targets)
	}
	return NewInvalidServerAction(fmt.Sprintf("Unknown action %s", targetAction.Action))
}

func GetFreeAimAction(g *game.GameInstance, msg game.FreeAimActionMessage, unit *game.UnitInstance) ServerAction {
	switch msg.Action {
	case "Shot":
		return NewServerActionFreeShot(g, unit, msg.CamPos, msg.TargetAngles)
		//return NewServerActionPerfectShot(g, unit, msg.RayStart, msg.RayEnd)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", msg.Action))
	return NewInvalidServerAction(fmt.Sprintf("Unknown action %s", msg.Action))
}
