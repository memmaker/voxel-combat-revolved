package client

import "github.com/go-gl/mathgl/mgl32"

type UnitGotoWaypointBehavior struct {
	unit             *Unit
	waitForAnimation bool
	yOffset          int
}

func (a *UnitGotoWaypointBehavior) Execute(deltaTime float64) TransitionEvent {
	animationFinished := a.unit.model.UpdateAnimations(deltaTime)
	if a.waitForAnimation {
		if animationFinished {
			a.waitForAnimation = false
			// re-position unit to match animation position
			// reset animation position
			a.unit.model.StartAnimationLoop("animation.idle", 1.0)
			wp := a.unit.GetWaypoint()
			fp := a.unit.GetFootPosition()
			resolvedPosition := mgl32.Vec3{wp.X(), fp.Y() + float32(a.yOffset), wp.Z()}
			a.unit.SetFootPosition(resolvedPosition)
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
		a.unit.model.PlayAnimation("animation.climb", 1.0)
		a.yOffset = 1
		a.waitForAnimation = true
	} else if a.unit.IsCurrentWaypointADrop() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.model.PlayAnimation("animation.drop", 1.0)
		a.yOffset = -1
		a.waitForAnimation = true
	} else {
		a.unit.model.StartAnimationLoop("animation.walk", 1.0)
	}
}

func (a *UnitGotoWaypointBehavior) GetName() ActorState {
	return UnitGotoWaypoint
}

func (a *UnitGotoWaypointBehavior) Init(unit *Unit) {
	a.unit = unit
	a.startWaypointAnimation()
}
