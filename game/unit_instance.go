package game

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type MeshAnimation string

func (a MeshAnimation) Str() string {
	return string(a)
}

const (
	AnimationIdle       MeshAnimation = "animation.idle"
	AnimationWeaponIdle MeshAnimation = "animation.weapon_idle"
	AnimationWallIdle   MeshAnimation = "animation.wall_idle"
	AnimationWalk       MeshAnimation = "animation.walk"
	AnimationWeaponWalk MeshAnimation = "animation.weapon_walk"
	AnimationClimb      MeshAnimation = "animation.climb"
	AnimationDrop       MeshAnimation = "animation.drop"
	AnimationDeath      MeshAnimation = "animation.death"
	AnimationHit        MeshAnimation = "animation.hit"
	AnimationDebug      MeshAnimation = "animation.debug"
)

type UnitClientDefinition struct {
	TextureFile string
}

type UnitCoreStats struct {
	Health          int     // Health points, unit dies when this reaches 0
	Accuracy        float64 // Accuracy (0.0 - 1.0) will impact the aiming of the unit. At 1.0 there is no deviation from the target.
	MovementPerAP   float64 // MovementPerAP Movement per action point
	MaxActionPoints float64 // MaxActionPoints Maximum action points
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
	// serialize as json
	Transform       *util.Transform `json:"transform"`
	GameUnitID      uint64          // ID of the unit in the current game instance
	Owner           uint64          // ID of the player controlling this unit
	Name            string
	Definition      *UnitDefinition // ID of the unit definition (= unit type)
	ActionPoints    float64
	MovementPerAP   float64
	voxelMap        *voxel.Map
	model           *util.CompoundMesh
	Weapon          *Weapon
	IsDead          bool
	Health          int
	DamageZones     map[util.DamageZone]int
	MovementPenalty float64
	AimPenalty      float64
}

func (u *UnitInstance) ControlledBy() uint64 {
	return u.Owner
}

func (u *UnitInstance) UnitID() uint64 {
	return u.GameUnitID
}

func (u *UnitInstance) GetName() string {
	return u.Name
}
func (u *UnitInstance) GetBlockPosition() voxel.Int3 {
	return u.Transform.GetBlockPosition()
}
func (u *UnitInstance) GetFriendlyDescription() string {
	desc := fmt.Sprintf("x> %s HP: %d/%d AP: %d TAcc: (%0.2f)\n", u.Name, u.Health, u.Definition.CoreStats.Health, u.GetIntegerAP(), u.GetFreeAimAccuracy())
	if u.Weapon != nil {
		desc += fmt.Sprintf("x> %s Ammo: %d/%d Acc: (%0.2f)\n", u.Weapon.Definition.UniqueName, u.Weapon.AmmoCount, u.Weapon.Definition.MagazineSize, u.Weapon.Definition.AccuracyModifier)
	}
	if len(u.DamageZones) > 0 {
		desc += fmt.Sprintf("x> Damage:\n")
		for _, zone := range getDamageZones() {
			if damage, ok := u.DamageZones[zone]; ok {
				desc += fmt.Sprintf("x> %s: %d\n", zone, damage)
			}
		}
	}
	return desc
}
func (u *UnitInstance) GetEnemyDescription() string {

	desc := fmt.Sprintf("o> %s HP: %d/%d\n", u.Name, u.Health, u.Definition.CoreStats.Health)
	if u.Weapon != nil {
		desc += fmt.Sprintf("o> %s\n", u.Weapon.Definition.UniqueName)
	}
	if len(u.DamageZones) > 0 {
		desc += fmt.Sprintf("o> Damage:\n")
		for _, zone := range getDamageZones() {
			if damage, ok := u.DamageZones[zone]; ok {
				desc += fmt.Sprintf("o> %s: %d\n", zone, damage)
			}
		}
	}

	return desc
}

func (u *UnitInstance) HasActionPointsLeft() bool {
	return u.ActionPoints > 0
}
func (u *UnitInstance) CanAct() bool {
	return u.HasActionPointsLeft() && u.IsActive()
}
func (u *UnitInstance) CanSnapshot() bool {
	apNeeded := u.Weapon.Definition.BaseAPForShot
	enoughAP := u.GetIntegerAP() >= int(apNeeded)
	return u.CanAct() && u.GetWeapon().IsReady() && enoughAP
}

func (u *UnitInstance) CanFreeAim() bool {
	apNeeded := u.Weapon.Definition.BaseAPForShot + 1
	enoughAP := u.GetIntegerAP() >= int(apNeeded)
	return u.CanAct() && u.GetWeapon().IsReady() && enoughAP
}
func (u *UnitInstance) EndTurn() {
	u.ActionPoints = 0
}

func (u *UnitInstance) NextTurn() {
	u.ActionPoints = u.Definition.CoreStats.MaxActionPoints
}

func (u *UnitInstance) CanMove() bool {
	return u.HasActionPointsLeft() && u.MovesLeft() > 0 && u.IsActive()
}

