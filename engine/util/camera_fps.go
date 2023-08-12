package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math"
	"math/rand"
)

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
	nearPlaneDist    float32
	fov              float32
}

func (c *FPSCamera) GetNearPlaneDist() float32 {
	return c.nearPlaneDist
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
		nearPlaneDist:   0.15,
		invertedY:       true,
		windowWidth:     windowWidth,
		windowHeight:    windowHeight,
		fov:             float32(45.0),
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
	aspect := float32(c.windowWidth) / float32(c.windowHeight)
	return mgl32.Perspective(mgl32.DegToRad(c.fov), aspect, c.nearPlaneDist, 100.0)
}
func (c *FPSCamera) GetRandomRayInCircleFrustum(accuracy float64) (mgl32.Vec3, mgl32.Vec3) {
	accuracy = Clamp(accuracy, 0.0, 0.99)
	accFactor := 1.0 - accuracy // 0.01..1.0

	randX := rand.Float64()*2.0 - 1.0
	randY := rand.Float64()*2.0 - 1.0

	println(fmt.Sprintf("randNorm: %0.2f, %0.2f", randX, randY))

	lengthOfVector := math.Sqrt(randX*randX + randY*randY)
	if lengthOfVector > 1.0 {
		// normalize
		randX /= lengthOfVector
		randY /= lengthOfVector
	}
	println(fmt.Sprintf("circled: %0.2f, %0.2f", randX, randY))

	// in range -1.0..1.0
	randX *= accFactor
	randY *= accFactor

	println(fmt.Sprintf("acc. adjusted: %0.2f, %0.2f", randX, randY))
	//randX, randY = AdjustForAspectRatio(randX, randY, c.windowWidth, c.windowHeight) // from -1.0..1.0 to -n..n on the x axis
	return GetRayFromCameraPlane(c, float32(randX), float32(randY))
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
	if c.rotatey > 89 {
		c.rotatey = 89
	}
	if c.rotatey < -89 {
		c.rotatey = -89
	}
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

func (c *FPSCamera) GetRotation() (float32, float32) {
	return c.rotatex, c.rotatey
}

func (c *FPSCamera) Reposition(pos mgl32.Vec3, rotX float32, rotY float32) {
	c.cameraPos = pos
	c.rotatex = rotX
	c.rotatey = rotY
	c.updateAngles()
}

func (c *FPSCamera) SetFOV(fov float32) {
	c.fov = fov
}

func (c *FPSCamera) GetFOV() float32 {
	return c.fov
}

func (c *FPSCamera) GetAspectRatio() float32 {
	return float32(c.windowWidth) / float32(c.windowHeight)
}

func (c *FPSCamera) GetScreenWidth() int {
	return c.windowWidth
}

func (c *FPSCamera) GetScreenHeight() int {
	return c.windowHeight
}
