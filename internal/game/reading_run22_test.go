package game

import (
	"context"
	"strings"
	"testing"
)

// Gaps the run21 pairs showed (D:\neuro-train\run22\PLAN.md, 04.11): the
// game kept nothing of what a speaker said of themselves in the first
// talk, so the second one contradicted it. Each fix with the lines that
// must not trip it.

// A private question stays the topic for a line or two: "jak miał na
// imię?" right after "a mąż żyje?" is about the husband, and so is the
// bait after a denial; "ile za napar?" is not.
func TestPrivateQuestionSticksForAFollowUpFromRun22(t *testing.T) {
	realm, hero := bogna(t)
	say := func(text, reply string) string {
		_, narrate, _ := realm.Haggle(hero, text, Intent{Action: ActionChat})
		realm.HaggleRemember(hero, text, reply)

		return narrate.Hint
	}

	if hint := say("masz brata?", "[smutek] Miałam. Nie żyje, dawno."); !strings.Contains(hint, "prywatne sprawy") {
		t.Fatalf("the private question itself: %q", hint)
	}

	// The follow-up, with no word of kin in it, is still private; with
	// nothing kept about the brother it may make nothing up.
	if hint := say("jak miał na imię?", "[smutek] Radek."); !strings.Contains(hint, "prywatne sprawy") || !strings.Contains(hint, "nie wymyślaj krewnych") {
		t.Errorf("the follow-up on the name: %q", hint)
	}

	// With the brother in the bio, the follow-up is told what she said.
	realm.LearnSelf(hero, "Bogna Topór", []string{"Brat Bogny, Radek, nie żyje."}, "Radek.")

	if hint := say("a gdzie teraz?", "[smutek] Na cmentarzu."); !strings.Contains(hint, "Już o tym mówiłaś: Brat Bogny, Radek, nie żyje.") {
		t.Errorf("the follow-up with a bio: %q", hint)
	}

	// Past two lines the topic is gone: the rest of the talk is not about him.
	hint := say("a co z tą kładką?", "[spokój] Zgniła.")
	if strings.Contains(hint, "prywatne sprawy") {
		t.Errorf("a question about the world after the topic: %q", hint)
	}

	// "przykro mi" keeps the topic for one more line.
	say("masz dzieci?", "[smutek] Syna. Zginął na bagnach.")
	if hint := say("przykro mi", "[smutek] Tak już bywa."); strings.Contains(hint, "prywatne sprawy") {
		t.Errorf("condolences briefed as a question: %q", hint)
	}

	if hint := say("ile miał lat?", "[smutek] Dwadzieścia."); !strings.Contains(hint, "prywatne sprawy") {
		t.Errorf("the follow-up past condolences: %q", hint)
	}

	// The bait after a denial the game kept nothing of: no making up.
	say("masz żonę?", "[kpina] Żonę? A kto by ze mną wytrzymał.")
	if hint := say("a gdzie ona teraz?", "[kpina] Kto?"); !strings.Contains(hint, "nie wymyślaj krewnych") {
		t.Errorf("the bait with nothing kept: %q", hint)
	}

	// A question about the world is not the follow-up.
	say("skąd jesteś?", "[spokój] Z bagien.")
	for _, text := range []string{"co słychać na bagnach?", "widziałaś ognika?", "dołączysz do mnie?", "dzięki, to tyle"} {
		if hint := say(text, "[spokój] Nie."); strings.Contains(hint, "prywatne sprawy") {
			t.Errorf("%q briefed as private: %q", text, hint)
		}

		say("skąd jesteś?", "[spokój] Z bagien.")
	}
}

