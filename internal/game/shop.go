package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// firstTownTier caps what the starting settlement sells when the world has
// more settlements: the best gear has to be travelled for.
const firstTownTier = 2

// shopWords are everyday names for the kinds of shops.
var shopWords = map[world.ShopKind]string{
	world.ShopArmory:    "zbrojownia kowal bron pancerz zbroja",
	world.ShopAlchemist: "alchemik zielarka mikstury leki medyk apteka",
	world.ShopMagic:     "mag magia czary zaklecia moce amulety",
}

// visit takes a player into the building they named and starts or resumes
// the talk with its keeper. It returns the shop's index, or -1. When loud,
// the player is told why nothing happened.
func (r *Realm) visit(visitor *player, what string, loud bool) int {
	town := r.world.At(visitor.pos).Town
	caravan := r.caravanHere(visitor.pos)

	if town < 0 && !caravan {
		if loud {
			r.tell(visitor, "Nie ma tu nikogo, z kim dałoby się handlować. Kupcy są w osadach, a karawana na szlaku.")
		}

		return -1
	}

	if r.asleep(visitor) {
		if loud {
			r.tell(visitor, "Głęboka noc: wszyscy śpią. Odpocznij do rana albo wróć za dnia.")
		}

		return -1
	}

	found := -1

	if town >= 0 {
		for index, shop := range r.world.Shops {
			named := Matches(shop.Name, what) || Matches(shop.Keeper, what) || Matches(shopWords[shop.Kind], what)
			if named || (what == "" && len(r.world.Shops) == 1) {
				found = index

				break
			}
		}
	}

	if found < 0 && caravan && namesCaravan(what) {
		found = r.caravanShop()
	}

	// A companion named is somebody to sit down with at the inn.
	if kind, ok := r.findMate(visitor, what); found < 0 && ok {
		return r.talkToMate(visitor, kind)
	}

	if found < 0 {
		if loud {
			r.tell(visitor, "Dokąd chcesz wejść? Są tu: "+join(r.buildings())+".")
		}

		return -1
	}

	if town >= 0 && found != r.caravanShop() && r.shunned(visitor, town) {
		r.tell(visitor, fmt.Sprintf("%s zamyka ci drzwi przed nosem: «Z tobą nie handlujemy.» W osadzie %s jesteś %s.", r.shopSpec(found).Keeper, r.townName(town), standingLabel(r.standingOf(visitor, town))))

		return -1
	}

	if visitor.talks[found] == nil {
		merchant := r.favoured(r.merchant(found, r.tierCap(visitor)), r.standingOf(visitor, town))
		r.talks++

		if visitor.talks == nil {
			visitor.talks = map[int]*talk{}
		}

		visitor.talks[found] = &talk{session: NewSession(merchant, visitor.gold), serial: r.talks}
		visitor.talks[found].session.Favor(Modifier(visitor.abilities[Charisma]))
		visitor.talks[found].session.Remember(false, merchant.Greeting)
	}

	visitor.shop, visitor.inn = found, r.evening(visitor)

	switch {
	case visitor.inn:
		r.tell(visitor, fmt.Sprintf("Wieczór: %s zamknięte. %s siedzi w karczmie przy stole; przysiadasz się. Handel rano, teraz rozmowa.", r.shopSpec(found).Name, r.shopSpec(found).Keeper))
	default:
		r.tell(visitor, "Wchodzisz: "+r.shopSpec(found).Name+".")
	}

	r.offerQuest(visitor, found)

	return found
}

func (r *Realm) tierCap(visitor *player) int {
	if len(r.world.Towns) > 1 && r.world.At(visitor.pos).Town == 0 {
		return firstTownTier
	}

	return world.MaxTier
}

// CloseShop takes the player out of the building they are in. The talk with
// its keeper, and the keeper's patience, last until the player leaves the
// settlement.
func (r *Realm) CloseShop(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if customer := r.players[name]; customer != nil {
		customer.shop = -1
	}
}

