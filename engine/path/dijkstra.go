package path

import (
	"math"
)

/*
function Dijkstra(Graph, source):
2      dist[source] ← 0                           // Initialization
3
4      create vertex priority queue Q
5
6      for each vertex v in Graph.Vertices:
7          if v ≠ source
8              dist[v] ← INFINITY                 // Unknown distance from source to v
9              prev[v] ← UNDEFINED                // Predecessor of v
10
11         Q.add_with_priority(v, dist[v])
12
13
14     while Q is not empty:                      // The main loop
15         u ← Q.extract_min()                    // Remove and return best vertex
16         for each neighbor v of u:              // Go through all v neighbors of u
17             alt ← dist[u] + Graph.Edges(u, v)
18             if alt < dist[v]:
19                 dist[v] ← alt
20                 prev[v] ← u
21                 Q.decrease_priority(v, alt)
22
23     return dist, prev
*/

// USAGE:
// type MyNode struct {
// 	PqItem
// 	// ...
// }

type DijkstraSource[T any] interface {
	GetNeighbors(node T) []T
	GetCost(currentNode T, neighbor T) float64
}

func Dijkstra[T comparable](source *PqItem[T], maxCost float64, dataSource DijkstraSource[T]) (dist map[T]float64, prev map[T]T) {
	dist = make(map[T]float64)
	prev = make(map[T]T)
	existingNodes := make(map[T]PathNode[T])
	dist[source.GetValue()] = 0
	getDist := func(n T) float64 {
		if d, ok := dist[n]; ok {
			return d
		} else {
			return math.MaxFloat64
		}
	}
	Q := NewPriorityQueue([]PathNode[T]{source})
	for Q.Len() > 0 {
		currentNode := Q.Pop().(PathNode[T])
		for _, n := range dataSource.GetNeighbors(currentNode.GetValue()) { // generates new instances, no state is preserved..!
			neighbor := n
			neighborDist := getDist(currentNode.GetValue()) + dataSource.GetCost(currentNode.GetValue(), neighbor)
			oldNeighborDist := getDist(neighbor)
			if neighborDist <= maxCost && neighborDist < oldNeighborDist {
				var neighborNode PathNode[T]
				if existingNode, ok := existingNodes[neighbor]; ok {
					existingNode.SetPriority(neighborDist)
					neighborNode = existingNode
				} else {
					neighborNode = NewNode(neighbor)
					neighborNode.SetPriority(neighborDist)
					existingNodes[neighbor] = neighborNode
				}
				dist[neighbor] = neighborDist
				prev[neighbor] = currentNode.GetValue()
				Q.Push(neighborNode)
			}
		}
	}
	return
}
