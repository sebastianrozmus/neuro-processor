package game

import (
	"regexp"
	"strings"
)

// stingCue finds in a keeper's card the name they will not be called:
// "obraża się, gdy ktoś nazwie go szarlatanem", "nie znosi, gdy ktoś
// nazywa ją piratką". Only a name for the keeper counts: "nazywa statek
// puszką" stings about the ship, not as a name.
var stingCue = regexp.MustCompile(`(?i)\b(?:nazwie|nazywa|nazwał|nazwała|nazywają|nazwą|nazywać)\s+(?:go|ją|jego|mnie)\s+[„"]?(\p{L}+)`)

// keeperCard is what the world says of a keeper: a shopkeeper's persona,
// a companion's or a townsperson's description.
func (r *Realm) keeperCard(index int) string {
	switch {
	case index < 0:
		return ""
	case r.investigation():
		if index < len(r.world.People) {
			return r.world.People[index].Description
		}
	case r.caravan != nil && index == r.caravanShop():
		return ""
	case index < len(r.world.Shops):
		return r.world.Shops[index].Persona
	}

	if kind, ok := r.mateOf(index); ok {
		return r.world.Companions[kind].Description
	}

	return ""
}

// stung reports a folded message that calls the keeper what their card
// says they will not be called, in any form: "ty szarlatanie".
func (r *Realm) stung(keeper int, folded string) bool {
	for _, match := range stingCue.FindAllStringSubmatch(r.keeperCard(keeper), -1) {
		sting := Fold(match[1])
		for _, word := range strings.Fields(folded) {
			if len(word) > minQueryWord && formOf(word, sting) {
				return true
			}
		}
	}

	return false
}
