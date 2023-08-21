package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
)

type ISOCamera struct {
	*PerspectiveTransform
	rotationAngle float32
	forward       mgl32.Vec3
	up            mgl32.Vec3
	lookTarget    mgl32.Vec3
	camDistance   float32
}

func (c *ISOCamera) GetTransform() Transform {
	return *c.Transform
}

func NewISOCamera(windowWidth, windowHeight int) *ISOCamera {
	i := &ISOCamera{
		PerspectiveTransform: NewDefaultPerspectiveTransform("ISO Camera", windowWidth, windowHeight),
		forward:              mgl32.Vec3{1, -1, 0}.Normalize(),
		camDistance:          float32(10),
		lookTarget:           mgl32.Vec3{0, 0, 0},
		rotationAngle:        45,
	}
	i.Transform.SetName("ISO Camera")
	i.updateTransform()
	return i
}

func (c *ISOCamera) getLookAt(camPos, target mgl32.Vec3) mgl32.Mat4 {
	lookDirection := target.Sub(camPos).Normalize()
	right := lookDirection.Cross(mgl32.Vec3{0, 1, 0})
	c.up = right.Cross(lookDirection).Normalize()
	lookAtMatrix := mgl32.QuatLookAtV(camPos, target, c.up).Mat4()
	return lookAtMatrix
}

func (c *ISOCamera) RotateRight(delta float64) { // rotate around y axis by 45 degrees
	rotationSpeed := 70
	c.rotationAngle += float32(delta) * float32(rotationSpeed)
	c.updateTransform()
}

func (c *ISOCamera) RotateLeft(delta float64) { // rotate around y axis by -45 degrees
	rotationSpeed := 70
	c.rotationAngle -= float32(delta) * float32(rotationSpeed)
	c.updateTransform()
}

func (c *ISOCamera) MoveInDirection(delta float32, dir [2]int) {
	scrollSpeed := float32(10.0)
	right := mgl32.Vec3{1, 0, 0}.Vec4(0)
	forwardOnPlane := mgl32.Vec3{0, 0, -1}.Vec4(0)
	rotationMatrix := mgl32.QuatRotate(mgl32.DegToRad(c.rotationAngle), mgl32.Vec3{0, 1, 0}).Mat4().Inv()
	right = rotationMatrix.Mul4x1(right).Normalize()
	forwardOnPlane = rotationMatrix.Mul4x1(forwardOnPlane).Normalize()

	moveBy := right.Mul(float32(dir[0]) * delta * scrollSpeed).Add(forwardOnPlane.Mul(float32(dir[1]) * delta * -scrollSpeed))

	c.lookTarget = c.lookTarget.Add(moveBy.Vec3())
	c.updateTransform()
}

func (c *ISOCamera) GetPickingRayFromScreenPosition(x float64, y float64) (mgl32.Vec3, mgl32.Vec3) {
	// normalize x and y to -1..1
	normalizedX := (float32(x)/float32(c.windowWidth))*2 - 1
	normalizedY := ((float32(y)/float32(c.windowHeight))*2 - 1) * -1

	return GetRayFromCameraPlane(c, normalizedX, normalizedY)
}

func (c *ISOCamera) ZoomIn(deltaTime float64, amount float64) {
	c.camDistance -= float32(amount) * float32(deltaTime) * 3
	c.updateTransform()
}

func (c *ISOCamera) ZoomOut(deltaTime float64, amount float64) {
	c.camDistance += float32(amount) * float32(deltaTime) * 3
	c.updateTransform()
}

func (c *ISOCamera) CenterOn(targetPos mgl32.Vec3) {
	//end := NewTransform(targetPos.Sub(c.relativeLookTarget), c.GetRotation(), mgl32.Vec3{1, 1, 1})
	//c.StartAnimation(*c.Transform, *end, 0.5)
	// TESTCASE #1
	//c.cameraPos = targetPos.Sub(c.relativeLookTarget)
	c.lookTarget = targetPos
	c.updateTransform()
}

func (c *ISOCamera) updateTransform() {
	// rotate around to face the object
	// translate to our camera offset
	// rotate around y to face the object again
	// translate everything to the lookTarget translation
	camOffset := c.forward.Mul(c.camDistance) // mgl32.Vec3{-1, 1, 0}
	target := c.lookTarget.Mul(-1)
	camPosFromTarget := target.Sub(camOffset)
	camOffsetInverted := camOffset

	translation := mgl32.Translate3D(camOffsetInverted.X(), camOffsetInverted.Y(), camOffsetInverted.Z())
	translateToLookTarget := mgl32.Translate3D(target.X(), target.Y(), target.Z())

	rotationAround := mgl32.QuatRotate(mgl32.DegToRad(c.rotationAngle), mgl32.Vec3{0, 1, 0}).Mat4()
	lookAtMatrix := c.getLookAt(camPosFromTarget, target)
	viewMatrix := lookAtMatrix.Mul4(translation).Mul4(rotationAround).Mul4(translateToLookTarget)

	camPos := ExtractPosition(viewMatrix)
	camRot := ExtractRotation(viewMatrix)

	c.Transform.SetPosition(camPos)
	c.Transform.SetRotation(camRot)
}
func (c *ISOCamera) GetPosition() mgl32.Vec3 {
	rotationAround := mgl32.QuatRotate(mgl32.DegToRad(c.rotationAngle), mgl32.Vec3{0, 1, 0}).Mat4().Inv()
	return rotationAround.Mul4x1(c.forward.Mul(-c.camDistance).Vec4(1)).Add(c.lookTarget.Vec4(1)).Vec3()
}

func (c *ISOCamera) GetForward() mgl32.Vec3 {
	return c.forward
}
func (c *ISOCamera) GetUp() mgl32.Vec3 {
	return c.up
}

func (c *ISOCamera) DebugString() string {
	return fmt.Sprintf("CameraPos: %v\nLookTarget: %v\nCamDir: %v\nCamDist: %0.2f\nRotationAngle: %0.2f\n", c.GetPosition(), c.lookTarget, c.forward, c.camDistance, c.rotationAngle)
}
