// Command synth generates training data for the game's own language model
// by playing conversations with shopkeepers and fights with a game master
// against the configured model. The game's rules decide everything, as
// they do in play; the model only words the replies, and only replies
// that pass the game's checks are kept. The output has exactly the shape
// of the talks the game records (NEURO_LOG_TALKS=1).
//
//	go run ./tools/synth -n 200 -fights 20 -out synth.jsonl
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/config"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const turnTimeout = 2 * time.Minute

var (
	heroNames = []string{"Ala", "Borys", "Celina", "Dobromir", "Ewa", "Florian", "Gaja", "Hubert", "Iga", "Jarek", "Kasia", "Leon", "Marta", "Nikodem", "Ola", "Piotr"}
	avatarIDs = []string{"wojownik", "lowczyni", "pilot", "czarodziejka", "lotrzyk", "mnich"}

	chatLines = []string{
		"Co słychać w okolicy?", "Kim jesteś i od kiedy tu handlujesz?", "Gdzie znajdę %s?", "Boję się iść do miejsca %s. Co o nim wiesz?",
		"Słyszałeś coś o potworze %s?", "Jaka jest najkrótsza droga do drugiej osady?", "Masz jakąś radę dla początkującego?",
		"Ładna pogoda, prawda?", "Czy ktoś ostatnio kupował u ciebie broń?", "Kto rządzi w tej osadzie?", "Opowiedz mi coś o sobie.",
		"Czy to prawda, co mówią o %s?", "Potrzebuję czegoś na długą drogę. Co radzisz?", "Nie mam dużo pieniędzy. Co mi polecisz?",
		"Widziałem na drodze %s. Co to było?", "Dlaczego tu tak pusto?", "Pracujesz tu sam?", "Co myślisz o %s?",
	}
	rudeLines  = []string{"Twój towar to złom, a ty złodziej.", "Zamknij się i podaj cenę, dziadu.", "Śmierdzi tu jak w chlewie."}
	waresLines = []string{"Co masz na sprzedaż?", "Pokaż towar.", "Co dziś w ofercie?", "Czym handlujesz?"}
	askLines   = []string{"Ile za %s?", "Co to jest %s?", "Opowiedz mi o %s.", "Ile chcesz za %s?", "Pokaż mi %s."}
	offerLines = []string{"Dam %d.", "Dam za to %d.", "Zapłacę %d i ani grosza więcej.", "Mogę dać %d.", "%d. To moja cena.", "Proponuję %d, bo rozbój to cena."}
	bulkLines  = []string{"Kupię %d sztuk %s za %d.", "Wezmę %d x %s, dam %d."}
	sellLines  = []string{"Sprzedam %s za %d.", "Odkupisz %s? Chcę %d.", "Kupisz ode mnie %s? Dam za %d.", "Oddam %s za %d."}
	byeLines   = []string{"Żegnaj.", "Do widzenia.", "Bywaj, wrócę jutro.", "Na razie."}
	okLines    = []string{"Biorę.", "Zgoda.", "Dobra, kupuję.", "Umowa stoi."}
)

