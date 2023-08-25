package game

import (
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type VoxelPather struct {
	voxelMap *voxel.Map
	unit     *UnitInstance
}

func (v *VoxelPather) GetNeighbors(node voxel.Int3) []voxel.Int3 {
	return v.voxelMap.GetNeighborsForGroundMovement(node, v.isWalkable)
}
func (v *VoxelPather) isWalkable(neighbor voxel.Int3) bool {
	placeable, _ := v.voxelMap.IsUnitPlaceable(v.unit, neighbor)
	return placeable
}

func (v *VoxelPather) GetCost(currentNode, neighbor voxel.Int3) float64 {
	return float64(util.EucledianDistance3D(currentNode.ToBlockCenterVec3D(), neighbor.ToBlockCenterVec3D()))
	//return int(voxel.ManhattanDistance2(currentNode, neighbor))
}
func NewPather(voxelMap *voxel.Map, unit *UnitInstance) *VoxelPather {
	return &VoxelPather{voxelMap: voxelMap, unit: unit}
}
