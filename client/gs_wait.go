package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

type GameStateWaitForEvents struct {
	IsoMovementState
}

func (g *GameStateWaitForEvents) OnMouseClicked(x float64, y float64) {

}

func (g *GameStateWaitForEvents) OnKeyPressed(key glfw.Key) {

}

func (g *GameStateWaitForEvents) Init(wasPopped bool) {
	g.engine.GetVoxelMap().ClearHighlights()
	g.engine.SwitchToGroundSelector()
	g.engine.actionbar.Hide()
}
