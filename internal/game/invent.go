package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// InventRequest asks whether a keeper's line offered something that does
// not exist yet.
type InventRequest struct {
	Reply string
	Wares []Ware
}

// Inventor reads a keeper's line and returns the item it offered, if it
// offered one that is not among the wares.
type Inventor interface {
	Invent(ctx context.Context, request InventRequest) (world.ItemSpec, bool, error)
}

// extra is an item that a keeper talked into existence.
type extra struct {
	shop int
	item world.ItemSpec
}

// Invent lets an item a keeper offered in conversation into the world: it
// joins that shop's stock, in the customer's talk right away and in every
// talk begun later. Items the world already has, under this or a similar
// name, are not doubled. It reports whether the item was added.
func (r *Realm) Invent(name string, item world.ItemSpec, line string) (world.ItemSpec, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	customer := r.active(name)
	if customer == nil || customer.shop < 0 || customer.talks[customer.shop] == nil {
		return world.ItemSpec{}, false
	}

	// Only a shopkeeper's talk brings goods into the world: a mercenary's
	// one ware is their service (a muddled line once made "Umowa Najmu" a
	// charm with attack +2), and a witness sells nothing.
	if _, mate := r.mateOf(customer.shop); mate || r.investigation() {
		return world.ItemSpec{}, false
	}

	item, ok := world.CleanItem(item)
	if !ok || r.known(item.Name) {
		return world.ItemSpec{}, false
	}

	item.ID = fmt.Sprintf("extra-%d", len(r.extras)+1)
	r.extras = append(r.extras, extra{shop: customer.shop, item: item})
	r.trace(DescribeItem, item.ID, Trace{How: HowInvented + " (" + r.keeperName(customer.shop) + ")", Prompt: line})
	customer.talks[customer.shop].session.Stock(ware(item))
	r.tell(customer, fmt.Sprintf("Do świata wpada: %s (%d %s). %s", item.Name, item.Price(), r.world.Currency, Effect(item)))

	return item, true
}

// Offered checks a keeper's line against the small model's verdict: the
// item counts as offered only in a sentence that names it and sells it,
// with a price or a word like "mam" or "polecam", and does not turn it
// away. "Jeśli chcesz jagody, idź po nie w kopalni" once put berries on a
// smith's shelf.
func Offered(line, name string) bool {
	if _, rest, tagged := strings.Cut(line, "] "); tagged && strings.HasPrefix(line, "[") {
		line = rest
	}

	for _, sentence := range strings.FieldsFunc(line, func(char rune) bool { return strings.ContainsRune(".!?;", char) }) {
		if !Matches(sentence, name) {
			continue
		}

		folded := " " + strings.Join(strings.Fields(Fold(sentence)), " ") + " "
		if containsAny(folded, refusalCues) {
			return false
		}

		if containsAny(folded, offerCues) || strings.ContainsAny(folded, "0123456789") {
			return true
		}
	}

	return false
}

// offerCues and refusalCues are folded words that begin with a space, so
// that "mam" does not match inside "mamy" only by accident of a suffix.
var (
	offerCues = []string{
		" mam ", " mamy ", " sprzeda", " oferu", " polec", " kup ", " kupisz ", " kupic ", " wez ", " wezcie ", " dam ", " dostaniesz ",
		" znajdzie sie ", " moge dac ", " moge sprzedac ", " trzymam ", " na polce ", " w cenie ", " kosztuje ",
	}
	refusalCues = []string{
		" nie mam ", " nie mamy ", " nie sprzeda", " nie trzymam ", " nie handluj", " nie prowadz", " nie znajdziesz ",
		" nie ma ", " brak ", " idz po ", " szukaj ", " poszukaj ", " zbierz ",
	}
)

func containsAny(text string, cues []string) bool {
	for _, cue := range cues {
		if strings.Contains(text, cue) {
			return true
		}
	}

	return false
}

// known reports whether the world already has an item of this or a
// similar name.
func (r *Realm) known(name string) bool {
	for _, item := range r.world.Items {
		if Matches(item.Name, name) || Matches(name, item.Name) {
			return true
		}
	}

	for _, added := range r.extras {
		if Matches(added.item.Name, name) || Matches(name, added.item.Name) {
			return true
		}
	}

	return false
}

// item finds an item of the world or one talked into it.
func (r *Realm) item(id string) (world.ItemSpec, bool) {
	if item, ok := r.world.Item(id); ok {
		return item, true
	}

	for _, added := range r.extras {
		if added.item.ID == id {
			return added.item, true
		}
	}

	return world.ItemSpec{}, false
}

// merchant builds the keeper of a shop with everything it sells here,
// including what was talked into existence.
func (r *Realm) merchant(shop, maxTier int) Merchant {
	if r.caravan != nil && shop == r.caravanShop() {
		merchant := r.caravanMerchant()
		for _, added := range r.extras {
			if added.shop == shop {
				merchant.Items = append(merchant.Items, ware(added.item))
			}
		}

		return merchant
	}

	merchant := MerchantFor(r.world.Spec, shop, maxTier)

	for _, added := range r.extras {
		if added.shop == shop {
			merchant.Items = append(merchant.Items, ware(added.item))
		}
	}

	// Prices drift with the days and with what was bought.
	for index := range merchant.Items {
		merchant.Items[index] = r.priced(merchant.Items[index])
	}

	return merchant
}
