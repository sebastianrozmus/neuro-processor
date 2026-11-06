package game

import (
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
