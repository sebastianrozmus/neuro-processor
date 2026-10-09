package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/gallery"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/inline"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

var sizes = [][2]int{{MinWidth, MinHeight}, {150, 46}}

func preset(t *testing.T, index int) world.Spec {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	return presets[index]
}

func newRealm(t *testing.T, index int) *game.Realm {
	t.Helper()

	built, err := world.Build(preset(t, index))
	if err != nil {
		t.Fatal(err)
	}

	return game.NewRealm(built, 1)
}

func newGame(t *testing.T, realm *game.Realm, name string, size [2]int, save Save) tea.Model {
	t.Helper()

	model, _ := NewGame(realm, realm.Join(name, "wojownik"), llm.Mock{}, avatars.Open(""), save, nil, nil, nil).Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})

	return model
}

// press sends keys one by one.
func press(model tea.Model, keys ...tea.KeyType) tea.Model {
	for _, key := range keys {
		model, _ = model.Update(tea.KeyMsg{Type: key})
	}

	return model
}

// enter types a line and presses Enter, running the commands it triggers.
func enter(model tea.Model, text string) tea.Model {
	if text != "" {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Shop talks answer through commands; the world listener would block.
	for cmd != nil && model.(Game).shop.waiting {
		model, cmd = model.Update(cmd())
	}

	return model
}

func assertFits(t *testing.T, model tea.Model, size [2]int, when string) {
	t.Helper()

	view := model.View()
	if width, height := lipgloss.Width(view), lipgloss.Height(view); width != size[0] || height != size[1] {
		t.Errorf("%s: view is %dx%d, want %dx%d", when, width, height, size[0], size[1])
	}
}

func assertShows(t *testing.T, model tea.Model, when string, wanted ...string) {
	t.Helper()

	view := model.View()
	for _, want := range wanted {
		if !strings.Contains(view, want) {
			t.Errorf("%s: screen misses %q", when, want)
		}
	}
}

// walkIntoFight types directions towards the closest monsters until a
// fight starts.
func walkIntoFight(t *testing.T, model tea.Model) tea.Model {
	t.Helper()

	built := model.(Game).view.World
	start := built.Start()
	target, best := start, 1<<30

	for index, tile := range built.Tiles {
		point := world.Point{X: index % world.Width, Y: index / world.Width}
		distance := abs(point.X-start.X) + abs(point.Y-start.Y)

		if len(tile.Monsters) > 0 && tile.Site < 0 && distance < best {
			target, best = point, distance
		}
	}

	for range world.Width + world.Height {
		current := model.(Game).view
		if current.Fight != nil {
			return model
		}

		switch pos := current.Me.Pos; {
		case pos.X < target.X:
			model = enter(model, "idź na wschód")
		case pos.X > target.X:
			model = enter(model, "zachód")
		case pos.Y < target.Y:
			model = enter(model, "s")
		default:
			model = enter(model, "idź na północ")
		}
	}

	t.Fatal("walked across the map without meeting a monster")

	return nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

func TestExploreAndFightByCommands(t *testing.T) {
	for _, size := range sizes {
		model := newGame(t, newRealm(t, 0), "Seba", size, nil)

		assertShows(t, model, "start", "Zatopione Krypty", "Szary Bród", "Seba", "Zbrojownia (Borzywoj)", "@", "wpisz polecenie",
			"Poziom 1", "HP  ", " 42/42", "MANA", " 10/10", "EXP ", " 0/40", "F1 Seba")
		assertFits(t, model, size, "start")

		model = enter(model, "rozejrzyj się")
		assertShows(t, model, "look", "> rozejrzyj się", "Drogi: północ:")

		model = walkIntoFight(t, model)
		assertShows(t, model, "fight", "Walka, runda 1", "Walka! Przeciwnicy:", "Atak", "Moc", "Przedmiot", "Ucieczka", "Enter dalej")
		assertFits(t, model, size, "fight")

		model = press(model, tea.KeyTab)
		assertShows(t, model, "sheet", "Plecak (0)", "Pusty.", "Moce (0)", "Cechy")
		assertFits(t, model, size, "bag")
		model = press(model, tea.KeyEsc)

		for range 200 {
			if model.(Game).view.Fight == nil {
				break
			}

			model = enter(model, "")
		}

		state := model.(Game).view
		if state.Fight != nil {
			t.Fatalf("%v: the fight never ended", size)
		}

		if state.Tile.Town < 0 && state.Me.Gold <= 60 {
			t.Errorf("%v: won without loot: %+v", size, state.Me)
		}

		assertFits(t, model, size, "after the fight")
	}
}

func TestFightMenu(t *testing.T) {
	model := newGame(t, newRealm(t, 0), "Seba", sizes[1], nil)

	// Spend the starting gold on a spell and learn it.
	model = enter(model, "idź do kramu morii")
	model = enter(model, "biorę zwój iskry")
	model = press(model, tea.KeyEsc)
	model = enter(model, "naucz się iskry")

	if spells := model.(Game).view.Me.Spells; len(spells) != 1 {
		t.Fatalf("spells = %+v", spells)
	}

	model = press(model, tea.KeyTab)
	assertShows(t, model, "sheet", "Moce (1)", "Zwój Błędnej Iskry", "Rani jednego wroga za 10, koszt 4 mocy.",
		"Cechy", "SIŁ 15 (+2)", "INT  8 (-1)", "Walka", "Dzień 1 w tym świecie, wiosna, pogodnie.")
	assertFits(t, model, sizes[1], "sheet")
	model = press(model, tea.KeyEsc)

	model = walkIntoFight(t, model)
	game := model.(Game)

	if want := map[bool]int{true: stepTarget, false: stepAction}[len(game.view.Fight.Monsters) > 1]; game.step != want {
		t.Fatalf("menu starts at step %d, want %d", game.step, want)
	}

	if game.step == stepTarget {
		model = press(model, tea.KeyEnter)
	}

	// Moc -> the list of spells -> cast.
	model = press(model, tea.KeyDown, tea.KeyEnter)
	assertShows(t, model, "spell list", "Zwój Błędnej Iskry (4)", "Rani jednego wroga za 10, koszt 4 mocy.", "Enter użyj")
	assertFits(t, model, sizes[1], "spell list")

	before := model.(Game).view.Fight.Monsters[0].HP
	model = press(model, tea.KeyEnter)
	game = model.(Game)

	if game.view.Me.Mana >= 12 || game.step != stepAction {
		t.Errorf("after casting: mana %d, step %d", game.view.Me.Mana, game.step)
	}

	if fight := game.view.Fight; fight != nil && len(fight.Monsters) > 0 && fight.Monsters[0].HP > before-10 && fight.Monsters[0].Name != "" && before > 12 {
		t.Errorf("the spell did not hurt: %d -> %d", before, fight.Monsters[0].HP)
	}

	if !strings.Contains(strings.Join(game.view.Log, "\n"), "Seba rzuca: Zwój Błędnej Iskry.") {
		t.Errorf("no cast in the log: %v", game.view.Log)
	}

	// Back out of the list, pick a plain attack and keep hitting Enter.
	if game.view.Fight != nil {
		model = press(model, tea.KeyEnter, tea.KeyLeft, tea.KeyUp)
		if got := model.(Game); got.view.Fight != nil && (got.step != stepAction || got.action != actAttack) {
			t.Errorf("menu at step %d action %d, want the attack", got.step, got.action)
		}
	}

	// With no potions the item list has nothing to open.
	if model.(Game).view.Fight != nil {
		model = press(model, tea.KeyDown, tea.KeyDown, tea.KeyEnter)
		if got := model.(Game); got.step != stepAction {
			t.Errorf("an empty item list opened: step %d", got.step)
		}

		assertShows(t, model, "no items", "nie ma przy sobie niczego")
		model = press(model, tea.KeyUp, tea.KeyUp)
	}

	// Typing still works in a fight: letters go to the command line.
	if model.(Game).view.Fight != nil {
		model = enter(model, "rozejrzyj się")
		if !strings.Contains(strings.Join(model.(Game).view.Log, "\n"), "> rozejrzyj się") {
			t.Error("a typed command was swallowed by the fight menu")
		}
	}

	for range 200 {
		if model.(Game).view.Fight == nil {
			break
		}

		model = press(model, tea.KeyEnter)
	}

	if model.(Game).view.Fight != nil {
		t.Fatal("the fight never ended")
	}

	if me := model.(Game).view.Me; me.XP == 0 && me.Level == 1 && model.(Game).view.Tile.Town < 0 {
		t.Errorf("no experience after a won fight: %+v", me)
	}
}

func TestArrowsMoveAndEmptyEnterRepeats(t *testing.T) {
	model := newGame(t, newRealm(t, 2), "Seba", sizes[0], nil)
	start := model.(Game).view.Me.Pos

	model = press(model, tea.KeyDown)
	if got := model.(Game).view.Me.Pos; got.Y != start.Y+1 && model.(Game).view.Fight == nil {
		t.Errorf("down arrow: %v -> %v", start, got)
	}

	realm := newRealm(t, 2)
	model = newGame(t, realm, "Seba", sizes[0], nil)
	model = enter(model, "rozejrzyj się")
	before := len(model.(Game).view.Log)
	model = enter(model, "")

	if log := model.(Game).view.Log; len(log) <= before || log[before] != "> rozejrzyj się" {
		t.Errorf("empty Enter did not repeat the last command: %v", log[before:])
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	model = press(model, tea.KeyLeft)

	if got := model.(Game).view.Me.Pos; got != start {
		t.Errorf("left arrow moved the hero while typing: %v", got)
	}

	model = press(model, tea.KeyEsc)
	if got := model.(Game).input.Value(); got != "" {
		t.Errorf("Esc left %q in the command line", got)
	}
}

func TestShopBuyAndEquip(t *testing.T) {
	for _, size := range sizes {
		model := newGame(t, newRealm(t, 0), "Seba", size, nil)

		model = enter(model, "idź do kramu morii")
		assertShows(t, model, "magic shop", "Kram Morii", "Podejdź bliżej, wędrowcze", "Zwój Błędnej Iskry")
		assertFits(t, model, size, "magic shop")

		model = press(model, tea.KeyEsc)
		model = enter(model, "zbrojownia")
		assertShows(t, model, "armory", "Zbrojownia", "Borzywoj", "Stal sama się nie sprzeda")
		assertFits(t, model, size, "armory")

		if strings.Contains(model.View(), "Miecz Runiczny") {
			t.Errorf("%v: the first settlement sells the best sword", size)
		}

		model = enter(model, "dam 20 za kord")
		assertShows(t, model, "counter offer", "Twoja oferta: 20", "●●●●●○")

		model = enter(model, "dam 30 za kord")
		assertShows(t, model, "purchase", "Kupiono: Zardzewiały Kord za 30 szt. złota.")
		assertFits(t, model, size, "shop after a purchase")

		model = press(model, tea.KeyEsc, tea.KeyTab)
		assertShows(t, model, "sheet", "Zardzewiały Kord", "atak +3")

		model = press(model, tea.KeyEnter, tea.KeyEsc)

		me := model.(Game).view.Me
		// A warrior's strength adds two to every blow.
		if me.Weapon != "Zardzewiały Kord" || me.Gold != 30 || me.Attack != 8 {
			t.Errorf("%v: after equipping: %+v", size, me)
		}

		assertShows(t, model, "hero panel", "Zardzewiały Kord", "30 szt. złota")

		model = enter(model, "pogadaj z borzywojem")
		if got := strings.Count(model.View(), "Stal sama się nie sprzeda"); got != 1 {
			t.Errorf("%v: reopening the shop shows the greeting %d times", size, got)
		}
	}
}

func TestSaveCommand(t *testing.T) {
	saved := 0
	model := newGame(t, newRealm(t, 0), "Seba", sizes[1], func() (string, error) {
		saved++

		return "/tmp/zapis.json", nil
	})

	model = enter(model, "zapisz")
	if saved != 1 {
		t.Fatalf("save called %d times", saved)
	}

	assertShows(t, model, "save", "Gra zapisana: /tmp/zapis.json")

	model = newGame(t, newRealm(t, 0), "Gość", sizes[1], nil)
	assertShows(t, enter(model, "zapisz"), "guest save", "tylko gospodarz")

	model = newGame(t, newRealm(t, 0), "Seba", sizes[1], func() (string, error) { return "", errors.New("dysk pełny") })
	assertShows(t, enter(model, "zapisz"), "failed save", "dysk pełny")

	assertShows(t, enter(model, "pomoc"), "help", "> pomoc", "Polecenia: idź na północ")

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("wyjdź")})
	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("the quit command did not quit")
	}
}

