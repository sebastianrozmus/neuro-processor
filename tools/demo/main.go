// Command demo records a tech demo of the game: it drives the screens the
// way a player would, without a terminal or a model, and writes every
// screen as a frame with how long it should stay. tools/demo/render.py
// turns the frames into a video.
//
//	go run ./tools/demo -out /tmp/demo-frames
//	python3 tools/demo/render.py /tmp/demo-frames demo.mp4
//
// The "generated" world of the last part comes from tools/demo/world.json,
// so the demo does not depend on a language model.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/inspect"
	"github.com/sebastianrozmus/neuro-processor/internal/ui"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

//go:embed world.json
var generatedWorld []byte

const (
	columns = 140
	rows    = 44
)

// recorder keeps the frames of the demo.
type recorder struct {
	dir   string
	index int
	list  []string
}

func (r *recorder) frame(view string, seconds float64) {
	name := fmt.Sprintf("frame-%04d.txt", r.index)
	r.index++

	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(view), 0o644); err != nil {
		panic(err)
	}

	r.list = append(r.list, fmt.Sprintf("%s %.2f", name, seconds))
}

// picture records a ready PNG (a browser screenshot) as a frame.
func (r *recorder) picture(path string, seconds float64) {
	name := fmt.Sprintf("frame-%04d.png", r.index)
	r.index++

	if err := os.Rename(path, filepath.Join(r.dir, name)); err != nil {
		panic(err)
	}

	r.list = append(r.list, fmt.Sprintf("%s %.2f", name, seconds))
}

// shot records the screen of a model.
func (r *recorder) shot(model tea.Model, seconds float64) {
	r.frame(model.View(), seconds)
}

// play records the blow animation of a game screen, frame by frame.
func (r *recorder) play(model tea.Model, hold float64) {
	screen, ok := model.(ui.Game)
	if !ok {
		r.shot(model, hold)

		return
	}

	frames := screen.Frames()
	for index, frame := range frames {
		seconds := 0.14
		if index == len(frames)-1 {
			seconds = hold
		}

		r.frame(frame, seconds)
	}
}

// card draws a title screen.
func (r *recorder) card(title, subtitle string, seconds float64) {
	block := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render(title) + "\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(subtitle)
	r.frame(lipgloss.Place(columns, rows, lipgloss.Center, lipgloss.Center, block), seconds)
}

func (r *recorder) save() {
	if err := os.WriteFile(filepath.Join(r.dir, "frames.txt"), []byte(strings.Join(r.list, "\n")+"\n"), 0o644); err != nil {
		panic(err)
	}
}

// demoBrain answers like a keeper without any model: rules read the
// player and canned lines, in a mood, word the keeper's answer.
type demoBrain struct{}

func (demoBrain) Interpret(_ context.Context, request game.InterpretRequest) (game.Intent, error) {
	return game.Read(request).Alone(), nil
}

func (demoBrain) Narrate(_ context.Context, request game.NarrateRequest) (string, error) {
	d := request.Decision

	switch d.Outcome {
	case game.OutcomeInfo:
		return fmt.Sprintf("[spokój] %s? Dobra rzecz, choć widziała lepsze czasy. %s Za %d %s jest twoja.", d.Item.Name, d.Item.Description, d.Price, d.Currency), nil
	case game.OutcomeCounter:
		return fmt.Sprintf("[kpina] %d? Za tyle to ci mogę opowiedzieć, jak wygląda. Niech będzie %d %s i ani grosza mniej.", d.Offer, d.Price, d.Currency), nil
	case game.OutcomeSold:
		return fmt.Sprintf("[radość] Umowa. %d %s na ladę i noś na zdrowie. Wracaj, jak zostaniesz przy życiu.", d.Price, d.Currency), nil
	case game.OutcomeChat:
		return "[spokój] Co słychać? Woda się podnosi, ceny też. Jak idziesz na północ, to nie sam i nie po zmroku.", nil
	case game.OutcomeFarewell:
		return "[czułość] Bywaj. Wracaj z całymi kośćmi.", nil
	}

	return request.Decision.Fallback(), nil
}

// launcher starts games the demo's menu picks.
type launcher struct {
	brain   game.Brain
	library *avatars.Library
	last    *game.Realm
}

func (l *launcher) New(spec world.Spec) (tea.Model, tea.Cmd, error) {
	built, err := world.Build(spec)
	if err != nil {
		return nil, nil, err
	}

	realm := game.NewRealm(built, 7)
	l.last = realm
	name := realm.Join("Seba", "wojownik")

	return ui.NewGame(realm, name, l.brain, l.library, nil, nil, nil, nil), nil, nil
}

