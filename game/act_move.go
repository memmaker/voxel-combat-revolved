package game

import (
	"github.com/memmaker/battleground/engine/path"
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionMove struct {
	gameMap         *voxel.Map
	selectedPath    []voxel.Int3
	previousNodeMap map[voxel.Int3]voxel.Int3
	distanceMap     map[voxel.Int3]int
	unit            *UnitInstance
	validTargets    []voxel.Int3
}

func (a *ActionMove) IsValidTarget(target voxel.Int3) bool {
	distance, ok := a.distanceMap[target]
	return ok && distance <= a.unit.MovesLeft()
}

func NewActionMove(gameMap *voxel.Map, unit *UnitInstance) *ActionMove {
	a := &ActionMove{
		gameMap:         gameMap,
		previousNodeMap: make(map[voxel.Int3]voxel.Int3),
		distanceMap:     make(map[voxel.Int3]int),
		unit:            unit,
	}
	a.updateTargetData()
	return a
}

func (a *ActionMove) GetName() string {
	return "Move"
}

func (a *ActionMove) GetValidTargets() []voxel.Int3 {
	return a.validTargets
}

func (a *ActionMove) GetPath(target voxel.Int3) []voxel.Int3 {
	pathToTarget := make([]voxel.Int3, a.distanceMap[target]+1)
	current := target
	index := len(pathToTarget) - 1
	for {
		pathToTarget[index] = current
		if prev, ok := a.previousNodeMap[current]; ok {
			current = prev
			index--
		} else {
			break
		}
	}
	// remove first element, which is the current position
	return pathToTarget[1:]
}

func (a *ActionMove) GetCost(target voxel.Int3) int {
	return a.distanceMap[target]
}

func (a *ActionMove) updateTargetData() {
	footPosInt := a.unit.GetBlockPosition()
	var valid []voxel.Int3
	dist, prevNodeMap := path.Dijkstra[voxel.Int3](path.NewNode(footPosInt), a.unit.MovesLeft(), NewPather(a.gameMap, a.unit))
	for node, distance := range dist {
		if node == footPosInt {
			continue
		}
		valid = append(valid, node)
		a.distanceMap[node] = distance
	}
	for node, prevNode := range prevNodeMap {
		a.previousNodeMap[node] = prevNode
	}
	a.validTargets = valid
}
