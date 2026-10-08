package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// presetRealm builds a realm from a built-in preset by name.
func presetRealm(t *testing.T, name string) (*Realm, string) {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	for _, spec := range presets {
		if spec.Name != name {
			continue
		}

		built, err := world.Build(spec)
		if err != nil {
			t.Fatal(err)
		}

		realm := NewRealm(built, 1)

		return realm, realm.Join("Detektyw", "")
	}

	t.Fatalf("no preset %q", name)

	return nil, ""
}

// at puts a hero into a named room without walking.
func at(t *testing.T, realm *Realm, hero string, room string) {
	t.Helper()

	pos, ok := realm.roomPos(room)
	if !ok {
		t.Fatalf("no room %q", room)
	}

	who := realm.players[hero]
	who.pos, who.prev = pos, pos
	who.shop, who.talks = -1, nil
}

// hourOf sets a hero's clock to the first step of a named time of day.
func hourOf(realm *Realm, hero string, hour string) {
	steps := map[string]int{"ranek": 0, "popołudnie": stepsPerDay / 2, "wieczór": innFrom, "noc": sleepFrom}
	realm.players[hero].steps = steps[hour]
}

func TestInvestigationFollowsTheClues(t *testing.T) {
	realm, hero := presetRealm(t, "Jeziorna, Kanał 7")
	ala := realm.players[hero]

	if !realm.investigation() || !realm.adventure() || realm.world.Case == nil {
		t.Fatalf("not an investigation: %v %v", realm.world.Mode, realm.world.Case)
	}

	// The client briefs the detective: a scripted talk that leaves a fact.
	do(realm, hero, "rozmawiaj z Haliną")
	if !realm.heardOf(ala, "Zlecenie Haliny") || !logged(realm, hero, "Zapisujesz w dzienniku: Zlecenie Haliny.") {
		t.Fatalf("no brief: %v", realm.View(hero).Log)
	}

	// A second talk is free, run by the model from what Halina knows.
	do(realm, hero, "rozmawiaj z Haliną")
	if ala.shop != 0 || realm.View(hero).Shop == nil || realm.View(hero).Shop.Keeper != "Halina Sowa" {
		t.Fatalf("no free talk with Halina: shop %d", ala.shop)
	}

	request, ok := realm.HaggleRequest(hero, "co pani wie?")
	if !ok || len(request.Wares) != 0 {
		t.Fatalf("talk request: %+v", request)
	}

	_, narrate, _ := realm.Haggle(hero, "co pani wie?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Nie prowadzisz sklepu. Jesteś: Halina Sowa") || !strings.Contains(narrate.Role, "detektyw") {
		t.Errorf("lore for a person: %q", narrate.Lore)
	}

	// The waitress knows about the park, but only tells somebody on the case.
	at(t, realm, hero, "Pod Sumem")

	do(realm, hero, "rozmawiaj z Basią")
	_, narrate, _ = realm.Haggle(hero, "widziałaś Marka w parku?", Intent{Action: ActionChat})
	if !strings.Contains(narrate.Lore, "Wiesz: Spotkania w parku") || !strings.Contains(narrate.Hint, "Spotkania w parku") || !realm.heardOf(ala, "Spotkania w parku") {
		t.Fatalf("Basia did not tell about the park: %q / %q", narrate.Lore, narrate.Hint)
	}

	// Rysiek's clue waits for that one; before it the technician knows nothing of it.
	if open := realm.openTo(ala, realm.personIndex("Rysiek")); len(open) != 2 {
		t.Errorf("Rysiek's open clues after the park: %d", len(open))
	}

	// Marek's locker opens only after his shift.
	realm.CloseShop(hero)
	at(t, realm, hero, "Zaplecze Pod Sumem")
	hourOf(realm, hero, "ranek")
	do(realm, hero, "zbadaj szafkę")

	if realm.heardOf(ala, "Numer z szafki") {
		t.Fatal("the locker gave up the number in the morning")
	}

	hourOf(realm, hero, "wieczór")
	do(realm, hero, "zbadaj szafkę")

	if !realm.heardOf(ala, "Numer z szafki") {
		t.Fatalf("the locker kept the number in the evening: %v", realm.View(hero).Log)
	}

	// The phone book turns the number into a name and an address.
	at(t, realm, hero, "Poczta")
	do(realm, hero, "zbadaj książkę telefoniczną")

	if !realm.heardOf(ala, "Adres Iwony") || !realm.story.flags["adres"] {
		t.Fatalf("the phone book gave no address: %v", realm.View(hero).Log)
	}

	// The right name without the photo is not enough; a wrong name costs.
	at(t, realm, hero, "Rynek")
	do(realm, hero, "oskarżam Iwonę")
	if realm.won || !logged(realm, hero, "brakuje 1 z 4 poszlak") {
		t.Fatalf("accusation without proof: %v", realm.View(hero).Log)
	}

	do(realm, hero, "oskarżam Zenona")
	if realm.story.mistakes != 1 || !realm.story.flags[refusal("Zenon Kruk")] {
		t.Fatalf("a wrong accusation: mistakes %d", realm.story.mistakes)
	}

	hourOf(realm, hero, "ranek")
	at(t, realm, hero, "Cmentarz")
	do(realm, hero, "rozmawiaj z Zenonem")
	if ala.shop >= 0 || !logged(realm, hero, "nie zamieni z tobą słowa") {
		t.Errorf("the accused still talks: %v", realm.View(hero).Log)
	}

	// At night both lovers are on the pier: waiting there takes the photo.
	at(t, realm, hero, "Park nad Jeziorem")
	hourOf(realm, hero, "noc")
	ala.steps = sleepFrom - 1
	do(realm, hero, "czekaj")

	if !realm.heardOf(ala, "Zdjęcie z parku") || !logged(realm, hero, "Przychodzi: Marek Sowa, Iwona Malec.") {
		t.Fatalf("no photo at night: %v", realm.View(hero).Log)
	}

	do(realm, hero, "oskarżam Iwonę Malec")
	if !realm.won || !logged(realm, hero, "Sprawa «Jeziorna, Kanał 7» rozwiązana!") {
		t.Fatalf("the case did not close: %v", realm.View(hero).Log)
	}

	if journal := realm.Journal(hero); len(journal) < 5 {
		t.Errorf("journal = %+v", journal)
	}
}

