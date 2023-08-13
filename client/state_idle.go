package client

type ActorIdleBehavior struct {
	actor *Unit
}

func (a *ActorIdleBehavior) GetName() ActorState {
	return ActorStateIdle
}

func (a *ActorIdleBehavior) Init(actor *Unit) {
	a.actor = actor
	actor.StartIdleAnimationLoop()
}

func (a *ActorIdleBehavior) Execute(deltaTime float64) TransitionEvent {
	return a.actor.GetIdleEvents()
}
