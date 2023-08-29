package client

import "github.com/go-gl/glfw/v3.3/glfw"

type GameState interface {
	OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64)
	OnDirectionKeys(elapsed float64, movementVector [2]int)
	OnMouseClicked(x float64, y float64)
	OnMouseReleased(x float64, y float64)
	OnKeyPressed(key glfw.Key)
	OnUpperRightAction(deltaTime float64)
	OnUpperLeftAction(deltaTime float64)
	OnScroll(deltaTime float64, xoff float64, yoff float64)
	Init(wasPopped bool)
	OnServerMessage(msgType string, json string)
}
