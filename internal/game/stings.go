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

// Cards say in many ways that a keeper cannot stand swearing: "nie znosi
// przekleństw", "brzydzi się wulgaryzmami" (prudeNoun); "nie lubi, gdy
// ktoś klnie", "strofuje każdego, kto przy niej przeklina" (prudeClause);
// "przekleństwa ją drażnią" (prudeAfter). One who swears themselves, "klnie
// jak dokerka, nie znosi garniturów", is no prude: the swearing must be
// what the dislike is about.
var (
	prudeVerb   = `(?:^|[^\p{L}])(?:nie\s+(?:znosi|cierpi|lubi|toleruje|trawi|zniesie|pozwala)|nienawidzi|brzydzi\s+się|gorszy\s+się|oburza\s+się|złości\s+się|strofuje|gani|beszta|karci|upomina|ruga|zwraca\s+uwagę)`
	prudeNoun   = regexp.MustCompile(`(?i)` + prudeVerb + `(?:[^\p{L}.;!?]+\p{L}+){0,2}?[^\p{L}.;!?]+(?:przekleństw|przeklinani|wulgar|bluzg|bluźnierstw|sprośn|brzydki\p{L}*\s+(?:słów|słowa|wyraz))`)
	prudeClause = regexp.MustCompile(`(?i)` + prudeVerb + `(?:[^\p{L}.;!?]+\p{L}+){0,3}?[^\p{L}.;!?]+(?:gdy|kiedy|jak\s+ktoś|kto|że|jeśli|jeżeli)(?:[^\p{L}.;!?]+\p{L}+){0,3}?[^\p{L}.;!?]+(?:kln|klą|klęł|przeklin|przeklnie|bluzga)`)
	prudeAfter  = regexp.MustCompile(`(?i)(?:przekleństw|przeklinani|wulgar|bluzg)\p{L}*\s+(?:\p{L}+\s+)?(?:drażni|irytuj|złoszcz|denerwuj|gorsz|oburzaj|brzydz)`)
)

// prudish reports a card that says the keeper cannot stand swearing.
func prudish(card string) bool {
	return prudeNoun.MatchString(card) || prudeClause.MatchString(card) || prudeAfter.MatchString(card)
}

// cursedAt reports a message that swears at a keeper whose card says they
// cannot stand it: "kurwa, nie wierzę" to a nun who will not hear it.
func (r *Realm) cursedAt(keeper int, text string) bool {
	return swearing(text) && prudish(r.keeperCard(keeper))
}

// curseHint tells a keeper who cannot stand swearing that the player swore.
const curseHint = "Gracz przeklina, a ty tego nie znosisz: zwróć mu uwagę po swojemu."
