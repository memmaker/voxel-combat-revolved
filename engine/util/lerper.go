package util

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
