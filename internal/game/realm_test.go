package game

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

var (
	home     = world.Point{X: world.Width / 2, Y: world.Height / 2}
	eastTile = world.Point{X: home.X + 1, Y: home.Y}
	westTile = world.Point{X: home.X - 1, Y: home.Y}
	cache    = world.Point{X: home.X, Y: home.Y - 1}
	shrine   = world.Point{X: home.X, Y: home.Y + 1}
	outpost  = world.Point{X: home.X, Y: home.Y + 2}
)

// testRealm builds a small hand-made world around the first settlement:
// one rat to the east, two rats to the west, a guarded cache with a charm to
// the north, a shrine to the south with a second settlement behind it, and
// the boss in the top left corner.
func testRealm(t *testing.T) *Realm {
	t.Helper()

	spec := world.Spec{
		Name:     "Test",
		Currency: "kapsli",
		Boss:     world.BossSpec{Name: "Szef"},
		Places: []world.PlaceSpec{
			{Name: "Osada", Description: "Domy.", Danger: 0},
			{Name: "Las", Description: "Drzewa.", Terrain: "forest", Danger: 1},
			{Name: "Przystań", Description: "Łodzie.", Danger: 0},
		},
		Sites: []world.SiteSpec{
			{Name: "Skrytka", Description: "Skrzynia.", Kind: world.SiteTreasure},
			{Name: "Źródło", Description: "Woda.", Kind: world.SiteShrine},
		},
		Monsters: []world.MonsterSpec{{Name: "Szczur Polny", Tier: 1}},
		Items: []world.ItemSpec{
			{Name: "Kij", Kind: world.KindWeapon, Tier: 1},
			{Name: "Kaftan", Kind: world.KindArmor, Tier: 1},
			{Name: "Napar", Kind: world.KindPotion, Effect: world.EffectHeal, Tier: 1},
			{Name: "Wywar", Kind: world.KindPotion, Effect: world.EffectMana, Tier: 1},
			{Name: "Zwój Iskry", Kind: world.KindSpell, Effect: world.EffectBolt, Tier: 1},
			{Name: "Księga Fali", Kind: world.KindSpell, Effect: world.EffectBlast, Tier: 1},
			{Name: "Modlitwa", Kind: world.KindSpell, Effect: world.EffectHeal, Tier: 1},
			{Name: "Amulet Siły", Kind: world.KindCharm, Effect: world.EffectVigor, Tier: 1},
			{Name: "Topór", Kind: world.KindWeapon, Tier: 3},
		},
		Shops: []world.ShopSpec{
			{Name: "Zbrojownia", Kind: world.ShopArmory, Keeper: "Kowal"},
			{Name: "Chata Zielarki", Kind: world.ShopAlchemist, Keeper: "Jagna"},
		},
	}
	if err := spec.Normalize(); err != nil {
		t.Fatal(err)
	}

	built := &world.World{
		Spec:  spec,
		Tiles: make([]world.Tile, world.Width*world.Height),
		Towns: []world.Point{home, outpost},
	}

	for index := range built.Tiles {
		built.Tiles[index] = world.Tile{Place: 1, Town: -1, Site: -1}
	}

	built.Tiles[built.Index(home)] = world.Tile{Place: 0, Town: 0, Site: -1}
	built.Tiles[built.Index(outpost)] = world.Tile{Place: 2, Town: 1, Site: -1}
	built.Tiles[built.Index(eastTile)].Monsters = []int{0}
	built.Tiles[built.Index(westTile)].Monsters = []int{0, 0}
	built.Tiles[built.Index(cache)] = world.Tile{Place: 1, Town: -1, Site: 0, Monsters: []int{0}, Loot: []int{7}}
	built.Tiles[built.Index(shrine)] = world.Tile{Place: 1, Town: -1, Site: 1}
	built.Tiles[built.Index(built.Lair)].Boss = true

	return NewRealm(built, 1)
}

// flatten gives every hero and companion plain scores of 10, so that the
// numbers the tests check do not depend on callings or dice.
func flatten(realm *Realm) {
	for _, hero := range realm.players {
		hero.abilities = Abilities{10, 10, 10, 10, 10, 10}
		hero.hp, hero.mana = hero.maxHP(), hero.maxMana()

		for _, member := range hero.party {
			member.abilities = Abilities{10, 10, 10, 10, 10, 10}
			member.hp, member.mana = member.maxHealth(), member.maxEnergy()
		}
	}
}

// enlist lets a hero with plain scores into the world.
func enlist(realm *Realm, name string) string {
	joined := realm.Join(name, "")
	flatten(realm)

	return joined
}

// do runs typed commands for a player.
func do(realm *Realm, name string, lines ...string) {
	for _, line := range lines {
		realm.Execute(name, Parse(line))
	}
}

// win attacks until the fight is over, keeping the heroes alive.
func win(t *testing.T, realm *Realm, names ...string) {
	t.Helper()

	for range 300 {
		if realm.View(names[0]).Fight == nil {
			return
		}

		for _, name := range names {
			// Nobody falls: the heroes are kept well beyond any blow.
			realm.players[name].hp = 999
			realm.Attack(name, 0)
		}
	}

	t.Fatal("the fight never ended")
}

func lastLog(realm *Realm, name string) string {
	log := realm.View(name).Log

	return log[len(log)-1]
}

func logged(realm *Realm, name, text string) bool {
	return strings.Contains(strings.Join(realm.View(name).Log, "\n"), text)
}

func give(realm *Realm, name string, itemNames ...string) {
	for _, wanted := range itemNames {
		for _, item := range realm.world.Items {
			if item.Name == wanted {
				realm.players[name].bag = append(realm.players[name].bag, item)
			}
		}
	}
}

func TestJoinGivesUniqueNamesAndSharedView(t *testing.T) {
	realm := testRealm(t)

	first, second := realm.Join("  Ala  ", "wojownik"), realm.Join("Ala", "pilot")
	if first != "Ala" || second != "Ala 2" {
		t.Fatalf("names = %q, %q", first, second)
	}

	if got := enlist(realm, ""); got != defaultName {
		t.Errorf("empty name became %q", got)
	}

	view := realm.View(first)
	me := view.Me

	if len(view.Others) != 2 || me.HP != BaseHP || me.Mana != BaseMana || me.Gold != startGold || me.Avatar != "wojownik" || view.Tile.Town != 0 {
		t.Errorf("view = %+v", me)
	}

	if !logged(realm, first, "Zbrojownia (Kowal), Chata Zielarki (Jagna)") {
		t.Errorf("no buildings in the arrival text: %v", view.Log)
	}

	revealed := 0
	for _, seen := range view.Revealed {
		if seen {
			revealed++
		}
	}

	if revealed != 9 {
		t.Errorf("%d tiles revealed at the start, want 9", revealed)
	}

	if mark := view.Marks[view.World.Index(westTile)]; mark.Monsters != 2 {
		t.Errorf("mark west of the settlement = %+v", mark)
	}

	if mark := view.Marks[view.World.Index(view.World.Lair)]; mark.Boss {
		t.Error("the boss is visible before anyone got near")
	}
}

func TestWalkingByCommands(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].mana = 3

	do(realm, hero, "idź na południe")

	view := realm.View(hero)
	if view.Me.Pos != shrine || view.Tile.Site != 1 || view.Me.Mana != BaseMana {
		t.Fatalf("after walking south: %+v", view.Me)
	}

	for _, want := range []string{"Idziesz na południe.", "Las. Drzewa.", "Źródło. Woda.", "Wracają ci siły"} {
		if !logged(realm, hero, want) {
			t.Errorf("log misses %q: %v", want, view.Log)
		}
	}

	realm.players[hero].mana = 3
	do(realm, hero, "s")

	view = realm.View(hero)
	if view.Me.Pos != outpost || view.Tile.Town != 1 || view.Me.Mana != 3+manaPerStep {
		t.Errorf("after the second step: %+v", view.Me)
	}

	do(realm, hero, "rozejrzyj się")
	if !logged(realm, hero, "Drogi: północ: Las, wschód: Las, południe: Las, zachód: Las.") {
		t.Errorf("no exits after looking around: %v", realm.View(hero).Log)
	}

	do(realm, hero, "idź")
	if !strings.Contains(lastLog(realm, hero), "Dokąd?") {
		t.Errorf("walking nowhere: %q", lastLog(realm, hero))
	}

	realm.players[hero].pos = world.Point{}
	do(realm, hero, "zachód")

	if realm.View(hero).Me.Pos != (world.Point{}) || lastLog(realm, hero) != "Dalej nie ma przejścia." {
		t.Errorf("walked off the map: %q", lastLog(realm, hero))
	}

	do(realm, hero, "zatańcz")
	if !strings.Contains(lastLog(realm, hero), "Nie rozumiem") {
		t.Errorf("unknown command: %q", lastLog(realm, hero))
	}
}

func TestSoloFight(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "e")

	view := realm.View(hero)
	if view.Fight == nil || len(view.Fight.Monsters) != 1 || view.Me.Pos != eastTile {
		t.Fatalf("no fight after stepping onto a monster: %+v", view.Fight)
	}

	do(realm, hero, "idź na wschód", "podnieś wszystko", "odpocznij")

	if realm.View(hero).Me.Pos != eastTile || lastLog(realm, hero) != "Najpierw walka albo ucieczka." {
		t.Errorf("left or looted in the middle of a fight: %q", lastLog(realm, hero))
	}

	do(realm, hero, "atakuj smoka")
	if lastLog(realm, hero) != "Nie ma tu takiego przeciwnika." || realm.View(hero).Fight.Round != 1 {
		t.Errorf("attacking a monster that is not there: %q", lastLog(realm, hero))
	}

	// The rat is quicker than the hero: it struck first, and after the
	// hero's blow the second round opens with its strike again.
	do(realm, hero, "atakuj szczura")

	view = realm.View(hero)
	if view.Fight.Round != 2 || view.Fight.Actor == nil || view.Me.HP >= BaseHP || view.Fight.Monsters[0].HP >= view.Fight.Monsters[0].MaxHP {
		t.Errorf("after one round: round %d, actor %v, hp %d, monster %+v", view.Fight.Round, view.Fight.Actor, view.Me.HP, view.Fight.Monsters[0])
	}

	if order := view.Fight.Order; len(order) != 2 || !order[0].Foe || order[1].Name != "Ala" || !order[1].Current {
		t.Errorf("order = %+v", order)
	}

	win(t, realm, hero)

	view = realm.View(hero)
	if view.Me.Gold <= startGold || view.Marks[view.World.Index(eastTile)].Monsters != 0 {
		t.Errorf("after victory: gold %d, marks %+v", view.Me.Gold, view.Marks[view.World.Index(eastTile)])
	}

	do(realm, hero, "w", "e")

	if realm.View(hero).Fight != nil {
		t.Error("beaten monsters came back")
	}
}

func TestSharedFightWaitsForEveryone(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "w")
	do(realm, bob, "w")

	view := realm.View(bob)
	if view.Fight == nil || len(view.Fight.Members) != 2 || len(view.Fight.Monsters) != 2 {
		t.Fatalf("bob did not join the fight: %+v", view.Fight)
	}

	before := realm.View(ala).Me.HP
	do(realm, ala, "atakuj", "atakuj")

	view = realm.View(ala)
	if view.Fight.Round != 1 || view.Fight.Actor != nil || view.Fight.Waiting != "Bob" || view.Me.HP != before {
		t.Errorf("the round went on without bob: round %d, waiting %q, hp %d -> %d", view.Fight.Round, view.Fight.Waiting, before, view.Me.HP)
	}

	if got := view.Fight.Monsters[0].HP; got < 12-4 {
		t.Errorf("ala acted twice in one round: monster at %d", got)
	}

	if !logged(realm, ala, "Teraz tura: Bob. Czekasz.") {
		t.Error("ala was not told whose turn it is")
	}

	realm.Attack(bob, 1)

	if got := realm.View(ala).Fight.Round; got != 2 {
		t.Errorf("round = %d after everyone acted, want 2", got)
	}

	win(t, realm, ala, bob)

	if a, b := realm.View(ala).Me.Gold, realm.View(bob).Me.Gold; a != b || a <= startGold {
		t.Errorf("loot not shared equally: %d and %d", a, b)
	}
}

