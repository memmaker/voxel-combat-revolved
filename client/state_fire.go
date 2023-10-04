package client

import (
    "fmt"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/game"
    "github.com/solarlune/gocoro"
    "time"
)

type ActorFireBehavior struct {
    unit                  *Unit
    forwardAfterFire      mgl32.Quat
    fireAnimationFinished bool
    lerper                *util.Lerper[mgl32.Quat]
    coroutine             gocoro.Coroutine
}

func (a *ActorFireBehavior) GetName() AnimationStateName {
    return StateFireWeapon
}

func (a *ActorFireBehavior) Init(actor *Unit, event TransitionEvent) {
    a.unit = actor
    a.coroutine = gocoro.NewCoroutine()
    dirEvent := event.(DirectionalEvent)
    should(a.coroutine.Run(a.GetFireScript, dirEvent.Direction))
}

func (a *ActorFireBehavior) Execute(deltaTime float64) TransitionEvent {
    if a.lerper != nil && !a.lerper.IsDone() {
        a.lerper.Update(deltaTime)
        return NewEvent(EventNone)
    } else if a.coroutine.Running() {
        a.coroutine.Update()
        return NewEvent(EventNone)
    }

    return NewEvent(EventAnimationFinished)
}

func (a *ActorFireBehavior) GetFireScript(exe *gocoro.Execution) {
    direction := exe.Args[0].(mgl32.Vec3)
    a.unit.turnToDirectionForAnimation(direction)
    util.LogGlobalUnitDebug(fmt.Sprintf("[ActorHitBehavior] Start fire script for %d (%v)", a.unit.UnitID(), direction))

    a.unit.GetModel().SetAnimation(game.AnimationWeaponFire.Str(), 1.0)
    should(exe.YieldFunc(a.unit.GetModel().IsHoldingAnimation))

    should(exe.YieldTime(time.Millisecond * 500))

    a.forwardAfterFire = a.unit.GetClientOnlyRotation()
    a.lerper = NewForwardLerper(a.unit, a.forwardAfterFire, mgl32.QuatIdent(), 0.5)

    should(exe.YieldFunc(a.lerper.IsDone))
    a.lerper = nil
}
