package game

import (
	"math/rand"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The six abilities of a character sheet, in their usual order.
const (
	Strength = iota
	Dexterity
	Constitution
	Intelligence
	Wisdom
	Charisma
	abilityCount
)

// Abilities are the six scores of a character.
type Abilities [abilityCount]int

// AbilityNames are the Polish names of the abilities, in order.
var AbilityNames = [abilityCount]string{"Siła", "Zręczność", "Kondycja", "Inteligencja", "Mądrość", "Charyzma"}

// AbilityShort are the three-letter marks of the abilities, in order.
var AbilityShort = [abilityCount]string{"SIŁ", "ZRĘ", "KON", "INT", "MĄD", "CHA"}

const (
	abilityDice   = 4
	abilityDieMax = 6
	// roleJitter is how much a companion's scores stray from their role's.
	roleJitter = 2
)

// roleScores are the typical scores of each role, before tier and chance.
var roleScores = map[world.Role]Abilities{
	world.RoleFighter: {15, 11, 14, 8, 10, 9},
	world.RoleHealer:  {9, 10, 12, 11, 15, 12},
	world.RoleMage:    {8, 13, 10, 15, 12, 10},
}

// primary is the ability a role leans on, which grows with tier.
var primary = map[world.Role]int{world.RoleFighter: Strength, world.RoleHealer: Wisdom, world.RoleMage: Intelligence}

// Modifier is what a score adds to rolls, as on a character sheet: zero
// at 10 or 11, one for every two points above or below.
func Modifier(score int) int {
	if score >= 10 {
		return (score - 10) / 2
	}

	return -((11 - score) / 2)
}

// archetypes are the standard array of scores arranged for each built-in
// portrait, so that the picture a player picks is also a calling.
var archetypes = map[string]Abilities{
	"wojownik":   {15, 13, 14, 8, 10, 12},
	"czarodziej": {8, 13, 12, 15, 14, 10},
	"lotrzyk":    {10, 15, 12, 13, 8, 14},
	"lowczyni":   {12, 15, 13, 10, 14, 8},
	"szabrownik": {14, 13, 15, 8, 12, 10},
	"pilot":      {8, 14, 12, 15, 10, 13},
}

// heroAbilities gives a hero the scores of their portrait's calling, or a
// classic roll for a portrait of their own.
func heroAbilities(rng *rand.Rand, avatar string) Abilities {
	if scores, ok := archetypes[avatar]; ok {
		return scores
	}

	return rollAbilities(rng)
}

// rollAbilities rolls a hero's scores the classic way: four dice, the
// lowest dropped, for each ability.
func rollAbilities(rng *rand.Rand) Abilities {
	var scores Abilities

	for index := range scores {
		lowest, total := abilityDieMax+1, 0

		for range abilityDice {
			die := rng.Intn(abilityDieMax) + 1
			total += die
			lowest = min(lowest, die)
		}

		scores[index] = total - lowest
	}

	return scores
}

// roleAbilities gives a companion scores fit for their role and tier, with
// a little chance in them. Legends get more of everything.
func roleAbilities(rng *rand.Rand, spec world.CompanionSpec) Abilities {
	scores := roleScores[spec.Role]
	scores[primary[spec.Role]] += spec.Tier - 1

	for index := range scores {
		scores[index] += rng.Intn(2*roleJitter+1) - roleJitter

		if spec.Legend {
			scores[index] += 2
		}

		scores[index] = min(max(scores[index], 3), 20)
	}

	return scores
}

// valid reports whether scores look like rolled ones, not an empty save.
func (a Abilities) valid() bool {
	for _, score := range a {
		if score < 3 || score > 20 {
			return false
		}
	}

	return true
}
