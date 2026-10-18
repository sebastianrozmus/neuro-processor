package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
)

// Settings is what the pause menu changes: the voice the game speaks with.
type Settings interface {
	Options() speech.Options
	Choices() speech.Choices
	// Apply switches the voice and keeps the choice for the next game.
	Apply(options speech.Options) error
}

// ModelChoice is the model of one role and whether it runs on the
// processor.
type ModelChoice struct {
	Model string
	CPU   bool
}

// Models is what the pause menu changes about the language models: which
// one each role uses, on the graphics card or the processor.
type Models interface {
	// Roles lists the roles in order; Picked the model of each.
	Roles() []string
	Picked() map[string]ModelChoice
	// Available lists the models the server has.
	Available() ([]string, error)
	// Use switches the roles and keeps the choice for the next game.
	Use(picked map[string]ModelChoice) error
}

// Pause is what the host gives the pause menu: the saved games and how to
// load one, and the settings. Any of them may be missing, and the menu
// leaves out what it cannot do.
type Pause struct {
	Saves    func() []game.State
	Load     func(state game.State) (tea.Model, tea.Cmd, error)
	Settings Settings
	Models   Models
}

// The screens of the pause menu.
const (
	pauseMain = iota
	pauseLoad
	pauseSettings
	pauseModels
)

// The rows of the settings screen.
const (
	rowEngine = iota
	rowSpeed
	rowVoice
	rowNarrator
	rowDevice
	rowApply
	rowBack
)

// pauseMenu is the menu Escape opens over the map.
type pauseMenu struct {
	open    bool
	screen  int
	cursor  int
	saves   []game.State
	draft   speech.Options
	choices speech.Choices
	// models is the draft of the models screen, available what the
	// server offers.
	models    map[string]ModelChoice
	available []string
	note      string
}

var (
	engineLabels = map[string]string{
		speech.EngineOff:        "wyłączony",
		speech.EngineChatterbox: "Chatterbox",
		speech.EnginePiper:      "Piper",
		speech.EngineSystem:     "systemowy",
	}
	engineNotes = map[string]string{
		speech.EngineOff:        "Gra milczy.",
		speech.EngineChatterbox: "Najlepszy: słychać nastroje i każda postać ma swój głos z próbki. Zajmuje ok. 4 GB karty graficznej i potrzebuje kilku sekund na zdanie.",
		speech.EnginePiper:      "Lekki: działa na procesorze, zdanie w ułamku sekundy, bez karty graficznej. Naturalny polski głos, ale bez nastrojów.",
		speech.EngineSystem:     "Głos systemu (Windows: Paulina, macOS: Zosia). Nic nie trzeba instalować, brzmi najbardziej mechanicznie.",
	}
	devices      = []string{"auto", "cuda", "cpu"}
	deviceLabels = map[string]string{"auto": "automatycznie", "cuda": "karta graficzna", "cpu": "procesor (wolno, bez karty)"}
)

// Pause gives the screen its pause menu.
func (g Game) Pause(pause Pause) Game {
	g.pauser = pause

	return g
}

// openPause opens the pause menu on its first screen.
func (g *Game) openPause() {
	g.pause = pauseMenu{open: true}
}

// pauseItems are the entries of the first screen.
func (g Game) pauseItems() []string {
	items := []string{"Wróć do gry"}
	if g.save != nil {
		items = append(items, "Zapisz grę")
	}

	if g.pauser.Load != nil && g.pauser.Saves != nil {
		items = append(items, "Wczytaj grę")
	}

	if g.pauser.Settings != nil {
		items = append(items, "Ustawienia głosu")
	}

	if g.pauser.Models != nil {
		items = append(items, "Modele językowe")
	}

	return append(items, "Wyjdź z gry")
}

// settingsRows are the rows the settings screen shows for the engine in
// the draft.
func (g Game) settingsRows() []int {
	rows := []int{rowEngine}

	switch g.pause.draft.Engine {
	case speech.EnginePiper:
		rows = append(rows, rowSpeed, rowVoice, rowNarrator)
	case speech.EngineSystem:
		rows = append(rows, rowSpeed)
		if len(g.pause.choices.SystemVoices) > 0 {
			rows = append(rows, rowVoice)
		}
	case speech.EngineChatterbox:
		rows = append(rows, rowDevice)
	}

	return append(rows, rowApply, rowBack)
}

