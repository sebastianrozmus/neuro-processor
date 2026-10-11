package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

func TestInspectionTakesStockOfTheRealm(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	give(realm, hero, "Kij")
	do(realm, hero, "idź do zbrojowni")

	report := realm.Inspect()
	if report.Name != "Test" || report.Width != world.Width || len(report.Tiles) != world.Width*world.Height || len(report.Places) == 0 {
		t.Fatalf("report = %+v", report)
	}

	if len(report.Players) != 1 || report.Players[0].Name != "Ala" || report.Players[0].Talking != "Kowal" || len(report.Players[0].Bag) != 1 {
		t.Errorf("players = %+v", report.Players)
	}

	tile := report.Tiles[report.Players[0].Pos.Y*world.Width+report.Players[0].Pos.X]
	if len(tile.Players) != 1 || tile.Town < 0 {
		t.Errorf("hero's tile = %+v", tile)
	}

	var smith *KeeperInfo
	for index := range report.Keepers {
		if report.Keepers[index].Name == "Kowal" {
			smith = &report.Keepers[index]
		}
	}

	if smith == nil || smith.Building != "Zbrojownia" || smith.Knows == 0 {
		t.Errorf("smith = %+v", smith)
	}

	if len(report.Facts) == 0 || len(report.Items) == 0 || len(report.Beasts) == 0 {
		t.Errorf("facts %d, items %d, beasts %d", len(report.Facts), len(report.Items), len(report.Beasts))
	}

	// Every fact names who knows it, and the boss is known to all.
	for _, fact := range report.Facts {
		if fact.Subject == "Szef" && len(fact.Known) != len(report.Keepers) {
			t.Errorf("the boss is known to %v of %d", fact.Known, len(report.Keepers))
		}
	}
}

func TestInspectorEditsTheWorld(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	// A fact added by hand is known to whom the inspector says.
	fact, ok := realm.SetFact(0, "Studnia", "Woda w studni na rynku smakuje żelazem.", []string{"Jagna", "nikt taki"})
	if !ok || fact.ID == 0 || len(fact.Known) != 1 || fact.Known[0] != "Jagna" || fact.Source != "podgląd" {
		t.Fatalf("added fact = %+v, %v", fact, ok)
	}

	edited, ok := realm.SetFact(fact.ID, "Studnia", "Woda w studni jest zatruta.", []string{"Kowal"})
	if !ok || edited.Text != "Woda w studni jest zatruta." || len(edited.Known) != 1 || edited.Known[0] != "Kowal" {
		t.Errorf("edited fact = %+v", edited)
	}

	if _, ok := realm.SetFact(fact.ID, "", "x", nil); ok {
		t.Error("an empty subject was accepted")
	}

	name, text, ok := realm.Description(DescribeFact, fact.ID)
	if !ok || name != "Studnia" || text != "Woda w studni jest zatruta." {
		t.Errorf("description = %q %q %v", name, text, ok)
	}

	if !realm.DropFact(fact.ID) || realm.DropFact(fact.ID) {
		t.Error("the fact was not dropped once")
	}

	// Descriptions of places, monsters, items, companions and keepers can
	// be reworded; the places the screens draw follow.
	for kind, index := range map[string]int{DescribePlace: 0, DescribeMonster: 0, DescribeBoss: 0, DescribeItem: 0, DescribeKeeper: 0} {
		if !realm.Describe(kind, index, "Nowy opis: "+kind+".") {
			t.Errorf("%s was not described", kind)
		}

		if _, text, _ := realm.Description(kind, index); text != "Nowy opis: "+kind+"." {
			t.Errorf("%s reads %q", kind, text)
		}
	}

	if realm.Describe(DescribePlace, 99, "x") || realm.Describe("nothing", 0, "x") || realm.Describe(DescribeItem, 0, "  ") {
		t.Error("a bad description was accepted")
	}

	if realm.View(hero).Place.Description != "Nowy opis: place." && realm.View(hero).Places[0].Description != "Nowy opis: place." {
		t.Errorf("the screens do not see the new place description")
	}

	// A hero's purse and health can be set, within reason.
	if !realm.Adjust(hero, "gold", 500) || realm.View(hero).Me.Gold != 500 || !realm.Adjust(hero, "hp", 1) || realm.View(hero).Me.HP != 1 {
		t.Errorf("adjust: %+v", realm.View(hero).Me)
	}

	if realm.Adjust(hero, "gold", -1) || realm.Adjust("nikt", "gold", 1) || realm.Adjust(hero, "level", 9) {
		t.Error("a bad adjustment was accepted")
	}

	if logged := strings.Join(realm.View(hero).Log, "\n"); !strings.Contains(logged, "Podgląd zmienia: gold = 500.") {
		t.Errorf("log = %q", logged)
	}
}
