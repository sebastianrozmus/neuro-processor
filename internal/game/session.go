package game

const (
	// Each point of goodwill moves the merchant's floor price by 2.5%.
	goodwillStepPermille = 25
	minGoodwill          = -2
	maxGoodwill          = 4

	// The merchant gives up this share of the gap to the player's offer.
	concessionPercent = 30

	// Buying several pieces at once is cheaper: every piece beyond the
	// first takes this much off the price of each, up to a limit.
	bulkStepPercent = 3
	maxBulkPercent  = 15
	// MaxQuantity is the most pieces that can be bought in one go.
	MaxQuantity = 20

	historyLimit = 12

	// A merchant pays this share of the list price for what the player
	// brings, and can be talked up to the higher share; each point of
	// goodwill adds a little.
	buyPercent         = 40
	maxBuyPercent      = 55
	goodwillBuyPercent = 2
)

// Session is the state of one conversation with a merchant.
type Session struct {
	Merchant  Merchant
	Gold      int
	Inventory []Item
	Over      bool
	// Owned is what the player carries and could sell; Sold is what the
	// merchant bought from them.
	Owned []Item
	Sold  []Item

	stock []Item
	asks  map[string]int
	// bids are what the merchant currently offers for the player's things;
	// selling marks that the focused item is one of them.
	bids     map[string]int
	selling  bool
	patience int
	goodwill int
	focus    string
	// quantity is how many pieces of the focused item the talk is about.
	quantity int
	history  []Line
}

// NewSession starts a conversation with the given merchant.
func NewSession(merchant Merchant, gold int) *Session {
	s := &Session{
		Merchant: merchant,
		Gold:     gold,
		stock:    append([]Item(nil), merchant.Items...),
		asks:     make(map[string]int, len(merchant.Items)),
		bids:     map[string]int{},
		patience: merchant.Patience,
	}

	for _, item := range merchant.Items {
		s.asks[item.ID] = item.ListPrice
	}

	return s
}

// Favor shifts the merchant's goodwill, as a charming customer earns it
// before a word is said.
func (s *Session) Favor(points int) {
	s.goodwill = clamp(s.goodwill+points, minGoodwill, maxGoodwill)
}

// Stock adds an item to what the merchant sells, at its list price.
func (s *Session) Stock(item Item) {
	s.Merchant.Items = append(s.Merchant.Items, item)
	s.stock = append(s.stock, item)
	s.asks[item.ID] = item.ListPrice
}

// Patience returns how many more setbacks the merchant will put up with.
func (s *Session) Patience() int {
	return s.patience
}

// Wares returns the items still for sale with their current asking prices.
func (s *Session) Wares() []Ware {
	wares := make([]Ware, 0, len(s.stock))
	for _, item := range s.stock {
		wares = append(wares, s.ware(item))
	}

	return wares
}

// OwnedWares returns the player's things as the merchant sees them, with
// what the merchant would pay for each.
func (s *Session) OwnedWares() []Ware {
	wares := make([]Ware, 0, len(s.Owned))

	for _, item := range s.Owned {
		ware := s.ware(item)
		ware.Price = s.bids[item.ID]

		if ware.Price == 0 {
			ware.Price = max(1, item.ListPrice*buyPercent/100)
		}

		wares = append(wares, ware)
	}

	return wares
}

// Focus returns the ID of the item the conversation is currently about.
func (s *Session) Focus() string {
	return s.focus
}

// History returns the most recent lines of dialogue.
func (s *Session) History() []Line {
	return append([]Line(nil), s.history...)
}

// Remember appends a line of dialogue to the history.
func (s *Session) Remember(player bool, text string) {
	s.history = append(s.history, Line{Player: player, Text: text})
	if len(s.history) > historyLimit {
		s.history = s.history[len(s.history)-historyLimit:]
	}
}

