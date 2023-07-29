package game

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"log"
	"os"
	"runtime/pprof"
)

func (a *BattleGame) isMouseExclusive() bool {
	return a.Window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled
}

func (a *BattleGame) isMouseInWindow() bool {
	if a.mousePosX > 0 && a.mousePosX < float64(a.WindowWidth) && a.mousePosY > 0 && a.mousePosY < float64(a.WindowHeight) {
		return true
	}
	return false
}

func (a *BattleGame) handleMousePosEvents(xpos float64, ypos float64) {
	if a.mousePosX == xpos && a.mousePosY == ypos {
		return
	}
	a.mousePosX = xpos
	a.mousePosY = ypos
	a.lastMousePosX = xpos
	a.lastMousePosY = ypos
	if a.state().OnMouseMoved != nil {
		a.state().OnMouseMoved(a.lastMousePosX, a.lastMousePosY, a.mousePosX, a.mousePosY)
	}
	//a.RayCast()
}

func (a *BattleGame) pollInput(deltaTime float64) (bool, [2]int) {
	cameraMoved := false
	movementVector := [2]int{0, 0}
	if a.Window.GetKey(glfw.KeyW) == glfw.Press {
		movementVector[1]--
		cameraMoved = true
	}
	if a.Window.GetKey(glfw.KeyS) == glfw.Press {
		movementVector[1]++
		cameraMoved = true
	}
	if a.Window.GetKey(glfw.KeyA) == glfw.Press {
		movementVector[0]--
		cameraMoved = true
	}
	if a.Window.GetKey(glfw.KeyD) == glfw.Press {
		movementVector[0]++
		cameraMoved = true
	}

	return cameraMoved, movementVector
}
func (a *BattleGame) handleMouseButtonEvents(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if !a.isMouseInWindow() {
		return
	}
	if button == glfw.MouseButtonLeft && action == glfw.Press {
		a.state().OnMouseClicked(a.mousePosX, a.mousePosY)
	}
}

func (a *BattleGame) handleScrollEvents(xoff float64, yoff float64) {
	if !a.isMouseInWindow() {
		return
	}
	a.scheduleUpdate(func(deltaTime float64) {
		a.state().OnScroll(deltaTime, xoff, yoff)
	})
}
func (a *BattleGame) handleKeyEvents(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.KeyEscape && action == glfw.Press {
		a.freeMouse()
		return
	}

	if key == glfw.KeyE && action == glfw.Press {
		a.state().OnUpperRightAction()
	}

	if key == glfw.KeyQ && action == glfw.Press {
		a.state().OnUpperLeftAction()
	}

	if key == glfw.Key1 && action == glfw.Press {
		a.blockTypeToPlace = a.blockTypeToPlace - 1
		if a.blockTypeToPlace < 1 {
			a.blockTypeToPlace = 1
		}
	}

	if key == glfw.Key2 && action == glfw.Press {
		a.blockTypeToPlace = a.blockTypeToPlace + 1
	}

	if key == glfw.KeyF1 && action == glfw.Press {
		a.showDebugInfo = !a.showDebugInfo
		if !a.showDebugInfo {
			a.textLabel = nil
		} else {
			a.timer.Reset()
		}
	}
	if key == glfw.KeyF2 && action == glfw.Press {
		a.debugToggleWireFrame()
	}
	if key == glfw.KeyF3 && action == glfw.Press {
		a.drawBoundingBoxes = !a.drawBoundingBoxes
	}

	if key == glfw.KeyF4 && action == glfw.Press {
		/*for _, coord := range a.lastVisitedBlocks {
		    a.PlaceBlock(coord, voxel.NewTestBlock())
		}*/
	}

	if key == glfw.KeyF5 && action == glfw.Press {
		a.SpawnProjectile(a.camera.GetPosition().Add(a.camera.GetFront()), a.camera.GetFront().Mul(8000))
	}
	if key == glfw.KeyF6 && action == glfw.Press {
		a.SpawnProjectile(a.camera.GetPosition().Add(a.camera.GetFront()), a.camera.GetFront().Mul(8))
	}

	if key == glfw.KeyF7 && action == glfw.Press {
		//a.player.SetHeight(1.9 * 0.5)
		a.SwitchToEditMap()
	}

	if key == glfw.KeyF8 && action == glfw.Press {
		//a.player.SetHeight(1.9)
	}

	if key == glfw.KeyF9 && action == glfw.Press {
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

func (a *BattleGame) captureMouse() {
	a.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	if glfw.RawMouseMotionSupported() {
		a.Window.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}
}

func (a *BattleGame) freeMouse() {
	a.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	if glfw.RawMouseMotionSupported() {
		a.Window.SetInputMode(glfw.RawMouseMotion, glfw.False)
	}
}

func (a *BattleGame) debugToggleWireFrame() {
	if a.wireFrame {
		a.wireFrame = false
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	} else {
		a.wireFrame = true
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}