func TestTwoPlayersSeeEachOther(t *testing.T) {
	realm := newRealm(t, 1)
	seba := newGame(t, realm, "Seba", sizes[0], nil)
	bob := newGame(t, realm, "Bob", sizes[0], nil)

	bob = enter(bob, "powiedz idę na południe")
	bob = press(bob, tea.KeyDown)
	seba, _ = seba.Update(worldMsg{})

	assertShows(t, seba, "second player", "Bob", "Inni gracze", "Bob: «idę na południe»")

	if seba.(Game).view.Others[0].Pos == seba.(Game).view.Me.Pos {
		t.Error("bob did not move")
	}

	assertFits(t, bob, sizes[0], "second player")
}

func TestTooSmallTerminal(t *testing.T) {
	model := newGame(t, newRealm(t, 0), "Seba", [2]int{80, 24}, nil)

	assertShows(t, model, "small terminal", "Powiększ okno")
}

func TestScenes(t *testing.T) {
	for index := range 3 {
		realm := newRealm(t, index)
		model := newGame(t, realm, "Seba", sizes[1], nil)
		view := model.(Game).view

		town := scene(view, 0, nil, 0, false, false)
		if got := lipgloss.Height(town); got != sceneRows || lipgloss.Width(town) != sceneWidth {
			t.Errorf("world %d: scene is %dx%d", index, lipgloss.Width(town), got)
		}

		model = walkIntoFight(t, model)
		if fight := scene(model.(Game).view, 0, nil, 0, false, false); fight == town || lipgloss.Height(fight) != sceneRows {
			t.Errorf("world %d: the fight scene shows no monsters", index)
		}

		for _, monster := range view.World.Monsters {
			for _, boss := range []bool{false, true} {
				if _, ok := loadSprite(game.MonsterView{Name: monster.Name, Shape: monster.Shape, Boss: boss}); !ok {
					t.Errorf("world %d: no picture for %s", index, monster.Name)
				}
			}
		}

		for _, place := range view.World.Places {
			for variant := 1; variant <= sceneVariants; variant++ {
				if _, ok := loadScene(place.Terrain, variant); !ok {
					t.Errorf("world %d: no picture for %s", index, place.Terrain)
				}
			}
		}
	}
}

