package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/inline"
)

const setupWidth = 56

// Enter lets a hero into the world and returns the interface to play with.
type Enter func(name, avatar string) (tea.Model, tea.Cmd)

// The setup screen is either picking a portrait, describing a new one,
// waiting for it to be drawn, or naming the result.
const (
	stagePick = iota
	stageDescribe
	stageDrawing
	stageName
)

// drawnMsg carries a freshly drawn portrait.
type drawnMsg struct {
	drawing avatars.Drawing
	err     error
}

// Setup is the screen where a player names their hero and picks an avatar
// before entering a world. With a generator, they can also have a new
// avatar drawn from a description and keep it.
type Setup struct {
	world     string
	library   *avatars.Library
	inline    bool
	generator avatars.Generator
	enter     Enter

	name   textinput.Model
	prompt textinput.Model
	title  textinput.Model
	avatar int
	stage  int
	draft  avatars.Drawing
	note   string
	width  int
	height int
}

// NewSetup builds the screen for entering the named world. The name is a
// suggestion the player can change. The generator may be nil.
// Inline turns showing player-made portraits as real images on or off.
func (s Setup) Inline(on bool) Setup {
	s.inline = on

	return s
}

func NewSetup(world, name string, library *avatars.Library, generator avatars.Generator, enter Enter) Setup {
	field := func(placeholder string, width, limit int) textinput.Model {
		input := textinput.New()
		input.Prompt = "> "
		input.Placeholder = placeholder
		input.CharLimit = limit
		input.Width = width

		return input
	}

	setup := Setup{
		world:     world,
		library:   library,
		generator: generator,
		enter:     enter,
		name:      field("", 20, 16),
		prompt:    field("np. stara wiedźma z krukiem na ramieniu", setupWidth-8, 200),
		title:     field("np. Wiedźma", 24, 24),
	}

	setup.name.SetValue(name)
	setup.name.CursorEnd()
	setup.name.Focus()

	return setup
}

// Init starts the cursor blinking.
func (s Setup) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles the name field, the avatar picker and making new avatars.
func (s Setup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height

		return s, nil
	case drawnMsg:
		if msg.err != nil {
			s.stage, s.note = stageDescribe, "Nie udało się narysować: "+msg.err.Error()

			return s, s.prompt.Focus()
		}

		s.stage, s.draft, s.note = stageName, msg.drawing, ""
		s.title.SetValue(suggestName(s.prompt.Value()))
		s.title.CursorEnd()

		return s, s.title.Focus()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return s, tea.Quit
		}

		switch s.stage {
		case stagePick:
			return s.pickKey(msg)
		case stageDescribe:
			return s.describeKey(msg)
		case stageName:
			return s.nameKey(msg)
		}
	}

	return s, nil
}

// entries is how many choices the picker shows: the avatars and "add new".
func (s Setup) entries() int {
	return len(s.library.List()) + 1
}

func (s Setup) pickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		s.avatar = (s.avatar + s.entries() - 1) % s.entries()

		return s, nil
	case tea.KeyDown, tea.KeyTab:
		s.avatar = (s.avatar + 1) % s.entries()

		return s, nil
	case tea.KeyEnter:
		if s.avatar < len(s.library.List()) {
			model, cmd := s.enter(strings.TrimSpace(s.name.Value()), s.library.List()[s.avatar].ID)
			model, _ = model.Update(tea.WindowSizeMsg{Width: s.width, Height: s.height})

			return model, cmd
		}

		if s.generator == nil {
			s.note = "Rysowanie awatarów nie jest skonfigurowane: uruchom tools/avatar/setup.sh (opis w README)."

			return s, nil
		}

		s.stage, s.note = stageDescribe, ""
		s.name.Blur()

		return s, s.prompt.Focus()
	}

	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)

	return s, cmd
}

func (s Setup) describeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		s.stage = stagePick
		s.prompt.Blur()

		return s, s.name.Focus()
	case tea.KeyEnter:
		prompt := strings.TrimSpace(s.prompt.Value())
		if prompt == "" {
			return s, nil
		}

		s.stage, s.note = stageDrawing, ""
		s.prompt.Blur()
		generator := s.generator

		return s, func() tea.Msg {
			drawing, err := generator.Generate(context.Background(), prompt)

			return drawnMsg{drawing: drawing, err: err}
		}
	}

	var cmd tea.Cmd
	s.prompt, cmd = s.prompt.Update(msg)

	return s, cmd
}

