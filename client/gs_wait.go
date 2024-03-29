package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

type GameStateWaitForEvents struct {
	IsoMovementState
}

func (g *GameStateWaitForEvents) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateWaitForEvents) OnServerMessage(msgType string, json string) {

}

func (g *GameStateWaitForEvents) OnMouseClicked(x float64, y float64) {

}

func (g *GameStateWaitForEvents) OnKeyPressed(key glfw.Key) {
	g.IsoMovementState.OnKeyPressed(key)
}

func (g *GameStateWaitForEvents) Init(wasPopped bool) {
	g.engine.highlights.ClearAll()
	g.engine.SwitchToGroundSelector()
	g.engine.actionbar.Hide()
}
