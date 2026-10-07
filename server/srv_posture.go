package server

import (
	"fmt"

	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

// ServerActionPosture makes a unit stand, crouch or go prone. It changes what the unit sees and how
// visible it is, so line of sight is recomputed in both directions and sent to every player.
type ServerActionPosture struct {
	engine  *game.GameInstance
	unit    *game.UnitInstance
	posture game.Posture
	facing  voxel.Int3
}

// NewServerActionPosture: a prone unit lies towards the optional target block, otherwise the way it faces.
func NewServerActionPosture(g *game.GameInstance, unit *game.UnitInstance, posture game.Posture, targets []voxel.Int3) *ServerActionPosture {
	facing := unit.GetForward2DCardinal()
	if len(targets) == 1 && targets[0] != unit.GetBlockPosition() {
		facing = targets[0].Sub(unit.GetBlockPosition()).ToCardinalDirection()
	}
	return &ServerActionPosture{engine: g, unit: unit, posture: posture, facing: facing}
}

func (a *ServerActionPosture) SetAPCost(int) {}

func (a *ServerActionPosture) IsTurnEnding() bool {
	return false
}

func (a *ServerActionPosture) IsValid() (bool, string) {
	if !a.unit.CanTakePosture(a.posture) || (a.posture == game.PostureProne && !a.unit.HasRoomToLieDown(a.facing)) {
		return false, fmt.Sprintf("%s cannot %s now. Needs %d AP (and room in front to lie down).", a.unit.GetName(), a.posture.Name(), game.PostureAPCost)
	}
	return true, ""
}

func (a *ServerActionPosture) Execute(mb *game.MessageBuffer) {
	a.unit.ConsumeAP(game.PostureAPCost)
	if a.posture == game.PostureProne {
		a.unit.GoProneFacing(a.facing)
	} else {
		a.unit.SetPosture(a.posture)
	}

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
			Posture:        a.unit.Posture,
			Facing:         a.unit.GetForward2DCardinal(),
			LOSMatrix:      losMatrix,
			PressureMatrix: a.engine.GetPressureMatrix(),
		}
		if userID == a.unit.ControlledBy() {
			msg.APCost = game.PostureAPCost
			msg.Spotted = visibleEnemies
		} else if a.engine.UnitIsVisibleToPlayer(userID, a.unit.UnitID()) {
			msg.UpdatedUnit = a.unit
		}
		mb.AddMessageFor(userID, msg)
	}
}