func (l *launcher) Load(state game.State) (tea.Model, tea.Cmd, error) {
	return nil, nil, fmt.Errorf("the demo loads nothing")
}

// generator hands the embedded world over as if a model had designed it.
type generator struct{}

func (generator) GenerateWorld(_ context.Context, mode world.Mode, theme string) (world.Spec, error) {
	var spec world.Spec
	if err := json.Unmarshal(generatedWorld, &spec); err != nil {
		return world.Spec{}, err
	}

	spec.Mode = mode

	return spec, spec.Normalize()
}

func main() {
	out := flag.String("out", "demo-frames", "folder for the frames")
	only := flag.String("only", "", "record one part alone: imagination, tabletop, adventure, generated or inspector")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}

	presets, err := world.Presets()
	if err != nil {
		panic(err)
	}

	byMode := map[world.Mode]world.Spec{}
	for _, spec := range presets {
		if _, taken := byMode[spec.Mode]; !taken {
			byMode[spec.Mode] = spec
		}
	}

	rec := &recorder{dir: *out}
	library := avatars.Open("")
	launch := &launcher{brain: demoBrain{}, library: library}

	parts := map[string]func(){
		"imagination": func() { imagination(rec, launch, byMode[world.ModeImagination]) },
		"tabletop":    func() { tabletop(rec, launch, byMode[world.ModeDnD]) },
		"adventure":   func() { adventure(rec, launch, byMode[world.ModeAdventure]) },
		"generated":   func() { generated(rec, launch, presets) },
		"inspector":   func() { inspector(rec, launch, byMode[world.ModeImagination]) },
	}

	if *only != "" {
		part, ok := parts[*only]
		if !ok {
			panic("no such part: " + *only)
		}

		part()
		rec.save()
		fmt.Printf("%d klatek w %s\n", rec.index, *out)

		return
	}

	rec.card("NEURO PROCESSOR", "Generator gier tekstowych w terminalu. Trzy rodzaje gry, jeden silnik.", 3)

	for _, name := range []string{"imagination", "tabletop", "adventure", "generated", "inspector"} {
		parts[name]()
	}

	rec.card("NEURO PROCESSOR", "Mapa, walka, handel z żywymi postaciami, drużyna, wiedza, która krąży, świat, który się zmienia, i podgląd tego wszystkiego w przeglądarce.", 4)
	rec.save()
	fmt.Printf("%d klatek w %s\n", rec.index, *out)
}

// size gives a model the demo's terminal.
func size(model tea.Model) tea.Model {
	model, _ = model.Update(tea.WindowSizeMsg{Width: columns, Height: rows})

	return model
}

func press(model tea.Model, keys ...tea.KeyType) tea.Model {
	for _, key := range keys {
		model, _ = model.Update(tea.KeyMsg{Type: key})
	}

	return model
}

