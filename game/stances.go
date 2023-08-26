package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
)

type HumanoidAnimation string

func (a HumanoidAnimation) Str() string {
	return string(a)
}

const (
	AnimationIdle       HumanoidAnimation = "animation.idle"
	AnimationWeaponIdle HumanoidAnimation = "animation.weapon_idle"
	AnimationWallIdle   HumanoidAnimation = "animation.wall_idle"
	AnimationWalk       HumanoidAnimation = "animation.walk"
	AnimationWeaponWalk HumanoidAnimation = "animation.weapon_walk"
	AnimationClimb      HumanoidAnimation = "animation.climb"
	AnimationDrop       HumanoidAnimation = "animation.drop"
	AnimationDeath      HumanoidAnimation = "animation.death"
	AnimationHit        HumanoidAnimation = "animation.hit"
	AnimationDebug      HumanoidAnimation = "animation.debug"
)

type Stance int

const (
	StanceLeanWall Stance = iota
	StanceWeaponReady
)

func HumanStanceFromID(id Stance) HumanoidStance {
	switch id {
	case StanceLeanWall:
		return LeanWall{}
	case StanceWeaponReady:
		return WeaponReady{}
	}
	println(fmt.Sprintf("[HumanStanceFromID] ERROR: Unknown stance %d", id))
	return nil
}

type HumanoidStance interface {
	GetName() string
	GetAnimation() HumanoidAnimation
	// GetOccupiedBlockOffsets returns a list of offsets units positions.
	// It expects the forward vector to be one of (0,0,1), (0,0,-1), (1,0,0), (-1,0,0)
	GetOccupiedBlockOffsets(forward voxel.Int3) []voxel.Int3
}
type LeanWall struct{}

func (s LeanWall) GetName() string {
	return "wall leaning"
}

func (s LeanWall) GetAnimation() HumanoidAnimation {
	return AnimationWallIdle
}

func (s LeanWall) GetOccupiedBlockOffsets(forward voxel.Int3) []voxel.Int3 {
	return []voxel.Int3{
		{0, 0, 0}, // legs
		{Y: 1},    // torso
	}
}

type WeaponReady struct{}

func (s WeaponReady) GetName() string {
	return "weapon ready"
}

func (s WeaponReady) GetAnimation() HumanoidAnimation {
	return AnimationWeaponIdle
}

func (s WeaponReady) GetOccupiedBlockOffsets(forward voxel.Int3) []voxel.Int3 {
	return []voxel.Int3{
		{0, 0, 0},                     // legs
		{Y: 1},                        // torso
		forward.Add(voxel.Int3{Y: 1}), // arms & weapon
	}
}
