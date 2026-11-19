package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// buyAt makes a hero buy the first ware of the shop they are in at the
// asking price.
func buyAt(t *testing.T, realm *Realm, hero string) Decision {
	t.Helper()

	shop := realm.View(hero).Shop
	if shop == nil || len(shop.Wares) == 0 {
		t.Fatalf("no shop to buy in: %+v", shop)
	}

	decision, _, ok := realm.Haggle(hero, "biorę", Intent{Action: ActionOffer, ItemID: shop.Wares[0].ID, Amount: shop.Wares[0].Price})
	if !ok || decision.Outcome != OutcomeSold {
		t.Fatalf("buying: %+v", decision)
	}

	return decision
}

func TestStandingMovesWithTradeAndInsults(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500

	do(realm, hero, "idź do zbrojowni")
	buyAt(t, realm, hero)

	if score := realm.standingOf(realm.players[hero], 0); score != tradeStanding {
		t.Errorf("standing after a purchase = %d", score)
	}

	if !logged(realm, hero, "W osadzie Osada jesteś teraz: znany (+1).") {
		t.Errorf("the change of standing was not told: %v", realm.View(hero).Log)
	}

	// An insulting offer costs two points.
	shop := realm.View(hero).Shop
	decision, _, _ := realm.Haggle(hero, "grosz", Intent{Action: ActionOffer, ItemID: shop.Wares[0].ID, Amount: 1})
	if decision.Outcome != OutcomeInsulted {
		t.Fatalf("a bid of one did not insult: %+v", decision)
	}

	if score := realm.standingOf(realm.players[hero], 0); score != tradeStanding+insultStanding {
		t.Errorf("standing after an insult = %d", score)
	}

	// The shunned are not served at all; the liked pay less.
	realm.CloseShop(hero)
	realm.players[hero].standing[0] = shunnedAt
	realm.players[hero].talks = nil
	do(realm, hero, "idź do zbrojowni")

	if realm.players[hero].shop >= 0 || !logged(realm, hero, "Z tobą nie handlujemy") {
		t.Errorf("the shunned hero was served: %v", realm.View(hero).Log)
	}

	realm.players[hero].standing[0] = standingMax
	do(realm, hero, "idź do zbrojowni")

	full := realm.merchant(0, world.MaxTier).Items[0].ListPrice
	if price := realm.View(hero).Shop.Wares[0].Price; price != full*(100-standingMax*standingDiscount)/100 {
		t.Errorf("price for a hero of the settlement = %d, list %d", price, full)
	}

	if fame := realm.View(hero).Me.Standing; len(fame) != 1 || fame[0].Label != "bohater osady" {
		t.Errorf("standing view = %+v", fame)
	}
}

func TestGuardingTheSettlementEarnsStanding(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")

	do(realm, hero, "e")
	win(t, realm, hero)

	if score := realm.standingOf(realm.players[hero], 0); score != guardStanding {
		t.Errorf("standing after a kill by the palisade = %d", score)
	}
}

