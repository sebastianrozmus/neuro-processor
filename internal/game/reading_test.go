package game

import (
	"strings"
	"testing"
)

func TestShortWordNamesNoLongerWare(t *testing.T) {
	wares := []Ware{{ID: "big", Name: "Wielka Mikstura Leczenia"}, {ID: "bow", Name: "Łuk"}}

	if got := namedItem(Fold("skad wiesz ze on wie"), wares); got != "" {
		t.Errorf("namedItem = %q, want none", got)
	}
	if got := namedItem(Fold("pokaz luk"), wares); got != "bow" {
		t.Errorf("namedItem = %q, want bow", got)
	}
}

func TestOldWomanIsNoOldForest(t *testing.T) {
	for _, text := range []string{"milcz starucho", "starczy ci odwagi"} {
		if touches(strings.Fields(Fold(text)), "Stary Bór") > 0 {
			t.Errorf("%q touches Stary Bór", text)
		}
	}

	if !Read(InterpretRequest{Text: "milcz starucho"}).rude {
		t.Error("an old crone is no insult")
	}
}
