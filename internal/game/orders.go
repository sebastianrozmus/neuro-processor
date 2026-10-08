package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// orderHelp is what a companion can be told to do.
const orderHelp = "Rozkazy dla towarzysza: ulecz · rzuć X (na Y) · atakuj X · wypij X · automat."

// Order gives a companion of the player an instruction in the player's
// words, as in "Lyrasa, ulecz siebie i załogę". In a fight the companion
// carries it out when their turn comes, or at once if it is their turn;
// outside one, healing and potions work right away. It reports false when
// the first word names nobody in the party.
func (r *Realm) Order(name, text string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	leader := r.active(name)
	if leader == nil {
		return false
	}

	words := strings.Fields(Fold(text))
	if len(words) < 2 {
		return false
	}

	member := r.mate(leader, words[0])
	if member == nil {
		return false
	}

	label := member.spec().Name

	// The rest of the name is skipped too: "Brat Jaromir, ulecz".
	rest := words[1:]
	for len(rest) > 1 && verbOf(rest[0]) == VerbUnknown && Matches(label, rest[0]) {
		rest = rest[1:]
	}

	command := Parse(strings.Join(rest, " "))

	switch command.Verb {
	case VerbAuto:
		member.auto = !member.auto
		r.tell(leader, label+map[bool]string{true: " działa teraz sam.", false: " czeka teraz na twoje decyzje."}[member.auto])

		return true
	case VerbHeal, VerbCast, VerbAttack, VerbUse:
	default:
		r.tell(leader, label+": tego nie zrobi. "+orderHelp)

		return true
	}

	if member.hp <= 0 {
		r.tell(leader, label+" leży nieprzytomny i niczego nie wykona.")

		return true
	}

	current := r.fightOf(leader)
	if current == nil {
		r.calmOrder(leader, member, command)

		return true
	}

	if now := r.current(current); now != nil && now.unit == unit(member) {
		if r.carry(current, member, command) {
			r.next(current)
		}

		return true
	}

	member.order = &command
	r.tell(leader, label+" wykona to w swojej turze.")

	return true
}

// carry makes a companion do what they were told in a fight. It reports
// whether the turn was used; an order that cannot be followed leaves the
// turn to be decided anew.
func (r *Realm) carry(current *fight, member *companion, command Command) bool {
	leader := member.owner

	switch command.Verb {
	case VerbHeal:
		power, ok := r.healing(member)
		if !ok {
			r.tell(leader, member.spec().Name+" nie ma dość mocy na leczenie.")

			return false
		}

		return r.castPower(current, member, power, "", 0)
	case VerbCast:
		power, ok := r.spell(leader, member.powers(), command.Arg)

		return ok && r.castPower(current, member, power, command.Target, command.Index)
	case VerbAttack:
		foe := r.target(leader, current, command.Arg, command.Index)
		if foe == nil {
			return false
		}

		r.strike(current, member, foe)

		return true
	case VerbUse:
		index := r.potion(member, command.Arg)
		if index < 0 {
			r.tell(leader, member.spec().Name+" nie ma takiej mikstury.")

			return false
		}

		r.drink(current, member, index)

		return true
	}

	return false
}

// calmOrder follows an order outside a fight: only healing and potions
// mean anything there.
func (r *Realm) calmOrder(leader *player, member *companion, command Command) {
	switch command.Verb {
	case VerbHeal:
		if power, ok := r.healing(member); ok {
			r.healOutside(leader, member, power)
		} else {
			r.tell(leader, member.spec().Name+" nie ma dość mocy na leczenie.")
		}
	case VerbUse:
		if index := r.potion(member, command.Arg); index >= 0 {
			r.sip(leader, member, index)
		} else {
			r.tell(leader, member.spec().Name+" nie ma takiej mikstury.")
		}
	default:
		r.tell(leader, "Nie ma tu z kim walczyć.")
	}
}

// healing picks the strongest healing power the unit can pay for.
func (r *Realm) healing(who unit) (world.ItemSpec, bool) {
	var best world.ItemSpec
	found := false

	for _, power := range who.powers() {
		cost := power.Cost()
		if isInnate(power) {
			cost = 0
		}

		if power.Effect == world.EffectHeal && who.energy() >= cost && (!found || power.Power() > best.Power()) {
			best, found = power, true
		}
	}

	return best, found
}

// healOutside lets a unit mend the whole party between fights, the fallen
// included, who come round with what the power gives.
func (r *Realm) healOutside(leader *player, who unit, power world.ItemSpec) {
	cost := power.Cost()
	if isInnate(power) {
		cost = 0
	}

	healed := max(1, power.Power()+Modifier(who.scores()[Wisdom]))
	who.setEnergy(who.energy() - cost)

	var mended []string

	for _, ally := range append([]unit{leader}, r.allies(leader)...) {
		if ally.health() < ally.maxHealth() {
			ally.setHealth(min(ally.maxHealth(), ally.health()+healed))
			mended = append(mended, ally.label())
		}
	}

	if len(mended) == 0 {
		r.tell(leader, fmt.Sprintf("%s rzuca: %s, ale nikt nie jest ranny.", who.label(), power.Name))

		return
	}

	r.tell(leader, fmt.Sprintf("%s rzuca: %s. Rany się zamykają (do %d zdrowia): %s.", who.label(), power.Name, healed, join(mended)))
}

// allies lists every companion of a player, standing or not.
func (r *Realm) allies(leader *player) []unit {
	all := make([]unit, 0, len(leader.party))
	for _, member := range leader.party {
		all = append(all, member)
	}

	return all
}

// potion finds a potion in a companion's bag by name, or any when no
// name is given.
func (r *Realm) potion(member *companion, what string) int {
	for index, item := range member.bag {
		if item.Kind == world.KindPotion && (what == "" || Matches(item.Name, what)) {
			return index
		}
	}

	return -1
}

// sip lets a companion drink a potion between fights.
func (r *Realm) sip(leader *player, member *companion, index int) {
	item := member.bag[index]
	member.bag = remove(member.bag, index)

	if item.Effect == world.EffectMana {
		gained := min(item.Power(), member.maxEnergy()-member.mana)
		member.mana += gained
		r.tell(leader, fmt.Sprintf("%s używa: %s (+%d mocy).", member.spec().Name, item.Name, gained))

		return
	}

	gained := min(item.Power(), member.maxHealth()-member.hp)
	member.hp, member.poison = member.hp+gained, 0
	r.tell(leader, fmt.Sprintf("%s używa: %s (+%d zdrowia).", member.spec().Name, item.Name, gained))
}

// heal is the hero's own "ulecz": the strongest healing they can pay for,
// on the party in a fight or between fights alike.
func (r *Realm) heal(healer *player) {
	current := r.fightOf(healer)
	if current == nil {
		if power, ok := r.healing(healer); ok {
			r.healOutside(healer, healer, power)
		} else if len(healer.spells) == 0 {
			r.tell(healer, "Nie znasz żadnej mocy leczącej. Towarzysza z taką mocą poprosisz: «Imię, ulecz».")
		} else {
			r.tell(healer, "Za mało mocy na leczenie.")
		}

		return
	}

	_, who := r.turn(healer)
	if who == nil {
		return
	}

	power, ok := r.healing(who)
	if !ok {
		r.tell(healer, who.label()+" nie ma dość mocy na leczenie.")

		return
	}

	if r.castPower(current, who, power, "", 0) {
		r.next(current)
	}
}
