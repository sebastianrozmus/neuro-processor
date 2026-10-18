package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
)

// stubSettings records the voice the pause menu applies.
type stubSettings struct {
	options speech.Options
	applied []speech.Options
}

func (s *stubSettings) Options() speech.Options { return s.options }

func (s *stubSettings) Choices() speech.Choices {
	return speech.Choices{
		Engines:     []string{speech.EngineOff, speech.EngineChatterbox, speech.EnginePiper, speech.EngineSystem},
		PiperVoices: []string{"pl_PL-darkman-medium.onnx", "pl_PL-gosia-medium.onnx"},
	}
}

func (s *stubSettings) Apply(options speech.Options) error {
	s.options = options
	s.applied = append(s.applied, options)

	return nil
}

func pausedGame(t *testing.T, pause Pause, save Save) tea.Model {
	t.Helper()

	realm := newRealm(t, 0)
	model, _ := NewGame(realm, realm.Join("Seba", "wojownik"), llm.Mock{}, avatars.Open(""), save, nil, nil, nil).
		Pause(pause).Update(tea.WindowSizeMsg{Width: sizes[1][0], Height: sizes[1][1]})

	return model
}

func TestEscapeClearsTypingThenOpensPause(t *testing.T) {
	model := pausedGame(t, Pause{}, nil)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("idz")})
	model = press(model, tea.KeyEsc)
	if model.(Game).pause.open || model.(Game).input.Value() != "" {
		t.Fatal("the first Escape should only clear what was typed")
	}

	model = press(model, tea.KeyEsc)
	assertShows(t, model, "pause", "Pauza", "Wróć do gry", "Wyjdź z gry")
	assertFits(t, model, sizes[1], "pause")

	model = press(model, tea.KeyEsc)
	if model.(Game).pause.open {
		t.Error("Escape in the pause menu should go back to the game")
	}
}

func TestPauseSavesAndQuits(t *testing.T) {
	saved := 0
	model := pausedGame(t, Pause{}, func() (string, error) {
		saved++

		return "/tmp/zapis.json", nil
	})

	model = press(model, tea.KeyEsc, tea.KeyDown, tea.KeyEnter)
	if saved != 1 {
		t.Fatalf("save called %d times", saved)
	}

	assertShows(t, model, "saved", "Gra zapisana")

	model = press(model, tea.KeyDown)
	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("Wyjdź z gry did not quit")
	}
}

func TestPauseLoadsSave(t *testing.T) {
	spec := preset(t, 1)
	loaded := ""
	next := pausedGame(t, Pause{}, nil)
	pause := Pause{
		Saves: func() []game.State {
			return []game.State{{Spec: spec, Host: "Seba", SavedAt: time.Now()}}
		},
		Load: func(state game.State) (tea.Model, tea.Cmd, error) {
			loaded = state.Spec.Name

			return next, nil, nil
		},
	}

	model := pausedGame(t, pause, nil)
	model = press(model, tea.KeyEsc, tea.KeyDown, tea.KeyEnter)
	assertShows(t, model, "load list", "Wczytaj grę", spec.Name)

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if loaded != spec.Name {
		t.Fatalf("loaded %q, want %q", loaded, spec.Name)
	}

	if cmd == nil {
		t.Error("the loaded game was not told the size of the terminal")
	}

	if model.(Game).pause.open {
		t.Error("the loaded game should start without the menu")
	}
}

func TestPauseSettingsSwitchVoice(t *testing.T) {
	settings := &stubSettings{options: speech.Options{Engine: speech.EngineChatterbox}}
	model := pausedGame(t, Pause{Settings: settings}, nil)

	// Pauza: Wróć do gry, Ustawienia, Wyjdź z gry.
	model = press(model, tea.KeyEsc, tea.KeyDown, tea.KeyEnter)
	assertShows(t, model, "settings", "Ustawienia: głos", "Chatterbox", "automatycznie")

	// Chatterbox → Piper shows the piper rows.
	model = press(model, tea.KeyRight)
	assertShows(t, model, "piper", "Piper", "Tempo mowy", "Głos postaci", "gosia", "Głos narratora", "jak postaci")

	// Faster, then the narrator gets a voice of its own.
	model = press(model, tea.KeyDown, tea.KeyRight, tea.KeyRight, tea.KeyDown, tea.KeyDown, tea.KeyRight)
	assertShows(t, model, "changed", "1.2×", "darkman")

	// Down to "Zastosuj i posłuchaj".
	model = press(model, tea.KeyDown, tea.KeyEnter)
	if len(settings.applied) != 1 {
		t.Fatalf("applied %d times", len(settings.applied))
	}

	got := settings.applied[0]
	if got.Engine != speech.EnginePiper || got.Speed != 1.2 || got.Narrator != "pl_PL-darkman-medium.onnx" {
		t.Errorf("applied %+v", got)
	}

	assertShows(t, model, "applied", "Zapisane.")
}