func TestRoundTimeoutSkipsIdlePlayers(t *testing.T) {
	realm := testRealm(t)
	realm.SetRoundTimeout(10 * time.Millisecond)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "w")
	do(realm, bob, "w")
	realm.Attack(ala, 0)

	deadline := time.After(2 * time.Second)
	for realm.View(ala).Fight.Round == 1 {
		select {
		case <-realm.Updates(ala):
		case <-deadline:
			t.Fatal("the round never timed out")
		}
	}
}

func TestFallenHeroWakesInTheLastSettlement(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "s", "s")
	realm.players[hero].pos, realm.players[hero].prev = home, home
	do(realm, hero, "e")
	realm.players[hero].hp = 1
	realm.Attack(hero, 0)

	view := realm.View(hero)
	if view.Fight != nil || view.Me.Pos != outpost || view.Me.HP != BaseHP/2 || view.Me.Gold != startGold-startGold*fallLossPercent/100 {
		t.Errorf("after falling: %+v", view.Me)
	}

	if foe := realm.monsters[view.World.Index(eastTile)][0]; foe.hp != foe.maxHP {
		t.Errorf("abandoned monster did not recover: %d/%d", foe.hp, foe.maxHP)
	}

	if len(realm.fights) != 0 {
		t.Error("an empty fight was left behind")
	}
}

func TestFlee(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "e")

	for range 100 {
		if realm.View(hero).Fight == nil {
			break
		}

		realm.players[hero].hp = BaseHP
		do(realm, hero, "uciekaj")
	}

	view := realm.View(hero)
	if view.Fight != nil || view.Tile.Town != 0 || len(realm.fights) != 0 {
		t.Errorf("did not escape back to the settlement: %+v", view.Me)
	}
}

func TestItemsOnTheGround(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "podnieś wszystko")
	if lastLog(realm, ala) != "Nic tu nie leży." {
		t.Errorf("picking up nothing: %q", lastLog(realm, ala))
	}

	do(realm, ala, "n")
	if !logged(realm, ala, "Skrytka. Skrzynia.") || realm.View(ala).Fight == nil {
		t.Fatal("the cache is not guarded")
	}

	win(t, realm, ala)

	view := realm.View(ala)
	if len(view.Ground) == 0 || view.Ground[0].Name != "Amulet Siły" || !logged(realm, ala, "Na ziemi leży: Amulet Siły") {
		t.Fatalf("ground = %+v", view.Ground)
	}

	do(realm, ala, "podnieś miecz")
	if lastLog(realm, ala) != "Nie widzisz tu niczego takiego." {
		t.Errorf("picking up something absent: %q", lastLog(realm, ala))
	}

	do(realm, ala, "podnieś amulet", "załóż amulet siły")

	view = realm.View(ala)
	if view.Me.Charm != "Amulet Siły" || view.Me.MaxHP != BaseHP+10 {
		t.Errorf("charm not worn: %+v", view.Me)
	}

	give(realm, ala, "Kij")
	do(realm, ala, "upuść kij")
	do(realm, bob, "n", "weź kij")

	if bag := realm.View(bob).Me.Bag; len(bag) != 1 || bag[0].Name != "Kij" {
		t.Errorf("bob did not get the dropped stick: %+v", bag)
	}

	for _, item := range realm.View(ala).Ground {
		if item.Name == "Kij" {
			t.Error("the stick is still on the ground")
		}
	}
}

func TestEquipmentAndPotions(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	give(realm, hero, "Kij", "Kaftan", "Napar", "Wywar", "Amulet Siły")

	do(realm, hero, "załóż kij", "ubierz kaftan", "użyj amuletu")

	view := realm.View(hero)
	me := view.Me

	if me.Weapon != "Kij" || me.Armor != "Kaftan" || me.Charm != "Amulet Siły" || me.Attack != baseAttack+3 || me.Defense != 1 || me.MaxHP != BaseHP+10 {
		t.Fatalf("equipment = %+v", me)
	}

	do(realm, hero, "wypij wywar")
	if got := len(realm.View(hero).Me.Bag); got != 2 {
		t.Errorf("a potion was wasted at full power: %d items left", got)
	}

	realm.players[hero].mana = 1
	do(realm, hero, "wypij wywar")

	if got := realm.View(hero).Me.Mana; got != 9 {
		t.Errorf("mana = %d after the potion, want 9", got)
	}

	do(realm, hero, "e")
	realm.players[hero].hp = 10
	do(realm, hero, "załóż kij", "użyj naparu")

	view = realm.View(hero)
	if len(view.Me.Bag) != 0 || view.Fight.Round != 2 || view.Me.HP <= 10 || !logged(realm, hero, "Ala nie ma niczego takiego.") {
		t.Errorf("potion in a fight: bag %d, round %d, hp %d", len(view.Me.Bag), view.Fight.Round, view.Me.HP)
	}

	realm.Use(hero, 5)
}

func TestSpells(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "rzuć iskrę")
	if !strings.Contains(lastLog(realm, ala), "Nie znasz jeszcze żadnych mocy") {
		t.Errorf("casting without spells: %q", lastLog(realm, ala))
	}

	give(realm, ala, "Zwój Iskry", "Księga Fali", "Modlitwa", "Zwój Iskry")
	do(realm, ala, "naucz się iskry", "przeczytaj księgę", "użyj modlitwy", "użyj zwoju")

	view := realm.View(ala)
	if len(view.Me.Spells) != 3 || len(view.Me.Bag) != 1 || lastLog(realm, ala) != "Już to umiesz." {
		t.Fatalf("spells = %+v, bag = %+v", view.Me.Spells, view.Me.Bag)
	}

	do(realm, ala, "rzuć")
	if !strings.Contains(lastLog(realm, ala), "Do wyboru: Zwój Iskry, Księga Fali, Modlitwa") {
		t.Errorf("casting without naming a spell: %q", lastLog(realm, ala))
	}

	do(realm, ala, "rzuć iskrę")
	if lastLog(realm, ala) != "Nie ma tu z kim walczyć." {
		t.Errorf("bolt outside a fight: %q", lastLog(realm, ala))
	}

	realm.players[ala].hp = 20
	do(realm, ala, "rzuć modlitwę")

	if me := realm.View(ala).Me; me.HP != 35 || me.Mana != BaseMana-5 {
		t.Errorf("healing outside a fight: %+v", me)
	}

	do(realm, ala, "w")
	do(realm, bob, "w")

	rats := realm.monsters[realm.world.Index(westTile)]
	fresh := func() {
		rats[0].hp, rats[1].hp = 100, 100
		realm.players[ala].mana = BaseMana
		realm.players[ala].hp, realm.players[bob].hp = BaseHP, BaseHP
	}

	fresh()
	do(realm, ala, "rzuć iskrę na szczura")

	view = realm.View(ala)
	if rats[0].hp < 88 || rats[0].hp > 90 || rats[1].hp != 100 || view.Me.Mana != BaseMana-4 || view.Fight.Actor != nil {
		t.Errorf("bolt: rats at %d and %d, mana %d", rats[0].hp, rats[1].hp, view.Me.Mana)
	}

	do(realm, bob, "atakuj")
	fresh()
	do(realm, ala, "rzuć falę")

	if rats[0].hp != 94 || rats[1].hp != 94 || realm.players[ala].mana != BaseMana-6 {
		t.Errorf("blast: rats at %d and %d", rats[0].hp, rats[1].hp)
	}

	do(realm, bob, "atakuj")
	fresh()
	realm.players[ala].hp, realm.players[bob].hp = 5, 5
	do(realm, ala, "rzuć modlitwę")

	if a, b := realm.players[ala].hp, realm.players[bob].hp; a != 20 || b != 20 {
		t.Errorf("party heal: %d and %d", a, b)
	}

	do(realm, bob, "atakuj")
	fresh()
	realm.players[ala].mana = 2
	do(realm, ala, "rzuć falę")

	if !strings.Contains(lastLog(realm, ala), "Za mało mocy") || realm.View(ala).Fight.Actor == nil {
		t.Errorf("casting without power: %q", lastLog(realm, ala))
	}
}

func TestShops(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "rozmawiaj")
	if realm.View(hero).Shop != nil || !strings.Contains(lastLog(realm, hero), "Dokąd chcesz wejść?") {
		t.Errorf("two shops, none named: %q", lastLog(realm, hero))
	}

	do(realm, hero, "idź do zbrojowni")

	shop := realm.View(hero).Shop
	// No shop here specialises in magic, so the first one sells it too.
	if shop == nil || shop.Keeper != "Kowal" || len(shop.Wares) != 6 {
		t.Fatalf("armory = %+v", shop)
	}

	request, ok := realm.HaggleRequest(hero, "dam 30 za kij")
	if !ok || len(request.Wares) != 6 {
		t.Fatalf("request = %+v", request)
	}

	decision, narrate, ok := realm.Haggle(hero, "dam 30 za kij", Intent{Action: ActionOffer, ItemID: "item-1", Amount: 30})
	if !ok || decision.Outcome != OutcomeSold || narrate.Merchant.Name != "Kowal" {
		t.Fatalf("decision = %+v", decision)
	}

	view := realm.View(hero)
	if view.Me.Gold != startGold-30 || len(view.Me.Bag) != 1 || view.Me.Bag[0].Name != "Kij" || len(view.Shop.Wares) != 5 {
		t.Errorf("after buying: gold %d, bag %+v", view.Me.Gold, view.Me.Bag)
	}

	if decision, _, _ := realm.Haggle(hero, "", Intent{Action: ActionOffer, ItemID: "item-2", Amount: 31}); decision.Outcome != OutcomeNoGold {
		t.Errorf("bought armor with gold already spent: %s", decision.Outcome)
	}

	serial := view.Shop.Serial
	realm.CloseShop(hero)
	do(realm, hero, "zielarka")

	if got := realm.View(hero).Shop; got == nil || got.Keeper != "Jagna" || len(got.Wares) != 2 {
		t.Fatalf("herbalist by a bare name: %+v", got)
	}

	do(realm, hero, "pogadaj z kowalem")
	if got := realm.View(hero).Shop; got.Serial != serial || len(got.Wares) != 5 {
		t.Errorf("returning to the armory started a new talk: %+v", got)
	}

	do(realm, hero, "s")
	if realm.View(hero).Shop != nil {
		t.Error("the shop followed the hero out of the settlement")
	}

	do(realm, hero, "kup coś")
	if !strings.Contains(lastLog(realm, hero), "Kupcy są w osadach") {
		t.Errorf("shopping in the wild: %q", lastLog(realm, hero))
	}

	do(realm, hero, "s", "zbrojownia")
	if got := realm.View(hero).Shop; got == nil || len(got.Wares) != 7 {
		t.Errorf("the second settlement should sell every tier: %+v", got)
	}
}

func TestLeaveAndReturn(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "w")
	do(realm, bob, "w")
	realm.Attack(ala, 0)
	realm.Leave(bob)

	view := realm.View(ala)
	if len(view.Fight.Members) != 1 || view.Fight.Round != 2 || len(view.Others) != 0 {
		t.Errorf("fight did not go on without bob: %+v", view.Fight)
	}

	if got := enlist(realm, "Bob"); got != bob {
		t.Fatalf("returning player got a new character: %q", got)
	}

	if got := realm.View(bob).Fight; got == nil || len(got.Members) != 2 {
		t.Errorf("bob did not rejoin the fight on his tile: %+v", got)
	}
}

