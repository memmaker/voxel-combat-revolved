package util

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Camera interface {
	GetTransformMatrix() mgl32.Mat4
	GetProjectionMatrix() mgl32.Mat4
	GetForward() mgl32.Vec3
	GetFrustumPlanes(matrix mgl32.Mat4) []mgl32.Vec4
	GetPosition() mgl32.Vec3
	ChangePosition(delta float32, dir [2]int)
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

// NewCameraAnimation idea is, that this should also satisfy the Camera interface
func NewCameraAnimation(camera Camera, start, end Transform, duration float64) *CameraAnimation {
	return &CameraAnimation{
		camera:   camera,
		start:    start,
		end:      end,
		duration: duration,
	}
}

type CameraAnimation struct {
	start          Transform
	end            Transform
	duration       float64
	animationTimer float64
	camera         Camera
}

func (c *CameraAnimation) IsFinished() bool {
	return c.animationTimer >= c.duration
}

func (c *CameraAnimation) Update(delta float64) {
	c.animationTimer += delta

	percent := Clamp(c.animationTimer/c.duration, 0, 1)
	currentPosition := Lerp3(c.start.GetPosition(), c.end.GetPosition(), percent)
	currentRotation := c.end.GetRotation()
	if c.start.GetRotation() != c.end.GetRotation() {
		currentRotation = mgl32.QuatSlerp(c.start.GetRotation(), c.end.GetRotation(), float32(percent))
	}
	// vs..
	// currentRotation := LerpQuat(c.start.GetRotation(), c.end.GetRotation(), percent)

	c.camera.SetPosition(currentPosition)
	c.camera.SetRotation(currentRotation)
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
