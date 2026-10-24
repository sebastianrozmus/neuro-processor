package game

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Rounds from test18 (run18), as the master got them.
const (
	roundSix = "Szczur Rynnowy podchodzi. Topielica z Tramwaju atakuje: Ludmiła Harpun traci 4 zdrowia. " +
		"Ludmiła Harpun atakuje: Szczur Rynnowy. Pudło. Szczur Rynnowy atakuje: Ludmiła Harpun. Pudło. " +
		"Ola atakuje: Topielica z Tramwaju traci 12 zdrowia. Chmara Komarów atakuje: Ola traci 9 zdrowia."
	roundFallen = "Nurek bez Twarzy zrasta się: odzyskuje 4 zdrowia. Mieczysław atakuje: Nurek bez Twarzy. Pudło. " +
		"Czarna Tonia atakuje: Nurek bez Twarzy traci 7 zdrowia. Nurek bez Twarzy atakuje: Czarna Tonia. Pudło. " +
		"Rak Kanałowy atakuje: Czarna Tonia traci 11 zdrowia. Czarna Tonia pada nieprzytomna. Odpoczynek w osadzie postawi ją na nogi."
	roundWon = "Topielica z Tramwaju zrasta się: odzyskuje 4 zdrowia. Wanda próbuje uciec, bez skutku. " +
		"Topielica z Tramwaju atakuje: Ludmiła Harpun. Pudło. Ludmiła Harpun atakuje: Topielica z Tramwaju traci 7 zdrowia. " +
		"Czarna Tonia atakuje: Topielica z Tramwaju traci 13 zdrowia. Topielica z Tramwaju pada. " +
		"Zwycięstwo! Łup na głowę: 30 korków, doświadczenie: 40. Na ziemi leży: Instrukcja Ładunku Wybuchowego, Raca Sygnałowa."
	roundSpells = "Spichrzowy Szczur atakuje: Mieczysław traci 4 zdrowia. " +
		"Ignacy Lont rzuca: Podmuch. Podkopnik traci 9 zdrowia. Spichrzowy Szczur traci 10 zdrowia. " +
		"Mira Dezerterka atakuje (trafienie krytyczne!): Podkopnik traci 12 zdrowia. Podkopnik atakuje: Mira Dezerterka traci 9 zdrowia. " +
		"Mieczysław rzuca: Sztuka Petardy. Spichrzowy Szczur traci 13 zdrowia. Spichrzowy Szczur pada."
	roundFour = "Szczur Rynnowy atakuje: Tomir traci 4 zdrowia. Tomir używa: Syrop z Rzęsy (+11 zdrowia). " +
		"Nurek bez Twarzy atakuje: Tomir traci 9 zdrowia. Rak Kanałowy atakuje: Tomir. Pudło."
	roundLate = "Czas na decyzję minął: Mieczysław atakuje. Mieczysław atakuje: Nurek bez Twarzy traci 8 zdrowia. " +
		"Ludmiła Harpun atakuje: Nurek bez Twarzy. Pudło. Nurek bez Twarzy atakuje: Ludmiła Harpun traci 10 zdrowia."
)

