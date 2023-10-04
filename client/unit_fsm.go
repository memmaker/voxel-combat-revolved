package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
)

func NewActorTransitionTable() *TransitionTable {
	t := NewTransitionTable()

	// waypoints
	t.AddTransition(StateIdle, EventNewPath, StateGotoWaypoint)
	t.AddTransition(StateGotoWaypoint, EventLastWaypointReached, StateIdle)
	//t.AddTransition(StateWaiting, EventFinishedWaiting, StateIdle)

	// firing
	t.AddTransition(StateIdle, EventFireWeapon, StateFireWeapon)
	t.AddTransition(StateFireWeapon, EventFireWeapon, StateFireWeapon)
	t.AddTransition(StateFireWeapon, EventAnimationFinished, StateIdle)

	// hits
	t.AddTransition(StateIdle, EventHit, StateHit)
	t.AddTransition(StateHit, EventHit, StateHit)
	t.AddTransition(StateHit, EventAnimationFinished, StateIdle)

	// dying & death
	t.AddTransition(StateIdle, EventLethalHit, StateDying)
	t.AddTransition(StateHit, EventLethalHit, StateDying)
	t.AddTransition(StateDying, EventAnimationFinished, StateDead)

	return t
}

var ActorTransitionTable = NewActorTransitionTable()

type TransitionEvent interface {
	Name() TransitionEventName
}

type EmptyEvent struct {
	name TransitionEventName
}

func (e EmptyEvent) Name() TransitionEventName {
	return e.name
}

type DirectionalEvent struct {
	name      TransitionEventName
	Direction mgl32.Vec3
}

func (d DirectionalEvent) Name() TransitionEventName {
	return d.name
}

type HitEvent struct {
	name          TransitionEventName
	ForceOfImpact mgl32.Vec3
	BodyPart      util.DamageZone
}

func (h HitEvent) Name() TransitionEventName {
	return h.name
}

func NewEvent(name TransitionEventName) TransitionEvent {
	return EmptyEvent{name}
}

func NewHitEvent(name TransitionEventName, forceOfImpact mgl32.Vec3, bodyPart util.DamageZone) TransitionEvent {
	return HitEvent{name, forceOfImpact, bodyPart}
}

func NewDirectionalEvent(name TransitionEventName, direction mgl32.Vec3) TransitionEvent {
	return DirectionalEvent{name, direction}
}
func (e TransitionEventName) ToString() string {
	switch e {
	case EventNone:
		return "None"
	case EventNewPath:
		return "NewPath"
	case EventFinishedWaiting:
		return "FinishedWaiting"
	case EventHit:
		return "Hit"
	case EventLethalHit:
		return "LethalHit"
	case EventAnimationFinished:
		return "AnimationFinished"
	case EventLastWaypointReached:
		return "LastWaypointReached"
	case EventWaypointReached:
		return "WaypointReached"
	default:
		return "Unknown"
	}
}

type TransitionEventName int

const (
	EventNone TransitionEventName = iota
	EventNewPath
	EventFinishedWaiting
	EventHit
	EventFireWeapon
	EventLethalHit
	EventAnimationFinished
	EventLastWaypointReached
	EventWaypointReached
)

type AnimationStateName int

func (s AnimationStateName) ToString() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateWaiting:
		return "Waiting"
	case StateGotoWaypoint:
		return "GotoWaypoint"
	case StateFireWeapon:
		return "FireWeapon"
	case StateDying:
		return "Dying"
	case StateDead:
		return "Dead"
	case StateHit:
		return "Hit"
	default:
		return "Unknown"
	}
}

const (
	StateIdle AnimationStateName = iota
	StateWaiting
	StateGotoWaypoint
	StateFireWeapon
	StateDying
	StateHit
	StateDead
	// Also change NewTransitionTable() below, if you add new states at the end or the beginning
)

type TransitionTable map[AnimationStateName]map[TransitionEventName]AnimationStateName

func NewTransitionTable() *TransitionTable {
	t := make(TransitionTable)
	for state := StateIdle; state <= StateDead; state++ {
		t[state] = make(map[TransitionEventName]AnimationStateName)
	}
	return &t
}

func (t *TransitionTable) AddTransition(fromState AnimationStateName, event TransitionEventName, toState AnimationStateName) {
	(*t)[fromState][event] = toState
}

func (t *TransitionTable) AddTransitionFromAllExcept(excludedStates []AnimationStateName, event TransitionEventName, toState AnimationStateName) {
	for state := range *t {
		if !contains(excludedStates, state) {
			(*t)[state][event] = toState
		}
	}
}

func (t *TransitionTable) Exists(currentState AnimationStateName, event TransitionEventName) bool {
	_, ok := (*t)[currentState][event]
	return ok
}

func (t *TransitionTable) GetNextState(currentState AnimationStateName, event TransitionEventName) AnimationStateName {
	return (*t)[currentState][event]
}

func contains(states []AnimationStateName, state AnimationStateName) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

type AnimationState interface {
	GetName() AnimationStateName
	Init(actor *Unit, event TransitionEvent)
	Execute(deltaTime float64) TransitionEvent
}

var BehaviorTable = map[AnimationStateName]func() AnimationState{
	StateIdle:         func() AnimationState { return &ActorIdleBehavior{} },
	StateGotoWaypoint: func() AnimationState { return &UnitGotoWaypointBehavior{} },
	StateWaiting:      func() AnimationState { return &ActorWaitingBehavior{} },
	StateHit:          func() AnimationState { return &ActorHitBehavior{} },
	StateFireWeapon:   func() AnimationState { return &ActorFireBehavior{} },
	StateDying:        func() AnimationState { return &ActorDyingBehavior{} },
	StateDead:         func() AnimationState { return &ActorDeadBehavior{} },
}

func BehaviorFactory(state AnimationStateName) AnimationState {
	return BehaviorTable[state]()
}
