package client

// state transition table
// currentState, event, nextState

// idle, newWaypoint, gotoWaypoint
// gotoWaypoint, nearWaypoint, waiting
// waiting, timeout, idle

// ALL (except dying & dead), hit, dying
// dying, animationFinished, dead

// map[AnimationStateName]map[TransitionEvent]AnimationStateName

func NewActorTransitionTable() *TransitionTable {
	t := NewTransitionTable()

	// waypoints
	t.AddTransition(ActorStateIdle, EventNewPath, UnitGotoWaypoint)
	t.AddTransition(UnitGotoWaypoint, EventLastWaypointReached, ActorStateIdle)
	//t.AddTransition(ActorStateWaiting, EventFinishedWaiting, ActorStateIdle)

	// hits
	t.AddTransition(ActorStateIdle, EventHit, ActorStateHit)
	t.AddTransition(ActorStateHit, EventHit, ActorStateHit)
	t.AddTransition(ActorStateHit, EventAnimationFinished, ActorStateIdle)

	// dying & death
	t.AddTransition(ActorStateIdle, EventLethalHit, ActorStateDying)
	t.AddTransition(ActorStateHit, EventLethalHit, ActorStateDying)
	t.AddTransition(ActorStateDying, EventAnimationFinished, ActorStateDead)

	return t
}

var ActorTransitionTable = NewActorTransitionTable()

type TransitionEvent int

func (e TransitionEvent) ToString() any {
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

const (
	EventNone TransitionEvent = iota
	EventNewPath
	EventFinishedWaiting
	EventHit
	EventLethalHit
	EventAnimationFinished
	EventLastWaypointReached
	EventWaypointReached
)

type AnimationStateName int

func (s AnimationStateName) ToString() string {
	switch s {
	case ActorStateIdle:
		return "Idle"
	case ActorStateWaiting:
		return "Waiting"
	case UnitGotoWaypoint:
		return "GotoWaypoint"
	case ActorStateDying:
		return "Dying"
	case ActorStateDead:
		return "Dead"
	case ActorStateHit:
		return "Hit"
	default:
		return "Unknown"
	}
}

const (
	ActorStateIdle AnimationStateName = iota
	ActorStateWaiting
	UnitGotoWaypoint
	ActorStateDying
	ActorStateHit
	ActorStateDead
	// Also change NewTransitionTable() below, if you add new states at the end or the beginning
)

type TransitionTable map[AnimationStateName]map[TransitionEvent]AnimationStateName

func NewTransitionTable() *TransitionTable {
	t := make(TransitionTable)
	for state := ActorStateIdle; state <= ActorStateDead; state++ {
		t[state] = make(map[TransitionEvent]AnimationStateName)
	}
	return &t
}

func (t *TransitionTable) AddTransition(fromState AnimationStateName, event TransitionEvent, toState AnimationStateName) {
	(*t)[fromState][event] = toState
}

func (t *TransitionTable) AddTransitionFromAllExcept(excludedStates []AnimationStateName, event TransitionEvent, toState AnimationStateName) {
	for state := range *t {
		if !contains(excludedStates, state) {
			(*t)[state][event] = toState
		}
	}
}

func (t *TransitionTable) Exists(currentState AnimationStateName, event TransitionEvent) bool {
	_, ok := (*t)[currentState][event]
	return ok
}

func (t *TransitionTable) GetNextState(currentState AnimationStateName, event TransitionEvent) AnimationStateName {
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
	Init(actor *Unit)
	Execute(deltaTime float64) TransitionEvent
}

var BehaviorTable = map[AnimationStateName]func() AnimationState{
	ActorStateIdle:    func() AnimationState { return &ActorIdleBehavior{} },
	UnitGotoWaypoint:  func() AnimationState { return &UnitGotoWaypointBehavior{} },
	ActorStateWaiting: func() AnimationState { return &ActorWaitingBehavior{} },
	ActorStateHit:     func() AnimationState { return &ActorHitBehavior{} },
	ActorStateDying:   func() AnimationState { return &ActorDyingBehavior{} },
	ActorStateDead:    func() AnimationState { return &ActorDeadBehavior{} },
}

func BehaviorFactory(state AnimationStateName) AnimationState {
	return BehaviorTable[state]()
}