func (u *UnitInstance) MovesLeft() int {
	return int(math.Floor(u.ActionPoints / u.APPerMovement()))
}

func (u *UnitInstance) APPerMovement() float64 {
	return (1.0 / u.MovementPerAP) + u.MovementPenalty
}

func (u *UnitInstance) UseMovement(cost int) {
	apCost := float64(cost) * u.APPerMovement()
	u.ActionPoints -= apCost
	println(fmt.Sprintf("[UnitInstance] %s used %d movement points, %d left", u.Name, cost, u.MovesLeft()))
	println(fmt.Sprintf("[UnitInstance] %s used %0.2f AP, %0.2f left", u.Name, apCost, u.ActionPoints))
}

func (u *UnitInstance) GetOccupiedBlockOffsets() []voxel.Int3 {
	return u.Definition.OccupiedBlockOffsets
}

func NewUnitInstance(name string, unitDef *UnitDefinition) *UnitInstance {
	compoundMesh := util.LoadGLTF(unitDef.ModelFile)
	compoundMesh.RootNode.CreateColliders()
	u := &UnitInstance{
		Transform:     util.NewDefaultTransform(name),
		Name:          name,
		Definition:    unitDef,
		ActionPoints:  unitDef.CoreStats.MaxActionPoints,
		MovementPerAP: unitDef.CoreStats.MovementPerAP,
		Health:        unitDef.CoreStats.Health,
		DamageZones:   make(map[util.DamageZone]int),
	}
	u.SetModel(compoundMesh)
	return u
}

func (u *UnitInstance) SetUnitID(id uint64) {
	u.GameUnitID = id
}
func (u *UnitInstance) SetControlledBy(playerID uint64) {
	u.Owner = playerID
}

func (u *UnitInstance) IsActive() bool {
	return !u.IsDead
}

func (u *UnitInstance) SetBlockPosition(pos voxel.Int3) {
	u.Transform.SetBlockPosition(pos)
	u.UpdateMapAndAnimation()
	//println(fmt.Sprintf("[UnitInstance] %s moved to %s", u.Name, pos.ToString()))
}

func (u *UnitInstance) SetPosition(pos mgl32.Vec3) {
	oldBlockPos := u.Transform.GetBlockPosition()
	newBlockPos := voxel.PositionToGridInt3(pos)

	u.Transform.SetPosition(pos)

	if oldBlockPos != newBlockPos { // only update if the block position has changed
		u.UpdateMapAndAnimation()
	}
	//println(fmt.Sprintf("[UnitInstance] %s moved to %v", u.Name, pos))
}

// UpdateMapAndModelAndAnimation updates the position of the unit in the voxel map and the model position and rotation.
// It will also set the animation pose on the model.
func (u *UnitInstance) UpdateMapAndAnimation() {
	u.UpdateMap()
	if u.model != nil && u.IsPlayingIdleAnimation() {
		u.UpdateAnimation()
	}
}

func (u *UnitInstance) UpdateMap() {
	u.voxelMap.SetUnit(u, u.Transform.GetBlockPosition())
}

func (u *UnitInstance) UpdateAnimation() {
	animation, newForward := GetIdleAnimationAndForwardVector(u.voxelMap, u.Transform.GetBlockPosition(), u.Transform.GetForward2DCardinal())
	//println(fmt.Sprintf("[UnitInstance] SetAnimationPose for %s(%d): %s -> %v", u.GetName(), u.UnitID(), animation.Str(), newForward))
	u.SetForward2DCardinal(newForward)
	u.model.SetAnimationLoop(animation.Str(), 1.0)
}

func (u *UnitInstance) GetEyePosition() mgl32.Vec3 {
	return u.Transform.GetBlockPosition().ToBlockCenterVec3().Add(u.GetEyeOffset())
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
	if u.model != nil {
		u.model.HideChildrenOfBoneExcept("Weapon", u.GetWeapon().Definition.Model)
	}
}

func (u *UnitInstance) GetWeapon() *Weapon {
	return u.Weapon
}

// SetForward2DCardinal sets the forward vector of the unit and rotates the model accordingly
func (u *UnitInstance) SetForward2DCardinal(forward voxel.Int3) {
	direction := forward.ToCardinalDirection()
	u.Transform.SetForward2DCardinal(direction)
}

func (u *UnitInstance) Kill() {
	u.ActionPoints = 0
	u.IsDead = true
	u.voxelMap.RemoveUnit(u)
}

func (u *UnitInstance) GetFreeAimAccuracy() float64 {
	return (u.Definition.CoreStats.Accuracy - u.AimPenalty) * u.Weapon.GetAccuracyModifier()
}

func (u *UnitInstance) SetModel(model *util.CompoundMesh) {
	if model == nil {
		return
	}
	u.model = model
	u.model.RootNode.SetParent(u.Transform)
}

func (u *UnitInstance) GetModel() *util.CompoundMesh {
	return u.model
}