func (s Setup) nameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Back to the description, to try again.
		s.stage, s.draft = stageDescribe, avatars.Drawing{}
		s.title.Blur()

		return s, s.prompt.Focus()
	case tea.KeyEnter:
		avatar, err := s.library.Add(s.title.Value(), s.draft)
		if err != nil {
			s.note = "Nie udało się zapisać: " + err.Error()

			return s, nil
		}

		s.stage, s.draft, s.note = stagePick, avatars.Drawing{}, "Zapisano awatar: "+avatar.Name
		s.title.Blur()

		for index, known := range s.library.List() {
			if known.ID == avatar.ID {
				s.avatar = index
			}
		}

		return s, s.name.Focus()
	}

	var cmd tea.Cmd
	s.title, cmd = s.title.Update(msg)

	return s, cmd
}

// suggestName proposes a name for a drawn avatar from its description.
func suggestName(prompt string) string {
	words := strings.Fields(prompt)
	if len(words) > 3 {
		words = words[:3]
	}

	name := strings.Join(words, " ")
	if name == "" {
		return name
	}

	return strings.ToUpper(name[:1]) + name[1:]
}

// View renders the screen.
func (s Setup) View() string {
	centre := lipgloss.NewStyle().Width(setupWidth).Align(lipgloss.Center)
	lines := []string{centre.Render(titleStyle.Render(s.world))}

	switch s.stage {
	case stageDescribe, stageDrawing:
		lines = append(lines,
			centre.Render(labelStyle.Render("Nowy awatar")),
			"",
			"Opisz postać (po polsku lub po angielsku):",
			s.prompt.View(),
			"",
		)

		if s.stage == stageDrawing {
			lines = append(lines, dimStyle.Render("Rysuję… pierwszy raz może potrwać dłużej, bo model się ładuje."))
		} else {
			lines = append(lines, labelStyle.Render("Enter rysuje · Esc wraca"))
		}
	case stageName:
		lines = append(lines,
			centre.Render(labelStyle.Render("Tak wygląda twoja postać")),
			"",
			lipgloss.PlaceHorizontal(setupWidth, lipgloss.Center, s.draft.Large),
			"",
			"Nazwa awatara:",
			s.title.View(),
			"",
			labelStyle.Render("Enter zapisuje · Esc rysuje od nowa"),
		)
	default:
		known := s.library.List()

		if s.avatar < len(known) {
			avatar := known[s.avatar]
			lines = append(lines,
				centre.Render(labelStyle.Render("Kim jesteś w tym świecie?")),
				"",
				lipgloss.PlaceHorizontal(setupWidth, lipgloss.Center, s.portrait(avatar.ID)),
				"",
				centre.Render(fmt.Sprintf("▲ %s ▼", selfStyle.Render(avatar.Name))),
				centre.Render(labelStyle.Render(fmt.Sprintf("%d z %d", s.avatar+1, len(known)))),
			)
		} else {
			lines = append(lines,
				centre.Render(labelStyle.Render("Kim jesteś w tym świecie?")),
				"",
				lipgloss.Place(setupWidth, portraitHeight/2, lipgloss.Center, lipgloss.Center, goldStyle.Render("+ Dodaj nowy awatar")),
				"",
				centre.Render(fmt.Sprintf("▲ %s ▼", selfStyle.Render("Dodaj nowy…"))),
				centre.Render(labelStyle.Render("Opiszesz postać, a lokalny model ją narysuje")),
			)
		}

		lines = append(lines, "", labelStyle.Render("Imię:"), s.name.View(), "", labelStyle.Render("↑↓ albo Tab zmienia postać · Enter zaczyna grę"))
	}

	if s.note != "" {
		lines = append(lines, "", warnStyle.Render(s.note))
	}

	screen := box.Padding(1, 2).Render(strings.Join(lines, "\n"))
	if s.width == 0 {
		return screen
	}

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, screen)
}

// portrait is the large picture of an avatar: the real one when the
// terminal shows images and the player drew it, else block art.
func (s Setup) portrait(id string) string {
	if s.inline {
		if png, path := s.library.Picture(id); len(png) > 0 {
			if picture := inline.Show(path, png, portraitWidth, portraitHeight); picture != "" {
				return picture
			}
		}
	}

	return s.library.Portrait(id)
}
