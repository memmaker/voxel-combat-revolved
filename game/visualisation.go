package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type VisualOwnUnitMoved struct {
	UnitID      uint64
	Forward     voxel.Int3
	Path        []voxel.Int3
	EndPosition voxel.Int3
	Spotted     []*UnitInstance
	Lost        []uint64
	Cost        int
}

func (v VisualOwnUnitMoved) MessageType() string {
	return "OwnUnitMoved"
}

type VisualEnemyUnitMoved struct {
	MovingUnit uint64
	PathParts  [][]voxel.Int3

	LOSAcquiredBy []uint64
	LOSLostBy     []uint64
	UpdatedUnit   *UnitInstance // UpdatedUnit will be nil, except if the unit became visible to the player

}

func (v VisualEnemyUnitMoved) MessageType() string {
	return "EnemyUnitMoved"
}

type VisualRangedAttack struct {
	Projectiles []VisualProjectile
	WeaponType  WeaponType
	AmmoCost    uint
	Attacker    uint64
}
type VisualProjectile struct {
	Origin      mgl32.Vec3
	Destination mgl32.Vec3
	Velocity    mgl32.Vec3
	UnitHit     int64
	BodyPart    util.DamageZone
	Damage      int
	IsLethal    bool
}

func (v VisualRangedAttack) MessageType() string {
	return "RangedAttack"
}