// Apply resolves an intent against the rules and updates the state.
func (s *Session) Apply(intent Intent) Decision {
	if s.Over {
		return s.decide(OutcomeFarewell, nil, 0)
	}

	s.goodwill = clamp(s.goodwill+clamp(intent.Persuasion, -2, 2), minGoodwill, maxGoodwill)

	// Selling is its own thread of the talk: it goes on until a thing is
	// sold, the player turns to buying or names nothing the merchant could
	// take.
	if item := s.resolveOwned(""); item != nil && intent.ItemID == "" && intent.Action == ActionAccept {
		return s.buy(*item, s.bids[item.ID])
	}

	if intent.Action == ActionSell || (s.selling && intent.ItemID == "" && intent.Action == ActionOffer) {
		return s.buyFrom(s.resolveOwned(intent.ItemID), intent.Amount)
	}

	s.selling = false
	item := s.resolve(intent.ItemID)
	s.count(item, intent.Quantity)

	if item != nil {
		s.focus = item.ID
	}

	trading := intent.Action == ActionAsk || intent.Action == ActionAccept || intent.Action == ActionOffer
	if trading && item != nil && !item.Unlimited && s.quantity > 1 {
		wanted := s.quantity
		s.quantity = 1

		decision := s.decide(OutcomeShort, item, 0)
		decision.Quantity = wanted

		return decision
	}

	switch intent.Action {
	case ActionLeave:
		s.Over = true

		return s.decide(OutcomeFarewell, item, 0)
	case ActionAsk:
		if item == nil {
			return s.decide(OutcomeWares, nil, 0)
		}

		return s.decide(OutcomeInfo, item, 0)
	case ActionAccept:
		if item == nil {
			return s.decide(OutcomeUnclear, nil, 0)
		}

		if s.asked(*item) > s.Gold {
			return s.decide(OutcomeNoGold, item, 0)
		}

		return s.sell(*item, 0, s.asked(*item))
	case ActionOffer:
		return s.offer(item, intent.Amount)
	}

	if intent.Persuasion <= -2 && s.irritate(1) {
		return s.decide(OutcomeWalkedAway, item, 0)
	}

	return s.decide(OutcomeChat, item, 0)
}

// count settles how many pieces the talk is about: what the player just
// said, otherwise what was said before about the same item, otherwise one.
func (s *Session) count(item *Item, wanted int) {
	switch {
	case item == nil:
	case wanted > 0:
		s.quantity = min(wanted, MaxQuantity)
	case item.ID != s.focus || s.quantity == 0:
		s.quantity = 1
	}
}

// offer weighs an amount proposed for the pieces the talk is about. All
// sums are for the whole lot.
func (s *Session) offer(item *Item, amount int) Decision {
	if item == nil || amount <= 0 {
		return s.decide(OutcomeUnclear, item, 0)
	}

	if amount > s.Gold {
		return s.decide(OutcomeNoGold, item, amount)
	}

	ask := s.asked(*item)
	floor := lot(s.floor(*item), s.quantity)

	if amount >= ask {
		return s.sell(*item, amount, ask)
	}

	if amount >= floor {
		return s.sell(*item, amount, amount)
	}

	if amount*2 < floor {
		if s.irritate(2) {
			return s.decide(OutcomeWalkedAway, item, amount)
		}

		return s.decide(OutcomeInsulted, item, amount)
	}

	// The concession is made on the lot and carried over to the price of
	// one piece, which never rises and never drops under the floor.
	conceded := max(floor, ask-max(1, (ask-amount)*concessionPercent/100))
	each := (conceded*100 + s.quantity*(100-bulk(s.quantity)) - 1) / (s.quantity * (100 - bulk(s.quantity)))
	s.asks[item.ID] = min(s.asks[item.ID], max(each, s.floor(*item)))

	if s.irritate(1) {
		return s.decide(OutcomeWalkedAway, item, amount)
	}

	return s.decide(OutcomeCounter, item, amount)
}

// buyFrom weighs the player's wish to sell a thing, for the sum they ask
// or for whatever the merchant gives.
func (s *Session) buyFrom(item *Item, asked int) Decision {
	if item == nil {
		s.selling, s.focus = false, ""

		return s.decide(OutcomeUnclear, nil, asked)
	}

	s.selling, s.focus, s.quantity = true, item.ID, 1

	if !s.wants(*item) {
		s.selling, s.focus = false, ""

		return s.decide(OutcomeUnwanted, item, asked)
	}

	if _, ok := s.bids[item.ID]; !ok {
		s.bids[item.ID] = max(1, item.ListPrice*buyPercent/100)
	}

	bid, ceiling := s.bids[item.ID], s.ceiling(*item)

	switch {
	case asked <= 0:
		return s.decide(OutcomeBid, item, 0)
	case asked <= max(bid, ceiling):
		return s.buy(*item, min(asked, max(bid, ceiling)))
	}

	// Too much: the bid creeps up towards the demand, never past the ceiling.
	s.bids[item.ID] = min(ceiling, max(bid, bid+max(1, (asked-bid)*concessionPercent/100)))

	if s.irritate(1) {
		return s.decide(OutcomeWalkedAway, item, asked)
	}

	return s.decide(OutcomeBid, item, asked)
}