// typed shows a command being typed, then runs it.
func typed(rec *recorder, model tea.Model, text string, hold float64) tea.Model {
	if text != "" {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
		rec.shot(model, 0.9)
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	for cmd != nil {
		screen, ok := model.(ui.Game)
		if !ok || !screen.Waiting() {
			break
		}

		model, cmd = model.Update(cmd())
	}

	rec.play(model, hold)

	return model
}

// walkToFight walks the hero towards the nearest monsters, recording the
// way, and returns once a fight is on.
func walkToFight(rec *recorder, model tea.Model, realm *game.Realm, name string) tea.Model {
	built := realm.World()
	start := realm.View(name).Me.Pos
	target, best := start, 1<<30

	for index, tile := range built.Tiles {
		point := world.Point{X: index % world.Width, Y: index / world.Width}
		distance := abs(point.X-start.X) + abs(point.Y-start.Y)

		if len(tile.Monsters) > 0 && tile.Site < 0 && distance < best {
			target, best = point, distance
		}
	}

	for range world.Width + world.Height {
		view := realm.View(name)
		if view.Fight != nil {
			return model
		}

		var command string

		switch pos := view.Me.Pos; {
		case pos.X < target.X:
			command = "idź na wschód"
		case pos.X > target.X:
			command = "idź na zachód"
		case pos.Y < target.Y:
			command = "idź na południe"
		default:
			command = "idź na północ"
		}

		model = typed(rec, model, command, 1.1)
	}

	return model
}

func imagination(rec *recorder, launch *launcher, spec world.Spec) {
	rec.card("1. Gra wyobraźni", spec.Name+": mapa, osady, kupcy, drużyna i walka opowiadana przez narratora.", 3)

	model, _, _ := launch.New(spec)
	model = size(model)
	name := model.(ui.Game).Name()
	realm := launch.last

	rec.shot(model, 3)
	model = typed(rec, model, "rozejrzyj się", 2.2)
	model = press(model, tea.KeyTab)
	rec.shot(model, 2.4)
	model = press(model, tea.KeyEsc)

	model = typed(rec, model, "idź do zbrojowni", 2)
	model = typed(rec, model, "co słychać?", 2.8)
	model = typed(rec, model, "ile za kord?", 2.6)
	model = typed(rec, model, "dam 30", 2.6)
	model = typed(rec, model, "biorę", 2.6)
	model = press(model, tea.KeyEsc)
	rec.shot(model, 1.2)
	model = typed(rec, model, "załóż kord", 1.4)

	model = walkToFight(rec, model, realm, name)
	rec.shot(model, 2.2)

	for round := 0; round < 6 && realm.View(name).Fight != nil; round++ {
		model = typed(rec, model, "", 1.2)
	}

	rec.shot(model, 2)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF4})
	rec.shot(model, 2.4)
	model = press(model, tea.KeyEsc)
	model = typed(rec, model, "dziennik", 2.2)
}

func tabletop(rec *recorder, launch *launcher, spec world.Spec) {
	rec.card("2. D&D", spec.Name+": k20 przeciw klasie pancerza, inicjatywa, rzuty obronne i pole bitwy z ruchem i atakami okazyjnymi.", 3.5)

	model, _, _ := launch.New(spec)
	model = size(model)
	name := model.(ui.Game).Name()
	realm := launch.last

	rec.shot(model, 2.2)
	model = typed(rec, model, "najmij "+firstName(spec.Companions[0].Name), 1.6)
	model = walkToFight(rec, model, realm, name)
	rec.shot(model, 2.4)

	for turn := 0; turn < 14 && realm.View(name).Fight != nil; turn++ {
		view := realm.View(name)
		if view.Fight.Actor == nil || view.Fight.Actor.Companion >= 0 {
			model = typed(rec, model, "", 1)

			continue
		}

		// Walk up to the nearest monster, showing the reach, then strike.
		model = press(model, tea.KeyUp, tea.KeyEnter)
		rec.shot(model, 1.4)

		for steps := 0; steps < 4; steps++ {
			dx, dy, near := towards(view, name)
			if near {
				break
			}

			model = press(model, keyFor(dx, dy))
			rec.shot(model, 0.5)
			view = realm.View(name)
		}

		model = press(model, tea.KeyEsc)
		model = typed(rec, model, "", 1.2)
	}

	rec.shot(model, 2.2)
}

func adventure(rec *recorder, launch *launcher, spec world.Spec) {
	rec.card("3. Przygodowa", spec.Name+": pomieszczenia, rzeczy, zagadki i ludzie, jak w Mózgoprocesorze.", 3)

	model, _, _ := launch.New(spec)
	model = size(model)

	rec.shot(model, 3)
	for _, command := range []string{"rozejrzyj się", "zbadaj łóżko", "podnieś żeton", "zbadaj kartę pacjenta", "idź na zachód", "otwórz kratkę", "idź na północ", "idź na wschód", "rozmawiaj z grabskim", "zbadaj biurko", "podnieś śrubokręt"} {
		model = typed(rec, model, command, 2)
	}

	model = press(model, tea.KeyTab)
	rec.shot(model, 2.2)
	model = press(model, tea.KeyEsc)
}

