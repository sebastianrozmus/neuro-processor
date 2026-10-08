package game

import (
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

func testMerchant() Merchant {
	return Merchant{
		Name:     "Test",
		Patience: 4,
		Items: []Item{
			{ID: "a", Name: "Amulet", ListPrice: 200, MinPrice: 100},
			{ID: "b", Name: "Berło", ListPrice: 50, MinPrice: 30},
		},
	}
}

func TestOffer(t *testing.T) {
	tests := []struct {
		name         string
		gold         int
		intent       Intent
		wantOutcome  Outcome
		wantPrice    int
		wantGold     int
		wantPatience int
	}{
		{"at floor sells at offer", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 100}, OutcomeSold, 100, 200, 4},
		{"above ask sells at ask", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 250}, OutcomeSold, 200, 100, 4},
		{"below floor gets counter", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 90}, OutcomeCounter, 167, 300, 3},
		{"counter never drops below floor", 300, Intent{Action: ActionOffer, ItemID: "b", Amount: 29}, OutcomeCounter, 44, 300, 3},
		{"lowball insults", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 49}, OutcomeInsulted, 200, 300, 2},
		{"more than gold", 80, Intent{Action: ActionOffer, ItemID: "a", Amount: 150}, OutcomeNoGold, 200, 80, 4},
		{"no amount", 300, Intent{Action: ActionOffer, ItemID: "a"}, OutcomeUnclear, 200, 300, 4},
		{"no item", 300, Intent{Action: ActionOffer, Amount: 100}, OutcomeUnclear, 0, 300, 4},
		{"accept buys at ask", 300, Intent{Action: ActionAccept, ItemID: "b"}, OutcomeSold, 50, 250, 4},
		{"accept without gold", 10, Intent{Action: ActionAccept, ItemID: "b"}, OutcomeNoGold, 50, 10, 4},
		{"persuasion lowers floor", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 95, Persuasion: 2}, OutcomeSold, 95, 205, 4},
		{"rudeness raises floor", 300, Intent{Action: ActionOffer, ItemID: "a", Amount: 100, Persuasion: -2}, OutcomeCounter, 170, 300, 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := NewSession(testMerchant(), test.gold)
			decision := session.Apply(test.intent)

			if decision.Outcome != test.wantOutcome {
				t.Fatalf("outcome = %s, want %s", decision.Outcome, test.wantOutcome)
			}

			if decision.Price != test.wantPrice {
				t.Errorf("price = %d, want %d", decision.Price, test.wantPrice)
			}

			if session.Gold != test.wantGold {
				t.Errorf("gold = %d, want %d", session.Gold, test.wantGold)
			}

			if session.Patience() != test.wantPatience {
				t.Errorf("patience = %d, want %d", session.Patience(), test.wantPatience)
			}
		})
	}
}

func TestSaleMovesItemToInventory(t *testing.T) {
	session := NewSession(testMerchant(), 300)
	session.Apply(Intent{Action: ActionAccept, ItemID: "a"})

	if len(session.Inventory) != 1 || session.Inventory[0].ID != "a" {
		t.Fatalf("inventory = %+v, want the amulet", session.Inventory)
	}

	if wares := session.Wares(); len(wares) != 1 || wares[0].ID != "b" {
		t.Fatalf("wares = %+v, want only the sceptre", wares)
	}

	if got := session.Apply(Intent{Action: ActionAccept, ItemID: "a"}).Outcome; got != OutcomeUnclear {
		t.Errorf("buying a sold item = %s, want %s", got, OutcomeUnclear)
	}
}

func TestPersuasionDiscountIsCapped(t *testing.T) {
	session := NewSession(testMerchant(), 300)
	for range 10 {
		session.Apply(Intent{Action: ActionChat, Persuasion: 2})
	}

	if got := session.Apply(Intent{Action: ActionOffer, ItemID: "a", Amount: 89}).Outcome; got != OutcomeCounter {
		t.Errorf("offer below the capped floor = %s, want %s", got, OutcomeCounter)
	}

	if got := session.Apply(Intent{Action: ActionOffer, ItemID: "a", Amount: 90}).Outcome; got != OutcomeSold {
		t.Errorf("offer at the capped floor = %s, want %s", got, OutcomeSold)
	}
}

