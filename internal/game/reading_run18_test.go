package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The cases come from the talk packs of run 18: lines the rules read
// otherwise than the people who wrote them meant.

// A word that only begins like a name is not the name: "dzieci" are no
// "Żołd dzienny", "kawał" no "Kawa", "służbę" no "Śluza".
func TestLooseWordsNameNothingFromRun18(t *testing.T) {
	wares := []Ware{
		{ID: "wage", Name: "Żołd dzienny", Price: 65}, {ID: "coffee", Name: "Czarna Kawa Turecka", Price: 12},
		{ID: "feather", Name: "Pióro Sokoła", Price: 90}, {ID: "mail", Name: "Kolczuga", Price: 80}, {ID: "sabre", Name: "Szabla Hajducka", Price: 70},
		{ID: "compass", Name: "Kompas Burzowy", Price: 60}, {ID: "dew", Name: "Krople Chmurnej Rosy", Price: 20},
	}

	for text, want := range map[string]string{
		"masz dzieci?": "", "opowiedz jakis kawal": "", "120 i ani piora wiecej": "", "czemu kulejesz? byles hajdukiem?": "", "bylas w oku burzy?": "", "a co jest pod chmurami?": "",
		"pokaz szable": "sabre", "kompas burzowy?": "compass",
		"kawy poprosze": "coffee", "ile za pioro?": "feather", "pokaz kolczuge": "mail", "co z ta kolczuga": "mail", "ile za zold?": "wage",
	} {
		if got := Read(InterpretRequest{Text: text, Wares: wares, Currency: "piór"}).Intent.ItemID; got != want {
			t.Errorf("%q names %q, want %q", text, got, want)
		}
	}

	if got := Read(InterpretRequest{Text: "masz dzieci?", Wares: wares}).Intent.Action; got == ActionAsk {
		t.Error("„masz dzieci?” asks the price of the service")
	}

	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	herbalist, armory := realm.shopOfKind(world.ShopAlchemist), realm.shopOfKind(world.ShopArmory)
	for _, subject := range []string{"Śluza Główna", "Chodniki Minerskie", "Hospodar Strzygoń", "Dzwonnik bez Głowy", "Oskar Bez Kasty", "Popielne Zbocza"} {
		realm.add(subject, "Coś, o czym się mówi.", "", herbalist)
	}

	realm.add("Obsydianowa Grota", "Grota pod wulkanem.", "", armory)

	customer := realm.players[hero]
	customer.shop = armory
	hint := func(text string) string {
		hint, _, _ := realm.hint(customer, text, Decision{Outcome: OutcomeChat}, nil)

		return hint
	}

	for text, wrong := range map[string]string{
		"ile bierzesz za sluzbe?":          "Śluza",
		"to chodz":                         "Chodniki",
		"widzialas kiedys strzyge?":        "Strzygoń",
		"pokaz co masz na glowe":           "Dzwonnik",
		"a reszta twojej kasty gdzie?":     "Oskar",
		"widzialas popielnego upiora?":     "Popielne",
		"dobra, to obsydianowy miecz ile?": "wspomina o: Obsydianowa",
	} {
		if got := hint(text); strings.Contains(got, wrong) {
			t.Errorf("%q: hint %q", text, got)
		}
	}

	// The names said in their own forms are still the names.
	for text, want := range map[string]string{
		"co wiesz o sluzie glownej?":     "Śluza Główna",
		"widzialas hospodara strzygonia": "Hospodar Strzygoń",
		"a dzwonnik?":                    "Dzwonnik bez Głowy",
		"bylas w obsydianowej grocie?":   "Obsydianowa Grota",
	} {
		if got := hint(text); !strings.Contains(got, want) {
			t.Errorf("%q: hint %q, want %s", text, got, want)
		}
	}
}

// A companion's own past is a place their card names by its head word:
// the "Siwy tropiciel" never was in the Siwy Żleb, and the magister's
// runaway pupil has no past in the magister's workshop for the word.
func TestOwnPastNeedsTheHeadWordFromRun18(t *testing.T) {
	realm := partyRealm(t)
	mate := world.CompanionSpec{Name: "Wiktor", Description: "Siwy tropiciel. Uciekł z kopalni, kiedy zawaliła się sztolnia."}

	for subject, want := range map[string]bool{"Siwy Żleb": false, "Kopalnia Siwy Żleb": true, "Pracownia Zbiegłego Magistra": false} {
		if got := realm.ownPast(mate, &Fact{Subject: subject}); got != want {
			t.Errorf("own past %q = %v, want %v", subject, got, want)
		}
	}

	mate.Description = "Terminował u magistra, aż ten uciekł."
	if realm.ownPast(mate, &Fact{Subject: "Pracownia Zbiegłego Magistra"}) {
		t.Error("the magister is the past, not his workshop")
	}
}

// A count said in words is a count: "dwa za 20?", "dwie lunety za 100?".
func TestSpelledCountsFromRun18(t *testing.T) {
	wares := []Ware{
		{ID: "glass", Name: "Luneta", Price: 60}, {ID: "vinegar", Name: "Ocet Siedmiu Złodziei", Price: 15}, {ID: "tea", Name: "Herbata z Chmur", Price: 15},
		{ID: "coat", Name: "Kaftan", Price: 25}, {ID: "hook", Name: "Bosak", Price: 30}, {ID: "jacket", Name: "Kurtka", Price: 45},
	}

	for _, test := range []struct {
		text, focus, item string
		quantity, amount  int
	}{
		{"dwa za 20?", "vinegar", "", 2, 20},
		{"a dwa za 20?", "vinegar", "", 2, 20},
		{"dwie lunety za 100?", "", "glass", 2, 100},
		{"dwa octy za 20", "", "vinegar", 2, 20},
		{"trzy herbaty za 30", "", "tea", 3, 30},
		{"dwie herbaty za 30", "", "tea", 2, 30},
		{"to trzy za 35", "tea", "", 3, 35},
		{"dwa kaftany za 40?", "", "coat", 2, 40},
		{"2 sztuki za 20", "vinegar", "", 2, 20},
	} {
		reading := Read(InterpretRequest{Text: test.text, Wares: wares, FocusID: test.focus})
		got := reading.Alone()
		if got.Quantity != test.quantity || got.Amount != test.amount || got.ItemID != test.item || got.Action != ActionOffer {
			t.Errorf("%q = %+v, want %d pieces for %d", test.text, got, test.quantity, test.amount)
		}
	}

	// Two wares for one sum are not one ware: the keeper asks which.
	for _, text := range []string{"biore bosak i kurtke za 70", "kurtka i bosak razem za 70"} {
		if got := Read(InterpretRequest{Text: text, Wares: wares}).Alone(); got.Action != ActionOffer || got.Amount != 0 || got.ItemID != "" {
			t.Errorf("%q = %+v, want an unclear offer", text, got)
		}
	}

	session := NewSession(Merchant{Items: []Item{{ID: "hook", Name: "Bosak", ListPrice: 30, MinPrice: 20}, {ID: "jacket", Name: "Kurtka", ListPrice: 45, MinPrice: 30}}}, 200)
	session.Apply(Intent{Action: ActionAsk, ItemID: "jacket"})
	if got := session.Apply(Read(InterpretRequest{Text: "biore bosak i kurtke za 70", Wares: session.Wares(), FocusID: session.Focus()}).Alone()); got.Outcome != OutcomeUnclear {
		t.Errorf("two wares for one sum: %+v", got)
	}
}
