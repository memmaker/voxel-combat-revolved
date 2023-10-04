package client

import "github.com/memmaker/battleground/game"

type ActorDyingBehavior struct {
	actor *Unit
}

func (a *ActorDyingBehavior) GetName() AnimationStateName {
	return StateDying
}

func (a *ActorDyingBehavior) Init(actor *Unit, event TransitionEvent) {
	a.actor = actor
	a.actor.Kill()
	hitEvent := event.(HitEvent)
	foi := hitEvent.ForceOfImpact
	direction := foi.Normalize().Mul(-1)
	a.actor.turnToDirectionForAnimation(direction)
	a.actor.GetModel().SetAnimation(game.AnimationDeath.Str(), 1.0)
}

func (a *ActorDyingBehavior) Execute(deltaTime float64) TransitionEvent {
	finished := a.actor.GetModel().IsHoldingAnimation()
	if finished {
		return NewEvent(EventAnimationFinished)
	}
	return NewEvent(EventNone)
}
