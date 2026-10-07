package game

// Reaction is a defensive stance a unit can end its turn in. It lasts until the unit's next turn.
// Reactions are hidden from the opponent until they trigger, so picking the right attack is a read
// on the opponent: bullets against a dodging unit are wasted, grenades against a braced one too.
type Reaction string

const (
	ReactionNone Reaction = ""
	// ReactionDodge makes every bullet that would hit miss with DodgeChance. Explosions ignore it.
	ReactionDodge Reaction = "Dodge"
	// ReactionBrace reduces every bullet hit by BraceBulletReduction and halves explosion damage.
	ReactionBrace Reaction = "Brace"
)

const (
	DodgeChance          = 0.5
	BraceBulletReduction = 1
	// ReactionAPCost is the minimum AP needed; the reaction always ends the unit's turn.
	ReactionAPCost = 1
)

// BracedBulletDamage applies a brace to the damage of a single bullet hit.
func BracedBulletDamage(damage int) int {
	return max(1, damage-BraceBulletReduction)
}

// BracedExplosionDamage applies a brace to explosion damage.
func BracedExplosionDamage(damage int) int {
	return (damage + 1) / 2
}

// VisualReaction tells the owner that its unit took a defensive stance (the opponent is not told).
type VisualReaction struct {
	UnitMessage
	Reaction Reaction
}

func (v VisualReaction) MessageType() string {
	return "Reaction"
}

func (u *UnitInstance) CanReact() bool {
	return u.CanAct() && u.GetIntegerAP() >= ReactionAPCost
}
