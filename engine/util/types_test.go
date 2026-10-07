package util

import "testing"

func TestZoneFromPartName(t *testing.T) {
	cases := map[string]DamageZone{
		"helmet":         ZoneHead,
		"jaw":            ZoneHead,
		"left_eyebrow":   ZoneHead,
		"body":           ZoneTorso,
		"back_gear":      ZoneTorso,
		"waist":          ZoneTorso,
		"weapon":         ZoneWeapon,
		"left_pad":       ZoneLeftArm,
		"right_forearm":  ZoneRightArm,
		"left_leg_pad":   ZoneLeftLeg,
		"right_knee":     ZoneRightLeg,
		"something_else": ZoneTorso,
	}
	for part, expected := range cases {
		if zone := ZoneFromPartName(part); zone != expected {
			t.Errorf("%s: expected %s, got %s", part, expected, zone)
		}
	}
}
