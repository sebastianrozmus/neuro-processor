package world

// minPricePercent is the share of the list price a merchant will not go under.
const minPricePercent = 65

type tiers = [MaxTier]int

// powers holds what an item does, by kind and effect.
var powers = map[Kind]map[Effect]tiers{
	KindWeapon: {EffectNone: {3, 6, 10}},
	KindArmor:  {EffectNone: {1, 3, 5}},
	KindPotion: {EffectHeal: {15, 30, 50}, EffectMana: {8, 14, 20}},
	KindSpell:  {EffectBolt: {10, 18, 28}, EffectBlast: {6, 10, 16}, EffectHeal: {15, 25, 40}},
	KindCharm:  {EffectPower: {2, 4, 6}, EffectWard: {1, 2, 4}, EffectVigor: {10, 20, 30}, EffectFocus: {6, 10, 14}},
}

// costs holds the power a spell drains when cast.
var costs = map[Effect]tiers{
	EffectBolt:  {4, 6, 9},
	EffectBlast: {6, 9, 12},
	EffectHeal:  {5, 7, 10},
}

var prices = map[Kind]tiers{
	KindWeapon: {40, 110, 240},
	KindArmor:  {35, 95, 210},
	KindPotion: {15, 35, 70},
	KindSpell:  {60, 140, 280},
	KindCharm:  {70, 160, 300},
}

var (
	monsterStats = [MaxTier]MonsterStats{
		{HP: 18, Attack: 5, GoldMin: 12, GoldMax: 20},
		{HP: 34, Attack: 8, GoldMin: 30, GoldMax: 45},
		{HP: 55, Attack: 12, GoldMin: 60, GoldMax: 90},
	}
	bossStats = MonsterStats{HP: 170, Attack: 16, GoldMin: 300, GoldMax: 300}
)

// MonsterStats are the combat numbers of a monster.
type MonsterStats struct {
	HP      int
	Attack  int
	GoldMin int
	GoldMax int
}

// StatsFor returns the combat numbers of a monster of the given tier.
func StatsFor(tier int, boss bool) MonsterStats {
	if boss {
		return bossStats
	}

	return monsterStats[min(max(tier, 1), MaxTier)-1]
}

// Power is the strength of what the item does: attack added by a weapon,
// damage absorbed by an armor, points restored by a potion, damage or
// healing of a spell, or the bonus of a charm.
func (i ItemSpec) Power() int {
	return powers[i.Kind][i.Effect][i.tier()]
}

// Cost is the power a spell drains when cast; other items cost nothing.
func (i ItemSpec) Cost() int {
	if i.Kind != KindSpell {
		return 0
	}

	return costs[i.Effect][i.tier()]
}

// Price is the merchant's list price of the item.
func (i ItemSpec) Price() int {
	return prices[i.Kind][i.tier()]
}

// MinPrice is the lowest price a merchant accepts for the item.
func (i ItemSpec) MinPrice() int {
	return i.Price() * minPricePercent / 100
}

func (i ItemSpec) tier() int {
	return min(max(i.Tier, 1), MaxTier) - 1
}

// CompanionStats are the numbers of a hero for hire.
type CompanionStats struct {
	HP int
	// Attack is the damage of a fighter's blow, a healer's weak strike or
	// a mage's blast on every foe.
	Attack int
	// Heal is what a healer restores to the most wounded ally.
	Heal  int
	Price int
}

var companionStats = map[Role][MaxTier]CompanionStats{
	RoleFighter: {{HP: 30, Attack: 5, Price: 60}, {HP: 45, Attack: 8, Price: 130}, {HP: 60, Attack: 11, Price: 220}},
	RoleHealer:  {{HP: 22, Attack: 2, Heal: 10, Price: 70}, {HP: 32, Attack: 3, Heal: 16, Price: 150}, {HP: 42, Attack: 4, Heal: 22, Price: 240}},
	RoleMage:    {{HP: 20, Attack: 4, Price: 70}, {HP: 30, Attack: 6, Price: 150}, {HP: 40, Attack: 8, Price: 240}},
}

// A legend is twice as tough and hits harder than the best of the rest;
// instead of a price they take a wage every day.
const (
	legendHPFactor    = 2
	legendAttackPlus  = 4
	legendWageBase    = 20
	legendWagePerTier = 15
)

// Stats returns the numbers of a companion.
func (c CompanionSpec) Stats() CompanionStats {
	stats, ok := companionStats[c.Role]
	if !ok {
		stats = companionStats[RoleFighter]
	}

	chosen := stats[min(max(c.Tier, 1), MaxTier)-1]
	if c.Legend {
		chosen.HP *= legendHPFactor
		chosen.Attack += legendAttackPlus
		chosen.Price = 0
	}

	return chosen
}

// Wage is what a legend costs per day; other companions take none.
func (c CompanionSpec) Wage() int {
	if !c.Legend {
		return 0
	}

	return legendWageBase + legendWagePerTier*min(max(c.Tier, 1), MaxTier)
}
