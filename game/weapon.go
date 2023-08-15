package game

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
	BulletsPerShot      int
	EffectiveRange      int
	MaxRange            int
	MagazineSize        int
	BaseDamagePerBullet int
}
type Weapon struct {
	Definition      *WeaponDefinition
	AmmoCount       int
	AccuracyPenalty float64
}

func (w *Weapon) IsReady() bool {
	return w.AmmoCount > 0
}

func (w *Weapon) ConsumeAmmo(amount int) {
	w.AmmoCount -= amount
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

func NewWeapon(definition *WeaponDefinition) *Weapon {
	return &Weapon{
		Definition: definition,
		AmmoCount:  definition.MagazineSize,
	}
}
