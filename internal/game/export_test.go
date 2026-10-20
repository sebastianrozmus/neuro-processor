package game

import "fmt"

// Hooks for the external tests: they set up a hero the way a long game
// would have, without playing it.

// SetGold gives a hero a purse.
func (r *Realm) SetGold(name string, gold int) {
	r.players[name].gold = gold
}

// SetHour moves a hero's day on to an hour: "wieczór" is when keepers sit
// in the inn, anything else the morning.
func (r *Realm) SetHour(name, hour string) {
	r.players[name].steps = 0
	if hour == "wieczór" {
		r.players[name].steps = innFrom
	}
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
