package util

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Collider interface {
	FindFurthestPoint(direction mgl32.Vec3) mgl32.Vec3
	ToString() string
	Draw()
	GetName() string
	SetName(name string)
	IntersectsRay(start mgl32.Vec3, end mgl32.Vec3) (bool, mgl32.Vec3)
}
type DamageZone string

const (
	ZoneNone     DamageZone = "None"
	ZoneHead     DamageZone = "Head"
	ZoneTorso    DamageZone = "Torso"
	ZoneRightArm DamageZone = "RightArm"
	ZoneLeftArm  DamageZone = "LeftArm"
	ZoneRightLeg DamageZone = "RightLeg"
	ZoneLeftLeg  DamageZone = "LeftLeg"
	ZoneWeapon   DamageZone = "Weapon"
)