func TestSetup(t *testing.T) {
	var gotName, gotAvatar string

	realm := newRealm(t, 0)
	model, _ := NewSetup("Zatopione Krypty", "seba", avatars.Open(""), nil, func(name, avatar string) (tea.Model, tea.Cmd) {
		gotName, gotAvatar = name, avatar

		return NewGame(realm, realm.Join(name, avatar), llm.Mock{}, avatars.Open(""), nil, nil, nil, nil), nil
	}).Update(tea.WindowSizeMsg{Width: sizes[0][0], Height: sizes[0][1]})

	assertShows(t, model, "setup", "Zatopione Krypty", "Wojownik", "1 z 6", "seba")

	// Past the last avatar comes the entry for drawing a new one; without a
	// generator it only explains what is missing.
	model = press(model, tea.KeyUp, tea.KeyEnter)
	assertShows(t, model, "no generator", "Dodaj nowy", "tools/avatar/setup.sh")
	model = press(model, tea.KeyDown)
	assertFits(t, model, sizes[0], "setup")

	model = press(model, tea.KeyDown, tea.KeyDown, tea.KeyUp, tea.KeyTab)
	assertShows(t, model, "avatar picker", "Łotrzyk", "3 z 6")

	model = press(model, tea.KeyBackspace, tea.KeyBackspace, tea.KeyBackspace, tea.KeyBackspace)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Sebastian")})
	model = press(model, tea.KeyEnter)

	if gotName != "Sebastian" || gotAvatar != "lotrzyk" {
		t.Errorf("entered as %q with %q", gotName, gotAvatar)
	}

	if _, ok := model.(Game); !ok {
		t.Fatal("the game did not take over the screen")
	}

	assertShows(t, model, "game after setup", "Sebastian", "Szary Bród")
	assertFits(t, model, sizes[0], "game after setup")
}