func (g Game) pauseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch g.pause.screen {
	case pauseLoad:
		return g.loadKey(msg)
	case pauseModels:
		g.modelsKey(msg)

		return g, nil
	case pauseSettings:
		g.settingsKey(msg)

		return g, nil
	}

	items := g.pauseItems()

	switch msg.Type {
	case tea.KeyEsc:
		g.pause.open = false
	case tea.KeyUp:
		g.pause.cursor = (g.pause.cursor + len(items) - 1) % len(items)
	case tea.KeyDown:
		g.pause.cursor = (g.pause.cursor + 1) % len(items)
	case tea.KeyEnter:
		switch items[g.pause.cursor] {
		case "Wróć do gry":
			g.pause.open = false
		case "Zapisz grę":
			g.pause.note = g.saved()
			g.realm.Tell(g.name, g.pause.note)
			g.refresh()
		case "Wczytaj grę":
			g.pause.saves = g.pauser.Saves()
			g.pause.screen, g.pause.cursor, g.pause.note = pauseLoad, 0, ""
			if len(g.pause.saves) == 0 {
				g.pause.screen, g.pause.note = pauseMain, "Nie ma jeszcze zapisanych gier."
			}
		case "Ustawienia głosu":
			g.pause.draft, g.pause.choices = g.pauser.Settings.Options(), g.pauser.Settings.Choices()
			if g.pause.draft.Engine == "" {
				g.pause.draft.Engine = speech.EngineOff
			}

			g.pause.screen, g.pause.cursor, g.pause.note = pauseSettings, 0, ""
		case "Modele językowe":
			g.pause.models = g.pauser.Models.Picked()
			available, err := g.pauser.Models.Available()
			g.pause.available, g.pause.screen, g.pause.cursor, g.pause.note = available, pauseModels, 0, ""

			if err != nil {
				g.pause.note = "Serwer modeli nie podał listy: " + err.Error()
			}
		case "Wyjdź z gry":
			return g, tea.Quit
		}
	}

	return g, nil
}

func (g Game) loadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		g.pause.screen, g.pause.cursor = pauseMain, 0
	case tea.KeyUp:
		g.pause.cursor = max(0, g.pause.cursor-1)
	case tea.KeyDown:
		g.pause.cursor = min(len(g.pause.saves)-1, g.pause.cursor+1)
	case tea.KeyEnter:
		model, cmd, err := g.pauser.Load(g.pause.saves[g.pause.cursor])
		if err != nil {
			g.pause.note = "Nie udało się wczytać gry: " + err.Error()

			return g, nil
		}

		// The new screen has not been told how big the terminal is.
		width, height := g.width, g.height

		return model, tea.Batch(cmd, func() tea.Msg { return tea.WindowSizeMsg{Width: width, Height: height} })
	}

	return g, nil
}

func (g *Game) settingsKey(msg tea.KeyMsg) {
	rows := g.settingsRows()
	g.pause.cursor = min(g.pause.cursor, len(rows)-1)
	row := rows[g.pause.cursor]

	switch msg.Type {
	case tea.KeyEsc:
		g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
	case tea.KeyUp:
		g.pause.cursor = (g.pause.cursor + len(rows) - 1) % len(rows)
	case tea.KeyDown:
		g.pause.cursor = (g.pause.cursor + 1) % len(rows)
	case tea.KeyLeft, tea.KeyRight:
		step := 1
		if msg.Type == tea.KeyLeft {
			step = -1
		}

		g.change(row, step)
	case tea.KeyEnter:
		switch row {
		case rowApply:
			if err := g.pauser.Settings.Apply(g.pause.draft); err != nil {
				g.pause.note = "Nie udało się: " + err.Error()

				return
			}

			g.pause.draft = g.pauser.Settings.Options()
			g.pause.note = "Zapisane."
			if g.pause.draft.Engine == speech.EngineChatterbox {
				g.pause.note += " Chatterbox wczytuje się do kilkudziesięciu sekund; wcześniejsze zdania przepadną."
			}

			if g.voice != nil {
				g.voice.Say("Tak brzmi teraz mój głos. Ruszamy dalej?", speech.Calm)
			}
		case rowBack:
			g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
		default:
			g.change(row, 1)
		}
	}
}

