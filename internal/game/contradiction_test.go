package game

import (
	"context"
	"strings"
	"testing"
)

// The contradictions the judges counted on the test20 pairs (run21 and
// run20v3, 04.11), and the lines about kin that are no contradiction.
func TestContradictsTheBioAboutKin(t *testing.T) {
	for _, c := range []struct {
		bio, line string
		want      bool
	}{
		// a husband dead, then never married (wieslawa s7, both models)
		{"Mąż Wiesławy nie żyje.", "[kpina] Męża nie mam, dziecinko. Żadnego nie było.", true},
		{"Mąż Wiesławy nie żyje od dawna.", "[kpina] Żadnego męża nie było, dziecinko.", true},
		// a husband alive, then none (wieslawa s11 run21)
		{"Mąż Wiesławy żyje i siedzi w karczmie.", "[kpina] Ja nie miałam żadnego męża, dziecinko.", true},
		// a husband alive, then dead (wieslawa s11 run20v3)
		{"Mąż Wiesławy żyje i tnie drwa w puszczy.", "[smutek] Mąż? Zmarł dawno, dziecinko, kto by go pamiętał.", true},
		// no children, then a son (benigna s7)
		{"Benigna nie ma dzieci ani wnuków.", "[smutek] Syn został na górze, nie zjeżdża.", true},
		{"Benigna nie ma dzieci.", "[smutek] Mój syn też będzie aptekarzem.", true},
		{"Benigna nie ma dzieci.", "[smutek] Syna? On nie zejdzie na dół, córko.", true},
		{"Jędrek ma żonę Hanię.", "[radość] Żonę? A skąd, kochana! Ja mam amulety.", true},
		{"Jędrek nie ma żony.", "[radość] Żonę? A skąd, kochana! Ja mam amulety.", false},
		// a son, then none (mszczuj s11 run21)
		{"Najstarszy syn Mszczuja, Mściwuj, pomaga w kuźni.", "[kpina] Żadne. Moje dzieci pilnują kozy.", false},
		{"Najstarszy syn Mszczuja, Mściwuj, pomaga w kuźni.", "[kpina] Dzieci nie mam, kowadło mi za rodzinę.", true},
		// a brother renamed (szczepan s11)
		{"Szczepan ma brata Jana, który rąbie drwa.", "[smutek] Mam brata, Michała. Rąbie drwa.", true},
		{"Szczepan ma brata Józka, który rąbie drwa pod Nadszybiem.", "[spokój] Józek rąbie drwa, a ja szyby.", false},
		{"Szczepan ma brata Józka.", "[smutek] Brat Józek śpiewa głośniej niż ja.", false},
		// no siblings, then a brother
		{"Szczepan nie ma rodzeństwa.", "[smutek] Brat zjechał na dół i tyle go widziałem.", true},

		// no contradiction: the same thing in other words
		{"Mąż Wiesławy nie żyje.", "[smutek] Nie mam już męża, dziecinko. Dawno zszedł.", false},
		{"Mąż Wiesławy nie żyje.", "[smutek] Mąż zginął pod sosną, dwadzieścia lat temu.", false},
		{"Benigna nie ma dzieci.", "[smutek] Dzieci nie było, córko. Same słoiki.", false},
		{"Benigna nie ma dzieci.", "[smutek] Nie, dzieci nie mam, panienko, i nie miałam.", false},
		{"Na dzieci nie starczyło Benignie już sił.", "[kpina] Nie będzie żadnego syna, bo syna też nie mam.", false},
		{"Miałam męża, sztygara, a dzieci nie.", "[smutek] Syn został na górze.", true},
		{"Benigna nie ma dzieci ani wnuków.", "[smutek] Nie ma kogo uczyć, syn został na górze.", true},
		{"Benigna nie ma dzieci ani wnuków.", "[smutek] Nie mam ani dzieci, ani wnuków. Syna nie było.", false},
		{"Benigna nie ma dzieci.", "[kpina] Syna? Jakiego syna? Nie mam żadnego.", false},
		{"Benigna nie ma dzieci.", "[smutek] Gdybym miała syna, też by zjeżdżał.", false},
		{"Benigna nie ma dzieci.", "[czułość] Traktuję cię jak córkę, dziecko.", false},
		{"Benigna nie ma dzieci.", "[spokój] Twój syn kaszle? Solanka z Miodem, 15 groszy.", false},
		{"Szczepan ma brata Józka.", "[smutek] Siostry nie mam, tylko brata.", false},
		{"Szczepan ma brata Józka.", "[smutek] Brata, Józka. Rąbie drwa pod Nadszybiem.", false},
		{"Bogna ma starszego brata Radka, kowala.", "[spokój] Radek bije młotem, ja toporem.", false},
		{"Wit pochodzi z Barciowej Osady.", "[radość] Z Polany Wrzosowej nie jestem, z Barciowej!", false},
		{"Ludmiła ma bliznę po pazurze rysia.", "[kpina] Z lasu, jak każda. Nie mam rodziny, mam puszczę.", false},
		{"Jagna ma córkę Zosię, która nurkuje po skarby.", "[czułość] Zosia nurkuje, a ja się boję.", false},
		{"Jagna ma córkę Zosię.", "[smutek] Córka? Zosia została w grodzie.", false},
		// a brother dead since is a change, not a slip
		{"Szczepan ma brata Józka.", "[smutek] Józek zginął w zeszłym roku.", false},
	} {
		got := contradicts(c.line, []string{c.bio}, "Ktoś Taki")
		if (got != "") != c.want {
			t.Errorf("contradicts(%q | %q) = %q, want %v", c.line, c.bio, got, c.want)
		}
	}

	// The speaker's own name is no kin: Babka Wiesława tells of no grandmother.
	if got := contradicts("[radość] Babka Wiesława nie ma czasu na gadanie.", []string{"Babka Wiesława nie ma dzieci."}, "Babka Wiesława"); got != "" {
		t.Errorf("the speaker's own name read as kin: %q", got)
	}
}

