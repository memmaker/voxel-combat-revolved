package game

import "github.com/memmaker/battleground/engine/voxel"

const (
	HealAPCost = 2
	HealAmount = 4
)

// ActionHeal patches up an adjacent wounded ally (or the medic itself).
type ActionHeal struct {
	unit  *UnitInstance
	valid map[voxel.Int3]bool
}

func NewActionHeal(engine *GameInstance, unit *UnitInstance) *ActionHeal {
	a := &ActionHeal{unit: unit, valid: make(map[voxel.Int3]bool)}
	for _, ally := range engine.GetPlayerUnits(unit.ControlledBy()) {
		if !ally.IsActive() || ally.Health >= ally.Definition.CoreStats.Health {
			continue
		}
		if ally == unit || IsInMeleeRange(unit.GetBlockPosition(), ally.GetBlockPosition()) {
			a.valid[ally.GetBlockPosition()] = true
		}
	}
	return a
}

func (a *ActionHeal) GetName() string { return "Heal" }

func (a *ActionHeal) GetValidTargets() []voxel.Int3 {
	result := make([]voxel.Int3, 0, len(a.valid))
	for pos := range a.valid {
		result = append(result, pos)
	}
	return result
}

func (a *ActionHeal) IsValidTarget(target voxel.Int3) bool { return a.valid[target] }

func (a *ActionHeal) IsTurnEnding() bool { return false }

func (u *UnitInstance) HealsLeft() int {
	return u.Definition.CoreStats.HealCharges - u.HealsUsed
}

func (u *UnitInstance) CanHeal() bool {
	return u.HealsLeft() > 0 && u.CanAct() && u.GetIntegerAP() >= HealAPCost
}

// Heal restores health, at most up to the unit's maximum, and returns the amount healed.
func (u *UnitInstance) Heal(amount int) int {
	healed := min(amount, u.Definition.CoreStats.Health-u.Health)
	u.Health += healed
	return healed
}

type VisualHeal struct {
	Healer uint64
	Target uint64
	Amount int
}

func (v VisualHeal) MessageType() string { return "Heal" }

// ApplyHeal applies a heal resolved by the server to the local state (clients).
func (g *GameInstance) ApplyHeal(msg VisualHeal) {
	if healer, known := g.GetUnit(msg.Healer); known {
		healer.ConsumeAP(HealAPCost)
		healer.HealsUsed++
	}
	if target, known := g.GetUnit(msg.Target); known {
		target.Heal(msg.Amount)
	}
}
