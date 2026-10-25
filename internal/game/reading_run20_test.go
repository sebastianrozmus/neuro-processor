package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Gaps the run20 data showed (D:\neuro-train\run20\STAN.md, 24.10), each
// with the lines that must not trip the fix.

// A question in one word of a family touches what the speaker said in
// another: siblings and a brother, children and a son, a husband and a
// widow, where from and where born.
func TestKinFamiliesTouchTheBioFromRun20(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	said := func(fact, question string) bool {
		realm.bios = map[string][]*bioFact{}
		realm.remember("Bogna Topór", fact, 1, hero)

		return strings.Contains(realm.bioHint("Bogna Topór", question), "Już o tym mówiłaś")
	}

	for _, pair := range [][2]string{
		{"Bogna ma starszego brata Radka, kowala.", "masz rodzeństwo?"},
		{"Bogna ma młodszą siostrę Dinarę.", "a rodzenstwo jakies masz?"},
		{"Bogna ma starszego brata Radka, kowala.", "a siostrę masz?"},
		{"Bogna ma syna Wita.", "masz dzieci?"},
		{"Bogna ma córkę Zosię.", "ma pani dzieci?"},
		{"Bogna ma syna Wita.", "a syn?"},
		{"Bogna była żoną rybaka, który utonął.", "mąż żyje?"},
		{"Bogna jest wdową po flisaku.", "masz męża?"},
		{"Bogna pochodzi z Wierzbowej Doliny.", "skąd jesteś?"},
		{"Bogna urodziła się w Białej Jurcie.", "skąd pochodzisz?"},
		{"Ojciec Bogny był kowalem.", "twoi rodzice żyją?"},
		{"Bogna ma starszego brata Radka, kowala.", "masz rodzinę?"},
	} {
		if !said(pair[0], pair[1]) {
			t.Errorf("%q did not touch %q", pair[1], pair[0])
		}
	}

	for _, pair := range [][2]string{
		{"Bogna ma starszego brata Radka, kowala.", "skąd jesteś?"},
		{"Bogna ma starszego brata Radka, kowala.", "masz dzieci?"},
		{"Bogna pochodzi z Wierzbowej Doliny.", "masz rodzinę?"},
		{"Bogna ma syna Wita.", "mąż żyje?"},
		{"Bogna ma bliznę od niedźwiedzia.", "masz rodzeństwo?"},
	} {
		if said(pair[0], pair[1]) {
			t.Errorf("%q touched %q", pair[1], pair[0])
		}
	}

	// Grounding stays strict: a sister is no brother said.
	if BioGrounded("Bogna ma siostrę.", "Mam brata, Radka.", "Bogna Topór") || !BioGrounded("Bogna ma syna.", "Syn mi rośnie.", "Bogna Topór") {
		t.Error("kin grounding")
	}
}

// Words of kin a keeper calls the player by, or the player the keeper,
// say nothing of a family.
func TestVocativesAreNoFamilyFromRun20(t *testing.T) {
	for line, want := range map[string]bool{
		"[spokój] Bierz, synu, za 20 kapsli.":                        false,
		"Formuła Iskry za 60 dukatów, dziecko, na mały ogień.":       false,
		"Córko, to najlepszy napar w osadzie.":                       false,
		"Dwadzieścia kapsli, dziecino. Tanio.":                       false,
		"Słuchaj, bracie, miecz jak marzenie.":                       false,
		"Po moim synu został mi tylko ten nóż.":                      true,
		"Mam dziecko, małe jeszcze.":                                 true,
		"Dziecko mi zachorowało, a ty o cenę pytasz.":                true,
		"Mówiłam ci o synu, poszedł na bagna.":                       true,
		"Mój brat też był kowalem, synu.":                            true,
		"Dziecko, mój syn kopie w dalekich chodnikach i nie wrócił.": true,
	} {
		if got := Personal(line, "Kowal"); got != want {
			t.Errorf("Personal(%q) = %v, want %v", line, got, want)
		}
	}

	for text, want := range map[string]bool{
		"synu, ile za miecz?":       false,
		"córko, masz napar?":        false,
		"dziecko, co masz na rany?": false,
		"ile za to, dziecko?":       false,
		"masz dziecko?":             true,
		"a twój syn gdzie?":         true,
		"masz dzieci, dziecko?":     true,
	} {
		if got := Private(text); got != want {
			t.Errorf("Private(%q) = %v, want %v", text, got, want)
		}
	}

	// A trade line with a vocative asks the Mind nothing.
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zielarki")

	if _, ok := realm.SelfNoteRequest(hero, "córko, ile za napar?", "[spokój] Dwadzieścia kapsli, synu."); ok {
		t.Error("a vocative asked the Mind for a bio")
	}

	if _, ok := realm.QuestHookRequest(hero, "synu, masz coś na strach?", "[spokój] Na strach mam napar, dziecko, potwory nie straszne."); ok {
		t.Error("a vocative asked the Mind for a rescue")
	}
}

// Questions about the keeper's body and past are private: told nothing,
// the model gets no leave to make one thing up.
func TestBodyAndPastArePrivateFromRun20(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	customer := realm.players[hero]
	customer.shop = realm.shopOfKind(world.ShopArmory)
	hint := func(text string) string {
		hint, _, _ := realm.hint(customer, text, Decision{Outcome: OutcomeChat}, nil)

		return hint
	}

	for _, text := range []string{
		"a co z tą ręką?", "co się stało z twoją ręką?", "co z twoim kolanem?", "co z twoim okiem?", "czemu ci się trzęsą ręce?", "skąd masz tę szramę?",
		"a jak się naprawdę nazywasz?", "babciu, jak cię nazywali na ringu?", "skąd się tu wzięłaś taka młoda?", "babciu, czemu uciekłaś z góry?",
		"czemu uciekłeś z zenitu?", "jak oślepłeś?", "babciu, spadłaś kiedyś?", "czemu pan prawie nie widzi?", "czemu nie zdejmujesz maski?",
		"jak zmarł pani mąż?", "a gdzie się urodziłeś?",
	} {
		if got := hint(text); !strings.Contains(got, "prywatne sprawy") {
			t.Errorf("%q: hint %q", text, got)
		}
	}

	for _, text := range []string{
		"masz coś na ból ręki?", "co masz na oczy?", "ile za maskę?", "jak przeleciałeś przez oko burzy?", "synu, ile za miecz?", "córko, masz napar?",
		"dziecko, co masz na rany?", "masz coś do ochrony głowy?", "co się stało z twoją barką?", "moja żona umarła wczoraj",
	} {
		if got := hint(text); strings.Contains(got, "prywatne sprawy") {
			t.Errorf("%q: hint %q", text, got)
		}
	}

	if got := hint("moja żona umarła wczoraj"); !strings.Contains(got, "o swojej stracie") {
		t.Errorf("the player's loss: %q", got)
	}
}
