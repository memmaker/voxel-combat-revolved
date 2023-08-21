package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math"
	"math/rand"
)

type FPSCamera struct {
	*PerspectiveTransform
	cameraPos        mgl32.Vec3
	cameraFront      mgl32.Vec3
	cameraRight      mgl32.Vec3
	cameraUp         mgl32.Vec3
	fpsWalkDirection mgl32.Vec3
	rotatex          float32
	rotatey          float32
	lookSensitivity  float32
	invertedY        bool
}

func (c *FPSCamera) GetTransform() Transform {
	return *c.Transform
}
func (c *FPSCamera) SetRotation(rotation mgl32.Quat) {
	// we need to calculate the rotation angles from the quaternion
	camFront := mgl32.Vec3{0, 0, -1}
	camUp := mgl32.Vec3{0, 1, 0}
	camRight := mgl32.Vec3{1, 0, 0}

	// calculate rotation matrix from quaternion
	rotationMatrix := rotation.Mat4()

	// calculate camera front vector
	c.cameraFront = rotationMatrix.Mul4x1(camFront.Vec4(1)).Vec3()
	c.cameraUp = rotationMatrix.Mul4x1(camUp.Vec4(1)).Vec3()
	c.cameraRight = rotationMatrix.Mul4x1(camRight.Vec4(1)).Vec3()
	c.updateTransform()
}
func (c *FPSCamera) GetUp() mgl32.Vec3 {
	return c.cameraUp
}
func (c *FPSCamera) MoveInDirection(delta float32, dir [2]int) {
	currentPos := c.cameraPos
	moveVector := mgl32.Vec3{0, 0, 0}
	if dir[1] != 0 {
		moveVector = moveVector.Add(c.LeftRight(float32(dir[1]) * delta))
	}
	if dir[0] != 0 {
		moveVector = moveVector.Add(c.PlanarForwardBackward(float32(dir[0]) * delta))
	}
	c.cameraPos = currentPos.Add(moveVector)
	c.updateTransform()
}

func NewFPSCamera(pos mgl32.Vec3, windowWidth, windowHeight int) *FPSCamera {
	f := &FPSCamera{
		PerspectiveTransform: NewDefaultPerspectiveTransform("ISO Camera", windowWidth, windowHeight),
		cameraPos:            pos,
		cameraFront:          mgl32.Vec3{0, 0, -1},
		cameraUp:             mgl32.Vec3{0, 1, 0},
		lookSensitivity:      0.08,
		rotatey:              0,
		rotatex:              -90,
		invertedY:            true,
	}
	f.updateTransform()
	return f
}
func (c *FPSCamera) GetPosition() mgl32.Vec3 {
	return c.cameraPos
}
func (c *FPSCamera) SetInvertedY(inverted bool) {
	c.invertedY = inverted
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

	c.updateTransform()
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

func (c *FPSCamera) FPSLookAt(position mgl32.Vec3) {
	front := position.Sub(c.cameraPos).Normalize()
	//dist := front.Len()

	c.rotatex = mgl32.RadToDeg(float32(math.Atan2(float64(front.Z()), float64(front.X()))))
	c.rotatey = mgl32.RadToDeg(float32(math.Asin(float64(front.Y()))))
	c.updateTransform()
}

func (c *FPSCamera) GetRotation() (float32, float32) {
	return c.rotatex, c.rotatey
}

func (c *FPSCamera) Reposition(pos mgl32.Vec3, rotX float32, rotY float32) {
	c.cameraPos = pos
	c.rotatex = rotX
	c.rotatey = rotY
	c.updateTransform()
}

func (c *FPSCamera) SetFOV(fov float32) {
	c.fov = fov
}

func (c *FPSCamera) GetFOV() float32 {
	return c.fov
}
func (c *FPSCamera) SetPosition(pos mgl32.Vec3) {
	c.cameraPos = pos
	c.updateTransform()
}
func (c *FPSCamera) updateTransform() {
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

	cameraPosition := c.cameraPos
	viewMatrix := mgl32.LookAtV(cameraPosition, cameraPosition.Add(c.cameraFront), c.cameraUp)

	camPos := ExtractPosition(viewMatrix)
	camRot := ExtractRotation(viewMatrix)

	c.Transform.SetPosition(camPos)
	c.Transform.SetRotation(camRot)
}
