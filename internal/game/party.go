package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	// MaxParty is how many companions a hero may have at once.
	MaxParty = 2

	companionBaseMana   = 6
	companionHPPerLevel = 4
	// Companions grow a little slower than heroes.
	companionLevelsPerMight = 2
	legendPace              = 2
)

// rolePace is how quick a companion of each role is.
var rolePace = map[world.Role]int{world.RoleFighter: 5, world.RoleHealer: 4, world.RoleMage: 6}

// roleAvatars are the portraits companions get unless their world names one.
var roleAvatars = map[world.Role]string{world.RoleFighter: "wojownik", world.RoleHealer: "lowczyni", world.RoleMage: "czarodziej"}

// companion is a hero for hire who follows a player and fights at their
// side, with gear and powers of their own. At zero health they are knocked
// out, not dead: rest brings them back.
type companion struct {
	kind      int
	owner     *player
	abilities Abilities
	level     int
	xp        int
	hp        int
	mana      int

	weapon *world.ItemSpec
	armor  *world.ItemSpec
	charm  *world.ItemSpec
	spells []world.ItemSpec
	bag    []world.ItemSpec
	// auto lets the companion pick their own actions in a fight.
	auto bool
	// poison is how many more rounds venom works in the companion.
	poison int
	// order is what the companion was told to do when their turn comes.
	order *Command
	// warned is set once the companion has spoken up against a place.
	warned bool
	// wage is what the companion takes each dawn: a legend's, as agreed
	// at the inn; zero for the rest.
	wage int
}

func (r *Realm) newCompanion(kind int, owner *player) *companion {
	member := &companion{kind: kind, owner: owner, level: 1}
	member.abilities = roleAbilities(r.rng, member.spec())
	member.hp, member.mana = member.maxHealth(), member.maxEnergy()

	return member
}

func (c *companion) spec() world.CompanionSpec {
	return c.owner.realmSpec().Companions[c.kind]
}

// realmSpec is the world the player lives in; companions reach their
// specification through it.
func (p *player) realmSpec() *world.World { return p.world }

func (c *companion) label() string         { return c.spec().Name }
func (c *companion) leader() *player       { return c.owner }
func (c *companion) health() int           { return c.hp }
func (c *companion) setHealth(value int)   { c.hp = value }
func (c *companion) energy() int           { return c.mana }
func (c *companion) setEnergy(value int)   { c.mana = value }
func (c *companion) venom() int            { return c.poison }
func (c *companion) setVenom(rounds int)   { c.poison = rounds }
func (c *companion) arms() *world.ItemSpec { return c.weapon }
func (c *companion) pack() *[]world.ItemSpec {
	return &c.bag
}

func (c *companion) scores() Abilities {
	return c.abilities
}

func (c *companion) maxHealth() int {
	return max(1, c.spec().Stats().HP+(c.level-1)*companionHPPerLevel+Modifier(c.abilities[Constitution])*c.level+charmBonus(c.charm, world.EffectVigor))
}

func (c *companion) maxEnergy() int {
	return max(0, companionBaseMana+c.level+Modifier(c.abilities[Intelligence])*2+charmBonus(c.charm, world.EffectFocus))
}

func (c *companion) might() int {
	might := c.spec().Stats().Attack + (c.level-1)/companionLevelsPerMight + Modifier(c.abilities[Strength]) + charmBonus(c.charm, world.EffectPower)
	if c.weapon != nil {
		might += c.weapon.Power()
	}

	return might
}

func (c *companion) guard() int {
	guard := charmBonus(c.charm, world.EffectWard)
	if c.armor != nil {
		guard += c.armor.Power()
	}

	return guard
}

func (c *companion) pace() int {
	pace := rolePace[c.spec().Role] + c.spec().Tier - 1 + Modifier(c.abilities[Dexterity])
	if c.spec().Legend {
		pace += legendPace
	}

	if c.armor != nil {
		pace -= c.armor.Tier
	}

	return max(pace, 1)
}

// powers are the companion's inborn ability, if their role has one, and
// the spells they were taught. Inborn abilities cost no power.
func (c *companion) powers() []world.ItemSpec {
	spec := c.spec()
	heal := world.ItemSpec{ID: "innate-heal", Name: "Opatrunek", Description: "Wrodzona umiejętność: leczy drużynę.", Kind: world.KindSpell, Effect: world.EffectHeal, Tier: spec.Tier}
	blast := world.ItemSpec{ID: "innate-blast", Name: "Podmuch", Description: "Wrodzona umiejętność: rani wszystkich wrogów.", Kind: world.KindSpell, Effect: world.EffectBlast, Tier: spec.Tier}

	var innate []world.ItemSpec

	switch {
	case spec.Legend:
		innate = []world.ItemSpec{blast, heal}
	case spec.Role == world.RoleHealer:
		innate = []world.ItemSpec{heal}
	case spec.Role == world.RoleMage:
		innate = []world.ItemSpec{blast}
	}

	return append(innate, c.spells...)
}

