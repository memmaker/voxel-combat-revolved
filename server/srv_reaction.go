package server

import (
	"fmt"

	"github.com/memmaker/battleground/game"
)

// ServerActionReaction puts a unit into a defensive stance (dodge or brace) until its next turn.
type ServerActionReaction struct {
	unit     *game.UnitInstance
	reaction game.Reaction
}

func NewServerActionReaction(unit *game.UnitInstance, reaction game.Reaction) *ServerActionReaction {
	return &ServerActionReaction{unit: unit, reaction: reaction}
}

func (a *ServerActionReaction) SetAPCost(int) {}

func (a *ServerActionReaction) IsTurnEnding() bool {
	return true
}

func (a *ServerActionReaction) IsValid() (bool, string) {
	if a.unit.GetIntegerAP() < game.ReactionAPCost {
		return false, fmt.Sprintf("Not enough AP to %s. Need %d, have %d", a.reaction, game.ReactionAPCost, a.unit.GetIntegerAP())
	}
	return true, ""
}

func (a *ServerActionReaction) Execute(mb *game.MessageBuffer) {
	a.unit.Reaction = a.reaction
	// only the owner learns about the stance; the opponent has to guess
	mb.AddMessageFor(a.unit.ControlledBy(), game.VisualReaction{
		UnitMessage: game.UnitMessage{GameUnitID: a.unit.UnitID()},
		Reaction:    a.reaction,
	})
}