func TestRoundEventsKeepEachActionWithItsOutcome(t *testing.T) {
	for scene, want := range map[string][]string{
		roundSix: {
			"Szczur Rynnowy podchodzi.",
			"Topielica z Tramwaju atakuje: Ludmiła Harpun traci 4 zdrowia.",
			"Ludmiła Harpun atakuje: Szczur Rynnowy. Pudło.",
			"Szczur Rynnowy atakuje: Ludmiła Harpun. Pudło.",
			"Ola atakuje: Topielica z Tramwaju traci 12 zdrowia.",
			"Chmara Komarów atakuje: Ola traci 9 zdrowia.",
		},
		roundFallen: {
			"Nurek bez Twarzy zrasta się: odzyskuje 4 zdrowia.",
			"Mieczysław atakuje: Nurek bez Twarzy. Pudło.",
			"Czarna Tonia atakuje: Nurek bez Twarzy traci 7 zdrowia.",
			"Nurek bez Twarzy atakuje: Czarna Tonia. Pudło.",
			"Rak Kanałowy atakuje: Czarna Tonia traci 11 zdrowia. Czarna Tonia pada nieprzytomna. Odpoczynek w osadzie postawi ją na nogi.",
		},
		roundWon: {
			"Topielica z Tramwaju zrasta się: odzyskuje 4 zdrowia.",
			"Wanda próbuje uciec, bez skutku.",
			"Topielica z Tramwaju atakuje: Ludmiła Harpun. Pudło.",
			"Ludmiła Harpun atakuje: Topielica z Tramwaju traci 7 zdrowia.",
			"Czarna Tonia atakuje: Topielica z Tramwaju traci 13 zdrowia. Topielica z Tramwaju pada.",
			"Zwycięstwo! Łup na głowę: 30 korków, doświadczenie: 40. Na ziemi leży: Instrukcja Ładunku Wybuchowego, Raca Sygnałowa.",
		},
		roundSpells: {
			"Spichrzowy Szczur atakuje: Mieczysław traci 4 zdrowia.",
			"Ignacy Lont rzuca: Podmuch. Podkopnik traci 9 zdrowia. Spichrzowy Szczur traci 10 zdrowia.",
			"Mira Dezerterka atakuje (trafienie krytyczne!): Podkopnik traci 12 zdrowia.",
			"Podkopnik atakuje: Mira Dezerterka traci 9 zdrowia.",
			"Mieczysław rzuca: Sztuka Petardy. Spichrzowy Szczur traci 13 zdrowia. Spichrzowy Szczur pada.",
		},
		roundLate: {
			"Czas na decyzję minął: Mieczysław atakuje. Mieczysław atakuje: Nurek bez Twarzy traci 8 zdrowia.",
			"Ludmiła Harpun atakuje: Nurek bez Twarzy. Pudło.",
			"Nurek bez Twarzy atakuje: Ludmiła Harpun traci 10 zdrowia.",
		},
	} {
		if got := RoundEvents(scene); !slices.Equal(got, want) {
			t.Errorf("events of %q:\n got %q\nwant %q", scene, got, want)
		}

		if got := strings.Join(RoundEvents(scene), " "); got != scene {
			t.Errorf("the events do not give the round back: %q", got)
		}
	}
}

func TestRoundEventsOfTheGamesOwnLines(t *testing.T) {
	scene := "Czas na decyzję minął: Ala atakuje. Wilk atakuje: Ala. Pudło (k20: 3 przeciw KP 13). " +
		"Wilk wyprowadza atak okazyjny na odchodzącego: Bob. Wilk atakuje: Bob traci 5 zdrowia. Jad działa! " +
		"Jad działa: Ala traci 2 zdrowia. Ala używa: Mikstura (+8 zdrowia). Jad przestaje działać. " +
		"Bob atakuje (k20: 20, trafienie krytyczne!): Wilk traci 9 zdrowia (widmo). Wilk pada. " +
		"Szef zostawia artefakt: Korona. Daje +2 do ataku. Zwycięstwo! Łup na głowę: 5 kapsli, doświadczenie: 10."
	want := []string{
		// A late hero's blow that never came is no part of the next one.
		"Czas na decyzję minął: Ala atakuje.",
		"Wilk atakuje: Ala. Pudło (k20: 3 przeciw KP 13).",
		"Wilk wyprowadza atak okazyjny na odchodzącego: Bob. Wilk atakuje: Bob traci 5 zdrowia. Jad działa!",
		"Jad działa: Ala traci 2 zdrowia.",
		"Ala używa: Mikstura (+8 zdrowia). Jad przestaje działać.",
		"Bob atakuje (k20: 20, trafienie krytyczne!): Wilk traci 9 zdrowia (widmo). Wilk pada.",
		"Szef zostawia artefakt: Korona. Daje +2 do ataku. Zwycięstwo! Łup na głowę: 5 kapsli, doświadczenie: 10.",
	}

	if got := RoundEvents(scene); !slices.Equal(got, want) {
		t.Errorf("events:\n got %q\nwant %q", got, want)
	}

	// The game cuts the log with dice, the evaluation without: alike.
	parts := RoundParts(scene)
	bare := RoundParts(withoutDice(scene))

	if len(parts) != 3 || len(bare) != len(parts) {
		t.Fatalf("parts = %q, without dice %q", parts, bare)
	}

	for index := range parts {
		if withoutDice(parts[index]) != bare[index] {
			t.Errorf("part %d: %q with dice, %q without", index+1, parts[index], bare[index])
		}
	}
}

