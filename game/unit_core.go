package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type UnitCore interface {
	GetFootPosition() mgl32.Vec3
	GetName() string
	SetPath(path []voxel.Int3)
	MovesLeft() int
	GetEyePosition() mgl32.Vec3
	GetPosition() mgl32.Vec3
	SetPosition(pos mgl32.Vec3)
	SetFootPosition(pos mgl32.Vec3)
	SetBlockPosition(pos voxel.Int3)
	UnitID() uint64
	ControlledBy() uint64
	GetOccupiedBlockOffsets() []voxel.Int3
}
type UnitClientDefinition struct {
	TextureFile string
}

type UnitCoreStats struct {
	Health int
	Speed  int
}

type UnitDefinition struct {
	ID uint64 // ID of the unit definition (= unit type)

	ClientRepresentation UnitClientDefinition
	CoreStats            UnitCoreStats

	ModelFile            string
	OccupiedBlockOffsets []voxel.Int3
}

type UnitInstance struct {
	GameUnitID     uint64 // ID of the unit in the current game instance
	controlledBy   uint64 // ID of the player controlling this unit
	Name           string
	Position       voxel.Int3
	Orientation    [2]int32
	UnitDefinition *UnitDefinition // ID of the unit definition (= unit type)
	canAct         bool
	voxelMap       *voxel.Map
	model          *util.CompoundMesh
	Weapon         string
}

func (u *UnitInstance) SetPath(path []voxel.Int3) {
	u.SetBlockPosition(path[len(path)-1])
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
	return u.UnitDefinition.CoreStats.Speed
}

func (u *UnitInstance) GetOccupiedBlockOffsets() []voxel.Int3 {
	return u.UnitDefinition.OccupiedBlockOffsets
}

func NewUnitInstance(name string, unitDef *UnitDefinition) *UnitInstance {
	compoundMesh := util.LoadGLTF(unitDef.ModelFile)
	compoundMesh.RootNode.CreateColliders()
	return &UnitInstance{
		Name:           name,
		UnitDefinition: unitDef,
		canAct:         true,
		model:          compoundMesh, // todo: cache models?
	}
}

func (u *UnitInstance) SetGameUnitID(id uint64) {
	u.GameUnitID = id
}
func (u *UnitInstance) SetControlledBy(playerID uint64) {
	u.controlledBy = playerID
}

func (u *UnitInstance) IsActive() bool {
	return true
}

func (u *UnitInstance) NextTurn() {
	u.canAct = true
}

func (u *UnitInstance) GetPosition() mgl32.Vec3 {
	return u.Position.ToBlockCenterVec3().Add(mgl32.Vec3{0, 1, 0})
}

func (u *UnitInstance) SetPosition(pos mgl32.Vec3) {
	footPosition := pos.Sub(mgl32.Vec3{0, 1, 0})
	u.SetBlockPosition(voxel.ToGridInt3(footPosition))
}

func (u *UnitInstance) SetFootPosition(pos mgl32.Vec3) {
	u.SetBlockPosition(voxel.ToGridInt3(pos))
}

func (u *UnitInstance) updateMapAndModelPosition(old voxel.Int3) {
	worldPos := u.Position.ToBlockCenterVec3()
	u.model.RootNode.Translate([3]float32{worldPos[0], worldPos[1], worldPos[2]})
	u.voxelMap.MoveUnitTo(u, old, u.Position)
}
func (u *UnitInstance) GetEyePosition() mgl32.Vec3 {
	return u.Position.ToBlockCenterVec3().Add(u.GetEyeOffset())
}

func (u *UnitInstance) GetFootPosition() mgl32.Vec3 {
	return u.Position.ToBlockCenterVec3()
}

func (u *UnitInstance) SetBlockPosition(pos voxel.Int3) {
	oldPos := u.Position
	u.Position = pos
	u.updateMapAndModelPosition(oldPos)
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

func (u *UnitInstance) SetWeapon(weapon string) {
	u.Weapon = weapon
}

func (u *UnitInstance) GetWeapon() string {
	return u.Weapon
}

func (u *UnitInstance) SetForward(forward voxel.Int3) {
	u.Orientation[0] = forward.X
	u.Orientation[1] = forward.Z
}

func (u *UnitInstance) GetForward() voxel.Int3 {
	return voxel.Int3{X: u.Orientation[0], Z: u.Orientation[1]}
}
