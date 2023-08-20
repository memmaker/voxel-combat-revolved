package game

import "github.com/memmaker/battleground/engine/voxel"

type VoxelPather struct {
	voxelMap *voxel.Map
	unit     *UnitInstance
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
	placeable, _ := v.voxelMap.IsUnitPlaceable(v.unit, neighbor)
	return placeable
}

func NewPather(voxelMap *voxel.Map, unit *UnitInstance) *VoxelPather {
	return &VoxelPather{voxelMap: voxelMap, unit: unit}
}
