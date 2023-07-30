package game

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

	// dying & death
	t.AddTransitionFromAllExcept([]ActorState{ActorStateDying, ActorStateDead}, EventHit, ActorStateDying)
	t.AddTransition(ActorStateDying, EventAnimationFinished, ActorStateDead)

	return t
}

var ActorTransitionTable = NewActorTransitionTable()

type TransitionEvent int

const (
	EventNone TransitionEvent = iota
	EventNewPath
	EventFinishedWaiting
	EventHit
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

type Behavior interface {
	GetName() ActorState
	Init(actor *Unit)
	Execute(deltaTime float64) TransitionEvent
}

var BehaviorTable = map[ActorState]func() Behavior{
	ActorStateIdle:    func() Behavior { return &ActorIdleBehavior{} },
	UnitGotoWaypoint:  func() Behavior { return &UnitGotoWaypointBehavior{} },
	ActorStateWaiting: func() Behavior { return &ActorWaitingBehavior{} },
	ActorStateDying:   func() Behavior { return &ActorDyingBehavior{} },
	ActorStateDead:    func() Behavior { return &ActorDeadBehavior{} },
}

func BehaviorFactory(state ActorState) Behavior {
	return BehaviorTable[state]()
}
