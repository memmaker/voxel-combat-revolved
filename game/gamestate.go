package game

import "github.com/go-gl/glfw/v3.3/glfw"

type GameState interface {
    OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64)
    OnDirectionKeys(movementVector [2]int, elapsed float64)
    OnMouseClicked(x float64, y float64)
    OnKeyPressed(key glfw.Key)
    OnUpperRightAction()
    OnUpperLeftAction()
    OnZoomIn(deltaTime float64)
    OnZoomOut(deltaTime float64)
    Init()
}
