package game

import (
	"strings"
	"testing"
)

// Gaps the run21 data showed (D:\neuro-train\run21, 03.11): pairs of talks
// where the second one checks what a speaker said of themselves in the
// first, each with the lines that must not trip the fix.

// A currency word alone is no sum: a lightning is no feather, a nebula of
// spores no price. A number with it is.
func TestCurrencyAloneIsNoSumFromRun21(t *testing.T) {
	for _, c := range []struct {
		fact, currency string
		want           bool
	}{
		{"Gromnica przeżyła uderzenie pioruna.", "piór", false},
		{"Ilian zbiera pióra orłów.", "piór", false},
		{"Ilian bierze 50 piór za dzień.", "piór", true},
		{"Kito urodził się w Mgławicy Zarodników.", "zarodników", false},
		{"Kito jest winien 200 zarodników lichwiarzowi.", "zarodników", true},
		{"Nurbek bierze 70 krążków.", "krążków", true},
		{"Bogna ma brata Radka.", "szt. złota", false},
		{"Bogna zarobiła raz 300 sztuk złota.", "szt. złota", true},
	} {
		if got := priced(strings.Fields(Fold(c.fact)), c.currency); got != c.want {
			t.Errorf("priced(%q, %q) = %v, want %v", c.fact, c.currency, got, c.want)
		}
	}
}

// Questions about the speaker's own people and road, as players ask them
// in a second talk.
func TestMorePrivateQuestionsFromRun21(t *testing.T) {
	for text, want := range map[string]bool{
		"twój brat dalej pilnuje pomp?":           true,
		"twoja mama dalej przy mieście?":          true,
		"jak miał na imię ten twój brat? jasiek?": true,
		"ten twój brat, malwin?":                  true,
		"masz babcię u kopistów?":                 true,
		"a twoja ciotka czym się zajmuje?":        true,
		"z dna studni jesteś?":                    true,
		"ty z gór jesteś, co?":                    true,
		"gdzieś ty się wychował?":                 true,
		"urodziła się pani w szklarni?":           true,
		"kto panią uczył nitować?":                true,
		"kto pana uczył kuć?":                     true,
		"z kim jesteś?":                           false,
		"jesteś z nami?":                          false,
		"z czego jesteś zrobiony?":                false,
		"ile za miecz z brązu?":                   false,
		"ma Babka Smolna coś na kaszel?":          false,
		"o matce grzybni co wiesz?":               false,
		"kto uczył smoka latać?":                  false,
		"moja ciotka mieszka w Ambarze, znasz ją?": false,
	} {
		if got := Private(text); got != want {
			t.Errorf("Private(%q) = %v, want %v", text, got, want)
		}
	}
}

// A question about who taught the speaker touches the teacher they named;
// a family in general does not.
func TestTeacherTouchesTheBioFromRun21(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	said := func(fact, question string) bool {
		realm.bios = map[string][]*bioFact{}
		realm.remember("Bogna Topór", fact, 1, hero)

		return strings.Contains(realm.bioHint("Bogna Topór", question), "Już o tym mówiłaś")
	}

	for _, pair := range [][2]string{
		{"Bogna terminowała u kowala Hartwiga.", "kto cię uczył?"},
		{"Uczyła ją stara Halina Kordelas.", "a nauczycielka żyje?"},
		{"Bogna wychowała się na Wyspie Gildii.", "gdzie dorastałaś?"},
	} {
		if !said(pair[0], pair[1]) {
			t.Errorf("%q did not touch %q", pair[1], pair[0])
		}
	}

	if said("Bogna terminowała u kowala Hartwiga.", "masz rodzinę?") {
		t.Error("a family touched the teacher")
	}
}

// Pieces of something counted are no sum asked: "mam 5 skór szczurów,
// kupisz je?" sells nothing for 5.
func TestCountedThingsAreNoSumFromRun21(t *testing.T) {
	owned := []Ware{{ID: "item-4", Name: "Skórzany Kaftan", Price: 40}}
	wares := []Ware{{ID: "item-7", Name: "Napar z Tataraku", Price: 15}}

	for _, text := range []string{"mam 5 skór szczurów, kupisz je?", "kupisz 3 kawałki kory?", "zaplacisz cos za 6 galazek ruty?"} {
		got := Read(InterpretRequest{Text: text, Wares: wares, Owned: owned, Currency: "kapsli"})
		if got.Intent.Action != ActionSell || got.Intent.Amount != 0 || got.Intent.ItemID != "" {
			t.Errorf("%q: %+v", text, got.Intent)
		}
	}

	for text, amount := range map[string]int{
		"sprzedam kaftan za 30":        30,
		"kupisz skórzany kaftan za 25?": 25,
		"sprzedam kaftan, 35 kapsli":   35,
	} {
		got := Read(InterpretRequest{Text: text, Wares: wares, Owned: owned, Currency: "kapsli"})
		if got.Intent.Action != ActionSell || got.Intent.Amount != amount || got.Intent.ItemID != "item-4" {
			t.Errorf("%q: %+v, want %d for item-4", text, got.Intent, amount)
		}
	}
}
