package client

type ActorIdleBehavior struct {
	actor *Unit
}

func (a *ActorIdleBehavior) GetName() AnimationStateName {
	return ActorStateIdle
}

func (a *ActorIdleBehavior) Init(actor *Unit) {
	a.actor = actor
	actor.StartStanceAnimation()
}

func (a *ActorIdleBehavior) Execute(deltaTime float64) TransitionEvent {
	return EventNone
}
