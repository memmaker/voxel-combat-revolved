package path

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

type DijkstraSource interface {
	GetNeighbors(node PathNode) []PathNode
	GetCost(currentNode PathNode, neighbor PathNode) int
}

func Dijkstra(source PathNode, maxCost int, dataSource DijkstraSource) (dist map[PathNode]int, prev map[PathNode]PathNode) {
	dist = make(map[PathNode]int)
	prev = make(map[PathNode]PathNode)
	dist[source] = 0
	Q := NewPriorityQueue([]PathNode{source})
	for Q.Len() > 0 {
		currentNode := Q.Pop().(PathNode)
		for _, neighbor := range dataSource.GetNeighbors(currentNode) {
			neighborDist := dist[currentNode] + dataSource.GetCost(currentNode, neighbor)
			oldNeighborDist, exists := dist[neighbor]
			if !exists && neighborDist <= maxCost {
				dist[neighbor] = neighborDist
				prev[neighbor] = currentNode
				Q.Push(neighbor)
			} else if neighborDist < oldNeighborDist {
				dist[neighbor] = neighborDist
				prev[neighbor] = currentNode
				Q.update(neighbor, neighborDist)
			}
		}
	}
	return
}
