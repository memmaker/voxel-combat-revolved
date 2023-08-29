package util

import (
	"fmt"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"math"
	"os"
	"unsafe"
)

type GlApplication struct {
	Window             *glfw.Window
	TerminateFunc      func()
	UpdateFunc         func(elapsed float64)
	DrawFunc           func(elapsed float64)
	KeyHandler         func(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey)
	MousePosHandler    func(xpos float64, ypos float64)
	MouseButtonHandler func(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey)
	ScrollHandler      func(xoff float64, yoff float64)
	WindowWidth        int
	WindowHeight       int
	ticks              uint64
	FramesPerSecond    float64
	FPSRunningAvg      float64
	FPSMin             float64
	FPSMax             float64
}

func (a *GlApplication) KeyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if a.KeyHandler != nil {
		a.KeyHandler(
			key,
			scancode,
			action,
			mods,
		)
	}
}
func (a *GlApplication) MousePosCallback(w *glfw.Window, xpos float64, ypos float64) {
	if a.MousePosHandler != nil {
		a.MousePosHandler(xpos, ypos)
	}
}

func (a *GlApplication) ScrollCallback(w *glfw.Window, xoff float64, yoff float64) {
	if a.ScrollHandler != nil {
		a.ScrollHandler(xoff, yoff)
	}
}

func (a *GlApplication) MouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if a.MouseButtonHandler != nil {
		a.MouseButtonHandler(button, action, mods)
	}
}
func (a *GlApplication) Run() {
	defer a.TerminateFunc()
	previousTime := glfw.GetTime()
	// Start Render Loop
	shouldQuit := false
	time := glfw.GetTime()
	for !shouldQuit {
		if a.Window.ShouldClose() {
			shouldQuit = true
		}

		// ClearFlat the window.
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Update
		time = glfw.GetTime()
		elapsed := time - previousTime
		previousTime = time
		a.UpdateFunc(elapsed)

		a.DrawFunc(elapsed)

		a.FramesPerSecond = 1.0 / elapsed
		if a.ticks%60 == 0 {
			sixtyTicksAverage := a.FPSRunningAvg
			a.Window.SetTitle(fmt.Sprintf("FPS: %.0f (Avg: %.0f, Min: %.0f, Max: %.0f) / Elapsed: %.3f", a.FramesPerSecond, sixtyTicksAverage, a.FPSMin, a.FPSMax, elapsed*1000))
			a.FPSRunningAvg = 0 + a.FramesPerSecond*(1.0/60.0)
			a.FPSMin = math.MaxFloat64
			a.FPSMax = 0
		} else {
			a.FPSRunningAvg = a.FPSRunningAvg + a.FramesPerSecond*(1.0/60.0)
			if a.FramesPerSecond < a.FPSMin {
				a.FPSMin = a.FramesPerSecond
			}
			if a.FramesPerSecond > a.FPSMax {
				a.FPSMax = a.FramesPerSecond
			}
		}

		a.Window.SwapBuffers()
		glfw.PollEvents()
		a.ticks++
	}
}

func InitOpenGL(title string, width, height int) (*glfw.Window, func()) {
	var win *glfw.Window
	glErr := glfw.Init()
	if glErr != nil {
		println("glfw: ", glErr.Error())
		panic(glErr)
		return nil, nil
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)

	var err error

	win, err = glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(err)
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(1) // enable (1) vsync

	glhf.Init()

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.DEPTH_TEST)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	/*
	if runtime.GOOS != "darwin" {
		gl.Enable(gl.DEBUG_OUTPUT)
		gl.DebugMessageCallback(gl.DebugProc(glErrorHandler), gl.Ptr(nil))
	}
	*/

	return win, func() {
		glfw.Terminate()
	}
}

func glErrorHandler(source uint32, gltype uint32, id uint32, severity uint32, length int32, message string, param unsafe.Pointer) {
	errorMessage := fmt.Sprintf("source: %d, type: %d, id: %d, severity: %d, length: %d, param: %d, message:\n%s", source, gltype, id, severity, length, param, message)
	println(errorMessage)
}

func MustLoadTexture(filePath string) *glhf.Texture {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	texture, err := NewTextureFromReader(file, false)
	if err != nil {
		panic(err)
	}
	return texture
}

func Get2DPixelCoordOrthographicProjectionMatrix(width, height int) mgl32.Mat4 {
	// we want 0,0 to be at the top left
	return mgl32.Ortho2D(0, float32(width), float32(height), 0)
}

func Get2DOrthographicProjectionMatrix() mgl32.Mat4 {
	return mgl32.Ortho(0, 1, 1, 0, 0, 0.15)
}
