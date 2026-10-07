package game

import (
	"math"

	"github.com/memmaker/battleground/engine/voxel"
)

const (
	MeleeAPCost    = 2
	MeleeDamage    = 4
	MeleeHitChance = 0.85
	// ParryCounterDamage is dealt to an attacker whose melee attack was parried.
	ParryCounterDamage = 3
	// ReactionParry makes melee attacks against the unit fail and hurts the attacker.
	ReactionParry Reaction = "Parry"
)

// ActionMelee strikes an adjacent, visible enemy with the butt of the weapon.
type ActionMelee struct {
	engine *GameInstance
	unit   *UnitInstance
	valid  map[voxel.Int3]bool
}

func NewActionMelee(engine *GameInstance, unit *UnitInstance) *ActionMelee {
	a := &ActionMelee{engine: engine, unit: unit, valid: make(map[voxel.Int3]bool)}
	for _, enemy := range engine.GetVisibleEnemyUnits(unit.UnitID()) {
		if IsInMeleeRange(unit.GetBlockPosition(), enemy.GetBlockPosition()) {
			a.valid[enemy.GetBlockPosition()] = true
		}
	}
	return a
}

// IsInMeleeRange: neighboring blocks, diagonals included, at most one level apart.
func IsInMeleeRange(from, to voxel.Int3) bool {
	dx, dy, dz := math.Abs(float64(from.X-to.X)), math.Abs(float64(from.Y-to.Y)), math.Abs(float64(from.Z-to.Z))
	return dx <= 1 && dz <= 1 && dy <= 1 && (dx+dz) > 0
}

func (a *ActionMelee) GetName() string { return "Melee" }

func (a *ActionMelee) GetValidTargets() []voxel.Int3 {
	result := make([]voxel.Int3, 0, len(a.valid))
	for pos := range a.valid {
		result = append(result, pos)
	}
	return result
}

func (a *ActionMelee) IsValidTarget(target voxel.Int3) bool { return a.valid[target] }

func (a *ActionMelee) IsTurnEnding() bool { return false }

func (u *UnitInstance) CanMelee() bool {
	return u.CanAct() && u.GetIntegerAP() >= MeleeAPCost
}

// VisualMelee describes the outcome of a melee attack, as resolved by the server.
type VisualMelee struct {
	Attacker       uint64
	Target         uint64
	Hit            bool
	Parried        bool
	Damage         int
	Braced         bool `json:",omitempty"`
	CounterDamage  int  `json:",omitempty"`
	APCost         int
	TargetPosition voxel.Int3
}

func (v VisualMelee) MessageType() string { return "Melee" }

// ApplyMelee applies a resolved melee attack to the local game state (clients).
func (g *GameInstance) ApplyMelee(msg VisualMelee) {
	attacker, attackerKnown := g.GetUnit(msg.Attacker)
	target, targetKnown := g.GetUnit(msg.Target)
	if attackerKnown {
		attacker.ConsumeAP(msg.APCost)
	}
	if g.isAuthoritative {
		return // the server already applied everything while resolving
	}
	if msg.Hit && targetKnown && target.IsActive() {
		g.ApplyDamage(attacker, target, msg.Damage, "Torso")
	}
	if msg.Parried && attackerKnown && attacker.IsActive() {
		g.ApplyDamage(target, attacker, msg.CounterDamage, "Torso")
	}
}
