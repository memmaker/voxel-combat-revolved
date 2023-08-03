package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/path"
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionMove struct {
	gameMap         *voxel.Map
	selectedPath    []voxel.Int3
	previousNodeMap map[voxel.Int3]voxel.Int3
	distanceMap     map[voxel.Int3]int
}

func (a *ActionMove) IsValidTarget(unit UnitCore, target voxel.Int3) bool {
	a.GetValidTargets(unit)
	distance, ok := a.distanceMap[target]
	return ok && distance <= unit.MovesLeft()
}

func NewActionMove(gameMap *voxel.Map) *ActionMove {
	return &ActionMove{
		gameMap:         gameMap,
		previousNodeMap: make(map[voxel.Int3]voxel.Int3),
		distanceMap:     make(map[voxel.Int3]int),
	}
}

func (a *ActionMove) GetName() string {
	return "Move"
}

func (a *ActionMove) GetValidTargets(unit UnitCore) []voxel.Int3 {
	footPosInt := voxel.ToGridInt3(unit.GetFootPosition())
	var valid []voxel.Int3
	dist, prevNodeMap := path.Dijkstra[voxel.Int3](path.NewNode(footPosInt), unit.MovesLeft(), NewPather(a.gameMap))
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
	return valid
}

func (a *ActionMove) Execute(unit UnitCore, target voxel.Int3) {
	currentPos := voxel.ToGridInt3(unit.GetFootPosition())
	distance := a.distanceMap[target]
	println(fmt.Sprintf("[ActionMove] Moving %s: from %s to %s (dist: %d)", unit.GetName(), currentPos.ToString(), target.ToString(), distance))

	foundPath := a.getPath(target)
	for _, pos := range foundPath {
		println(fmt.Sprintf(" -> %s", pos.ToString()))
	}

	unit.SetPath(foundPath)
}

func (a *ActionMove) getPath(target voxel.Int3) []voxel.Int3 {
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

type VoxelPather struct {
	voxelMap *voxel.Map
}

func (v *VoxelPather) GetNeighbors(node voxel.Int3) []voxel.Int3 {
	currentBlock := node
	neighbors := v.voxelMap.GetNeighborsForGroundMovement(currentBlock, v.isWalkable)
	result := make([]voxel.Int3, len(neighbors))
	for i, neighbor := range neighbors {
		result[i] = neighbor
	}
	return result
}

func (v *VoxelPather) GetCost(currentNode, neighbor voxel.Int3) int {
	return int(voxel.ManhattanDistance2(currentNode, neighbor))
}

func (v *VoxelPather) isWalkable(neighbor voxel.Int3) bool {
	tBlock := v.voxelMap.GetGlobalBlock(neighbor.X, neighbor.Y+1, neighbor.Z)
	nBlock := v.voxelMap.GetGlobalBlock(neighbor.X, neighbor.Y, neighbor.Z)
	bBlock := v.voxelMap.GetGlobalBlock(neighbor.X, neighbor.Y-1, neighbor.Z)
	return !nBlock.IsOccupied() && tBlock.IsAir() && nBlock.IsAir() && !bBlock.IsAir()
}

func NewPather(voxelMap *voxel.Map) *VoxelPather {
	return &VoxelPather{voxelMap: voxelMap}
}