func TestSayReachesEveryone(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "powiedz Idę na Północ, kto ze mną?")

	if want := "Ala: «Idę na Północ, kto ze mną?»"; lastLog(realm, bob) != want || lastLog(realm, ala) != want {
		t.Errorf("bob heard %q", lastLog(realm, bob))
	}
}

func TestBeatingTheBossWinsTheWorld(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	realm.players[ala].pos = world.Point{X: realm.world.Lair.X + 1, Y: realm.world.Lair.Y}
	if realm.world.Lair.X > 0 {
		realm.players[ala].pos.X = realm.world.Lair.X - 1
		do(realm, ala, "e")
	} else {
		do(realm, ala, "w")
	}

	if fight := realm.View(ala).Fight; fight == nil || !fight.Monsters[0].Boss {
		t.Fatalf("no boss fight: %+v", fight)
	}

	win(t, realm, ala)

	if !realm.View(ala).Won || !realm.View(bob).Won {
		t.Error("the world is not won after the boss fell")
	}
}

func TestSaveAndRestore(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	built, err := world.Build(presets[0])
	if err != nil {
		t.Fatal(err)
	}

	realm := NewRealm(built, 1)
	ala, bob := realm.Join("Ala", "lowczyni"), realm.Join("Bob", "pilot")
	flatten(realm)
	give(realm, ala, "Zardzewiały Kord", "Napar z Tataraku", "Zwój Błędnej Iskry", "Pierścień Strażnika", "Skórzany Kaftan")
	do(realm, ala, "załóż kord", "naucz się iskry", "załóż pierścień", "upuść kaftan", "s", "s")
	do(realm, bob, "n")
	realm.players[ala].gold, realm.players[ala].hp = 123, 17

	dir := t.TempDir()

	path, err := WriteState(dir, realm.State(ala))
	if err != nil {
		t.Fatal(err)
	}

	if want := dir + "/zatopione-krypty.json"; path != want {
		t.Errorf("path = %s, want %s", path, want)
	}

	states := ReadStates(dir)
	if len(states) != 1 || states[0].Host != ala || len(states[0].Players) != 2 {
		t.Fatalf("states = %+v", states)
	}

	restored, err := Restore(states[0], 2)
	if err != nil {
		t.Fatal(err)
	}

	if view := restored.View(ala); len(view.Others) != 0 {
		t.Errorf("heroes are connected right after loading: %+v", view.Others)
	}

	if got := restored.Join("Ala", "ignored"); got != ala {
		t.Fatalf("the saved hero was not resumed: %q", got)
	}

	before, after := realm.View(ala), restored.View(ala)
	before.Log, after.Log = nil, nil
	before.Others, after.Others = nil, nil

	// Fights are not saved: the resumed one opens anew, with the monsters'
	// first blows struck again.
	before.Blows, after.Blows = nil, nil

	if before.Fight != nil {
		before.Fight, after.Fight = nil, nil
		before.Me.Fighting, after.Me.Fighting = false, false
		before.Me.HP, after.Me.HP = 0, 0
	}

	if !reflect.DeepEqual(before, after) {
		t.Errorf("the restored game differs:\n before %+v\n after  %+v", before.Me, after.Me)
	}

	restoredBob := restored.players[bob]
	if restoredBob == nil || restoredBob.pos != realm.players[bob].pos || restoredBob.avatar != "pilot" {
		t.Errorf("bob was not saved: %+v", restoredBob)
	}

	if ground := restored.ground[built.Index(built.Start())]; len(ground) != 1 || ground[0].Name != "Skórzany Kaftan" {
		t.Errorf("the dropped armor is gone: %+v", ground)
	}

	states[0].Revealed = nil
	if _, err := Restore(states[0], 1); err == nil {
		t.Error("a damaged save was accepted")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		text string
		want Command
	}{
		{"Idź na wschód", Command{Verb: VerbGo, Dir: east}},
		{"idz na zachod", Command{Verb: VerbGo, Dir: west}},
		{"północ", Command{Verb: VerbGo, Dir: north}},
		{"na południe", Command{Verb: VerbGo, Dir: south}},
		{"w", Command{Verb: VerbGo, Dir: west}},
		{"idź", Command{Verb: VerbGo}},
		{"idź do zbrojowni", Command{Verb: VerbVisit, Arg: "zbrojowni"}},
		{"porozmawiaj z Morią", Command{Verb: VerbVisit, Arg: "moria"}},
		{"Podnieś Miecz Runiczny!", Command{Verb: VerbTake, Arg: "miecz runiczny"}},
		{"weź wszystko", Command{Verb: VerbTake, Arg: "wszystko"}},
		{"upuść kord", Command{Verb: VerbDrop, Arg: "kord"}},
		{"załóż kolczugę", Command{Verb: VerbUse, Arg: "kolczuge"}},
		{"naucz się iskry", Command{Verb: VerbUse, Arg: "iskry"}},
		{"atakuj topielca", Command{Verb: VerbAttack, Arg: "topielca"}},
		{"a", Command{Verb: VerbAttack}},
		{"rzuć kulę ognia na topielca", Command{Verb: VerbCast, Arg: "kule ognia", Target: "topielca"}},
		{"rzuć runę gromu", Command{Verb: VerbCast, Arg: "rune gromu"}},
		{"uciekaj!", Command{Verb: VerbFlee}},
		{"odpocznij", Command{Verb: VerbRest}},
		{"rozejrzyj się", Command{Verb: VerbLook}},
		{"powiedz Cześć, Bob!", Command{Verb: VerbSay, Arg: "Cześć, Bob!"}},
		{"plecak", Command{Verb: VerbBag}},
		{"zapisz", Command{Verb: VerbSave}},
		{"?", Command{Verb: VerbHelp}},
		{"wyjdź", Command{Verb: VerbQuit}},
		{"zielarka", Command{Arg: "zielarka"}},
		{"   ", Command{}},
	}

	for _, test := range tests {
		if got := Parse(test.text); got != test.want {
			t.Errorf("Parse(%q) = %+v, want %+v", test.text, got, test.want)
		}
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		name, query string
		want        bool
	}{
		{"Miecz Runiczny", "mieczem", true},
		{"Miecz Runiczny", "runiczny miecz", true},
		{"Zardzewiały Kord", "zardzewialy", true},
		{"Zwój Błędnej Iskry", "iskrę", true},
		{"Napar z Tataraku", "z naparu", true},
		{"Miecz Runiczny", "topór", false},
		{"Miecz Runiczny", "miecz ognisty", false},
		{"Miecz Runiczny", "", false},
		{"Miecz Runiczny", "na do", false},
	}

	for _, test := range tests {
		if got := Matches(test.name, test.query); got != test.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", test.name, test.query, got, test.want)
		}
	}
}

func TestTalkAboutTheGoodsPointsAtThem(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")

	// A kind of ware asked for without a name goes to the wares of that
	// kind, not to whatever fact has the word in its body.
	_, request, ok := realm.Haggle(hero, "jaką masz najlepszą zbroję?", Intent{Action: ActionChat})
	if !ok || !strings.Contains(request.Hint, "Pasuje: «Kaftan» 35") || strings.Contains(request.Hint, "Kij") {
		t.Errorf("hint about armour = %q", request.Hint)
	}

	// A bare greeting is answered as a greeting; banter with no fact in it
	// still gets a direction instead of nothing.
	_, request, ok = realm.Haggle(hero, "siema mordo!", Intent{Action: ActionChat})
	if !ok || !strings.Contains(request.Hint, "tylko się wita") {
		t.Errorf("hint for a greeting = %q", request.Hint)
	}

	_, request, ok = realm.Haggle(hero, "siema, nudno dziś u ciebie", Intent{Action: ActionChat})
	if !ok || request.Hint == "" || strings.Contains(request.Hint, "wita") || strings.Contains(request.Hint, "Pasuje") {
		t.Errorf("hint for banter = %q", request.Hint)
	}
}

func TestWhatAKeeperLacksBecomesALead(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")
	herbalist := realm.shopOfKind(world.ShopAlchemist)
	fact := realm.add("Kurhany Królów", "Pod kurhanami śpi stary król, a jego miecz świeci nocą. Nikt stamtąd nie wrócił.", "", herbalist)

	// The smith lacks the fact: he passes on the crumb he heard, names
	// who knows it, and the name must come out right.
	_, request, ok := realm.Haggle(hero, "co wiesz o kurhanach?", Intent{Action: ActionChat})
	if !ok || !strings.Contains(request.Hint, "czego nie znasz: Kurhany Królów. Słyszałeś tylko tyle: „Pod kurhanami śpi stary król, a jego miecz świeci nocą”.") ||
		!strings.Contains(request.Hint, "zna sprawę: Jagna (Chata Zielarki w każdej osadzie). Imię podaj dokładnie tak.") {
		t.Errorf("referral hint = %q", request.Hint)
	}

	if len(request.Required) != 1 || request.Required[0] != "Jagna" || request.Lead != "Trop: Kurhany Królów → Jagna." {
		t.Errorf("required %q, lead %q", request.Required, request.Lead)
	}

	journal := realm.Journal(hero)
	if len(journal) != 1 || journal[0].Lead != "Jagna" || journal[0].Text != "" || JournalLines(journal)[1] != "• Trop: Kurhany Królów — pytaj: Jagna" {
		t.Errorf("journal = %+v", journal)
	}

	// Asking further stays on the lead instead of a random fact, and
	// the lead is not written down twice.
	_, request, ok = realm.Haggle(hero, "a co dokładnie słyszałeś?", Intent{Action: ActionChat})
	if !ok || !strings.Contains(request.Hint, "Gracz dopytuje o to samo: Kurhany Królów.") || !strings.Contains(request.Hint, "więcej wie Jagna") || request.Lead == "" || len(realm.Journal(hero)) != 1 {
		t.Errorf("follow-up hint = %q, journal %d", request.Hint, len(realm.Journal(hero)))
	}

	// Another topic lets the lead go.
	_, request, _ = realm.Haggle(hero, "siema mordo", Intent{Action: ActionChat})
	if request.Lead != "" || !strings.Contains(request.Hint, "tylko się wita") || realm.players[hero].talks[realm.players[hero].shop].lead != nil {
		t.Errorf("after a greeting: lead %q, hint %q", request.Lead, request.Hint)
	}

	// Hearing the fact itself turns the lead into knowledge.
	realm.learn(realm.players[hero], fact)
	if journal := realm.Journal(hero); len(journal) != 1 || journal[0].Lead != "" || journal[0].Text == "" {
		t.Errorf("journal after hearing = %+v", journal)
	}

	if crumb(&Fact{Text: "Jedno dwa trzy cztery pięć sześć siedem osiem dziewięć dziesięć jedenaście dwanaście trzynaście czternaście"}) != "Jedno dwa trzy cztery pięć sześć siedem osiem dziewięć dziesięć jedenaście dwanaście …" {
		t.Error("a long crumb is not cut")
	}
}

func TestKeepersKnowTheWorld(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].hp = 5
	give(realm, hero, "Kij")
	do(realm, hero, "załóż kij", "idź do zbrojowni")

	_, request, ok := realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	if !ok {
		t.Fatal("no talk")
	}

	for _, want := range []string{
		"Twój towar i ceny za sztukę (kapsli): «Kij» 40, «Kaftan» 35,",
		"Świat: Test.", "Osada, w której mieszkasz: Osada.", "Twoi sąsiedzi w osadzie: Jagna (Chata Zielarki).",
		"Inna osada: Przystań, kierunek południe, blisko.",
		"Największe zagrożenie: Szef.", "Jego siedziba: kierunek północny zachód, daleko.",
		"Wiesz: Szef.", "Twój rozmówca: Ala, ciężko ranny. Broń: Kij. Pancerz: brak.",
	} {
		if !strings.Contains(request.Lore, want) {
			t.Errorf("lore misses %q:\n%s", want, request.Lore)
		}
	}
}

