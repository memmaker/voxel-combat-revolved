package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
)

type ISOCamera struct {
	*Transform
	windowWidth     int
	windowHeight    int
	nearPlaneDist   float32
	fov             float32
	defaultFOV      float32
	rotationAngle   float32
	camDirection    mgl32.Vec3
	lookTarget      mgl32.Vec3
	transformMatrix mgl32.Mat4
	camDistance     float32
}

func (c *ISOCamera) GetTransform() Transform {
	return *c.Transform
}

func NewISOCamera(windowWidth, windowHeight int) *ISOCamera {
	i := &ISOCamera{
		Transform:     NewDefaultTransform("ISO Camera"),
		camDirection:  mgl32.Vec3{-1, 1, 0}.Normalize(),
		camDistance:   float32(10),
		lookTarget:    mgl32.Vec3{0, 0, 0},
		windowWidth:   windowWidth,
		windowHeight:  windowHeight,
		nearPlaneDist: 0.15,
		rotationAngle: 45,
		fov:           float32(45.0),
		defaultFOV:    float32(45.0),
	}
	i.Transform.SetName("ISO Camera")
	i.updateMatrix()
	return i
}

// GetProjectionMatrix returns the projection matrix for the camera.
// A projection matrix will transform a point from camera space to screen space. (3D -> 2D)
func (c *ISOCamera) GetProjectionMatrix() mgl32.Mat4 {
	return mgl32.Perspective(mgl32.DegToRad(c.fov), float32(c.windowWidth)/float32(c.windowHeight), c.nearPlaneDist, 512.0)
}
func (c *ISOCamera) getLookAt(camPos, target mgl32.Vec3) mgl32.Mat4 {
	lookDirection := target.Sub(camPos).Normalize()
	right := lookDirection.Cross(mgl32.Vec3{0, 1, 0})
	up := right.Cross(lookDirection)
	lookAtMatrix := mgl32.QuatLookAtV(camPos, target, up).Mat4()
	return lookAtMatrix
}

func (c *ISOCamera) RotateRight(delta float64) { // rotate around y axis by 45 degrees
	rotationSpeed := 70
	c.rotationAngle += float32(delta) * float32(rotationSpeed)
	c.updateMatrix()
}

func (c *ISOCamera) RotateLeft(delta float64) { // rotate around y axis by -45 degrees
	rotationSpeed := 70
	c.rotationAngle -= float32(delta) * float32(rotationSpeed)
	c.updateMatrix()
}

func (c *ISOCamera) ChangePosition(delta float32, dir [2]int) {
	scrollSpeed := float32(10.0)
	right := mgl32.Vec3{1, 0, 0}.Vec4(0)
	forwardOnPlane := mgl32.Vec3{0, 0, -1}.Vec4(0)
	rotationMatrix := mgl32.QuatRotate(mgl32.DegToRad(c.rotationAngle), mgl32.Vec3{0, 1, 0}).Mat4().Inv()
	right = rotationMatrix.Mul4x1(right).Normalize()
	forwardOnPlane = rotationMatrix.Mul4x1(forwardOnPlane).Normalize()

	moveBy := right.Mul(float32(dir[0]) * delta * scrollSpeed).Add(forwardOnPlane.Mul(float32(dir[1]) * delta * -scrollSpeed))

	c.lookTarget = c.lookTarget.Add(moveBy.Vec3())
	c.updateMatrix()
}

func (c *ISOCamera) GetFrustumPlanes(projection mgl32.Mat4) []mgl32.Vec4 {
	mat := projection.Mul4(c.GetTransformMatrix())
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

func (c *ISOCamera) GetNearPlaneDist() float32 {
	return c.nearPlaneDist
}

func (c *ISOCamera) GetPickingRayFromScreenPosition(x float64, y float64) (mgl32.Vec3, mgl32.Vec3) {
	// normalize x and y to -1..1
	normalizedX := (float32(x)/float32(c.windowWidth))*2 - 1
	normalizedY := ((float32(y)/float32(c.windowHeight))*2 - 1) * -1

	return GetRayFromCameraPlane(c, normalizedX, normalizedY)
}

func (c *ISOCamera) ZoomIn(deltaTime float64, amount float64) {
	c.camDistance -= float32(amount) * float32(deltaTime) * 3
	c.updateMatrix()
}

func (c *ISOCamera) ZoomOut(deltaTime float64, amount float64) {
	c.camDistance += float32(amount) * float32(deltaTime) * 3
	c.updateMatrix()
}

func (c *ISOCamera) CenterOn(targetPos mgl32.Vec3) {
	//end := NewTransform(targetPos.Sub(c.relativeLookTarget), c.GetRotation(), mgl32.Vec3{1, 1, 1})
	//c.StartAnimation(*c.Transform, *end, 0.5)
	// TESTCASE #1
	//c.cameraPos = targetPos.Sub(c.relativeLookTarget)
	c.lookTarget = targetPos
	c.updateMatrix()
}

func (c *ISOCamera) updateMatrix() {
	// rotate around to face the object
	// translate to our camera offset
	// rotate around y to face the object again
	// translate everything to the lookTarget position
	camOffset := c.camDirection.Mul(-c.camDistance)
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

func (c *ISOCamera) DebugString() string {
	return fmt.Sprintf("CameraPos: %v\nLookTarget: %v\nCamDir: %v\nCamDist: %0.2f\nRotationAngle: %0.2f\n", c.Transform.GetPosition(), c.lookTarget, c.camDirection, c.camDistance, c.rotationAngle)
}
