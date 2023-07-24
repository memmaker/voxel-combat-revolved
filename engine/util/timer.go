package util

import (
	"fmt"
	"math"
	"time"
)

type TimerState struct {
	name         string
	lastDuration float64

	totalDuration  float64
	executionCount int64

	minDuration float64
	maxDuration float64
}

func (t *TimerState) averageDuration() float64 {
	return t.totalDuration / float64(t.executionCount)
}
func (t *TimerState) String() string {
	return fmt.Sprintf("%s last: %.2fms, avg: %.2fms\n> min: %.2fms, max: %.2fms", t.name, t.lastDuration, t.averageDuration(), t.minDuration, t.maxDuration)
}

type Timer struct {
	states     map[string]*TimerState
	timerNames []string
}

func NewTimer() *Timer {
	return &Timer{
		states: make(map[string]*TimerState),
	}
}
func (t *Timer) GetState(name string) *TimerState {
	return t.states[name]
}
func (t *Timer) Reset() {
	for _, state := range t.states {
		state.lastDuration = 0
		state.totalDuration = 0
		state.executionCount = 0
		state.minDuration = math.MaxInt64
		state.maxDuration = math.MinInt64
	}
}
func (t *Timer) String() string {
	var str string
	for _, name := range t.timerNames {
		str += t.states[name].String() + "\n"
	}
	return str
}
func (t *Timer) Start(name string) func() float64 {
	var state *TimerState
	var ok bool
	if state, ok = t.states[name]; !ok {
		t.timerNames = append(t.timerNames, name)
		state = &TimerState{
			name:        name,
			minDuration: math.MaxInt64,
			maxDuration: math.MinInt64,
		}
		t.states[name] = state
	}
	start := time.Now()
	return func() float64 {
		durationInMS := float64(time.Since(start).Microseconds()) / 1000.0
		state.lastDuration = durationInMS
		state.totalDuration += durationInMS
		state.executionCount++
		if durationInMS < state.minDuration {
			state.minDuration = durationInMS
		}
		if durationInMS > state.maxDuration {
			state.maxDuration = durationInMS
		}
		return durationInMS
	}
}