func TestNarrateAllowsNumbersFromNames(t *testing.T) {
	request := NarrateRequest{
		Decision: Decision{Outcome: OutcomeChat},
		Lore:     "Osada, w której mieszkasz: Schron 7.",
	}

	narrator := &scriptedNarrator{replies: []string{"W Schronie 7 zawsze tak było."}}
	if text, err := Narrate(context.Background(), narrator, request); err != nil || !strings.Contains(text, "Schronie 7") {
		t.Errorf("a name with a digit was rejected: %q", text)
	}

	narrator = &scriptedNarrator{replies: []string{"Oddam ci to za 15."}}
	if _, err := Narrate(context.Background(), narrator, request); err == nil {
		t.Error("a made-up price slipped through in small talk")
	}

	// In small talk the keeper may quote their own list prices.
	request.Lore = "Twój towar i ceny za sztukę (kapsli): «Kij» 40."
	narrator = &scriptedNarrator{replies: []string{"Na psy weź Kij, 40 i jest twój."}}

	if _, err := Narrate(context.Background(), narrator, request); err != nil {
		t.Errorf("a list price was rejected in small talk: %v", err)
	}
}

func TestExperienceAndLevels(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	do(realm, ala, "e")
	win(t, realm, ala)

	if me := realm.View(ala).Me; me.Level != 1 || me.XP != 10 || me.NextXP != 40 || !logged(realm, ala, "doświadczenie: 10") {
		t.Fatalf("after one rat: %+v", me)
	}

	do(realm, ala, "w", "w")
	do(realm, bob, "w")
	realm.players[ala].xp = 25
	win(t, realm, ala, bob)

	me := realm.View(ala).Me
	if me.Level != 2 || me.XP != 5 || me.NextXP != 80 || me.MaxHP != BaseHP+hpPerLevel || me.MaxMana != BaseMana+manaPerLevel || me.Attack != baseAttack+1 {
		t.Errorf("after levelling up: %+v", me)
	}

	if me.HP != me.MaxHP || me.Mana != me.MaxMana || !logged(realm, ala, "Awans! Poziom 2") {
		t.Errorf("a new level should restore the hero: %+v", me)
	}

	if other := realm.View(bob).Me; other.XP != 20 || other.Level != 1 {
		t.Errorf("experience is not shared in full: %+v", other)
	}

	hero := realm.players[ala]
	realm.reward(hero, 100000)

	if hero.level != MaxLevel || hero.xp != 0 {
		t.Errorf("level %d with %d xp, want the cap", hero.level, hero.xp)
	}

	restored, err := Restore(realm.State(ala), 1)
	if err != nil {
		t.Fatal(err)
	}

	if got := restored.players[ala]; got.level != MaxLevel || got.maxHP() != hero.maxHP() {
		t.Errorf("level lost in the save: %+v", got)
	}
}

func TestCastByPosition(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	give(realm, hero, "Zwój Iskry", "Księga Fali")
	do(realm, hero, "użyj zwoju", "użyj księgi", "w")

	rats := realm.monsters[realm.world.Index(westTile)]
	rats[0].hp, rats[1].hp = 100, 100
	rats[0].attack, rats[1].attack = 1, 1

	realm.Cast(hero, 0, 1)

	if rats[0].hp != 100 || rats[1].hp > 90 || realm.View(hero).Me.Mana != BaseMana-4 {
		t.Errorf("bolt at the second rat: %d and %d", rats[0].hp, rats[1].hp)
	}

	realm.players[hero].hp, realm.players[hero].mana = BaseHP, BaseMana
	realm.Cast(hero, 1, 0)

	if rats[0].hp != 94 {
		t.Errorf("blast: first rat at %d", rats[0].hp)
	}

	realm.Cast(hero, 7, 0)
}

func TestKeepersTalkItemsIntoTheWorld(t *testing.T) {
	realm := testRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")

	if _, ok := realm.Invent(ala, world.ItemSpec{Name: "Kij", Kind: world.KindWeapon}); ok {
		t.Error("an item was invented outside a shop")
	}

	do(realm, ala, "idź do zbrojowni")
	before := len(realm.View(ala).Shop.Wares)

	item, ok := realm.Invent(ala, world.ItemSpec{Name: " Hełm  z Rondem ", Description: "Garnek z daszkiem.", Kind: world.KindArmor, Effect: "bolt", Tier: 9})
	if !ok || item.ID != "extra-1" || item.Name != "Hełm z Rondem" || item.Effect != world.EffectNone || item.Tier != 3 {
		t.Fatalf("invented = %+v, ok=%v", item, ok)
	}

	if !logged(realm, ala, "Do świata wpada: Hełm z Rondem (210 kapsli). Pancerz, obrona +5.") {
		t.Errorf("no word of the new item: %v", realm.View(ala).Log)
	}

	for _, double := range []world.ItemSpec{
		{Name: "Hełm z rondem", Kind: world.KindArmor},
		{Name: "kije", Kind: world.KindWeapon},
		{Name: "", Kind: world.KindWeapon},
		{Name: "Dziwo", Kind: "relic"},
	} {
		if _, ok := realm.Invent(ala, double); ok {
			t.Errorf("%q was invented although it exists or is unusable", double.Name)
		}
	}

	wares := realm.View(ala).Shop.Wares
	if len(wares) != before+1 || wares[len(wares)-1].ID != "extra-1" || wares[len(wares)-1].Price != 210 {
		t.Errorf("wares = %+v", wares)
	}

	realm.players[ala].gold = 1000
	decision, _, _ := realm.Haggle(ala, "biorę hełm", Intent{Action: ActionAccept, ItemID: "extra-1"})

	if decision.Outcome != OutcomeSold {
		t.Fatalf("buying the invented item: %+v", decision)
	}

	bag := realm.View(ala).Me.Bag
	if len(bag) != 1 || bag[0].Name != "Hełm z Rondem" {
		t.Fatalf("bag = %+v", bag)
	}

	do(realm, ala, "załóż hełm")
	if me := realm.View(ala).Me; me.Armor != "Hełm z Rondem" || me.Defense != 5 {
		t.Errorf("the invented armor does not work: %+v", me)
	}

	// Bob's talk with the same keeper, begun later, has the item too.
	do(realm, bob, "idź do zbrojowni")
	if wares := realm.View(bob).Shop.Wares; len(wares) != before+1 {
		t.Errorf("bob's wares = %+v", wares)
	}

	do(realm, bob, "zielarka")
	if wares := realm.View(bob).Shop.Wares; len(wares) != 2 {
		t.Errorf("the herbalist got the armourer's item: %+v", wares)
	}

	restored, err := Restore(realm.State(ala), 1)
	if err != nil {
		t.Fatal(err)
	}

	if hero := restored.players[ala]; hero.armor == nil || hero.armor.ID != "extra-1" || hero.armor.Name != "Hełm z Rondem" {
		t.Errorf("the invented item did not survive the save: %+v", hero.armor)
	}

	restored.Join("Ala", "")
	do(restored, ala, "idź do zbrojowni")

	if wares := restored.View(ala).Shop.Wares; len(wares) != before+1 {
		t.Errorf("restored wares = %+v", wares)
	}
}

// echoNarrator answers with a fixed line and keeps the requests it saw.
type echoNarrator struct {
	mu       sync.Mutex
	line     string
	requests []NarrateRequest
}

func (n *echoNarrator) Narrate(_ context.Context, request NarrateRequest) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.requests = append(n.requests, request)

	return n.line, nil
}

func TestGameMasterNarratesRounds(t *testing.T) {
	realm := testRealm(t)
	master := &echoNarrator{line: "Szczur skacze, ale Ala jest szybsza."}

	var (
		recordMu sync.Mutex
		recorded []string
	)

	realm.SetMaster(master, func(_ NarrateRequest, text string) {
		recordMu.Lock()
		defer recordMu.Unlock()

		recorded = append(recorded, text)
	})

	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")
	do(realm, ala, "w")
	do(realm, bob, "w")
	do(realm, ala, "atakuj")
	do(realm, bob, "atakuj")

	deadline := time.After(2 * time.Second)
	for !logged(realm, bob, MasterMark+"Szczur skacze") {
		select {
		case <-realm.Updates(bob):
		case <-deadline:
			t.Fatalf("the master never spoke: %v", realm.View(bob).Log)
		}
	}

	master.mu.Lock()
	request := master.requests[0]
	master.mu.Unlock()

	if request.Decision.Outcome != OutcomeScene || request.Merchant.Name != MasterName {
		t.Errorf("request = %+v", request.Decision)
	}

	for _, want := range []string{"Miejsce walki: Las.", "- Ala, poziom 1, zdrowie", "- Szczur Polny (bestia), zdrowie", "broń: brak"} {
		if !strings.Contains(request.Lore, want) {
			t.Errorf("lore misses %q:\n%s", want, request.Lore)
		}
	}

	if !strings.Contains(request.Decision.Scene, "Ala atakuje: Szczur Polny traci") || !strings.Contains(request.Decision.Brief(), "Przebieg rundy:") {
		t.Errorf("scene = %q", request.Decision.Scene)
	}

	// The master may quote the round's own numbers and nothing else.
	if !NumbersOK(request.Decision.Scene, request.Decision.Numbers()) || NumbersOK("Szczur traci 99 zdrowia.", request.Decision.Numbers()) {
		t.Errorf("the allowed numbers %v do not match the round %q", request.Decision.Numbers(), request.Decision.Scene)
	}

	recordMu.Lock()
	defer recordMu.Unlock()

	if len(recorded) != 1 || !logged(realm, ala, MasterMark) {
		t.Errorf("recorded %d lines", len(recorded))
	}
}

func TestSceneWithoutMasterIsSilent(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "e")
	win(t, realm, hero)

	if logged(realm, hero, MasterMark) {
		t.Error("somebody narrated a fight in a world without a master")
	}

	if (Decision{Outcome: OutcomeScene}).Fallback() != "" {
		t.Error("a round should have no canned narration")
	}
}

// partyRealm is the test world with three companions for hire.
func partyRealm(t *testing.T) *Realm {
	t.Helper()

	realm := testRealm(t)
	realm.world.Companions = []world.CompanionSpec{
		{Name: "Bogna Topór", Description: "Bije mocno.", Role: world.RoleFighter, Tier: 1},
		{Name: "Brat Jaromir", Description: "Modli się nad rannymi.", Role: world.RoleHealer, Tier: 2},
		{Name: "Wiesza", Description: "Rzuca iskry.", Role: world.RoleMage, Tier: 1},
	}

	return realm
}

