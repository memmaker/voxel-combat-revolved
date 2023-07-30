package game

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type GameStateEditMap struct {
	engine     *BattleGame
	lastMouseX float64
	lastMouseY float64
}

func (g *GameStateEditMap) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		g.engine.camera.ZoomOut(deltaTime, yoff)
	} else {
		g.engine.camera.ZoomIn(deltaTime, -yoff)
	}
}

func (g *GameStateEditMap) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyF {
		g.engine.PlaceBlockAtCurrentSelection()
	}

	if key == glfw.KeyR {
		g.engine.RemoveBlock()
	}

	if key == glfw.KeyO {
		g.engine.voxelMap.SaveToDisk()
	}

	if key == glfw.KeyP {
		g.engine.voxelMap.LoadFromDisk("assets/maps/map.bin")
	}
}

func (g *GameStateEditMap) Init(bool) {
	g.engine.SwitchToBlockSelector()
	println(fmt.Sprintf("[GameStateEditMap] Entered"))
}

func (g *GameStateEditMap) OnUpperRightAction() {
	g.engine.camera.RotateRight()
}

func (g *GameStateEditMap) OnUpperLeftAction() {
	g.engine.camera.RotateLeft()
}

func (g *GameStateEditMap) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[Picking] Clicked at %0.2f, %0.2f", x, y))
	// project point from screen space to camera space
	rayStart, rayEnd := g.engine.camera.GetPickingRayFromScreenPosition(x, y)
	hitInfo := g.engine.RayCast(rayStart, rayEnd)
	if hitInfo != nil && hitInfo.Hit {
		prevGrid := hitInfo.PreviousGridPosition.ToVec3()
		println(fmt.Sprintf("[Picking] Block %s", hitInfo.PreviousGridPosition.ToString()))
		g.engine.selector.SetPosition(prevGrid)
	}
}

func (g *GameStateEditMap) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	g.engine.camera.ChangePosition(movementVector, float32(elapsed))
	g.engine.UpdateMousePicking(g.lastMouseX, g.lastMouseY)
}

func (g *GameStateEditMap) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.lastMouseX = newX
	g.lastMouseY = newY
	g.engine.UpdateMousePicking(newX, newY)
}