func TestMerchantWalksAwayWhenPatienceRunsOut(t *testing.T) {
	session := NewSession(testMerchant(), 300)
	lowball := Intent{Action: ActionOffer, ItemID: "a", Amount: 10}

	if got := session.Apply(lowball).Outcome; got != OutcomeInsulted {
		t.Fatalf("first lowball = %s, want %s", got, OutcomeInsulted)
	}

	if got := session.Apply(lowball).Outcome; got != OutcomeWalkedAway {
		t.Fatalf("second lowball = %s, want %s", got, OutcomeWalkedAway)
	}

	if !session.Over {
		t.Error("session should be over")
	}

	session.Apply(Intent{Action: ActionAccept, ItemID: "b"})

	if session.Gold != 300 {
		t.Errorf("gold = %d after the merchant left, want 300", session.Gold)
	}
}

func TestFocusCarriesOverBetweenTurns(t *testing.T) {
	session := NewSession(testMerchant(), 300)
	session.Apply(Intent{Action: ActionAsk, ItemID: "b"})

	decision := session.Apply(Intent{Action: ActionOffer, Amount: 30})
	if decision.Outcome != OutcomeSold || decision.Item.ID != "b" {
		t.Errorf("decision = %+v, want the sceptre sold", decision)
	}
}

func TestUnlimitedItemStaysInStockAtListPrice(t *testing.T) {
	merchant := testMerchant()
	merchant.Items[1].Unlimited = true

	session := NewSession(merchant, 300)
	session.Apply(Intent{Action: ActionOffer, ItemID: "b", Amount: 29})

	if got := session.Apply(Intent{Action: ActionAccept, ItemID: "b"}).Price; got != 44 {
		t.Fatalf("price = %d, want the haggled 44", got)
	}

	wares := session.Wares()
	if len(wares) != 2 || wares[1].Price != 50 {
		t.Errorf("wares = %+v, want the sceptre back at 50", wares)
	}
}

