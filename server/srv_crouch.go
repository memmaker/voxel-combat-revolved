package server

import (
	"fmt"

	"github.com/memmaker/battleground/game"
)

// ServerActionCrouch makes a unit kneel or stand up again. It changes what the unit sees and how
// visible it is, so line of sight is recomputed in both directions and sent to every player.
type ServerActionCrouch struct {
	engine *game.GameInstance
	unit   *game.UnitInstance
}

func NewServerActionCrouch(g *game.GameInstance, unit *game.UnitInstance) *ServerActionCrouch {
	return &ServerActionCrouch{engine: g, unit: unit}
}

func (a *ServerActionCrouch) SetAPCost(int) {}

func (a *ServerActionCrouch) IsTurnEnding() bool {
	return false
}

func (a *ServerActionCrouch) IsValid() (bool, string) {
	if !a.unit.CanCrouch() {
		return false, fmt.Sprintf("%s cannot crouch now. Needs %d AP.", a.unit.GetName(), game.CrouchAPCost)
	}
	return true, ""
}

func (a *ServerActionCrouch) Execute(mb *game.MessageBuffer) {
	a.unit.ConsumeAP(game.CrouchAPCost)
	a.unit.SetCrouched(!a.unit.Crouched)

	visibles, invisibles, _ := a.engine.GetLOSChanges(a.unit, a.unit.GetBlockPosition())
	for _, unit := range visibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), true)
	}
	for _, unit := range invisibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), false)
	}
	for _, enemyUserID := range mb.UserIDs() {
		if enemyUserID == a.unit.ControlledBy() {
			continue
		}
		seenBy, hiddenTo := a.engine.GetReverseLOSChangesForUser(enemyUserID, a.unit)
		for _, observer := range seenBy {
			a.engine.SetLOS(observer, a.unit.UnitID(), true)
		}
		for _, observer := range hiddenTo {
			a.engine.SetLOS(observer, a.unit.UnitID(), false)
		}
	}
	a.engine.UpdatePressureAfterMove(a.unit)

	for _, userID := range mb.UserIDs() {
		losMatrix, visibleEnemies := a.engine.GetLOSState(userID)
		msg := game.VisualStanceChange{
			UnitMessage:    game.UnitMessage{GameUnitID: a.unit.UnitID()},
			Crouched:       a.unit.Crouched,
			LOSMatrix:      losMatrix,
			PressureMatrix: a.engine.GetPressureMatrix(),
		}
		if userID == a.unit.ControlledBy() {
			msg.APCost = game.CrouchAPCost
			msg.Spotted = visibleEnemies
		} else if a.engine.UnitIsVisibleToPlayer(userID, a.unit.UnitID()) {
			msg.UpdatedUnit = a.unit
		}
		mb.AddMessageFor(userID, msg)
	}
}
