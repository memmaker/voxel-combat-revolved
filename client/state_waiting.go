package client

type ActorWaitingBehavior struct {
	actor *Unit
}

func (a *ActorWaitingBehavior) Execute(deltaTime float64) TransitionEvent {
	if a.actor.shouldContinue(deltaTime) {
		return NewEvent(EventFinishedWaiting)
	}
	return NewEvent(EventNone)
}

func (a *ActorWaitingBehavior) GetName() AnimationStateName {
	return StateWaiting
}

func (a *ActorWaitingBehavior) Init(actor *Unit, event TransitionEvent) {
	a.actor = actor
	actor.StartStanceAnimation()
}
