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
	a.actor.model.UpdateAnimations(deltaTime)
	return a.actor.GetIdleEvents()
}
