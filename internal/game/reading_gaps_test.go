package game

import (
	"strings"
	"testing"
)

// Gaps the agents writing the run22 data reported (06.11): each with the
// lines that must not trip the fix.

// A goodbye with a sum in it is a goodbye: "nara, 70 było za dużo" leaves,
// it offers no 70. An offer with a word of paying, or "za N", still offers,
// and so does a purchase before the goodbye.
func TestAGoodbyeWithASumLeavesFromGaps(t *testing.T) {
	wares := []Ware{{ID: "kaftan", Name: "Kaftan", Price: 80}}

	for _, text := range []string{
		"nara, 70 bylo za duzo", "dzięki, 50 to i tak za dużo, cześć", "70? za dużo, nara", "no trudno, 70 to nie dla mnie. idę",
		"nie stać mnie na 70, bywaj",
	} {
		for _, focus := range []string{"", "kaftan"} {
			reading := Read(InterpretRequest{Text: text, Wares: wares, FocusID: focus})
			if reading.Intent.Action != ActionLeave || !reading.Sure {
				t.Errorf("%q (focus %q) = %+v, want a sure leave", text, focus, reading)
			}

			if merged := reading.Merge(Intent{Action: ActionOffer}); merged.Action != ActionLeave || merged.Amount != 0 {
				t.Errorf("%q (focus %q) merged with an offer = %+v", text, focus, merged)
			}
		}
	}

	for _, c := range []struct {
		text string
		want Action
	}{
		{"nara, dam 70", ActionOffer},
		{"dobra, 70 za kaftan i spadam", ActionOffer},
		{"trzymaj się, 100 za kaftan to rozbój", ActionOffer},
		{"płacę 70 i idę", ActionOffer},
		{"ok biorę kaftan za 70, bywaj", ActionOffer},
		{"ok biore jeden kaftan, bywaj", ActionAccept},
	} {
		if got := Read(InterpretRequest{Text: c.text, Wares: wares}).Intent; got.Action != c.want {
			t.Errorf("%q = %+v, want %s", c.text, got, c.want)
		}
	}
}

// A companion in the party asked about their pay is told the deal stands:
// paid once at the hire, no sum and no haggling; a legend's wage is paid
// at dawn. "ile mamy bandaży?" and "ile wilków zabiłaś?" ask nothing of
// the kind.
func TestACompanionAskedAboutPayFromGaps(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	do(realm, hero, "najmij bognę", "rozmawiaj z bogną")

	if view := realm.View(hero).Shop; view == nil || view.Keeper != "Bogna Topór" {
		t.Fatalf("no talk with Bogna: %+v", view)
	}

	for _, text := range []string{"ile chcesz za dzień?", "ile ci płacę?", "jaki masz żołd?", "ile ci jestem winien?", "a ile bierzesz?", "ile kosztujesz"} {
		_, narrate, _ := realm.Haggle(hero, text, Intent{Action: ActionChat})
		if !strings.Contains(narrate.Hint, "umowa dawno stoi") || !strings.Contains(narrate.Hint, "przy dołączeniu") || !narrate.Quiet {
			t.Errorf("%q: hint %q, quiet %v", text, narrate.Hint, narrate.Quiet)
		}

		if strings.Contains(narrate.Hint, "proponuje ci pieniądze") {
			t.Errorf("%q briefed as an offer: %q", text, narrate.Hint)
		}
	}

	for _, text := range []string{"ile mamy bandaży?", "ile wilków zabiłaś?", "ile masz lat?", "co masz w plecaku?"} {
		_, narrate, _ := realm.Haggle(hero, text, Intent{Action: ActionChat})
		if strings.Contains(narrate.Hint, "umowa dawno stoi") || narrate.Quiet {
			t.Errorf("%q briefed as a question of pay: %q", text, narrate.Hint)
		}
	}

	// An offer of money is still an offer.
	if _, narrate, _ := realm.Haggle(hero, "dam ci 20 więcej, ile chcesz?", Intent{Action: ActionChat}); narrate.Hint != paidMate {
		t.Errorf("an offer with a question: %q", narrate.Hint)
	}

	// A legend's wage is paid at dawn.
	realm.world.Companions[1].Legend = true
	realm.recruit(realm.players[hero], 1, 30)
	do(realm, hero, "rozmawiaj z jaromirem")
	if view := realm.View(hero).Shop; view == nil || view.Keeper != "Brat Jaromir" {
		t.Fatalf("no talk with Jaromir: %+v", view)
	}

	if _, narrate, _ := realm.Haggle(hero, "ile ci płacę za dzień?", Intent{Action: ActionChat}); !strings.Contains(narrate.Hint, "co świt") || strings.Contains(narrate.Hint, "przy dołączeniu") || !narrate.Quiet {
		t.Errorf("a legend's pay: hint %q, quiet %v", narrate.Hint, narrate.Quiet)
	}
}