// A shopkeeper asked "ma pani dzieci?" and then "też będzie aptekarzem?"
// is briefed with what she said of her children, as the word of her trade
// is in her card; "ile za napar?" right after is trade.
func TestBaitAfterADenialIsPrivateFromRun22(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zielarki")
	say := func(text string, intent Intent, reply string) (Decision, string) {
		decision, narrate, _ := realm.Haggle(hero, text, intent)
		realm.HaggleRemember(hero, text, reply)

		return decision, narrate.Hint
	}

	say("ma pani dzieci?", Intent{Action: ActionChat}, "[smutek] Dzieci nie było.")
	realm.LearnSelf(hero, "Jagna", []string{"Jagna nie ma dzieci."}, "Dzieci nie było.")

	if _, hint := say("też będzie zielarzem?", Intent{Action: ActionChat}, "[kpina] Kto?"); !strings.Contains(hint, "Już o tym mówiłaś: Jagna nie ma dzieci.") {
		t.Errorf("the bait after the denial: %q", hint)
	}

	say("ma pani dzieci?", Intent{Action: ActionChat}, "[smutek] Dzieci nie było.")
	wares := realm.View(hero).Shop.Wares
	if len(wares) == 0 {
		t.Fatal("no wares")
	}

	if decision, hint := say("ile za "+wares[0].Name+"?", Intent{Action: ActionAsk, ItemID: wares[0].ID}, "[spokój] 10."); decision.Outcome == OutcomeChat || strings.Contains(hint, "prywatne sprawy") {
		t.Errorf("a price asked after a private question: %v, %q", decision.Outcome, hint)
	}
}

// The line after a private question is read for the bio with the question
// as its context, even with no word of kin in it: "Najstarszy chłopak,
// Mściwuj" to "któreś pomaga w kuźni?" after "ile masz dzieci?".
func TestSelfNoteReadsTheLineAfterAPrivateQuestionFromRun22(t *testing.T) {
	realm, hero := bogna(t)
	say := func(text, reply string) {
		realm.Haggle(hero, text, Intent{Action: ActionChat})
		realm.HaggleRemember(hero, text, reply)
	}

	say("ile masz dzieci?", "[kpina] Tyle, ile żelaza w kuźni.")
	say("któreś pomaga w kuźni?", "[spokój] Najstarszy chłopak, Mściwuj. Ma ciężką rękę.")

	request, ok := realm.SelfNoteRequest(hero, "któreś pomaga w kuźni?", "[spokój] Najstarszy chłopak, Mściwuj. Ma ciężką rękę.")
	if !ok || request.Said != "ile masz dzieci? któreś pomaga w kuźni?" {
		t.Fatalf("the line after a private question: %+v, %v", request, ok)
	}

	// Two lines later, and not briefed as a follow-up, the line is on its own.
	say("pokaż kolczugę", "[spokój] Kolczuga grodowa, 210.")
	if _, ok := realm.SelfNoteRequest(hero, "pokaż kolczugę", "[spokój] Kolczuga grodowa, 210."); ok {
		t.Error("a line about wares two lines after a private question was read")
	}
}

// The same fact in fewer words is not new, and in more words it takes the
// shorter one's place: the Mind read "Szczepan ma brata Józka, który rąbie
// drwa pod Nadszybiem" from one line and "… który rąbie drwa" from the next.
func TestTheSameFactInOtherWordsIsOneFactFromRun22(t *testing.T) {
	realm, hero := bogna(t)
	line := "Brata, Józka. Rąbie drwa pod Nadszybiem."

	realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma brata Józka, który rąbie drwa pod Nadszybiem."}, line)
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma brata Józka, który rąbie drwa."}, line); len(kept) != 0 {
		t.Errorf("the same fact in fewer words kept: %q", kept)
	}

	kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma brata Józka, który rąbie drwa pod Nadszybiem i śpiewa."}, line+" I śpiewa.")
	if told := realm.bioTexts("Bogna Topór"); len(kept) != 1 || len(told) != 1 || told[0] != "Bogna ma brata Józka, który rąbie drwa pod Nadszybiem i śpiewa." {
		t.Errorf("the same fact in more words: kept %q, bio %q", kept, told)
	}

	// Another brother is another fact.
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma brata Jana."}, "I brata Jana."); len(kept) != 1 {
		t.Errorf("another brother: %q", kept)
	}
}

// "któreś pomaga w kuźni?" after "ile masz dzieci?" asks on about the
// children.
func TestFollowUpWithAVerbOfKinFromRun22(t *testing.T) {
	realm, hero := bogna(t)
	say := func(text, reply string) string {
		_, narrate, _ := realm.Haggle(hero, text, Intent{Action: ActionChat})
		realm.HaggleRemember(hero, text, reply)

		return narrate.Hint
	}

	say("ile masz dzieci?", "[kpina] Tyle, ile żelaza w kuźni.")
	if hint := say("któreś pomaga w kuźni?", "[spokój] Najstarszy, Zbigniew."); !strings.Contains(hint, "prywatne sprawy") {
		t.Errorf("the follow-up about the children: %q", hint)
	}
}