func TestPeopleKeepSchedulesAndCanBeFollowed(t *testing.T) {
	realm, hero := presetRealm(t, "Jeziorna, Kanał 7")
	ala := realm.players[hero]

	at(t, realm, hero, "Pod Sumem")
	hourOf(realm, hero, "popołudnie")
	ala.steps = innFrom - 1

	if people := personNames(realm.peopleAt(ala.pos, ala)); !contains(people, "Marek Sowa") {
		t.Fatalf("Marek is not at work in the afternoon: %v", people)
	}

	do(realm, hero, "śledź Marka")
	if ala.following != "Marek Sowa" {
		t.Fatalf("not following: %q", ala.following)
	}

	do(realm, hero, "czekaj")
	if realm.roomOf(ala).Name != "Dom Sowów" || !logged(realm, hero, "Wychodzi: Marek Sowa (do: Dom Sowów)") || !logged(realm, hero, "Idziesz za Marek Sowa.") {
		t.Fatalf("did not follow Marek home: in %s, %v", realm.roomOf(ala).Name, realm.View(hero).Log)
	}

	// Views carry the hour and the clock.
	view := realm.View(hero)
	if view.Hour != "wieczór" || view.HoursLeft != 36-innFrom || !view.Investigation() || !view.Rooms() {
		t.Errorf("view clock: %s, %d left", view.Hour, view.HoursLeft)
	}

	// The deadline ends the case.
	ala.day, ala.steps = 3, stepsPerDay-1
	do(realm, hero, "czekaj")

	if !realm.lost || !logged(realm, hero, "Halina przestaje odbierać telefon.") || !realm.View(hero).Lost {
		t.Fatalf("the deadline did not pass: lost %v", realm.lost)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil || !restored.lost || restored.players[hero].following != "Marek Sowa" {
		t.Errorf("the case was not saved: %v, lost %v", err, restored.lost)
	}
}

func TestEscapeRoomCodesAndClock(t *testing.T) {
	realm, hero := presetRealm(t, "Pracownia Zegarmistrza")
	ala := realm.players[hero]

	do(realm, hero, "n")
	if realm.roomOf(ala).Name != "Piwnica" || !logged(realm, hero, "Bez światła skręcisz kark.") {
		t.Fatalf("walked into the dark: %v", realm.View(hero).Log)
	}

	do(realm, hero, "podnieś wszystko", "użyj zapałek na świecę", "n", "e", "podnieś śrubokręt", "zbadaj księgę", "w", "użyj śrubokrętu na zegar", "s", "użyj klucza na skrzynię", "użyj korby na kratę", "e")
	if realm.roomOf(ala).Name != "Przedsionek" {
		t.Fatalf("did not reach the exit: in %s, %v", realm.roomOf(ala).Name, realm.View(hero).Log)
	}

	do(realm, hero, "wpisz 0415")
	if realm.won || !logged(realm, hero, "Zamek brzęczy") {
		t.Fatalf("the wrong code: %v", realm.View(hero).Log)
	}

	left := realm.View(hero).HoursLeft
	if left != 10-ala.steps {
		t.Errorf("hours left = %d, steps %d", left, ala.steps)
	}

	do(realm, hero, "1879")
	if !realm.won || !logged(realm, hero, "Przygoda «Pracownia Zegarmistrza» ukończona!") {
		t.Fatalf("the right code: %v", realm.View(hero).Log)
	}

	// Dawdling drowns.
	realm, hero = presetRealm(t, "Pracownia Zegarmistrza")
	realm.players[hero].steps = 9
	do(realm, hero, "czekaj")

	if !realm.lost || !logged(realm, hero, "Woda sięga sufitu.") {
		t.Errorf("the flood did not come: %v", realm.View(hero).Log)
	}
}
