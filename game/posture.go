package game

import "github.com/memmaker/battleground/engine/voxel"

// Posture is how low a unit holds itself. Getting lower makes a unit a smaller target and steadies its aim,
// but it also lowers the eyes, so low walls block its view too.
type Posture string

const (
	PostureStand Posture = ""
	// PostureCrouch kneels. Moving stands the unit up for free.
	PostureCrouch Posture = "Crouch"
	// PostureProne lies flat, head towards the forward block, which it also occupies.
	// A prone unit cannot move; standing up is a separate action.
	PostureProne Posture = "Prone"
)

// PostureAPCost is paid for every change of posture, except for standing up by moving out of a crouch.
const PostureAPCost = 1

type postureStats struct {
	eyeHeight    float32
	centerOfMass float32
	// spreadFactor scales the remaining spread of shots fired in this posture
	spreadFactor float64
	stance       Stance
	animation    HumanoidAnimation
}

var postures = map[Posture]postureStats{
	PostureStand:  {eyeHeight: 1.75, centerOfMass: 1.25, spreadFactor: 1, stance: StanceWeaponReady, animation: AnimationWeaponIdle},
	PostureCrouch: {eyeHeight: 1.35, centerOfMass: 0.95, spreadFactor: 0.75, stance: StanceCrouch, animation: AnimationCrouch},
	PostureProne:  {eyeHeight: 0.75, centerOfMass: 0.4, spreadFactor: 0.5, stance: StanceProne, animation: AnimationProne},
}

func (p Posture) IsValid() bool {
	_, known := postures[p]
	return known
}

func (p Posture) Name() string {
	if p == PostureStand {
		return "Stand"
	}
	return string(p)
}

// VisualStanceChange tells every player that a unit changed its posture.
// That changes what the unit sees and who sees it, so it carries a fresh LOS state.
type VisualStanceChange struct {
	UnitMessage
	Posture        Posture
	Facing         voxel.Int3 // the direction a prone unit lies in
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

// CanTakePosture tells whether the unit can switch to the posture now.
func (u *UnitInstance) CanTakePosture(posture Posture) bool {
	if posture == u.Posture || !posture.IsValid() || !u.CanAct() || u.GetIntegerAP() < PostureAPCost {
		return false
	}
	if posture != PostureStand && !u.hasAnimation(postures[posture].animation) {
		return false // monsters without a kneeling or lying pose (3x3 walkers)
	}
	if posture == PostureProne {
		return u.HasRoomToLieDown(u.GetForward2DCardinal())
	}
	return true
}

// HasRoomToLieDown checks the block the head would go to: free, not occupied and with ground below.
func (u *UnitInstance) HasRoomToLieDown(facing voxel.Int3) bool {
	voxelMap := u.GetVoxelMap()
	if voxelMap == nil {
		return false
	}
	head := u.GetBlockPosition().Add(facing)
	if !voxelMap.Contains(head.X, head.Y, head.Z) || voxelMap.IsSolidBlockAt(head.X, head.Y, head.Z) || !voxelMap.IsSolidBlockAt(head.X, head.Y-1, head.Z) {
		return false
	}
	if occupant := voxelMap.GetMapObjectAt(head); occupant != nil && occupant.(*UnitInstance) != u {
		return false
	}
	return true
}

func (u *UnitInstance) hasAnimation(animation HumanoidAnimation) bool {
	for _, name := range u.Definition.AnimationMap {
		if name == animation.Str() {
			return true
		}
	}
	return false
}

// GoProneFacing turns the unit towards a cardinal direction and lies down that way.
func (u *UnitInstance) GoProneFacing(facing voxel.Int3) {
	u.SetForward(facing)
	u.SetPosture(PostureProne)
}

// SetPosture switches the stance and re-registers the unit's blocks on the map.
func (u *UnitInstance) SetPosture(posture Posture) {
	if u.Posture == posture {
		return
	}
	u.Posture = posture
	u.CurrentStance = postures[posture].stance
	if u.voxelMap != nil {
		u.voxelMap.RemoveUnit(u)
		u.AutoSetStanceAndForwardAndUpdateMap()
	}
	u.StartStanceAnimation()
}

// OnStanceChange applies a posture change on the client side.
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
	if msg.Posture == PostureProne {
		unit.GoProneFacing(msg.Facing)
	} else {
		unit.SetPosture(msg.Posture)
	}
}

// proneOffsets: a prone unit lies across its own block and the one in front of it.
func proneOffsets(forward voxel.Int3) []voxel.Int3 {
	return []voxel.Int3{{}, forward}
}
