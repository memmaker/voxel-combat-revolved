package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type GameStateEditMap struct {
	IsoMovementState
}

func (g *GameStateEditMap) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyF {
		g.engine.PlaceBlockAtCurrentSelection()
	}

	if key == glfw.KeyR {
		g.engine.RemoveBlock()
	}

	if key == glfw.KeyO {
		g.engine.GetVoxelMap().SaveToDisk()
	}

	if key == glfw.KeyP {
		g.engine.GetVoxelMap().LoadFromDisk("assets/maps/map.bin")
		g.engine.GetVoxelMap().GenerateAllMeshes()
	}
}

func (g *GameStateEditMap) Init(bool) {
	g.engine.SwitchToBlockSelector()
	println(fmt.Sprintf("[GameStateEditMap] Entered"))
}

func (g *GameStateEditMap) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[Picking] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to isoCamera space
	rayStart, rayEnd := g.engine.isoCamera.GetPickingRayFromScreenPosition(x, y)
	hitInfo := g.engine.RayCast(rayStart, rayEnd)
	if hitInfo != nil && hitInfo.Hit {
		prevGrid := hitInfo.PreviousGridPosition.ToVec3()
		println(fmt.Sprintf("[Picking] Block %s", hitInfo.PreviousGridPosition.ToString()))
		g.engine.selector.SetPosition(prevGrid)
	}
}