func (c *companion) nextLevel() int {
	return c.level * xpPerLevel
}

func (c *companion) avatar() string {
	return roleAvatars[c.spec().Role]
}

// isInnate reports whether a power is one a companion was born with, and
// so costs nothing to use.
func isInnate(power world.ItemSpec) bool {
	return power.ID == "innate-heal" || power.ID == "innate-blast"
}

// innOf tells which settlement a companion waits in: they are spread
// across the settlements in turn.
func (r *Realm) innOf(kind int) int {
	return kind % len(r.world.Towns)
}

// forHire lists the companions waiting in the settlement of a tile.
func (r *Realm) forHire(town int) []int {
	var waiting []int

	for kind := range r.world.Companions {
		if r.innOf(kind) == town && r.hired[kind] == "" {
			waiting = append(waiting, kind)
		}
	}

	return waiting
}

// innLine words who is waiting for hire in a settlement.
func (r *Realm) innLine(town int) string {
	waiting := r.forHire(town)
	if len(waiting) == 0 {
		return "W karczmie nie ma nikogo do wynajęcia."
	}

	names := make([]string, 0, len(waiting))
	for _, kind := range waiting {
		spec := r.world.Companions[kind]
		if spec.Legend {
			names = append(names, fmt.Sprintf("%s (legenda, żołd %d %s dziennie)", spec.Name, spec.Wage(), r.world.Currency))
		} else {
			names = append(names, fmt.Sprintf("%s (%s, %d %s)", spec.Name, roleName(spec.Role), spec.Stats().Price, r.world.Currency))
		}
	}

	return "W karczmie czeka do wynajęcia: " + join(names) + ". Wynajmiesz poleceniem «najmij», a potargujesz się przez «rozmawiaj z X»."
}

func (r *Realm) hire(leader *player, what string) {
	town := r.world.At(leader.pos).Town
	if town < 0 {
		r.tell(leader, "Towarzyszy najmuje się w karczmach osad.")

		return
	}

	waiting := r.forHire(town)
	found := -1

	for _, kind := range waiting {
		if Matches(r.world.Companions[kind].Name, what) || (what == "" && len(waiting) == 1) {
			found = kind

			break
		}
	}

	switch {
	case found < 0:
		r.tell(leader, r.innLine(town))

		return
	case len(leader.party) >= MaxParty:
		r.tell(leader, fmt.Sprintf("Drużyna jest pełna: najwyżej %d towarzyszy. Kogoś musisz odprawić.", MaxParty))

		return
	}

	spec := r.world.Companions[found]

	// Companions have opinions: nobody follows the disliked, and a legend
	// follows only the respected.
	switch standing := r.standingOf(leader, town); {
	case standing < 0:
		r.tell(leader, fmt.Sprintf("%s: «Z kimś, kogo tu mają za %s? Nie.» Popraw sławę w osadzie.", spec.Name, standingLabel(standing)))

		return
	case spec.Legend && standing < legendNeeds:
		r.tell(leader, fmt.Sprintf("%s: «Legendy nie chodzą z byle kim. Zrób coś dla tej osady, to pogadamy.»", spec.Name))

		return
	}

	// Charm knocks a little off the price: a tenth for every point of modifier.
	price := max(0, spec.Stats().Price*(10-Modifier(leader.abilities[Charisma]))/10)

	if leader.gold < max(price, spec.Wage()) {
		r.tell(leader, fmt.Sprintf("%s chce %d %s, a masz %d.", spec.Name, max(price, spec.Wage()), r.world.Currency, leader.gold))

		return
	}

	leader.gold -= price
	r.recruit(leader, found, spec.Wage())
}

