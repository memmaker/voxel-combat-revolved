package client

type ActorDeadBehavior struct {
	actor *Unit
}

func (a *ActorDeadBehavior) GetName() AnimationStateName {
	return StateDead
}

func (a *ActorDeadBehavior) Init(actor *Unit, event TransitionEvent) {
	a.actor = actor
}

func (a *ActorDeadBehavior) Execute(deltaTime float64) TransitionEvent {
	return NewEvent(EventNone)
}