func TestRoundPartsSplitLongRoundsEvenly(t *testing.T) {
	if got := RoundParts(roundFour); !slices.Equal(got, []string{roundFour}) {
		t.Errorf("a round of 4 events was cut: %q", got)
	}

	if got := RoundParts(roundSix); len(got) != 2 || len(RoundEvents(got[0])) != 3 || len(RoundEvents(got[1])) != 3 {
		t.Errorf("6 events = %q, want 3+3", got)
	}

	if got := RoundParts(roundFallen); len(got) != 2 || len(RoundEvents(got[0])) != 3 || !strings.HasPrefix(got[1], "Nurek bez Twarzy atakuje") {
		t.Errorf("5 events = %q, want 3+2", got)
	}

	if got := RoundParts(roundWon); len(got) != 2 || !strings.HasSuffix(got[1], "Raca Sygnałowa.") || strings.Contains(got[0], "Zwycięstwo") {
		t.Errorf("the victory is not in the last part: %q", got)
	}

	blow := "Ala atakuje: Wilk traci 1 zdrowia. "
	for events, want := range map[int][]int{5: {3, 2}, 6: {3, 3}, 7: {3, 2, 2}, 9: {3, 3, 3}, 10: {4, 3, 3}, 12: {4, 4, 4}} {
		parts := RoundParts(strings.TrimSpace(strings.Repeat(blow, events)))

		sizes := make([]int, 0, len(parts))
		for _, part := range parts {
			sizes = append(sizes, len(RoundEvents(part)))
		}

		if !slices.Equal(sizes, want) {
			t.Errorf("%d events in parts of %v, want %v", events, sizes, want)
		}
	}
}

// partNarrator answers every part of a round with its own blows, but
// leaves out the blows of the scenes in skip, and counts its calls.
type partNarrator struct {
	mu       sync.Mutex
	skip     string
	requests []NarrateRequest
}

func (n *partNarrator) Narrate(_ context.Context, request NarrateRequest) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.requests = append(n.requests, request)
	if request.Decision.Scene == n.skip {
		return "Wszyscy machają bronią, nic się nie dzieje.", nil
	}

	blows := roundNumbers(request.Decision.Scene)

	return "Część rundy, ciosy za " + strings.Join(blows, ", ") + ".", nil
}

func TestGuardChecksEachPartAgainstItsOwnEvents(t *testing.T) {
	parts := RoundParts(roundSix)
	first := Decision{Outcome: OutcomeScene, Scene: parts[0], Part: true}

	// A part may not be told with the other part's blows.
	if _, err := Narrate(context.Background(), &fixedNarrator{text: "Topielica zadaje 4, Ola zadaje 12, a komary 9."},
		NarrateRequest{Merchant: Merchant{Name: MasterName}, Decision: first}); err == nil {
		t.Error("a part told with the blows of another one passed")
	}

	// Nor without its own.
	text, err := Narrate(context.Background(), &fixedNarrator{text: "Szczur podchodzi i wszyscy pudłują."},
		NarrateRequest{Merchant: Merchant{Name: MasterName}, Decision: first})
	if err == nil || text != "" {
		t.Errorf("a part without its blow passed: %q, %v", text, err)
	}

	good := &partNarrator{}
	if text, err := Narrate(context.Background(), good, NarrateRequest{Merchant: Merchant{Name: MasterName}, Decision: first}); err != nil || !strings.Contains(text, "4") || len(good.requests) != 1 {
		t.Errorf("a good part was refused: %q, %v, %d calls", text, err, len(good.requests))
	}
}