func TestBuyingSeveralPieces(t *testing.T) {
	merchant := testMerchant()
	merchant.Items[1].Unlimited = true // the sceptre: 50 each, never under 30

	t.Run("a lot is cheaper per piece", func(t *testing.T) {
		session := NewSession(merchant, 1000)

		decision := session.Apply(Intent{Action: ActionAsk, ItemID: "b", Quantity: 5})
		if decision.Outcome != OutcomeInfo || decision.Quantity != 5 || decision.Price != 220 || decision.Item.Price != 50 {
			t.Fatalf("asking about five: %+v", decision)
		}

		// The count is remembered: a bare offer still means five pieces.
		decision = session.Apply(Intent{Action: ActionOffer, Amount: 100})
		if decision.Outcome != OutcomeCounter || decision.Quantity != 5 || decision.Offer != 100 || decision.Price != 185 {
			t.Fatalf("offering 100 for five: %+v", decision)
		}

		decision = session.Apply(Intent{Action: ActionOffer, Amount: 132})
		if decision.Outcome != OutcomeSold || decision.Quantity != 5 || decision.Price != 132 {
			t.Fatalf("offering the floor for five: %+v", decision)
		}

		if session.Gold != 868 || len(session.Inventory) != 5 {
			t.Errorf("gold %d, %d pieces", session.Gold, len(session.Inventory))
		}

		// After the sale the talk is about single pieces at list price again.
		if decision = session.Apply(Intent{Action: ActionAsk, ItemID: "b"}); decision.Quantity != 1 || decision.Price != 50 {
			t.Errorf("after the sale: %+v", decision)
		}
	})

	t.Run("the lot has its own floor", func(t *testing.T) {
		session := NewSession(merchant, 1000)

		if got := session.Apply(Intent{Action: ActionOffer, ItemID: "b", Quantity: 5, Amount: 131}).Outcome; got != OutcomeCounter {
			t.Errorf("one under the floor of the lot: %s", got)
		}

		if got := session.Apply(Intent{Action: ActionOffer, ItemID: "b", Quantity: 5, Amount: 60}).Outcome; got != OutcomeInsulted {
			t.Errorf("a lowball for the lot: %s", got)
		}

		session = NewSession(merchant, 1000)

		if got := session.Apply(Intent{Action: ActionAccept, ItemID: "b", Quantity: 3}); got.Outcome != OutcomeSold || got.Price != 141 || got.Quantity != 3 {
			t.Errorf("taking three at the asked price: %+v", got)
		}
	})

	t.Run("no more than the purse or the shelf holds", func(t *testing.T) {
		session := NewSession(merchant, 100)

		if got := session.Apply(Intent{Action: ActionAccept, ItemID: "b", Quantity: 5}); got.Outcome != OutcomeNoGold || session.Gold != 100 {
			t.Errorf("five pieces with 100 gold: %+v", got)
		}

		decision := session.Apply(Intent{Action: ActionOffer, ItemID: "a", Quantity: 3, Amount: 90})
		if decision.Outcome != OutcomeShort || decision.Quantity != 3 || decision.Price != 200 || session.Gold != 100 {
			t.Errorf("three amulets when there is one: %+v", decision)
		}

		if got := session.Apply(Intent{Action: ActionOffer, ItemID: "b", Quantity: 500, Amount: 1}); got.Quantity != MaxQuantity {
			t.Errorf("quantity = %d, want the cap", got.Quantity)
		}
	})

	t.Run("words stay within the rules' numbers", func(t *testing.T) {
		session := NewSession(merchant, 1000)

		for _, intent := range []Intent{
			{Action: ActionAsk, ItemID: "b", Quantity: 5},
			{Action: ActionOffer, Amount: 100},
			{Action: ActionOffer, Amount: 10},
			{Action: ActionOffer, ItemID: "a", Quantity: 2, Amount: 300},
			{Action: ActionOffer, ItemID: "b", Quantity: 4, Amount: 150},
		} {
			decision := session.Apply(intent)
			if !NumbersOK(decision.Fallback(), decision.Numbers()) || !NumbersOK(decision.Brief(), append(decision.Numbers(), decision.Gold)) {
				t.Errorf("%s: %q / %q use numbers outside %v", decision.Outcome, decision.Fallback(), decision.Brief(), decision.Numbers())
			}
		}
	})
}

func TestSellingToTheMerchant(t *testing.T) {
	merchant := testMerchant()
	merchant.Kinds = []world.Kind{world.KindCharm}

	session := NewSession(merchant, 10)
	session.Owned = []Item{
		{ID: "x", Name: "Pierścień", Kind: world.KindCharm, ListPrice: 100, MinPrice: 65},
		{ID: "y", Name: "Miecz", Kind: world.KindWeapon, ListPrice: 100, MinPrice: 65},
	}

	// Asking what the merchant gives names the bid: 40% of the list price.
	decision := session.Apply(Intent{Action: ActionSell, ItemID: "x"})
	if decision.Outcome != OutcomeBid || decision.Price != 40 || decision.Offer != 0 {
		t.Fatalf("bid = %+v", decision)
	}

	// Too much moves the bid up a little and costs patience; the ceiling is 55%.
	decision = session.Apply(Intent{Action: ActionSell, Amount: 90})
	if decision.Outcome != OutcomeBid || decision.Price != 55 || decision.Offer != 90 || session.Patience() != 3 {
		t.Fatalf("counter = %+v, patience %d", decision, session.Patience())
	}

	// Agreeing while selling sells at the bid; the thing goes on the shelf.
	decision = session.Apply(Intent{Action: ActionAccept})
	if decision.Outcome != OutcomeBought || decision.Price != 55 || session.Gold != 65 || len(session.Owned) != 1 || len(session.Sold) != 1 {
		t.Fatalf("sale = %+v, gold %d, owned %d", decision, session.Gold, len(session.Owned))
	}

	if session.resolve("x") == nil || session.asks["x"] != 100 {
		t.Error("the bought ring is not for sale at its list price")
	}

	// A fair demand is paid as asked; a thing of a strange kind is refused.
	session.Owned = append(session.Owned, Item{ID: "z", Name: "Talizman", Kind: world.KindCharm, ListPrice: 100, MinPrice: 65})
	if decision = session.Apply(Intent{Action: ActionSell, ItemID: "z", Amount: 50}); decision.Outcome != OutcomeBought || decision.Price != 50 {
		t.Errorf("fair demand = %+v", decision)
	}

	if decision = session.Apply(Intent{Action: ActionSell, ItemID: "y", Amount: 10}); decision.Outcome != OutcomeUnwanted {
		t.Errorf("sword = %+v", decision)
	}

	// Buying again leaves the selling thread.
	if decision = session.Apply(Intent{Action: ActionAsk, ItemID: "a"}); decision.Outcome != OutcomeInfo || session.selling {
		t.Errorf("ask = %+v, selling %v", decision, session.selling)
	}
}

