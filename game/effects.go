package game

type EffectDefinition struct {
	Effect      TargetedEffect
	TurnsToLive int
	Radius      float64
}
type TargetedEffect string

const (
	TargetedEffectNone        TargetedEffect = ""
	TargetedEffectSmokeCloud  TargetedEffect = "SmokeCloud"
	TargetedEffectPoisonCloud TargetedEffect = "PoisonCloud"
	TargetedEffectFire        TargetedEffect = "Fire"
	TargetedEffectExplosion   TargetedEffect = "Explosion"
)

type BlockStatusEffectInstance struct {
	Effect BlockEffect
	Turns  int
}
type BlockStatusEffect int

const (
	BlockStatusEffectBlocksLOS BlockStatusEffect = 1 << iota
	BlockStatusEffectBlocksMovement
	BlockStatusEffectDamagesOnTouch
)

type BlockEffect int

const (
	BlockEffectSmoke BlockEffect = 1 << iota
	BlockEffectPoison
	BlockEffectFire
)

type ItemSize int

const (
	ItemSizeSmall ItemSize = iota
	ItemSizeMedium
	ItemSizeLarge
)
