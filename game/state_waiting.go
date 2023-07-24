package game

type ActorWaitingBehavior struct {
	actor *Unit
}

func (a *ActorWaitingBehavior) Execute(deltaTime float64) TransitionEvent {
	a.actor.model.UpdateAnimations(deltaTime)
	if a.actor.shouldContinue(deltaTime) {
		return EventFinishedWaiting
	}
	return EventNone
}

func (a *ActorWaitingBehavior) GetName() ActorState {
	return ActorStateWaiting
}

func (a *ActorWaitingBehavior) Init(actor *Unit) {
	a.actor = actor
	actor.model.StartAnimationLoop("animation.idle", 0.25)
}
