package client

type ActorWaitingBehavior struct {
	actor *Unit
}

func (a *ActorWaitingBehavior) Execute(deltaTime float64) TransitionEvent {
	if a.actor.shouldContinue(deltaTime) {
		return EventFinishedWaiting
	}
	return EventNone
}

func (a *ActorWaitingBehavior) GetName() AnimationStateName {
	return ActorStateWaiting
}

func (a *ActorWaitingBehavior) Init(actor *Unit) {
	a.actor = actor
	actor.StartStanceAnimation()
}
