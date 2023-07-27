package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/path"
	"github.com/memmaker/battleground/engine/voxel"
)

type ActionMove struct {
	engine *BattleGame
}

func (a ActionMove) GetName() string {
	return "Move"
}

func (a ActionMove) GetValidTargets(unit *Unit) []voxel.Int3 {
	footPosInt := voxel.ToGridInt3(unit.GetFootPosition())
	valid := []voxel.Int3{}
	dist, _ := path.Dijkstra(path.NewNode(footPosInt), 5, NewPather(a.engine.voxelMap))
	for node, _ := range dist {
		valid = append(valid, node.GetValue().(voxel.Int3))
	}
	return valid
}

type VoxelPather struct {
	voxelMap *voxel.Map
}

func (v VoxelPather) GetNeighbors(node path.PathNode) []path.PathNode {
	currentBlock := node.GetValue().(voxel.Int3)
	neighbors := v.voxelMap.GetNeighbors(currentBlock, v.isWalkable)
	result := make([]path.PathNode, len(neighbors))
	for i, neighbor := range neighbors {
		result[i] = path.NewNode(neighbor)
	}
	return result
}

func (v VoxelPather) GetCost(currentNode path.PathNode, neighbor path.PathNode) int {
	return int(voxel.ManhattanDistance3(currentNode.GetValue().(voxel.Int3), neighbor.GetValue().(voxel.Int3)))
}

func (v VoxelPather) isWalkable(neighbor voxel.Int3) bool {
	nBlock := v.voxelMap.GetGlobalBlock(neighbor.X, neighbor.Y, neighbor.Z)
	bBlock := v.voxelMap.GetGlobalBlock(neighbor.X, neighbor.Y-1, neighbor.Z)
	return nBlock.IsAir() && !bBlock.IsAir()
}

func NewPather(voxelMap *voxel.Map) *VoxelPather {
	return &VoxelPather{voxelMap: voxelMap}
}

func (a ActionMove) Execute(unit *Unit, target voxel.Int3) {
	println(fmt.Sprintf("Moving %s to %s", unit.GetName(), target.ToString()))
	unit.SetWaypoint(target.ToBlockCenterVec3())
}
