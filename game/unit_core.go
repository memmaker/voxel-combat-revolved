package game

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type MeshAnimation string

func (a MeshAnimation) Str() string {
	return string(a)
}

const (
	AnimationIdle       MeshAnimation = "animation.idle"
	AnimationGunIdle    MeshAnimation = "animation.weapon_idle"
	AnimationWallIdle   MeshAnimation = "animation.wall_idle"
	AnimationWalk       MeshAnimation = "animation.walk"
	AnimationWeaponWalk MeshAnimation = "animation.weapon_walk"
	AnimationClimb      MeshAnimation = "animation.climb"
	AnimationDrop       MeshAnimation = "animation.drop"
	AnimationDeath      MeshAnimation = "animation.death"
	AnimationDebug      MeshAnimation = "animation.debug"
)

type UnitCore interface {
	GetName() string
	MovesLeft() int
	GetEyePosition() mgl32.Vec3
	SetBlockPositionAndUpdateMapAndModel(pos voxel.Int3)
	GetBlockPosition() voxel.Int3
	UnitID() uint64
	ControlledBy() uint64
	GetOccupiedBlockOffsets() []voxel.Int3
}
type UnitClientDefinition struct {
	TextureFile string
}

type UnitCoreStats struct {
	Health   int     // Health points, unit dies when this reaches 0
	Speed    int     // Speed is the number of grid cells the unit can move per turn
	Accuracy float64 // Accuracy (0.0 - 1.0) will impact the aiming of the unit. At 1.0 there is no deviation from the target.
}

// UnitDefinition is the definition of a unit type. It contains the static information about the unit type.
// This is a basic unit archetype, from which the player chooses.
type UnitDefinition struct {
	ID uint64 // ID of the unit definition (= unit type)

	ClientRepresentation UnitClientDefinition
	CoreStats            UnitCoreStats

	ModelFile            string
	OccupiedBlockOffsets []voxel.Int3
}

// UnitInstance is an instance of an unit on the battlefield. It contains the dynamic information about the unit.
type UnitInstance struct {
	GameUnitID    uint64 // ID of the unit in the current game instance
	controlledBy  uint64 // ID of the player controlling this unit
	Name          string
	Position      voxel.Int3
	Definition    *UnitDefinition // ID of the unit definition (= unit type)
	canAct        bool
	movesLeft     int
	voxelMap      *voxel.Map
	model         *util.CompoundMesh
	Weapon        *Weapon
	ForwardVector voxel.Int3
	isDead        bool
	health        int
}

func (u *UnitInstance) ControlledBy() uint64 {
	return u.controlledBy
}

func (u *UnitInstance) UnitID() uint64 {
	return u.GameUnitID
}

func (u *UnitInstance) GetName() string {
	return u.Name
}

func (u *UnitInstance) MovesLeft() int {
	return u.movesLeft
}

func (u *UnitInstance) UseMovement(cost int) {
	u.movesLeft -= cost
}

func (u *UnitInstance) GetOccupiedBlockOffsets() []voxel.Int3 {
	return u.Definition.OccupiedBlockOffsets
}

func NewUnitInstance(name string, unitDef *UnitDefinition) *UnitInstance {
	compoundMesh := util.LoadGLTF(unitDef.ModelFile)
	compoundMesh.RootNode.CreateColliders()
	return &UnitInstance{
		Name:       name,
		Definition: unitDef,
		canAct:     true,
		movesLeft:  unitDef.CoreStats.Speed,
		health:     unitDef.CoreStats.Health,
		model:      compoundMesh, // todo: cache models?
	}
}

func (u *UnitInstance) SetGameUnitID(id uint64) {
	u.GameUnitID = id
}
func (u *UnitInstance) SetControlledBy(playerID uint64) {
	u.controlledBy = playerID
}

func (u *UnitInstance) IsActive() bool {
	return !u.isDead
}

func (u *UnitInstance) NextTurn() {
	u.canAct = true
	u.movesLeft = u.Definition.CoreStats.Speed
}

// UpdateMapAndModelPosition updates the position of the unit in the voxel map and the model position and rotation.
// It will also set the animation pose on the model.
func (u *UnitInstance) UpdateMapAndModelPosition() {
	u.voxelMap.SetUnit(u, u.Position)
	if u.model != nil {
		u.UpdateModelPositionAndRotation()
	}
	//println(u.model.GetAnimationDebugString())
}

func (u *UnitInstance) UpdateModelPositionAndRotation() {
	worldPos := u.Position.ToBlockCenterVec3()
	u.model.RootNode.Translate([3]float32{worldPos[0], worldPos[1], worldPos[2]})
	println(fmt.Sprintf("[UnitInstance] Moved %s(%d) to %v facing %v", u.GetName(), u.UnitID(), u.Position, u.ForwardVector))
	animation, newForward := GetIdleAnimationAndForwardVector(u.voxelMap, u.Position, u.ForwardVector)
	println(fmt.Sprintf("[UnitInstance] SetAnimationPose for %s(%d): %s -> %v", u.GetName(), u.UnitID(), animation.Str(), newForward))
	u.SetForward(newForward)
	u.model.SetAnimationLoop(animation.Str(), 1.0)
}
func (u *UnitInstance) GetEyePosition() mgl32.Vec3 {
	return u.Position.ToBlockCenterVec3().Add(u.GetEyeOffset())
}

