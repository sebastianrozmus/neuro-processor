package game

import (
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// everything is what "podnieś wszystko" asks for.
const everything = "wszystko"

// Execute carries out a command typed by a player. Commands that only
// concern the interface (bag, save, help, quit) are not handled here.
func (r *Realm) Execute(name string, command Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	actor := r.active(name)
	if actor == nil {
		return
	}

	fighting := r.fightOf(actor) != nil

	if r.adventure() && r.explore(actor, command) {
		return
	}

	switch command.Verb {
	case VerbGo:
		r.walk(actor, command.Dir)
	case VerbLook:
		r.look(actor)
	case VerbAttack:
		r.attack(actor, command.Arg, command.Index)
	case VerbCast:
		r.cast(actor, command)
	case VerbHeal:
		r.heal(actor)
	case VerbFlee:
		r.flee(actor)
	case VerbUse:
		r.use(actor, command.Arg)
	case VerbSay:
		r.say(actor, command.Arg)
	case VerbAuto:
		actor.autoFight = !actor.autoFight
		r.tell(actor, map[bool]string{true: "Towarzysze walczą teraz sami.", false: "Towarzyszami kierujesz teraz sam."}[actor.autoFight])

		if current := r.fightOf(actor); current != nil {
			r.proceed(current)
		}
	case VerbGive:
		r.give(actor, command.Arg)
	case VerbTakeBack:
		r.takeBack(actor, command.Arg)
	case VerbQuest:
		for _, line := range QuestLines(r.questViews(actor)) {
			r.tell(actor, line)
		}
	case VerbAccept:
		r.acceptQuest(actor)
	case VerbTake, VerbDrop, VerbRest, VerbVisit, VerbHire, VerbDismiss:
		if fighting {
			r.tell(actor, "Najpierw walka albo ucieczka.")

			return
		}

		r.calm(actor, command)
	default:
		if fighting || command.Arg == "" || r.visit(actor, command.Arg, false) < 0 {
			r.tell(actor, "Nie rozumiem. Wpisz «pomoc», żeby zobaczyć polecenia.")
		}
	}
}

// explore carries out the commands that mean something else in an
// adventure, reporting whether the command was one of them.
func (r *Realm) explore(actor *player, command Command) bool {
	switch command.Verb {
	case VerbLook:
		r.examine(actor, command.Arg)
	case VerbUse:
		r.handle(actor, world.StepUse, command.Arg)
	case VerbOpen:
		r.handle(actor, world.StepOpen, command.Arg)
	case VerbGive:
		r.hand(actor, command.Arg)
	case VerbVisit:
		r.chat(actor, command.Arg)
	case VerbWait:
		r.wait(actor)
	case VerbFollow:
		r.shadow(actor, command.Arg)
	case VerbAccuse:
		r.accuse(actor, command.Arg)
	case VerbCode:
		if !r.act(actor, world.StepCode, "", command.Arg) {
			r.tell(actor, "Nic się nie dzieje.")
		}
	case VerbAttack, VerbCast, VerbHeal, VerbFlee, VerbRest, VerbHire, VerbDismiss, VerbTakeBack, VerbAuto, VerbQuest, VerbAccept:
		r.tell(actor, "W tej przygodzie nie ma walki ani drużyny. Wpisz «pomoc», żeby zobaczyć polecenia.")
	case VerbUnknown:
		if command.Arg != "" && r.act(actor, world.StepUse, "", command.Arg) {
			return true
		}

		// A bare name is somebody to talk to.
		if _, ok := r.personAt(actor, command.Arg); ok && command.Arg != "" {
			r.chat(actor, command.Arg)

			return true
		}

		if r.act(actor, world.StepCode, "", command.Arg) {
			return true
		}

		r.tell(actor, "Nie rozumiem. Wpisz «pomoc», żeby zobaczyć polecenia.")
	default:
		return false
	}

	return true
}

// calm carries out the commands that are only possible outside a fight.
func (r *Realm) calm(actor *player, command Command) {
	switch command.Verb {
	case VerbTake:
		r.take(actor, command.Arg)
	case VerbDrop:
		r.drop(actor, command.Arg)
	case VerbRest:
		r.rest(actor)
	case VerbVisit:
		r.visit(actor, command.Arg, true)
	case VerbHire:
		r.hire(actor, command.Arg)
	case VerbDismiss:
		r.dismiss(actor, command.Arg)
	}
}

// Use uses the item at the given bag position. In a fight the bag is that
// of the unit whose turn it is, and only potions can be used.
func (r *Realm) Use(name string, index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	user := r.active(name)
	if user == nil {
		return
	}

	if r.fightOf(user) != nil {
		r.useInFight(user, index)

		return
	}

	if index >= 0 && index < len(user.bag) {
		r.useAt(user, index)
	}
}

// useInFight drinks the potion at a bag position of the unit whose turn it is.
func (r *Realm) useInFight(leader *player, index int) {
	current, who := r.turn(leader)
	if current == nil {
		return
	}

	bag := *who.pack()
	if index < 0 || index >= len(bag) {
		return
	}

	if bag[index].Kind != world.KindPotion {
		r.tell(leader, "W walce nie ma na to czasu.")

		return
	}

	r.drink(current, who, index)
	r.next(current)
}

// Rest heals a player who is in a settlement.
func (r *Realm) Rest(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	if resting := r.active(name); resting != nil {
		r.rest(resting)
	}
}

func (r *Realm) look(viewer *player) {
	tile := r.world.At(viewer.pos)
	place := r.placeAt(viewer.pos)
	r.tell(viewer, place.Name+". "+place.Description)

	if r.adventure() {
		if people := r.peopleAt(viewer.pos, viewer); len(people) > 0 {
			r.tell(viewer, "Jest tu: "+join(personNames(people))+".")
		}

		if left := r.hoursLeft(viewer); left >= 0 {
			r.tell(viewer, fmt.Sprintf("Dzień %d, %s. Zostało %d godzin.", viewer.day, r.hourName(viewer), left))
		} else if r.investigation() {
			r.tell(viewer, fmt.Sprintf("Dzień %d, %s.", viewer.day, r.hourName(viewer)))
		}

		r.tell(viewer, "Wyjścia: "+join(r.exits(viewer))+".")

		if lying := r.ground[r.world.Index(viewer.pos)]; len(lying) > 0 {
			r.tell(viewer, "Widzisz tu: "+join(itemNames(lying))+".")
		}

		return
	}

	if tile.Town >= 0 {
		r.tell(viewer, "Możesz tu odwiedzić: "+join(r.buildings())+". Odpoczynek jest za darmo.")

		if len(r.world.Companions) > 0 {
			r.tell(viewer, r.innLine(tile.Town))
		}

		r.tell(viewer, r.standingLine(viewer, tile.Town))
	}

	if r.caravanHere(viewer.pos) {
		r.tell(viewer, caravanLine())
	}

	if len(viewer.party) > 0 {
		r.tell(viewer, "Twoja drużyna: "+join(r.partyNames(viewer))+".")
	}

	if tile.Site >= 0 {
		r.tell(viewer, r.world.Sites[tile.Site].Name+". "+r.world.Sites[tile.Site].Description)
	}

	var exits []string

	for _, step := range []world.Point{north, east, south, west} {
		next := world.Point{X: viewer.pos.X + step.X, Y: viewer.pos.Y + step.Y}
		if !r.world.In(next) {
			continue
		}

		exit := directionNames[step] + ": " + r.placeAt(next).Name
		if r.alive(next) > 0 {
			exit += " (wrogowie)"
		}

		exits = append(exits, exit)
	}

	r.tell(viewer, "Drogi: "+join(exits)+".")

	if lying := r.ground[r.world.Index(viewer.pos)]; len(lying) > 0 {
		r.tell(viewer, "Na ziemi leży: "+join(itemNames(lying))+".")
	}

	var company []string

	for _, name := range r.order {
		if other := r.players[name]; other != viewer && other.connected && other.pos == viewer.pos {
			company = append(company, name)
		}
	}

	if len(company) > 0 {
		r.tell(viewer, "Są tu z tobą: "+join(company)+".")
	}
}

func (r *Realm) take(taker *player, what string) {
	index := r.world.Index(taker.pos)
	lying := r.ground[index]

	switch {
	case len(lying) == 0:
		r.tell(taker, "Nic tu nie leży.")
	case what == "" || Matches(everything, what):
		var kept, taken []world.ItemSpec
		for _, item := range lying {
			if item.Fixed {
				kept = append(kept, item)
			} else {
				taken = append(taken, item)
			}
		}

		if len(taken) == 0 {
			r.tell(taker, "Nie ma tu nic do zabrania.")

			return
		}

		taker.bag = append(taker.bag, taken...)
		r.ground[index] = kept
		r.tell(taker, "Podnosisz: "+join(itemNames(taken))+".")
	default:
		found := find(lying, what)
		if found < 0 {
			r.tell(taker, "Nie widzisz tu niczego takiego.")

			return
		}

		if lying[found].Fixed {
			r.tell(taker, lying[found].Name+": tego nie da się zabrać.")

			return
		}

		taker.bag = append(taker.bag, lying[found])
		r.tell(taker, "Podnosisz: "+lying[found].Name+".")
		r.ground[index] = remove(lying, found)
	}
}

func (r *Realm) drop(owner *player, what string) {
	found := find(owner.bag, what)
	if found < 0 {
		r.tell(owner, "Nie masz niczego takiego w plecaku.")

		return
	}

	index := r.world.Index(owner.pos)
	r.ground[index] = append(r.ground[index], owner.bag[found])
	r.tell(owner, "Odkładasz na ziemię: "+owner.bag[found].Name+".")
	owner.bag = remove(owner.bag, found)
}

func (r *Realm) use(user *player, what string) {
	if current := r.fightOf(user); current != nil {
		_, who := r.turn(user)
		if who == nil {
			return
		}

		found := find(*who.pack(), what)
		if found < 0 {
			r.tell(user, who.label()+" nie ma niczego takiego.")

			return
		}

		r.useInFight(user, found)

		return
	}

	found := find(user.bag, what)
	if found < 0 {
		r.tell(user, "Nie masz niczego takiego w plecaku.")

		return
	}

	r.useAt(user, found)
}

// useAt equips, learns or drinks the item at a bag position, outside a fight.
func (r *Realm) useAt(user *player, index int) {
	item := user.bag[index]

	if item.Kind != world.KindPotion {
		r.equip(user, index)

		return
	}

	stat, limit, label := &user.hp, user.maxHP(), "zdrowia"
	if item.Effect == world.EffectMana {
		stat, limit, label = &user.mana, user.maxMana(), "mocy"
	}

	if *stat >= limit {
		r.tell(user, "Teraz ci to niepotrzebne.")

		return
	}

	user.bag = remove(user.bag, index)
	gained := min(item.Power(), limit-*stat)
	*stat += gained
	r.tell(user, fmt.Sprintf("%s używa: %s (+%d %s).", user.name, item.Name, gained, label))
}

// equip puts on a weapon, armor or charm, or learns a spell.
func (r *Realm) equip(owner *player, index int) {
	item := owner.bag[index]

	if item.Kind == world.KindSpell {
		if owner.knows(item) {
			r.tell(owner, "Już to umiesz.")

			return
		}

		owner.bag = remove(owner.bag, index)
		owner.spells = append(owner.spells, item)
		r.tell(owner, fmt.Sprintf("Uczysz się: %s. %s Rzucasz poleceniem «rzuć».", item.Name, SpellEffect(item)))

		return
	}

	slot := equipSlot(&owner.weapon, &owner.armor, &owner.charm, item.Kind)
	owner.bag = remove(owner.bag, index)
	if *slot != nil {
		owner.bag = append(owner.bag, **slot)
	}

	*slot = &item
	owner.hp, owner.mana = min(owner.hp, owner.maxHP()), min(owner.mana, owner.maxMana())
	r.tell(owner, "Zakładasz: "+item.Name+".")
}

func (r *Realm) rest(resting *player) {
	if r.world.At(resting.pos).Town < 0 {
		r.tell(resting, "Odpocząć można tylko w osadzie.")

		return
	}

	resting.hp, resting.mana = resting.maxHP(), resting.maxMana()
	r.cure(resting)
	r.restParty(resting)
	r.tell(resting, "Odpoczywasz do rana. Zdrowie i moc wracają do pełna, towarzysze też dochodzą do siebie.")
	r.newDay(resting)
}

func (r *Realm) say(speaker *player, text string) {
	if text == "" {
		r.tell(speaker, "Co chcesz powiedzieć?")

		return
	}

	for _, name := range r.order {
		if listener := r.players[name]; listener.connected {
			r.tell(listener, speaker.name+": «"+text+"»")
		}
	}
}

// cast uses the power a player named: in a fight, among those of the unit
// whose turn it is; outside, among the hero's own.
func (r *Realm) cast(caster *player, command Command) {
	current := r.fightOf(caster)
	if current == nil {
		if power, ok := r.spell(caster, caster.spells, command.Arg); ok {
			r.castOutside(caster, power)
		}

		return
	}

	_, who := r.turn(caster)
	if who == nil {
		return
	}

	if power, ok := r.spell(caster, who.powers(), command.Arg); ok && r.castPower(current, who, power, command.Target, command.Index) {
		r.next(current)
	}
}

// castOutside uses a healing power on the hero themselves, between fights.
func (r *Realm) castOutside(caster *player, power world.ItemSpec) {
	switch {
	case power.Effect != world.EffectHeal:
		r.tell(caster, "Nie ma tu z kim walczyć.")
	case caster.hp >= caster.maxHP():
		r.tell(caster, "Teraz ci to niepotrzebne.")
	case caster.mana < power.Cost():
		r.tell(caster, fmt.Sprintf("Za mało mocy: %s wymaga %d, masz %d.", power.Name, power.Cost(), caster.mana))
	default:
		caster.mana -= power.Cost()
		caster.hp = min(caster.maxHP(), caster.hp+power.Power())
		r.tell(caster, fmt.Sprintf("%s rzuca: %s. Rany się zamykają.", caster.name, power.Name))
	}
}

// castPower uses a power of a unit at a monster picked by name or
// position. It reports whether the power was cast; the caller ends the
// turn.
func (r *Realm) castPower(current *fight, who unit, power world.ItemSpec, named string, index int) bool {
	cost := power.Cost()
	if isInnate(power) {
		cost = 0
	}

	leader := who.leader()
	if who.energy() < cost {
		r.tell(leader, fmt.Sprintf("Za mało mocy: %s wymaga %d, %s ma %d.", power.Name, cost, who.label(), who.energy()))

		return false
	}

	switch power.Effect {
	case world.EffectBolt:
		foe := r.target(leader, current, named, index)
		if foe == nil {
			return false
		}

		if r.dnd() && distance(r.cellOf(current, who), r.cellOf(current, foe)) > boltRange {
			r.tell(leader, fmt.Sprintf("%s jest poza zasięgiem mocy (%d pól).", foe.name, boltRange))

			return false
		}

		damage := power.Power() + r.rng.Intn(power.Power()/4+1)

		if r.dnd() {
			// A spell attack: the d20 plus proficiency and intellect.
			hit, crit, roll := r.attackRoll(proficiency(levelOf(who))+Modifier(who.scores()[Intelligence]), monsterClass(foe))

			switch {
			case !hit:
				r.tellFight(current, fmt.Sprintf("%s rzuca: %s. Pudło (k20: %d przeciw KP %d).", who.label(), power.Name, roll, monsterClass(foe)))
				who.setEnergy(who.energy() - cost)

				return true
			case crit:
				damage *= 2
			}
		}

		r.hit(current, foe, damage, BlowBolt, who.label(), fmt.Sprintf("%s rzuca: %s. ", who.label(), power.Name))
	case world.EffectBlast:
		r.tellFight(current, fmt.Sprintf("%s rzuca: %s.", who.label(), power.Name))

		caught := r.foes(current.pos)

		if r.dnd() {
			// The blast bursts around a target; the quick take half.
			centre := r.target(leader, current, named, index)
			if centre == nil {
				return false
			}

			caught = nil
			for _, foe := range r.foes(current.pos) {
				if distance(r.cellOf(current, centre), r.cellOf(current, foe)) <= blastRadius {
					caught = append(caught, foe)
				}
			}
		}

		difficulty := 8 + proficiency(levelOf(who)) + Modifier(who.scores()[Intelligence])

		for _, foe := range caught {
			damage := power.Power()
			if r.dnd() && r.save(foe.tier, difficulty) {
				r.hit(current, foe, max(1, damage/2), BlowBlast, who.label(), "Rzut obronny udany, ")

				continue
			}

			r.hit(current, foe, damage, BlowBlast, who.label(), "")
		}
	default:
		// Wisdom strengthens healing.
		healed := max(1, power.Power()+Modifier(who.scores()[Wisdom]))

		for _, name := range current.members {
			for _, ally := range r.units(r.players[name]) {
				ally.setHealth(min(ally.maxHealth(), ally.health()+healed))
			}
		}

		r.blowBy(current.pos, -1, healed, BlowHeal, who.label(), "")
		r.tellFight(current, fmt.Sprintf("%s rzuca: %s. Drużyna odzyskuje do %d zdrowia.", who.label(), power.Name, healed))
	}

	who.setEnergy(who.energy() - cost)

	return true
}

// spell picks the power a player means among the given ones, telling them
// when it is unclear.
func (r *Realm) spell(caster *player, known []world.ItemSpec, what string) (world.ItemSpec, bool) {
	switch found := find(known, what); {
	case len(known) == 0:
		r.tell(caster, "Nie znasz jeszcze żadnych mocy. Naucz się ich z przedmiotów poleceniem «użyj».")
	case found >= 0:
		return known[found], true
	case what == "" && len(known) == 1:
		return known[0], true
	default:
		r.tell(caster, "Co chcesz rzucić? Do wyboru: "+join(itemNames(known))+".")
	}

	return world.ItemSpec{}, false
}

// find returns the position of the first item matching what the player
// typed, or -1.
func find(items []world.ItemSpec, what string) int {
	for index, item := range items {
		if Matches(item.Name, what) {
			return index
		}
	}

	return -1
}

func remove(items []world.ItemSpec, index int) []world.ItemSpec {
	return append(append([]world.ItemSpec(nil), items[:index]...), items[index+1:]...)
}