// fixedNarrator always answers the same.
type fixedNarrator struct{ text string }

func (n *fixedNarrator) Narrate(context.Context, NarrateRequest) (string, error) {
	return n.text, nil
}

// masterRound has the master tell a round of the given log in a fight of
// the test world and returns the log of its player.
func masterRound(t *testing.T, master Narrator, recorded *[]NarrateRequest, scene string) []string {
	t.Helper()

	realm := testRealm(t)
	ala := enlist(realm, "Ala")
	do(realm, ala, "w")

	var recordMu sync.Mutex

	// Set only now, so that no round before the one told here is narrated.
	realm.SetMaster(master, func(request NarrateRequest, _ string) {
		recordMu.Lock()
		defer recordMu.Unlock()

		*recorded = append(*recorded, request)
	})

	realm.mu.Lock()
	current := realm.fightOf(realm.players[ala])

	if current == nil {
		realm.mu.Unlock()
		t.Fatal("no fight")
	}

	current.events, current.narrating = []string{scene}, false
	realm.narrateRound(current, []string{ala})
	realm.mu.Unlock()

	deadline := time.After(2 * time.Second)
	for !logged(realm, ala, MasterMark) {
		select {
		case <-realm.Updates(ala):
		case <-deadline:
			t.Fatalf("the master never spoke: %v", realm.View(ala).Log)
		}
	}

	return realm.View(ala).Log
}

func TestMasterTellsALongRoundInParts(t *testing.T) {
	master := &partNarrator{}

	var recorded []NarrateRequest

	log := masterRound(t, master, &recorded, roundSpells)
	parts := RoundParts(roundSpells)

	master.mu.Lock()
	defer master.mu.Unlock()

	if len(master.requests) != 2 || len(parts) != 2 {
		t.Fatalf("%d calls for %d parts", len(master.requests), len(parts))
	}

	for index, request := range master.requests {
		if request.Decision.Scene != parts[index] || !request.Decision.Part || request.Decision.Limit() != PartLimit {
			t.Errorf("call %d = %+v, want part %q", index+1, request.Decision, parts[index])
		}
	}

	want := MasterMark + "Część rundy, ciosy za 4, 9, 10, 12. Część rundy, ciosy za 9, 13."
	if !slices.Contains(log, want) || len(recorded) != 2 {
		t.Errorf("log = %q, want one line %q; recorded %d", log, want, len(recorded))
	}
}

func TestMasterTellsAShortRoundAtOnce(t *testing.T) {
	master := &partNarrator{}

	var recorded []NarrateRequest

	masterRound(t, master, &recorded, roundFour)

	master.mu.Lock()
	defer master.mu.Unlock()

	if len(master.requests) != 1 || master.requests[0].Decision.Part || master.requests[0].Decision.Scene != roundFour {
		t.Errorf("requests = %+v", master.requests)
	}
}

func TestMasterDropsOnlyThePartItFailed(t *testing.T) {
	parts := RoundParts(roundSix)
	master := &partNarrator{skip: parts[1]}

	var recorded []NarrateRequest

	log := masterRound(t, master, &recorded, roundSix)

	master.mu.Lock()
	defer master.mu.Unlock()

	// The first part at once, the second twice and then left to the rules.
	if len(master.requests) != 3 || master.requests[2].Decision.Scene != parts[1] || master.requests[2].Correction == "" {
		t.Fatalf("%d calls: %+v", len(master.requests), master.requests)
	}

	if want := MasterMark + "Część rundy, ciosy za 4."; !slices.Contains(log, want) || len(recorded) != 1 {
		t.Errorf("log = %q, want %q; recorded %d", log, want, len(recorded))
	}
}