func TestHiringCompanions(t *testing.T) {
	realm := partyRealm(t)
	ala, bob := enlist(realm, "Ala"), enlist(realm, "Bob")
	realm.players[ala].gold = 500

	// Companions are spread across the settlements: the first and third
	// wait here, the second in the outpost.
	if !logged(realm, ala, "W karczmie czeka do wynajęcia: Bogna Topór (wojownik, 60 kapsli), Wiesza (mag, 70 kapsli).") {
		t.Fatalf("inn line missing: %v", realm.View(ala).Log)
	}

	do(realm, ala, "najmij jaromira")
	if len(realm.View(ala).Me.Party) != 0 || !strings.Contains(lastLog(realm, ala), "W karczmie czeka") {
		t.Errorf("hired someone from another settlement: %q", lastLog(realm, ala))
	}

	do(realm, ala, "najmij bognę")
	flatten(realm)

	me := realm.View(ala).Me
	if len(me.Party) != 1 || me.Party[0].Name != "Bogna Topór" || me.Party[0].HP != 30 || me.Gold != 440 {
		t.Fatalf("after hiring: %+v, gold %d", me.Party, me.Gold)
	}

	if !logged(realm, bob, "Ala najmuje towarzysza: Bogna Topór.") {
		t.Error("the others were not told")
	}

	do(realm, bob, "najmij bognę")
	if len(realm.View(bob).Me.Party) != 0 {
		t.Error("the same companion was hired twice")
	}

	realm.players[bob].gold = 10
	do(realm, bob, "najmij wieszę")
	if !strings.Contains(lastLog(realm, bob), "Wiesza chce 70 kapsli, a masz 10.") {
		t.Errorf("hired without gold: %q", lastLog(realm, bob))
	}

	do(realm, ala, "zwerbuj wieszę")
	do(realm, ala, "s", "s", "najmij jaromira")

	if party := realm.View(ala).Me.Party; len(party) != 2 || !strings.Contains(lastLog(realm, ala), "Drużyna jest pełna") {
		t.Errorf("party = %+v, last %q", party, lastLog(realm, ala))
	}

	do(realm, ala, "odpraw bognę", "najmij jaromira")

	party := realm.View(ala).Me.Party
	if len(party) != 2 || party[0].Name != "Wiesza" || party[1].Name != "Brat Jaromir" {
		t.Errorf("party after dismissing and hiring: %+v", party)
	}

	do(realm, ala, "n", "n")
	do(realm, ala, "rozejrzyj się")
	if !logged(realm, ala, "Twoja drużyna: Wiesza, Brat Jaromir.") || !strings.Contains(strings.Join(realm.View(ala).Log[len(realm.View(ala).Log)-6:], " "), "Bogna Topór (wojownik, 60 kapsli)") {
		t.Errorf("the dismissed companion did not return to her inn: %v", realm.View(ala).Log)
	}

	do(realm, ala, "e", "najmij bognę")
	if lastLog(realm, ala) != "Najpierw walka albo ucieczka." {
		t.Errorf("hiring in a fight: %q", lastLog(realm, ala))
	}
}

