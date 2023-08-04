package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type VisualUnitMoved struct {
	UnitID uint64
	Path   []voxel.Int3
}
type VisualUnitLOSUpdated struct {
	ObserverPosition voxel.Int3
	Observer         uint64
	Spotted          []*UnitInstance
	Lost             []*UnitInstance
}

type VisualProjectileFired struct {
	Origin      mgl32.Vec3
	Velocity    mgl32.Vec3
	Destination mgl32.Vec3
	UnitHit     int64
	BodyPart    util.PartName
}