func (u *UnitInstance) GetVoxelMap() *voxel.Map {
	return u.voxelMap
}

func (u *UnitInstance) GetCenterOfMassPosition() mgl32.Vec3 {
	if u.model == nil || !u.model.HasBone("Torso") {
		return u.GetPosition().Add(mgl32.Vec3{0, 1.25, 0})
	}
	return u.GetModel().GetNodeByName("Torso").GetWorldPosition()
}

func (u *UnitInstance) ApplyDamage(damage int, part util.DamageZone) bool {
	// modify hp damage
	hpDamage := damage
	if part == util.ZoneHead {
		hpDamage *= 2
	} else if part == util.ZoneWeapon {
		hpDamage = 0
	}

	if _, ok := u.DamageZones[part]; !ok {
		u.DamageZones[part] = damage
	} else {
		u.DamageZones[part] += damage
	}

	u.updatePenalties()

	u.Health -= hpDamage
	println(fmt.Sprintf("[UnitInstance] %s(%d) took %d damage to %s, Health was reduced by %d and is now %d", u.GetName(), u.UnitID(), damage, part, hpDamage, u.Health))
	if u.Health <= 0 {
		return true
	}
	return false
}

func (u *UnitInstance) updatePenalties() {
	//maxHealth := u.Definition.CoreStats.Health
	totalDamageToLegs := 0
	totalDamageToArms := 0
	totalDamageToWeapon := 0

	for part, damage := range u.DamageZones {
		if part == util.ZoneLeftLeg || part == util.ZoneRightLeg {
			totalDamageToLegs += damage
		} else if part == util.ZoneLeftArm || part == util.ZoneRightArm {
			totalDamageToArms += damage
		} else if part == util.ZoneWeapon {
			totalDamageToWeapon += damage
		}
	}

	if totalDamageToWeapon > 0 { // each point of damage to the weapon reduces accuracy by 2%
		u.Weapon.SetAccuracyPenalty((float64(totalDamageToWeapon) / 100.0) * 2)
		// TODO: destroy weapon if damage is too high
	}

	if totalDamageToLegs > 0 { // each 1 point of damage to the legs increases the AP cost of movement by 0.1
		u.MovementPenalty = float64(totalDamageToLegs) / 10.0
	}

	if totalDamageToArms > 0 { // each 5 points of damage to the arms reduces accuracy by 10%
		aimPenalty := totalDamageToArms / 5
		u.AimPenalty = float64(aimPenalty) / 10.0
	}
}

func (u *UnitInstance) GetIntegerAP() int {
	return int(math.Floor(u.ActionPoints))
}

func (u *UnitInstance) ConsumeAP(shot int) {
	u.ActionPoints -= float64(shot)
	println(fmt.Sprintf("[UnitInstance] %s(%d) consumed %d AP, %f AP left", u.GetName(), u.UnitID(), shot, u.ActionPoints))
}

func (u *UnitInstance) CanReload() bool {
	if u.GetWeapon().IsMagazineFull() {
		return false
	}
	apNeeded := u.Weapon.Definition.BaseAPForReload
	return u.GetIntegerAP() >= int(apNeeded)
}

func (u *UnitInstance) Reload() {
	apNeeded := u.Weapon.Definition.BaseAPForReload
	u.ConsumeAP(int(apNeeded))
	u.Weapon.Reload()
}

func (u *UnitInstance) GetForward2DCardinal() voxel.Int3 {
	return u.Transform.GetForward2DCardinal()
}

func (u *UnitInstance) GetPosition() mgl32.Vec3 {
	return u.Transform.GetPosition()
}

func (u *UnitInstance) IsPlayingIdleAnimation() bool {
	currentAnimation := u.model.GetAnimationName()
	return currentAnimation == AnimationWeaponIdle.Str() ||
		currentAnimation == AnimationIdle.Str() ||
		currentAnimation == AnimationWallIdle.Str()
}

func getDamageZones() []util.DamageZone {
	allZones := []util.DamageZone{util.ZoneHead, util.ZoneLeftArm, util.ZoneRightArm, util.ZoneLeftLeg, util.ZoneRightLeg, util.ZoneWeapon}
	return allZones
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
		//println(fmt.Sprintf("[UnitInstance] no wall next to %s, returning given forward vector: %s", unitPosition.ToString(), unitForward.ToString()))
		return AnimationWeaponIdle, unitForward
	} else {
		// if there is a wall next to us, we need to turn to face it
		newFront := getWallIdleDirection(solidNeighbors[0].Sub(unitPosition))
		return AnimationWallIdle, newFront
	}
}
func getWallIdleDirection(wallDirection voxel.Int3) voxel.Int3 {
	switch wallDirection {
	case voxel.NorthDir:
		return voxel.EastDir
	case voxel.EastDir:
		return voxel.SouthDir
	case voxel.SouthDir:
		return voxel.WestDir
	case voxel.WestDir:
		return voxel.NorthDir
	}
	return voxel.NorthDir
}
