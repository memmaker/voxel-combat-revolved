package game

import "fmt"

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
	println(fmt.Sprintf("[Weapon] %s ammo count is now %d", w.Definition.UniqueName, w.AmmoCount))
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
