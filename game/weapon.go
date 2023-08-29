package game

import (
	"math"
)

type WeaponType string

const (
	WeaponAutomatic WeaponType = "Automatic"
	WeaponShotgun   WeaponType = "Shotgun"
	WeaponSniper    WeaponType = "Sniper"
	WeaponPistol    WeaponType = "Pistol"
)

type WeaponDefinition struct {
	UniqueName          string
	Model               string
	WeaponType          WeaponType
	AccuracyModifier    float64
	BulletsPerShot      uint
	EffectiveRange      uint
	MaxRange            uint
	MagazineSize        uint
	BaseDamagePerBullet int
	MinFOVForZoom       uint
	BaseAPForShot       uint
	BaseAPForReload     uint
}
type Weapon struct {
	Definition      *WeaponDefinition
	AmmoCount       uint
	AccuracyPenalty float64
}

func (w *Weapon) IsReady() bool {
	return w.AmmoCount > 0
}

func (w *Weapon) ConsumeAmmo(amount uint) {
	w.AmmoCount -= amount
	//println(fmt.Sprintf("[Weapon] %s ammo count is now %d", w.Definition.UniqueName, w.AmmoCount))
}

func (w *Weapon) Reload() {
	w.AmmoCount = w.Definition.MagazineSize
}

func (w *Weapon) SetAccuracyPenalty(penalty float64) {
	w.AccuracyPenalty = penalty
}

func (w *Weapon) GetAccuracyModifier() float64 {
	return w.Definition.AccuracyModifier - w.AccuracyPenalty
}

func (w *Weapon) GetMinFOVForZoom() uint {
	return w.Definition.MinFOVForZoom
}

func (w *Weapon) IsMagazineFull() bool {
	return w.AmmoCount == w.Definition.MagazineSize
}

func NewWeapon(definition *WeaponDefinition) *Weapon {
	return &Weapon{
		Definition: definition,
		AmmoCount:  definition.MagazineSize,
	}
}

func (w *Weapon) AdjustDamageForDistance(distance float32, projectileBaseDamage int) int {
	rangeAdjustedDamage := projectileBaseDamage
	effectiveWeaponRange := float32(w.Definition.EffectiveRange)
	inEffectiveRange := distance <= effectiveWeaponRange
	if !inEffectiveRange { // damage falloff
		falloff := float64(1.0 - (distance-effectiveWeaponRange)/(float32(w.Definition.MaxRange)-effectiveWeaponRange))
		rangeAdjustedDamage = int(math.Ceil(float64(projectileBaseDamage) * falloff))
	}
	return rangeAdjustedDamage
}

func (w *Weapon) GetEstimatedDamage(distance float32) int {
	damagePerBullet := w.AdjustDamageForDistance(distance, w.Definition.BaseDamagePerBullet)
	maxDamage := damagePerBullet * int(w.Definition.BulletsPerShot)
	return maxDamage
}
