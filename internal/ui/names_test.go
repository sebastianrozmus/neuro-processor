package ui

import (
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

func TestNamerPaintsNamesInAnyForm(t *testing.T) {
	w := &world.World{Spec: world.Spec{
		Name:   "Smocza Góra",
		Places: []world.PlaceSpec{{Name: "Kopalnia Durgara"}},
		Items:  []world.ItemSpec{{Name: "Krótki Miecz", Kind: world.KindWeapon}, {Name: "Zwój Iskry", Kind: world.KindSpell}},
		Shops:  []world.ShopSpec{{Name: "Kuźnia Pod Kowadłem", Keeper: "Thorin Żelazna Broda"}},
	}}
	names := newNamer(w)

	cases := map[string]string{
		"Cień Smoczej Góry":                    "Cień " + placeName.Render("Smoczej Góry"),
		"Idź do Kopalni Durgara.":              "Idź do " + placeName.Render("Kopalni Durgara") + ".",
		"Thorin Żelazna Broda ma Krótki Miecz": personName.Render("Thorin Żelazna Broda") + " ma " + itemName.Render("Krótki Miecz"),
		"Kuźnia Pod Kowadłem i Zwój Iskry":     placeName.Render("Kuźnia Pod Kowadłem") + " i " + spellName.Render("Zwój Iskry"),
		// Names are capitalised; plain words that share a stem are not names.
		"krótki miecz z kopalni": "krótki miecz z kopalni",
	}

	for text, want := range cases {
		if got := names.paint(text); got != want {
			t.Errorf("paint(%q) = %q, want %q", text, got, want)
		}
	}

	if got := (*namer)(nil).paint("Smocza Góra"); got != "Smocza Góra" {
		t.Errorf("a nil namer changed the text: %q", got)
	}
}
