package client

// state transition table
// currentState, event, nextState

// idle, newWaypoint, gotoWaypoint
// gotoWaypoint, nearWaypoint, waiting
// waiting, timeout, idle

// ALL (except dying & dead), hit, dying
// dying, animationFinished, dead

// map[ActorState]map[TransitionEvent]ActorState

func NewActorTransitionTable() *TransitionTable {
	t := NewTransitionTable()

	// waypoints
	t.AddTransition(ActorStateIdle, EventNewPath, UnitGotoWaypoint)
	t.AddTransition(UnitGotoWaypoint, EventLastWaypointReached, ActorStateIdle)
	//t.AddTransition(ActorStateWaiting, EventFinishedWaiting, ActorStateIdle)

	// hits
	t.AddTransitionFromAllExcept([]ActorState{ActorStateDying, ActorStateDead}, EventHit, ActorStateHit)
	t.AddTransition(ActorStateHit, EventAnimationFinished, ActorStateIdle)

	// dying & death
	t.AddTransitionFromAllExcept([]ActorState{ActorStateDying, ActorStateDead}, EventLethalHit, ActorStateDying)
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

type ActorState int

func (s ActorState) ToString() string {
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
	default:
		return "Unknown"
	}
}

const (
	ActorStateIdle ActorState = iota
	ActorStateWaiting
	UnitGotoWaypoint
	ActorStateDying
	ActorStateHit
	ActorStateDead
	// Also change NewTransitionTable() below, if you add new states at the end or the beginning
)

type TransitionTable map[ActorState]map[TransitionEvent]ActorState

func NewTransitionTable() *TransitionTable {
	t := make(TransitionTable)
	for state := ActorStateIdle; state <= ActorStateDead; state++ {
		t[state] = make(map[TransitionEvent]ActorState)
	}
	return &t
}

func (t *TransitionTable) AddTransition(fromState ActorState, event TransitionEvent, toState ActorState) {
	(*t)[fromState][event] = toState
}

func (t *TransitionTable) AddTransitionFromAllExcept(excludedStates []ActorState, event TransitionEvent, toState ActorState) {
	for state := range *t {
		if !contains(excludedStates, state) {
			(*t)[state][event] = toState
		}
	}
}

func (t *TransitionTable) Exists(currentState ActorState, event TransitionEvent) bool {
	_, ok := (*t)[currentState][event]
	return ok
}

func (t *TransitionTable) GetNextState(currentState ActorState, event TransitionEvent) ActorState {
	return (*t)[currentState][event]
}

func contains(states []ActorState, state ActorState) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

type AnimationState interface {
	GetName() ActorState
	Init(actor *Unit)
	Execute(deltaTime float64) TransitionEvent
}

var BehaviorTable = map[ActorState]func() AnimationState{
	ActorStateIdle:    func() AnimationState { return &ActorIdleBehavior{} },
	UnitGotoWaypoint:  func() AnimationState { return &UnitGotoWaypointBehavior{} },
	ActorStateWaiting: func() AnimationState { return &ActorWaitingBehavior{} },
	ActorStateHit: func() AnimationState { return &ActorHitBehavior{} },
	ActorStateDying:   func() AnimationState { return &ActorDyingBehavior{} },
	ActorStateDead:    func() AnimationState { return &ActorDeadBehavior{} },
}

func BehaviorFactory(state ActorState) AnimationState {
	return BehaviorTable[state]()
}
