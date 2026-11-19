package game

import (
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The world does not end with its boss. When the threat falls the land
// breathes out: prices drop, the worst places grow quieter, every
// settlement owes the heroes. Then the strongest of what survived gathers
// in the empty lair and, a few days later, a new threat has a name.

const (
	// heirDays is how long after a boss falls that its heir rises.
	heirDays = 3
	// peaceMarkup is what victory takes off prices; peaceStanding what
	// every settlement grants.
	peaceMarkup   = -20
	peaceStanding = 3
	heirSuffix    = "Alfa"
)

// heir is the threat growing in the lair.
type heir struct {
	kind int
	due  int
}

// aftermath settles the world after the boss has fallen and sets the
// next threat growing.
func (r *Realm) aftermath(fallen string) {
	for _, item := range r.world.Items {
		r.markup[item.ID] = clamp(r.markupOf(item.ID)+peaceMarkup, minMarkup, maxMarkup)
	}

	for index := range r.places {
		if r.places[index].Danger == world.MaxTier {
			r.places[index].Danger--
			r.places[index].Description += " Od upadku, który nazywano " + fallen + ", jest tu spokojniej."
		}
	}

	for _, name := range r.order {
		hero := r.players[name]
		for town := range r.world.Towns {
			r.esteem(hero, town, peaceStanding, "Osady pamiętają, kto je ocalił.")
		}

		r.tell(hero, "Kupcy opuszczają ceny, najgroźniejsze ziemie cichną. Ale świat się nie kończy: coś zbiera się w pustym leżu.")
	}

	kind := r.strongestKind()
	if kind < 0 {
		return
	}

	r.heir = &heir{kind: kind, due: r.day + heirDays}

	lair := r.placeAt(r.world.Lair).Name
	r.add(r.world.Monsters[kind].Name, fmt.Sprintf("Po upadku, który nazywano %s, ocalałe stworzenia tego rodzaju ciągną do miejsca: %s. Za kilka dni będą miały przywódcę.", fallen, lair), "wydarzenie", r.everyone()...)
}

// strongestKind is the kind of monster with the most survivors, the
// higher tier winning ties, or -1 when nothing is left.
func (r *Realm) strongestKind() int {
	counts := map[int]int{}

	for _, group := range r.monsters {
		for _, foe := range group {
			if foe.hp > 0 && foe.kind >= 0 {
				counts[foe.kind]++
			}
		}
	}

	best := -1
	for kind, count := range counts {
		if best < 0 || count > counts[best] || (count == counts[best] && r.world.Monsters[kind].Tier > r.world.Monsters[best].Tier) {
			best = kind
		}
	}

	return best
}

// rise brings the heir into the lair when its day has come.
func (r *Realm) rise() {
	if r.heir == nil || r.day < r.heir.due {
		return
	}

	kind := r.heir.kind
	r.heir = nil

	leader := r.heirMonster(kind)
	index := r.world.Index(r.world.Lair)
	r.monsters[index] = append(r.monsters[index], leader)
	r.won = false

	for _, name := range r.order {
		r.tell(r.players[name], fmt.Sprintf("Nowe zagrożenie: %s zbiera siły w miejscu, które nazywa się %s. Świat znów czeka na ratunek.", leader.name, r.placeAt(r.world.Lair).Name))
	}

	r.add(leader.name, fmt.Sprintf("Przywódca stworzeń: %s. Siedziba: %s, %s.", r.world.Monsters[kind].Name, r.placeAt(r.world.Lair).Name, bearing(r.world.Start(), r.world.Lair)), "wydarzenie", r.everyone()...)
}

// heirMonster makes a boss out of a kind of monster.
func (r *Realm) heirMonster(kind int) *monster {
	r.minted++
	spec := r.world.Monsters[kind]
	stats := world.StatsFor(0, true)

	return &monster{id: r.minted, kind: kind, name: spec.Name + " " + heirSuffix, shape: spec.Shape, tier: world.MaxTier, boss: true, hp: stats.HP, maxHP: stats.HP, attack: stats.Attack}
}
