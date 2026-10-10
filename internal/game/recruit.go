package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Companions for hire sit in the inn and can be talked to like keepers:
// the talk is a haggle over one ware, their service, and ends with them
// joining the party. A companion already in the party talks for free.

const (
	// hireFloorPercent is how far under the asking price a companion can
	// be talked; wageFloorPercent the same for a legend's daily wage.
	hireFloorPercent = 70
	wageFloorPercent = 80

	hirePrefix = "najem:"
	wagePrefix = "zold:"
)

// mateTalk is the index a talk with a companion is kept under, past the
// shops and the caravan.
func (r *Realm) mateTalk(kind int) int {
	return len(r.world.Shops) + 1 + kind
}

// mateOf tells which companion a talk index means, if any.
func (r *Realm) mateOf(index int) (int, bool) {
	kind := index - len(r.world.Shops) - 1
	if index < 0 || kind < 0 || kind >= len(r.world.Companions) {
		return 0, false
	}

	return kind, true
}

// mateSpec is a companion as the keeper of a talk: at the inn, with their
// own character.
func (r *Realm) mateSpec(kind int) world.ShopSpec {
	spec := r.world.Companions[kind]
	greeting := "Siadaj. Jeśli szukasz kogoś do roboty, to trafiłeś; o cenie pogadamy."

	switch {
	case r.hired[kind] != "":
		greeting = "Słucham. Co robimy dalej?"
	case spec.Legend:
		greeting = "Siadaj. Ze mną nie chodzi się za darmo: żołd co dzień o świcie."
	}

	return world.ShopSpec{
		Name:     "Karczma",
		Keeper:   spec.Name,
		Persona:  fmt.Sprintf("%s, %s. %s Mówisz jak najemnik: o pieniądzach wprost, o walce ze znawstwem, o swoich warunkach bez wstydu.", spec.Name, roleName(spec.Role), spec.Description),
		Greeting: greeting,
	}
}

// WageName and HireName are the wares a companion for hire talks over: a
// legend's daily wage, or the price of joining.
const (
	WageName = "Żołd dzienny"
	HireName = "Umowa najmu"
)

// hireWare is a companion's service as a thing to haggle over: the price
// of joining, or for a legend the daily wage.
func hireWare(kind int, spec world.CompanionSpec) Item {
	if spec.Legend {
		return Item{
			ID: fmt.Sprintf("%s%d", wagePrefix, kind), Name: WageName,
			Description: fmt.Sprintf("%s idzie z tobą za dzienny żołd, płatny o świcie.", spec.Name),
			Aliases:     []string{"zold", "dzien", "umow", "najem", "sluzb"},
			ListPrice:   spec.Wage(), MinPrice: max(1, spec.Wage()*wageFloorPercent/100),
		}
	}

	return Item{
		ID: fmt.Sprintf("%s%d", hirePrefix, kind), Name: HireName,
		Description: fmt.Sprintf("%s dołącza do drużyny jako %s.", spec.Name, roleName(spec.Role)),
		Aliases:     []string{"umow", "najem", "sluzb", "wynaj"},
		ListPrice:   spec.Stats().Price, MinPrice: max(1, spec.Stats().Price*hireFloorPercent/100),
	}
}

// hireOf tells which companion a ware is the service of, if any.
func hireOf(id string) (int, bool) {
	var kind int

	for _, prefix := range []string{hirePrefix, wagePrefix} {
		if _, err := fmt.Sscanf(strings.TrimPrefix(id, prefix), "%d", &kind); err == nil && strings.HasPrefix(id, prefix) {
			return kind, true
		}
	}

	return 0, false
}

// mateMerchant is a companion as a talking partner, with their service on
// the table while they are free and the party has room.
func (r *Realm) mateMerchant(leader *player, kind int) Merchant {
	spec := r.mateSpec(kind)
	merchant := Merchant{Name: spec.Keeper, Persona: spec.Persona, Greeting: spec.Greeting, Patience: merchantPatience, Currency: r.world.Currency}

	if r.recruitable(leader, kind) {
		merchant.Items = []Item{hireWare(kind, r.world.Companions[kind])}
	}

	return merchant
}

// recruitable reports whether a hero could take a companion on right now.
func (r *Realm) recruitable(leader *player, kind int) bool {
	return r.hired[kind] == "" && len(leader.party) < MaxParty
}

// findMate finds the companion a hero names among those waiting in the inn
// here and those already following them.
func (r *Realm) findMate(leader *player, what string) (int, bool) {
	if what == "" {
		return 0, false
	}

	if town := r.townOf(leader); town >= 0 {
		for _, kind := range r.forHire(town) {
			if Matches(r.world.Companions[kind].Name, what) {
				return kind, true
			}
		}
	}

	for _, member := range leader.party {
		if Matches(member.spec().Name, what) {
			return member.kind, true
		}
	}

	return 0, false
}

