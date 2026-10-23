package game

import (
	"fmt"
	"strings"
	"testing"
)

func TestPersonalCues(t *testing.T) {
	for line, want := range map[string]bool{
		"[smutek] Mam starszego brata, został w kuźni.":          true,
		"Moja matka piekła najlepszy chleb w osadzie.":           true,
		"Tę bliznę mam od niedźwiedzia.":                         true,
		"Urodziłam się w wiosce nad rzeką.":                      true,
		"Pochodzę z gór, tam zimy są długie.":                    true,
		"Służyłem w straży, dziesięć lat temu.":                  true,
		"Stary Wit uczył mnie miecza, kiedy byłem mały.":         true,
		"Za młodu kradłam jabłka.":                               true,
		"Moja wieś spłonęła.":                                    true,
		"Żona czeka na mnie w domu.":                             true,
		"Wiesz, jak jest. Bagna nie wybaczają.":                  false,
		"Kurhany leżą na północy, idź ostrożnie.":                false,
		"Mamy przed sobą długą drogę, ruszajmy.":                 false,
		"Szczury na bagnach są wielkie jak psy.":                 false,
		"Siadaj. O cenie pogadamy, kiedy powiesz, czego chcesz.": false,
	} {
		if got := Personal(line, "Bogna Topór"); got != want {
			t.Errorf("Personal(%q) = %v, want %v", line, got, want)
		}
	}

	// A name is no cue: Brat Jaromir saying his name says nothing of a brother.
	if Personal("Brat Jaromir zawsze pomoże rannym.", "Brat Jaromir") {
		t.Error("the speaker's own name was taken for family")
	}

	if !Personal("Mój brat też był mnichem.", "Brat Jaromir") {
		t.Error("a brother of Brat Jaromir was missed")
	}
}

func TestBioGrounded(t *testing.T) {
	line := "Mam starszego brata, Radka. Kowal z niego, łapy jak bochny."

	for fact, want := range map[string]bool{
		"Bogna ma starszego brata Radka.": true,
		"Brat Bogny, Radek, jest kowalem": true,
		"Bogna ma siostrę Jadwigę.":       false,
		"Bogna Topór pochodzi z gór.":     false,
	} {
		if got := BioGrounded(fact, line, "Bogna Topór"); got != want {
			t.Errorf("BioGrounded(%q) = %v, want %v", fact, got, want)
		}
	}

	if !BioGrounded("Bogna ma 34 lata.", "Mam 34 wiosny na karku.", "Bogna Topór") {
		t.Error("a number said was not grounded")
	}
}

// bogna sits a hero down with Bogna Topór at the inn.
func bogna(t *testing.T) (*Realm, string) {
	t.Helper()

	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "rozmawiaj z bogną")

	if view := realm.View(hero).Shop; view == nil || view.Keeper != "Bogna Topór" {
		t.Fatalf("no talk with Bogna: %+v", view)
	}

	return realm, hero
}

func TestSelfNoteAsksOnlyCompanionsAboutTheirLife(t *testing.T) {
	realm, hero := bogna(t)

	request, ok := realm.SelfNoteRequest(hero, "co słychać?", "[smutek] Mam brata, Radka, kowala.")
	if !ok || request.Speaker != "Bogna Topór" || !strings.Contains(request.Card, "Bije mocno.") || request.Reply != "Mam brata, Radka, kowala." || len(request.Known) != 0 {
		t.Fatalf("request = %+v, %v", request, ok)
	}

	// A private question is enough, whatever the answer sounds like.
	if _, ok := realm.SelfNoteRequest(hero, "skąd jesteś?", "Z daleka. Nie twoja sprawa."); !ok {
		t.Error("an answer to a private question was not read")
	}

	if _, ok := realm.SelfNoteRequest(hero, "co słychać?", "Bagna dziś spokojne, ruszajmy."); ok {
		t.Error("a line about the world was read as one about herself")
	}

	// A shopkeeper's life is not noted this way.
	shop := realm.players[hero]
	realm.mu.Lock()
	shop.shop = 0
	realm.mu.Unlock()

	if _, ok := realm.SelfNoteRequest(hero, "masz dzieci?", "Mam córkę, Zosię."); ok {
		t.Error("a shopkeeper's line was read for a bio")
	}
}

