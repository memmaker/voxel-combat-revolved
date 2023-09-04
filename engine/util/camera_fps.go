package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math"
	"math/rand"
)

type FPSCamera struct {
	*PerspectiveTransform
	cameraFront      mgl32.Vec3
	cameraRight      mgl32.Vec3
	cameraUp         mgl32.Vec3
	fpsWalkDirection mgl32.Vec3
	rotatex          float32
	rotatey          float32
	lookSensitivity  float32
	invertedY        bool
	parent           Transformer
	lookTarget       mgl32.Vec3
}

func (c *FPSCamera) GetUp() mgl32.Vec3 {
	return c.cameraUp
}

func (c *FPSCamera) SetTransform(transform Transform) {
	c.Transform = &transform
}

func (c *FPSCamera) RotateRight(deltaTime float64) {

}

func (c *FPSCamera) RotateLeft(deltaTime float64) {

}

func (c *FPSCamera) GetPickingRayFromScreenPosition(x float64, y float64) (mgl32.Vec3, mgl32.Vec3) {
	// normalize x and y to -1..1
	normalizedX := (float32(x)/float32(c.windowWidth))*2 - 1
	normalizedY := ((float32(y)/float32(c.windowHeight))*2 - 1) * -1

	return GetRayFromCameraPlane(c, normalizedX, normalizedY)
}


func (c *FPSCamera) GetProjectionViewMatrix() mgl32.Mat4 {
	return c.GetProjectionMatrix().Mul4(c.GetViewMatrix())
}

func (c *FPSCamera) GetTransform() Transform {
	return *c.Transform
}
func (c *FPSCamera) MoveInDirection(delta float32, dir [2]int) {
	currentPos := c.GetPosition()
	moveVector := mgl32.Vec3{0, 0, 0}
	if dir[0] != 0 {
		moveVector = moveVector.Add(c.LeftRight(float32(dir[0]) * delta))
	}
	if dir[1] != 0 {
		moveVector = moveVector.Add(c.PlanarForwardBackward(float32(dir[1]) * delta))
	}
	c.SetPosition(currentPos.Add(moveVector))
	c.updateTransform()
}

func NewFPSCamera(pos mgl32.Vec3, windowWidth, windowHeight int, sensitivity float32) *FPSCamera {
	f := &FPSCamera{
		PerspectiveTransform: NewDefaultPerspectiveTransform("ISO Camera", windowWidth, windowHeight),
		cameraFront:          mgl32.Vec3{0, 0, -1},
		cameraUp:             mgl32.Vec3{0, 1, 0},
		lookSensitivity:      sensitivity,
		rotatey:              0,
		rotatex:              -90,
		invertedY:            true,
	}
	f.SetPosition(pos)
	f.updateTransform()
	return f
}

func (c *FPSCamera) GetViewMatrix() mgl32.Mat4 {
	if c.parent != nil {
		parentTransform := c.parent.GetTransformMatrix()
		offsetTrans := mgl32.Translate3D(0, 0.15, 0.2) // right behind our parent
		transformationMatrix := parentTransform.Mul4(offsetTrans)
		return transformationMatrix.Inv()
	}

	return c.Transform.GetViewMatrix()
}
func (c *FPSCamera) SetInvertedY(inverted bool) {
	c.invertedY = inverted
}

func (c *FPSCamera) GetRandomRayInCircleFrustum(accuracy float64) (mgl32.Vec3, mgl32.Vec3) {
	accuracy = Clamp(accuracy, 0.0, 1.0)
	accFactor := 1.0 - accuracy // 0.01..1.0

	randX := rand.Float64()*2.0 - 1.0
	randY := rand.Float64()*2.0 - 1.0

	//println(fmt.Sprintf("randNorm: %0.2f, %0.2f", randX, randY))

	lengthOfVector := math.Sqrt(randX*randX + randY*randY)
	if lengthOfVector > 1.0 {
		// normalize
		randX /= lengthOfVector
		randY /= lengthOfVector
	}
	//println(fmt.Sprintf("circled: %0.2f, %0.2f", randX, randY))

	// in range -1.0..1.0
	randX *= accFactor
	randY *= accFactor

	//println(fmt.Sprintf("acc. adjusted: %0.2f, %0.2f", randX, randY))

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

func (c *FPSCamera) SetLookTarget(position mgl32.Vec3) {
	c.lookTarget = position
	front := position.Sub(c.GetPosition()).Normalize()
	c.rotatex = mgl32.RadToDeg(float32(math.Atan2(float64(front.Z()), float64(front.X()))))
	c.rotatey = mgl32.RadToDeg(float32(math.Asin(float64(front.Y()))))
	c.updateTransform()
}

func (c *FPSCamera) GetLookTarget() mgl32.Vec3 {
	return c.lookTarget
}

func (c *FPSCamera) GetRotation() (float32, float32) {
	return c.rotatex, c.rotatey
}

func (c *FPSCamera) Reposition(pos mgl32.Vec3, rotX float32, rotY float32) {
	c.SetPosition(pos)
	c.rotatex = rotX
	c.rotatey = rotY
	c.updateTransform()
}
func (c *FPSCamera) AttachTo(t Transformer) {
	c.parent = t
}
func (c *FPSCamera) SetFOV(fov float32) {
	c.fov = fov
}

func (c *FPSCamera) GetFOV() float32 {
	return c.fov
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

	cameraPosition := c.GetPosition()
	transformationMatrix := mgl32.LookAtV(cameraPosition, cameraPosition.Add(c.cameraFront), c.cameraUp).Inv()

	camRot := ExtractRotation(transformationMatrix)

	c.Transform.SetRotation(camRot)
}

func (c *FPSCamera) Detach() {
	c.parent = nil
}

func (c *FPSCamera) DebugAim() string {
	pos := c.GetPosition()
	return fmt.Sprintf("Pos: (%0.2f, %0.2f, %0.2f) Aim: (%0.2f, %0.2f)", pos.X(), pos.Y(), pos.Z(), c.rotatex, c.rotatey)
}

func (c *FPSCamera) SetScreenSize(width int, height int) {
	c.windowWidth = width
	c.windowHeight = height
	c.updateTransform()
}