// talkToMate sits a hero down with a companion: a haggle over their
// service when they are for hire, a plain talk when they already follow.
// It returns the talk's index, or -1 when the companion will not talk.
func (r *Realm) talkToMate(leader *player, kind int) int {
	spec := r.world.Companions[kind]
	town := r.townOf(leader)

	if r.hired[kind] == "" {
		switch standing := r.standingOf(leader, town); {
		case standing < 0:
			r.tell(leader, fmt.Sprintf("%s: «Z kimś, kogo tu mają za %s? Nie.» Popraw sławę w osadzie.", spec.Name, standingLabel(standing)))

			return -1
		case spec.Legend && standing < legendNeeds:
			r.tell(leader, fmt.Sprintf("%s: «Legendy nie chodzą z byle kim. Zrób coś dla tej osady, to pogadamy.»", spec.Name))

			return -1
		}
	}

	if leader.talks == nil {
		leader.talks = map[int]*talk{}
	}

	index := r.mateTalk(kind)
	ware := hireWare(kind, spec)
	current := leader.talks[index]

	if current == nil {
		r.talks++
		merchant := r.mateMerchant(leader, kind)
		current = &talk{session: NewSession(merchant, leader.gold), serial: r.talks}
		current.session.Favor(Modifier(leader.abilities[Charisma]))
		current.session.Remember(false, merchant.Greeting)
		leader.talks[index] = current
	} else {
		// The service goes back on the table, or comes off it, as the
		// party has changed since the last talk.
		switch onSale := current.session.resolve(ware.ID) != nil; {
		case r.recruitable(leader, kind) && !onSale:
			current.session.Stock(ware)
		case !r.recruitable(leader, kind) && onSale:
			current.session.Unstock(ware.ID)
		}
	}

	// With one thing to talk about, a bare sum is an offer for it.
	if current.session.resolve(ware.ID) != nil && current.session.focus == "" {
		current.session.focus = ware.ID
	}

	leader.shop, leader.inn = index, false

	switch {
	case r.hired[kind] != "":
		r.tell(leader, fmt.Sprintf("Rozmawiasz z: %s, twój towarzysz. %s", spec.Name, spec.Description))
	case len(leader.party) >= MaxParty:
		r.tell(leader, fmt.Sprintf("Przysiadasz się do: %s. %s Drużyna jest pełna: pogadać możesz, ale najpierw kogoś odpraw.", spec.Name, spec.Description))
	case spec.Legend:
		r.tell(leader, fmt.Sprintf("Przysiadasz się do: %s. %s Żołd: %d %s dziennie; targuj się albo powiedz «zgoda».", spec.Name, spec.Description, ware.ListPrice, r.world.Currency))
	default:
		r.tell(leader, fmt.Sprintf("Przysiadasz się do: %s. %s Cena: %d %s; targuj się albo powiedz «zgoda».", spec.Name, spec.Description, ware.ListPrice, r.world.Currency))
	}

	return index
}

// recruit takes a companion into a hero's party, for the given daily wage
// (zero for all but legends), and says so.
func (r *Realm) recruit(leader *player, kind, wage int) {
	spec := r.world.Companions[kind]
	member := r.newCompanion(kind, leader)
	member.wage = wage

	leader.party = append(leader.party, member)
	r.hired[kind] = leader.name
	r.tell(leader, fmt.Sprintf("%s dołącza do drużyny. %s Rzeczy dasz poleceniem «daj X %s», kartę zobaczysz pod F%d.",
		spec.Name, spec.Description, spec.Name, len(leader.party)+1))

	if wage > 0 {
		r.tell(leader, fmt.Sprintf("Umowa: %d %s za każdy dzień, płatne o świcie. Gdy zabraknie, %s odchodzi.", wage, r.world.Currency, spec.Name))
	}

	r.tellOthers(leader, fmt.Sprintf("%s najmuje towarzysza: %s.", leader.name, spec.Name))
}

// settle carries out a deal struck in a talk with a companion: the sum
// agreed buys their service, or for a legend sets their wage and costs
// nothing up front. It reports whether the ware was a companion's service.
func (r *Realm) settle(customer *player, session *Session, decision Decision) bool {
	kind, ok := hireOf(decision.Item.ID)
	if !ok {
		return false
	}

	spec := r.world.Companions[kind]
	customer.gold = session.Gold

	switch {
	case !r.recruitable(customer, kind):
		customer.gold += decision.Price
		r.tell(customer, fmt.Sprintf("%s: «Nie masz dla mnie miejsca.» Pieniądze wracają do sakwy.", spec.Name))
	case spec.Legend:
		customer.gold += decision.Price
		r.recruit(customer, kind, decision.Price)
	default:
		r.recruit(customer, kind, 0)
	}

	return true
}

// mateRole opens the narrator's instructions for a companion instead of a
// shopkeeper.
func (r *Realm) mateRole(kind int) string {
	if r.hired[kind] != "" {
		return RoleCompanion
	}

	return RoleMercenary
}