func TestCompanionsFightAlongside(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	realm.players[hero].autoFight = true
	do(realm, hero, "najmij bognę", "najmij wieszę")
	flatten(realm)
	do(realm, hero, "w")

	// Rats (pace 8) go first; the hero and the mage are both 6, and the
	// hero comes first among equals; the fighter (5) is last.
	rats := realm.monsters[realm.world.Index(westTile)]
	rats[0].hp, rats[1].hp = 100, 100
	rats[0].attack, rats[1].attack = 1, 1

	if order := realm.View(hero).Fight.Order; len(order) != 5 || order[2].Name != "Ala" || order[3].Name != "Wiesza" || order[4].Name != "Bogna Topór" {
		t.Fatalf("order = %+v", order)
	}

	do(realm, hero, "atakuj")

	// The mage blasted both rats (6 each), the hero and the fighter hit the first.
	first, second := 100-rats[0].hp, 100-rats[1].hp
	if second != 6 || first < 6+3+5 {
		t.Errorf("damage after one round: %d and %d", first, second)
	}

	if !logged(realm, hero, "Bogna Topór atakuje:") || !logged(realm, hero, "Wiesza rzuca: Podmuch.") {
		t.Errorf("companions silent: %v", realm.View(hero).Log)
	}

	fight := realm.View(hero).Fight
	if len(fight.Members) != 3 || fight.Members[1].Name != "Bogna Topór" || fight.Round != 2 {
		t.Errorf("members = %+v, round %d", fight.Members, fight.Round)
	}

	// Monsters may strike companions; a companion at zero is out, not dead.
	bogna := realm.players[hero].party[0]
	bogna.hp = 1
	rats[0].attack, rats[1].attack = 50, 50

	for range 20 {
		if bogna.hp <= 0 || realm.View(hero).Fight == nil {
			break
		}

		realm.players[hero].hp = BaseHP
		do(realm, hero, "atakuj")
	}

	if bogna.hp > 0 && realm.View(hero).Fight != nil {
		t.Skip("the dice never aimed at the companion")
	}

	if len(realm.players[hero].party) != 2 {
		t.Error("a knocked-out companion left the party")
	}

	realm.players[hero].pos = home
	realm.players[hero].prev = home
	delete(realm.fights, westTile)
	do(realm, hero, "odpocznij")

	if bogna.hp != bogna.maxHealth() {
		t.Errorf("rest did not revive the companion: %d", bogna.hp)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	if party := restored.players[hero].party; len(party) != 2 || party[0].kind != 0 || restored.hired[2] != hero {
		t.Errorf("party lost in the save: %+v", party)
	}
}

func TestHealerMendsTheWounded(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	realm.players[hero].autoFight = true
	do(realm, hero, "s", "s", "najmij jaromira")
	flatten(realm)

	realm.players[hero].pos, realm.players[hero].prev = home, home
	do(realm, hero, "e")
	realm.players[hero].hp = 5
	do(realm, hero, "atakuj")

	// The healer's inborn ability of tier 2 restores 25; the rat then hits
	// once, for ten at most.
	if hp := realm.players[hero].hp; hp < 5+25-10 {
		t.Errorf("the healer did not mend the hero: %d", hp)
	}

	if !logged(realm, hero, "Brat Jaromir rzuca: Opatrunek. Drużyna odzyskuje do 25 zdrowia.") {
		t.Errorf("no healing in the log: %v", realm.View(hero).Log)
	}
}

func TestCompanionsAreSteeredByTheirLeader(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	give(realm, hero, "Kij", "Napar", "Zwój Iskry")
	do(realm, hero, "najmij wieszę")
	flatten(realm)
	do(realm, hero, "daj kij wieszy", "daj napar wieszy", "daj zwój wieszy")

	ally := realm.players[hero].party[0]
	if ally.weapon == nil || ally.weapon.Name != "Kij" || len(ally.bag) != 1 || len(ally.spells) != 1 || len(realm.players[hero].bag) != 0 {
		t.Fatalf("after giving: weapon %v, bag %d, spells %d", ally.weapon, len(ally.bag), len(ally.spells))
	}

	if view := realm.View(hero).Me.Party[0]; view.Weapon != "Kij" || view.Attack != 4+3 || len(view.Spells) != 2 || view.Spells[0].Name != "Podmuch" {
		t.Errorf("companion view = %+v", view)
	}

	do(realm, hero, "odbierz kij od wieszy")
	if ally.weapon != nil || len(realm.players[hero].bag) != 1 {
		t.Errorf("taking back failed: %v / %d", ally.weapon, len(realm.players[hero].bag))
	}

	do(realm, hero, "daj kij wieszy", "w")

	rats := realm.monsters[realm.world.Index(westTile)]
	rats[0].hp, rats[1].hp = 100, 100
	rats[0].attack, rats[1].attack = 1, 1

	// The hero decides first, then the fight waits for a decision for the mage.
	if actor := realm.View(hero).Fight.Actor; actor == nil || actor.Companion != -1 {
		t.Fatalf("actor = %+v", actor)
	}

	realm.Attack(hero, 0)

	view := realm.View(hero)
	if view.Fight.Actor == nil || view.Fight.Actor.Name != "Wiesza" || view.Fight.Actor.Companion != 0 || len(view.Fight.Actor.Powers) != 2 {
		t.Fatalf("actor = %+v", view.Fight.Actor)
	}

	realm.Cast(hero, 1, 1) // the learnt bolt at the second rat
	if rats[1].hp > 90 || ally.mana != ally.maxEnergy()-4 {
		t.Errorf("companion's bolt: rat at %d, mana %d", rats[1].hp, ally.mana)
	}

	// Round two: the hero again, then the mage, whom the hero lets decide.
	if actor := realm.View(hero).Fight.Actor; actor == nil || actor.Companion != -1 || realm.View(hero).Fight.Round != 2 {
		t.Fatalf("round two should open with the hero: %+v", actor)
	}

	realm.Attack(hero, 0)
	ally.hp = 1
	realm.Auto(hero)

	if len(ally.bag) != 0 || ally.hp <= 1 {
		t.Errorf("auto did not drink the potion: hp %d, bag %d", ally.hp, len(ally.bag))
	}

	do(realm, hero, "autowalka")
	realm.Attack(hero, 0)

	if actor := realm.View(hero).Fight.Actor; actor != nil && actor.Companion != -1 {
		t.Errorf("with autowalka the companion should act alone: %+v", actor)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	if saved := restored.players[hero].party[0]; saved.weapon == nil || len(saved.spells) != 1 || !restored.players[hero].autoFight {
		t.Errorf("companion gear lost in the save: %+v", saved)
	}
}

func TestCallingsShapeHeroes(t *testing.T) {
	realm := testRealm(t)
	warrior, wizard := realm.Join("Ala", "wojownik"), realm.Join("Bob", "czarodziej")

	a, b := realm.View(warrior).Me, realm.View(wizard).Me
	if a.Abilities[Strength] != 15 || a.Attack != 5 || a.MaxHP != 42 || a.MaxMana != 10 || a.Speed != 7 {
		t.Errorf("warrior = %+v", a)
	}

	if b.Abilities[Intelligence] != 15 || b.Attack != 2 || b.MaxMana != 16 || b.MaxHP != 41 {
		t.Errorf("wizard = %+v", b)
	}

	if rolled := realm.View(realm.Join("Cy", "moj-wlasny")).Me.Abilities; !rolled.valid() {
		t.Errorf("a custom portrait should get rolled scores: %v", rolled)
	}

	if Modifier(10) != 0 || Modifier(11) != 0 || Modifier(15) != 2 || Modifier(8) != -1 || Modifier(3) != -4 {
		t.Error("modifiers are not those of a character sheet")
	}

	// Charm sways keepers: the wizard's goodwill is worth a lower floor.
	realm.players[wizard].abilities[Charisma] = 18
	do(realm, wizard, "idź do zbrojowni")

	session := realm.players[wizard].talks[0].session
	if session.goodwill != 4 || session.floor(session.Merchant.Items[0]) != 24 {
		t.Errorf("goodwill %d, floor %d", session.goodwill, session.floor(session.Merchant.Items[0]))
	}
}

func TestLegendsTakeWagesByTheDay(t *testing.T) {
	realm := partyRealm(t)
	realm.world.Companions = append(realm.world.Companions, world.CompanionSpec{Name: "Sir Gniewomir", Description: "Legenda.", Role: world.RoleFighter, Tier: 3, Legend: true})
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 100

	// The legend waits in the second settlement (fourth of four), for no
	// price up front but a wage of 65 a day.
	do(realm, hero, "s", "s")
	if !logged(realm, hero, "Sir Gniewomir (legenda, żołd 65 kapsli dziennie)") {
		t.Fatalf("no legend at the inn: %v", realm.View(hero).Log)
	}

	// Legends follow only the respected: a point of standing in the
	// settlement opens the door.
	realm.players[hero].standing = map[int]int{1: 1}
	do(realm, hero, "najmij gniewomira")
	flatten(realm)

	me := realm.View(hero).Me
	if len(me.Party) != 1 || !me.Party[0].Legend || me.Party[0].Wage != 65 || me.Gold != 100 || me.Party[0].MaxHP != 120 || me.Party[0].Attack != 15 {
		t.Fatalf("after hiring the legend: %+v, gold %d", me.Party, me.Gold)
	}

	if powers := me.Party[0].Spells; len(powers) != 2 || powers[0].Name != "Podmuch" || powers[1].Name != "Opatrunek" {
		t.Errorf("a legend has both inborn powers: %+v", powers)
	}

	// Twelve steps make a day; the wage is taken at daybreak.
	for step := 0; step < 12; step++ {
		realm.players[hero].pos = home
		do(realm, hero, "n")
	}

	me = realm.View(hero).Me
	if me.Day != 2 || me.Gold != 35 || !logged(realm, hero, "Sir Gniewomir bierze żołd: 65 kapsli. Zostaje ci 35.") {
		t.Errorf("after a day: day %d, gold %d", me.Day, me.Gold)
	}

	// Rest ends the day too, and this time the purse is too light.
	realm.players[hero].pos = home
	do(realm, hero, "odpocznij")

	me = realm.View(hero).Me
	if me.Day != 3 || len(me.Party) != 0 || me.Gold != 35 || !logged(realm, hero, "Sir Gniewomir nie dostaje żołdu i odchodzi") {
		t.Errorf("unpaid legend: day %d, party %+v, gold %d", me.Day, me.Party, me.Gold)
	}

	if realm.hired[3] != "" {
		t.Error("the legend did not go back to the inn")
	}

	// Dismissing a legend stops the wage.
	realm.players[hero].gold = 200
	do(realm, hero, "s", "s", "najmij gniewomira", "odpraw gniewomira")

	if !logged(realm, hero, "Żołd przestaje biec.") || len(realm.View(hero).Me.Party) != 0 {
		t.Errorf("dismissing the legend: %v", realm.View(hero).Log[len(realm.View(hero).Log)-3:])
	}
}

func TestSellingMovesGoldAndGoods(t *testing.T) {
	realm := testRealm(t)
	name := enlist(realm, "Kupiec")
	give(realm, name, "Kij", "Napar")
	do(realm, name, "idź do zbrojowni")

	request, ok := realm.HaggleRequest(name, "sprzedam kij za 20")
	if !ok || len(request.Owned) != 2 {
		t.Fatalf("request = %+v, ok %v", request, ok)
	}

	intent := request.Sanitize(Read(request).Alone())
	decision, _, _ := realm.Haggle(name, request.Text, intent)
	if decision.Outcome != OutcomeBought || decision.Price != 20 {
		t.Fatalf("decision = %+v", decision)
	}

	me := realm.View(name).Me
	if me.Gold != startGold+20 || len(me.Bag) != 1 || me.Bag[0].Name != "Napar" {
		t.Errorf("after the sale: gold %d, bag %v", me.Gold, itemNames(me.Bag))
	}

	// The smith has no use for a potion.
	request, _ = realm.HaggleRequest(name, "kupisz napar?")
	if decision, _, _ = realm.Haggle(name, request.Text, request.Sanitize(Read(request).Alone())); decision.Outcome != OutcomeUnwanted {
		t.Errorf("potion = %+v", decision)
	}
}

// shapedRealm is the test world with one monster of the given shape and
// tier waiting east of the settlement.
func shapedRealm(t *testing.T, shape world.Shape, tier int) (*Realm, string, *monster) {
	t.Helper()

	realm := testRealm(t)
	realm.world.Monsters = append(realm.world.Monsters, world.MonsterSpec{Name: "Okaz", Shape: shape, Tier: tier})
	foe := realm.newMonster(len(realm.world.Monsters) - 1)
	realm.monsters[realm.world.Index(eastTile)] = []*monster{foe}

	hero := enlist(realm, "Ala")

	return realm, hero, foe
}

func TestSerpentsPoisonAndPotionsCure(t *testing.T) {
	realm, hero, snake := shapedRealm(t, "serpent", 2)
	give(realm, hero, "Napar")
	snake.attack = 1
	do(realm, hero, "e")

	// The snake is quicker than the hero and bit first: the venom is in.
	me := realm.players[hero]
	if me.poison != poisonRounds || !logged(realm, hero, "Jad działa!") {
		t.Fatalf("poison %d after the bite; log %q", me.poison, lastLog(realm, hero))
	}

	before := me.hp
	realm.Attack(hero, 0)

	// Round two opened: the venom took its points before the snake's next bite.
	if !logged(realm, hero, fmt.Sprintf("Jad działa: Ala traci %d zdrowia.", poisonDamage)) || me.hp >= before-poisonDamage {
		t.Errorf("after a round: poison %d, hp %d (was %d)", me.poison, me.hp, before)
	}

	// The potion cures; the snake's next bite, which ends the turn, poisons anew.
	realm.Use(hero, 0)
	if !logged(realm, hero, "Jad przestaje działać.") || len(me.bag) != 0 {
		t.Errorf("the potion did not cure: bag %d", len(me.bag))
	}

	healed := false
	for _, blow := range realm.View(hero).Blows {
		healed = healed || blow.Kind == BlowHeal
	}

	if view := realm.View(hero); !healed || view.Me.Poison != poisonRounds {
		t.Errorf("view = poison %d, blows %+v", view.Me.Poison, view.Blows)
	}
}

func TestUndeadKnitAndSpiritsShrugOffSteel(t *testing.T) {
	realm, hero, ghoul := shapedRealm(t, "undead", 2)
	ghoul.attack = 1
	do(realm, hero, "e")

	ghoul.hp = 10
	realm.Attack(hero, 0)

	// The blow landed, then the new round healed the ghoul by twice its tier.
	struck := 0
	for _, blow := range realm.View(hero).Blows {
		if blow.Foe == ghoul.id && blow.Kind == BlowHit {
			struck = -blow.Amount
		}
	}

	if struck == 0 || ghoul.hp != 10-struck+4 || !logged(realm, hero, "Okaz zrasta się: odzyskuje 4 zdrowia.") {
		t.Errorf("ghoul at %d after a blow of %d", ghoul.hp, struck)
	}

	realm, hero, ghost := shapedRealm(t, "spirit", 1)
	ghost.attack = 1
	do(realm, hero, "e")

	// Steel does half: a hero with might 3 hits a ghost for 1 or 2.
	realm.Attack(hero, 0)
	if lost := ghost.maxHP - ghost.hp; lost < 1 || lost > 2 || !logged(realm, hero, "(widmo)") {
		t.Errorf("the ghost lost %d to a blow of 3-4", lost)
	}
}

func TestBestiaryFillsWithMeetingsAndKills(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	pages := realm.View(hero).Me.Bestiary
	if len(pages) != 2 || pages[0].Known || pages[1].Kind != -1 || !pages[1].Boss || pages[1].Trait.Name != "Furia i Szał" {
		t.Fatalf("pages before any fight: %+v", pages)
	}

	do(realm, hero, "e")
	if pages = realm.View(hero).Me.Bestiary; !pages[0].Known || pages[0].Slain != 0 || pages[0].Name != "Szczur Polny" || pages[0].Trait.Name != "Szał" {
		t.Errorf("page after meeting: %+v", pages[0])
	}

	win(t, realm, hero)
	if pages = realm.View(hero).Me.Bestiary; pages[0].Slain != 1 {
		t.Errorf("page after the kill: %+v", pages[0])
	}

	restored, err := Restore(realm.State(hero), 3)
	if err != nil {
		t.Fatal(err)
	}

	restored.Join("Ala", "")
	if pages = restored.View(hero).Me.Bestiary; !pages[0].Known || pages[0].Slain != 1 || pages[1].Known {
		t.Errorf("pages after loading: %+v", pages)
	}
}

func TestOrdersForCompanions(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500
	do(realm, hero, "s", "s", "najmij jaromira")
	flatten(realm)

	me := realm.players[hero]
	monk := me.party[0]
	me.pos, me.prev, me.hp = home, home, 10

	// Outside a fight the healer mends the party at once; the full name
	// and punctuation of spoken text do no harm.
	if !realm.Order(hero, "Brat Jaromir, ulecz siebie i załogę.") || me.hp <= 10 || !logged(realm, hero, "Brat Jaromir rzuca: Opatrunek. Rany się zamykają") {
		t.Fatalf("heal outside: hp %d, log %q", me.hp, lastLog(realm, hero))
	}

	if realm.Order(hero, "Zenobiusz, ulecz") || realm.Order(hero, "jaromir") {
		t.Error("an order for nobody, or without a verb, was taken")
	}

	realm.Order(hero, "jaromir tańcz")
	if !logged(realm, hero, "Brat Jaromir: tego nie zrobi.") {
		t.Errorf("unknown order: %q", lastLog(realm, hero))
	}

	// The hero knows no healing; told so, with the hint.
	do(realm, hero, "ulecz")
	if !logged(realm, hero, "Nie znasz żadnej mocy leczącej.") {
		t.Errorf("hero heal: %q", lastLog(realm, hero))
	}

	// In a fight the order waits for the healer's turn.
	do(realm, hero, "w")
	for _, rat := range realm.monsters[realm.world.Index(westTile)] {
		rat.attack, rat.hp = 1, 100
	}

	if actor := realm.View(hero).Fight.Actor; actor == nil || actor.Companion != -1 {
		t.Fatalf("actor = %+v", actor)
	}

	me.hp = 10
	realm.Order(hero, "Jaromirze, ulecz załogę")
	if monk.order == nil || !logged(realm, hero, "Brat Jaromir wykona to w swojej turze.") || me.hp != 10 {
		t.Fatalf("order not kept: %v, hp %d", monk.order, me.hp)
	}

	realm.Attack(hero, 0)

	view := realm.View(hero)
	if monk.order != nil || me.hp <= 10 || !logged(realm, hero, "Brat Jaromir rzuca: Opatrunek. Drużyna odzyskuje") || view.Fight.Round != 2 {
		t.Errorf("order in the fight: order %v, hp %d, round %d, log %q", monk.order, me.hp, view.Fight.Round, lastLog(realm, hero))
	}

	// On his own turn an order is carried out at once.
	realm.Attack(hero, 0)
	for realm.View(hero).Fight != nil && realm.View(hero).Fight.Actor != nil && realm.View(hero).Fight.Actor.Companion != 0 {
		realm.Attack(hero, 0)
	}

	before := realm.monsters[realm.world.Index(westTile)][0].hp
	realm.Order(hero, "Jaromir, atakuj szczura")
	if after := realm.monsters[realm.world.Index(westTile)][0].hp; after >= before || !logged(realm, hero, "Brat Jaromir atakuje:") {
		t.Errorf("attack order: rat %d -> %d, log %q", before, after, lastLog(realm, hero))
	}
}

// adventureRealm builds the shipped adventure.
func adventureRealm(t *testing.T) (*Realm, string) {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	for _, spec := range presets {
		if spec.Mode != world.ModeAdventure {
			continue
		}

		built, err := world.Build(spec)
		if err != nil {
			t.Fatal(err)
		}

		realm := NewRealm(built, 1)

		return realm, realm.Join("Ewa", "lowczyni")
	}

	t.Fatal("no adventure among the presets")

	return nil, ""
}

func TestHipokampCanBeFinished(t *testing.T) {
	realm, hero := adventureRealm(t)

	view := realm.View(hero)
	if !view.Adventure() || view.Place.Name != "Sala Pooperacyjna" || len(view.Me.Bestiary) != 0 || !logged(realm, hero, "Cel: Wydostać się") {
		t.Fatalf("start: %+v", view.Place)
	}

	// Shut doors and the void stop the player; fixed things stay put.
	do(realm, hero, "e")
	if realm.View(hero).Place.Name != "Sala Pooperacyjna" || !logged(realm, hero, "Czytnik mruga czerwono") {
		t.Fatalf("walked through a shut door: %q", lastLog(realm, hero))
	}

	do(realm, hero, "s")
	if lastLog(realm, hero) != "Dalej nie ma przejścia." {
		t.Errorf("void: %q", lastLog(realm, hero))
	}

	do(realm, hero, "n")
	if !logged(realm, hero, "Ściana. Zza niej dobiega czyjeś chrapanie.") {
		t.Errorf("wall: %q", lastLog(realm, hero))
	}

	do(realm, hero, "podnieś łóżko")
	if lastLog(realm, hero) != "Łóżko: tego nie da się zabrać." {
		t.Errorf("fixed thing: %q", lastLog(realm, hero))
	}

	// The chain of the story.
	do(realm, hero, "zbadaj łóżko", "podnieś żeton", "otwórz szafkę", "podnieś szlafrok", "podnieś kartę", "zbadaj kartę pacjenta")
	me := realm.View(hero).Me
	if len(me.Bag) != 3 || !realm.story.flags["kod"] {
		t.Fatalf("after the ward: bag %v, flags %v", itemNames(me.Bag), realm.story.flags)
	}

	do(realm, hero, "w", "zbadaj lustro", "n")
	if realm.View(hero).Place.Name != "Łazienka" {
		t.Fatalf("the grate let the player through")
	}

	do(realm, hero, "otwórz kratkę", "n", "e")
	if realm.View(hero).Place.Name != "Dyżurka" || !logged(realm, hero, "Jest tu: Pielęgniarz Grabski.") {
		t.Fatalf("not in the duty room: %s", realm.View(hero).Place.Name)
	}

	do(realm, hero, "rozmawiaj z grabskim")
	if !logged(realm, hero, "Przynieś mi kawę") {
		t.Errorf("talk: %q", lastLog(realm, hero))
	}

	do(realm, hero, "daj kawę grabskiemu")
	if lastLog(realm, hero) != "Nie masz niczego takiego." {
		t.Errorf("giving what one has not: %q", lastLog(realm, hero))
	}

	do(realm, hero, "zbadaj biurko", "podnieś śrubokręt", "e", "użyj żetonu na automat")
	if me = realm.View(hero).Me; find(me.Bag, "kawa") < 0 || find(me.Bag, "zeton") >= 0 {
		t.Fatalf("coffee: %v", itemNames(me.Bag))
	}

	do(realm, hero, "w", "daj kawę grabskiemu", "rozmawiaj z grabskim")
	if me = realm.View(hero).Me; find(me.Bag, "karta dostepu") < 0 || !logged(realm, hero, "Siedem… trzy… jeden") {
		t.Fatalf("after the coffee: %v, %q", itemNames(me.Bag), lastLog(realm, hero))
	}

	do(realm, hero, "e", "s", "użyj karty dostępu na drzwi", "e", "użyj bezpiecznika na skrzynkę")
	if realm.View(hero).Place.Name != "Rozdzielnia" || lastLog(realm, hero) != "Nie masz niczego takiego." {
		t.Fatalf("switch room: %s, %q", realm.View(hero).Place.Name, lastLog(realm, hero))
	}

	do(realm, hero, "użyj śrubokręta na skrzynkę", "n", "zbadaj półki", "podnieś bezpiecznik", "s", "użyj bezpiecznika na skrzynkę")
	if !realm.story.flags["winda"] || !logged(realm, hero, "winda serwisowa ma zasilanie") {
		t.Fatalf("power: %v, %q", realm.story.flags, lastLog(realm, hero))
	}

	// The corridor changed; the save keeps the story.
	do(realm, hero, "w", "n")
	if view = realm.View(hero); view.Place.Name != "Korytarz Północny" || !strings.Contains(view.Place.Description, "jasno oświetlony") {
		t.Fatalf("changed room: %+v", view.Place)
	}

	restored, err := Restore(realm.State(hero), 2)
	if err != nil {
		t.Fatal(err)
	}

	restored.Join("Ewa", "")
	if again := restored.View(hero); !strings.Contains(again.Place.Description, "jasno oświetlony") || !restored.story.flags["winda"] || len(restored.story.done) != len(realm.story.done) {
		t.Errorf("the save lost the story: %+v", again.Place)
	}

	do(realm, hero, "n", "użyj panelu", "n", "rozmawiaj z wernicke", "użyj transportera")
	if view = realm.View(hero); !view.Won || view.Place.Name != "Lądowisko" || !logged(realm, hero, "Przygoda «Hipokamp» ukończona!") {
		t.Errorf("end: won %v at %s, %q", view.Won, view.Place.Name, lastLog(realm, hero))
	}
}

// dndRealm is the test world played by the tabletop rules.
func dndRealm(t *testing.T) (*Realm, string) {
	t.Helper()

	realm := testRealm(t)
	realm.world.Mode = world.ModeDnD

	return realm, enlist(realm, "Ala")
}

func TestDnDFightsAreOnTheBoard(t *testing.T) {
	realm, hero := dndRealm(t)
	me := realm.players[hero]
	me.hp = 999
	do(realm, hero, "e")

	current := realm.fightOf(me)
	if current == nil || current.board == nil {
		t.Fatal("no board")
	}

	rat := realm.foes(eastTile)[0]
	rat.attack = 1

	view := realm.View(hero)
	if view.Fight.Board == nil || len(view.Fight.Board.Pieces) != 2 || view.Me.AC != 10 || view.Fight.Monsters[0].AC != 12 {
		t.Fatalf("board view = %+v, AC %d, rat AC %d", view.Fight.Board, view.Me.AC, view.Fight.Monsters[0].AC)
	}

	// The hero stands left; the rat was seated right and may have run in.
	if at, foe := current.board.cells[unit(me)], current.board.cells[rat]; at.X != 1 || foe.X <= at.X {
		t.Fatalf("seats: hero %v, rat %v", at, foe)
	}

	// Wait for the hero's turn: the rat may be quicker and closes in first.
	for range 10 {
		if realm.View(hero).Fight.Actor != nil {
			break
		}

		realm.Attack(hero, 0)
	}

	if !realm.adjacent(current, unit(me), rat) {
		realm.Attack(hero, 0)
		if !logged(realm, hero, "stoi za daleko od: Szczur Polny") {
			t.Fatalf("attacking from afar: %q", lastLog(realm, hero))
		}

		// Walking: six cells a turn, never into the void.
		if current.board.moves != heroCells {
			t.Fatalf("moves = %d", current.board.moves)
		}

		for range heroCells {
			realm.Step(hero, 1, 0)
		}

		// Either the movement ran out or the rat stood in the way.
		realm.Step(hero, 1, 0)
		if !logged(realm, hero, "nie ma już ruchu w tej turze") && !logged(realm, hero, "To pole jest zajęte.") {
			t.Errorf("walked beyond the movement: %q", lastLog(realm, hero))
		}
	}

	// Strikes are d20 rolls: over many turns both hits and misses show up.
	hits, misses := 0, 0
	for range 60 {
		if realm.View(hero).Fight == nil {
			break
		}

		rat.hp = 100
		before := len(realm.View(hero).Log)
		realm.Attack(hero, 0)

		for _, line := range realm.View(hero).Log[before:] {
			switch {
			case strings.Contains(line, "Ala atakuje: Szczur Polny. Pudło (k20:"):
				misses++
			case strings.HasPrefix(line, "Ala atakuje (k20:"):
				hits++
			}
		}
	}

	if hits == 0 || misses == 0 {
		t.Errorf("hits %d, misses %d", hits, misses)
	}
}

func TestDnDFleeingProvokesAndSavesSoftenVenom(t *testing.T) {
	realm, hero, snake := shapedRealm(t, "serpent", 2)
	realm.world.Mode = world.ModeDnD
	me := realm.players[hero]
	me.hp, me.abilities[Constitution] = 999, 30
	snake.attack = 1
	do(realm, hero, "e")

	current := realm.fightOf(me)
	if current == nil {
		t.Fatal("no fight")
	}

	// Side by side; with a tough Constitution the venom is often resisted.
	current.board.cells[unit(me)] = world.Point{X: 5, Y: 2}
	current.board.cells[snake] = world.Point{X: 6, Y: 2}

	for range 40 {
		if realm.View(hero).Fight == nil || realm.View(hero).Fight.Actor != nil {
			break
		}
	}

	bitten := false
	for range 30 {
		if realm.View(hero).Fight == nil {
			break
		}

		if realm.View(hero).Fight.Actor == nil {
			continue
		}

		snake.hp = 100
		realm.Attack(hero, 0)
		bitten = bitten || logged(realm, hero, "Rzut obronny udany: jad nie działa.")
	}

	if !bitten {
		t.Error("no saving throw against the venom was ever made")
	}

	// Running from reach draws an attack of opportunity.
	for realm.View(hero).Fight != nil && realm.View(hero).Fight.Actor == nil {
		realm.Attack(hero, 0)
	}

	if realm.View(hero).Fight != nil {
		realm.Flee(hero)
		if !logged(realm, hero, "wyprowadza atak okazyjny na uciekającego") {
			t.Errorf("no opportunity attack: %q", lastLog(realm, hero))
		}
	}
}

func TestKnowledgeIsSharedSpreadAndLearnt(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	// Everybody knows the boss; the specialists know their monsters.
	if len(realm.knownBy(0)) == 0 || len(realm.knownBy(1)) == 0 {
		t.Fatal("keepers know nothing")
	}

	for _, fact := range realm.wisdom.facts {
		if fact.Subject == "Szef" && (!fact.known[0] || !fact.known[1]) {
			t.Error("the boss is not common knowledge")
		}
	}

	// A keeper's new line joins the graph and the hero's journal.
	do(realm, hero, "idź do zbrojowni")
	if !realm.Learn(hero, FactSpec{Subject: "Młyn nad rzeką", Text: "Nocą świeci się tam zielone światło."}) {
		t.Fatal("the fact was not learnt")
	}

	if realm.Learn(hero, FactSpec{Subject: "Młyn nad rzeką", Text: "Nocą świeci się tam zielone światło."}) {
		t.Error("the same fact was learnt twice")
	}

	journal := realm.Journal(hero)
	if len(journal) != 1 || journal[0].Subject != "Młyn nad rzeką" || journal[0].Source != "Kowal" {
		t.Fatalf("journal = %+v", journal)
	}

	// The herbalist has not heard it; after enough days she has.
	mill := realm.wisdom.facts[len(realm.wisdom.facts)-1]
	if mill.known[1] {
		t.Fatal("the herbalist knew at once")
	}

	for day := 2; day < 40 && !mill.known[1]; day++ {
		realm.worldDay(day)
	}

	if !mill.known[1] {
		t.Error("the rumour never spread")
	}

	// A keeper's lore says what she does not know and whom to ask.
	_, request, _ := realm.Haggle(hero, "co wiesz o źródle?", Intent{Action: ActionChat})
	if !strings.Contains(request.Lore, "Wiesz:") {
		t.Errorf("lore without facts: %s", request.Lore)
	}

	if request.Hint == "" {
		t.Error("no hint for a chat")
	}
}

func TestEventsChangeTheWorld(t *testing.T) {
	realm := testRealm(t)
	realm.world.Events = []world.EventSpec{
		{When: world.WhenSpec{Enter: "Las"}, Once: true, Text: "Las płonie!", Global: true, Flag: "pozar",
			Transform: &world.RoomChange{Room: "Las", Description: "Zgliszcza.", Terrain: "wasteland"},
			Fact:      &world.FactSpec{Subject: "Pożar lasu", Text: "Las spłonął."}},
		{When: world.WhenSpec{Killed: "Szczur Polny", Count: 1}, Once: true, Text: "Z nor wychodzi więcej szczurów.",
			Spawn: &world.SpawnSpec{Monster: "Szczur Polny", Count: 2, Where: "near:Osada"}, Gold: 10},
		{When: world.WhenSpec{Day: 2, Flag: "pozar"}, Once: true, Text: "Ceny rosną po pożarze.", Prices: 30},
		{When: world.WhenSpec{Talked: "Kowal", Has: "Kij"}, Once: true, Text: "Kowal patrzy na kij.", Gives: "Kaftan"},
	}

	hero := enlist(realm, "Ala")
	bob := enlist(realm, "Bob")
	me := realm.players[hero]
	me.hp = 999

	// Entering the forest burns it for everyone, on the map and in the scene.
	do(realm, hero, "s")
	if !logged(realm, hero, "Las płonie!") || !logged(realm, bob, "Las płonie!") || realm.View(hero).Place.Terrain != "wasteland" || !strings.Contains(realm.View(hero).Place.Description, "Zgliszcza") {
		t.Fatalf("the forest did not burn: %+v, %q", realm.View(hero).Place, lastLog(realm, hero))
	}

	if facts := realm.Journal(hero); len(facts) == 0 || facts[len(facts)-1].Subject != "Pożar lasu" {
		t.Errorf("the fire is not in the journal: %+v", facts)
	}

	// The second visit changes nothing more.
	do(realm, hero, "n", "s")
	if n := strings.Count(strings.Join(realm.View(hero).Log, "\n"), "Las płonie!"); n != 1 {
		t.Errorf("the fire happened %d times", n)
	}

	// A kill spawns rats next to the settlement and pays.
	gold := me.gold
	do(realm, hero, "n", "e")
	win(t, realm, hero)

	if !logged(realm, hero, "Z nor wychodzi więcej szczurów.") || me.gold < gold+10 {
		t.Errorf("the kill event: %q, gold %d -> %d", lastLog(realm, hero), gold, me.gold)
	}

	around := 0
	for _, step := range []world.Point{north, east, south, west} {
		around += len(realm.foes(world.Point{X: home.X + step.X, Y: home.Y + step.Y}))
	}

	if around < 2 {
		t.Errorf("only %d monsters around the settlement", around)
	}

	// Day two raises prices once the forest has burnt.
	me.pos, me.prev = home, home
	before := realm.markupOf("item-1")
	for range stepsPerDay {
		realm.hour(me)
	}

	// The day's drift settles a little of the rise back.
	if !logged(realm, hero, "Ceny rosną po pożarze.") || realm.markupOf("item-1") < before+30-driftMarkup {
		t.Errorf("prices: %d -> %d, %q", before, realm.markupOf("item-1"), lastLog(realm, hero))
	}

	// Talking to the smith with a stick in the bag earns a coat.
	give(realm, hero, "Kij")
	do(realm, hero, "idź do zbrojowni")
	realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	if find(me.bag, "kaftan") < 0 || !logged(realm, hero, "Kowal patrzy na kij.") {
		t.Errorf("talk event: bag %v", itemNames(me.bag))
	}
}

func TestThePulseOfTheWorld(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	me := realm.players[hero]

	// Buying raises a price; days bring it back down.
	do(realm, hero, "idź do zbrojowni")
	request, _ := realm.HaggleRequest(hero, "biorę kij")
	realm.Haggle(hero, request.Text, request.Sanitize(Read(request).Alone()))
	if realm.markupOf("item-1") != 100+buyMarkup {
		t.Errorf("markup after a sale: %d", realm.markupOf("item-1"))
	}

	do(realm, hero, "s", "n")
	if price := realm.View(hero).Shop; price == nil {
		do(realm, hero, "idź do zbrojowni")
	}

	// Night falls late in the day and lifts at dawn.
	me.steps = nightFrom
	if !realm.View(hero).Night {
		t.Error("no night")
	}

	for range stepsPerDay - nightFrom {
		realm.hour(me)
	}

	if realm.View(hero).Night || realm.View(hero).Day != 2 {
		t.Errorf("after dawn: night %v, day %d", realm.View(hero).Night, realm.View(hero).Day)
	}

	// Over many days the monsters move or multiply somewhere.
	before := 0
	for index := range realm.monsters {
		before += len(realm.monsters[index])
	}

	for day := 3; day < 30; day++ {
		realm.worldDay(day)
	}

	after := 0
	for index := range realm.monsters {
		after += len(realm.monsters[index])
	}

	if after <= before {
		t.Errorf("the wilds did not fill: %d -> %d monsters", before, after)
	}
}

func TestMercenariesInventNoGoods(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "rozmawiaj z bogną")

	contract := world.ItemSpec{Name: "Umowa Najmu", Description: "Kontrakt.", Kind: world.KindCharm, Effect: world.EffectPower, Tier: 1}
	if _, ok := realm.Invent(hero, contract); ok || realm.known("Umowa Najmu") {
		t.Error("a mercenary's contract became a charm")
	}
}

func TestCompanionsHaggleAtTheInn(t *testing.T) {
	realm := partyRealm(t)
	realm.world.Companions = append(realm.world.Companions, world.CompanionSpec{Name: "Sir Gniewomir", Description: "Legenda.", Role: world.RoleFighter, Tier: 3, Legend: true})
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]
	ala.gold = 200
	price := realm.world.Companions[0].Stats().Price

	// Sitting down with a companion for hire opens a talk over one ware:
	// their service, at their price.
	do(realm, hero, "rozmawiaj z bogną")

	view := realm.View(hero)
	if view.Shop == nil || view.Shop.Keeper != "Bogna Topór" || view.Shop.Avatar != "wojownik" || len(view.Shop.Wares) != 1 || view.Shop.Wares[0].Price != price {
		t.Fatalf("no talk with Bogna: %+v, %v", view.Shop, view.Log)
	}

	if !logged(realm, hero, "Przysiadasz się do: Bogna Topór.") {
		t.Fatalf("log: %v", view.Log)
	}

	// A bare sum is an offer for it, even without a model.
	request, ok := realm.HaggleRequest(hero, "30")
	if reading := Read(request).Alone(); !ok || reading.Action != ActionOffer || reading.Amount != 30 {
		t.Errorf("a bare sum: %+v", reading)
	}

	// The model is told who it plays.
	_, narrate, _ := realm.Haggle(hero, "co umiesz?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Nie prowadzisz sklepu. Jesteś: Bogna Topór, wojownik") || !strings.Contains(narrate.Role, "najemnikiem") || !strings.Contains(narrate.Lore, "Twoja służba kosztuje") {
		t.Errorf("lore for a mercenary: %q / %q", narrate.Lore, narrate.Role)
	}

	// The fee is in the lore, but small talk may not quote it.
	if !slices.Contains(narrate.Forbidden, price) {
		t.Errorf("the fee is allowed in small talk: %v", narrate.Forbidden)
	}

	// A low offer is countered; taking the counter seals the deal.
	low := price * 38 / 100
	decision, _, _ := realm.Haggle(hero, fmt.Sprintf("dam %d", low), Intent{Action: ActionOffer, Amount: low})
	if decision.Outcome != OutcomeCounter || decision.Price >= price {
		t.Fatalf("a low offer: %+v", decision)
	}

	decision, narrate, _ = realm.Haggle(hero, "zgoda", Intent{Action: ActionAccept})
	if decision.Outcome != OutcomeSold || len(ala.party) != 1 || ala.gold != 200-decision.Price || !logged(realm, hero, "Bogna Topór dołącza do drużyny.") {
		t.Fatalf("the deal: %+v, party %d, gold %d, %v", decision, len(ala.party), ala.gold, realm.View(hero).Log)
	}

	if wares := realm.View(hero).Shop.Wares; len(wares) != 0 || !strings.Contains(narrate.Role, "towarzyszem") {
		t.Errorf("after joining: wares %+v, role %q", wares, narrate.Role)
	}

	// A companion in the party talks for free; let go, they are for hire again.
	realm.CloseShop(hero)
	do(realm, hero, "rozmawiaj z bogną")
	if !logged(realm, hero, "Rozmawiasz z: Bogna Topór, twój towarzysz.") || len(realm.View(hero).Shop.Wares) != 0 {
		t.Fatalf("talk with a party member: %v", realm.View(hero).Log)
	}

	realm.CloseShop(hero)
	do(realm, hero, "odpraw bognę", "rozmawiaj z bogną")
	if wares := realm.View(hero).Shop.Wares; len(wares) != 1 || wares[0].Price != price {
		t.Fatalf("the service is not back on the table: %+v", wares)
	}

	// A legend haggles over the daily wage, pays nothing up front and
	// keeps the agreed wage through a save.
	realm.CloseShop(hero)
	do(realm, hero, "s", "s")
	ala.standing = map[int]int{1: 1}
	do(realm, hero, "rozmawiaj z gniewomirem")

	view = realm.View(hero)
	if view.Shop == nil || len(view.Shop.Wares) != 1 || view.Shop.Wares[0].Name != "Żołd dzienny" || view.Shop.Wares[0].Price != 65 {
		t.Fatalf("no talk with the legend: %+v, %v", view.Shop, view.Log)
	}

	gold := ala.gold
	decision, _, _ = realm.Haggle(hero, "dam 55 dziennie", Intent{Action: ActionOffer, Amount: 55})
	if decision.Outcome != OutcomeSold || decision.Price != 55 || ala.gold != gold || len(ala.party) != 1 || ala.party[0].wage != 55 {
		t.Fatalf("the legend's wage: %+v, gold %d, party %d", decision, ala.gold, len(ala.party))
	}

	if me := realm.View(hero).Me; len(me.Party) != 1 || me.Party[0].Wage != 55 || !logged(realm, hero, "Umowa: 55 kapsli za każdy dzień") {
		t.Errorf("the view of the wage: %+v", me.Party)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil || len(restored.players[hero].party) != 1 || restored.players[hero].party[0].wage != 55 {
		t.Errorf("the wage was not saved: %v", err)
	}

	// Nobody talks to the disliked.
	realm.CloseShop(hero)
	ala.standing[1] = -1
	do(realm, hero, "odpraw gniewomira", "rozmawiaj z jaromirem")
	if ala.shop >= 0 || !logged(realm, hero, "Brat Jaromir: «Z kimś, kogo tu mają za") {
		t.Errorf("the disliked got a talk: %v", realm.View(hero).Log)
	}
}

func TestStyleCardReachesTheKeepersAndTheMaster(t *testing.T) {
	realm := testRealm(t)
	realm.world.Style = &world.StyleSpec{Look: "Kosmiczny western lat 90.", Dialogue: "Znużeni, ironiczni. «Jasne, a kto płaci?»"}
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")

	_, request, ok := realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	if !ok {
		t.Fatal("no talk")
	}

	if !strings.Contains(request.Lore, "Styl świata: Kosmiczny western lat 90. Tak tu mówią: Znużeni, ironiczni. «Jasne, a kto płaci?»") {
		t.Errorf("keeper lore misses the style card:\n%s", request.Lore)
	}

	if lore := realm.sceneLore(&fight{pos: realm.players[hero].pos}, []string{hero}); !strings.Contains(lore, "Styl opowieści: Kosmiczny western lat 90.") {
		t.Errorf("master lore misses the style card:\n%s", lore)
	}

	realm.world.Style = nil
	if _, request, _ = realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat}); strings.Contains(request.Lore, "Styl") {
		t.Errorf("a world without a card got a style line:\n%s", request.Lore)
	}
}

