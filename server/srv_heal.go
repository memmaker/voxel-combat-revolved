package server

import (
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionHeal struct {
	engine  *game.GameInstance
	unit    *game.UnitInstance
	targets []voxel.Int3
	action  *game.ActionHeal
}

func NewServerActionHeal(g *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) *ServerActionHeal {
	return &ServerActionHeal{engine: g, unit: unit, targets: targets, action: game.NewActionHeal(g, unit)}
}

func (a *ServerActionHeal) SetAPCost(int) {}

func (a *ServerActionHeal) IsTurnEnding() bool { return false }

func (a *ServerActionHeal) IsValid() (bool, string) {
	if !a.unit.CanHeal() {
		return false, "This unit cannot heal now"
	}
	if len(a.targets) != 1 || !a.action.IsValidTarget(a.targets[0]) {
		return false, "No wounded ally in reach"
	}
	return true, ""
}

func (a *ServerActionHeal) Execute(mb *game.MessageBuffer) {
	target := a.engine.GetVoxelMap().GetMapObjectAt(a.targets[0]).(*game.UnitInstance)
	if a.targets[0] == a.unit.GetBlockPosition() {
		target = a.unit
	}
	a.unit.ConsumeAP(game.HealAPCost)
	a.unit.HealsUsed++
	healed := target.Heal(game.HealAmount)
	// opponents only learn about heals they can see
	heal := game.VisualHeal{Healer: a.unit.UnitID(), Target: target.UnitID(), Amount: healed}
	for _, userID := range mb.UserIDs() {
		if userID == a.unit.ControlledBy() || a.engine.UnitIsVisibleToPlayer(userID, a.unit.UnitID()) || a.engine.UnitIsVisibleToPlayer(userID, target.UnitID()) {
			mb.AddMessageFor(userID, heal)
		}
	}
}
