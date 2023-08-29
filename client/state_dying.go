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
	a.actor.Kill()
	direction := a.actor.hitInfo.ForceOfImpact.Normalize().Mul(-1)
	a.actor.turnToDirectionForAnimation(direction)
	a.actor.GetModel().SetAnimation(game.AnimationDeath.Str(), 1.0)
}

func (a *ActorDyingBehavior) Execute(deltaTime float64) TransitionEvent {
	finished := a.actor.GetModel().IsHoldingAnimation()
	if finished {
		return EventAnimationFinished
	}
	return EventNone
}