// payWages settles a hero's legends at daybreak; whoever cannot be paid
// walks away with nothing of the hero's, and so does whoever will not be
// seen with somebody their home settlement has cast out.
func (r *Realm) payWages(leader *player) {
	for index := 0; index < len(leader.party); index++ {
		member := leader.party[index]
		spec := member.spec()

		if home := r.innOf(member.kind); r.shunned(leader, home) {
			r.release(leader, index)
			r.tell(leader, fmt.Sprintf("%s odchodzi: «Nie będę chodzić z kimś, kogo w osadzie %s mają za wyklętego.» Oddaje twoje rzeczy.", spec.Name, r.townName(home)))
			index--

			continue
		}

		if member.wage == 0 {
			continue
		}

		if leader.gold >= member.wage {
			leader.gold -= member.wage
			r.tell(leader, fmt.Sprintf("%s bierze żołd: %d %s. Zostaje ci %d.", spec.Name, member.wage, r.world.Currency, leader.gold))

			continue
		}

		r.release(leader, index)
		r.tell(leader, fmt.Sprintf("%s nie dostaje żołdu i odchodzi: «Bez zapłaty nie ma usługi.» Oddaje twoje rzeczy.", spec.Name))
		index--
	}
}

// release lets a companion go: the hero's things come back and the inn
// gets its guest back.
func (r *Realm) release(leader *player, index int) {
	member := leader.party[index]

	leader.bag = append(leader.bag, member.bag...)
	for _, slot := range []*world.ItemSpec{member.weapon, member.armor, member.charm} {
		if slot != nil {
			leader.bag = append(leader.bag, *slot)
		}
	}

	leader.party = append(leader.party[:index], leader.party[index+1:]...)
	delete(r.hired, member.kind)
}

func (r *Realm) dismiss(leader *player, what string) {
	for index, member := range leader.party {
		spec := member.spec()
		if !Matches(spec.Name, what) && !(what == "" && len(leader.party) == 1) {
			continue
		}

		r.release(leader, index)
		r.tell(leader, fmt.Sprintf("%s odchodzi i oddaje twoje rzeczy. Wróci do karczmy w osadzie %s.", spec.Name, r.placeAt(r.world.Towns[r.innOf(member.kind)]).Name))

		if member.wage > 0 {
			r.tell(leader, "Żołd przestaje biec.")
		}

		return
	}

	if len(leader.party) == 0 {
		r.tell(leader, "Nie masz towarzyszy.")

		return
	}

	r.tell(leader, "Kogo odprawić? Twoja drużyna: "+join(r.partyNames(leader))+".")
}

func (r *Realm) partyNames(leader *player) []string {
	names := make([]string, 0, len(leader.party))
	for _, member := range leader.party {
		names = append(names, member.spec().Name)
	}

	return names
}

// mate finds the companion a player means by name.
func (r *Realm) mate(leader *player, what string) *companion {
	for _, member := range leader.party {
		if Matches(member.spec().Name, what) {
			return member
		}
	}

	return nil
}

// give hands an item from the hero's bag to a companion: gear is put on at
// once (what it replaces comes back), a spell is learnt, a potion kept.
func (r *Realm) give(leader *player, what string) {
	member, rest := r.splitMate(leader, what)
	if member == nil {
		r.tell(leader, "Komu? Twoja drużyna: "+join(r.partyNames(leader))+". Napisz np. «daj kord "+firstName(leader, r)+"».")

		return
	}

	found := find(leader.bag, rest)
	if found < 0 {
		r.tell(leader, "Nie masz niczego takiego w plecaku.")

		return
	}

	if r.fightOf(leader) != nil {
		r.tell(leader, "W walce nie ma na to czasu.")

		return
	}

	item := leader.bag[found]
	leader.bag = remove(leader.bag, found)
	spec := member.spec()

	switch item.Kind {
	case world.KindPotion:
		member.bag = append(member.bag, item)
		r.tell(leader, fmt.Sprintf("%s chowa: %s.", spec.Name, item.Name))
	case world.KindSpell:
		for _, known := range member.spells {
			if known.ID == item.ID {
				leader.bag = append(leader.bag, item)
				r.tell(leader, spec.Name+" już to umie.")

				return
			}
		}

		member.spells = append(member.spells, item)
		r.tell(leader, fmt.Sprintf("%s uczy się: %s.", spec.Name, item.Name))
	default:
		slot := equipSlot(&member.weapon, &member.armor, &member.charm, item.Kind)
		if *slot != nil {
			leader.bag = append(leader.bag, **slot)
		}

		*slot = &item
		member.hp, member.mana = min(member.hp, member.maxHealth()), min(member.mana, member.maxEnergy())
		r.tell(leader, fmt.Sprintf("%s zakłada: %s.", spec.Name, item.Name))
	}
}