type stubGenerator struct {
	spec world.Spec
	err  error
	mode world.Mode
}

func (g *stubGenerator) GenerateWorld(_ context.Context, mode world.Mode, _ string) (world.Spec, error) {
	g.mode = mode

	return g.spec, g.err
}

type stubLauncher struct {
	started []string
	loaded  []string
}

func (l *stubLauncher) New(spec world.Spec) (tea.Model, tea.Cmd, error) {
	l.started = append(l.started, spec.Name)

	return NewSetup(spec.Name, "seba", avatars.Open(""), nil, nil), nil, nil
}

func (l *stubLauncher) Load(state game.State) (tea.Model, tea.Cmd, error) {
	l.loaded = append(l.loaded, state.Spec.Name+"/"+state.Host)

	return nil, nil, errors.New("zepsuty zapis")
}

func newMenu(t *testing.T, saves []game.State, generator Generator) (tea.Model, *stubLauncher, string) {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	dir, launcher := t.TempDir(), &stubLauncher{}
	model, _ := NewMenu(saves, presets, nil, generator, dir, "test", launcher).Update(tea.WindowSizeMsg{Width: sizes[0][0], Height: sizes[0][1]})

	return model, launcher, dir
}

func TestMenuStartsPresetOverBackdrop(t *testing.T) {
	model, launcher, _ := newMenu(t, nil, nil)

	assertShows(t, model, "menu", "NEURO PROCESSOR", "Zatopione Krypty", "Popiół i Rdza", "Stacja Erebus", "Wygeneruj nowy świat", "Kraina wokół osady")
	assertFits(t, model, sizes[0], "menu")

	first := model.View()
	model, _ = model.Update(slideMsg{})

	if model.View() == first {
		t.Error("the backdrop did not change with the slide")
	}

	model = press(model, tea.KeyDown)
	assertShows(t, model, "second world", "Trzydzieści lat po Wielkim Błysku")
	assertFits(t, model, sizes[0], "menu on the second world")

	model = press(model, tea.KeyEnter)

	if len(launcher.started) != 1 || launcher.started[0] != "Popiół i Rdza" {
		t.Fatalf("started = %v", launcher.started)
	}

	if _, ok := model.(Setup); !ok {
		t.Error("the setup screen did not take over")
	}

	assertFits(t, model, sizes[0], "setup launched from the menu")
}

func TestMenuLoadsSave(t *testing.T) {
	realm := newRealm(t, 2)
	state := realm.State(realm.Join("Seba", "pilot"))

	model, launcher, _ := newMenu(t, []game.State{state}, nil)
	assertShows(t, model, "menu with a save", "Wczytaj: Stacja Erebus", "(Seba, ")

	model = press(model, tea.KeyUp, tea.KeyEnter)

	if len(launcher.loaded) != 1 || launcher.loaded[0] != "Stacja Erebus/Seba" {
		t.Fatalf("loaded = %v", launcher.loaded)
	}

	assertShows(t, model, "failed load", "zepsuty zapis")
}

// toGenerate walks the menu cursor down to "Wygeneruj nowy świat…".
func toGenerate(t *testing.T, model tea.Model) tea.Model {
	t.Helper()

	specs, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	for range specs {
		model = press(model, tea.KeyDown)
	}

	return press(model, tea.KeyEnter)
}

func TestMenuGeneratesWorld(t *testing.T) {
	model, launcher, _ := newMenu(t, nil, nil)
	model = toGenerate(t, model)

	if len(launcher.started) != 0 {
		t.Error("generation started without a model")
	}

	assertShows(t, model, "no model", "NEURO_LLM_MODEL")

	spec := preset(t, 2)
	spec.Name, spec.Backdrops = "Nowy Świat", nil

	// The kind of game is chosen first; the generator is told which.
	generator := &stubGenerator{spec: spec}
	model, launcher, dir := newMenu(t, nil, generator)
	model = toGenerate(t, model)
	assertShows(t, model, "choosing", "Jaki to ma być rodzaj gry?", "Gra wyobraźni", "D&D", "Przygodowa")
	model = press(model, tea.KeyDown, tea.KeyDown, tea.KeyEnter)
	assertShows(t, model, "typing", "O czym ma być gra? (Przygodowa)")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("kosmos")})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assertShows(t, model, "generating", "Tworzę świat")

	model, _ = model.Update(cmd())

	if len(launcher.started) != 1 || launcher.started[0] != "Nowy Świat" || generator.mode != world.ModeAdventure {
		t.Fatalf("started = %v, mode %q", launcher.started, generator.mode)
	}

	if saved := world.SavedSpecs(dir); len(saved) != 1 || saved[0].Name != "Nowy Świat" {
		t.Errorf("the generated world was not saved: %v", saved)
	}

	model, _, _ = newMenu(t, nil, &stubGenerator{err: errors.New("boom")})
	model = press(toGenerate(t, model), tea.KeyEnter)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(cmd())

	assertShows(t, model, "failed generation", "boom")
}