func TestReadingSellTalk(t *testing.T) {
	request := InterpretRequest{
		Wares:    []Ware{{ID: "a", Name: "Amulet", Aliases: Aliases("Amulet")}},
		Owned:    []Ware{{ID: "k", Name: "Kij", Aliases: Aliases("Kij")}},
		Currency: "szt. złota",
	}

	tests := []struct {
		text   string
		action Action
		item   string
		amount int
	}{
		{"sprzedam kij za 30", ActionSell, "k", 30},
		{"ile dasz za kij?", ActionSell, "k", 0},
		{"kupisz ode mnie kij?", ActionSell, "k", 0},
		{"sprzedam ci coś", ActionSell, "", 0},
		{"ile za amulet", ActionAsk, "a", 0},
	}

	for _, test := range tests {
		request.Text = test.text
		intent := Read(request).Alone()

		if intent.Action != test.action || intent.ItemID != test.item || intent.Amount != test.amount {
			t.Errorf("%q read as %+v", test.text, intent)
		}
	}
}

func TestReadingPicksTheBestNamedWareAndRudeness(t *testing.T) {
	request := InterpretRequest{
		Wares: []Ware{
			{ID: "m", Name: "Mikstura Leczenia", Aliases: Aliases("Mikstura Leczenia")},
			{ID: "w", Name: "Wielka Mikstura Leczenia", Aliases: Aliases("Wielka Mikstura Leczenia")},
			{ID: "i", Name: "Program Impuls", Aliases: Aliases("Program Impuls")},
			{ID: "k", Name: "Program Kaskada", Aliases: Aliases("Program Kaskada")},
		},
		Currency: "szt. złota",
	}

	tests := map[string]string{
		"ile za wielką miksturę leczenia": "w",
		"ile za miksturę leczenia":        "m",
		"pokaż program kaskada":           "k",
		"pokaż mi impuls":                 "i",
		"co to za programy":               "i",
	}

	for text, want := range tests {
		request.Text = text
		if got := Read(request).Alone().ItemID; got != want {
			t.Errorf("%q named %q, want %q", text, got, want)
		}
	}

	request.Text = "zamknij się i podaj cenę, dziadu"
	reading := Read(request)
	if reading.Intent.Persuasion != -2 || reading.Merge(Intent{Persuasion: 2}).Persuasion != -2 {
		t.Errorf("rudeness not read: %+v", reading.Intent)
	}

	// An insult that mentions the wares is still an insult, not a question.
	request.Text = "twój towar to złom, a ty złodziej"
	if reading = Read(request); reading.Intent.Action != ActionChat || reading.Intent.Persuasion != -2 {
		t.Errorf("insult with 'towar' read as %+v", reading.Intent)
	}

	// A short stem does not catch a longer word that happens to begin alike.
	request.Wares = append(request.Wares, Ware{ID: "o", Name: "Kurtka z Opon", Aliases: Aliases("Kurtka z Opon")})
	request.Text = "opowiedz mi coś o sobie"
	if got := Read(request).Alone().ItemID; got != "" {
		t.Errorf("%q named %q", request.Text, got)
	}

	request.Text = "ile za kurtkę z opony"
	if got := Read(request).Alone().ItemID; got != "o" {
		t.Errorf("%q named %q", request.Text, got)
	}
}
