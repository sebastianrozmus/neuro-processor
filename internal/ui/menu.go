package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	generateTimeout = 5 * time.Minute
	slideInterval   = 6 * time.Second
	menuWidth       = 72
	themeLimit      = 400
)

// Generator designs a world around a theme given by the player.
type Generator interface {
	GenerateWorld(ctx context.Context, mode world.Mode, theme string) (world.Spec, error)
}

// Launcher starts games picked in the menu and returns the screen to show.
type Launcher interface {
	// New starts a fresh game in a world.
	New(spec world.Spec) (tea.Model, tea.Cmd, error)
	// Load resumes a saved game.
	Load(state game.State) (tea.Model, tea.Cmd, error)
}

// generatedMsg carries a freshly generated world.
type generatedMsg struct {
	spec world.Spec
	err  error
}

// slideMsg changes the picture behind the menu.
type slideMsg struct{}

// Menu is the first screen: it lets the player resume a saved game, pick a
// built-in world or one generated earlier, or have a new one generated.
type Menu struct {
	saves     []game.State
	worlds    []world.Spec
	presets   int
	generator Generator
	worldsDir string
	backend   string
	launcher  Launcher

	cursor int
	slide  int
	// choosing is the step of picking the kind of game before describing
	// it; mode is the kind picked.
	choosing bool
	mode     int
	typing   bool
	busy     bool
	note     string
	input    textinput.Model
	width    int
	height   int
}

// NewMenu builds the menu. The generator may be nil when no model is
// configured; generated worlds are stored in worldsDir.
func NewMenu(saves []game.State, presets, generated []world.Spec, generator Generator, worldsDir, backend string, launcher Launcher) Menu {
	input := textinput.New()
	input.Placeholder = "np. wikingowie na zamarzniętym morzu, w którym coś się budzi"
	input.Prompt = "> "
	input.CharLimit = themeLimit
	input.Width = menuWidth - 8

	return Menu{
		saves:     saves,
		worlds:    append(append([]world.Spec(nil), presets...), generated...),
		presets:   len(presets),
		generator: generator,
		worldsDir: worldsDir,
		backend:   backend,
		launcher:  launcher,
		cursor:    len(saves),
		input:     input,
	}
}

// Init starts the slideshow behind the menu.
func (m Menu) Init() tea.Cmd {
	return nextSlide()
}

func nextSlide() tea.Cmd {
	return tea.Tick(slideInterval, func(time.Time) tea.Msg { return slideMsg{} })
}

// Update handles the menu keys and the result of a generation.
func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		return m, nil
	case slideMsg:
		m.slide++

		return m, nextSlide()
	case generatedMsg:
		m.busy = false

		if msg.err != nil {
			m.note = "Nie udało się wygenerować świata: " + msg.err.Error()

			return m, nil
		}

		if _, err := world.SaveSpec(m.worldsDir, msg.spec); err != nil {
			m.note = "Świat nie został zapisany: " + err.Error()
		}

		m.worlds = append(m.worlds, msg.spec)

		return m.launch(m.launcher.New(msg.spec))
	case tea.KeyMsg:
		return m.key(msg)
	}

	return m, nil
}

func (m Menu) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if m.busy {
		return m, nil
	}

	if m.typing {
		return m.typingKey(msg)
	}

	if m.choosing {
		return m.choosingKey(msg)
	}

	entries := len(m.saves) + len(m.worlds)

	switch msg.String() {
	case "up", "w", "k":
		m.cursor, m.slide = max(0, m.cursor-1), 0
	case "down", "s", "j":
		m.cursor, m.slide = min(entries, m.cursor+1), 0
	case "q", "esc":
		return m, tea.Quit
	case "enter", " ":
		switch {
		case m.cursor < len(m.saves):
			return m.launch(m.launcher.Load(m.saves[m.cursor]))
		case m.cursor < entries:
			return m.launch(m.launcher.New(m.worlds[m.cursor-len(m.saves)]))
		case m.generator == nil:
			m.note = "Generator wymaga modelu językowego: ustaw NEURO_LLM_MODEL (opis w README)."
		default:
			m.choosing, m.note = true, ""
		}
	}

	return m, nil
}