func (u *UnitInstance) GetFootPosition() mgl32.Vec3 {
	return u.Position.ToBlockCenterVec3()
}

// SetBlockPositionAndUpdateMapAndModel sets the position of the unit in the voxel map.
// NOTE: THIS WILL CALL updateMapAndModelPosition.
// CALLING THIS WILL FREEZE THE ANIMATION OF THE UNIT.
func (u *UnitInstance) SetBlockPositionAndUpdateMapAndModel(pos voxel.Int3) {
	u.Position = pos
	u.UpdateMapAndModelPosition()
}

func (u *UnitInstance) SetBlockPositionAndUpdateMap(pos voxel.Int3) {
	u.Position = pos
	u.voxelMap.SetUnit(u, u.Position)
}

func (u *UnitInstance) GetBlockPosition() voxel.Int3 {
	return u.Position
}

func (u *UnitInstance) CanAct() bool {
	return u.canAct && u.IsActive()
}

func (u *UnitInstance) EndTurn() {
	u.canAct = false
}

func (u *UnitInstance) SetVoxelMap(voxelMap *voxel.Map) {
	u.voxelMap = voxelMap
}

func (u *UnitInstance) GetEyeOffset() mgl32.Vec3 {
	return mgl32.Vec3{0, 1.75, 0}
}

func (u *UnitInstance) GetColliders() []util.Collider {
	return u.model.RootNode.GetColliders()
}

func (u *UnitInstance) SetWeapon(weapon *Weapon) {
	u.Weapon = weapon
}

func (u *UnitInstance) GetWeapon() *Weapon {
	return u.Weapon
}

// SetForward sets the forward vector of the unit and rotates the model accordingly
func (u *UnitInstance) SetForward(forward voxel.Int3) {
	//println(fmt.Sprintf("[UnitInstance] SetForward for %s(%d): %v", u.GetName(), u.UnitID(), forward))
	u.ForwardVector = forward
	u.model.SetYRotationAngle(util.DirectionToAngle(forward))
}

func (u *UnitInstance) GetForward() voxel.Int3 {
	return u.ForwardVector
}

func (u *UnitInstance) Kill() {
	u.canAct = false
	u.isDead = true
	u.voxelMap.RemoveUnit(u)
}

func (u *UnitInstance) GetFreeAimAccuracy() float64 {
	return u.Definition.CoreStats.Accuracy * u.Weapon.Definition.AccuracyModifier
}

func (u *UnitInstance) SetModel(model *util.CompoundMesh) {
	u.model = model
}

func (u *UnitInstance) GetModel() *util.CompoundMesh {
	return u.model
}

func (u *UnitInstance) GetVoxelMap() *voxel.Map {
	return u.voxelMap
}

func (u *UnitInstance) GetCenterOfMassPosition() mgl32.Vec3 {
	return u.Position.Add(voxel.Int3{Y: 1}).ToBlockCenterVec3()
}

func (u *UnitInstance) ApplyDamage(damage int, part util.PartName) bool {
	u.health -= damage
	println(fmt.Sprintf("[UnitInstance] %s(%d) took %d damage to %s, health is now %d", u.GetName(), u.UnitID(), damage, part, u.health))
	if u.health <= 0 {
		return true
	}
	return false
}

func GetIdleAnimationAndForwardVector(voxelMap *voxel.Map, unitPosition, unitForward voxel.Int3) (MeshAnimation, voxel.Int3) {
	//return AnimationDebug, unitForward
	solidNeighbors := voxelMap.GetNeighborsForGroundMovement(unitPosition, func(neighbor voxel.Int3) bool {
		if neighbor != unitPosition && voxelMap.IsSolidBlockAt(neighbor.X, neighbor.Y, neighbor.Z) {
			return true
		}
		return false
	})
	if len(solidNeighbors) == 0 {
		// if no wall next to us, we can idle normally
		return AnimationGunIdle, unitForward
	} else {
		// if there is a wall next to us, we need to turn to face it
		newFront := getWallIdleDirection(solidNeighbors[0].Sub(unitPosition))
		return AnimationWallIdle, newFront
	}
}
func getWallIdleDirection(wallDirection voxel.Int3) voxel.Int3 {
	switch wallDirection {
	case voxel.North:
		return voxel.East
	case voxel.East:
		return voxel.South
	case voxel.South:
		return voxel.West
	case voxel.West:
		return voxel.North
	}
	return voxel.North
}
