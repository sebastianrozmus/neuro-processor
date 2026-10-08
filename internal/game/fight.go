package game

import (
	"fmt"
	"sort"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// shapePace is how quick monsters of each shape are.
var shapePace = map[world.Shape]int{
	"beast": 7, "spirit": 7, "serpent": 6, "insect": 6, "humanoid": 5, "machine": 4, "undead": 3, "blob": 2,
}

const bossPaceBonus = 2

// actor is one combatant waiting for a turn: a unit of the heroes' side or
// a monster.
type actor struct {
	unit unit
	foe  *monster
}

type fight struct {
	pos     world.Point
	members []string
	round   int
	timer   *time.Timer
	// board is the battlefield of the D&D mode; nil otherwise. began
	// notes the turn whose movement was last dealt out.
	board *board
	began [2]int
	// queue is the order of this round's turns; turn points at the current.
	queue []actor
	turn  int
	// events are the rules' lines of the current round, for the master.
	events    []string
	narrating bool
}

// Attack hits the chosen living monster with whoever has the turn, if the
// turn belongs to the player's party.
func (r *Realm) Attack(name string, target int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	if attacker := r.active(name); attacker != nil {
		r.attack(attacker, "", target)
	}
}

// Cast uses the power at the given position among those the unit with the
// turn knows, aimed at the chosen living monster when it needs a target.
func (r *Realm) Cast(name string, power, target int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	caster := r.active(name)
	if caster == nil {
		return
	}

	current, who := r.turn(caster)
	if current == nil {
		return
	}

	known := who.powers()
	if power < 0 || power >= len(known) {
		return
	}

	if r.castPower(current, who, known[power], "", target) {
		r.next(current)
	}
}

// Auto lets the companion whose turn it is act on their own judgement.
func (r *Realm) Auto(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	leader := r.active(name)
	if leader == nil {
		return
	}

	current, who := r.turn(leader)
	if current == nil {
		return
	}

	member, ok := who.(*companion)
	if !ok {
		r.tell(leader, "Automat jest dla towarzyszy; swoją postacią kierujesz sam.")

		return
	}

	r.autoAct(current, member)
	r.next(current)
}

// Flee tries to escape from a fight to the tile the player came from.
func (r *Realm) Flee(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	if runner := r.active(name); runner != nil {
		r.flee(runner)
	}
}

func (r *Realm) attack(leader *player, named string, index int) {
	current, who := r.turn(leader)
	if current == nil {
		return
	}

	foe := r.target(leader, current, named, index)
	if foe == nil || !r.inReach(current, who, foe) {
		return
	}

	r.strike(current, who, foe)
	r.next(current)
}

// strike is a plain blow of a unit at a monster: by the D&D rules a d20
// decides whether it lands and the weapon's die what it does.
func (r *Realm) strike(current *fight, who unit, foe *monster) {
	if !r.dnd() {
		r.hit(current, foe, r.roll(who.might()), BlowHit, who.label(), who.label()+" atakuje: ")

		return
	}

	strength := Modifier(who.scores()[Strength])
	hit, crit, roll := r.attackRoll(proficiency(levelOf(who))+strength, monsterClass(foe))

	if !hit {
		r.tellFight(current, fmt.Sprintf("%s atakuje: %s. Pudło (k20: %d przeciw KP %d).", who.label(), foe.name, roll, monsterClass(foe)))

		return
	}

	die := weaponDie(who)
	damage := r.d(die) + strength
	prefix := fmt.Sprintf("%s atakuje (k20: %d): ", who.label(), roll)

	if crit {
		damage += r.d(die)
		prefix = fmt.Sprintf("%s atakuje (k20: 20, trafienie krytyczne!): ", who.label())
	}

	r.hit(current, foe, max(1, damage), BlowHit, who.label(), prefix)
}

func (r *Realm) flee(runner *player) {
	current, _ := r.turn(runner)
	if current == nil {
		return
	}

	// Running from a monster's reach gives it a free blow first.
	if r.dnd() {
		for _, foe := range r.foes(current.pos) {
			if r.adjacent(current, foe, unit(runner)) {
				r.tellFight(current, foe.name+" wyprowadza atak okazyjny na uciekającego.")
				r.maul(current, foe, runner, 1)

				if runner.hp <= 0 || r.fightOf(runner) == nil {
					return
				}
			}
		}
	}

	// Nimble heroes slip away more often.
	if r.rng.Intn(100) >= fleePercent+5*Modifier(runner.abilities[Dexterity]) {
		r.tellFight(current, runner.name+" próbuje uciec, bez skutku.")
		r.next(current)

		return
	}

	r.tellFight(current, runner.name+" ucieka z walki.")
	current.stopTimer()
	r.cure(runner)
	r.leaveFight(current, runner)
	r.place(runner, runner.prev, -1)
	r.proceed(current)
}

// turn returns the player's fight and the unit of theirs whose turn it
// is, telling them why not otherwise.
func (r *Realm) turn(leader *player) (*fight, unit) {
	current := r.fightOf(leader)
	if current == nil {
		r.tell(leader, "Nie ma tu z kim walczyć.")

		return nil, nil
	}

	now := r.current(current)
	if now == nil || now.unit == nil {
		return nil, nil
	}

	if now.unit.leader() != leader {
		r.tell(leader, "Teraz tura: "+now.unit.label()+". Czekasz.")

		return nil, nil
	}

	return current, now.unit
}

// current returns the actor whose turn it is, or nil between rounds.
func (r *Realm) current(current *fight) *actor {
	if current.turn < 0 || current.turn >= len(current.queue) {
		return nil
	}

	return &current.queue[current.turn]
}

// target picks the monster a player means: by name if they gave one, by
// position otherwise.
func (r *Realm) target(leader *player, current *fight, named string, index int) *monster {
	foes := r.foes(current.pos)

	if named != "" {
		for _, foe := range foes {
			if Matches(foe.name, named) {
				return foe
			}
		}

		r.tell(leader, "Nie ma tu takiego przeciwnika.")

		return nil
	}

	return foes[min(max(index, 0), len(foes)-1)]
}

// hit wounds a monster, as far as its trait lets a blow of that kind
// through, and says so after the prefix.
func (r *Realm) hit(current *fight, foe *monster, damage int, kind BlowKind, from, prefix string) {
	note := ""

	switch {
	case foe.shape == "spirit" && kind == BlowHit:
		damage, note = max(1, damage/2), " (widmo)"
	case foe.shape == "humanoid" && kind == BlowHit && r.rng.Intn(shieldChance) == 0:
		damage, note = max(1, damage/2), " (tarcza)"
	case foe.shape == "machine":
		damage, note = max(1, damage-foe.tier-1), " (pancerz)"
	}

	foe.hp -= damage
	r.blowBy(current.pos, foe.id, -damage, kind, from, foe.name)
	r.tellFight(current, fmt.Sprintf("%s%s traci %d zdrowia%s.", prefix, foe.name, damage, note))

	if foe.hp <= 0 {
		r.tellFight(current, foe.name+" pada.")
		r.fell(current, foe)
	}
}

// engage adds a player to the fight on their tile, starting it if needed.
// A new fight begins at once with whoever is quickest.
func (r *Realm) engage(joiner *player) {
	current := r.fights[joiner.pos]
	names := make([]string, 0, r.alive(joiner.pos))

	for _, foe := range r.foes(joiner.pos) {
		names = append(names, foe.name+" ("+foe.traitNames()+")")
	}

	r.meet(joiner, joiner.pos)
	r.tell(joiner, "Walka! Przeciwnicy: "+join(names)+".")

	if current == nil {
		current = &fight{pos: joiner.pos, members: []string{joiner.name}}
		r.fights[joiner.pos] = current
		r.startRound(current)
		r.proceed(current)

		return
	}

	r.tellFight(current, joiner.name+" dołącza do walki.")
	current.members = append(current.members, joiner.name)

	if r.dnd() {
		r.seat(current)
	}

	// Latecomers act at the end of the round that is going on.
	for _, who := range r.units(joiner) {
		current.queue = append(current.queue, actor{unit: who})
	}

	r.proceed(current)
}

// units lists a player's hero and standing companions.
func (r *Realm) units(leader *player) []unit {
	all := []unit{leader}

	for _, member := range leader.party {
		if member.hp > 0 {
			all = append(all, member)
		}
	}

	return all
}

// startRound opens a round: everyone still standing gets a turn, the
// quickest first. Heroes go before monsters of the same pace.
func (r *Realm) startRound(current *fight) {
	current.stopTimer()
	current.round++
	current.turn = 0
	current.queue = nil

	if current.round > 1 {
		r.tick(current)
	}

	if r.dnd() {
		r.seat(current)
	}

	for _, name := range current.members {
		for _, who := range r.units(r.players[name]) {
			current.queue = append(current.queue, actor{unit: who})
		}
	}

	for _, foe := range r.foes(current.pos) {
		current.queue = append(current.queue, actor{foe: foe})
	}

	sort.SliceStable(current.queue, func(a, b int) bool {
		return r.paceOf(current, current.queue[a]) > r.paceOf(current, current.queue[b])
	})
}

// dealMoves gives the actor whose turn begins its movement, once per turn.
func (r *Realm) dealMoves(current *fight, who any) {
	if current.board == nil {
		return
	}

	if turn := [2]int{current.round, current.turn}; turn != current.began {
		current.began = turn
		current.board.moves = r.cellsOf(who)
	}
}

// tick is what a new round does by itself: venom works in the poisoned,
// and the undead knit together. Venom never kills; it leaves its victim
// at a point of health.
func (r *Realm) tick(current *fight) {
	for _, name := range current.members {
		for _, who := range r.units(r.players[name]) {
			if who.venom() <= 0 {
				continue
			}

			who.setVenom(who.venom() - 1)
			hurt := min(poisonDamage, who.health()-1)

			if hurt > 0 {
				who.setHealth(who.health() - hurt)
				r.blowBy(current.pos, -1, -hurt, BlowPoison, "", who.label())
				r.tellFight(current, fmt.Sprintf("Jad działa: %s traci %d zdrowia.", who.label(), hurt))
			}
		}
	}

	for _, foe := range r.foes(current.pos) {
		if gained := min(2*foe.tier, foe.maxHP-foe.hp); foe.shape == "undead" && gained > 0 {
			foe.hp += gained
			r.blowBy(current.pos, foe.id, gained, BlowRegen, "", foe.name)
			r.tellFight(current, fmt.Sprintf("%s zrasta się: odzyskuje %d zdrowia.", foe.name, gained))
		}
	}
}

// paceOf is what orders the turns: pace, or the initiative rolled for the
// fight in the D&D mode.
func (r *Realm) paceOf(current *fight, who actor) int {
	if current.board != nil {
		if who.foe != nil {
			return current.board.initiative[who.foe]
		}

		return current.board.initiative[who.unit]
	}

	if who.foe != nil {
		return who.foe.pace()
	}

	return who.unit.pace()
}

func (m *monster) pace() int {
	pace := shapePace[m.shape] + m.tier - 1
	if m.boss {
		pace += bossPaceBonus
	}

	return pace
}

// next ends the current actor's turn and plays the fight on. The clock
// set for that turn is stopped: the next one gets its own.
func (r *Realm) next(current *fight) {
	current.stopTimer()
	current.turn++
	r.proceed(current)
}

// proceed plays the fight until it needs a decision from a player or ends:
// monsters strike at once, companions on automatic act at once, fallen and
// departed actors are skipped and a finished round opens the next.
func (r *Realm) proceed(current *fight) {
	for r.fights[current.pos] == current {
		switch {
		case len(current.members) == 0:
			r.disband(current)

			return
		case r.alive(current.pos) == 0:
			r.victory(current)

			return
		case current.turn >= len(current.queue):
			r.narrateRound(current, append([]string(nil), current.members...))
			r.startRound(current)

			continue
		}

		now := current.queue[current.turn]

		switch {
		case now.foe != nil && now.foe.hp > 0:
			r.dealMoves(current, now.foe)
			r.monsterStrikes(current, now.foe)
		case now.foe != nil:
		case !current.has(now.unit.leader().name) || now.unit.health() <= 0:
		default:
			r.dealMoves(current, now.unit)
			member, isCompanion := now.unit.(*companion)
			if isCompanion && member.order != nil {
				told := *member.order
				member.order = nil

				if r.carry(current, member, told) {
					break
				}
			}

			if isCompanion && (member.auto || member.owner.autoFight) {
				r.autoAct(current, member)

				break
			}

			r.armTimer(current)

			return
		}

		current.turn++
	}
}

// monsterStrikes lets a monster hit the heroes' side the way its shape
// does: a frenzied beast twice, a swarm two at once, the rest once.
func (r *Realm) monsterStrikes(current *fight, foe *monster) {
	blows := 1
	if foe.shape == "beast" && foe.hp*2 < foe.maxHP {
		blows = 2
	}

	// On the board the monster first closes in on whoever is nearest.
	if r.dnd() {
		target := r.nearestUnit(current, foe)
		if target == nil {
			return
		}

		r.approach(current, foe, target)
	}

	for range blows {
		targets := r.standing(current)

		if r.dnd() {
			var near []unit
			for _, who := range targets {
				if r.adjacent(current, foe, who) {
					near = append(near, who)
				}
			}

			if len(near) == 0 {
				r.tellFight(current, foe.name+" zbliża się.")

				return
			}

			targets = near
		}

		if len(targets) == 0 {
			return
		}

		if foe.shape == "insect" && len(targets) > 1 {
			first := r.rng.Intn(len(targets))
			second := (first + 1 + r.rng.Intn(len(targets)-1)) % len(targets)
			r.maul(current, foe, targets[first], 2)
			r.maul(current, foe, targets[second], 2)

			continue
		}

		r.maul(current, foe, targets[r.rng.Intn(len(targets))], 1)
	}
}

// standing lists everyone on the heroes' side of a fight still on their feet.
func (r *Realm) standing(current *fight) []unit {
	var targets []unit
	for _, name := range current.members {
		targets = append(targets, r.units(r.players[name])...)
	}

	return targets
}

// maul is one blow of a monster at a unit, a share of its full force. By
// the D&D rules it has to beat the armour class first and venom allows a
// saving throw.
func (r *Realm) maul(current *fight, foe *monster, struck unit, share int) {
	damage, note := 0, ""

	if r.dnd() {
		bonus, class := 2+foe.tier, armourClass(struck)
		if foe.boss {
			bonus += 2
		}

		// Acid eats through armour: only nimbleness counts against a blob.
		if foe.shape == "blob" {
			class = 10 + Modifier(struck.scores()[Dexterity])
			if struck.guard() > 0 {
				note = " Kwas przeżera zbroję."
			}
		}

		hit, crit, roll := r.attackRoll(bonus, class)
		if !hit {
			r.tellFight(current, fmt.Sprintf("%s atakuje: %s. Pudło (k20: %d przeciw KP %d).", foe.name, struck.label(), roll, class))

			return
		}

		count, sides, plus := monsterDice(foe)
		if crit {
			count *= 2
			note += " Trafienie krytyczne!"
		}

		for range count {
			damage += r.d(sides)
		}

		damage = (damage + plus) / share
	} else {
		damage = r.roll(foe.attack) / share

		if foe.shape == "blob" {
			if struck.guard() > 0 {
				note = " Kwas przeżera zbroję."
			}
		} else {
			damage -= struck.guard()
		}
	}

	// Furia: a cornered boss hits half as hard again.
	if foe.boss && foe.hp*3 < foe.maxHP {
		damage = damage * 3 / 2
	}

	damage = max(1, damage)
	struck.setHealth(struck.health() - damage)
	r.blowBy(current.pos, -1, -damage, BlowWound, foe.name, struck.label())

	if foe.shape == "serpent" && struck.health() > 0 {
		if r.dnd() && r.save(Modifier(struck.scores()[Constitution]), 10+2*foe.tier) {
			note += " Rzut obronny udany: jad nie działa."
		} else {
			struck.setVenom(poisonRounds)
			note += " Jad działa!"
		}
	}

	r.tellFight(current, fmt.Sprintf("%s atakuje: %s traci %d zdrowia.%s", foe.name, struck.label(), damage, note))

	if struck.health() > 0 {
		return
	}

	if member, ok := struck.(*companion); ok {
		member.hp, member.poison = 0, 0
		r.tellFight(current, member.spec().Name+" pada nieprzytomny. Odpoczynek w osadzie postawi go na nogi.")

		return
	}

	r.fall(current, struck.(*player))
}

// autoAct lets a companion choose: a potion when badly hurt, healing when
// an ally is wounded, a blast when they have one, a blow otherwise.
func (r *Realm) autoAct(current *fight, member *companion) {
	foes := r.foes(current.pos)
	if len(foes) == 0 {
		return
	}

	if member.hp*5 < member.maxHealth()*2 {
		for index, item := range member.bag {
			if item.Kind == world.KindPotion && item.Effect == world.EffectHeal {
				r.drink(current, member, index)

				return
			}
		}
	}

	var heal, blast, bolt *world.ItemSpec
	for _, power := range member.powers() {
		power := power
		if member.mana < power.Cost() && !isInnate(power) {
			continue
		}

		switch power.Effect {
		case world.EffectHeal:
			heal = &power
		case world.EffectBlast:
			blast = &power
		case world.EffectBolt:
			bolt = &power
		}
	}

	weakest := foes[0]
	for _, foe := range foes {
		if foe.hp < weakest.hp {
			weakest = foe
		}
	}

	// On the board a companion walks up to its prey before anything else.
	if r.dnd() {
		r.approach(current, member, weakest)
	}

	switch {
	case heal != nil && r.wounded(member.owner) && r.castPower(current, member, *heal, "", 0):
	case blast != nil && len(foes) > 1 && r.castPower(current, member, *blast, "", 0):
	case bolt != nil && r.castPower(current, member, *bolt, "", r.indexOf(foes, weakest)):
	case r.dnd() && !r.adjacent(current, member, weakest):
		r.tellFight(current, member.spec().Name+" podchodzi.")
	default:
		r.strike(current, member, weakest)
	}
}

// wounded reports whether somebody in a player's party is below half health.
func (r *Realm) wounded(leader *player) bool {
	for _, who := range r.units(leader) {
		if who.health()*2 < who.maxHealth() {
			return true
		}
	}

	return false
}

func (r *Realm) indexOf(foes []*monster, wanted *monster) int {
	for index, foe := range foes {
		if foe == wanted {
			return index
		}
	}

	return 0
}

// drink makes a unit use the potion at a bag position during a fight.
func (r *Realm) drink(current *fight, who unit, index int) {
	bag := who.pack()
	item := (*bag)[index]
	*bag = remove(*bag, index)

	if item.Effect == world.EffectMana {
		gained := min(item.Power(), who.maxEnergy()-who.energy())
		who.setEnergy(who.energy() + gained)
		r.tellFight(current, fmt.Sprintf("%s używa: %s (+%d mocy).", who.label(), item.Name, gained))

		return
	}

	gained := min(item.Power(), who.maxHealth()-who.health())
	who.setHealth(who.health() + gained)
	r.blowBy(current.pos, -1, gained, BlowHeal, who.label(), who.label())

	cured := ""
	if who.venom() > 0 {
		who.setVenom(0)
		cured = " Jad przestaje działać."
	}

	r.tellFight(current, fmt.Sprintf("%s używa: %s (+%d zdrowia).%s", who.label(), item.Name, gained, cured))
}

// fall takes a beaten player, with their companions, out of the fight and
// back to the settlement they visited last.
func (r *Realm) fall(current *fight, fallen *player) {
	r.tellFight(current, fallen.name+" pada bez sił.")
	r.cure(fallen)
	r.leaveFight(current, fallen)

	home := r.world.Towns[fallen.home]
	lost := fallen.gold * fallLossPercent / 100
	fallen.gold -= lost
	fallen.hp = fallen.maxHP() / 2
	fallen.pos, fallen.prev = home, home
	fallen.shop, fallen.talks = -1, nil

	r.tell(fallen, fmt.Sprintf("Budzisz się w osadzie %s. Po drodze przepadło %d %s.", r.placeAt(home).Name, lost, r.world.Currency))
}

// leaveFight takes a player and their companions out of a fight.
func (r *Realm) leaveFight(current *fight, leaving *player) {
	current.remove(leaving.name)

	for _, who := range r.units(leaving) {
		r.unseat(current, who)
	}
}

// disband ends a fight nobody is left in; the monsters recover.
func (r *Realm) disband(current *fight) {
	current.stopTimer()
	current.events = nil
	delete(r.fights, current.pos)

	for _, foe := range r.monsters[r.world.Index(current.pos)] {
		if foe.hp > 0 {
			foe.hp = foe.maxHP
		}
	}
}

// victory ends a won fight: gold is shared, experience given to everyone
// who fought and loot drops to the ground.
func (r *Realm) victory(current *fight) {
	current.stopTimer()
	delete(r.fights, current.pos)

	index := r.world.Index(current.pos)
	beaten := r.monsters[index]
	r.monsters[index] = nil

	gold, xp, boss, fallen := 0, 0, false, ""

	for _, foe := range beaten {
		stats := world.StatsFor(foe.tier, foe.boss)
		gold += stats.GoldMin + r.rng.Intn(stats.GoldMax-stats.GoldMin+1)
		boss = boss || foe.boss

		if foe.boss {
			fallen = foe.name
		}

		if foe.boss {
			xp += bossXP
		} else {
			xp += xpFor[foe.tier-1]
		}

		if r.rng.Intn(100) < lootPercent {
			r.ground[index] = append(r.ground[index], r.loot(foe))
		}
	}

	// Gold is split; experience is not, so fighting together never costs.
	share := gold / len(current.members)
	r.tellFight(current, fmt.Sprintf("Zwycięstwo! Łup na głowę: %d %s, doświadczenie: %d.", share, r.world.Currency, xp))

	for _, name := range current.members {
		r.players[name].gold += share
		r.cure(r.players[name])
		r.forget(r.players[name])
		r.reward(r.players[name], xp)
		r.rewardParty(r.players[name], xp)
	}

	if lying := r.ground[index]; len(lying) > 0 {
		r.tellFight(current, "Na ziemi leży: "+join(itemNames(lying))+".")
	}

	if boss {
		r.won = true

		for _, name := range r.order {
			r.tell(r.players[name], fmt.Sprintf("%s pokonany. Świat «%s» jest ocalony!", fallen, r.world.Name))
		}

		r.aftermath(fallen)
	}

	r.narrateRound(current, append([]string(nil), current.members...))
}

// loot picks an item a monster of that strength might carry.
func (r *Realm) loot(foe *monster) world.ItemSpec {
	tier := foe.tier
	if foe.boss {
		tier = world.MaxTier
	}

	var fitting []world.ItemSpec

	for _, item := range r.world.Items {
		if item.Tier <= tier {
			fitting = append(fitting, item)
		}
	}

	if len(fitting) == 0 {
		fitting = r.world.Items
	}

	return fitting[r.rng.Intn(len(fitting))]
}

// armTimer makes sure a turn ends even if its player never decides: the
// unit then simply attacks.
func (r *Realm) armTimer(current *fight) {
	if current.timer != nil {
		return
	}

	round, turn := current.round, current.turn
	current.timer = time.AfterFunc(r.roundTimeout, func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		current.timer = nil

		if r.fights[current.pos] != current || current.round != round || current.turn != turn {
			return
		}

		now := current.queue[current.turn]
		if now.unit != nil && now.unit.health() > 0 && len(r.foes(current.pos)) > 0 {
			foe := r.foes(current.pos)[0]

			if r.dnd() {
				foe = r.nearestFoe(current, now.unit)
				r.approach(current, now.unit, foe)
			}

			if r.dnd() && !r.adjacent(current, now.unit, foe) {
				r.tellFight(current, "Czas na decyzję minął: "+now.unit.label()+" podchodzi.")
			} else {
				r.tellFight(current, "Czas na decyzję minął: "+now.unit.label()+" atakuje.")
				r.strike(current, now.unit, foe)
			}
		}

		r.next(current)
		r.notify()
	})
}

func (r *Realm) roll(base int) int {
	return base + r.rng.Intn(base/2+1)
}

func (r *Realm) fightOf(member *player) *fight {
	if current := r.fights[member.pos]; current != nil && current.has(member.name) {
		return current
	}

	return nil
}

func (r *Realm) tellFight(current *fight, text string) {
	current.events = append(current.events, text)

	for _, name := range current.members {
		r.tell(r.players[name], text)
	}
}

func (f *fight) has(name string) bool {
	return contains(f.members, name)
}

func (f *fight) remove(name string) {
	for index, member := range f.members {
		if member == name {
			f.members = append(f.members[:index], f.members[index+1:]...)

			break
		}
	}
}

func (f *fight) stopTimer() {
	if f.timer != nil {
		f.timer.Stop()
		f.timer = nil
	}
}
