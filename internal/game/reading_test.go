package game

import "testing"

func TestShortWordNamesNoLongerWare(t *testing.T) {
	wares := []Ware{{ID: "big", Name: "Wielka Mikstura Leczenia"}, {ID: "bow", Name: "Łuk"}}

	if got := namedItem(Fold("skad wiesz ze on wie"), wares); got != "" {
		t.Errorf("namedItem = %q, want none", got)
	}
	if got := namedItem(Fold("pokaz luk"), wares); got != "bow" {
		t.Errorf("namedItem = %q, want bow", got)
	}
}
