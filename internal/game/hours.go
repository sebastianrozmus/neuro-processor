package game

// Keepers keep hours. Through the day they stand in their shops; when
// night falls they sit in the inn, where they trade nothing but talk the
// more; late at night the settlement sleeps and only rest is to be had.

const (
	// innFrom is the hour keepers move to the inn; sleepFrom when the
	// settlement goes to bed. Both are hours of a hero's day, in steps.
	innFrom   = nightFrom
	sleepFrom = nightFrom + 2
)

// hourName words the time of a hero's day.
func (r *Realm) hourName(hero *player) string {
	switch {
	case hero.steps >= sleepFrom:
		return "noc"
	case hero.steps >= innFrom:
		return "wieczór"
	case hero.steps >= stepsPerDay/2:
		return "popołudnie"
	}

	return "ranek"
}

// asleep reports whether the settlement has gone to bed for a hero.
func (r *Realm) asleep(hero *player) bool {
	return hero.steps >= sleepFrom && !r.adventure()
}

// evening reports whether keepers have moved to the inn for a hero.
func (r *Realm) evening(hero *player) bool {
	return hero.steps >= innFrom && hero.steps < sleepFrom && !r.adventure()
}

// gossip picks a fact a keeper does not know yet but a neighbour does,
// as told over a table at the inn; the keeper knows it from now on.
func (r *Realm) gossip(keeper int) *Fact {
	var fresh []*Fact

	for _, fact := range r.wisdom.facts {
		if fact.known[keeper] || len(fact.known) == 0 {
			continue
		}

		fresh = append(fresh, fact)
	}

	if len(fresh) == 0 {
		return nil
	}

	fact := fresh[r.rng.Intn(len(fresh))]
	fact.known[keeper] = true

	return fact
}
