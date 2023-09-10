package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type IsoMovementState struct {
	engine     *BattleClient
	lastMouseY float64
	lastMouseX float64
}

func (i *IsoMovementState) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	speedFactor := i.engine.settings.ISOCameraScrollZoomSpeed
	if yoff > 0 {
		i.engine.isoCamera.ZoomOut(deltaTime, yoff*float64(speedFactor))
	} else {
		i.engine.isoCamera.ZoomIn(deltaTime, -yoff*float64(speedFactor))
	}
}
func (i *IsoMovementState) OnUpperRightAction(deltaTime float64) {
	i.engine.isoCamera.RotateRight(deltaTime)
}
func (i *IsoMovementState) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyZ {
		i.OnLowerLevel()
	} else if key == glfw.KeyC {
		i.OnRaiseLevel()
	}
}
func (i *IsoMovementState) OnUpperLeftAction(deltaTime float64) {
	i.engine.isoCamera.RotateLeft(deltaTime)
}
func (i *IsoMovementState) OnRaiseLevel() {
	changedTo := i.engine.GetVoxelMap().ChangeMaxChunkHeightForDraw(1)
	i.engine.Print(fmt.Sprintf("Max Height changed to %d", changedTo))
}

func (i *IsoMovementState) OnLowerLevel() {
	changedTo := i.engine.GetVoxelMap().ChangeMaxChunkHeightForDraw(-1)
	i.engine.Print(fmt.Sprintf("Max Height changed to %d", changedTo))
}
func (i *IsoMovementState) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	i.engine.isoCamera.MoveInDirection(float32(elapsed), movementVector)
	i.engine.UpdateMousePicking(i.lastMouseX, i.lastMouseY)
}

func (i *IsoMovementState) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	i.lastMouseX = newX
	i.lastMouseY = newY
	i.engine.UpdateMousePicking(newX, newY)
}