func main() {
	var (
		talks  = flag.Int("n", 50, "how many conversations to play")
		fights = flag.Int("fights", 0, "how many fights to narrate")
		seed   = flag.Int64("seed", time.Now().UnixNano(), "seed of the situations")
		out    = flag.String("out", "", "file to append the talks to (default: synth.jsonl in the game's data folder)")
	)
	flag.Parse()

	cfg := config.Load()
	if cfg.Offline() {
		fmt.Fprintln(os.Stderr, "synth: a model is needed: set NEURO_LLM_MODEL (and NEURO_LLM_BASE_URL) or put them in .env.local")
		os.Exit(1)
	}

	if *out == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "synth:", err)
			os.Exit(1)
		}

		*out = filepath.Join(dir, "neuro-processor", "synth.jsonl")
	}

	client := &llm.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model, MaxTokens: cfg.MaxTokens, Context: cfg.Context}
	presets, err := world.Presets()
	if err != nil {
		fmt.Fprintln(os.Stderr, "synth:", err)
		os.Exit(1)
	}

	var worlds []world.Spec
	for _, spec := range presets {
		if spec.Mode != world.ModeAdventure {
			worlds = append(worlds, spec)
		}
	}

	rng := rand.New(rand.NewSource(*seed))
	sink := newSink(*out)
	kept, tried := 0, 0

	for talk := 0; talk < *talks; talk++ {
		spec := worlds[rng.Intn(len(worlds))]
		built, err := world.Build(spec)
		if err != nil {
			continue
		}

		realm := game.NewRealm(built, rng.Int63())
		hero := realm.Join(heroNames[rng.Intn(len(heroNames))], avatarIDs[rng.Intn(len(avatarIDs))])
		shop := built.Shops[rng.Intn(len(built.Shops))]
		realm.Execute(hero, game.Parse("idź do "+shop.Name))

		if realm.View(hero).Shop == nil {
			continue
		}

		for _, line := range plan(rng, realm, hero) {
			tried++

			reply, ok := exchange(client, sink, realm, hero, line)
			if !ok {
				break
			}

			kept++
			fmt.Fprintf(os.Stderr, "[%d/%d] %s | %s: %s\n   %s: %s\n", kept, tried, spec.Name, hero, line, shop.Keeper, reply)

			if realm.View(hero).Shop == nil || realm.View(hero).Shop.Over {
				break
			}
		}
	}

	for fight := 0; fight < *fights; fight++ {
		spec := worlds[rng.Intn(len(worlds))]
		built, err := world.Build(spec)
		if err != nil {
			continue
		}

		realm := game.NewRealm(built, rng.Int63())
		realm.SetMaster(client, func(request game.NarrateRequest, reply string) {
			sink.write(request, reply)
			fmt.Fprintf(os.Stderr, "[walka %d] %s\n", fight+1, reply)
		})

		brawl(rng, realm, built)
	}

	// Narrations arrive asynchronously; give the last ones time to land.
	if *fights > 0 {
		time.Sleep(masterGrace)
	}

	fmt.Fprintf(os.Stderr, "zapisano %d wymian do %s\n", sink.count(), *out)
}

const masterGrace = 90 * time.Second

// plan draws a short conversation that fits the shop: what the player
// will say, turn by turn. Later turns depend on earlier replies only
// through the rules, never through the model.
func plan(rng *rand.Rand, realm *game.Realm, hero string) []string {
	view := realm.View(hero)
	wares := view.Shop.Wares
	spec := view.World

	pick := func(list []string) string { return list[rng.Intn(len(list))] }
	ware := func() game.Ware { return wares[rng.Intn(len(wares))] }
	lore := func() string {
		switch rng.Intn(4) {
		case 0:
			return spec.Boss.Name
		case 1:
			return spec.Places[rng.Intn(len(spec.Places))].Name
		case 2:
			return spec.Monsters[rng.Intn(len(spec.Monsters))].Name
		}

		return spec.Shops[rng.Intn(len(spec.Shops))].Keeper
	}

	switch rng.Intn(9) {
	case 0:
		return []string{fill(pick(chatLines), lore()), fill(pick(chatLines), lore())}
	case 1:
		item := ware()

		return []string{fmt.Sprintf(pick(askLines), item.Name), fmt.Sprintf(pick(offerLines), item.Price*(50+rng.Intn(30))/100), pick(okLines)}
	case 2:
		item := ware()

		return []string{fmt.Sprintf(pick(askLines), item.Name), fmt.Sprintf(pick(offerLines), item.Price/5), pick(byeLines)}
	case 3:
		return []string{pick(waresLines), fmt.Sprintf(pick(askLines), ware().Name)}
	case 4:
		return []string{fill(pick(chatLines), lore()), pick(rudeLines)}
	case 5:
		if potion := cheapest(wares); potion.ID != "" {
			count := 2 + rng.Intn(4)

			return []string{fmt.Sprintf(pick(bulkLines), count, potion.Name, potion.Price*count*(60+rng.Intn(30))/100)}
		}

		return []string{pick(waresLines)}
	case 6:
		// Buy the cheapest thing, then try to sell it back.
		if potion := cheapest(wares); potion.ID != "" && potion.Price <= view.Me.Gold {
			return []string{fmt.Sprintf("Interesuje mnie %s.", potion.Name), pick(okLines), fmt.Sprintf(pick(sellLines), potion.Name, potion.Price*(40+rng.Intn(40))/100)}
		}

		return []string{fill(pick(chatLines), lore())}
	case 7:
		return []string{fill(pick(chatLines), lore()), fmt.Sprintf(pick(askLines), ware().Name), pick(byeLines)}
	}

	return []string{fmt.Sprintf(pick(askLines), ware().Name), fmt.Sprintf(pick(offerLines), ware().Price*2)}
}

