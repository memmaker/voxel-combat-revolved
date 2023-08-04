package game

import (
	"github.com/memmaker/battleground/engine/voxel"
)

// what is the client side of a unit action?
// the selection mode (aoe, single target, fps free aim, etc)
// ---> these actions are all single target..
// the valid targets (locations)
// the status of the action (uses, cooldown, etc)
// ---> isn't this just asking if the action is applicable to the unit?

type TargetAction interface {
	GetName() string
	// GetValidTargets returns a list of valid targets for the given unit and action.
	// It should work for both the server and the client
	GetValidTargets(unit UnitCore) []voxel.Int3
	IsValidTarget(unit UnitCore, target voxel.Int3) bool
}
