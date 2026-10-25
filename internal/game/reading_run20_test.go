package game

import (
	"slices"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Gaps the run20 data showed (D:\neuro-train\run20\STAN.md, 24.10), each
// with the lines that must not trip the fix.

// A speaker who has a seeded past and a rescue starts a script as a long
// game would have them.
func TestSeedsForScriptsFromRun20(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	if kept, err := realm.SeedBio(hero, "Jagna", []string{"Jagna ma córkę Zosię", "Jagna ma córkę Zosię."}); err != nil || kept != 1 {
		t.Fatalf("seeded %d, %v", kept, err)
	}

	if _, err := realm.SeedBio(hero, "Nikt", []string{"Nikt ma brata."}); err == nil {
		t.Error("a bio seeded for nobody")
	}

	if err := realm.SeedRescue(hero, "Jagna", "Zosia", "córka", "Szczur Polny", "Las", "", "open"); err != nil {
		t.Fatal(err)
	}

	if job := realm.openRescue(1); job == nil || job.subject != "Zosia" || job.taker != hero || job.finished {
		t.Fatalf("job = %+v", job)
	}

	realm.SetDay(5)
	if realm.day != 5 {
		t.Errorf("day = %d", realm.day)
	}

	// Paid before the talk: the keeper has it in her past, and no thanks
	// are due in a talk that opens later.
	paid := testRealm(t)
	ola := enlist(paid, "Ola")
	if err := paid.SeedRescue(ola, "Jagna", "Antek", "syn", "", "", "", "paid"); err != nil {
		t.Fatal(err)
	}

	do(paid, ola, "idź do zielarki")

	_, narrate, _ := paid.Haggle(ola, "co słychać?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Jagna odzyskała syna Antka: uwolniła go Ola.") || strings.Contains(narrate.Lore, "Dziękujesz") {
		t.Errorf("lore after a paid rescue = %q", narrate.Lore)
	}
}

// Freed and paid as the talk opens, the keeper thanks the hero in it.
func TestThanksInThePayingTalkFromRun20(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	if err := realm.SeedRescue(hero, "Jagna", "Paweł", "brat", "", "", "", "freed"); err != nil {
		t.Fatal(err)
	}

	do(realm, hero, "idź do zielarki")

	if job := realm.quests[len(realm.quests)-1]; !job.paid {
		t.Fatalf("not paid as the talk opened: %+v", job)
	}

	_, narrate, _ := realm.Haggle(hero, "dzień dobry", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Gracz uwolniła twojego brata Pawła i odprowadziła go do domu. Dziękujesz z całego serca.") {
		t.Errorf("lore in the paying talk = %q", narrate.Lore)
	}

	// Somebody else who walks in hears no thanks.
	bob := enlist(realm, "Bob")
	do(realm, bob, "idź do zielarki")

	if _, narrate, _ := realm.Haggle(bob, "dzień dobry", Intent{Action: ActionChat}); strings.Contains(narrate.Lore, "Dziękujesz") {
		t.Errorf("thanks to a stranger: %q", narrate.Lore)
	}
}

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

// What a keeper told of their own past is no gap to send the player off
// about.
func TestBioPlaceIsNoGapFromRun20(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.add("Archiwum Zakazane", "Piwnice pod biblioteką, gdzie księgi przykuwa się łańcuchami.", "", 0)
	do(realm, hero, "idź do zielarki")

	_, narrate, _ := realm.Haggle(hero, "a archiwum zakazane? co tam jest?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Hint, "odeślij go do osoby, która zna sprawę: Kowal") {
		t.Fatalf("no referral without a past: %q", narrate.Hint)
	}

	realm.remember("Jagna", "Jagna przez lata sprzątała w Archiwum Zakazanym.", 1, hero)

	_, narrate, _ = realm.Haggle(hero, "a archiwum zakazane? co tam jest?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Hint, "To część twojej przeszłości; już o tym mówiłaś: Jagna przez lata sprzątała w Archiwum Zakazanym.") ||
		strings.Contains(narrate.Hint, "odeślij") || strings.Contains(narrate.Lore, "Archiwum Zakazane (wie o tym") {
		t.Errorf("hint with a past = %q, lore %q", narrate.Hint, narrate.Lore)
	}

	// A past elsewhere leaves the gap as it was.
	other := testRealm(t)
	ola := enlist(other, "Ola")
	other.add("Archiwum Zakazane", "Piwnice pod biblioteką.", "", 0)
	other.remember("Jagna", "Jagna sprzątała w Kolegium Feniksa.", 1, ola)
	do(other, ola, "idź do zielarki")

	if _, narrate, _ := other.Haggle(ola, "a archiwum zakazane? co tam jest?", Intent{Action: ActionChat}); !strings.Contains(narrate.Hint, "odeślij") {
		t.Errorf("a past elsewhere: %q", narrate.Hint)
	}
}

// A name said whole wins over one that only begins like it.
func TestWholeNameWinsFromRun20(t *testing.T) {
	for _, c := range []struct {
		name, query string
		want        bool
	}{
		{"Szczepan Rębacz", "Szczepan", true}, {"Szczepan Rębacz", "rębacza", true}, {"Jędrek Szczęściarz", "jędrka", true},
		{"Jędrek Szczęściarz", "Szczepan", false}, {"Jędrek Szczęściarz", "szczę", false}, {"Bogna Topór", "bogną", true},
	} {
		if got := MatchesWhole(c.name, c.query); got != c.want {
			t.Errorf("MatchesWhole(%q, %q) = %v", c.name, c.query, got)
		}
	}

	realm := partyRealm(t)
	realm.world.Shops[0].Keeper = "Jędrek Szczęściarz"
	realm.world.Companions[0].Name = "Szczepan Rębacz"
	hero := enlist(realm, "Ala")

	for query, want := range map[string]string{"szczepan": "Szczepan Rębacz", "jędrkiem": "Jędrek Szczęściarz", "szczęściarzem": "Jędrek Szczęściarz", "zbrojownia": "Jędrek Szczęściarz", "wieszą": "Wiesza"} {
		realm.CloseShop(hero)
		do(realm, hero, "rozmawiaj z "+query)

		if view := realm.View(hero).Shop; view == nil || view.Keeper != want {
			t.Errorf("rozmawiaj z %s: %+v", query, view)
		}
	}
}

// A missing person is named in any form, sought in the right one, and a
// keeper's kin from the card counts as kin said.
func TestRescueFormsFromRun20(t *testing.T) {
	for name, want := range map[string]string{
		"Paweł": "Pawła", "Rafi": "Rafiego", "Antoni": "Antoniego", "Ali": "Alego", "Antek": "Antka", "Marek": "Marka", "Jacek": "Jacka",
		"Staś": "Stasia", "Maniek": "Mańka", "Michał": "Michała", "Daniel": "Daniela", "Gabriel": "Gabriela", "Wit": "Wita",
	} {
		if got := accusative(name, rescueFeminine(name, "")); got != want {
			t.Errorf("accusative(%q) = %q, want %q", name, got, want)
		}
	}

	for _, c := range []struct {
		text, person string
		want         bool
	}{
		{"co z Antkiem?", "Antek", true}, {"Antka nie ma od tygodnia.", "Antek", true}, {"Pawła porwały topielce.", "Paweł", true},
		{"o wnuka Rafiego się boję", "Rafi", true}, {"Witem się martwię", "Wit", true}, {"Antoni poszedł w las.", "Antek", false},
		{"Marta poszła w las.", "Marek", false}, {"Zosia zaginęła.", "Zofia", false},
	} {
		if got := personNamed(c.text, c.person); got != c.want {
			t.Errorf("personNamed(%q, %q) = %v", c.text, c.person, got)
		}
	}

	if !RescueGrounded(QuestHook{Person: "Antek", Quote: "Antka porwały topielce"}, "Antka porwały topielce w lesie.", nil) {
		t.Error("a declined name was not grounded")
	}

	if kin := cardKin("Była kopaczka. Nie znosi gwizdania, a jej syn Wit kopie w dalekich chodnikach. Ma córkę Zosię i brata, Radka."); !slices.Equal(kin, []string{"Wit", "Zosię", "Radka"}) {
		t.Errorf("kin of the card = %q", kin)
	}

	realm := testRealm(t)
	realm.world.Shops[1].Persona = "Zielarka. Jej syn Wit kopie w dalekich chodnikach."
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zielarki")

	if _, ok := realm.QuestHookRequest(hero, "co z witem?", "[strach] Zszedł do lasu i nie wrócił."); !ok {
		t.Error("the card's son named by the player was not read")
	}

	if _, ok := realm.QuestHookRequest(hero, "co słychać?", "[strach] Wit zszedł do lasu i nie wrócił."); !ok {
		t.Error("the card's son named in the line was not read")
	}

	if _, ok := realm.QuestHookRequest(hero, "co słychać?", "[strach] Bartek zszedł do lasu i nie wrócił."); ok {
		t.Error("a stranger lost was read as kin")
	}

	if _, ok := realm.QuestHookRequest(hero, "co z witem?", "[radość] Kopie dalej, zdrów jak ryba."); ok {
		t.Error("the son safe was read")
	}
}

// A prude bristles at a curse whatever the line is about.
func TestCurseOverWaresFromRun20(t *testing.T) {
	realm := testRealm(t)
	realm.world.Shops[1].Persona = "Zielarka. Nie znosi przekleństw."
	hero := enlist(realm, "Ala")
	say := func(shop, text string) (Decision, NarrateRequest) {
		realm.CloseShop(hero)
		do(realm, hero, "idź do "+shop)
		request, _ := realm.HaggleRequest(hero, text)
		decision, narrate, _ := realm.Haggle(hero, text, Read(request).Alone())

		return decision, narrate
	}

	for _, text := range []string{"o kurwa, ile za napar?", "kurde, co masz na rany?", "cholera, pokaż towar"} {
		decision, narrate := say("zielarki", text)
		if decision.Outcome == OutcomeChat || decision.Mood != MoodAnnoyed || !strings.HasPrefix(narrate.Hint, curseHint) {
			t.Errorf("%q to a prude: %s %s, %q", text, decision.Outcome, decision.Mood, narrate.Hint)
		}

		decision, narrate = say("zbrojowni", text)
		if decision.Mood == MoodAnnoyed || strings.Contains(narrate.Hint, "przeklina") {
			t.Errorf("%q to the smith: %s, %q", text, decision.Mood, narrate.Hint)
		}
	}

	if decision, narrate := say("zielarki", "ile za napar?"); decision.Mood == MoodAnnoyed || strings.Contains(narrate.Hint, "przeklina") {
		t.Errorf("a plain price to a prude: %s, %q", decision.Mood, narrate.Hint)
	}
}

// "Dobra, w drogę" sets off with a companion as "dobra, ruszamy" does.
func TestCompanionSetsOffFromRun20(t *testing.T) {
	for text, want := range map[string]bool{
		"dobra, ruszamy": true, "dobra, jedziemy dalej": true, "dobra, w drogę": true, "w drogę": true, "ruszajmy": true, "czas w drogę": true,
		"jedziemy": true, "dobra, idziemy": true, "w drogę do krypty?": false, "jedziemy na krypty?": false, "gdzie jedziemy": false,
		"przygotuj się w drogę": false, "nie jedziemy": false, "w drogę do Kurhanów idziesz sam": false, "kiedy w drogę": false,
	} {
		if got := moving(text); got != want {
			t.Errorf("moving(%q) = %v", text, got)
		}
	}

	// An insult with it is answered first: the talk goes on.
	realm := partyRealm(t)
	realm.world.Companions[0].Description = "Wścieka się, gdy ktoś nazywa go dzieciakiem."
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	do(realm, hero, "najmij bognę", "rozmawiaj z bogną")

	for _, c := range []struct {
		text string
		want Outcome
	}{{"wstawaj dzieciaku, jedziemy", OutcomeChat}, {"dobra, w drogę", OutcomeFarewell}} {
		request, _ := realm.HaggleRequest(hero, c.text)
		if decision, _, _ := realm.Haggle(hero, c.text, Read(request).Alone()); decision.Outcome != c.want {
			t.Errorf("%q: %s, want %s", c.text, decision.Outcome, c.want)
		}
	}
}
