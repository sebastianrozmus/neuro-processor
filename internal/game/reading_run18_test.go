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
