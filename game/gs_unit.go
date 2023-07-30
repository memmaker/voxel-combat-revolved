package game

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type GameStateUnit struct {
	engine       *BattleGame
	selectedUnit *Unit
	lastMouseY   float64
	lastMouseX   float64
}

func (g *GameStateUnit) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		g.engine.camera.ZoomOut(deltaTime, yoff)
	} else {
		g.engine.camera.ZoomIn(deltaTime, -yoff)
	}
}

func (g *GameStateUnit) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeySpace && g.selectedUnit.CanAct() {
		g.engine.SwitchToAction(g.selectedUnit, &ActionMove{engine: g.engine, previousNodeMap: make(map[voxel.Int3]voxel.Int3), distanceMap: make(map[voxel.Int3]int)})
	} else if key == glfw.KeyEnter {
		g.engine.SwitchToAction(g.selectedUnit, &ActionAttack{engine: g.engine})
	} else if key == glfw.KeyTab {
		g.nextUnit()
	}
}

func (g *GameStateUnit) nextUnit() {
	nextUnit, exists := g.engine.GetNextUnit(g.selectedUnit)
	if !exists {
		println("[GameStateUnit] No unit left to act.")
	} else {
		g.selectedUnit = nextUnit
		g.Init(false)
	}
}

func (g *GameStateUnit) Init(wasPopped bool) {
	if !wasPopped {
		println(fmt.Sprintf("[GameStateUnit] Entered for %s", g.selectedUnit.GetName()))
		footPos := util.ToGrid(g.selectedUnit.GetFootPosition())
		g.engine.SwitchToGroundSelector()
		g.engine.unitSelector.SetPosition(footPos)
		g.engine.camera.CenterOn(footPos.Add(mgl32.Vec3{0.5, 0, 0.5}))
	}
}

func (g *GameStateUnit) OnZoomIn(deltaTime float64) {
	g.engine.camera.ZoomIn(deltaTime, 0)
}

func (g *GameStateUnit) OnZoomOut(deltaTime float64) {
	g.engine.camera.ZoomOut(deltaTime, 0)
}

func (g *GameStateUnit) OnUpperRightAction() {
	g.engine.camera.RotateRight()
}

func (g *GameStateUnit) OnUpperLeftAction() {
	g.engine.camera.RotateLeft()
}

func (g *GameStateUnit) OnMouseClicked(x float64, y float64) {
	println(fmt.Sprintf("[GameStateUnit] Screen clicked at (%0.1f, %0.1f)", x, y))
	// project point from screen space to camera space
	rayStart, rayEnd := g.engine.camera.GetPickingRayFromScreenPosition(x, y)
	hitInfo := g.engine.RayCast(rayStart, rayEnd)
	if hitInfo != nil && hitInfo.UnitHit != nil && hitInfo.UnitHit.CanAct() && hitInfo.UnitHit.faction == g.engine.CurrentFaction() {
		g.selectedUnit = hitInfo.UnitHit
		println(fmt.Sprintf("[GameStateUnit] Selected unit at %s", g.selectedUnit.GetBlockPosition().ToString()))
		g.Init(false)
	}
}

func (g *GameStateUnit) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	g.engine.camera.ChangePosition(movementVector, float32(elapsed))
	g.engine.UpdateMousePicking(g.lastMouseX, g.lastMouseY)
}

func (g *GameStateUnit) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.lastMouseX = newX
	g.lastMouseY = newY
	g.engine.UpdateMousePicking(newX, newY)
}
