package world

import (
	"reflect"
	"strings"
	"testing"
)

func presets(t *testing.T) []Spec {
	t.Helper()

	specs, err := Presets()
	if err != nil {
		t.Fatal(err)
	}

	if len(specs) != 7 {
		t.Fatalf("got %d presets, want 7", len(specs))
	}

	return specs
}

// worlds returns the presets that are played on a map with fights.
func worlds(t *testing.T) []Spec {
	t.Helper()

	var fighting []Spec
	for _, spec := range presets(t) {
		if !spec.Mode.Rooms() {
			fighting = append(fighting, spec)
		}
	}

	return fighting
}

func TestPresetsBuildPlayableWorlds(t *testing.T) {
	for _, spec := range worlds(t) {
		built, err := Build(spec)
		if err != nil {
			t.Fatalf("%s: %v", spec.Name, err)
		}

		legends := 0
		for _, companion := range built.Companions {
			if companion.Legend {
				legends++
			}
		}

		if len(built.Towns) != 2 || len(built.Shops) != 3 || len(built.Backdrops) == 0 || len(built.Companions) != 4 || legends != 1 {
			t.Errorf("%s: %d towns, %d shops, %d backdrops", spec.Name, len(built.Towns), len(built.Shops), len(built.Backdrops))
		}

		bosses, monsters, sites := 0, 0, map[int]bool{}

		for index, tile := range built.Tiles {
			point := Point{index % Width, index / Width}
			place := built.Places[tile.Place]

			if (place.Danger == 0) != (tile.Town >= 0) {
				t.Errorf("%s: %v is %q but town is %d", spec.Name, point, place.Name, tile.Town)
			}

			if built.near(point) && len(tile.Monsters) > 1 {
				t.Errorf("%s: %d monsters right next to a settlement", spec.Name, len(tile.Monsters))
			}

			if tile.Town >= 0 && (len(tile.Monsters) > 0 || tile.Boss || built.Towns[tile.Town] != point) {
				t.Errorf("%s: settlement at %v is not safe", spec.Name, point)
			}

			if tile.Site >= 0 {
				sites[tile.Site] = true
				treasure := built.Sites[tile.Site].Kind == SiteTreasure

				if treasure != (len(tile.Loot) > 0) || treasure != (len(tile.Monsters) == treasurePack) {
					t.Errorf("%s: site %q has %d items and %d guards", spec.Name, built.Sites[tile.Site].Name, len(tile.Loot), len(tile.Monsters))
				}
			}

			if tile.Boss {
				bosses++
			}

			monsters += len(tile.Monsters)
		}

		if bosses != 1 || !built.At(built.Lair).Boss {
			t.Errorf("%s: %d bosses", spec.Name, bosses)
		}

		if len(sites) != len(built.Sites) {
			t.Errorf("%s: %d of %d sites placed", spec.Name, len(sites), len(built.Sites))
		}

		if monsters < 30 {
			t.Errorf("%s: only %d monsters", spec.Name, monsters)
		}

		for _, kind := range Kinds {
			found := false
			for _, item := range built.Items {
				found = found || item.Kind == kind
			}

			if !found {
				t.Errorf("%s: no %s", spec.Name, kind)
			}
		}
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	spec := presets(t)[0]

	first, _ := Build(spec)
	second, _ := Build(spec)

	if !reflect.DeepEqual(first, second) {
		t.Error("the same spec gave two different maps")
	}
}

func TestNormalizeRepairsAndRejects(t *testing.T) {
	spec := Spec{
		Name: "  Test  ",
		Boss: BossSpec{Name: "Szef", Shape: "dragon"},
		Shops: []ShopSpec{
			{Name: "Kram", Kind: "bazaar", Keeper: "Kupiec"},
			{Name: "Drugi kram", Kind: ShopArmory, Keeper: "Inny"},
			{Name: "Bez gospodarza", Kind: ShopMagic},
		},
		Places: []PlaceSpec{
			{Name: "Osada", Terrain: "lava", Danger: -4},
			{Name: "Las", Terrain: "forest", Danger: 9},
			{Name: "", Danger: 1},
			{Name: "Druga", Danger: 0}, {Name: "Trzecia", Danger: 0}, {Name: "Czwarta", Danger: 0},
		},
		Sites:    []SiteSpec{{Name: "Kopiec", Kind: "dungeon"}, {Name: "Kopiec", Kind: SiteShrine}},
		Monsters: []MonsterSpec{{Name: "Wilk", Tier: 7}},
		Items: []ItemSpec{
			{Name: "Kij", Kind: KindWeapon, Tier: 0, Effect: EffectBolt},
			{Name: "Róg", Kind: "trinket", Tier: 1},
			{Name: "Zwój", Kind: KindSpell, Effect: EffectVigor, Tier: 2},
			{Name: "Flaszka", Kind: KindPotion, Effect: EffectMana, Tier: 1},
		},
	}

	if err := spec.Normalize(); err != nil {
		t.Fatal(err)
	}

	if spec.Name != "Test" || spec.Currency == "" || spec.Boss.Shape != "beast" {
		t.Errorf("defaults not applied: %+v", spec)
	}

	if len(spec.Places) != 4 || spec.Places[0].Terrain != "plains" || spec.Places[0].Danger != 0 || spec.Places[1].Danger != 3 {
		t.Errorf("places = %+v", spec.Places)
	}

	if len(spec.Shops) != 1 || spec.Shops[0].Kind != ShopArmory || spec.Shops[0].Greeting == "" {
		t.Errorf("shops = %+v", spec.Shops)
	}

	if len(spec.Sites) != 1 || spec.Sites[0].Kind != SiteTreasure || spec.Monsters[0].Tier != 3 || spec.Monsters[0].Shape != "beast" {
		t.Errorf("sites = %+v, monsters = %+v", spec.Sites, spec.Monsters)
	}

	want := []ItemSpec{
		{ID: "item-1", Name: "Kij", Kind: KindWeapon, Effect: EffectNone, Tier: 1},
		{ID: "item-2", Name: "Zwój", Kind: KindSpell, Effect: EffectBolt, Tier: 2},
		{ID: "item-3", Name: "Flaszka", Kind: KindPotion, Effect: EffectMana, Tier: 1},
	}
	if !reflect.DeepEqual(spec.Items, want) {
		t.Errorf("items = %+v", spec.Items)
	}

	if spec.ShopFor(KindSpell) != 0 {
		t.Error("items no shop specialises in should go to the first shop")
	}

	spec.Items = spec.Items[1:]
	if err := spec.Normalize(); err == nil {
		t.Error("a world without weapons should be rejected")
	}
}

func TestSaveAndLoadSpec(t *testing.T) {
	dir := t.TempDir()
	spec := presets(t)[2]
	spec.Name = "Żółta Łódź 7"

	path, err := SaveSpec(dir, spec)
	if err != nil {
		t.Fatal(err)
	}

	if want := dir + "/zolta-lodz-7.json"; path != want {
		t.Errorf("path = %s, want %s", path, want)
	}

	saved := SavedSpecs(dir)
	if len(saved) != 1 || !reflect.DeepEqual(saved[0], spec) {
		t.Errorf("loaded world differs from the saved one")
	}
}

func TestStats(t *testing.T) {
	sword := ItemSpec{Kind: KindWeapon, Effect: EffectNone, Tier: 2}
	if sword.Power() != 6 || sword.Price() != 110 || sword.MinPrice() != 71 || sword.Cost() != 0 {
		t.Errorf("tier 2 weapon: power %d, price %d, min %d", sword.Power(), sword.Price(), sword.MinPrice())
	}

	blast := ItemSpec{Kind: KindSpell, Effect: EffectBlast, Tier: 3}
	if blast.Power() != 16 || blast.Cost() != 12 || blast.Price() != 280 {
		t.Errorf("tier 3 blast: power %d, cost %d, price %d", blast.Power(), blast.Cost(), blast.Price())
	}

	for _, kind := range Kinds {
		for _, effect := range effectsOf[kind] {
			if (ItemSpec{Kind: kind, Effect: effect, Tier: 1}).Power() == 0 {
				t.Errorf("%s/%s has no power", kind, effect)
			}
		}
	}

	if got := StatsFor(9, false); got.HP != 55 {
		t.Errorf("tier is not clamped: %+v", got)
	}

	if got := StatsFor(1, true); got.HP != 170 {
		t.Errorf("boss stats = %+v", got)
	}
}

func TestAdventurePresetBuildsRooms(t *testing.T) {
	var spec Spec
	for _, candidate := range presets(t) {
		if candidate.Mode == ModeAdventure && spec.Name == "" {
			spec = candidate
		}
	}

	built, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	if len(built.Rooms) < 8 || len(built.Places) != len(built.Rooms) || len(built.Items) != len(built.Things) || len(built.Towns) != 1 || built.Monsters != nil || built.Shops != nil {
		t.Fatalf("adventure: %d rooms, %d places, %d items, %d towns", len(built.Rooms), len(built.Places), len(built.Items), len(built.Towns))
	}

	rooms := 0
	for index, tile := range built.Tiles {
		point := Point{index % Width, index / Width}
		room, ok := built.RoomAt(point)

		switch {
		case ok && (room.X != point.X || room.Y != point.Y):
			t.Errorf("room %q drawn at %v", room.Name, point)
		case ok:
			rooms++
		case tile.Place != -1 || len(tile.Loot) > 0:
			t.Errorf("void at %v has place %d and loot %v", point, tile.Place, tile.Loot)
		}
	}

	if rooms != len(built.Rooms) || built.Start() != (Point{X: built.Rooms[0].X, Y: built.Rooms[0].Y}) {
		t.Errorf("%d rooms placed, start %v", rooms, built.Start())
	}

	// Every thing of the story exists; every door has a text.
	for _, step := range built.Steps {
		for _, name := range []string{step.Thing, step.Gives, step.Reveals} {
			if _, ok := built.Item(itemID(built.Spec, name)); name != "" && !ok {
				t.Errorf("step %q names an unknown thing %q", step.Text, name)
			}
		}
	}
}

// itemID finds the ID of a thing by name.
func itemID(spec Spec, name string) string {
	for _, item := range spec.Items {
		if item.Name == name {
			return item.ID
		}
	}

	return ""
}

func TestStyleCard(t *testing.T) {
	var none *StyleSpec
	if !none.Empty() || none.Card() != "" {
		t.Error("a missing card should be empty")
	}

	style := &StyleSpec{Look: "Neon i deszcz.", Dialogue: "Krótko, z ironią.", Characters: "Łowcy nagród.", Art: "neon, rain"}
	want := "Świat i estetyka: Neon i deszcz.\nTon rozmów: Krótko, z ironią.\nPostacie: Łowcy nagród."
	if got := style.Card(); got != want {
		t.Errorf("card = %q, want %q", got, want)
	}

	presets, err := Presets()
	if err != nil {
		t.Fatal(err)
	}

	spec := presets[0]
	spec.Style = &StyleSpec{Look: strings.Repeat("x", maxText+100), Source: "  dużo   spacji  "}
	if err := spec.Normalize(); err != nil {
		t.Fatal(err)
	}

	if len([]rune(spec.Style.Look)) != maxText+1 || !strings.HasSuffix(spec.Style.Look, "…") || spec.Style.Source != "dużo spacji" {
		t.Errorf("style not clipped: %d runes, source %q", len([]rune(spec.Style.Look)), spec.Style.Source)
	}

	spec.Style = &StyleSpec{Source: "tylko hasło"}
	if err := spec.Normalize(); err != nil || spec.Style != nil {
		t.Errorf("a card without content should be dropped: %v %v", spec.Style, err)
	}
}

func TestClipEndsAtASentence(t *testing.T) {
	long := strings.Repeat("Deszcz padał na miasto. ", 30)
	if got := clip(long, 100); !strings.HasSuffix(got, "miasto.") || len([]rune(got)) > 100 {
		t.Errorf("clip = %q", got)
	}

	// Without a sentence end in reach the cut falls on a word, marked.
	words := strings.Repeat("neon ", 40)
	if got := clip(words, 50); !strings.HasSuffix(got, "neon…") || len([]rune(got)) > 51 {
		t.Errorf("clip = %q", got)
	}

	if got := clip("  Krótki   opis. ", 100); got != "Krótki opis." {
		t.Errorf("clip = %q", got)
	}
}

func TestScatteredRoomsAreClustered(t *testing.T) {
	spec := Spec{
		Name: "Neon", Intro: "Deszcz.", Mode: ModeInvestigation,
		Rooms: []RoomSpec{
			{Name: "Rynek", X: 10, Y: 2}, {Name: "Bar", X: 3, Y: 3}, {Name: "Dom", X: 8, Y: 5}, {Name: "Stacja", X: 14, Y: 3},
			{Name: "Park", X: 2, Y: 7}, {Name: "Cmentarz", X: 9, Y: 1}, {Name: "Poczta", X: 6, Y: 4}, {Name: "Radio", X: 12, Y: 7},
		},
		People: []PersonSpec{{Name: "Reed", Room: "Bar"}},
		Case:   &CaseSpec{Answer: "Reed", Client: "Reed"},
	}

	if connected(spec.Rooms) {
		t.Fatal("scattered rooms counted as connected")
	}

	if err := spec.Normalize(); err != nil {
		t.Fatal(err)
	}

	if !connected(spec.Rooms) || spec.Rooms[0].Name != "Rynek" || spec.Rooms[0].X != Width/2 || spec.Rooms[0].Y != Height/2 {
		t.Errorf("rooms after normalizing: %+v", spec.Rooms)
	}

	cells := map[Point]bool{}
	for _, room := range spec.Rooms {
		if cells[Point{X: room.X, Y: room.Y}] {
			t.Errorf("two rooms share %d,%d", room.X, room.Y)
		}

		cells[Point{X: room.X, Y: room.Y}] = true
	}

	// A map that already hangs together is left as it is.
	jeziorna := presets(t)[5]
	before := append([]RoomSpec(nil), jeziorna.Rooms...)
	if err := jeziorna.Normalize(); err != nil {
		t.Fatal(err)
	}

	for index, room := range jeziorna.Rooms {
		if room.X != before[index].X || room.Y != before[index].Y {
			t.Errorf("preset room %s moved to %d,%d", room.Name, room.X, room.Y)
		}
	}
}