func TestEveryKeeperHasAPortrait(t *testing.T) {
	specs := []world.Spec{preset(t, 0), preset(t, 1), preset(t, 2)}

	for _, terrain := range world.Terrains {
		specs = append(specs, world.Spec{Places: []world.PlaceSpec{{Terrain: terrain}}})
	}

	seen := map[string]bool{}

	for _, spec := range specs {
		for _, kind := range world.ShopKinds {
			portrait := keeperPortrait(spec, kind)
			if lipgloss.Width(portrait) != portraitWidth || lipgloss.Height(portrait) != portraitHeight {
				t.Errorf("%s %s: portrait is %dx%d", lookOf(spec), kind, lipgloss.Width(portrait), lipgloss.Height(portrait))
			}

			seen[portrait] = true
		}
	}

	if len(seen) != 9 {
		t.Errorf("%d different portraits, want 9", len(seen))
	}

	for index := range 3 {
		model := newGame(t, newRealm(t, index), "Seba", sizes[0], nil)

		for _, shop := range model.(Game).view.World.Shops {
			model = enter(model, "idź do "+shop.Name)
			if model.(Game).shop.portrait == "" || model.(Game).shop.merchant != shop.Keeper {
				t.Errorf("world %d: %s has no portrait on screen", index, shop.Keeper)
			}

			assertFits(t, model, sizes[0], shop.Name)
			model = press(model, tea.KeyEsc)
		}
	}
}

func TestBackdropsExistForEveryWorld(t *testing.T) {
	specs := []world.Spec{preset(t, 0), preset(t, 1), preset(t, 2)}

	for _, terrain := range world.Terrains {
		specs = append(specs, world.Spec{Places: []world.PlaceSpec{{Terrain: terrain}}})
	}

	for _, spec := range specs {
		for _, name := range backdropsFor(spec) {
			picture := loadBackdrop(name)
			if len(picture) < 30 || len(picture[0]) < 100 {
				t.Errorf("backdrop %q is missing or too small", name)
			}

			if over := picture.under("ab\ncd", 200, 70); lipgloss.Width(over) != 200 || lipgloss.Height(over) != 70 || !strings.Contains(over, "ab") {
				t.Errorf("backdrop %q does not fill a large screen", name)
			}
		}
	}
}

// inventingBrain talks like the mock but offers a helmet in every idle line.
type inventingBrain struct {
	llm.Mock
	spoken   []string
	recorded []string
}

func (b *inventingBrain) Invent(context.Context, game.InventRequest) (world.ItemSpec, bool, error) {
	return world.ItemSpec{Name: "Hełm z Rondem", Description: "Garnek z daszkiem.", Kind: world.KindArmor, Tier: 1}, true, nil
}

func (b *inventingBrain) Say(text string, mood speech.Emotion) {
	b.spoken = append(b.spoken, string(mood)+": "+text)
}

func TestInventedItemsAndSpeech(t *testing.T) {
	realm := newRealm(t, 0)
	brain := &inventingBrain{}
	model, _ := NewGame(realm, realm.Join("Seba", "wojownik"), brain, avatars.Open(""), nil, brain, nil, func(request game.NarrateRequest, reply string) {
		brain.recorded = append(brain.recorded, request.Merchant.Name+": "+reply)
	}).Update(tea.WindowSizeMsg{Width: sizes[1][0], Height: sizes[1][1]})

	model = enter(model, "idź do zbrojowni")
	if len(brain.spoken) != 1 || !strings.HasPrefix(brain.spoken[0], "spokój: Stal sama się nie sprzeda") {
		t.Fatalf("the greeting was not spoken: %q", brain.spoken)
	}

	// A deal invents nothing.
	model = enter(model, "dam 20 za kord")
	if len(brain.spoken) != 2 || strings.Contains(model.View(), "Do świata wpada") {
		t.Errorf("after a counter offer: spoken %q", brain.spoken)
	}

	// Idle talk does; the item then shows up in the keeper's list.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("co słychać?")})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	for cmd != nil {
		msg := cmd()
		model, cmd = model.Update(msg)

		if _, done := msg.(inventedMsg); done {
			break
		}
	}

	assertShows(t, model, "invented item", "Do świata wpada: Hełm z Rondem (35 szt. złota). Pancerz, obrona +1.", "Hełm z Rondem")

	if wares := model.(Game).view.Shop.Wares; wares[len(wares)-1].Name != "Hełm z Rondem" {
		t.Errorf("wares = %+v", wares)
	}

	if len(brain.spoken) != 3 || !strings.HasSuffix(brain.spoken[2], model.(Game).shop.lines[len(model.(Game).shop.lines)-2].text) {
		t.Errorf("the reply was not spoken: %q", brain.spoken)
	}

	assertFits(t, model, sizes[1], "shop with an invented item")

	if len(brain.recorded) != 2 || !strings.HasPrefix(brain.recorded[0], "Borzywoj: ") {
		t.Errorf("exchanges recorded: %q", brain.recorded)
	}
}

type stubPainter struct {
	prompts []string
	err     error
}

