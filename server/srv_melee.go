package server

import (
	"fmt"
	"math/rand"

	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionMelee struct {
	engine  *game.GameInstance
	unit    *game.UnitInstance
	targets []voxel.Int3
	action  *game.ActionMelee
}

func NewServerActionMelee(g *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) *ServerActionMelee {
	return &ServerActionMelee{engine: g, unit: unit, targets: targets, action: game.NewActionMelee(g, unit)}
}

func (a *ServerActionMelee) SetAPCost(int) {}

func (a *ServerActionMelee) IsTurnEnding() bool { return false }

func (a *ServerActionMelee) IsValid() (bool, string) {
	if a.unit.GetIntegerAP() < game.MeleeAPCost {
		return false, fmt.Sprintf("Not enough AP for melee. Need %d, have %d", game.MeleeAPCost, a.unit.GetIntegerAP())
	}
	if len(a.targets) != 1 || !a.action.IsValidTarget(a.targets[0]) {
		return false, "No enemy in reach"
	}
	return true, ""
}

func (a *ServerActionMelee) Execute(mb *game.MessageBuffer) {
	target := a.engine.GetVoxelMap().GetMapObjectAt(a.targets[0]).(*game.UnitInstance)
	a.unit.ConsumeAP(game.MeleeAPCost)
	result := game.VisualMelee{Attacker: a.unit.UnitID(), Target: target.UnitID(), APCost: game.MeleeAPCost, TargetPosition: a.targets[0]}
	switch {
	case target.Reaction == game.ReactionParry:
		result.Parried, result.CounterDamage = true, game.ParryCounterDamage
		a.engine.ApplyDamage(target, a.unit, game.ParryCounterDamage, util.ZoneTorso)
	case rand.Float64() < game.MeleeHitChance:
		result.Hit, result.Damage = true, game.MeleeDamage
		if target.Reaction == game.ReactionBrace {
			result.Damage, result.Braced = game.BracedBulletDamage(game.MeleeDamage), true
		}
		a.engine.ApplyDamage(a.unit, target, result.Damage, util.ZoneTorso)
	}
	a.unit.SetForward(voxel.DirectionToGridInt3(a.targets[0].Sub(a.unit.GetBlockPosition()).ToVec3()))
	mb.AddMessageForAll(result)
}
