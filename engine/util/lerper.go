package util

import "github.com/go-gl/mathgl/mgl32"

type Lerper[V any] struct {
    start, finish V
    duration      float64
    timer         float64
    setValue      func(V)
    lerpValue     func(V, V, float64) V
    isDone        bool
}

func NewLerper[V any](lerpValue func(V, V, float64) V, setValue func(V), start, finish V, duration float64) *Lerper[V] {
    return &Lerper[V]{
        start:     start,
        finish:    finish,
        duration:  duration,
        lerpValue: lerpValue,
        setValue:  setValue,
    }
}
func (l *Lerper[V]) IsDone() bool {
    return l.isDone
}
func (l *Lerper[V]) Update(deltaTime float64) bool {
    if l.isDone {
        return true
    }

    l.timer += deltaTime
    if l.timer > l.duration {
        l.setValue(l.finish)
        l.isDone = true
        return l.isDone
    }

    percent := l.timer / l.duration
    lerpedValue := l.lerpValue(l.start, l.finish, percent)
    l.setValue(lerpedValue)
    return false
}

func (l *Lerper[V]) Reset(startPos V, finish V) {
    l.finish = finish
    l.start = startPos
    l.timer = 0
    l.isDone = false
}

type Positionable interface {
    GetPosition() mgl32.Vec3
    SetPosition(mgl32.Vec3)
}

func NewPositionLerper(thing Positionable, start, finish mgl32.Vec3, duration float64) *Lerper[mgl32.Vec3] {
    setValue := func(v mgl32.Vec3) { thing.SetPosition(v) }
    return NewLerper[mgl32.Vec3](Lerp3, setValue, start, finish, duration)
}

type WaypointLerper struct {
    path    []mgl32.Vec3
    lerper  *Lerper[mgl32.Vec3]
    current int
}

func NewWaypointLerper(thing Positionable, path []mgl32.Vec3, durationBetweenWaypoints float64) *WaypointLerper {
    return &WaypointLerper{
        path:   path,
        lerper: NewPositionLerper(thing, path[0], path[1], durationBetweenWaypoints),
    }
}

func (w *WaypointLerper) Update(deltaTime float64) {
    if !w.lerper.IsDone() {
        w.lerper.Update(deltaTime)
    } else if w.current < len(w.path)-2 {
        w.current++
        w.lerper.Reset(w.getCurrent(), w.getNext())
    }
}

func (w *WaypointLerper) IsDone() bool {
    return w.current+2 >= len(w.path) && w.lerper.IsDone()
}

func (w *WaypointLerper) getCurrent() mgl32.Vec3 {
    return w.path[w.current]
}

func (w *WaypointLerper) getNext() mgl32.Vec3 {
    return w.path[w.current+1]
}