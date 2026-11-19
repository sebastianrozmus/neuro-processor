package game

import (
	"slices"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

func TestPresetArtifacts(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	for _, spec := range presets {
		if spec.Mode.Rooms() || spec.Mode == world.ModeInvestigation {
			continue
		}

		best := map[world.Kind]int{}
		for _, item := range spec.Items {
			best[item.Kind] = max(best[item.Kind], item.Power())
		}

		kinds := map[world.Kind]int{}

		for _, item := range spec.Artifacts {
			kinds[item.Kind]++

			if !item.Artifact || item.Kind != world.KindCharm && item.Power() <= best[item.Kind] {
				t.Errorf("%s: %s is no stronger than the shops' best (%d against %d)", spec.Name, item.Name, item.Power(), best[item.Kind])
			}
		}

		if kinds[world.KindWeapon] != 5 || kinds[world.KindArmor] != 5 || kinds[world.KindCharm] != 5 {
			t.Errorf("%s: artifacts by kind %v, want 5 of each", spec.Name, kinds)
		}

		merchant := MerchantFor(spec, spec.ShopFor(world.KindWeapon), world.MaxTier)
		if slices.ContainsFunc(merchant.Items, func(item Item) bool { return strings.HasPrefix(item.ID, "artifact") }) {
			t.Errorf("%s: a shop sells an artifact", spec.Name)
		}
	}
}

func TestBossesLeaveArtifacts(t *testing.T) {
	realm := testRealm(t)
	realm.world.Artifacts = []world.ItemSpec{
		{ID: "artifact-1", Name: "Miecz Króla", Kind: world.KindWeapon, Tier: 3, Artifact: true, Strength: 15},
		{ID: "artifact-2", Name: "Pierścień Króla", Kind: world.KindCharm, Effect: world.EffectPower, Tier: 3, Artifact: true, Strength: 9},
	}

	hero := enlist(realm, "Ala")
	ala := realm.players[hero]

	ala.pos = world.Point{X: realm.world.Lair.X + 1, Y: realm.world.Lair.Y}
	do(realm, hero, "w")
	win(t, realm, hero)

	ground := realm.ground[realm.world.Index(realm.world.Lair)]
	if !slices.ContainsFunc(ground, func(item world.ItemSpec) bool { return item.Artifact }) || !logged(realm, hero, "Szef zostawia artefakt") || len(realm.dropped) != 1 {
		t.Fatalf("no artifact after the boss: ground %v, dropped %v, log %v", itemNames(ground), realm.dropped, realm.View(hero).Log)
	}

	// The heir leaves the other one: the first ones won are all different.
	realm.worldDay(realm.day + heirDays)
	ala.pos = world.Point{X: realm.world.Lair.X + 1, Y: realm.world.Lair.Y}
	do(realm, hero, "w")
	win(t, realm, hero)

	if len(realm.dropped) != 2 || realm.dropped[0] == realm.dropped[1] {
		t.Fatalf("the heir's artifact: dropped %v", realm.dropped)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(restored.dropped, realm.dropped) {
		t.Errorf("dropped artifacts were not saved: %v", restored.dropped)
	}
}

func TestEditorSetsAHeroUp(t *testing.T) {
	realm := testRealm(t)
	realm.world.Artifacts = []world.ItemSpec{{ID: "artifact-1", Name: "Miecz Króla", Kind: world.KindWeapon, Tier: 3, Artifact: true, Strength: 15}}
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]
	attack := realm.View(hero).Me.Attack

	for _, change := range []Change{
		{Op: "equip", Field: "weapon", Item: "artifact-1"},
		{Op: "give", Item: "item-3"},
		{Op: "teach", Item: "item-5"},
		{Op: "stat", Field: "sil", Value: 16},
		{Op: "stat", Field: "level", Value: 4},
		{Op: "stat", Field: "gold", Value: 999},
	} {
		if err := realm.Edit(hero, change); err != nil {
			t.Fatalf("%+v: %v", change, err)
		}
	}

	if ala.weapon == nil || ala.weapon.ID != "artifact-1" || ala.abilities[Strength] != 16 || ala.level != 4 || ala.gold != 999 || realm.View(hero).Me.Attack <= attack+15 {
		t.Errorf("after the edits: weapon %v, abilities %v, level %d, gold %d, attack %d", ala.weapon, ala.abilities, ala.level, ala.gold, realm.View(hero).Me.Attack)
	}

	if !slices.ContainsFunc(ala.spells, func(item world.ItemSpec) bool { return item.ID == "item-5" }) {
		t.Errorf("spells: %v", itemNames(ala.spells))
	}

	for _, wrong := range []Change{
		{Op: "teach", Item: "item-1"},
		{Op: "equip", Field: "armor", Item: "artifact-1"},
		{Op: "give", Item: "nothing"},
		{Op: "stat", Field: "speed", Value: 3},
		{Member: "Nikt", Op: "stat", Field: "hp", Value: 3},
	} {
		if realm.Edit(hero, wrong) == nil {
			t.Errorf("%+v was allowed", wrong)
		}
	}

	bag := len(ala.bag)
	if err := realm.Edit(hero, Change{Op: "take", Field: "bag", Value: bag - 1}); err != nil || len(ala.bag) != bag-1 {
		t.Errorf("taking from the bag: %v, %d left", err, len(ala.bag))
	}

	if err := realm.Edit(hero, Change{Op: "equip", Field: "weapon"}); err != nil || ala.weapon != nil {
		t.Errorf("taking the weapon off: %v, %v", err, ala.weapon)
	}
}

func TestCraftedItemsLastThroughASave(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	made, err := realm.Craft(world.ItemSpec{Name: "Gwiazda Zarania", Description: "Ręcznie zrobione zaklęcie.", Kind: world.KindSpell, Effect: world.EffectBolt, Tier: 3, Strength: 40})
	if err != nil {
		t.Fatal(err)
	}

	if made.Power() != 40 || made.Cost() == 0 {
		t.Errorf("made: power %d, cost %d", made.Power(), made.Cost())
	}

	if _, err := realm.Craft(world.ItemSpec{Name: "Gwiazda Zarania", Kind: world.KindCharm}); err == nil {
		t.Error("a second item of the same name was made")
	}

	if err := realm.Edit(hero, Change{Op: "teach", Item: made.ID}); err != nil {
		t.Fatal(err)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	spells := restored.players[hero].spells
	if len(spells) != 1 || spells[0].Name != "Gwiazda Zarania" || spells[0].Power() != 40 {
		t.Errorf("the made spell after a save: %+v", spells)
	}

	for index := range realm.world.Shops {
		if slices.ContainsFunc(realm.merchant(index, world.MaxTier).Items, func(item Item) bool { return item.ID == made.ID }) {
			t.Error("a shop sells the made item")
		}
	}
}
