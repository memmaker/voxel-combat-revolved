package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type VisualOwnUnitMoved struct {
	UnitID      uint64
	Path        []voxel.Int3
	EndPosition voxel.Int3
	Spotted     []*UnitInstance
	Lost        []uint64
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

type VisualProjectileFired struct {
	Origin      mgl32.Vec3
	Velocity    mgl32.Vec3
	Destination mgl32.Vec3
	UnitHit     int64
	BodyPart    util.PartName
}

func (v VisualProjectileFired) MessageType() string {
	return "ProjectileFired"
}
