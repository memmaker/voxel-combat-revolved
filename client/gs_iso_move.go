package client

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

func (i *IsoMovementState) OnZoomIn(deltaTime float64) {
	i.engine.isoCamera.ZoomIn(deltaTime, 0)
}

func (i *IsoMovementState) OnZoomOut(deltaTime float64) {
	i.engine.isoCamera.ZoomOut(deltaTime, 0)
}

func (i *IsoMovementState) OnUpperRightAction(deltaTime float64) {
	i.engine.camera().RotateRight(deltaTime)
}

func (i *IsoMovementState) OnUpperLeftAction(deltaTime float64) {
	i.engine.camera().RotateLeft(deltaTime)
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