func TestQuestsGrowFromFacts(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 0

	var ratFact *Fact
	for _, fact := range realm.wisdom.facts {
		if fact.Subject == "Szczur Polny" {
			ratFact = fact
		}
	}

	if ratFact == nil || !ratFact.known[0] {
		t.Fatalf("the smith does not know the rats: %+v", ratFact)
	}

	// Facts become jobs once they have aged.
	realm.day = 1 + questAge
	job := realm.questFrom(0, ratFact)
	if job == nil || job.kind != questHunt || job.monster != 0 || job.count < 2 {
		t.Fatalf("job from the rat fact: %+v", job)
	}

	job.count = 3
	realm.quests = append(realm.quests, job)

	do(realm, hero, "idź do zbrojowni")
	if !logged(realm, hero, "Zlecenie od Kowal: ubij 3 sztuk: Szczur Polny. Nagroda: 45 kapsli") {
		t.Fatalf("the job was not offered: %v", realm.View(hero).Log)
	}

	do(realm, hero, "przyjmij")
	if job.taker != hero || !logged(realm, hero, "Przyjmujesz zlecenie od Kowal") {
		t.Fatalf("the job was not taken: %+v", job)
	}

	realm.CloseShop(hero)
	do(realm, hero, "e")
	win(t, realm, hero)

	if job.done != 1 || !logged(realm, hero, "Zlecenie: 1 z 3.") {
		t.Fatalf("progress after the first rat: %+v", job)
	}

	do(realm, hero, "w", "w")
	win(t, realm, hero)

	if !job.finished || !logged(realm, hero, "Zlecenie wykonane. Wróć do Kowal po nagrodę.") {
		t.Fatalf("the hunt did not finish: %+v", job)
	}

	gold := realm.players[hero].gold
	do(realm, hero, "e", "idź do zbrojowni")

	if !job.paid || realm.players[hero].gold != gold+45 || !logged(realm, hero, "Kowal płaci za zlecenie: 45 kapsli.") {
		t.Fatalf("the job was not paid: %+v, gold %d", job, realm.players[hero].gold)
	}

	// Standing: three kills by the palisade, the job, and the purchase-free rest.
	if score := realm.standingOf(realm.players[hero], 0); score != 3*guardStanding+questStanding {
		t.Errorf("standing after the job = %d", score)
	}

	last := realm.wisdom.facts[len(realm.wisdom.facts)-1]
	if !strings.Contains(last.Text, "Ala wykonał zlecenie Kowal") || !last.known[1] {
		t.Errorf("the deed is not known: %+v", last)
	}

	if jobs := realm.Quests(hero); len(jobs) != 0 {
		t.Errorf("paid jobs still listed: %+v", jobs)
	}

	// Keepers post jobs by themselves, given a few days.
	for range 20 {
		realm.postQuests()
	}

	if realm.openQuest(0) == nil && realm.openQuest(1) == nil {
		t.Error("no keeper posted a job")
	}
}

func TestWeatherShapesTheRoad(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]

	// Fog: the land beyond one's own tile stays hidden.
	realm.weather = WeatherFog
	do(realm, hero, "s")

	if ala.steps != 1 || realm.revealed[realm.world.Index(outpost)] || !logged(realm, hero, "Mgła: widać tylko to, na czym stoisz") {
		t.Errorf("the fog did not hide the outpost: %d steps, %v", ala.steps, realm.View(hero).Log)
	}

	// Heat: no power comes back on the march.
	realm.weather = WeatherHeat
	ala.mana = 0
	do(realm, hero, "n")

	if ala.mana != 0 || ala.steps != 2 {
		t.Errorf("in the heat: mana %d, steps %d", ala.mana, ala.steps)
	}

	// Rain: every step takes two hours.
	realm.weather = WeatherRain
	do(realm, hero, "s")

	if ala.steps != 4 || !realm.revealed[realm.world.Index(outpost)] || !logged(realm, hero, "Deszcz: każdy krok zajmuje dwie godziny.") {
		t.Errorf("rain did not slow the hero: %d steps, %v", ala.steps, realm.View(hero).Log)
	}

	realm.weather = WeatherClear
	ala.mana = 0
	do(realm, hero, "n")

	if ala.mana != manaPerStep || ala.steps != 5 {
		t.Errorf("in clear weather: mana %d, steps %d", ala.mana, ala.steps)
	}

	// Winter dresses the land in snow and makes everything dearer.
	realm.day = 3*daysPerSeason + 1
	if realm.season() != "zima" || realm.View(hero).Places[1].Terrain != "snow" || realm.View(hero).Season != "zima" {
		t.Errorf("winter did not come: %s, %s", realm.season(), realm.View(hero).Places[1].Terrain)
	}

	for range 30 {
		realm.drift()
	}

	if markup := realm.markupOf("item-1"); markup < 105 {
		t.Errorf("winter prices = %d", markup)
	}

	// Every season has its weather.
	seen := map[Weather]bool{}
	for day := 1; day <= 4*daysPerSeason; day++ {
		realm.day = day
		realm.rollWeather()
		seen[realm.weather] = true
	}

	if len(seen) < 4 {
		t.Errorf("weathers seen over a year: %v", seen)
	}
}

