package client

import "github.com/go-gl/mathgl/mgl32"

type AABBCollisionHandler func(prevPos mgl32.Vec3, curPos mgl32.Vec3, extents mgl32.Vec3) (mgl32.Vec3, bool)
