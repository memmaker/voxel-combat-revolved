package game

import "github.com/go-gl/mathgl/mgl32"

type ActorGotoWaypointBehavior struct {
	actor *Unit
}

func (a *ActorGotoWaypointBehavior) Execute(deltaTime float64) TransitionEvent {
	a.actor.model.UpdateAnimations(deltaTime)
	a.actor.MoveTowardsWaypoint()
	if a.actor.IsNearWaypoint() {
		a.actor.velocity = mgl32.Vec3{0, 0, 0}
		return EventNearWaypoint
	}
	return EventNone
}

func (a *ActorGotoWaypointBehavior) GetName() ActorState {
	return ActorGotoWaypoint
}

func (a *ActorGotoWaypointBehavior) Init(actor *Unit) {
	a.actor = actor
	a.actor.model.StartAnimationLoop("animation.walk", 1.0)
	a.actor.TurnTowardsWaypoint()
}