// buy takes a thing off the player for the price; what the merchant deals
// in goes on the shelf, to be bought back at the list price.
func (s *Session) buy(item Item, price int) Decision {
	s.Gold += price
	s.Sold = append(s.Sold, item)

	for i := range s.Owned {
		if s.Owned[i].ID == item.ID {
			s.Owned = append(s.Owned[:i], s.Owned[i+1:]...)

			break
		}
	}

	if s.resolve(item.ID) == nil && !item.Unlimited {
		s.stock = append(s.stock, item)
		s.asks[item.ID] = item.ListPrice
	}

	decision := s.decide(OutcomeBought, &item, 0)
	decision.Price = price

	s.selling, s.focus, s.quantity = false, "", 0
	delete(s.bids, item.ID)

	return decision
}

// wants reports whether the merchant deals in things of the item's kind.
func (s *Session) wants(item Item) bool {
	for _, kind := range s.Merchant.Kinds {
		if kind == item.Kind {
			return true
		}
	}

	return len(s.Merchant.Kinds) == 0
}

// ceiling is the most the merchant will ever pay for the player's thing.
func (s *Session) ceiling(item Item) int {
	return max(1, item.ListPrice*(maxBuyPercent+goodwillBuyPercent*s.goodwill)/100)
}

func (s *Session) resolveOwned(id string) *Item {
	if id == "" && s.selling {
		id = s.focus
	}

	for i := range s.Owned {
		if s.Owned[i].ID == id {
			return &s.Owned[i]
		}
	}

	return nil
}

func (s *Session) sell(item Item, offer, price int) Decision {
	s.Gold -= price

	for range s.quantity {
		s.Inventory = append(s.Inventory, item)
	}

	decision := s.decide(OutcomeSold, &item, offer)
	decision.Price = price

	s.focus, s.quantity = "", 0
	s.asks[item.ID] = item.ListPrice

	for i := range s.stock {
		if s.stock[i].ID == item.ID && !item.Unlimited {
			s.stock = append(s.stock[:i], s.stock[i+1:]...)

			break
		}
	}

	return decision
}

// asked is what the merchant currently wants for the pieces the talk is
// about.
func (s *Session) asked(item Item) int {
	return lot(s.asks[item.ID], s.quantity)
}

// bulk is the discount, in percent, on each of several pieces bought at once.
func bulk(quantity int) int {
	return min(maxBulkPercent, bulkStepPercent*(quantity-1))
}

// lot prices a number of pieces from the price of one, rounding up.
func lot(each, quantity int) int {
	return (each*quantity*(100-bulk(quantity)) + 99) / 100
}

// floor is the lowest price the merchant accepts right now. Goodwill earned
// by persuasion lowers it by at most 10% and bad manners raise it by 5%.
func (s *Session) floor(item Item) int {
	permille := 1000 - s.goodwill*goodwillStepPermille

	return (item.MinPrice*permille + 999) / 1000
}

// irritate wears the merchant's patience down and reports whether it ran out.
func (s *Session) irritate(points int) bool {
	s.patience = max(0, s.patience-points)
	if s.patience == 0 {
		s.Over = true
	}

	return s.Over
}

func (s *Session) resolve(id string) *Item {
	if id == "" {
		id = s.focus
	}

	for i := range s.stock {
		if s.stock[i].ID == id {
			return &s.stock[i]
		}
	}

	return nil
}

func (s *Session) ware(item Item) Ware {
	return Ware{ID: item.ID, Name: item.Name, Description: item.Description, Aliases: item.Aliases, Price: s.asks[item.ID]}
}

func (s *Session) decide(outcome Outcome, item *Item, offer int) Decision {
	decision := Decision{
		Outcome:  outcome,
		Offer:    offer,
		Gold:     s.Gold,
		Mood:     s.mood(outcome),
		Currency: s.Merchant.Currency,
	}

	if item != nil {
		decision.Item = s.ware(*item)
		decision.Quantity = max(s.quantity, 1)
		decision.Price = s.asked(*item)

		if bid, ok := s.bids[item.ID]; ok && (outcome == OutcomeBid || outcome == OutcomeWalkedAway) {
			decision.Item.Price, decision.Price = bid, bid
		}
	}

	if outcome == OutcomeWares {
		decision.Wares = s.Wares()
	}

	return decision
}

func (s *Session) mood(outcome Outcome) Mood {
	switch {
	case outcome == OutcomeSold || outcome == OutcomeBought:
		return MoodPleased
	case outcome == OutcomeInsulted || outcome == OutcomeWalkedAway:
		return MoodOffended
	case s.patience >= s.Merchant.Patience-1:
		return MoodCalm
	case s.patience*2 >= s.Merchant.Patience:
		return MoodImpatient
	}

	return MoodAnnoyed
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}
