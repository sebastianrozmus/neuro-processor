package game

import (
	"fmt"

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

	decision := session.Apply(intent)
	town := r.townOf(customer)

	switch decision.Outcome {
	case OutcomeChat:
		r.fire(customer, onTalk, session.Merchant.Name)
	case OutcomeInsulted:
		r.esteem(customer, town, insultStanding, "Obelgi w kramie niosą się po osadzie.")
	case OutcomeSold:
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
	request.Hint = r.hint(customer, text, decision)

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
