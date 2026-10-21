package game

import (
	"strings"
	"testing"
)

func TestShortWordNamesNoLongerWare(t *testing.T) {
	wares := []Ware{{ID: "big", Name: "Wielka Mikstura Leczenia"}, {ID: "bow", Name: "Łuk"}}

	if got := namedItem(Fold("skad wiesz ze on wie"), wares); got != "" {
		t.Errorf("namedItem = %q, want none", got)
	}
	if got := namedItem(Fold("pokaz luk"), wares); got != "bow" {
		t.Errorf("namedItem = %q, want bow", got)
	}
}

func TestOldWomanIsNoOldForest(t *testing.T) {
	for _, text := range []string{"milcz starucho", "starczy ci odwagi"} {
		if touches(strings.Fields(Fold(text)), "Stary Bór") > 0 {
			t.Errorf("%q touches Stary Bór", text)
		}
	}

	if !Read(InterpretRequest{Text: "milcz starucho"}).rude {
		t.Error("an old crone is no insult")
	}
}

func TestOldDwarfIsNoOldForest(t *testing.T) {
	if touches(strings.Fields(Fold("ty stary kurduplu")), "Stary Bór") > 0 {
		t.Error("an old dwarf touches Stary Bór")
	}

	if touches(strings.Fields(Fold("Pola Karw czy Stary Bór?")), "Stary Bór") == 0 {
		t.Error("Stary Bór named outright is not touched")
	}

	for _, text := range []string{"ty stary kurduplu", "jesteś bezużyteczna w walce"} {
		if !Read(InterpretRequest{Text: text}).rude {
			t.Errorf("%q is no insult", text)
		}
	}
}

func TestSessionWordsFromTest13(t *testing.T) {
	if !Read(InterpretRequest{Text: "podpalę ci wieżę"}).rude {
		t.Error("burning the tower down is no threat")
	}

	if Read(InterpretRequest{Text: "no dobra, sorry"}).Intent.Action != ActionChat {
		t.Error("an apology is a deal")
	}

	if Read(InterpretRequest{Text: "dobra, idę sobie"}).Intent.Action != ActionLeave {
		t.Error("going away is not leaving")
	}

	if !greeting(strings.Fields(Fold("Dzień dobry magu."))) {
		t.Error("a greeting with an address is no greeting")
	}

	if greeting(strings.Fields(Fold("dzień dobry, masz miecz"))) {
		t.Error("a question about a sword is a greeting")
	}

	if leads("Przełęcz Wichrów", "przepraszam") || !leads("Przełęcz Wichrów", "przeleczy") || !leads("Ogniste Widmo", "widma") || leads("Stary Bór", "borin") || leads("Zmutowany Dzik", "dzien") || !leads("Zmutowany Dzik", "dzika") {
		t.Error("an apology leads to the pass, or the pass does not")
	}

	if !asked("a co potrafisz w walce") || !asked("co potrafisz w walce") || !asked("ok ok, przepraszam. ile bierzesz") || asked("dobra, idziemy") {
		t.Error("a question without a mark is not asked")
	}
}

func TestHalfThePriceIsAnOffer(t *testing.T) {
	wares := []Ware{{ID: "umowa", Name: "Umowa najmu", Price: 130}, {ID: "harpun", Name: "Harpun", Price: 40}}

	for text, want := range map[string]int{"drogo. dam połowę": 65, "dam pół ceny": 65, "połowa za harpun": 20} {
		got := Read(InterpretRequest{Text: text, Wares: wares, FocusID: "umowa"}).Alone()
		if got.Action != ActionOffer || got.Amount != want {
			t.Errorf("%q = %+v, want an offer of %d", text, got, want)
		}
	}

	// Without a ware in view half of nothing is no offer.
	if got := Read(InterpretRequest{Text: "dam połowę", Wares: wares}).Alone(); got.Action == ActionOffer {
		t.Errorf("half of what: %+v", got)
	}
}
