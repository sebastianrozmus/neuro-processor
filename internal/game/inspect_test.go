package game

import (
	"strconv"
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
		if !realm.Describe(kind, index, "Nowy opis: "+kind+".", "", "") {
			t.Errorf("%s was not described", kind)
		}

		if _, text, _ := realm.Description(kind, index); text != "Nowy opis: "+kind+"." {
			t.Errorf("%s reads %q", kind, text)
		}
	}

	if realm.Describe(DescribePlace, 99, "x", "", "") || realm.Describe("nothing", 0, "x", "", "") || realm.Describe(DescribeItem, 0, "  ", "", "") {
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

func TestTheWorldRemembersWhereThingsCameFrom(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")

	// A fact noted from a line and an item talked into the world keep
	// the line; a rewrite keeps the old text and the prompt.
	if !realm.Learn(hero, FactSpec{Subject: "Młyn", Text: "W młynie straszy.", Quote: "W młynie straszy"}, "[kpina] W młynie straszy, mówię ci.") {
		t.Fatal("not learnt")
	}

	item, ok := realm.Invent(hero, world.ItemSpec{Name: "Hełm z Rondem", Description: "Garnek.", Kind: world.KindArmor, Tier: 1}, "[spokój] Mam też hełm z rondem.")
	if !ok {
		t.Fatal("not invented")
	}

	report := realm.Inspect()
	var fact *FactInfo
	for index := range report.Facts {
		if report.Facts[index].Subject == "Młyn" {
			fact = &report.Facts[index]
		}
	}

	if fact == nil {
		t.Fatal("no fact")
	}

	steps := report.Traces[traceKey(DescribeFact, fact.ID)]
	if len(steps) != 1 || !strings.HasPrefix(steps[0].How, HowNoted) || steps[0].Prompt != "[kpina] W młynie straszy, mówię ci." {
		t.Errorf("fact traces = %+v", steps)
	}

	if steps := report.Traces[traceKey(DescribeItem, item.ID)]; len(steps) != 1 || !strings.HasPrefix(steps[0].How, HowInvented+" (Kowal)") {
		t.Errorf("item traces = %+v", steps)
	}

	last := report.Items[len(report.Items)-1]
	if last.ID != item.ID || last.Keeper != "Kowal" {
		t.Errorf("invented item in the report = %+v", last)
	}

	if !realm.Describe(DescribeItem, last.Index, "Garnek z daszkiem.", HowRewritten, "PROMPT") {
		t.Fatal("the invented item was not described")
	}

	if steps := realm.Inspect().Traces[traceKey(DescribeItem, item.ID)]; len(steps) != 2 || steps[1].Was != "Garnek." || steps[1].Prompt != "PROMPT" || steps[1].How != HowRewritten {
		t.Errorf("rewrite traces = %+v", steps)
	}

	// Exchanges keep the request, and marks land on them and in the notes.
	request := NarrateRequest{Merchant: Merchant{Name: "Kowal"}, Text: "siema", Decision: Decision{Outcome: OutcomeChat, Mood: MoodPleased}, Hint: "H"}
	realm.Exchange(hero, request, "[spokój] Siema.", nil)
	realm.Exchange(hero, request, "Nie mój interes.", &Rejection{Attempts: []Attempt{{Text: "x 7", Reason: reasonNumbers}}})

	report = realm.Inspect()
	if len(report.Exchanges) != 2 || report.Exchanges[0].ID != 1 || report.Exchanges[1].Canned != true || len(report.Exchanges[1].Attempts) != 1 || report.Exchanges[0].Brief == "" {
		t.Fatalf("exchanges = %+v", report.Exchanges)
	}

	if got, entry, ok := realm.ExchangeRequest(2); !ok || got.Hint != "H" || entry.Reply != "Nie mój interes." {
		t.Errorf("exchange request = %+v %+v %v", got, entry, ok)
	}

	if _, ok := realm.Mark("exchange", []string{"2"}, "odczepka zamiast odpowiedzi"); !ok {
		t.Fatal("not marked")
	}

	if _, ok := realm.Mark("exchange", []string{"2"}, "  "); ok {
		t.Error("an empty comment was accepted")
	}

	realm.Mark(DescribeFact, []string{strconv.Itoa(fact.ID)}, "to nie fakt")
	report = realm.Inspect()
	if len(report.Notes) != 2 || len(report.Exchanges[1].Marks) != 1 || report.Exchanges[1].Marks[0] != "odczepka zamiast odpowiedzi" {
		t.Errorf("notes %+v, marks %+v", report.Notes, report.Exchanges[1].Marks)
	}

	if steps := report.Traces[traceKey(DescribeFact, fact.ID)]; len(steps) != 2 || steps[1].How != "uwaga" || steps[1].Why != "to nie fakt" {
		t.Errorf("fact traces after a mark = %+v", steps)
	}
}

func TestFactsMustBeGroundedInTheLine(t *testing.T) {
	line := "[spokój] Karczma to karczma, piwo sprzedaje sprzedawca. Jeśli chcesz wiedzieć coś więcej o wodzie, idź do Braumainera przy warzelni."
	for spec, want := range map[FactSpec]bool{
		{Subject: "Karczma i Woda", Text: "W karczmie sprzedają wodę.", Quote: ""}:                                 false,
		{Subject: "Karczma i Woda", Text: "W karczmie sprzedają wodę.", Quote: "woda w karczmie jest zatruta"}:     false,
		{Subject: "Braumainer", Text: "Braumainer prowadzi warzelnię.", Quote: "idź do Braumainera przy warzelni"}: true,
		{Subject: "Braumainer", Text: "Braumainer prowadzi warzelnię.", Quote: "Idź do braumainera przy warzelni"}: true,
		{Subject: "Stary Młyn", Text: "W młynie straszy.", Quote: "piwo sprzedaje sprzedawca"}:                     false,
	} {
		if Grounded(spec, line) != want {
			t.Errorf("Grounded(%+v) = %v", spec, !want)
		}
	}

	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")
	if realm.Learn(hero, FactSpec{Subject: "Karczma i Woda", Text: "W karczmie sprzedają wodę."}, line) {
		t.Error("an ungrounded fact was learnt")
	}
}
