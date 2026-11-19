package game

import (
	"fmt"
	"slices"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Hooks for the external tests: they set up a hero the way a long game
// would have, without playing it.

// SetGold gives a hero a purse.
func (r *Realm) SetGold(name string, gold int) {
	r.players[name].gold = gold
}

// SetHour moves a hero's day on to an hour: "popołudnie", "wieczór" (when
// keepers sit in the inn), "noc"; anything else the morning.
func (r *Realm) SetHour(name, hour string) {
	r.players[name].steps = map[string]int{"popołudnie": stepsPerDay / 2, "wieczór": innFrom, "noc": sleepFrom}[hour]
}

// SetHealth wounds or heals a hero.
func (r *Realm) SetHealth(name string, hp int) {
	r.players[name].hp = hp
}

// SetStanding sets what a settlement thinks of a hero.
func (r *Realm) SetStanding(name, place string, score int) error {
	for town := range r.world.Towns {
		if r.townName(town) == place {
			if r.players[name].standing == nil {
				r.players[name].standing = map[int]int{}
			}

			r.players[name].standing[town] = score

			return nil
		}
	}

	return fmt.Errorf("no settlement %q", place)
}

// Teleport puts a hero on the first tile of a place.
func (r *Realm) Teleport(name, place string) error {
	for index, spec := range r.places {
		if spec.Name != place {
			continue
		}

		if point, ok := r.placeTile(index); ok {
			r.players[name].pos = point

			return nil
		}
	}

	return fmt.Errorf("no place %q", place)
}

// Recruit takes a companion into a hero's party for free.
func (r *Realm) Recruit(name, companion string) error {
	for kind, spec := range r.world.Companions {
		if spec.Name == companion {
			r.recruit(r.players[name], kind, spec.Wage())

			return nil
		}
	}

	return fmt.Errorf("no companion %q", companion)
}

// SeedBio keeps facts a speaker said of themselves before, heard by a
// player, as LearnSelf would after the Mind read them. It reports how many
// facts were kept.
func (r *Realm) SeedBio(name, speaker string, facts []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !slices.Contains(r.bioSpeakers(), speaker) {
		return 0, fmt.Errorf("%s nie ma biografii (ani towarzysz, ani kupiec o jedynym imieniu)", speaker)
	}

	kept := 0

	for _, fact := range facts {
		if text := bioSentence(fact); text != "" && r.remember(speaker, text, r.day, name) {
			kept++
		}
	}

	return kept, nil
}

// SeedRescue opens a keeper's rescue as SpawnRescue would after the Mind
// heard the call for help, with no hook to check: the missing person held
// on a wild tile under guards. taker is who went looking ("" for the hero
// named); state is "open", "freed" (the captive is back, not yet paid for)
// or "paid" (settled: the keeper's bio has the rescue in it).
func (r *Realm) SeedRescue(name, keeper, person, relation, monster, place, taker, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hero := r.players[name]
	shop := slices.IndexFunc(r.world.Shops, func(spec world.ShopSpec) bool { return spec.Keeper == keeper })

	if hero == nil || shop < 0 {
		return fmt.Errorf("brak bohatera %q albo kupca %q", name, keeper)
	}

	if !r.rescuable(shop) {
		return fmt.Errorf("%s nie może teraz mieć zlecenia ratunkowego", keeper)
	}

	if taker == "" {
		taker = name
	}

	town := r.townOf(hero)
	if town < 0 {
		town = hero.home
	}

	kind := named(r.monsterNames(), monster)
	if monster != "" && kind < 0 {
		return fmt.Errorf("nie ma potwora %q", monster)
	}

	at := -1
	if found := named(r.wildNames(), place); found >= 0 {
		at = slices.IndexFunc(r.places, func(spec world.PlaceSpec) bool { return spec.Name == r.wildNames()[found] })
	} else if place != "" {
		return fmt.Errorf("nie ma groźnego miejsca %q", place)
	}

	pos, ok := r.rescueTile(r.world.Towns[town], kind, at)
	if !ok {
		return fmt.Errorf("brak pola na niewolę")
	}

	job := &quest{
		id: len(r.quests) + 1, keeper: shop, kind: questRescue, subject: person, relation: relation,
		monster: r.guard(pos, kind), pos: pos, town: town, taker: taker, day: r.day,
	}

	for _, foe := range r.foes(pos) {
		job.reward += foe.tier * rescueGold
	}

	r.withItem(job)
	r.quests = append(r.quests, job)
	r.revealed[r.world.Index(pos)] = true

	she := rescueFeminine(person, relation)
	news := fmt.Sprintf("%s martwi się o %s %s: %s w miejscu %s, %s %s.",
		keeper, accusative(relation, she), accusative(person, she), gendered(she, "zaginął", "zaginęła"), r.placeAt(pos).Name,
		gendered(she, "strzeże go", "strzeże jej"), r.world.Monsters[job.monster].Name)
	if fact := r.add(person, news, keeper, r.everyone()...); fact != nil {
		r.learn(hero, fact)
	}

	switch state {
	case "", "open":
	case "freed":
		job.finished = true
	case "paid":
		if taker != name {
			return fmt.Errorf("zapłacić można tylko bohaterowi, który uwolnił")
		}

		job.finished = true
		r.payQuest(hero, job)
	default:
		return fmt.Errorf("nieznany stan zlecenia %q", state)
	}

	return nil
}

// SetDay moves the calendar, so that a second talk can be days after the
// first.
func (r *Realm) SetDay(day int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if day > 0 {
		r.day = day
	}
}