func TestLearnSelfKeepsGroundedFactsOnce(t *testing.T) {
	realm, hero := bogna(t)
	line := "[spokój] Mam starszego brata, Radka. A bliznę mam od niedźwiedzia, Ala."

	kept := realm.LearnSelf(hero, "Bogna Topór", []string{
		"Bogna ma starszego brata Radka",
		"Bogna ma bliznę od niedźwiedzia.",
		"Bogna ma siostrę Jadwigę.",           // not in the line
		"Bogna ma bliznę od niedźwiedzia, 2.", // past the three a line may add
	}, line)
	if len(kept) != 2 || kept[0] != "Bogna ma starszego brata Radka." || kept[1] != "Bogna ma bliznę od niedźwiedzia." {
		t.Fatalf("kept = %q", kept)
	}

	if !logged(realm, hero, "Zapamiętujesz: Bogna Topór o sobie: Bogna ma starszego brata Radka.") {
		t.Errorf("the player was not told: %v", realm.View(hero).Log)
	}

	// The same thing said again, in other case and punctuation, is not new.
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"bogna ma starszego brata, Radka!"}, line); len(kept) != 0 {
		t.Errorf("a fact kept twice: %q", kept)
	}

	// What she thinks of the player is no part of her life story.
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna lubi niedźwiedzie, a Ala jej płaci."}, line); len(kept) != 0 {
		t.Errorf("a fact about the player kept: %q", kept)
	}

	// Money and the hire are no part of a life story.
	if kept := realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna bierze 60 kapsli za brata."}, "Za brata wezmę 60 kapsli."); len(kept) != 0 {
		t.Errorf("a price kept as a fact: %q", kept)
	}

	// Someone who is no companion has no bio.
	if kept := realm.LearnSelf(hero, "Nikt", []string{"Nikt ma brata."}, "Mam brata."); len(kept) != 0 {
		t.Errorf("a stranger's bio: %q", kept)
	}

	// The journal has it.
	journal := JournalLines(realm.Journal(hero))
	if !strings.Contains(strings.Join(journal, "\n"), "• Bogna Topór o sobie: Bogna ma starszego brata Radka.") {
		t.Errorf("journal = %q", journal)
	}

	// Past the cap the oldest stay, and nothing more is asked.
	for index := range maxBio + 3 {
		realm.LearnSelf(hero, "Bogna Topór", []string{fmt.Sprintf("Bogna ma kozę numer %d.", index)}, fmt.Sprintf("Mam kozę numer %d.", index))
	}

	if told := realm.bioTexts("Bogna Topór"); len(told) != maxBio || told[0] != "Bogna ma starszego brata Radka." {
		t.Errorf("after the cap: %d, %q", len(told), told)
	}

	if _, ok := realm.SelfNoteRequest(hero, "masz brata?", "Mam, Radka."); ok {
		t.Error("a full bio still asks the Mind")
	}
}

func TestCompanionsKeepToWhatTheySaid(t *testing.T) {
	realm, hero := bogna(t)

	// Nothing said yet: she may make one thing up.
	_, narrate, _ := realm.Haggle(hero, "masz rodzeństwo?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Hint, "prywatne sprawy") || !strings.Contains(narrate.Hint, "możesz wymyślić jedną konkretną rzecz") || strings.Contains(narrate.Hint, "nie wymyślaj") {
		t.Errorf("hint with no bio = %q", narrate.Hint)
	}

	if strings.Contains(narrate.Lore, "Co już o sobie") {
		t.Errorf("an empty bio in the lore: %q", narrate.Lore)
	}

	realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma starszego brata Radka, kowala."}, "Mam starszego brata, Radka. Kowal.")

	// Said once, it is in the card of every later talk.
	_, narrate, _ = realm.Haggle(hero, "co umiesz?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Co już o sobie mówiłaś (to prawda, trzymaj się tego): Bogna ma starszego brata Radka, kowala.\n") {
		t.Errorf("lore = %q", narrate.Lore)
	}

	// Asked about it, she keeps to it.
	_, narrate, _ = realm.Haggle(hero, "a twój brat gdzie teraz?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Hint, "Już o tym mówiłaś: Bogna ma starszego brata Radka, kowala. Trzymaj się tego") {
		t.Errorf("hint about the brother = %q", narrate.Hint)
	}

	// Asked about something else, she may make that up, but not against the rest.
	_, narrate, _ = realm.Haggle(hero, "skąd masz bliznę?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Hint, "możesz wymyślić jedną") || !strings.HasSuffix(narrate.Hint, "Nie zaprzeczaj temu, co już o sobie mówiłaś.") {
		t.Errorf("hint about the scar = %q", narrate.Hint)
	}

	// A shopkeeper still says only what is in the card.
	if hint, _ := aside("masz dzieci?"); hint != privateHint || !strings.Contains(hint, "nie wymyślaj krewnych") {
		t.Errorf("a shopkeeper's private hint = %q", hint)
	}
}

func TestBioLastsThroughASave(t *testing.T) {
	realm, hero := bogna(t)
	realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma starszego brata Radka."}, "Mam starszego brata, Radka.")

	state := realm.State(hero)
	if len(state.Bios) != 1 || state.Bios[0].Name != "Bogna Topór" || state.Bios[0].Heard[0] != hero {
		t.Fatalf("saved bios = %+v", state.Bios)
	}

	restored, err := Restore(state, 1)
	if err != nil {
		t.Fatal(err)
	}

	if told := restored.bioTexts("Bogna Topór"); len(told) != 1 || told[0] != "Bogna ma starszego brata Radka." {
		t.Errorf("restored bio = %q", told)
	}

	if journal := restored.bioJournal(hero); len(journal) != 1 || journal[0].Subject != "Bogna Topór o sobie" {
		t.Errorf("restored journal = %+v", journal)
	}

	// A save from before bios loads with none.
	state.Bios = nil
	if restored, err = Restore(state, 1); err != nil || len(restored.bios["Bogna Topór"]) != 0 {
		t.Errorf("an old save: %v, %v", err, restored.bios)
	}
}
