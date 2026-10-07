package client

import "github.com/go-gl/mathgl/mgl32"

type ActorIdleBehavior struct {
	actor *Unit
}

func (a *ActorIdleBehavior) GetName() AnimationStateName {
	return StateIdle
}

func (a *ActorIdleBehavior) Init(actor *Unit, event TransitionEvent) {
	a.actor = actor
	actor.SetVelocity(mgl32.Vec3{}) // an idle unit must never keep drifting with a leftover walking velocity
	actor.StartStanceAnimation()
}

func (a *ActorIdleBehavior) Execute(deltaTime float64) TransitionEvent {
	return NewEvent(EventNone)
}
