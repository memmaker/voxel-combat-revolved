package client

import "github.com/memmaker/battleground/game"

type ActorDyingBehavior struct {
	actor *Unit
}

func (a *ActorDyingBehavior) GetName() ActorState {
	return ActorStateDying
}

func (a *ActorDyingBehavior) Init(actor *Unit) {
	a.actor = actor
	direction := a.actor.hitInfo.ForceOfImpact.Normalize().Mul(-1)
	a.actor.turnToDirectionForDeathAnimation(direction)
	a.actor.model.PlayAnimation(game.AnimationDeath.Str(), 1.0)
}

func (a *ActorDyingBehavior) Execute(deltaTime float64) TransitionEvent {
	finished := a.actor.model.UpdateAnimations(deltaTime)
	if finished {
		return EventAnimationFinished
	}
	return EventNone
}
