package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
	"github.com/solarlune/gocoro"
	"time"
)

type ActorHitBehavior struct {
	unit                 *Unit
	forwardAfterHit      mgl32.Quat
	hitAnimationFinished bool
	lerper               *util.Lerper[mgl32.Quat]
	coroutine            gocoro.Coroutine
}

func (a *ActorHitBehavior) GetName() AnimationStateName {
	return ActorStateHit
}

func (a *ActorHitBehavior) Init(actor *Unit) {
	a.unit = actor
	a.coroutine = gocoro.NewCoroutine()
	should(a.coroutine.Run(a.GetHitScript))
}

func (a *ActorHitBehavior) Execute(deltaTime float64) TransitionEvent {
	if a.lerper != nil && !a.lerper.IsDone() {
		a.lerper.Update(deltaTime)
		return EventNone
	} else if a.coroutine.Running() {
		a.coroutine.Update()
		return EventNone
	}

	return EventAnimationFinished
}

func (a *ActorHitBehavior) GetHitScript(exe *gocoro.Execution) {
	direction := a.unit.hitInfo.ForceOfImpact.Normalize().Mul(-1)
	a.unit.turnToDirectionForAnimation(direction)
	util.LogGlobalUnitDebug(fmt.Sprintf("[ActorHitBehavior] Start hit script for %d (%v)", a.unit.UnitID(), direction))

	a.unit.GetModel().SetAnimation(game.AnimationHit.Str(), 1.0)
	should(exe.YieldFunc(a.unit.GetModel().IsHoldingAnimation))

	should(exe.YieldTime(time.Millisecond * 500))

	a.forwardAfterHit = a.unit.GetClientOnlyRotation()
	a.lerper = NewForwardLerper(a.unit, a.forwardAfterHit, mgl32.QuatIdent(), 0.5)

	should(exe.YieldFunc(a.lerper.IsDone))
	a.lerper = nil
}

func NewForwardLerper(actor *Unit, start, finish mgl32.Quat, duration float64) *util.Lerper[mgl32.Quat] {
	setValue := func(v mgl32.Quat) { actor.setClientOnlyRotation(v) }
	return util.NewLerper[mgl32.Quat](util.LerpQuatMgl, setValue, start, finish, duration)
}
