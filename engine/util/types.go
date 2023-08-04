package util

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Transform struct {
	Position mgl32.Vec3
	Rotation mgl32.Quat
	Scale    mgl32.Vec3
}

type Collider interface {
	FindFurthestPoint(direction mgl32.Vec3) mgl32.Vec3
	ToString() string
	Draw()
	GetName() PartName
	SetName(name string)
	IntersectsRay(start mgl32.Vec3, end mgl32.Vec3) (bool, mgl32.Vec3)
}
type PartName string

const (
	BodyPartNone  PartName = "None"
	BodyPartHead  PartName = "Head"
	BodyPartTorso PartName = "Torso"
	BodyPartLegs  PartName = "Legs"
)
