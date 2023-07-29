package path

import (
	"container/heap"
)

func NewNode[T comparable](value T) *PqItem[T] {
	return &PqItem[T]{value: value}
}

// An PathNode is something we manage in a priority queue.
type PqItem[T comparable] struct {
	value    T   // The value of the item; arbitrary.
	priority int // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

func (item *PqItem[T]) GetPriority() int {
	return item.priority
}

func (item *PqItem[T]) SetPriority(priority int) {
	item.priority = priority
}

func (item *PqItem[T]) GetIndex() int {
	return item.index
}

func (item *PqItem[T]) SetIndex(index int) {
	item.index = index
}

func (item *PqItem[T]) GetValue() T {
	return item.value
}

type PathNode[T comparable] interface {
	GetPriority() int
	SetPriority(int)
	GetIndex() int
	SetIndex(int)
	GetValue() T
}

func NewPriorityQueue[T comparable](items []PathNode[T]) PriorityQueue[T] {
	pq := make(PriorityQueue[T], len(items))
	i := 0
	for _, item := range items {
		pq[i] = item
		i++
	}
	heap.Init(&pq)
	return pq
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue[T comparable] []PathNode[T]

func (pq *PriorityQueue[T]) Len() int { return len(*pq) }

func (pq *PriorityQueue[T]) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return (*pq)[i].GetPriority() < (*pq)[j].GetPriority()
}

func (pq *PriorityQueue[T]) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].SetIndex(i)
	(*pq)[j].SetIndex(j)
}

func (pq *PriorityQueue[T]) Push(x any) {
	n := len(*pq)
	item := x.(PathNode[T])
	item.SetIndex(n)
	*pq = append(*pq, item)
	heap.Fix(pq, item.GetIndex())
}
func (pq *PriorityQueue[T]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil    // avoid memory leak
	item.SetIndex(-1) // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue[T]) Top() PathNode[T] {
	return (*pq)[0]
}

func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.Len() == 0
}

// update modifies the priority and value of an PathNode in the queue.
func (pq *PriorityQueue[T]) update(item PathNode[T], priority int) {
	item.SetPriority(priority)
	heap.Fix(pq, item.GetIndex())
}