func (p *stubPainter) Generate(_ context.Context, prompt string) (avatars.Drawing, error) {
	p.prompts = append(p.prompts, prompt)
	small := avatars.Open("").Art("pilot")

	return avatars.Drawing{Small: small, Large: small + "\n" + small}, p.err
}

func TestSetupDrawsAndKeepsNewAvatars(t *testing.T) {
	library := avatars.Open(t.TempDir())
	painter := &stubPainter{}
	entered := ""

	model, _ := NewSetup("Stacja Erebus", "seba", library, painter, func(name, avatar string) (tea.Model, tea.Cmd) {
		entered = avatar

		return NewSetup("", "", library, nil, nil), nil
	}).Update(tea.WindowSizeMsg{Width: sizes[0][0], Height: sizes[0][1]})

	model = press(model, tea.KeyUp, tea.KeyEnter)
	assertShows(t, model, "describe", "Nowy awatar", "Opisz postać")
	assertFits(t, model, sizes[0], "describe")

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("stara wiedźma z krukiem")})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assertShows(t, model, "drawing", "Rysuję…")

	model, _ = model.Update(cmd())
	assertShows(t, model, "naming", "Tak wygląda twoja postać", "Nazwa awatara", "Stara wiedźma z")
	assertFits(t, model, sizes[0], "naming")

	if len(painter.prompts) != 1 || painter.prompts[0] != "stara wiedźma z krukiem" {
		t.Errorf("prompts = %q", painter.prompts)
	}

	model = press(model, tea.KeyEnter)
	assertShows(t, model, "saved", "Zapisano awatar: Stara wiedźma z", "7 z 7")

	if known := library.List(); len(known) != 7 || known[6].Name != "Stara wiedźma z" || known[6].ID != "stara-wiedzma-z" {
		t.Fatalf("library = %+v", known)
	}

	if art := library.Art("stara-wiedzma-z"); lipgloss.Width(art) != portraitWidth/2 || lipgloss.Height(art) != portraitHeight/2 {
		t.Errorf("stored art is %dx%d", lipgloss.Width(art), lipgloss.Height(art))
	}

	if big := library.Portrait("stara-wiedzma-z"); lipgloss.Height(big) != portraitHeight {
		t.Errorf("stored portrait is %d rows high", lipgloss.Height(big))
	}

	if library.Portrait("pilot") != library.Art("pilot") {
		t.Error("a built-in avatar should fall back to its small picture")
	}

	model = press(model, tea.KeyEnter)
	if entered != "stara-wiedzma-z" {
		t.Errorf("entered with %q", entered)
	}

	// The library is read back from disk, and a failing painter is reported.
	if again := avatars.Open(library.Dir()); len(again.List()) != 7 || again.Art("stara-wiedzma-z") == "" {
		t.Error("the new avatar did not survive a reload")
	}

	painter.err = errors.New("brak modelu")
	model, _ = NewSetup("Stacja Erebus", "seba", library, painter, nil).Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = press(model, tea.KeyUp, tea.KeyEnter)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(cmd())
	assertShows(t, model, "failed drawing", "Nie udało się narysować: brak modelu")
}

func TestFunctionKeysPickCards(t *testing.T) {
	model := newGame(t, newRealm(t, 0), "Seba", sizes[1], nil)

	// With no companions F2 and F3 stay on the hero instead of crashing.
	model = press(model, tea.KeyF2, tea.KeyF3)
	if got := model.(Game).card; got != 0 {
		t.Errorf("card = %d without companions", got)
	}

	assertFits(t, model, sizes[1], "after F3 without companions")

	model = enter(model, "najmij bognę")
	if party := model.(Game).view.Me.Party; len(party) != 1 {
		t.Fatalf("party = %+v", party)
	}

	model = press(model, tea.KeyF2)
	if got := model.(Game).card; got != 1 {
		t.Errorf("card = %d after F2", got)
	}

	assertShows(t, model, "two cards", "F1 Seba", "F2 Bogna Topór", "wojownik, poziom 1", "Broń ")
	assertFits(t, model, sizes[1], "two cards")

	// On a small terminal only the selected card is drawn in full.
	small, _ := model.Update(tea.WindowSizeMsg{Width: sizes[0][0], Height: sizes[0][1]})
	assertShows(t, small, "folded cards", "F1 Seba", "F2 Bogna Topór")
	assertFits(t, small, sizes[0], "folded cards")

	model = press(model, tea.KeyF1)
	if got := model.(Game).card; got != 0 {
		t.Errorf("card = %d after F1", got)
	}
}

// moodyBrain answers every shop line with a tagged mood.
type moodyBrain struct{ llm.Mock }

func (moodyBrain) Narrate(_ context.Context, request game.NarrateRequest) (string, error) {
	return "[kpina] " + request.Decision.Fallback(), nil
}

type stubEar struct {
	on    bool
	heard string
	err   error
}

func (e *stubEar) Start() error          { e.on = true; return e.err }
func (e *stubEar) Recording() bool       { return e.on }
func (e *stubEar) Stop() (string, error) { e.on = false; return e.heard, e.err }