func fill(line, lore string) string {
	if strings.Contains(line, "%s") {
		return fmt.Sprintf(line, lore)
	}

	return line
}

func cheapest(wares []game.Ware) game.Ware {
	var best game.Ware
	for _, ware := range wares {
		if best.ID == "" || ware.Price < best.Price {
			best = ware
		}
	}

	return best
}

// exchange plays one line of the player: the rules read it, decide, and
// the model words the keeper's reply, which is kept only if it passes.
func exchange(client *llm.Client, sink *sink, realm *game.Realm, hero, line string) (string, bool) {
	request, ok := realm.HaggleRequest(hero, line)
	if !ok {
		return "", false
	}

	intent := request.Sanitize(game.Read(request).Alone())

	_, narrate, ok := realm.Haggle(hero, line, intent)
	if !ok {
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), turnTimeout)
	defer cancel()

	reply, err := game.Narrate(ctx, client, narrate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   odrzucone: %v\n", err)

		return "", false
	}

	realm.HaggleRemember(hero, line, reply)
	sink.write(narrate, reply)

	return reply, true
}

// brawl walks a hero with a hired companion into the nearest monsters and
// lets the fight play out on its own, so the master has rounds to tell.
func brawl(rng *rand.Rand, realm *game.Realm, built *world.World) {
	hero := realm.Join(heroNames[rng.Intn(len(heroNames))], avatarIDs[rng.Intn(len(avatarIDs))])
	realm.Execute(hero, game.Parse("autowalka"))

	if len(built.Companions) > 0 {
		realm.Execute(hero, game.Parse("najmij "+built.Companions[0].Name))
	}

	start := built.Start()
	target, best := start, 1<<30

	for index, tile := range built.Tiles {
		point := world.Point{X: index % world.Width, Y: index / world.Width}
		if d := abs(point.X-start.X) + abs(point.Y-start.Y); len(tile.Monsters) > 0 && tile.Site < 0 && d < best {
			target, best = point, d
		}
	}

	for range world.Width + world.Height {
		view := realm.View(hero)
		if view.Fight != nil {
			break
		}

		switch pos := view.Me.Pos; {
		case pos.X < target.X:
			realm.Move(hero, 1, 0)
		case pos.X > target.X:
			realm.Move(hero, -1, 0)
		case pos.Y < target.Y:
			realm.Move(hero, 0, 1)
		default:
			realm.Move(hero, 0, -1)
		}
	}

	for range 40 {
		view := realm.View(hero)
		if view.Fight == nil {
			return
		}

		if view.Fight.Actor != nil {
			realm.Attack(hero, rng.Intn(max(1, len(view.Fight.Monsters))))
		}

		// Let the master catch up between rounds.
		time.Sleep(15 * time.Second)
	}
}

type sink struct {
	mu   sync.Mutex
	path string
	n    int
}

func newSink(path string) *sink {
	return &sink{path: path}
}

// write appends one exchange, in the shape the game itself records.
func (s *sink) write(request game.NarrateRequest, reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	line, err := json.Marshal(map[string]any{
		"time":     time.Now().Format(time.RFC3339),
		"keeper":   request.Merchant.Name,
		"outcome":  request.Decision.Outcome,
		"source":   "synth",
		"messages": llm.Transcript(request, reply),
	})
	if err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err := file.Write(append(line, '\n')); err == nil {
		s.n++
	}
}

func (s *sink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.n
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
