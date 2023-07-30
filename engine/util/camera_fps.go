package util

import (
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

type Camera interface {
	GetViewMatrix() mgl32.Mat4
	GetProjectionMatrix() mgl32.Mat4
	GetFront() mgl32.Vec3
	GetFrustumPlanes(matrix mgl32.Mat4) []mgl32.Vec4
	GetPosition() mgl32.Vec3
	ChangePosition(dir [2]int, delta float32)
}

type FPSCamera struct {
	cameraPos        mgl32.Vec3
	cameraFront      mgl32.Vec3
	cameraRight      mgl32.Vec3
	cameraUp         mgl32.Vec3
	fpsWalkDirection mgl32.Vec3
	rotatex          float32
	rotatey          float32
	lookSensitivity  float32
	invertedY        bool
	windowWidth      int
	windowHeight     int
}

func (c *FPSCamera) ChangePosition(dir [2]int, delta float32) {

}

func NewFPSCamera(pos mgl32.Vec3, windowWidth, windowHeight int) *FPSCamera {
	return &FPSCamera{
		cameraPos:       pos,
		cameraFront:     mgl32.Vec3{0, 0, -1},
		cameraUp:        mgl32.Vec3{0, 1, 0},
		lookSensitivity: 0.08,
		rotatey:         0,
		rotatex:         -90,
		invertedY:       true,
		windowWidth:     windowWidth,
		windowHeight:    windowHeight,
	}
}

// GetViewMatrix returns the view matrix for the camera.
// A view matrix will transform a point from world space to camera space.
func (c *FPSCamera) GetViewMatrix() mgl32.Mat4 {
	camera := mgl32.LookAtV(c.cameraPos, c.cameraPos.Add(c.cameraFront), c.cameraUp)
	return camera
}

// GetProjectionMatrix returns the projection matrix for the camera.
// A projection matrix will transform a point from camera space to screen space. (3D -> 2D)
func (c *FPSCamera) GetProjectionMatrix() mgl32.Mat4 {
	fov := float32(45.0)
	return mgl32.Perspective(mgl32.DegToRad(fov), float32(c.windowWidth)/float32(c.windowHeight), 0.15, 512.0)
}

// ChangeAngles changes the camera's angles by dx and dy.
// Used for mouse look in FPS look mode.
func (c *FPSCamera) ChangeAngles(dx, dy float32) {
	if mgl32.Abs(dx) > 200 || mgl32.Abs(dy) > 200 {
		return
	}
	c.rotatex += dx * c.lookSensitivity
	yChange := dy * c.lookSensitivity
	if c.invertedY {
		c.rotatey -= yChange
	} else {
		c.rotatey += yChange
	}
	if c.rotatey > 89 {
		c.rotatey = 89
	}
	if c.rotatey < -89 {
		c.rotatey = -89
	}
	c.updateAngles()
}
func (c *FPSCamera) ForwardBackward(delta float32) mgl32.Vec3 {
	return c.cameraFront.Mul(delta)
}

func (c *FPSCamera) PlanarForwardBackward(delta float32) mgl32.Vec3 {
	return c.fpsWalkDirection.Mul(delta)
}

func (c *FPSCamera) LeftRight(delta float32) mgl32.Vec3 {
	return c.cameraRight.Mul(delta)
}

func (c *FPSCamera) UpDown(delta float32) mgl32.Vec3 {
	return mgl32.Vec3{0, 1, 0}.Mul(delta)
}

func (c *FPSCamera) updateAngles() {
	front := mgl32.Vec3{
		Cos(ToRadian(c.rotatey)) * Cos(ToRadian(c.rotatex)),
		Sin(ToRadian(c.rotatey)),
		Cos(ToRadian(c.rotatey)) * Sin(ToRadian(c.rotatex)),
	}
	c.cameraFront = front.Normalize()
	c.cameraRight = c.cameraFront.Cross(mgl32.Vec3{0, 1, 0}).Normalize()
	c.cameraUp = c.cameraRight.Cross(c.cameraFront).Normalize()
	c.fpsWalkDirection = mgl32.Vec3{0, 1, 0}.Cross(c.cameraRight).Normalize()
}

func (c *FPSCamera) GetPosition() mgl32.Vec3 {
	return c.cameraPos
}

func (c *FPSCamera) GetFront() mgl32.Vec3 {
	return c.cameraFront
}

func (c *FPSCamera) SetPosition(pos mgl32.Vec3) {
	c.cameraPos = pos
}

func (c *FPSCamera) FPSLookAt(position mgl32.Vec3) {
	front := position.Sub(c.cameraPos).Normalize()
	//dist := front.Len()

	c.rotatex = mgl32.RadToDeg(float32(math.Atan2(float64(front.Z()), float64(front.X()))))
	c.rotatey = mgl32.RadToDeg(float32(math.Asin(float64(front.Y()))))
	c.updateAngles()
}

func (c *FPSCamera) GetFrustumPlanes(projection mgl32.Mat4) []mgl32.Vec4 {
	mat := projection.Mul4(c.GetViewMatrix())
	c1, c2, c3, c4 := mat.Rows()
	return []mgl32.Vec4{
		c4.Add(c1),            // left
		c4.Sub(c1),            // right
		c4.Sub(c2),            // top
		c4.Add(c2),            // bottom
		c4.Mul(0.15).Add(c3),  // front
		c4.Mul(512.0).Sub(c3), // back
	}
}