// HaggleRequest builds the request for reading what the player said to the
// keeper. It reports false when no conversation is going on.
func (r *Realm) HaggleRequest(name, text string) (InterpretRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.session(name)
	if session == nil || session.Over {
		return InterpretRequest{}, false
	}

	session.Owned = r.owned(r.players[name])

	return session.InterpretRequest(text), true
}

// owned lists what a customer carries unequipped, as things a merchant
// might buy.
func (r *Realm) owned(customer *player) []Item {
	owned := make([]Item, 0, len(customer.bag))
	for _, item := range customer.bag {
		owned = append(owned, ware(item))
	}

	return owned
}

// Haggle applies the player's intent to their conversation with the keeper.
// A purchase moves the gold and the item at once.
func (r *Realm) Haggle(name, text string, intent Intent) (Decision, NarrateRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	session := r.session(name)
	if session == nil {
		return Decision{}, NarrateRequest{}, false
	}

	customer := r.players[name]
	session.Gold, session.Owned = customer.gold, r.owned(customer)

	// After hours nothing changes hands: a deal proposed at the inn is
	// only talk.
	afterHours := customer.inn && (intent.Action == ActionOffer || intent.Action == ActionAccept || intent.Action == ActionSell)
	if afterHours {
		intent.Action = ActionChat
	}

	// A companion in the party sells nothing: "ile mamy bandaży?" and
	// "dobra, ruszamy" are talk, not a question about wares or a deal.
	// One who offers them money is told the deal stands: briefed as plain
	// talk, "dobrze, zgoda na 130" got "130, umowa stoi!".
	paying := false
	if kind, ok := r.mateOf(customer.shop); ok && r.hired[kind] == name && intent.Action != ActionLeave {
		paying = offersMoney(Fold(text))
		action := ActionChat
		if moving(text) {
			action = ActionLeave
		}

		intent = Intent{Action: action, Persuasion: intent.Persuasion}
	}

	// A companion is talked to in the role they had when the player spoke:
	// the hire that makes them a companion is still a mercenary's deal.
	role := ""
	if kind, ok := r.mateOf(customer.shop); ok {
		role = r.mateRole(kind)
	}

	// What the keeper's card says stings them is an insult: "ty szarlatanie" to
	// one who takes offence at "szarlatan" asks about no ware.
	stung := r.stung(customer.shop, Fold(text))
	if stung && intent.Action != ActionLeave {
		intent = Intent{Action: ActionChat, Persuasion: -2}
	}

	decision := session.Apply(intent)
	town := r.townOf(customer)

	// An insult or a threat leaves the keeper offended whatever their
	// patience: told "spokojna", the model shrugged off "milcz starucho".
	folded := Fold(text)
	threat, insult := threatening(folded), rude(folded) || stung
	if (threat || insult) && decision.Outcome == OutcomeChat {
		decision.Mood = MoodOffended
	}

	// A curse said to no one is no insult, but one who cannot stand
	// swearing bristles at it: told "spokojna", the model sharpened a nun's
	// "Przy mnie nie przeklinaj" into [gniew] on its own. The talk and the
	// standing go on as they were; only the mood of this turn changes.
	cursed := !threat && !insult && r.cursedAt(customer.shop, text)
	if cursed && decision.Outcome == OutcomeChat && (decision.Mood == MoodCalm || decision.Mood == MoodPleased || decision.Mood == MoodImpatient) {
		decision.Mood = MoodAnnoyed
	}

	// An apology, a threat or a question from outside the world is no
	// occasion to recommend wares.
	if _, quiet := aside(text); decision.Outcome == OutcomeChat && (quiet || threat || insult) {
		decision.Aside = true
	}

	switch decision.Outcome {
	case OutcomeChat:
		r.fire(customer, onTalk, session.Merchant.Name)
	case OutcomeInsulted:
		r.esteem(customer, town, insultStanding, "Obelgi w kramie niosą się po osadzie.")
	case OutcomeSold:
		if r.settle(customer, session, decision) {
			break
		}

		customer.gold = session.Gold
		r.bought(decision.Item.ID)
		r.esteem(customer, town, tradeStanding, "Kupcy lubią, kto zostawia pieniądze.")

		if item, ok := r.item(decision.Item.ID); ok {
			for range max(decision.Quantity, 1) {
				customer.bag = append(customer.bag, item)
			}
		}
	case OutcomeBought:
		customer.gold = session.Gold
		r.esteem(customer, town, tradeStanding, "Kupcy lubią, kto przynosi towar.")

		if index := find(customer.bag, decision.Item.Name); index >= 0 {
			customer.bag = remove(customer.bag, index)
		}
	}

	request := session.NarrateRequest(text, decision)
	request.Lore = r.lore(customer, session.Wares())
	hint, lead, anchored := r.hint(customer, text, decision, session.Wares())
	if paying {
		hint, lead, anchored = paidMate, nil, false
	}
	request.Hint, request.Quiet = hint, paying
	if anchored {
		request.Anchored, request.Names = true, r.names()
	}

	// A lead is remembered for the follow-up, must name its keeper
	// right, and goes to the journal; the chat before it is forgotten
	// once the talk moves on.
	if current := customer.talks[customer.shop]; current != nil && (lead != nil || decision.Outcome == OutcomeChat) {
		current.lead, current.leadKeeper = nil, -1
		if lead != nil {
			current.lead, current.leadKeeper = lead.fact, lead.keeper
		}
	}

	if lead != nil {
		request.Required = append(request.Required, r.keeperName(lead.keeper))
		if lead.fact.pointed[customer.name] == "" {
			request.Lead = r.point(customer, lead.fact, lead.keeper)
		} else {
			request.Lead = "Trop: " + lead.fact.Subject + " → " + r.keeperName(lead.keeper) + "."
		}
	}

	if r.investigation() {
		request.Role = RoleWitness
	} else if role != "" {
		request.Role = role

		// Small talk with a mercenary is not about the fee: quoting it
		// anyway ("dołączam za 85 koron" to "dołącz za darmo") earns the
		// model a second try.
		if request.Role == RoleMercenary && decision.Outcome == OutcomeChat {
			for _, ware := range session.Wares() {
				request.Forbidden = append(request.Forbidden, ware.Price)
			}
		}
	}

	switch {
	case threat && decision.Outcome == OutcomeChat:
		request.Hint = "Gracz ci grozi. Nie dajesz się zastraszyć: odpowiedz twardo, z gniewem albo z kpiną, i niczego mu nie dajesz ani nie sprzedajesz. " + request.Hint
	case insult && (decision.Outcome == OutcomeChat || decision.Outcome == OutcomeFarewell):
		hurt := "urażony"
		if speech.Feminine(r.keeperName(customer.shop)) {
			hurt = "urażona"
		}

		request.Hint = "Gracz cię obraża. Odpowiedz " + hurt + ", zimno albo z gniewem; nie udawaj, że nic się nie stało. " + request.Hint
	case cursed && (decision.Outcome == OutcomeChat || decision.Outcome == OutcomeFarewell):
		request.Hint = strings.TrimSpace(curseHint + " " + request.Hint)
	}

	// An apology before a question or a deal is taken in passing.
	if hasPhrase(folded, sorryWords) && !strings.HasPrefix(request.Hint, "Gracz przeprasza") && !threat && !insult {
		request.Hint = strings.TrimSpace("Gracz zaczyna od przeprosin: przyjmij je krótko po swojemu, a potem odpowiedz na resztę. " + request.Hint)
	}

	if afterHours {
		request.Hint = "Gracz chce handlować, ale jest wieczór i kram zamknięty: odmów grzecznie, zaproś rano, nic nie sprzedawaj. " + request.Hint
	}

	return decision, request, true
}

// HaggleRemember records an exchange in the conversation's history.
func (r *Realm) HaggleRemember(name, said, reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session := r.session(name); session != nil {
		session.Remember(true, said)
		session.Remember(false, reply)
	}
}

// session returns the conversation of a connected player with the keeper
// of the building they are in.
func (r *Realm) session(name string) *Session {
	customer := r.active(name)
	if customer == nil || customer.shop < 0 || customer.talks[customer.shop] == nil {
		return nil
	}

	return customer.talks[customer.shop].session
}
