package client

import "github.com/memmaker/battleground/engine/voxel"

type BlockPlacer interface {
	GetName() string
	StartDragAt(blockPos voxel.Int3)
	DraggedOver(blockPos voxel.Int3) []voxel.Int3
	StopDragAt(blockPos voxel.Int3) []voxel.Int3
	SetFill(fill bool)
	GetFill() bool
    IsDragging() bool
}

type RectanglePlacer struct {
    start      voxel.Int3
    fill       bool
    isDragging bool
}

func NewRectanglePlacer() *RectanglePlacer {
	return &RectanglePlacer{}
}
func (a *RectanglePlacer) GetName() string {
	return "Rectangle"
}

func (a *RectanglePlacer) SetFill(fill bool) {
	a.fill = fill
}
func (a *RectanglePlacer) GetFill() bool {
	return a.fill
}
func (a *RectanglePlacer) StartDragAt(blockPos voxel.Int3) {
	a.start = blockPos
    a.isDragging = true
}

func (a *RectanglePlacer) DraggedOver(blockPos voxel.Int3) []voxel.Int3 {
	return a.outlinedRectangle(a.start, blockPos)
}

func (a *RectanglePlacer) IsDragging() bool {
    return a.isDragging
}

func (a *RectanglePlacer) StopDragAt(blockPos voxel.Int3) []voxel.Int3 {
    a.isDragging = false
	return a.outlinedRectangle(a.start, blockPos)
}

func (a *RectanglePlacer) outlinedRectangle(start voxel.Int3, end voxel.Int3) []voxel.Int3 {
	height := start.Y
	result := make([]voxel.Int3, 0)
	startX := min(start.X, end.X)
	endX := max(start.X, end.X)
	startZ := min(start.Z, end.Z)
	endZ := max(start.Z, end.Z)

	for x := startX; x <= endX; x++ {
		for z := startZ; z <= endZ; z++ {
			if a.fill || x == startX || x == endX || z == startZ || z == endZ {
				result = append(result, voxel.Int3{X: x, Y: height, Z: z})
			}
		}
	}
	return result
}
