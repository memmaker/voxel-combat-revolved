package game

type WeaponType string

const (
	WeaponAutomatic WeaponType = "Automatic"
	WeaponShotgun   WeaponType = "Shotgun"
	WeaponSniper    WeaponType = "Sniper"
	WeaponPistol    WeaponType = "Pistol"
)

type WeaponDefinition struct {
	UniqueName       string
	Model            string
	WeaponType       WeaponType
	AccuracyModifier float64
	BulletsPerShot   int
	EffectiveRange   int
	MaxRange         int
	MagazineSize     int
	BaseDamage       int
}
type Weapon struct {
	Definition *WeaponDefinition
	AmmoCount  int
}

func NewWeapon(definition *WeaponDefinition) *Weapon {
	return &Weapon{
		Definition: definition,
		AmmoCount:  definition.MagazineSize,
	}
}
