package game

import (
    "fmt"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/engine/voxel"
    "math"
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

    ModelFile string
}

// IDEA: we want a stance system, where the unit can be in different stances. A stance defines which blocks are occupied.

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
    CurrentStance   Stance
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
    desc := fmt.Sprintf("> %s HP: %d/%d AP: %d TAcc: (%0.2f)\n", u.Name, u.Health, u.Definition.CoreStats.Health, u.GetIntegerAP(), u.GetFreeAimAccuracy())
    if u.Weapon != nil {
        desc += fmt.Sprintf("> %s Ammo: %d/%d Acc: (%0.2f)\n", u.Weapon.Definition.UniqueName, u.Weapon.AmmoCount, u.Weapon.Definition.MagazineSize, u.Weapon.Definition.AccuracyModifier)
    }
    if len(u.DamageZones) > 0 {
        desc += fmt.Sprintf("> Damage:\n")
        for _, zone := range getDamageZones() {
            if damage, ok := u.DamageZones[zone]; ok {
                desc += fmt.Sprintf("> %s: %d\n", zone, damage)
            }
        }
    }
    return desc
}
func (u *UnitInstance) GetEnemyDescription() string {

    desc := fmt.Sprintf("> %s HP: %d/%d\n", u.Name, u.Health, u.Definition.CoreStats.Health)
    if u.Weapon != nil {
        desc += fmt.Sprintf("> %s\n", u.Weapon.Definition.UniqueName)
    }
    if len(u.DamageZones) > 0 {
        desc += fmt.Sprintf("> Damage:\n")
        for _, zone := range getDamageZones() {
            if damage, ok := u.DamageZones[zone]; ok {
                desc += fmt.Sprintf("> %s: %d\n", zone, damage)
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
    //println(fmt.Sprintf("[UnitInstance] %s(%d) ended turn. AP=0.", u.GetName(), u.UnitID()))
    u.ActionPoints = 0
}

func (u *UnitInstance) NextTurn() {
    //println(fmt.Sprintf("[UnitInstance] %s(%d) next turn. AP=%0.2f", u.GetName(), u.UnitID(), u.Definition.CoreStats.MaxActionPoints))
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

func (u *UnitInstance) UseMovement(cost float64) {
    apCost := cost * u.APPerMovement()
    u.ActionPoints -= apCost
    //println(fmt.Sprintf("[UnitInstance] %s used %0.2f movement points, %d left", u.Name, cost, u.MovesLeft()))
    //println(fmt.Sprintf("[UnitInstance] %s used %0.2f AP for moving, %0.2f left", u.Name, apCost, u.ActionPoints))
}
func (u *UnitInstance) GetStance() HumanoidStance {
    return HumanStanceFromID(u.CurrentStance)
}
func (u *UnitInstance) GetOccupiedBlockOffsets(atPos voxel.Int3) []voxel.Int3 {
    if atPos == u.GetBlockPosition() {
        return u.GetStance().GetOccupiedBlockOffsets(u.GetForward2DCardinal())
    }
    chosenStance, chosenForward := AutoChoseStanceAndForward(u.GetVoxelMap(), u.UnitID(), atPos, u.GetForward2DCardinal())
    stance := HumanStanceFromID(chosenStance)
    return stance.GetOccupiedBlockOffsets(chosenForward)
}

func NewUnitInstance(loader *Assets, name string, unitDef *UnitDefinition) *UnitInstance {
    compoundMesh := loader.LoadMeshWithoutTextures(unitDef.ModelFile)
    compoundMesh.RootNode.CreateColliders()
    u := &UnitInstance{
        Transform:     util.NewScaledTransform(name, 1.0),
        Name:          name,
        Definition:    unitDef,
        ActionPoints:  unitDef.CoreStats.MaxActionPoints,
        MovementPerAP: unitDef.CoreStats.MovementPerAP,
        Health:        unitDef.CoreStats.Health,
        DamageZones:   make(map[util.DamageZone]int),
        CurrentStance: StanceWeaponReady,
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

func (u *UnitInstance) SetBlockPositionAndUpdateStance(pos voxel.Int3) {
    u.SetBlockPosition(pos)
    u.AutoSetStanceAndForwardAndUpdateMap()
}

func (u *UnitInstance) SetBlockPosition(pos voxel.Int3) {
    if u.Transform.GetBlockPosition() == pos {
        return
    }
    u.Transform.SetBlockPosition(pos)
}

func (u *UnitInstance) UpdateMapPosition() {
    u.voxelMap.SetUnit(u, u.Transform.GetBlockPosition())
}

func (u *UnitInstance) ForceMapPosition(pos voxel.Int3, direction voxel.Int3) {
    stance, forward := AutoChoseStanceAndForward(u.GetVoxelMap(), u.UnitID(), pos, direction)
    offsets := HumanStanceFromID(stance).GetOccupiedBlockOffsets(forward)
    u.voxelMap.SetUnitWithOffsets(u, pos, offsets)
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
    torso, exists := u.GetModel().GetNodeByName("Torso")
    if !exists {
        return u.GetPosition().Add(mgl32.Vec3{0, 1.25, 0})
    }
    return torso.GetWorldPosition()
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
    return u.Transform.GetForward2DDiagonal()
}

func (u *UnitInstance) GetPosition() mgl32.Vec3 {
    return u.Transform.GetPosition()
}

func (u *UnitInstance) SetPosition(pos mgl32.Vec3) {
    u.Transform.SetPosition(pos)
}

func (u *UnitInstance) IsPlayingIdleAnimation() bool {
    currentAnimation := u.model.GetAnimationName()
    return currentAnimation == AnimationWeaponIdle.Str() ||
        currentAnimation == AnimationIdle.Str() ||
        currentAnimation == AnimationWallIdle.Str()
}

func (u *UnitInstance) HasModel() bool {
    return u.model != nil
}

func (u *UnitInstance) DebugString(caller string) string {
    forward := u.Transform.GetForward()
    debugInfo := fmt.Sprintf("[%s] %s(%d) is at %s facing (%0.2f, %0.2f, %0.2f)", caller, u.GetName(), u.UnitID(), u.GetBlockPosition().ToString(), forward.X(), forward.Y(), forward.Z())
    debugInfo += fmt.Sprintf("\n[%s] -> Exact position is (%0.2f, %0.2f, %0.2f)", caller, u.GetPosition().X(), u.GetPosition().Y(), u.GetPosition().Z())
    if u.HasModel() {
        weapon := u.GetWeapon()
        weaponName := weapon.Definition.Model
        model := u.GetModel()
        weaponMesh, exists := model.GetNodeByName(weaponName)
        if exists {
            weaponWorldPos := weaponMesh.GetWorldPosition()
            debugInfo += fmt.Sprintf("\n[%s] -> '%s' weapon mesh world position: %0.2f, %0.2f, %0.2f", caller, weaponName, weaponWorldPos.X(), weaponWorldPos.Y(), weaponWorldPos.Z())
        } else {
            debugInfo += fmt.Sprintf("\n[%s] -> Weapon mesh %s not found", caller, weaponName)
        }
    } else {
        debugInfo += fmt.Sprintf("\n[%s] -> Unit %d has no model", caller, u.UnitID())
    }

    return debugInfo
}

func (u *UnitInstance) GetExactAP() float64 {
    return u.ActionPoints
}

// this is the official way to set the forward vector
func (u *UnitInstance) SetForward(forward2d voxel.Int3) {
    currentForward := u.Transform.GetForward2DCardinal()
    if forward2d == currentForward {
        return
    }
    u.Transform.SetForward2DCardinal(forward2d)
}
func (u *UnitInstance) UpdateStanceAndForward(stance Stance, forward2d voxel.Int3) {
    currentForward := u.Transform.GetForward2DDiagonal()

    if u.CurrentStance == stance && forward2d == currentForward {
        return
    }
    util.LogUnitDebug(fmt.Sprintf("[UnitInstance] %s(%d) UpdateStanceAndForward(%d, %s)", u.GetName(), u.UnitID(), stance, forward2d.ToString()))

    u.Transform.SetForward2DDiagonal(forward2d)
    u.CurrentStance = stance
    u.StartStanceAnimation()
}

func (u *UnitInstance) StartStanceAnimation() {
    if u.HasModel() && u.IsActive() {
        u.GetModel().SetAnimationLoop(u.GetStance().GetAnimation().Str(), 1.0)
    }
}

func getDamageZones() []util.DamageZone {
    allZones := []util.DamageZone{util.ZoneHead, util.ZoneLeftArm, util.ZoneRightArm, util.ZoneLeftLeg, util.ZoneRightLeg, util.ZoneWeapon}
    return allZones
}
func (u *UnitInstance) AutoSetStanceAndForwardAndUpdateMap() {
    if !u.IsActive() {
        return
    }
    u.UpdateStanceAndForward(AutoChoseStanceAndForward(u.GetVoxelMap(), u.UnitID(), u.GetBlockPosition(), u.GetForward2DCardinal()))
    u.UpdateMapPosition()
}

func (u *UnitInstance) GetForward() voxel.Int3 {
    return u.Transform.GetForward2DDiagonal()
}

func AutoChoseStanceAndForward(voxelMap *voxel.Map, unitID uint64, unitPosition, unitForward voxel.Int3) (Stance, voxel.Int3) {
    // TODO: make this clever, needs to reacts to walls and nearby units and their stances.
    //return AnimationDebug, unitForward

    solidNeighbors := voxelMap.GetBlockedCardinalNeighborsUpToHeight(unitID, unitPosition, 2)
    if len(solidNeighbors) == 0 {
        // if no wall next to us, we can idle normally
        //println(fmt.Sprintf("[UnitInstance] no wall next to %s, returning given forward vector: %s", unitPosition.ToString(), unitForward.ToString()))
        return StanceWeaponReady, unitForward.ToCardinalDirection()
    } else {
        // if there is a wall next to us, we need to turn to face it
        newFront := getWallIdleDirection(solidNeighbors[0].Sub(unitPosition))
        return StanceLeanWall, newFront
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
