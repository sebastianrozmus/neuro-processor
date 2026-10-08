package game

import (
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Standing is what a settlement thinks of a hero: every deed near it,
// every coin spent in it and every insult thrown in its shops moves the
// score, and the score moves prices, who will trade and who will follow.

const (
	standingMin = -10
	standingMax = 10
	// A purchase or a sale earns a point; an insult costs two; a monster
	// slain within guardRadius of a settlement earns a point there.
	tradeStanding  = 1
	insultStanding = -2
	guardStanding  = 1
	guardRadius    = 2
	// shunnedAt is the standing at which keepers stop trading and
	// companions walk away; legends follow only the respected.
	shunnedAt     = -5
	legendNeeds   = 1
	questStanding = 2
	// standingDiscount is the percentage a point knocks off prices.
	standingDiscount = 3
)

// StandingView is a hero's standing with one settlement.
type StandingView struct {
	Town  string
	Score int
	Label string
}

// standingLabel words a score.
func standingLabel(score int) string {
	switch {
	case score <= shunnedAt:
		return "wyklęty"
	case score < 0:
		return "podejrzany"
	case score == 0:
		return "obcy"
	case score <= 3:
		return "znany"
	case score <= 7:
		return "szanowany"
	}

	return "bohater osady"
}

// standingOf is a hero's score with a settlement.
func (r *Realm) standingOf(hero *player, town int) int {
	if town < 0 {
		return 0
	}

	return hero.standing[town]
}

// esteem moves a hero's standing with a settlement and says so when the
// settlement's opinion of them changes its word.
func (r *Realm) esteem(hero *player, town, change int, why string) {
	if town < 0 || town >= len(r.world.Towns) || change == 0 {
		return
	}

	if hero.standing == nil {
		hero.standing = map[int]int{}
	}

	before := hero.standing[town]
	after := clamp(before+change, standingMin, standingMax)
	hero.standing[town] = after

	if standingLabel(before) != standingLabel(after) {
		r.tell(hero, fmt.Sprintf("%s W osadzie %s jesteś teraz: %s (%+d).", why, r.townName(town), standingLabel(after), after))
	}
}

// townName names a settlement.
func (r *Realm) townName(town int) string {
	return r.placeAt(r.world.Towns[town]).Name
}

// townNear is the settlement within guardRadius of a point, or -1.
func (r *Realm) townNear(at world.Point) int {
	for index, town := range r.world.Towns {
		if distance(town, at) <= guardRadius {
			return index
		}
	}

	return -1
}

// closestTown is the settlement nearest a point.
func (r *Realm) closestTown(at world.Point) int {
	best, bestSteps := -1, 0

	for index, town := range r.world.Towns {
		if steps := distance(town, at); best < 0 || steps < bestSteps {
			best, bestSteps = index, steps
		}
	}

	return best
}

// townOf is the settlement of a shop's keeper: keepers live where the
// customer meets them, so it is the settlement the hero stands in.
func (r *Realm) townOf(hero *player) int {
	return r.world.At(hero.pos).Town
}

// shunned reports whether a settlement will have nothing to do with a hero.
func (r *Realm) shunned(hero *player, town int) bool {
	return town >= 0 && r.standingOf(hero, town) <= shunnedAt
}

// favoured applies a hero's standing to a merchant's prices: the liked pay
// less, the disliked more.
func (r *Realm) favoured(merchant Merchant, standing int) Merchant {
	if standing == 0 {
		return merchant
	}

	percent := 100 - standing*standingDiscount

	for index := range merchant.Items {
		merchant.Items[index].ListPrice = max(1, merchant.Items[index].ListPrice*percent/100)
		merchant.Items[index].MinPrice = max(1, merchant.Items[index].MinPrice*percent/100)
	}

	return merchant
}

// standingView lists a hero's standing with every settlement they have a
// score with.
func (r *Realm) standingView(hero *player) []StandingView {
	var view []StandingView

	for town := range r.world.Towns {
		if score, ok := hero.standing[town]; ok {
			view = append(view, StandingView{Town: r.townName(town), Score: score, Label: standingLabel(score)})
		}
	}

	return view
}

// standingLine words a hero's standing with the settlement they are in.
func (r *Realm) standingLine(hero *player, town int) string {
	score := r.standingOf(hero, town)

	return fmt.Sprintf("Twoja sława w osadzie %s: %s (%+d).", r.townName(town), standingLabel(score), score)
}

// guarded credits everyone in a fight for a kill near a settlement.
func (r *Realm) guarded(current *fight) {
	town := r.townNear(current.pos)
	if town < 0 {
		return
	}

	for _, name := range current.members {
		r.esteem(r.players[name], town, guardStanding, "Osada widzi, kto broni jej okolic.")
	}
}
