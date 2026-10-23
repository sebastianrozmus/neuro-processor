package game

import (
	"slices"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// diverLine is Jagna telling where her daughter wanders.
const diverLine = "[strach] Boję się, że kręci się koło Nurka bez Twarzy, tego w mosiężnym hełmie."

// diverHook is what the Mind reads in it.
var diverHook = QuestHook{Found: true, Person: "Zosia", Relation: "córka", Monster: "Nurek bez Twarzy", Quote: "kręci się koło Nurka bez Twarzy"}

// diverRealm is the test world with a faceless diver in it and a hero
// talking with Jagna, who has told of her daughter Zosia.
func diverRealm(t *testing.T) (*Realm, string) {
	t.Helper()

	realm := testRealm(t)
	realm.world.Monsters = append(realm.world.Monsters, world.MonsterSpec{Name: "Nurek bez Twarzy", Tier: 1})
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zielarki")

	if view := realm.View(hero).Shop; view == nil || view.Keeper != "Jagna" {
		t.Fatalf("no talk with Jagna: %+v", realm.View(hero).Log)
	}

	realm.LearnSelf(hero, "Jagna", []string{"Jagna ma córkę Zosię, która nurkuje po skarby."}, "Córkę, Zosię. Nurkuje po skarby.")

	return realm, hero
}

func TestRescueNamesAndForms(t *testing.T) {
	names := []string{"Szczur Polny", "Nurek bez Twarzy"}

	for text, want := range map[string]int{
		"Nurek bez Twarzy": 1, "nurka bez twarzy": 1, "Nurkiem bez Twarzy": 1, "Szczura Polnego": 0, "Twarz": -1, "Nurek": -1, "": -1,
	} {
		if got := named(names, text); got != want {
			t.Errorf("named(%q) = %d, want %d", text, got, want)
		}
	}

	for name, want := range map[string]string{"Zosia": "Zosię", "Radek": "Radka", "Jan": "Jana", "Janko": "Janka", "córka": "córkę", "syn": "syna", "ojciec": "ojca", "mąż": "męża", "Ruth": "Ruth"} {
		if got := accusative(name, rescueFeminine(name, "")); got != want {
			t.Errorf("accusative(%q) = %q, want %q", name, got, want)
		}
	}

	if !rescueFeminine("Kim", "córka") || rescueFeminine("Kuba", "brat") || rescueFeminine("Ola", "tata") || !rescueFeminine("Ola", "") {
		t.Error("the gender of the missing is wrong")
	}
}

func TestRescueGrounded(t *testing.T) {
	line := "Boję się, że kręci się koło Nurka bez Twarzy, tego w mosiężnym hełmie."
	known := []string{"Jagna ma córkę Zosię, która nurkuje po skarby."}

	if !RescueGrounded(diverHook, line, known) {
		t.Error("a hook from the bio and the line was not grounded")
	}

	for _, hook := range []QuestHook{
		{Person: "Zosia", Quote: "porwali ją zbójcy"},               // not said
		{Person: "Marta", Quote: "kręci się koło Nurka bez Twarzy"}, // nobody named so
		{Person: "Zosia", Quote: ""},
	} {
		if RescueGrounded(hook, line, known) {
			t.Errorf("grounded: %+v", hook)
		}
	}
}

func TestQuestHookAsksOnlyShopkeepersInFear(t *testing.T) {
	realm, hero := diverRealm(t)

	request, ok := realm.QuestHookRequest(hero, "a gdzie teraz jest twoja córka?", diverLine)
	if !ok || request.Speaker != "Jagna" || request.Reply != strings.TrimPrefix(diverLine, "[strach] ") || len(request.Known) != 1 ||
		!slices.Contains(request.Monsters, "Nurek bez Twarzy") || !slices.Contains(request.Places, "Las") || slices.Contains(request.Places, "Osada") {
		t.Fatalf("request = %+v, %v", request, ok)
	}

	// No fear, no hook: a daughter who dives is only a daughter.
	if _, ok := realm.QuestHookRequest(hero, "ma pani dzieci?", "Córkę, Zosię. Nurkuje po skarby."); ok {
		t.Error("a line with no danger was read")
	}

	// Fear of the world alone is no call for help, unless the speaker has
	// a past to say it of.
	other := testRealm(t)
	stranger := enlist(other, "Ola")
	do(other, stranger, "idź do zielarki")

	if _, ok := other.QuestHookRequest(stranger, "co słychać?", "Szczury Polne grasują, boję się wychodzić."); ok {
		t.Error("a line about the world was read")
	}

	if _, ok := realm.QuestHookRequest(hero, "co słychać?", "Boję się o nią, nie wróciła z lasu."); !ok {
		t.Error("fear after a told daughter was not read")
	}

	// One open rescue a keeper, and none while it lasts.
	if _, ok := realm.SpawnRescue(hero, "Jagna", diverHook, diverLine); !ok {
		t.Fatal("no rescue")
	}

	if _, ok := realm.QuestHookRequest(hero, "a gdzie teraz jest twoja córka?", diverLine); ok {
		t.Error("a keeper with a rescue open was asked again")
	}

	// A companion's fear is not a job.
	party := partyRealm(t)
	ala := enlist(party, "Ala")
	do(party, ala, "rozmawiaj z bogną")

	if _, ok := party.QuestHookRequest(ala, "masz brata?", "Mój brat zaginął na bagnach, boję się o niego."); ok {
		t.Error("a companion was asked for a rescue")
	}
}

func TestRescueTileIsWild(t *testing.T) {
	realm, _ := diverRealm(t)
	forest := slices.IndexFunc(realm.places, func(place world.PlaceSpec) bool { return place.Name == "Las" })

	for index := range 60 {
		at, ok := realm.rescueTile(home, index%2-1, forest*(index%3/2)-(1-index%3/2))
		tile := realm.world.At(at)

		if !ok || tile.Town >= 0 || tile.Site >= 0 || tile.Boss || at == realm.world.Lair || realm.world.Near(at) || realm.placeAt(at).Danger < 1 {
			t.Fatalf("a captive held at %v: %+v", at, tile)
		}
	}
}

func TestRescueFromTalkToReward(t *testing.T) {
	realm, hero := diverRealm(t)
	ala := realm.players[hero]

	told, ok := realm.SpawnRescue(hero, "Jagna", diverHook, diverLine)
	if !ok || !strings.HasPrefix(told, "Nowe zlecenie od Jagna: odnajdź i uwolnij Zosię (córka): strzeże jej Nurek bez Twarzy; miejsce: Las, od osady Osada") ||
		!strings.HasSuffix(told, "Po uwolnieniu wróć do Jagna po nagrodę.") || !logged(realm, hero, told) {
		t.Fatalf("told = %q, %v", told, ok)
	}

	job := realm.openRescue(1)
	diver := len(realm.world.Monsters) - 1

	if job == nil || job.taker != hero || job.monster != diver || job.reward <= 0 || !realm.revealed[realm.world.Index(job.pos)] {
		t.Fatalf("job = %+v", job)
	}

	guards := realm.foes(job.pos)
	if len(guards) < rescueGuards || !slices.ContainsFunc(guards, func(foe *monster) bool { return foe.kind == diver }) ||
		slices.ContainsFunc(guards, func(foe *monster) bool { return !foe.pinned }) {
		t.Fatalf("guards = %+v", guards)
	}

	// Everyone knows; the keeper speaks of it as of her own daughter.
	news := realm.wisdom.facts[len(realm.wisdom.facts)-1]
	if news.Subject != "Zosia" || news.Text != "Jagna martwi się o córkę Zosię: zaginęła w miejscu Las, strzeże jej Nurek bez Twarzy. Odnaleźć ją obiecała Ala." || !news.known[0] || !news.heard[hero] {
		t.Errorf("news = %+v", news)
	}

	_, narrate, _ := realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Twoja córka Zosia zaginęła; strzeże jej Nurek bez Twarzy w miejscu Las (kierunek") ||
		!strings.Contains(narrate.Lore, "Gracz obiecał ją odnaleźć i uwolnić. Martwisz się i czekasz na wieści; nie mów, że już wróciła.") {
		t.Errorf("lore = %q", narrate.Lore)
	}

	if lines := strings.Join(QuestLines(realm.Quests(hero)), "\n"); !strings.Contains(lines, "• Jagna: odnajdź i uwolnij Zosię (córka)") || !strings.Contains(lines, "Strażnicy wciąż stoją.") {
		t.Errorf("journal = %q", lines)
	}

	if mark := realm.View(hero).Marks[realm.world.Index(job.pos)]; !mark.Rescue || mark.Monsters < rescueGuards {
		t.Errorf("mark = %+v", mark)
	}

	// The guards never wander off.
	for range 40 {
		realm.roam()
	}

	if after := realm.foes(job.pos); len(after) != len(guards) {
		t.Fatalf("guards roamed: %d of %d left", len(after), len(guards))
	}

	// The hero walks up and fights; the guards fall and Zosia goes free.
	realm.mu.Lock()
	ala.shop, ala.pos = -1, world.Point{X: job.pos.X - 1, Y: job.pos.Y}
	if !realm.world.In(ala.pos) {
		ala.pos.X += 2
	}
	step := job.pos.X - ala.pos.X
	realm.mu.Unlock()

	realm.Move(hero, step, 0)
	win(t, realm, hero)

	if !job.finished || !logged(realm, hero, "Uwalniasz Zosię. Wraca z tobą do osady Osada. Wróć do Jagna po nagrodę.") {
		t.Fatalf("Zosia was not freed: %+v %q", job, realm.View(hero).Log)
	}

	// Back home Jagna pays: gold, experience, standing and news.
	realm.mu.Lock()
	ala.pos, ala.steps, ala.xp = home, 0, 0
	realm.mu.Unlock()

	gold, xp, standing := ala.gold, ala.xp, realm.standingOf(ala, 0)
	do(realm, hero, "idź do zielarki")

	if !job.paid || ala.gold != gold+job.reward || ala.xp != xp+rescueXP || realm.standingOf(ala, 0) != standing+rescueStanding ||
		!logged(realm, hero, "Jagna: Zosia wraca do domu cała. Doświadczenie: +25.") {
		t.Fatalf("not paid: %+v, gold %d→%d, xp %d→%d", job, gold, ala.gold, xp, ala.xp)
	}

	if told := realm.bioTexts("Jagna"); told[len(told)-1] != "Jagna odzyskała córkę Zosię: uwolniła ją Ala." {
		t.Errorf("Jagna's past = %q", told)
	}

	_, narrate, _ = realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	if strings.Contains(narrate.Lore, "zaginęła") || !strings.Contains(narrate.Lore, "Co już o sobie mówiłaś (to prawda, trzymaj się tego): Jagna ma córkę Zosię, która nurkuje po skarby. Jagna odzyskała córkę Zosię: uwolniła ją Ala.") ||
		!strings.Contains(narrate.Lore, "Wiesz: Zosia. Jagna odzyskała córkę Zosię: uwolniła ją Ala.") {
		t.Errorf("lore after the rescue = %q", narrate.Lore)
	}

	if realm.rescues() != 0 || len(realm.Quests(hero)) != 0 {
		t.Errorf("the rescue is still open: %+v", realm.Quests(hero))
	}
}

func TestRescueOfAClearedTile(t *testing.T) {
	realm, hero := diverRealm(t)
	if _, ok := realm.SpawnRescue(hero, "Jagna", diverHook, diverLine); !ok {
		t.Fatal("no rescue")
	}

	job := realm.openRescue(1)

	// Somebody else beat the guards; the hero only has to get there.
	realm.mu.Lock()
	realm.monsters[realm.world.Index(job.pos)] = nil
	ala := realm.players[hero]
	ala.shop, ala.pos = -1, world.Point{X: job.pos.X, Y: job.pos.Y - 1}
	if !realm.world.In(ala.pos) {
		ala.pos.Y += 2
	}
	step := job.pos.Y - ala.pos.Y
	realm.monsters[realm.world.Index(ala.pos)] = nil
	realm.mu.Unlock()

	realm.Move(hero, 0, step)

	if !job.finished || !logged(realm, hero, "Uwalniasz Zosię.") {
		t.Errorf("not freed on a cleared tile: %+v", job)
	}
}

func TestRescueRefusesWhatWasNotSaid(t *testing.T) {
	realm, hero := diverRealm(t)

	for _, hook := range []QuestHook{
		{Found: false, Person: "Zosia", Relation: "córka", Quote: "kręci się koło Nurka bez Twarzy"},
		{Found: true, Person: "Marta", Relation: "córka", Quote: "kręci się koło Nurka bez Twarzy"},
		{Found: true, Person: "Zosia", Relation: "", Quote: "kręci się koło Nurka bez Twarzy"},
		{Found: true, Person: "zosia", Relation: "córka", Quote: "kręci się koło Nurka bez Twarzy"},
		{Found: true, Person: "Jagna", Relation: "córka", Quote: "kręci się koło Nurka bez Twarzy"},
		{Found: true, Person: "Zosia", Relation: "córka", Quote: "porwali ją"},
	} {
		if _, ok := realm.SpawnRescue(hero, "Jagna", hook, diverLine); ok {
			t.Errorf("spawned: %+v", hook)
		}
	}

	// The caravan's merchant or a stranger has nobody to lose.
	if _, ok := realm.SpawnRescue(hero, "Nikt", diverHook, diverLine); ok {
		t.Error("a stranger asked for a rescue")
	}

	// At most a few rescues are open in the world.
	realm.world.Shops = append(realm.world.Shops, world.ShopSpec{Name: "Kram", Keeper: "Hanna"}, world.ShopSpec{Name: "Buda", Keeper: "Wit"})
	for _, keeper := range []string{"Kowal", "Jagna", "Hanna"} {
		if _, ok := realm.SpawnRescue(hero, keeper, diverHook, "Moja Zosia kręci się koło Nurka bez Twarzy."); !ok {
			t.Fatalf("no rescue for %s", keeper)
		}
	}

	if _, ok := realm.SpawnRescue(hero, "Wit", diverHook, "Moja Zosia kręci się koło Nurka bez Twarzy."); ok {
		t.Error("past the cap of open rescues")
	}
}

func TestRescueLastsThroughASave(t *testing.T) {
	realm, hero := diverRealm(t)
	if _, ok := realm.SpawnRescue(hero, "Jagna", diverHook, diverLine); !ok {
		t.Fatal("no rescue")
	}

	job := realm.openRescue(1)
	state := realm.State(hero)

	restored, err := Restore(state, 1)
	if err != nil {
		t.Fatal(err)
	}

	again := restored.openRescue(1)
	if again == nil || again.subject != "Zosia" || again.relation != "córka" || again.pos != job.pos || again.town != job.town || again.taker != hero {
		t.Fatalf("restored job = %+v", again)
	}

	guards := restored.foes(job.pos)
	if len(guards) == 0 || slices.ContainsFunc(guards, func(foe *monster) bool { return !foe.pinned }) {
		t.Errorf("restored guards = %+v", guards)
	}

	// A rescue without its tile is dropped; a job of a companion at the
	// inn survives; monsters of old saves roam.
	state.Spec.Companions = []world.CompanionSpec{{Name: "Bogna Topór", Description: "Bije mocno.", Role: world.RoleFighter, Tier: 1}}
	state.Quests = append(state.Quests,
		QuestState{Keeper: 1, Kind: string(questRescue), Subject: "Ola"},
		QuestState{Keeper: len(state.Spec.Shops) + 1, Kind: string(questHunt), Subject: "Szczur Polny", Count: 2, Reward: 30})
	for index := range state.Monsters {
		for slot := range state.Monsters[index] {
			state.Monsters[index][slot].Pinned = false
		}
	}

	if restored, err = Restore(state, 1); err != nil {
		t.Fatal(err)
	}

	if len(restored.quests) != 2 || restored.quests[1].keeper != len(state.Spec.Shops)+1 {
		t.Errorf("restored jobs = %+v", restored.quests)
	}

	if slices.ContainsFunc(restored.foes(job.pos), func(foe *monster) bool { return foe.pinned }) {
		t.Error("an old save pinned monsters")
	}
}