func TestTheCaravanTravelsAndTrades(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].gold = 500

	if realm.caravan == nil || realm.caravan.pos != outpost {
		t.Fatalf("the caravan did not set out from the outpost: %+v", realm.caravan)
	}

	// Two steps a day: the outpost, the shrine, home.
	realm.travel()
	if realm.caravan.pos != home || !realm.View(hero).Marks[realm.world.Index(home)].Caravan {
		t.Fatalf("the caravan after a day: %+v", realm.caravan)
	}

	do(realm, hero, "rozejrzyj się")
	if !logged(realm, hero, "Stoi tu karawana wędrownego kupca") {
		t.Errorf("the caravan is not seen: %v", realm.View(hero).Log)
	}

	do(realm, hero, "handluj z karawaną")
	shop := realm.View(hero).Shop
	if shop == nil || shop.Keeper != caravanKeeper || shop.Kind != world.ShopCaravan {
		t.Fatalf("talking to the caravan: %+v", shop)
	}

	// The caravan carries the best, which the first settlement does not,
	// at road prices.
	var axe *Ware
	for _, ware := range shop.Wares {
		if ware.Name == "Topór" {
			axe = &ware
		}
	}

	list, _ := realm.world.Item("item-9")
	if axe == nil || axe.Price != list.Price()*caravanMarkup/100 {
		t.Fatalf("the axe at the caravan: %+v, list %d", axe, list.Price())
	}

	buyAt(t, realm, hero)
	if len(realm.players[hero].bag) != 1 {
		t.Errorf("nothing bought from the caravan: %+v", realm.players[hero].bag)
	}

	// The caravan knows what the keepers know and lends its ear to the graph.
	if facts := realm.knownBy(realm.caravanShop()); len(facts) == 0 {
		t.Error("the caravan knows nothing")
	}

	// It travels on and waits where monsters bar the way.
	realm.travel()
	realm.travel()
	if realm.caravan.goal != 1 || realm.caravan.pos == home {
		t.Errorf("the caravan did not head back: %+v", realm.caravan)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil || restored.caravan == nil || restored.caravan.pos != realm.caravan.pos {
		t.Errorf("the caravan was not saved: %+v, %v", restored.caravan, err)
	}
}

func TestKeepersKeepHours(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]
	ala.gold = 500

	ala.steps = innFrom
	do(realm, hero, "idź do zbrojowni")

	if !ala.inn || !logged(realm, hero, "Kowal siedzi w karczmie przy stole") {
		t.Fatalf("the smith is not at the inn: %v", realm.View(hero).Log)
	}

	shop := realm.View(hero).Shop
	decision, request, _ := realm.Haggle(hero, "biorę", Intent{Action: ActionOffer, ItemID: shop.Wares[0].ID, Amount: shop.Wares[0].Price})
	if decision.Outcome != OutcomeChat || ala.gold != 500 || !strings.Contains(request.Hint, "kram zamknięty") || !strings.Contains(request.Lore, "siedzisz w karczmie") {
		t.Errorf("a deal went through after hours: %+v, gold %d, hint %q", decision, ala.gold, request.Hint)
	}

	// Over the table the keeper picks up what the neighbours know.
	realm.add("Młyn", "W młynie nocą świeci światło.", "", 1)
	known := len(realm.knownBy(0))
	for range 10 {
		realm.Haggle(hero, "co słychać?", Intent{Action: ActionChat})
	}

	if len(realm.knownBy(0)) <= known {
		t.Error("no gossip at the inn")
	}

	realm.CloseShop(hero)
	ala.steps = sleepFrom
	do(realm, hero, "idź do zbrojowni")

	if ala.shop >= 0 || !logged(realm, hero, "Głęboka noc: wszyscy śpią.") {
		t.Errorf("served at night: %v", realm.View(hero).Log)
	}

	// Morning comes with rest.
	do(realm, hero, "odpocznij", "idź do zbrojowni")
	if ala.inn || ala.shop != 0 || !logged(realm, hero, "Wchodzisz: Zbrojownia.") {
		t.Errorf("the shop did not open in the morning: %v", realm.View(hero).Log)
	}
}