// choosingKey picks the kind of game a new world is played as.
func (m Menu) choosingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.mode = max(0, m.mode-1)
	case "down", "j":
		m.mode = min(len(world.Modes)-1, m.mode+1)
	case "esc", "q":
		m.choosing = false
	case "enter", " ":
		m.choosing, m.typing = false, true

		return m, m.input.Focus()
	}

	return m, nil
}

func (m Menu) typingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.typing = false
		m.input.Blur()

		return m, nil
	case tea.KeyEnter:
		theme := strings.TrimSpace(m.input.Value())
		if theme == "" {
			return m, nil
		}

		m.typing, m.busy = false, true
		m.input.Blur()
		generator, mode := m.generator, world.Modes[m.mode]

		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), generateTimeout)
			defer cancel()

			spec, err := generator.GenerateWorld(ctx, mode, theme)

			return generatedMsg{spec: spec, err: err}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

// launch hands the screen over to a started game.
func (m Menu) launch(model tea.Model, cmd tea.Cmd, err error) (tea.Model, tea.Cmd) {
	if err != nil {
		m.note = "Nie udało się uruchomić gry: " + err.Error()

		return m, nil
	}

	model, _ = model.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	return model, cmd
}

// selected returns the world the cursor points at, if it points at one.
func (m Menu) selected() (world.Spec, bool) {
	switch {
	case m.cursor < len(m.saves):
		return m.saves[m.cursor].Spec, true
	case m.cursor < len(m.saves)+len(m.worlds):
		return m.worlds[m.cursor-len(m.saves)], true
	}

	return world.Spec{}, false
}

// View renders the menu over a picture of the selected world.
func (m Menu) View() string {
	lines := []string{titleStyle.Render("NEURO PROCESSOR"), labelStyle.Render("Wybierz świat albo opisz własny."), ""}

	for index, save := range m.saves {
		when := save.SavedAt.Local().Format("02.01.2006 15:04")
		lines = append(lines, m.pointer(index)+goodStyle.Render("Wczytaj: ")+save.Spec.Name+labelStyle.Render(fmt.Sprintf(" (%s, %s)", save.Host, when)))
	}

	for index, spec := range m.worlds {
		tag := " · " + spec.Mode.Title()
		if index >= m.presets {
			tag += ", wygenerowany"
		}

		lines = append(lines, m.pointer(len(m.saves)+index)+spec.Name+labelStyle.Render(tag))
	}

	lines = append(lines, m.pointer(len(m.saves)+len(m.worlds))+goldStyle.Render("Wygeneruj nowy świat…"), "")

	spec, pointsAtWorld := m.selected()

	switch {
	case m.busy:
		lines = append(lines, dimStyle.Render("Tworzę świat, to może potrwać do kilku minut…"))
	case m.choosing:
		lines = append(lines, "Jaki to ma być rodzaj gry?")

		for index, mode := range world.Modes {
			pointer := "  "
			if index == m.mode {
				pointer = selfStyle.Render("> ")
			}

			lines = append(lines, pointer+mode.Title()+"  "+dimStyle.Render(mode.Blurb()))
		}

		lines = append(lines, labelStyle.Render("↑↓ wybór · Enter dalej · Esc wraca"))
	case m.typing:
		lines = append(lines, "O czym ma być gra? ("+world.Modes[m.mode].Title()+")", m.input.View(), labelStyle.Render("Enter generuje · Esc wraca"))
	case pointsAtWorld:
		lines = append(lines, labelStyle.Render(spec.Mode.Title()+": ")+dimStyle.Render(spec.Mode.Blurb()), dimStyle.Render(spec.Intro))
	default:
		lines = append(lines, dimStyle.Render("Opisz w paru słowach, o czym ma być gra, a model wymyśli miejsca, przeciwników, przedmioty i mieszkańców osad. Mapę, statystyki i ceny wyliczy gra."))
	}

	if m.note != "" {
		lines = append(lines, "", warnStyle.Render(m.note))
	}

	lines = append(lines, "", labelStyle.Render(fmt.Sprintf("↑↓ wybór · Enter start · Q wyjście · model: %s", m.backend)))

	menu := box.Padding(1, 2).Width(menuWidth).Render(strings.Join(lines, "\n"))
	if m.width == 0 {
		return menu
	}

	return backdropFor(spec, m.slide).under(menu, m.width, m.height)
}

func (m Menu) pointer(index int) string {
	if index == m.cursor {
		return selfStyle.Render("> ")
	}

	return "  "
}
