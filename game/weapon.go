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
	Definition *WeaponDefinition
	ammoCount  int
}

func (w *Weapon) IsReady() bool {
	return w.ammoCount > 0
}

func (w *Weapon) ConsumeAmmo(amount int) {
	w.ammoCount -= amount
}

func (w *Weapon) Reload() {
	w.ammoCount = w.Definition.MagazineSize
}

func NewWeapon(definition *WeaponDefinition) *Weapon {
	return &Weapon{
		Definition: definition,
		ammoCount:  definition.MagazineSize,
	}
}