// mateLore words what a companion knows and wants, for the model.
func (r *Realm) mateLore(leader *player, kind int, wares []Ware) string {
	built := r.world
	spec := built.Companions[kind]
	here := leader.pos

	var lore strings.Builder

	fmt.Fprintf(&lore, "Nie prowadzisz sklepu. Jesteś: %s, %s. %s\n", spec.Name, roleName(spec.Role), spec.Description)

	switch {
	case len(wares) > 0 && spec.Legend:
		fmt.Fprintf(&lore, "Twoja służba kosztuje: «%s» %d %s za każdy dzień, płatne o świcie; nic z góry. Poniżej tej kwoty nie zejdziesz bez powodu.\n", wares[0].Name, wares[0].Price, built.Currency)
	case len(wares) > 0:
		fmt.Fprintf(&lore, "Twoja służba kosztuje: «%s» %d %s, płatne raz, przy dołączeniu. Poniżej tej kwoty nie zejdziesz bez powodu.\n", wares[0].Name, wares[0].Price, built.Currency)
	case r.hired[kind] == leader.name:
		lore.WriteString("Jesteś już w drużynie bohatera: nie targujesz się, tylko radzisz, narzekasz albo opowiadasz.\n")
	default:
		lore.WriteString("Bohater ma pełną drużynę: nie możesz dołączyć, póki kogoś nie odprawi. Możecie pogadać.\n")
	}

	fmt.Fprintf(&lore, "Twoje rzemiosło: %s.\n", craft(spec))
	fmt.Fprintf(&lore, "Świat: %s. %s\n", built.Name, built.Intro)
	fmt.Fprintf(&lore, "Siedzisz w karczmie osady: %s. %s\n", built.PlaceAt(here).Name, built.PlaceAt(here).Description)
	fmt.Fprintf(&lore, "Dzień %d. %s\n", r.day, r.sky())

	var dangers []string

	for index, place := range r.places {
		if place.Danger == 0 || len(dangers) >= referrals {
			continue
		}

		if point, ok := r.placeTile(index); ok {
			dangers = append(dangers, fmt.Sprintf("%s (%s, zagrożenie %d)", place.Name, bearing(here, point), place.Danger))
		}
	}

	if len(dangers) > 0 {
		fmt.Fprintf(&lore, "Strony, w których bywałeś: %s. Najgroźniejsze (zagrożenie %d) omijasz, póki nie zmężniejesz.\n", join(dangers), world.MaxTier)
	}

	monsters := make([]string, 0, len(built.Monsters))
	for _, monster := range built.Monsters {
		monsters = append(monsters, monster.Name)
	}

	if len(monsters) > 0 {
		fmt.Fprintf(&lore, "Potwory, z którymi miałeś do czynienia: %s.\n", join(monsters))
	}

	fmt.Fprintf(&lore, "Największe zagrożenie: %s. %s Jego siedziba: %s.\n", built.Boss.Name, built.Boss.Description, bearing(here, built.Lair))

	if member := r.mateByKind(leader, kind); member != nil {
		fmt.Fprintf(&lore, "Masz poziom %d, zdrowie %d/%d. ", member.level, member.hp, member.maxHealth())
	}

	fmt.Fprintf(&lore, "Twój rozmówca: %s, poziom %d, %s. Broń: %s. Pancerz: %s.",
		leader.name, leader.level, condition(leader), orNothing(leader.weapon), orNothing(leader.armor))

	if town := r.townOf(leader); town >= 0 {
		score := r.standingOf(leader, town)
		fmt.Fprintf(&lore, " Opinia o nim w osadzie: %s (%+d); traktuj go odpowiednio.", standingLabel(score), score)
	}

	return lore.String()
}

// craft words what a companion of a role does in a fight.
func craft(spec world.CompanionSpec) string {
	switch {
	case spec.Legend:
		return "legenda; walczysz, leczysz drużynę i rażesz wszystkich wrogów naraz"
	case spec.Role == world.RoleHealer:
		return "opatrujesz rannych w drużynie w środku walki"
	case spec.Role == world.RoleMage:
		return "rażesz podmuchem wszystkich wrogów naraz"
	}

	return "walczysz wręcz w pierwszym szeregu"
}

// placeTile finds a tile of a place, to tell where it lies from here.
func (r *Realm) placeTile(place int) (world.Point, bool) {
	for y := range world.Height {
		for x := range world.Width {
			if point := (world.Point{X: x, Y: y}); r.world.At(point).Place == place {
				return point, true
			}
		}
	}

	return world.Point{}, false
}

// mateByKind finds a companion of a kind in a hero's party.
func (r *Realm) mateByKind(leader *player, kind int) *companion {
	for _, member := range leader.party {
		if member.kind == kind {
			return member
		}
	}

	return nil
}