func generated(rec *recorder, launch *launcher, presets []world.Spec) {
	rec.card("4. Generowanie świata", "Opisujesz temat, a model projektuje świat; mapę, zasady i ceny wylicza gra.", 3)

	menu := size(ui.NewMenu(nil, presets, nil, generator{}, os.TempDir(), "demo", launch))
	rec.shot(menu, 2.4)

	for range presets {
		menu = press(menu, tea.KeyDown)
	}

	rec.shot(menu, 1.2)
	menu = press(menu, tea.KeyEnter)
	rec.shot(menu, 2.4)
	menu = press(menu, tea.KeyEnter)
	rec.shot(menu, 1.2)
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("wyspa piratów z przeklętym kapitanem i wulkanem")})
	rec.shot(menu, 2.2)

	model, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rec.shot(model, 2)
	model, _ = model.Update(cmd())
	model = size(model)
	name := model.(ui.Game).Name()
	realm := launch.last

	rec.shot(model, 3)
	model = typed(rec, model, "rozejrzyj się", 2.2)
	model = typed(rec, model, "idź do kramu", 2)
	model = typed(rec, model, "co słychać?", 2.6)
	model = press(model, tea.KeyEsc)
	model = typed(rec, model, "najmij bosmana", 1.8)
	model = walkToFight(rec, model, realm, name)
	rec.shot(model, 2)

	for round := 0; round < 5 && realm.View(name).Fight != nil; round++ {
		model = typed(rec, model, "", 1.1)
	}

	rec.shot(model, 2.4)
}

// inspector plays a little, then shows the same world in the browser
// inspector: the map, the knowledge graph, the bestiary and the items,
// as screenshots taken with a headless Chrome. Without Chrome the part
// is skipped.
func inspector(rec *recorder, launch *launcher, spec world.Spec) {
	chrome := chromePath()
	if chrome == "" {
		fmt.Fprintln(os.Stderr, "brak Chrome: pomijam podgląd w przeglądarce")

		return
	}

	rec.card("5. Podgląd w przeglądarce", "Ten sam świat na żywo: mapa, graf wiedzy, bestiariusz, przedmioty. Do czytania i poprawiania.", 3.5)

	model, _, _ := launch.New(spec)
	model = size(model)
	name := model.(ui.Game).Name()
	realm := launch.last

	model = typed(rec, model, "idź do zbrojowni", 1.6)
	model = typed(rec, model, "co słychać?", 2.2)
	model = press(model, tea.KeyEsc)
	model = walkToFight(rec, model, realm, name)

	for round := 0; round < 4 && realm.View(name).Fight != nil; round++ {
		model = typed(rec, model, "", 0.9)
	}

	rec.shot(model, 1.6)

	server, err := inspect.Start("127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	server.Watch(realm, nil)

	for _, tab := range []struct {
		name    string
		seconds float64
	}{{"world", 4}, {"graph", 4}, {"facts", 3.5}, {"beasts", 3.5}, {"items", 3}, {"people", 3}} {
		shot := filepath.Join(rec.dir, "browser-"+tab.name+".png")
		command := exec.Command(chrome, "--headless=new", "--disable-gpu", "--hide-scrollbars", "--force-device-scale-factor=1",
			fmt.Sprintf("--window-size=%d,%d", columns*9, rows*18), "--virtual-time-budget=6000", "--screenshot="+shot, server.URL()+"#"+tab.name)
		command.Stderr = nil

		if err := command.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "zrzut z Chrome nie wyszedł:", err)

			continue
		}

		rec.picture(shot, tab.seconds)
		time.Sleep(100 * time.Millisecond)
	}
}

// chromePath finds a Chrome to take screenshots with: NEURO_CHROME, or the
// usual places.
func chromePath() string {
	candidates := []string{
		os.Getenv("NEURO_CHROME"),
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}

	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, path)
		}
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return ""
}

// towards says which way the hero's piece should step to reach the
// nearest monster, and whether it already stands next to one.
func towards(view game.View, name string) (int, int, bool) {
	var me, foe *game.PieceView
	for index := range view.Fight.Board.Pieces {
		piece := &view.Fight.Board.Pieces[index]

		switch {
		case piece.Current && piece.Mine:
			me = piece
		case piece.Foe && (foe == nil || abs(piece.X-me.X) < abs(foe.X-me.X)):
			foe = piece
		}
	}

	if me == nil || foe == nil {
		return 0, 0, true
	}

	if abs(me.X-foe.X) <= 1 && abs(me.Y-foe.Y) <= 1 {
		return 0, 0, true
	}

	switch {
	case foe.X > me.X:
		return 1, 0, false
	case foe.X < me.X:
		return -1, 0, false
	case foe.Y > me.Y:
		return 0, 1, false
	}

	return 0, -1, false
}

func keyFor(dx, dy int) tea.KeyType {
	switch {
	case dx > 0:
		return tea.KeyRight
	case dx < 0:
		return tea.KeyLeft
	case dy > 0:
		return tea.KeyDown
	}

	return tea.KeyUp
}

func firstName(name string) string {
	return strings.Fields(name)[0]
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