func TestShutDoorsHoldFromBothSides(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	index := slices.IndexFunc(presets, func(spec world.Spec) bool { return strings.HasPrefix(spec.Name, "Jeziorna") })
	built, err := world.Build(presets[index])
	if err != nil {
		t.Fatal(err)
	}

	realm := NewRealm(built, 1)
	hero := realm.Join("Ewa", "lowczyni")

	// From the park the boarded stairs keep the player out of Młyńska 3.
	do(realm, hero, "s")
	if realm.View(hero).Place.Name != "Park nad Jeziorem" {
		t.Fatalf("south of the market: %s", realm.View(hero).Place.Name)
	}

	do(realm, hero, "e")
	if realm.View(hero).Place.Name != "Park nad Jeziorem" || !logged(realm, hero, "zabite deskami") {
		t.Fatalf("walked in through a boarded door: %s, %q", realm.View(hero).Place.Name, lastLog(realm, hero))
	}

	// The flat is entered from the inn's side, and left the same way.
	do(realm, hero, "n")
	do(realm, hero, "e")
	do(realm, hero, "s")
	if realm.View(hero).Place.Name != "Młyńska 3" {
		t.Fatalf("via the inn: %s", realm.View(hero).Place.Name)
	}

	do(realm, hero, "w")
	if realm.View(hero).Place.Name != "Młyńska 3" {
		t.Errorf("walked out through the boarded door")
	}

	do(realm, hero, "n")
	if realm.View(hero).Place.Name != "Pod Sumem" {
		t.Errorf("could not leave the flat: %s", realm.View(hero).Place.Name)
	}
}
