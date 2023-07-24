package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
)

type ISOCamera struct {
	cameraPos          mgl32.Vec3
	cameraRight        mgl32.Vec3
	cameraUp           mgl32.Vec3
	windowWidth        int
	windowHeight       int
	relativeLookTarget mgl32.Vec3
	moveMap            map[[2]int]mgl32.Vec3
	nearPlaneDist      float32
}

func NewISOCamera(windowWidth, windowHeight int) *ISOCamera {
	camPos := mgl32.Vec3{-5, 7, -5}
	lookTarget := mgl32.Vec3{0, 0, 0}
	relativeLookTarget := lookTarget.Sub(camPos)
	return &ISOCamera{
		cameraPos:          camPos,
		cameraUp:           mgl32.Vec3{0, 1, 0},
		relativeLookTarget: relativeLookTarget,
		windowWidth:        windowWidth,
		windowHeight:       windowHeight,
		nearPlaneDist: 0.15,
		moveMap: map[[2]int]mgl32.Vec3{
			[2]int{0, -1}:  mgl32.Vec3{1, 0, 1},   // up
			[2]int{0, 1}:   mgl32.Vec3{-1, 0, -1}, // down
			[2]int{-1, 0}:  mgl32.Vec3{1, 0, -1},  // left
			[2]int{1, 0}:   mgl32.Vec3{-1, 0, 1},  // right
			[2]int{1, -1}:  mgl32.Vec3{0, 0, 1},   // up-right
			[2]int{-1, -1}: mgl32.Vec3{1, 0, 0},   // up-left
			[2]int{1, 1}:   mgl32.Vec3{-1, 0, 0},  // down-right
			[2]int{-1, 1}:  mgl32.Vec3{0, 0, -1},  // down-left
		},
	}
}

// GetViewMatrix returns the view matrix for the camera.
// A view matrix will transform a point from world space to camera space.
func (c *ISOCamera) GetViewMatrix() mgl32.Mat4 {
	camera := mgl32.LookAtV(c.cameraPos, c.cameraPos.Add(c.relativeLookTarget), c.cameraUp)
	return camera
}

// GetProjectionMatrix returns the projection matrix for the camera.
// A projection matrix will transform a point from camera space to screen space. (3D -> 2D)
func (c *ISOCamera) GetProjectionMatrix() mgl32.Mat4 {
	fov := float32(45.0)
	return mgl32.Perspective(mgl32.DegToRad(fov), float32(c.windowWidth)/float32(c.windowHeight), c.nearPlaneDist, 512.0)
}

func (c *ISOCamera) RotateRight() { // rotate around y axis by 45 degrees
	lt := c.relativeLookTarget
	absoluteLookTarget := c.cameraPos.Add(lt)
	c.relativeLookTarget = mgl32.Vec3{lt.Z(), lt.Y(), -lt.X()}
	c.cameraPos = absoluteLookTarget.Sub(c.relativeLookTarget)
	println(fmt.Sprintf("Relative look target: %v", c.relativeLookTarget))
}

func (c *ISOCamera) RotateLeft() { // rotate around y axis by -45 degrees
	lt := c.relativeLookTarget
	absoluteLookTarget := c.cameraPos.Add(lt)
	c.relativeLookTarget = mgl32.Vec3{-lt.Z(), lt.Y(), lt.X()}
	c.cameraPos = absoluteLookTarget.Sub(c.relativeLookTarget)
	println(fmt.Sprintf("Relative look target: %v", c.relativeLookTarget))
}

func (c *ISOCamera) ChangePosition(dir [2]int, delta float32) {
	signX := 0
	if c.relativeLookTarget.X() > 0 {
		signX = 1
	} else if c.relativeLookTarget.X() < 0 {
		signX = -1
	}
	signZ := 0
	if c.relativeLookTarget.Z() > 0 {
		signZ = 1
	} else if c.relativeLookTarget.Z() < 0 {
		signZ = -1
	}
	if signX == signZ { // side
		dir[0] *= signX
		dir[1] *= signZ
	} else { // front
		dir[0], dir[1] = dir[1]*signZ*-1, dir[0]*signX*-1
	}
	if moveVec, ok := c.moveMap[dir]; ok {
		speed := float32(20.0)
		moveVec = moveVec.Normalize().Mul(delta * speed)
		c.cameraPos = c.cameraPos.Add(moveVec)
	}
}
func (c *ISOCamera) GetPosition() mgl32.Vec3 {
	return c.cameraPos
}

func (c *ISOCamera) GetFront() mgl32.Vec3 {
	view := c.GetViewMatrix()
	_, _, z, _ := view.Rows()
	return z.Vec3()
}

func (c *ISOCamera) SetPosition(pos mgl32.Vec3) {
	c.cameraPos = pos
}

func (c *ISOCamera) GetFrustumPlanes(projection mgl32.Mat4) []mgl32.Vec4 {
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

func (c *ISOCamera) GetNearPlaneDist() float32 {
	return c.nearPlaneDist
}
