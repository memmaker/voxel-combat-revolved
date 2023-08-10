package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type UnitGotoWaypointBehavior struct {
	unit             *Unit
	waitForAnimation bool
	yOffset          int32
}

func (a *UnitGotoWaypointBehavior) Execute(deltaTime float64) TransitionEvent {
	animationFinished := a.unit.model.UpdateAnimations(deltaTime)
	if a.waitForAnimation {
		if animationFinished {
			a.waitForAnimation = false
			// re-position unit to match animation position
			// reset animation position
			a.unit.StartIdleAnimationLoop()
			wp := a.unit.GetWaypoint()
			fp := a.unit.GetBlockPosition()
			resolvedPosition := voxel.Int3{X: wp.X, Y: fp.Y + a.yOffset, Z: wp.Z}
			a.unit.SetBlockPosition(resolvedPosition)
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
		a.unit.model.PlayAnimation(AnimationClimb.Str(), 1.0)
		a.yOffset = 1
		a.waitForAnimation = true
	} else if a.unit.IsCurrentWaypointADrop() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.model.PlayAnimation(AnimationDrop.Str(), 1.0)
		a.yOffset = -1
		a.waitForAnimation = true
	} else {
		a.unit.model.StartAnimationLoop(AnimationWeaponWalk.Str(), 1.0)
	}
}

func (a *UnitGotoWaypointBehavior) GetName() ActorState {
	return UnitGotoWaypoint
}

func (a *UnitGotoWaypointBehavior) Init(unit *Unit) {
	a.unit = unit
	a.startWaypointAnimation()
}
