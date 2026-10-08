package game

import "github.com/sebastianrozmus/neuro-processor/internal/world"

// unit is anyone on the heroes' side of a fight: a player's hero or one of
// their companions. Both carry gear and powers and both get a turn.
type unit interface {
	label() string
	leader() *player
	health() int
	setHealth(value int)
	maxHealth() int
	energy() int
	setEnergy(value int)
	maxEnergy() int
	// might is the damage of a plain blow, guard what is taken off a blow
	// received and pace what decides the order of turns.
	might() int
	guard() int
	pace() int
	powers() []world.ItemSpec
	// pack is what the unit carries unequipped: potions, for the most part.
	pack() *[]world.ItemSpec
	scores() Abilities
	// venom is how many rounds of poison are left in the unit.
	venom() int
	setVenom(rounds int)
	// arms is the weapon wielded, if any.
	arms() *world.ItemSpec
}

const (
	heroPace = 6
	// Armor slows its wearer down by its tier.
	levelsPerPace = 2
)

func (p *player) label() string            { return p.name }
func (p *player) leader() *player          { return p }
func (p *player) health() int              { return p.hp }
func (p *player) setHealth(value int)      { p.hp = value }
func (p *player) maxHealth() int           { return p.maxHP() }
func (p *player) energy() int              { return p.mana }
func (p *player) setEnergy(value int)      { p.mana = value }
func (p *player) maxEnergy() int           { return p.maxMana() }
func (p *player) might() int               { return p.attack() }
func (p *player) guard() int               { return p.defense() }
func (p *player) powers() []world.ItemSpec { return p.spells }
func (p *player) pack() *[]world.ItemSpec  { return &p.bag }
func (p *player) scores() Abilities        { return p.abilities }
func (p *player) venom() int               { return p.poison }
func (p *player) setVenom(rounds int)      { p.poison = rounds }
func (p *player) arms() *world.ItemSpec    { return p.weapon }

// pace of a hero grows with experience and dexterity and drops under
// heavy armor.
func (p *player) pace() int {
	pace := heroPace + (p.level-1)/levelsPerPace + Modifier(p.abilities[Dexterity])
	if p.armor != nil {
		pace -= p.armor.Tier
	}

	return max(pace, 1)
}

// charmBonus is what a worn charm adds, if it has the given effect.
func charmBonus(charm *world.ItemSpec, effect world.Effect) int {
	if charm != nil && charm.Effect == effect {
		return charm.Power()
	}

	return 0
}

// equipSlot returns where an item of the given kind is worn.
func equipSlot(weapon, armor, charm **world.ItemSpec, kind world.Kind) **world.ItemSpec {
	switch kind {
	case world.KindArmor:
		return armor
	case world.KindCharm:
		return charm
	}

	return weapon
}
