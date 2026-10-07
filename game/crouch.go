package game

const (
	// CrouchAPCost is paid both to crouch and to stand up again; moving stands a unit up for free.
	CrouchAPCost = 1
	// CrouchEyeHeight replaces the standing eye height of 1.75 for line of sight and shots.
	CrouchEyeHeight = 1.35
	// CrouchSpreadFactor scales the remaining spread of shots fired from a crouch.
	CrouchSpreadFactor = 0.75
)

// VisualStanceChange tells every player that a unit crouched or stood up.
// Kneeling or rising changes what the unit sees and who sees it, so it carries a fresh LOS state.
type VisualStanceChange struct {
	UnitMessage
	Crouched       bool
	APCost         int
	LOSMatrix      map[uint64]map[uint64]bool
	PressureMatrix map[uint64]map[uint64]float64
	// Spotted lists enemies this player can now see; UpdatedUnit is set when the changed unit is visible.
	Spotted     []*UnitInstance
	UpdatedUnit *UnitInstance
}

func (v VisualStanceChange) MessageType() string {
	return "StanceChange"
}

func (u *UnitInstance) CanCrouch() bool {
	return u.CanAct() && u.GetIntegerAP() >= CrouchAPCost && u.GetStance() != nil && u.hasCrouchAnimation()
}

// hasCrouchAnimation keeps monsters without a kneeling pose (3x3 walkers) out of crouching.
func (u *UnitInstance) hasCrouchAnimation() bool {
	for _, animation := range u.Definition.AnimationMap {
		if animation == AnimationCrouch.Str() {
			return true
		}
	}
	return false
}

// SetCrouched switches the stance and re-registers the unit's blocks on the map.
func (u *UnitInstance) SetCrouched(crouched bool) {
	if u.Crouched == crouched {
		return
	}
	u.Crouched = crouched
	if !crouched {
		u.CurrentStance = StanceWeaponReady // AutoSet picks the right standing stance next
	}
	u.AutoSetStanceAndForwardAndUpdateMap()
	u.StartStanceAnimation()
}

// OnStanceChange applies a crouch or stand-up on the client side.
func (a *GameClient[U]) OnStanceChange(msg VisualStanceChange) {
	a.SetLOSAndPressure(msg.LOSMatrix, msg.PressureMatrix)
	for _, spotted := range msg.Spotted {
		a.AddOrUpdateUnit(spotted)
	}
	if msg.UpdatedUnit != nil {
		a.AddOrUpdateUnit(msg.UpdatedUnit)
	}
	unit, known := a.GetUnit(msg.UnitID())
	if !known {
		return
	}
	if a.IsMyUnit(unit.UnitID()) {
		unit.ConsumeAP(msg.APCost)
	}
	unit.SetCrouched(msg.Crouched)
}
