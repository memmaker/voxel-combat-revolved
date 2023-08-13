package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type UnitGotoWaypointBehavior struct {
	unit             *Unit
	waitForAnimation bool
	yOffset          int32
}

func (a *UnitGotoWaypointBehavior) Execute(deltaTime float64) TransitionEvent {
	animationFinished := a.unit.GetModel().IsHoldingAnimation()
	if a.waitForAnimation {
		if animationFinished {
			a.waitForAnimation = false
			// re-position unit to match animation position
			// reset animation position
			wp := a.unit.GetWaypoint()
			fp := a.unit.GetBlockPosition()
			resolvedPosition := voxel.Int3{X: wp.X, Y: fp.Y + a.yOffset, Z: wp.Z}
			a.unit.SetBlockPositionAndUpdateMapAndModel(resolvedPosition)
		} else {
			return EventNone
		}
	}

	if a.unit.HasReachedWaypoint() {
		return a.onWaypointReached()
	} else {
		a.unit.MoveTowardsWaypoint()
	}
	return EventNone
}

func (a *UnitGotoWaypointBehavior) onWaypointReached() TransitionEvent {
	if a.unit.IsLastWaypoint() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		return EventLastWaypointReached
	}

	a.unit.NextWaypoint()
	a.startWaypointAnimation()
	return EventWaypointReached
}

func (a *UnitGotoWaypointBehavior) startWaypointAnimation() {
	a.unit.TurnTowardsWaypoint()
	if a.unit.IsCurrentWaypointAClimb() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.GetModel().SetAnimation(game.AnimationClimb.Str(), 1.0)
		a.yOffset = 1
		a.waitForAnimation = true
	} else if a.unit.IsCurrentWaypointADrop() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.GetModel().SetAnimation(game.AnimationDrop.Str(), 1.0)
		a.yOffset = -1
		a.waitForAnimation = true
	} else {
		a.unit.GetModel().SetAnimationLoop(game.AnimationWeaponWalk.Str(), 1.0)
	}
}

func (a *UnitGotoWaypointBehavior) GetName() ActorState {
	return UnitGotoWaypoint
}

func (a *UnitGotoWaypointBehavior) Init(unit *Unit) {
	a.unit = unit
	a.startWaypointAnimation()
}
