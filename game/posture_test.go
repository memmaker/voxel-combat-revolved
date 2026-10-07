package game

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

// topOfBody is the highest point of any hit collider above the unit's feet.
func topOfBody(unit *UnitInstance) float32 {
	top := float32(-100)
	for _, collider := range unit.GetColliders() {
		top = max(top, collider.FindFurthestPoint(mgl32.Vec3{0, 1, 0}).Y())
	}
	return top - unit.GetPosition().Y()
}

func newTestSoldier(t *testing.T) *UnitInstance {
	content, err := LoadContent()
	if err != nil {
		t.Fatal(err)
	}
	var soldier *UnitDefinition
	for _, faction := range content.Factions {
		for i := range faction.Units {
			if faction.Units[i].ModelFile == "soldier" && soldier == nil {
				soldier = &faction.Units[i]
			}
		}
	}
	voxelMap := voxel.NewMapWithEmptyChunks(16, 8, 16, 16, 8)
	voxelMap.SetFloorAtHeight(0, voxel.NewBlock(1))
	unit := NewUnitInstance(NewAssets(), "Soldier", soldier)
	unit.SetVoxelMap(voxelMap)
	unit.SetBlockPositionAndUpdateStance(voxel.Int3{X: 5, Y: 1, Z: 5})
	unit.StartStanceAnimation()
	return unit
}

func TestCrouchLowersTheBodyAndEyes(t *testing.T) {
	unit := newTestSoldier(t)
	if !unit.CanTakePosture(PostureCrouch) {
		t.Fatal("a soldier with full AP should be able to crouch")
	}

	standingTop, standingEyes := topOfBody(unit), unit.GetEyePosition().Y()
	unit.SetPosture(PostureCrouch)
	if unit.CurrentStance != StanceCrouch {
		t.Fatalf("stance is %d, want crouch", unit.CurrentStance)
	}
	crouchedTop, crouchedEyes := topOfBody(unit), unit.GetEyePosition().Y()
	t.Logf("top of body: standing %.2f, crouched %.2f", standingTop, crouchedTop)
	if crouchedTop > standingTop-0.25 {
		t.Errorf("crouching should lower the body by a quarter block or more: %.2f -> %.2f", standingTop, crouchedTop)
	}
	if crouchedEyes >= standingEyes {
		t.Errorf("crouching should lower the eyes: %.2f -> %.2f", standingEyes, crouchedEyes)
	}

	unit.SetPosture(PostureStand)
	if unit.CurrentStance == StanceCrouch || topOfBody(unit) < standingTop-0.01 {
		t.Error("standing up should restore the standing pose")
	}
}

func TestProneLiesAcrossTheForwardBlock(t *testing.T) {
	unit := newTestSoldier(t)
	if !unit.CanTakePosture(PostureProne) {
		t.Fatal("a soldier with full AP on open ground should be able to go prone")
	}
	standingTop := topOfBody(unit)
	facing := voxel.Int3{X: 1}
	unit.GoProneFacing(facing)
	if unit.CurrentStance != StanceProne {
		t.Fatalf("stance is %d, want prone", unit.CurrentStance)
	}
	proneTop := topOfBody(unit)
	t.Logf("top of body: standing %.2f, prone %.2f", standingTop, proneTop)
	if proneTop > 1.0 {
		t.Errorf("a prone soldier should stay below one block: %.2f", proneTop)
	}
	head := unit.GetBlockPosition().Add(facing)
	if unit.GetVoxelMap().GetMapObjectAt(head) != unit {
		t.Error("a prone soldier should occupy the block in front of it")
	}
	if unit.CanMove() {
		t.Error("a prone soldier should not be able to move")
	}

	unit.SetPosture(PostureStand)
	if unit.GetVoxelMap().GetMapObjectAt(head) == unit {
		t.Error("standing up should free the forward block")
	}
}
