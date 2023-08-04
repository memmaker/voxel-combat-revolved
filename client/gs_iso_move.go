package client

type IsoMovementState struct {
	engine     *BattleClient
	lastMouseY float64
	lastMouseX float64
}

func (i *IsoMovementState) OnScroll(deltaTime float64, xoff float64, yoff float64) {
	if yoff > 0 {
		i.engine.isoCamera.ZoomOut(deltaTime, yoff)
	} else {
		i.engine.isoCamera.ZoomIn(deltaTime, -yoff)
	}
}

func (i *IsoMovementState) OnZoomIn(deltaTime float64) {
	i.engine.isoCamera.ZoomIn(deltaTime, 0)
}

func (i *IsoMovementState) OnZoomOut(deltaTime float64) {
	i.engine.isoCamera.ZoomOut(deltaTime, 0)
}

func (i *IsoMovementState) OnUpperRightAction() {
	i.engine.isoCamera.RotateRight()
}

func (i *IsoMovementState) OnUpperLeftAction() {
	i.engine.isoCamera.RotateLeft()
}

func (i *IsoMovementState) OnDirectionKeys(elapsed float64, movementVector [2]int) {
	i.engine.isoCamera.ChangePosition(movementVector, float32(elapsed))
	i.engine.UpdateMousePicking(i.lastMouseX, i.lastMouseY)
}

func (i *IsoMovementState) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	i.lastMouseX = newX
	i.lastMouseY = newY
	i.engine.UpdateMousePicking(newX, newY)
}
