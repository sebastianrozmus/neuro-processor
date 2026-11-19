package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Events are what makes a world move besides the player: scripted turns
// of the story that fire when their conditions are met, change places,
// bring monsters, give things, add facts and announce themselves. They
// are the adventure's steps brought to every kind of world.

// trigger is the kind of moment an event may fire at.
type trigger string

const (
	onEnter  trigger = "enter"
	onKill   trigger = "kill"
	onDay    trigger = "day"
	onTalk   trigger = "talk"
	onArrive trigger = "arrive"
)

// fire runs every event due at a moment for a player. The subject is the
// place, site or settlement entered, the monster killed or the keeper
// talked to, as the moment has it.
func (r *Realm) fire(hero *player, at trigger, subject string) {
	for index, event := range r.world.Events {
		if r.story.done[eventKey(index)] && event.Once {
			continue
		}

		if !r.due(hero, event, at, subject) {
			continue
		}

		if event.Chance > 0 && r.rng.Intn(100) >= event.Chance {
			continue
		}

		r.story.done[eventKey(index)] = true
		r.happen(hero, event)
	}
}

// eventKey keeps events apart from an adventure's steps in the done set.
func eventKey(index int) int {
	return 1000 + index
}

// due reports whether an event's conditions hold at this moment.
func (r *Realm) due(hero *player, event world.EventSpec, at trigger, subject string) bool {
	when := event.When

	switch {
	case when.Enter != "" && (at != onEnter || !strings.EqualFold(when.Enter, subject)):
		return false
	case when.Killed != "" && (at != onKill || !strings.EqualFold(when.Killed, subject) || r.slain[when.Killed] < max(when.Count, 1)):
		return false
	case when.Day > 0 && (at != onDay || hero.day < when.Day):
		return false
	case when.Talked != "" && (at != onTalk || !strings.EqualFold(when.Talked, subject)):
		return false
	case when.Enter == "" && when.Killed == "" && when.Day == 0 && when.Talked == "" && at != onArrive:
		// An event with no moment of its own fires on any arrival.
		return false
	case when.Has != "" && find(hero.bag, when.Has) < 0 && !wields(hero, when.Has):
		return false
	case when.Flag != "" && !r.story.flags[when.Flag]:
		return false
	case when.Unless != "" && r.story.flags[when.Unless]:
		return false
	}

	return true
}

// wields reports whether a hero wears or wields the named thing.
func wields(hero *player, name string) bool {
	for _, worn := range []*world.ItemSpec{hero.weapon, hero.armor, hero.charm} {
		if worn != nil && Matches(worn.Name, name) {
			return true
		}
	}

	return false
}

// happen applies an event.
func (r *Realm) happen(hero *player, event world.EventSpec) {
	if event.Flag != "" {
		r.story.flags[event.Flag] = true
	}

	if event.Text != "" {
		if event.Global {
			for _, name := range r.order {
				r.tell(r.players[name], event.Text)
			}
		} else {
			r.tell(hero, event.Text)
		}
	}

	if event.Transform != nil {
		r.change(*event.Transform)
	}

	if event.Spawn != nil {
		r.summon(hero, *event.Spawn)
	}

	if item, ok := r.thing(event.Gives); ok {
		hero.bag = append(hero.bag, item)
		r.tell(hero, "Masz teraz: "+item.Name+".")
	}

	if event.Takes != "" {
		if index := find(hero.bag, event.Takes); index >= 0 {
			r.tell(hero, "Tracisz: "+hero.bag[index].Name+".")
			hero.bag = remove(hero.bag, index)
		}
	}

	if event.Gold != 0 {
		hero.gold = max(0, hero.gold+event.Gold)
		if event.Gold > 0 {
			r.tell(hero, fmt.Sprintf("Zyskujesz %d %s.", event.Gold, r.world.Currency))
		} else {
			r.tell(hero, fmt.Sprintf("Tracisz %d %s.", -event.Gold, r.world.Currency))
		}
	}

	if event.Fact != nil {
		if fact := r.add(event.Fact.Subject, event.Fact.Text, "wydarzenie", r.everyone()...); fact != nil {
			r.learn(hero, fact)
		}
	}

	if event.Standing != 0 {
		r.esteem(hero, r.closestTown(hero.pos), event.Standing, "Osada zapamięta ten dzień.")
	}

	if event.Prices != 0 {
		for _, item := range r.world.Items {
			r.markup[item.ID] = clamp(r.markupOf(item.ID)+event.Prices, minMarkup, maxMarkup)
		}
	}

	if event.Win {
		r.won = true

		for _, name := range r.order {
			r.tell(r.players[name], fmt.Sprintf("Świat «%s» jest ocalony!", r.world.Name))
		}
	}
}

// summon brings monsters where an event says: here, into a kind of place,
// or next to a settlement.
func (r *Realm) summon(hero *player, spawn world.SpawnSpec) {
	kind := -1
	for index, monster := range r.world.Monsters {
		if strings.EqualFold(monster.Name, spawn.Monster) {
			kind = index
		}
	}

	if kind < 0 && !strings.EqualFold(spawn.Monster, r.world.Boss.Name) {
		return
	}

	var spots []world.Point

	switch {
	case spawn.Where == "" || spawn.Where == "here":
		spots = []world.Point{hero.pos}
	case strings.HasPrefix(spawn.Where, "near:"):
		town := strings.TrimPrefix(spawn.Where, "near:")
		for _, at := range r.world.Towns {
			if strings.EqualFold(r.placeAt(at).Name, town) {
				for _, step := range []world.Point{north, east, south, west} {
					if next := (world.Point{X: at.X + step.X, Y: at.Y + step.Y}); r.world.In(next) && r.world.At(next).Town < 0 {
						spots = append(spots, next)
					}
				}
			}
		}
	default:
		name := strings.TrimPrefix(spawn.Where, "place:")
		for index, tile := range r.world.Tiles {
			if tile.Town < 0 && tile.Site < 0 && strings.EqualFold(r.placeAt(world.Point{X: index % world.Width, Y: index / world.Width}).Name, name) {
				spots = append(spots, world.Point{X: index % world.Width, Y: index / world.Width})
			}
		}
	}

	if len(spots) == 0 {
		return
	}

	for range max(spawn.Count, 1) {
		at := spots[r.rng.Intn(len(spots))]
		index := r.world.Index(at)
		r.monsters[index] = append(r.monsters[index], r.newMonster(kind))
		r.revealed[index] = r.revealed[index] || at == hero.pos
	}

	// Monsters that arrive where somebody stands start a fight at once.
	for _, name := range r.order {
		if who := r.players[name]; who.connected && r.alive(who.pos) > 0 && r.fightOf(who) == nil {
			r.engage(who)
		}
	}
}
