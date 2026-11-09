package game

import (
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// A companion who follows the hero can be talked to on the road, not only
// at the inn.
func TestTalkToCompanionOnTheRoad(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	built, err := world.Build(presets[0])
	if err != nil {
		t.Fatal(err)
	}

	realm := NewRealm(built, 7)
	hero := realm.Join("Radomir", "")
	mate := built.Companions[0].Name

	if err := realm.Recruit(hero, mate); err != nil {
		t.Fatal(err)
	}

	// East of Szary Bród lies the Stary Bór, with no monsters.
	realm.Move(hero, 1, 0)

	if view := realm.View(hero); realm.World().At(view.Me.Pos).Town >= 0 || view.Fight != nil {
		t.Fatalf("not on a quiet road: %v", view.Log)
	}

	realm.Execute(hero, Parse("rozmawiaj z "+mate))

	if shop := realm.View(hero).Shop; shop == nil || shop.Keeper != mate {
		t.Fatalf("no talk with %s on the road: %v", mate, realm.View(hero).Log)
	}
}