// takeBack takes an item a companion carries or wears back into the
// hero's bag.
func (r *Realm) takeBack(leader *player, what string) {
	member, rest := r.splitMate(leader, what)
	if member == nil {
		r.tell(leader, "Od kogo? Twoja drużyna: "+join(r.partyNames(leader))+".")

		return
	}

	if r.fightOf(leader) != nil {
		r.tell(leader, "W walce nie ma na to czasu.")

		return
	}

	if found := find(member.bag, rest); found >= 0 {
		leader.bag = append(leader.bag, member.bag[found])
		r.tell(leader, fmt.Sprintf("Zabierasz od %s: %s.", member.spec().Name, member.bag[found].Name))
		member.bag = remove(member.bag, found)

		return
	}

	for _, slot := range []**world.ItemSpec{&member.weapon, &member.armor, &member.charm} {
		if *slot != nil && Matches((*slot).Name, rest) {
			leader.bag = append(leader.bag, **slot)
			r.tell(leader, fmt.Sprintf("Zabierasz od %s: %s.", member.spec().Name, (*slot).Name))
			*slot = nil
			member.hp = min(member.hp, member.maxHealth())

			return
		}
	}

	r.tell(leader, member.spec().Name+" nie ma niczego takiego.")
}

// splitMate finds which companion a phrase names and returns the rest of
// the phrase, which names the item.
func (r *Realm) splitMate(leader *player, what string) (*companion, string) {
	words := strings.Fields(Fold(what))

	for _, member := range leader.party {
		nameWords := strings.Fields(Fold(member.spec().Name))
		var rest []string
		named := false

		for _, word := range words {
			matches := false
			for _, part := range nameWords {
				matches = matches || (len(word) >= minQueryWord && strings.HasPrefix(part, Stem(word)))
			}

			if matches && !contains(fillers, word) {
				named = true
			} else if !contains([]string{"od", "dla"}, word) {
				rest = append(rest, word)
			}
		}

		if named {
			return member, strings.Join(rest, " ")
		}
	}

	return nil, what
}

func firstName(leader *player, r *Realm) string {
	if len(leader.party) == 0 {
		return "bognie"
	}

	return strings.ToLower(strings.Fields(leader.party[0].spec().Name)[0])
}

// nerve lets companions speak up when their leader walks them into the
// deadliest country: the first time they warn, the next they turn back.
func (r *Realm) nerve(leader *player, place world.PlaceSpec) {
	if place.Danger < world.MaxTier || r.adventure() {
		return
	}

	for index := 0; index < len(leader.party); index++ {
		member := leader.party[index]
		spec := member.spec()

		if member.level+spec.Tier >= world.MaxTier+1 {
			continue
		}

		if !member.warned {
			member.warned = true
			r.tell(leader, fmt.Sprintf("%s: «To śmiertelne strony, a ja nie jestem gotów. Zawróć, albo idę sam.»", spec.Name))

			continue
		}

		r.release(leader, index)
		r.tell(leader, fmt.Sprintf("%s zawraca do karczmy w osadzie %s i oddaje twoje rzeczy: «Mówiłem.»", spec.Name, r.townName(r.innOf(member.kind))))
		index--
	}
}

// restParty brings a hero's companions back to full health and power.
func (r *Realm) restParty(leader *player) {
	for _, member := range leader.party {
		member.hp, member.mana, member.poison = member.maxHealth(), member.maxEnergy(), 0
	}
}

// forget drops the orders a player's companions still hold.
func (r *Realm) forget(leader *player) {
	for _, member := range leader.party {
		member.order = nil
	}
}

// cure takes the venom out of a player and their companions.
func (r *Realm) cure(leader *player) {
	for _, who := range r.units(leader) {
		who.setVenom(0)
	}
}

// rewardParty gives a hero's companions experience and raises their level
// when it is due.
func (r *Realm) rewardParty(leader *player, xp int) {
	for _, member := range leader.party {
		if member.level >= MaxLevel {
			continue
		}

		member.xp += xp

		for member.level < MaxLevel && member.xp >= member.nextLevel() {
			member.xp -= member.nextLevel()
			member.level++
			member.hp, member.mana = member.maxHealth(), member.maxEnergy()
			r.tell(leader, fmt.Sprintf("%s awansuje na poziom %d.", member.spec().Name, member.level))
		}
	}
}

func roleName(role world.Role) string {
	switch role {
	case world.RoleHealer:
		return "uzdrowiciel"
	case world.RoleMage:
		return "mag"
	}

	return "wojownik"
}