// change moves a setting one step.
func (g *Game) change(row, step int) {
	draft := &g.pause.draft

	switch row {
	case rowEngine:
		draft.Engine = cycle(g.pause.choices.Engines, draft.Engine, step)

		// A voice of one engine means nothing to another.
		switch voices := g.pause.choices.PiperVoices; {
		case draft.Engine == speech.EnginePiper && !slices.Contains(voices, draft.Voice) && len(voices) > 0:
			draft.Voice = voices[0]
			for _, voice := range voices {
				if speech.PiperLabel(voice) == "gosia" {
					draft.Voice = voice
				}
			}
		case draft.Engine == speech.EngineSystem && !slices.Contains(g.pause.choices.SystemVoices, draft.Voice):
			draft.Voice = ""
		}
	case rowSpeed:
		speed := draft.Speed
		if speed <= 0 {
			speed = 1
		}

		draft.Speed = max(0.5, min(2, float64(int((speed+0.1*float64(step))*10+0.5))/10))
	case rowVoice:
		voices := g.pause.choices.SystemVoices
		if draft.Engine == speech.EnginePiper {
			voices = g.pause.choices.PiperVoices
		}

		draft.Voice = cycle(voices, draft.Voice, step)
	case rowNarrator:
		draft.Narrator = cycle(append([]string{""}, g.pause.choices.PiperVoices...), draft.Narrator, step)
	case rowDevice:
		device := draft.Device
		if device == "" {
			device = "auto"
		}

		draft.Device = cycle(devices, device, step)
	}
}

// cycle is the item after (or before) current in list, round the end.
func cycle(list []string, current string, step int) string {
	if len(list) == 0 {
		return current
	}

	for index, item := range list {
		if item == current {
			return list[(index+step+len(list))%len(list)]
		}
	}

	return list[0]
}

// pauseView draws the pause menu in the middle of the screen.
func (g Game) pauseView() string {
	var lines []string

	switch g.pause.screen {
	case pauseLoad:
		lines = append(lines, titleStyle.Render("Wczytaj grę"), "")
		for index, save := range g.pause.saves {
			when := save.SavedAt.Local().Format("02.01.2006 15:04")
			lines = append(lines, pointer(index == g.pause.cursor)+save.Spec.Name+labelStyle.Render(fmt.Sprintf(" (%s, %s)", save.Host, when)))
		}

		lines = append(lines, "", dimStyle.Render("Enter: wczytaj · Esc: wróć"))
	case pauseSettings:
		lines = append(lines, titleStyle.Render("Ustawienia: głos"), "")
		for index, row := range g.settingsRows() {
			lines = append(lines, pointer(index == g.pause.cursor)+g.settingsLine(row))
		}

		lines = append(lines, "", wrap(dimStyle.Render(engineNotes[g.pause.draft.Engine]), 56),
			"", dimStyle.Render("←/→: zmień · Enter: wybierz · Esc: wróć"))
	case pauseModels:
		lines = append(lines, titleStyle.Render("Ustawienia: modele językowe"), "")
		rows := g.modelRows()
		for index := range rows {
			lines = append(lines, pointer(index == g.pause.cursor)+g.modelLine(rows[index]))
		}

		lines = append(lines, "", wrap(dimStyle.Render("Procesor: model nie zajmuje karty graficznej, ale odpowiada kilka razy wolniej. Dobre dla umysłu, który pyta rzadko i w tle; zła rada dla rozmów."), 56),
			"", dimStyle.Render("←/→: zmień · Enter: wybierz · Esc: wróć"))
	default:
		lines = append(lines, titleStyle.Render("Pauza"), "")
		for index, item := range g.pauseItems() {
			lines = append(lines, pointer(index == g.pause.cursor)+item)
		}

		lines = append(lines, "", dimStyle.Render("Enter: wybierz · Esc: wróć do gry"))
	}

	if g.pause.note != "" {
		lines = append(lines, "", wrap(warnStyle.Render(g.pause.note), 56))
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("214")).Padding(1, 3).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(g.width, g.height, lipgloss.Center, lipgloss.Center, box)
}