func TestTheWorldOutlivesItsBoss(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]

	ala.pos = world.Point{X: realm.world.Lair.X + 1, Y: realm.world.Lair.Y}
	do(realm, hero, "w")
	win(t, realm, hero)

	if !realm.won || realm.heir == nil || realm.heir.kind != 0 || realm.heir.due != realm.day+heirDays {
		t.Fatalf("after the boss: won %v, heir %+v", realm.won, realm.heir)
	}

	if markup := realm.markupOf("item-1"); markup != 100+peaceMarkup {
		t.Errorf("prices after the victory = %d", markup)
	}

	if realm.standingOf(ala, 0) != peaceStanding || realm.standingOf(ala, 1) != peaceStanding {
		t.Errorf("standing after the victory: %v", ala.standing)
	}

	if !logged(realm, hero, "coś zbiera się w pustym leżu") {
		t.Errorf("no word of what comes next: %v", realm.View(hero).Log)
	}

	// Days pass; the heir rises in the lair.
	realm.worldDay(realm.day + heirDays)

	lair := realm.foes(realm.world.Lair)
	if realm.won || realm.heir != nil || len(lair) != 1 || !lair[0].boss || lair[0].name != "Szczur Polny Alfa" {
		t.Fatalf("the heir did not rise: won %v, lair %+v", realm.won, lair)
	}

	if !logged(realm, hero, "Nowe zagrożenie: Szczur Polny Alfa") {
		t.Errorf("the heir was not announced: %v", realm.View(hero).Log)
	}

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	if saved := restored.foes(realm.world.Lair); len(saved) != 1 || !saved[0].boss || saved[0].name != "Szczur Polny Alfa" {
		t.Errorf("the heir was not saved: %+v", saved)
	}

	// Beating the heir starts the cycle again.
	ala.pos = world.Point{X: realm.world.Lair.X + 1, Y: realm.world.Lair.Y}
	do(realm, hero, "w")
	win(t, realm, hero)

	if !realm.won || !logged(realm, hero, "Szczur Polny Alfa pokonany.") {
		t.Errorf("the heir's fall: won %v, %v", realm.won, realm.View(hero).Log)
	}
}

func TestCompanionsHaveOpinions(t *testing.T) {
	realm := partyRealm(t)
	hero := enlist(realm, "Ala")
	ala := realm.players[hero]
	ala.gold = 500

	do(realm, hero, "s", "s")
	ala.standing = map[int]int{1: -1}
	do(realm, hero, "najmij jaromira")

	if len(ala.party) != 0 || !logged(realm, hero, "Brat Jaromir: «Z kimś, kogo tu mają za podejrzany? Nie.»") {
		t.Fatalf("the disliked hero was followed: %v", realm.View(hero).Log)
	}

	ala.standing[1] = 0
	do(realm, hero, "najmij jaromira")
	if len(ala.party) != 1 {
		t.Fatalf("hiring failed: %v", realm.View(hero).Log)
	}

	// Deadly country: a warning first, then the companion turns back.
	realm.places[1].Danger = world.MaxTier
	ala.pos, ala.prev = home, home
	do(realm, hero, "s")

	if len(ala.party) != 1 || !ala.party[0].warned || !logged(realm, hero, "Brat Jaromir: «To śmiertelne strony") {
		t.Fatalf("no warning: %v", realm.View(hero).Log)
	}

	do(realm, hero, "n", "s")
	if len(ala.party) != 0 || !logged(realm, hero, "Brat Jaromir zawraca do karczmy w osadzie Przystań") {
		t.Fatalf("the companion stayed: %v", realm.View(hero).Log)
	}

	// Nobody walks with the outcast of their home settlement.
	realm.places[1].Danger = 1
	ala.pos, ala.prev = outpost, outpost
	do(realm, hero, "najmij jaromira")
	ala.standing[1] = shunnedAt
	do(realm, hero, "odpocznij")

	if len(ala.party) != 0 || !logged(realm, hero, "Brat Jaromir odchodzi: «Nie będę chodzić z kimś, kogo w osadzie Przystań mają za wyklętego.»") {
		t.Errorf("the outcast kept the companion: %v", realm.View(hero).Log)
	}
}

func TestStandingSurvivesTheSave(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	realm.players[hero].standing = map[int]int{0: 4, 1: -2}
	realm.weather = WeatherFog

	restored, err := Restore(realm.State(hero), 1)
	if err != nil {
		t.Fatal(err)
	}

	if saved := restored.players[hero].standing; saved[0] != 4 || saved[1] != -2 || restored.weather != WeatherFog {
		t.Errorf("after the save: standing %v, weather %s", saved, restored.weather)
	}
}

func TestEventsMoveStanding(t *testing.T) {
	realm := testRealm(t)
	realm.world.Events = []world.EventSpec{{ID: "pomoc", When: world.WhenSpec{Enter: "Las"}, Once: true, Text: "Ratujesz drwala.", Standing: 2}}
	hero := enlist(realm, "Ala")

	do(realm, hero, "s")
	if score := realm.standingOf(realm.players[hero], 0); score != 2 {
		t.Errorf("standing after the event = %d", score)
	}
}
