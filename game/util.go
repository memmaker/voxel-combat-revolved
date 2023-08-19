package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type RayCastHit struct {
	util.HitInfo3D
	VisitedBlocks []voxel.Int3
	UnitHit       voxel.MapObject
	InsideMap     bool
}

func (h RayCastHit) HitUnit() bool {
	var noUnit *UnitInstance
	return h.UnitHit != noUnit && h.UnitHit != nil
}

type FreeAimHit struct {
	RayCastHit
	BodyPart util.DamageZone
	Origin   mgl32.Vec3
}
