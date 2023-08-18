package util

import (
	"encoding/json"
	"fmt"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
	"os"
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

		// Clear the window.
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

	var err error

	win, err = glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(err)
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(1) // enable (1) vsync

	glhf.Init()
	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.DEPTH_TEST)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	return win, func() {
		glfw.Terminate()
	}
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

type Transform struct {
	position    mgl32.Vec3
	rotation    mgl32.Quat
	scale       mgl32.Vec3
	nameOfOwner string
}

func (t *Transform) GetName() string {
	return t.nameOfOwner
}
func (t *Transform) SetName(name string) {
	t.nameOfOwner = name
}
func NewDefaultTransform(name string) *Transform {
	return &Transform{
		position:    mgl32.Vec3{0, 0, 0},
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
		nameOfOwner: name,
	}
}

func NewTransform(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) *Transform {
	return &Transform{
		position: position,
		rotation: rotation,
		scale:    scale,
	}
}

func (t *Transform) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Position mgl32.Vec3 `json:"position"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}{
		Position: t.position,
		Rotation: t.rotation,
		Scale:    t.scale,
	})
}

func (t *Transform) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Position mgl32.Vec3 `json:"position"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	t.position = tmp.Position
	t.rotation = tmp.Rotation
	t.scale = tmp.Scale
	return nil
}
func (t *Transform) GetTransformMatrix() mgl32.Mat4 {
	translation := mgl32.Translate3D(t.position.X(), t.position.Y(), t.position.Z())
	rotation := t.rotation.Mat4()
	scale := mgl32.Scale3D(t.scale.X(), t.scale.Y(), t.scale.Z())
	return translation.Mul4(rotation).Mul4(scale)
}
func (t *Transform) GetPosition() mgl32.Vec3 {
	return t.position
}

func (t *Transform) GetBlockPosition() voxel.Int3 {
	return voxel.PositionToGridInt3(t.GetPosition())
}

func (t *Transform) GetRotation() mgl32.Quat {
	return t.rotation
}
func (t *Transform) GetForward() mgl32.Vec3 {
	return t.rotation.Rotate(mgl32.Vec3{0, 0, -1})
}

func (t *Transform) GetForward2DCardinal() voxel.Int3 {
	forward := t.GetForward()
	gridForward := voxel.DirectionToGridInt3(forward)
	cardinalForward := gridForward.ToCardinalDirection()
	return cardinalForward
}
func (t *Transform) GetScale() mgl32.Vec3 {
	return t.scale
}

func (t *Transform) setYRotationAngle(angle float32) {
	t.rotation = mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0})
	println(fmt.Sprintf("[Transform] setYRotationAngle for %s: %v", t.GetName(), angle))
}

func (t *Transform) SetForward2D(forward mgl32.Vec3) {
	t.setYRotationAngle(DirectionToAngleVec(forward))
}

func (t *Transform) SetForward2DCardinal(forward voxel.Int3) {
	t.setYRotationAngle(DirectionToAngle(forward))
}

func (t *Transform) SetBlockPosition(position voxel.Int3) {
	t.SetPosition(position.ToBlockCenterVec3())
}
func (t *Transform) SetPosition(position mgl32.Vec3) {
	t.position = position
}