func TestMoodsAndPushToTalk(t *testing.T) {
	realm := newRealm(t, 0)
	voice := &inventingBrain{}
	ear := &stubEar{heard: "ile za kord"}
	model, _ := NewGame(realm, realm.Join("Seba", "wojownik"), moodyBrain{}, avatars.Open(""), nil, voice, ear, nil).Update(tea.WindowSizeMsg{Width: sizes[1][0], Height: sizes[1][1]})

	assertShows(t, model, "help", "F5 mów")

	// F5 starts recording; F5 again stops it and the words land in the line.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF5})
	assertShows(t, model, "recording", "Nagrywam")

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyF5})
	assertShows(t, model, "recognising", "Rozpoznaję")
	model, _ = model.Update(cmd())

	if got := model.(Game).input.Value(); got != "ile za kord" || ear.on {
		t.Errorf("dictated %q, recording %v", got, ear.on)
	}

	// Dictation in a shop goes to the shop's line and the reply is spoken in its mood.
	model = press(model, tea.KeyEsc)
	model = enter(model, "idź do zbrojowni")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF5})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyF5})
	model, _ = model.Update(cmd())

	if got := model.(Game).shop.input.Value(); got != "ile za kord" {
		t.Errorf("shop line = %q", got)
	}

	model = enter(model, "")
	assertShows(t, model, "mood shown", "(kpina)")

	if last := voice.spoken[len(voice.spoken)-1]; !strings.HasPrefix(last, "kpina: ") {
		t.Errorf("spoken = %q", last)
	}

	// Outside, a microphone that fails says so in the log.
	model = press(model, tea.KeyEsc)
	ear.err = errors.New("brak mikrofonu")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF5})
	assertShows(t, model, "mic error", "Mikrofon nie działa: brak mikrofonu")
}

func TestBlowsAreAnimated(t *testing.T) {
	model := newGame(t, newRealm(t, 0), "Seba", sizes[1], nil)
	model = walkIntoFight(t, model)

	game := model.(Game)
	if game.animating() {
		t.Fatal("blows from before the screen looked are replayed")
	}

	// Walking into the fight already drew the monsters' first blows; the
	// hero's attack starts the animation of what it did.
	model = press(model, tea.KeyEnter)
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	game = model.(Game)
	if !game.animating() || cmd == nil || game.frame != 0 {
		t.Fatalf("no animation after an attack: effects %+v, cmd %v", game.effects, cmd != nil)
	}

	still := scene(game.view, 0, nil, 0, false, false)
	frames := make([]string, 0, effectFrames)

	for frame := range effectFrames {
		game = model.(Game)
		frames = append(frames, scene(game.view, 0, game.effects, frame, false, false))
		assertFits(t, model, sizes[1], fmt.Sprintf("frame %d", frame))

		var next tea.Cmd
		model, next = model.Update(frameMsg{serial: game.shown})

		if frame < effectFrames-1 && (next == nil || !model.(Game).animating()) {
			t.Errorf("animation stopped at frame %d", frame)
		}
	}

	if model.(Game).animating() {
		t.Error("the animation did not end")
	}

	for frame, drawn := range frames {
		if drawn == still || lipgloss.Height(drawn) != sceneRows {
			t.Errorf("frame %d shows nothing", frame)
		}
	}

	// A stale tick of an earlier animation changes nothing.
	if _, next := model.Update(frameMsg{serial: -5}); next != nil {
		t.Error("a stale frame scheduled another")
	}
}

func TestBestiaryScreen(t *testing.T) {
	model := newGame(t, newRealm(t, 0), "Seba", sizes[1], nil)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF4})
	assertShows(t, model, "book", "Bestiariusz", "poznane 0/", "???", "Nieznany", "Spotkaj go w walce")
	assertFits(t, model, sizes[1], "book")

	model = press(model, tea.KeyDown, tea.KeyDown, tea.KeyEsc)
	if model.(Game).book || model.(Game).beast != 2 {
		t.Errorf("after Esc: book %v, page %d", model.(Game).book, model.(Game).beast)
	}

	model = walkIntoFight(t, model)
	model = enter(model, "bestiariusz")

	game := model.(Game)
	met := game.view.Fight.Monsters[0].Name
	assertShows(t, model, "book in a fight", "poznane 1/", met)
	assertFits(t, model, sizes[1], "book in a fight")

	// The page of the monster met shows its numbers and trait.
	for index, page := range game.view.Me.Bestiary {
		if page.Name == met {
			game.beast = index
		}
	}

	assertShows(t, game, "page", "Zdrowie", "Cios", "Szybkość", game.view.Fight.Monsters[0].Trait)
}

func TestLinesAddressedToCompanionsAreOrders(t *testing.T) {
	realm := newRealm(t, 0)
	name := realm.Join("Seba", "wojownik")
	model := newGame(t, realm, name, sizes[1], nil)

	// Nobody in the party: the line is an ordinary, unknown command.
	model = enter(model, "Lyrasa, ulecz załogę")
	assertShows(t, model, "no party", "Nie rozumiem")

	// Whoever waits in the inn here is hired and addressed by first name.
	hired := ""
	for _, line := range model.(Game).view.Log {
		if _, names, ok := strings.Cut(line, "W karczmie czeka do wynajęcia: "); ok {
			hired, _, _ = strings.Cut(names, " (")
		}
	}

	model = enter(model, "najmij "+hired)
	if len(model.(Game).view.Me.Party) != 1 {
		t.Fatalf("hiring %q failed: %q", hired, model.(Game).view.Log)
	}

	first, _, _ := strings.Cut(hired, " ")
	model = enter(model, first+", ulecz siebie i załogę")
	// A fighter has no healing; the answer still comes from the companion.
	assertShows(t, model, "order", hired+" nie ma dość mocy na leczenie")
}

