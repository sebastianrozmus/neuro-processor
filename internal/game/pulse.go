package game

import (
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The pulse of the world: what changes by itself as days pass. Monsters
// roam and return to emptied lands, prices drift with what is bought,
// rumours travel between keepers, night falls and the roads hold
// chance meetings.

const (
	// nightFrom is the hour of a day (in steps) when night falls.
	nightFrom = 8

	wanderPercent  = 30
	respawnPercent = 8
	// buyMarkup is how much a sale raises a thing's price; driftMarkup how
	// much prices wander a day; prices settle back towards the list.
	buyMarkup   = 15
	driftMarkup = 8
	minMarkup   = 60
	maxMarkup   = 170

	meetingPercent = 10
	ambushPercent  = 12
)

// worldDay moves the world on to a day, once, however many heroes reach it.
func (r *Realm) worldDay(day int) {
	if day <= r.day || r.adventure() {
		return
	}

	r.day = day
	r.rollWeather()
	r.spreadKnowledge()
	r.drift()
	r.roam()
	r.travel()
	r.postQuests()
	r.rise()
}

// night reports whether it is dark for a hero.
func (r *Realm) night(hero *player) bool {
	return hero.steps >= nightFrom && !r.adventure()
}

// markupOf is a thing's price as a percentage of its list price.
func (r *Realm) markupOf(id string) int {
	if markup, ok := r.markup[id]; ok {
		return markup
	}

	return 100
}

// priced applies a thing's markup to a merchant's item.
func (r *Realm) priced(item Item) Item {
	markup := r.markupOf(item.ID)
	item.ListPrice = max(1, item.ListPrice*markup/100)
	item.MinPrice = max(1, item.MinPrice*markup/100)

	return item
}

// bought raises the price of a thing that just sold: it is scarcer now.
func (r *Realm) bought(id string) {
	r.markup[id] = clamp(r.markupOf(id)+buyMarkup, minMarkup, maxMarkup)
}

// drift lets prices wander a little each day and settle back.
func (r *Realm) drift() {
	for _, item := range r.world.Items {
		markup := r.markupOf(item.ID)
		markup += r.rng.Intn(2*driftMarkup+1) - driftMarkup

		// Towards the list price, by a step; in winter towards dearer.
		target := 100
		if r.winter() {
			target = winterMarkup
		}

		switch {
		case markup > target:
			markup -= driftMarkup / 2
		case markup < target:
			markup += driftMarkup / 2
		}

		r.markup[item.ID] = clamp(markup, minMarkup, maxMarkup)
	}
}

// roam moves packs of monsters about and brings new ones into emptied
// wilds, never into settlements, sites or onto heroes.
func (r *Realm) roam() {
	moved := map[int]bool{}

	for index := range r.monsters {
		if moved[index] || len(r.foes(world.Point{X: index % world.Width, Y: index / world.Width})) == 0 || r.rng.Intn(100) >= wanderPercent {
			continue
		}

		from := world.Point{X: index % world.Width, Y: index / world.Width}
		if r.fights[from] != nil {
			continue
		}

		steps := []world.Point{north, east, south, west}
		r.rng.Shuffle(len(steps), func(a, b int) { steps[a], steps[b] = steps[b], steps[a] })

		for _, step := range steps {
			to := world.Point{X: from.X + step.X, Y: from.Y + step.Y}
			if !r.world.In(to) || r.world.At(to).Town >= 0 || r.world.At(to).Site >= 0 || r.world.At(to).Place < 0 || len(r.monsters[r.world.Index(to)]) > 0 || r.someoneAt(to) {
				continue
			}

			r.monsters[r.world.Index(to)], r.monsters[index] = r.monsters[index], nil
			moved[r.world.Index(to)] = true

			break
		}
	}

	for index, tile := range r.world.Tiles {
		at := world.Point{X: index % world.Width, Y: index / world.Width}
		place := r.placeAt(at)

		if tile.Town >= 0 || tile.Site >= 0 || tile.Place < 0 || place.Danger == 0 || len(r.monsters[index]) > 0 || r.someoneAt(at) || r.world.Near(at) {
			continue
		}

		if r.rng.Intn(100) < respawnPercent*place.Danger {
			for _, kind := range r.world.Pack(r.rng, place.Danger, 1) {
				r.monsters[index] = append(r.monsters[index], r.newMonster(kind))
			}
		}
	}
}

// someoneAt reports whether a connected hero stands on a tile.
func (r *Realm) someoneAt(at world.Point) bool {
	for _, name := range r.order {
		if who := r.players[name]; who.connected && who.pos == at {
			return true
		}
	}

	return false
}

// meeting is what the road may hold for a hero who arrives somewhere
// wild and finds no monsters there: a traveller with news, a dropped
// purse, a snare, or at night an ambush.
func (r *Realm) meeting(hero *player) {
	tile := r.world.At(hero.pos)
	if r.adventure() || tile.Town >= 0 || tile.Site >= 0 || r.alive(hero.pos) > 0 {
		return
	}

	if (r.night(hero) || r.blind()) && r.rng.Intn(100) < ambushPercent {
		place := r.placeAt(hero.pos)
		for _, kind := range r.world.Pack(r.rng, max(place.Danger, 1), 1) {
			index := r.world.Index(hero.pos)
			r.monsters[index] = append(r.monsters[index], r.newMonster(kind))
		}

		r.tell(hero, "Z ciemności coś wyskakuje!")
		r.engage(hero)

		return
	}

	if r.rng.Intn(100) >= meetingPercent {
		return
	}

	switch r.rng.Intn(3) {
	case 0:
		unheard := r.unheardFacts(hero)
		if len(unheard) == 0 {
			break
		}

		fact := unheard[r.rng.Intn(len(unheard))]
		r.learn(hero, fact)
		r.tell(hero, fmt.Sprintf("Spotykasz wędrowca. Przy ognisku opowiada o tym, co nazywa się %s: %s", fact.Subject, fact.Text))
	case 1:
		found := 5 + r.rng.Intn(16)
		hero.gold += found
		r.tell(hero, fmt.Sprintf("W trawie leży zgubiona sakiewka: %d %s.", found, r.world.Currency))
	default:
		hurt := min(3+r.rng.Intn(5), hero.hp-1)
		if hurt > 0 {
			hero.hp -= hurt
			r.tell(hero, fmt.Sprintf("Wdeptujesz w sidła. Tracisz %d zdrowia.", hurt))
		}
	}
}

// unheardFacts lists what a hero has not come across yet.
func (r *Realm) unheardFacts(hero *player) []*Fact {
	var unheard []*Fact
	for _, fact := range r.wisdom.facts {
		if !fact.heard[hero.name] {
			unheard = append(unheard, fact)
		}
	}

	return unheard
}
