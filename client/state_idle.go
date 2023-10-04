package client

type ActorIdleBehavior struct {
	actor *Unit
}

func (a *ActorIdleBehavior) GetName() AnimationStateName {
	return StateIdle
}

func (a *ActorIdleBehavior) Init(actor *Unit, event TransitionEvent) {
	a.actor = actor
	actor.StartStanceAnimation()
}

func (a *ActorIdleBehavior) Execute(deltaTime float64) TransitionEvent {
	return NewEvent(EventNone)
}
