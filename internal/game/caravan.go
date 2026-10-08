package game

import (
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// A caravan is a shop on the road: a wandering merchant who travels from
// settlement to settlement, carries everything the world has to sell, the
// best of it included, asks more for it and knows the news of every place
// they passed through.

const (
	// caravanSteps is how far the caravan travels in a day.
	caravanSteps = 2
	// caravanMarkup is what the road adds to the caravan's prices.
	caravanMarkup = 130
	caravanName   = "Karawana"
	caravanKeeper = "Wędrowny Kupiec"
)

// caravan is the state of the wandering merchant.
type caravan struct {
	pos  world.Point
	goal int
}

// caravanShop is the shop index the caravan talks under: one past the
// settlements' shops.
func (r *Realm) caravanShop() int {
	return len(r.world.Shops)
}

// caravanSpec describes the caravan as a shop.
var caravanSpec = world.ShopSpec{
	Name:     caravanName,
	Kind:     world.ShopCaravan,
	Keeper:   caravanKeeper,
	Persona:  "Wędrowny kupiec, który zna każdą drogę i każdą plotkę między osadami. Uprzejmy, wygadany, drogi; wie, że kto kupuje na szlaku, nie ma wyboru. Chętnie opowiada, co widział po drodze.",
	Greeting: "Witaj na szlaku. Mam, czego nie ma w osadach, ale i cena jest ze szlaku.",
}

// shopSpec returns a shop of the world or the caravan.
func (r *Realm) shopSpec(index int) world.ShopSpec {
	if index == r.caravanShop() {
		return caravanSpec
	}

	return r.world.Shops[index]
}

// newCaravan sets the caravan out from the second settlement towards the
// first, when the world has roads worth travelling.
func (r *Realm) newCaravan() {
	if r.adventure() || len(r.world.Towns) < 2 {
		return
	}

	r.caravan = &caravan{pos: r.world.Towns[1], goal: 0}
}

// caravanHere reports whether the caravan stands at a point.
func (r *Realm) caravanHere(at world.Point) bool {
	return r.caravan != nil && r.caravan.pos == at
}

// travel moves the caravan a day's way towards its goal, around monsters;
// at the goal it picks the next settlement.
func (r *Realm) travel() {
	if r.caravan == nil {
		return
	}

	for range caravanSteps {
		goal := r.world.Towns[r.caravan.goal]
		if r.caravan.pos == goal {
			r.caravan.goal = (r.caravan.goal + 1) % len(r.world.Towns)

			break
		}

		best, bestSteps := r.caravan.pos, distance(r.caravan.pos, goal)

		for _, step := range []world.Point{north, east, south, west} {
			next := world.Point{X: r.caravan.pos.X + step.X, Y: r.caravan.pos.Y + step.Y}
			if r.world.In(next) && r.alive(next) == 0 && distance(next, goal) < bestSteps {
				best, bestSteps = next, distance(next, goal)
			}
		}

		if best == r.caravan.pos {
			// Monsters bar every way forward: the caravan waits a day.
			break
		}

		r.caravan.pos = best
		r.revealed[r.world.Index(best)] = true
	}
}

// caravanMerchant stocks the caravan: every kind of thing, up to the
// highest tier, at road prices.
func (r *Realm) caravanMerchant() Merchant {
	merchant := Merchant{
		Name:     caravanSpec.Keeper,
		Persona:  caravanSpec.Persona,
		Greeting: caravanSpec.Greeting,
		Patience: merchantPatience,
		Currency: r.world.Currency,
		Kinds:    append([]world.Kind(nil), world.Kinds...),
	}

	for _, item := range r.world.Items {
		priced := r.priced(ware(item))
		priced.ListPrice = max(1, priced.ListPrice*caravanMarkup/100)
		priced.MinPrice = max(1, priced.MinPrice*caravanMarkup/100)
		merchant.Items = append(merchant.Items, priced)
	}

	return merchant
}

// caravanWords are what a player may call the caravan.
const caravanWords = "karawana kupiec wedrowny woz handlarz"

// namesCaravan reports whether a player's words mean the caravan.
func namesCaravan(what string) bool {
	return what == "" || Matches(caravanWords, what) || Matches(caravanName, what) || Matches(caravanKeeper, what)
}

// caravanLine is what a traveller sees when the caravan is at hand.
func caravanLine() string {
	return "Stoi tu " + strings.ToLower(caravanName) + " wędrownego kupca: wozy, płachty, towar z całego świata. Handlujesz poleceniem «handluj z karawaną»."
}