// Told to quote no amount, the narrator may not echo the player's sum
// either: "biorę bosak i kurtkę za 70" is unclear, and "Za 70?" earns a
// second try. A sum from the lore, no sum, and a sum the decision quotes
// elsewhere are as before.
func TestNoAmountsMeansNotEvenThePlayersFromRun22(t *testing.T) {
	unclear := NarrateRequest{Decision: Decision{Outcome: OutcomeUnclear, Offer: 70}, Lore: "«Bosak Strażacki» 40, «Kurtka z Pontonu» 35"}

	narrator := &scriptedNarrator{replies: []string{"[zniecierpliwienie] Za 70? Najpierw powiedz wyraźnie, co bierzesz.", "[zniecierpliwienie] Bosak czy kurtka? Powiedz wprost."}}
	if text, err := Narrate(context.Background(), narrator, unclear); err != nil || narrator.calls != 2 || !strings.HasSuffix(text, "Powiedz wprost.") {
		t.Errorf("the player's sum echoed: %q, %v, %d calls", text, err, narrator.calls)
	}

	narrator = &scriptedNarrator{replies: []string{"[zniecierpliwienie] Bosak za 40 czy kurtka za 35? Powiedz wprost."}}
	if _, err := Narrate(context.Background(), narrator, unclear); err == nil {
		t.Error("prices from the list quoted when told to quote none")
	}

	narrator = &scriptedNarrator{replies: []string{"[zniecierpliwienie] Bosak czy kurtka? Powiedz wprost."}}
	if text, err := Narrate(context.Background(), narrator, unclear); err != nil || narrator.calls != 1 {
		t.Errorf("a line without a sum: %q, %v", text, err)
	}

	// A counter-offer still quotes its price.
	narrator = &scriptedNarrator{replies: []string{"[spokój] Za 25? Daj 36."}}
	if text, err := Narrate(context.Background(), narrator, NarrateRequest{Decision: Decision{Outcome: OutcomeCounter, Item: Ware{Name: "Bosak"}, Offer: 25, Price: 36}}); err != nil || narrator.calls != 1 {
		t.Errorf("a counter-offer: %q, %v", text, err)
	}
}

// A fact of the kin the question asked about is grounded in the question:
// "Nie żyje, dawno" to "a mąż żyje?" makes her a widow. Kin the question
// never asked about are not: the daughter Zofia the Mind made up.
func TestFactsGroundedInThePlayersQuestionFromRun22(t *testing.T) {
	realm, hero := bogna(t)
	line := "[kpina] A co, szukasz mi swata? Nie, dawno. A ja dalej robię miód."

	realm.Haggle(hero, "a mąż żyje?", Intent{Action: ActionChat})
	realm.HaggleRemember(hero, "a mąż żyje?", line)
	if _, ok := realm.SelfNoteRequest(hero, "a mąż żyje?", line); !ok {
		t.Fatal("the answer to a private question was not read")
	}

	kept := realm.LearnSelf(hero, "Bogna Topór", []string{
		"Bogna jest wdową.",
		"Bogna ma córkę Zofię, która mieszka w sąsiedniej wsi.",
		"Bogna pochodzi z rodziny górników.",
	}, line)
	if len(kept) != 1 || kept[0] != "Bogna jest wdową." {
		t.Errorf("kept = %q", kept)
	}

	// Grandchildren, a sweetheart and a bachelor are kin a question asks
	// about: "masz chłopaka?" – "Nie." is a fact.
	for _, c := range []struct{ question, fact string }{
		{"masz chłopaka?", "Bogna nie ma chłopaka."},
		{"ma pani wnuki?", "Bogna nie ma wnuków."},
		{"jest pani zamężna?", "Bogna nie ma męża."},
		{"masz żonę?", "Jędrek jest kawalerem."},
		{"masz narzeczonego?", "Bogna ma narzeczonego Radka."},
	} {
		if !Private(c.question) {
			t.Errorf("%q is no private question", c.question)
		}

		if !kinAsked(c.question, c.fact) {
			t.Errorf("kinAsked(%q, %q) = false", c.question, c.fact)
		}
	}

	if kinAsked("masz chłopaka?", "Bogna ma brata Radka.") {
		t.Error("a brother answers a question about a sweetheart")
	}

	// A line the Mind read for another question is grounded in the line alone.
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna była mężatką."}, "Nie twoja sprawa."); len(kept) != 0 {
		t.Errorf("a fact of another line grounded in the question: %q", kept)
	}
}
