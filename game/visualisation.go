package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type VisualOwnUnitMoved struct {
	UnitID         uint64
	Forward        voxel.Int3
	Path           []voxel.Int3
	EndPosition    voxel.Int3
	Spotted        []*UnitInstance
	LOSMatrix      map[uint64]map[uint64]bool
	PressureMatrix map[uint64]map[uint64]float64
	Cost           float64
}

func (v VisualOwnUnitMoved) MessageType() string {
	return "OwnUnitMoved"
}

type VisualEnemyUnitMoved struct {
	MovingUnit uint64
	PathParts  [][]voxel.Int3

	LOSMatrix      map[uint64]map[uint64]bool
	PressureMatrix map[uint64]map[uint64]float64
	UpdatedUnit    *UnitInstance // UpdatedUnit will be nil, except if the unit became visible to the player

}

func (v VisualEnemyUnitMoved) MessageType() string {
	return "EnemyUnitMoved"
}

type TargetedEffect string

const (
	TargetedEffectSmokeCloud  TargetedEffect = "SmokeCloud"
	TargetedEffectPoisonCloud TargetedEffect = "PoisonCloud"
	TargetedEffectFire        TargetedEffect = "Fire"
	TargetedEffectExplosion   TargetedEffect = "Explosion"
)

type BlockStatusEffectInstance struct {
	Effect BlockEffect
	Turns  int
}
type BlockStatusEffect int

const (
	BlockStatusEffectBlocksLOS BlockStatusEffect = 1 << iota
	BlockStatusEffectBlocksMovement
	BlockStatusEffectDamagesOnTouch
)

type BlockEffect int

const (
	BlockEffectSmoke BlockEffect = 1 << iota
	BlockEffectPoison
	BlockEffectFire
)

type MessageTargetedEffect struct {
	Position    voxel.Int3
	Effect      TargetedEffect
	TurnsToLive int
	Radius      float64
}

type VisualFlightWithImpact struct {
	Trajectory    []mgl32.Vec3
	FinalWorldPos mgl32.Vec3
	Consequence   MessageTargetedEffect
	VisitedBlocks []voxel.Int3
}

type VisualThrow struct {
	UnitID            uint64
	Forward           voxel.Int3
	Flyers            []VisualFlightWithImpact
	IsTurnEnding      bool
	APCostForAttacker int
	AmmoCost          uint
}

func (v VisualThrow) MessageType() string {
	return "Throw"
}
type VisualRangedAttack struct {
	Projectiles       []VisualProjectile
	WeaponType        WeaponType
	AmmoCost          uint
	Attacker          uint64
	AimDirection      mgl32.Vec3
	APCostForAttacker int
	IsTurnEnding      bool
}
type VisualProjectile struct {
	Origin        mgl32.Vec3
	Destination   mgl32.Vec3
	Velocity      mgl32.Vec3
	UnitHit       int64
	BodyPart      util.DamageZone
	Damage        int
	IsLethal      bool
	BlocksHit     []voxel.Int3 // BlocksHit will only contain blocks that have an OnDamageReceived effect
	VisitedBlocks []voxel.Int3 // VisitedBlocks will contain all blocks that the projectile passed through
}

func (v VisualRangedAttack) MessageType() string {
	return "RangedAttack"
}

type VisualBeginOverwatch struct {
	Watcher          uint64
	WatchedLocations []voxel.Int3
	APCost           int
}

func (v VisualBeginOverwatch) MessageType() string {
	return "BeginOverwatch"
}
