package util

import (
	"strings"

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

// ZoneFromPartName maps the name of a model part (e.g. "helmet", "left_forearm", "right_leg_pad")
// to the damage zone it belongs to. Unknown parts count as torso hits.
func ZoneFromPartName(partName string) DamageZone {
	name := strings.ToLower(partName)
	containsAny := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(name, word) {
				return true
			}
		}
		return false
	}
	isLeft := strings.Contains(name, "left")
	switch {
	case containsAny("weapon", "rifle", "gun", "pistol", "sniper", "mossberg", "launcher"):
		return ZoneWeapon
	case containsAny("head", "helmet", "jaw", "eye", "face", "neck", "nose", "mouth", "hat", "visor"):
		return ZoneHead
	case containsAny("leg", "knee", "foot", "feet", "shin", "boot", "thigh"):
		if isLeft {
			return ZoneLeftLeg
		}
		return ZoneRightLeg
	case containsAny("arm", "hand", "pad", "shoulder", "elbow", "wrist", "finger"):
		if isLeft {
			return ZoneLeftArm
		}
		return ZoneRightArm
	}
	return ZoneTorso
}