// The cases come from the narrations of test18 rounds that both blind
// judges read, as they wrote them.
func TestFlippedFindsHitsToldAsMisses(t *testing.T) {
	cases := []struct{ scene, text, want string }{
		// Struck for 4, told missing in the same breath.
		{"Szczur Rynnowy zbliża się. Tomir rzuca: Schemat Harpuna Elektrycznego. Rak Kanałowy traci 14 zdrowia. Nurek bez Twarzy atakuje: Tomir. Pudło. Rak Kanałowy atakuje: Tomir. Pudło. Szczur Rynnowy atakuje: Tomir traci 4 zdrowia.",
			"Tomir odpala Schemat Harpuna Elektrycznego i Rak Kanałowy traci 14. Nurek bez Twarzy, Rak Kanałowy i Szczur Rynnowy pudłują, ale szczur zdążył Tomira podrapać za 4.", "Szczur Rynnowy"},
		// The one missed told as the one missing.
		{"Spichrzowy Szczur atakuje: Zofia traci 11 zdrowia. Zgłodniały Dezerter atakuje: Szymek Dobosz. Pudło. Zofia atakuje: Spichrzowy Szczur traci 12 zdrowia. Spichrzowy Szczur pada. Szymek Dobosz atakuje: Podkopnik traci 10 zdrowia. Podkopnik pada.",
			"Szczur gryzie Zofię za 11, a Szymek chybia Dezertera. Zofia wbija muszkiet w Spichrzowego Szczura za 12 i bestia pada.", "Szymek Dobosz"},
		// Someone who did not attack at all.
		{"Kasia atakuje: Szczur Rynnowy. Pudło. Szczur Rynnowy atakuje: Siostra Bronisława. Pudło. Rataj Bosak atakuje: Szczur Rynnowy traci 8 zdrowia.",
			"Kasia, Szczur i Siostra Bronisława pudłują, ale Rataj Bosak trafia Szczura za 8.", "Siostra Bronisława"},
		// With the dice the game writes.
		{"Rak Kanałowy atakuje: Zofia. Pudło (k20: 3 przeciw KP 13). Topielica z Tramwaju atakuje (k20: 20, trafienie krytyczne!): Zofia traci 4 zdrowia.",
			"Rak i Topielica pudłują w Zofię.", "Topielica z Tramwaju"},

		// The swarm is what Ola missed, not who missed.
		{"Ola atakuje: Chmara Komarów. Pudło. Chmara Komarów atakuje: Ola traci 10 zdrowia.",
			"Ola tnie harpunem w chmarę komarów i pudłuje. Komary tną ją za 10.", ""},
		// "Czarną" is whom the corpse missed.
		{"Topielica z Tramwaju atakuje: Czarna Tonia. Pudło. Czarna Tonia atakuje: Topielica z Tramwaju traci 22 zdrowia.",
			"Jej atak na Czarną Tonię chybia, a Tonia tnie ją za 22.", ""},
		// A clause that opens with someone else.
		{"Wilk z Pobojowiska podchodzi. Podkopnik atakuje: Ola. Pudło. Wilk z Pobojowiska atakuje: Rotmistrz Hieronim Wilk traci 8 zdrowia.",
			"Z popiołu wyłania się Wilk z Pobojowiska, Podkopnik pudłuje w Olę. Wilk gryzie Rotmistrza za 8.", ""},
		// One who hit and missed may be told either way.
		{"Ola atakuje: Goblin. Pudło. Ola atakuje: Goblin traci 5 zdrowia.", "Ola najpierw pudłuje, potem tnie za 5.", ""},
	}

	for _, test := range cases {
		if got := flipped(test.text, test.scene); got != test.want {
			t.Errorf("flipped(%q) = %q, want %q", test.text, got, test.want)
		}
	}
}
