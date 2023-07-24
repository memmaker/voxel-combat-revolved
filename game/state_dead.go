package game

type ActorDeadBehavior struct {
	actor *Unit
}

func (a *ActorDeadBehavior) GetName() ActorState {
	return ActorStateDead
}

func (a *ActorDeadBehavior) Init(actor *Unit) {
	a.actor = actor
}

func (a *ActorDeadBehavior) Execute(deltaTime float64) TransitionEvent {
	return EventNone
}
