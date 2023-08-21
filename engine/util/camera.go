package util

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Camera interface {
	GetTransformMatrix() mgl32.Mat4
	GetProjectionMatrix() mgl32.Mat4
	GetForward() mgl32.Vec3
	GetFrustumPlanes() []mgl32.Vec4
	GetPosition() mgl32.Vec3
	MoveInDirection(delta float32, dir [2]int)
	GetNearPlaneDist() float32
	SetPosition(position mgl32.Vec3)
	SetRotation(rotation mgl32.Quat)
	GetTransform() Transform
}
type CamAnimator struct {
	currentAnimation *CameraAnimation
}

func (c *CamAnimator) Update(deltaTime float64) {
	if c.currentAnimation != nil {
		c.currentAnimation.Update(deltaTime)
		if c.currentAnimation.IsFinished() {
			c.currentAnimation = nil
		}
	}
}
func (c *CamAnimator) IsCurrentlyAnimating() bool {
	return c.currentAnimation != nil
}

type PerspectiveTransform struct {
	*Transform
	fov           float32
	defaultFOV    float32
	windowWidth   int
	windowHeight  int
	nearPlaneDist float32
	farPlaneDist  float32
}

// GetProjectionMatrix returns the projection matrix for the camera.
// A projection matrix will transform a point from camera space to screen space. (3D -> 2D)
func (c *PerspectiveTransform) GetProjectionMatrix() mgl32.Mat4 {
	return mgl32.Perspective(mgl32.DegToRad(c.fov), float32(c.windowWidth)/float32(c.windowHeight), c.nearPlaneDist, c.farPlaneDist)
}

func (c *PerspectiveTransform) GetFrustumPlanes() []mgl32.Vec4 {
	mat := c.GetProjectionMatrix().Mul4(c.GetTransformMatrix())
	c1, c2, c3, c4 := mat.Rows()
	return []mgl32.Vec4{
		c4.Add(c1),                      // left
		c4.Sub(c1),                      // right
		c4.Sub(c2),                      // top
		c4.Add(c2),                      // bottom
		c4.Mul(c.nearPlaneDist).Add(c3), // front
		c4.Mul(c.farPlaneDist).Sub(c3),  // back
	}
}

func (c *PerspectiveTransform) GetNearPlaneDist() float32 {
	return c.nearPlaneDist
}

func (c *PerspectiveTransform) GetAspectRatio() float32 {
	return float32(c.windowWidth) / float32(c.windowHeight)
}

func (c *PerspectiveTransform) GetScreenWidth() int {
	return c.windowWidth
}

func (c *PerspectiveTransform) GetScreenHeight() int {
	return c.windowHeight
}

func (c *PerspectiveTransform) ChangeFOV(change int, minimum uint) {
	minFOV := float32(minimum)
	maxFOV := c.defaultFOV
	newFOV := c.fov + float32(change)
	if newFOV < minFOV {
		newFOV = minFOV
	}
	if newFOV > maxFOV {
		newFOV = maxFOV
	}
	c.fov = newFOV
}

func (c *PerspectiveTransform) ResetFOV() {
	c.fov = c.defaultFOV
}
func NewDefaultPerspectiveTransform(name string, width, height int) *PerspectiveTransform {
	return &PerspectiveTransform{
		Transform: &Transform{
			translation: mgl32.Vec3{0, 0, 0},
			rotation:    mgl32.QuatIdent(),
			scale:       mgl32.Vec3{1, 1, 1},
			nameOfOwner: name,
		},
		fov:           float32(45.0),
		defaultFOV:    float32(45.0),
		nearPlaneDist: 0.15,
		farPlaneDist:  100.0,
		windowWidth:   width,
		windowHeight:  height,
	}
}

type CameraAnimation struct {
	*PerspectiveTransform
	start          Transform
	end            Transform
	duration       float64
	animationTimer float64
}

// NewCameraAnimation idea is, that this should also satisfy the Camera interface
func NewCameraAnimation(start, end Transform, duration float64, width, height int) *CameraAnimation {
	c := &CameraAnimation{
		PerspectiveTransform: NewDefaultPerspectiveTransform("Camera Animation", width, height),
		start:                start,
		end:                  end,
		duration:             duration,
	}
	c.init()
	return c
}
func (c *CameraAnimation) init() {
	c.SetPosition(c.start.GetPosition())
	c.SetRotation(c.start.GetRotation())
}

func (c *CameraAnimation) MoveInDirection(delta float32, dir [2]int) {

}

func (c *CameraAnimation) GetTransform() Transform {
	return *c.Transform
}

func (c *CameraAnimation) IsFinished() bool {
	return c.animationTimer >= c.duration
}

func (c *CameraAnimation) Update(delta float64) {
	// TODO: Add ease-in and ease-out
	c.animationTimer += delta
	percent := Clamp(c.animationTimer/c.duration, 0, 1)

	easingFactor := float32(EaseOutSine(percent))

	currentPosition := Lerp3(c.start.GetPosition(), c.end.GetPosition(), float64(easingFactor))
	currentRotation := c.end.GetRotation()

	if c.start.GetRotation() != c.end.GetRotation() {
		currentRotation = mgl32.QuatNlerp(c.start.GetRotation(), c.end.GetRotation(), easingFactor)
	}

	c.SetPosition(currentPosition)
	c.SetRotation(currentRotation)
}

// GetRayFromCameraPlane returns a ray from the camera plane to the far plane.
// NOTE: Length of ray is hardcoded to 100 units.
func GetRayFromCameraPlane(cam Camera, normalizedX float32, normalizedY float32) (mgl32.Vec3, mgl32.Vec3) {
	rayLength := float32(100)

	normalizedNearPos := mgl32.Vec4{normalizedX, normalizedY, cam.GetNearPlaneDist(), 1}
	normalizedFarPos := mgl32.Vec4{normalizedX, normalizedY, cam.GetNearPlaneDist() + rayLength, 1}

	proj := cam.GetProjectionMatrix()
	view := cam.GetTransformMatrix()
	projViewInverted := proj.Mul4(view).Inv()

	// project point from camera space to world space
	nearWorldPos := projViewInverted.Mul4x1(normalizedNearPos)
	farWorldPos := projViewInverted.Mul4x1(normalizedFarPos)
	// perspective divide
	rayStart := nearWorldPos.Vec3().Mul(1 / nearWorldPos.W())
	farPosCorrected := farWorldPos.Vec3().Mul(1 / farWorldPos.W())
	dir := rayStart.Sub(farPosCorrected).Normalize()
	rayEnd := rayStart.Add(dir.Mul(rayLength))
	return rayStart, rayEnd
}