// A line that contradicts the bio earns the narrator a second try with
// the fact it contradicted; two such tries still speak, as a canned line
// would be worse.
func TestNarrateRetriesALineThatDeniesTheBio(t *testing.T) {
	narrator := &scriptedNarrator{replies: []string{"[kpina] Męża nie mam, żadnego nie było.", "[smutek] Mąż nie żyje, dawno."}}
	request := NarrateRequest{Merchant: Merchant{Name: "Babka Wiesława"}, Decision: Decision{Outcome: OutcomeChat}, Bio: []string{"Mąż Wiesławy nie żyje."}}

	text, err := Narrate(context.Background(), narrator, request)
	if err != nil || text != "[smutek] Mąż nie żyje, dawno." || narrator.calls != 2 {
		t.Fatalf("text = %q, err = %v, calls = %d", text, err, narrator.calls)
	}

	if !strings.Contains(narrator.corrections[1], "przeczy temu, co już o sobie mówiłaś: „Mąż Wiesławy nie żyje.”") {
		t.Errorf("correction = %q", narrator.corrections[1])
	}

	narrator = &scriptedNarrator{replies: []string{"[kpina] Męża nie mam, żadnego nie było.", "[kpina] Nie było męża i tyle."}}
	text, err = Narrate(context.Background(), narrator, request)
	if text != "[kpina] Nie było męża i tyle." || err == nil {
		t.Errorf("after two denials: %q, %v", text, err)
	}

	// With no bio, nothing is checked.
	narrator = &scriptedNarrator{replies: []string{"[kpina] Męża nie mam, żadnego nie było."}}
	request.Bio = nil
	if text, err := Narrate(context.Background(), narrator, request); err != nil || narrator.calls != 1 {
		t.Errorf("without a bio: %q, %v, %d calls", text, err, narrator.calls)
	}
}

// The bio goes to the narrator with the request of a companion's talk.
func TestNarrateRequestCarriesTheBio(t *testing.T) {
	realm, hero := bogna(t)
	realm.LearnSelf(hero, "Bogna Topór", []string{"Bogna ma starszego brata Radka."}, "Mam starszego brata, Radka.")

	if _, narrate, _ := realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat}); len(narrate.Bio) != 1 || narrate.Bio[0] != "Bogna ma starszego brata Radka." {
		t.Errorf("bio in the request = %q", narrate.Bio)
	}
}
