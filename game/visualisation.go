package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type VisualUnitMoved struct {
	UnitID uint64
	Path   []voxel.Int3
}
type VisualUnitsSpotted struct {
	ObserverPosition voxel.Int3
	Observer         uint64
	Spotted          []*UnitInstance
}

type VisualProjectileFired struct {
	SourcePosition mgl32.Vec3
	Velocity       mgl32.Vec3
}