// settingsLine is one row of the settings screen.
func (g Game) settingsLine(row int) string {
	draft := g.pause.draft
	value := func(text string) string { return goldStyle.Render("‹ " + text + " ›") }

	switch row {
	case rowEngine:
		return "Głos: " + value(engineLabels[draft.Engine])
	case rowSpeed:
		speed := draft.Speed
		if speed <= 0 {
			speed = 1
		}

		return "Tempo mowy: " + value(fmt.Sprintf("%.1f×", speed))
	case rowVoice:
		voice := draft.Voice
		if draft.Engine == speech.EnginePiper {
			voice = speech.PiperLabel(voice)
		}

		if voice == "" {
			voice = "domyślny"
		}

		return "Głos postaci: " + value(voice)
	case rowNarrator:
		voice := speech.PiperLabel(draft.Narrator)
		if draft.Narrator == "" {
			voice = "jak postaci"
		}

		return "Głos narratora: " + value(voice)
	case rowDevice:
		device := draft.Device
		if device == "" {
			device = "auto"
		}

		return "Chatterbox działa na: " + value(deviceLabels[device])
	case rowApply:
		return goodStyle.Render("Zastosuj i posłuchaj")
	default:
		return "Wróć"
	}
}

func pointer(on bool) string {
	if on {
		return goldStyle.Render("› ")
	}

	return "  "
}

// wrap breaks a text to the width.
func wrap(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

// roleNotes say what each role of the models does.
var roleNotes = map[string]string{
	"rozmowy": "kwestie postaci",
	"walka":   "narrator walk",
	"umysł":   "rozumie gracza, wyłapuje fakty i przedmioty",
	"światy":  "generuje światy",
	"obrazki": "opisy do obrazków po angielsku",
}

// modelRow is a row of the models screen: a role's model, a role's
// device, or one of the buttons.
type modelRow struct {
	role   string
	device bool
	action string
}

func (g Game) modelRows() []modelRow {
	rows := []modelRow{}
	for _, role := range g.pauser.Models.Roles() {
		rows = append(rows, modelRow{role: role}, modelRow{role: role, device: true})
	}

	return append(rows, modelRow{action: "apply"}, modelRow{action: "back"})
}

func (g Game) modelLine(row modelRow) string {
	value := func(text string) string { return goldStyle.Render("‹ " + text + " ›") }
	choice := g.pause.models[row.role]

	switch {
	case row.action == "apply":
		return goodStyle.Render("Zastosuj i zapisz")
	case row.action == "back":
		return "Wróć"
	case row.device:
		where := "karta graficzna"
		if choice.CPU {
			where = "procesor"
		}

		return "    działa na: " + value(where)
	}

	return fmt.Sprintf("%s %s: %s", titleStyle.Render(row.role), labelStyle.Render("("+roleNotes[row.role]+")"), value(choice.Model))
}

func (g *Game) modelsKey(msg tea.KeyMsg) {
	rows := g.modelRows()
	g.pause.cursor = min(g.pause.cursor, len(rows)-1)
	row := rows[g.pause.cursor]

	switch msg.Type {
	case tea.KeyEsc:
		g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
	case tea.KeyUp:
		g.pause.cursor = (g.pause.cursor + len(rows) - 1) % len(rows)
	case tea.KeyDown:
		g.pause.cursor = (g.pause.cursor + 1) % len(rows)
	case tea.KeyLeft, tea.KeyRight, tea.KeyEnter:
		step := 1
		if msg.Type == tea.KeyLeft {
			step = -1
		}

		switch {
		case row.action == "back":
			g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
		case row.action == "apply":
			if msg.Type != tea.KeyEnter {
				return
			}

			if err := g.pauser.Models.Use(g.pause.models); err != nil {
				g.pause.note = "Nie udało się: " + err.Error()

				return
			}

			g.pause.note = "Zapisane. Nowy model wczyta się przy pierwszym pytaniu."
		case row.device:
			choice := g.pause.models[row.role]
			choice.CPU = !choice.CPU
			g.pause.models[row.role] = choice
		default:
			choice := g.pause.models[row.role]
			choice.Model = cycle(g.pause.available, choice.Model, step)
			g.pause.models[row.role] = choice
		}
	}
}
