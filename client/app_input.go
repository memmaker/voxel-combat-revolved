package client

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/voxel"
	"log"
	"os"
	"runtime/pprof"
)

func (a *BattleClient) isMouseExclusive() bool {
	return a.Window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled
}

func (a *BattleClient) isMouseInWindow() bool {
	if a.isMouseExclusive() {
		return true
	}
	if a.lastMousePosX > 0 && a.lastMousePosX < float64(a.WindowWidth) && a.lastMousePosY > 0 && a.lastMousePosY < float64(a.WindowHeight) {
		return true
	}
	return false
}

func (a *BattleClient) handleMousePosEvents(xpos float64, ypos float64) {
	if a.lastMousePosX == xpos && a.lastMousePosY == ypos {
		return
	}
	overActionBar := a.actionbar.IsMouseOver(xpos, ypos)
	if !overActionBar {
		a.state().OnMouseMoved(a.lastMousePosX, a.lastMousePosY, xpos, ypos)
	} else if overActionBar {
		toolTip := a.actionbar.HoverText()
		if toolTip != "" {
			a.Print(toolTip)
		}
	}
	a.lastMousePosX = xpos
	a.lastMousePosY = ypos
}
func (a *BattleClient) handleMouseButtonEvents(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if !a.isMouseInWindow() {
		return
	}
	if a.actionbar.IsMouseOver(a.lastMousePosX, a.lastMousePosY) {
		if button == glfw.MouseButtonLeft && action == glfw.Press {
			a.actionbar.OnMouseClicked(a.lastMousePosX, a.lastMousePosY)
		}
		return
	}
	if button == glfw.MouseButtonLeft {
		if action == glfw.Press {
			a.state().OnMouseClicked(a.lastMousePosX, a.lastMousePosY)
		} else if action == glfw.Release {
			a.state().OnMouseReleased(a.lastMousePosX, a.lastMousePosY)
		}
	}
}
func (a *BattleClient) pollInput(deltaTime float64) (bool, [2]int) {
	cameraMoved := false
	movementVector := [2]int{0, 0}
	if a.Window.GetKey(glfw.KeyW) == glfw.Press {
		movementVector[1] = 1
		cameraMoved = true
	} else if a.Window.GetKey(glfw.KeyS) == glfw.Press {
		movementVector[1] = -1
		cameraMoved = true
	}
	if a.Window.GetKey(glfw.KeyA) == glfw.Press {
		movementVector[0] = -1
		cameraMoved = true
	} else if a.Window.GetKey(glfw.KeyD) == glfw.Press {
		movementVector[0] = 1
		cameraMoved = true
	}

	if a.Window.GetKey(glfw.KeyE) == glfw.Press {
		a.state().OnUpperRightAction(deltaTime)
	} else if a.Window.GetKey(glfw.KeyQ) == glfw.Press {
		a.state().OnUpperLeftAction(deltaTime)
	}

	return cameraMoved, movementVector
}

func (a *BattleClient) handleScrollEvents(xoff float64, yoff float64) {
	if !a.isMouseInWindow() {
		return
	}
	a.scheduleUpdate(func(deltaTime float64) {
		a.state().OnScroll(deltaTime, xoff, yoff)
	})
}
func (a *BattleClient) handleKeyEvents(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	/*
	if key == glfw.KeyE && action == glfw.Press {
			a.state().OnUpperRightAction(0)
	}

	if key == glfw.KeyQ && action == glfw.Press {
			a.state().OnUpperLeftAction(0)
	}

	*/
	if key == glfw.KeyF6 && action == glfw.Press {
		a.ToggleFullscren()
	}
	if key == glfw.KeyF7 && action == glfw.Press {
		//a.player.SetHeight(1.9 * 0.5)
		a.smoker.AddPoisonCloud(voxel.PositionToGridInt3(a.groundSelector.GetPosition()), 5, 1)
		//a.smoker.AddFire(voxel.PositionToGridInt3(a.groundSelector.GetPosition()), 5)
		//a.CreateSmokeEffect()
	}
	if key == glfw.KeyF9 && action == glfw.Press {
		//a.player.SetHeight(1.9 * 0.5)
		//a.smoker.ClearAllAbove()
		//util.MustSend(a.server.DebugRequest(""))
	}
	if key == glfw.KeyF10 && action == glfw.Press {
		//a.player.SetHeight(1.9 * 0.5)
		a.SwitchToEditMap()
	}

	if key == glfw.KeyF11 && action == glfw.Press {
		pprof.StopCPUProfile()
		f, err := os.Create("cpu_running.prof")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		//defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		//defer pprof.StopCPUProfile()
	}

	if key == glfw.KeyF12 && action == glfw.Press {
		a.Window.SetShouldClose(true)
		return
	}
	if action == glfw.Press {
		a.state().OnKeyPressed(key)
	}
}

func (a *BattleClient) captureMouse() {
	a.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	if glfw.RawMouseMotionSupported() {
		a.Window.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}
}

func (a *BattleClient) freeMouse() {
	a.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	if glfw.RawMouseMotionSupported() {
		a.Window.SetInputMode(glfw.RawMouseMotion, glfw.False)
	}
}

func (a *BattleClient) debugToggleWireFrame() {
	if a.wireFrame {
		a.wireFrame = false
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	} else {
		a.wireFrame = true
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}