// presetIndex finds a shipped world by mode.
func presetIndex(t *testing.T, mode world.Mode) int {
	t.Helper()

	specs, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	for index, spec := range specs {
		if spec.Mode == mode {
			return index
		}
	}

	t.Fatalf("no preset of mode %q", mode)

	return -1
}

func TestDnDBoardAndMovement(t *testing.T) {
	model := newGame(t, newRealm(t, presetIndex(t, world.ModeDnD)), "Seba", sizes[1], nil)
	assertShows(t, model, "sheet", "KP")

	model = walkIntoFight(t, model)
	game := model.(Game)
	if game.view.Fight.Board == nil {
		t.Fatal("a D&D fight has no board")
	}

	assertShows(t, model, "fight", "Ruch", "Atak", "KP ")
	assertFits(t, model, sizes[1], "board")

	if still := scene(game.view, 0, nil, 0, false, false); lipgloss.Height(still) != sceneRows {
		t.Errorf("board scene is %d rows", lipgloss.Height(still))
	}

	// Wait for the hero's turn, then walk: Ruch, arrows, Esc.
	for range 20 {
		if model.(Game).view.Fight == nil || model.(Game).view.Fight.Actor != nil {
			break
		}

		model = press(model, tea.KeyEnter)
	}

	if model.(Game).view.Fight == nil {
		t.Skip("the fight ended before the hero could move")
	}

	game = model.(Game)
	if game.act() != actAttack {
		t.Fatalf("menu rests on action %d, want the attack", game.act())
	}

	model = press(model, tea.KeyUp, tea.KeyEnter)
	if got := model.(Game); got.step != stepMove || got.act() != actMove {
		t.Fatalf("step %d, action %d after choosing Ruch", got.step, got.act())
	}

	assertShows(t, model, "moving", "Ruch: strzałki")
	before := model.(Game).view.Fight.Board.Moves

	// Up or down, one of the two rows beside the hero is free.
	model = press(model, tea.KeyUp, tea.KeyDown)

	if after := model.(Game).view.Fight.Board.Moves; model.(Game).view.Fight != nil && after >= before {
		t.Errorf("moves %d -> %d after stepping aside", before, after)
	}

	model = press(model, tea.KeyEsc)
	if model.(Game).step != stepAction {
		t.Errorf("Esc did not end the walk: step %d", model.(Game).step)
	}
}

func TestAdventureScreens(t *testing.T) {
	model := newGame(t, newRealm(t, presetIndex(t, world.ModeAdventure)), "Seba", sizes[1], nil)
	assertShows(t, model, "start", "Hipokamp", "Sala Pooperacyjna", "Cel: Wydostać się", "Rzeczy: 0", "zbadaj X")
	assertFits(t, model, sizes[1], "adventure")

	model = enter(model, "zbadaj łóżko")
	model = enter(model, "podnieś żeton")
	assertShows(t, model, "took", "Podnosisz: Żeton", "Rzeczy: 1")

	model = press(model, tea.KeyTab)
	assertShows(t, model, "things", "Rzeczy (1)", "Żeton", "Mosiężny żeton")
	assertFits(t, model, sizes[1], "things")

	// The bestiary has nothing to show in an adventure.
	model = press(model, tea.KeyEsc)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyF4})
	if model.(Game).book {
		t.Error("the bestiary opened in an adventure")
	}

	model = enter(model, "pomoc")
	assertShows(t, model, "help", "użyj X na Y")
}

func TestInlinePicturesReplaceTheScene(t *testing.T) {
	realm := newRealm(t, 0)
	model := newGame(t, realm, "Seba", sizes[1], nil)
	view := model.(Game).view

	art := gallery.Open(t.TempDir())
	UseGallery(art)
	t.Cleanup(func() { UseGallery(nil) })

	var terminal bytes.Buffer
	inline.Configure("kitty", &terminal)
	t.Cleanup(func() { inline.Configure("off", nil) })

	// Without a painting the block art stays, pictures or not.
	if plain := scene(view, 0, nil, 0, false, true); plain != scene(view, 0, nil, 0, false, false) {
		t.Error("a world without paintings should look the same")
	}

	dir := art.Dir(view.World.Spec)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	variant := (view.Me.Pos.X*7+view.Me.Pos.Y*13)%sceneVariants + 1
	name := fmt.Sprintf("place-%d-%d.png", view.Tile.Place+1, variant)
	if err := os.WriteFile(filepath.Join(dir, name), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	picture := scene(view, 0, nil, 0, false, true)
	if lipgloss.Height(picture) != sceneRows || lipgloss.Width(picture) != sceneWidth || !strings.Contains(picture, "\U0010EEEE") {
		t.Errorf("the painting was not placed: %dx%d", lipgloss.Width(picture), lipgloss.Height(picture))
	}

	if !strings.HasPrefix(terminal.String(), "\x1b_G") {
		t.Error("the painting was not sent to the terminal")
	}

	// A guest's screen, or one with pictures off, keeps the block art.
	if blocks := scene(view, 0, nil, 0, false, false); strings.Contains(blocks, "\U0010EEEE") {
		t.Error("block art screens got placeholders")
	}

	// In a fight the monsters must be seen, so the block art returns.
	model = walkIntoFight(t, model)
	if fight := scene(model.(Game).view, 0, nil, 0, false, true); strings.Contains(fight, "\U0010EEEE") {
		t.Error("a fight was hidden behind the painting")
	}
}
