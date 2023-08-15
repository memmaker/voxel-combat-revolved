package game

import (
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
	return h.UnitHit != nil
}

type FreeAimHit struct {
	RayCastHit
	BodyPart util.DamageZone
}
