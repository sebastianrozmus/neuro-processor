package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	// MasterName is the name the game master's lines are shown under.
	MasterName = "Mistrz Gry"
	// MasterMark opens every line of the game master in the log.
	MasterMark = MasterName + ": "

	masterPersona = "Mistrz Gry prowadzący tę przygodę. Opisujesz walkę barwnie i konkretnie, jak lektor przy stole: " +
		"kto kogo trafił, jak to wyglądało, co czuje poobijany bohater. Masz lekki, nieco złośliwy humor i słabość do dramatycznych pauz."

	masterTimeout = 90 * time.Second
)

// fighters names the heroes and companions on the players' side of a
// fight: the master may leave them out, but not mangle them.
func (r *Realm) fighters(members []string) []string {
	names := append([]string(nil), members...)
	for _, name := range members {
		if hero := r.players[name]; hero != nil {
			for _, mate := range hero.party {
				names = append(names, mate.spec().Name)
			}
		}
	}

	return names
}

// SetMaster gives the world a game master: a narrator asked, after every
// round of combat, to describe what the rules just resolved. Its lines
// arrive asynchronously and go to everyone in the fight. Record, when set,
// receives every line the master answered well.
func (r *Realm) SetMaster(narrator Narrator, record func(NarrateRequest, string)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.master, r.record = narrator, record
}

// narrateRound asks the game master to describe the round that just ended
// in a fight. While one description is still being written, the next
// rounds go unnarrated rather than queue up.
func (r *Realm) narrateRound(current *fight, members []string) {
	if r.master == nil || len(current.events) == 0 || current.narrating {
		current.events = nil

		return
	}

	// A long round is told in parts, one call each, every part checked
	// against its own events: told at once, five events and more came out
	// with blows missed or turned into misses.
	parts := RoundParts(strings.Join(current.events, " "))
	requests := make([]NarrateRequest, 0, len(parts))
	lore, guarded := r.sceneLore(current, members), r.fighters(members)

	for _, part := range parts {
		requests = append(requests, NarrateRequest{
			Merchant: Merchant{Name: MasterName, Persona: masterPersona, Currency: r.world.Currency},
			Lore:     lore,
			Decision: Decision{Outcome: OutcomeScene, Scene: part, Part: len(parts) > 1, Currency: r.world.Currency},
			Guarded:  guarded,
		})
	}

	current.events, current.narrating = nil, true
	master, record := r.master, r.record

	go func() {
		var (
			answered []NarrateRequest
			told     []string
		)

		for _, request := range requests {
			// A part the master failed is left to the rules' own lines,
			// which the player sees anyway.
			if text, err := narratePart(master, request); err == nil && text != "" {
				answered, told = append(answered, request), append(told, text)
			}
		}

		r.mu.Lock()
		defer r.mu.Unlock()
		defer r.notify()

		current.narrating = false

		if len(told) == 0 {
			return
		}

		said := make([]string, 0, len(told))

		for index, text := range told {
			if record != nil {
				record(answered[index], text)
			}

			// The master narrates in no particular mood; a tag is dropped.
			_, text = speech.SplitMood(text)
			said = append(said, text)
		}

		for _, name := range members {
			if listener := r.players[name]; listener != nil {
				r.tell(listener, MasterMark+strings.Join(said, " "))
			}
		}
	}()
}

// narratePart has the master tell a round, or one part of it, each part
// with time of its own.
func narratePart(master Narrator, request NarrateRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), masterTimeout)
	defer cancel()

	return Narrate(ctx, master, request)
}

// sceneLore words the state of a fight for the game master.
func (r *Realm) sceneLore(current *fight, members []string) string {
	var lore strings.Builder

	place := r.placeAt(current.pos)
	fmt.Fprintf(&lore, "Świat: %s. Miejsce walki: %s. %s\n", r.world.Name, place.Name, place.Description)

	if style := r.world.Style; !style.Empty() {
		fmt.Fprintf(&lore, "Styl opowieści: %s\n", style.Look)
	}
	lore.WriteString("Bohaterowie:\n")

	for _, name := range members {
		hero := r.players[name]
		if hero == nil {
			continue
		}

		fmt.Fprintf(&lore, "- %s, poziom %d, zdrowie %d/%d, broń: %s, pancerz: %s\n",
			hero.name, hero.level, max(hero.hp, 0), hero.maxHP(), orNothing(hero.weapon), orNothing(hero.armor))

		for _, member := range hero.party {
			spec := member.spec()
			fmt.Fprintf(&lore, "- %s, %s w drużynie %s, zdrowie %d/%d\n", spec.Name, roleName(spec.Role), hero.name, max(member.hp, 0), member.maxHealth())
		}
	}

	lore.WriteString("Przeciwnicy:\n")

	for _, foe := range r.monsters[r.world.Index(current.pos)] {
		state := fmt.Sprintf("zdrowie %d/%d", max(foe.hp, 0), foe.maxHP)
		if foe.hp <= 0 {
			state = "pokonany"
		}

		fmt.Fprintf(&lore, "- %s (%s), %s\n", foe.name, describeShape(foe.shape), state)
	}

	return lore.String()
}

// describeShape words what kind of creature a monster is.
func describeShape(shape world.Shape) string {
	switch shape {
	case "beast":
		return "bestia"
	case "undead":
		return "nieumarły"
	case "humanoid":
		return "humanoid"
	case "spirit":
		return "duch"
	case "serpent":
		return "wąż"
	case "insect":
		return "owad lub rój"
	case "machine":
		return "maszyna"
	}

	return "galareta"
}
