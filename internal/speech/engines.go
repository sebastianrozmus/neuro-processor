package speech

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// The voices a player can pick in the game's settings.
const (
	EngineChatterbox = "chatterbox"
	EnginePiper      = "piper"
	EngineSystem     = "system"
	EngineOff        = "off"
)

// Options are the player's choice of voice. They are kept in the game's
// settings and can be changed while it runs.
type Options struct {
	Engine string `json:"engine"`
	// Speed above 1 speaks faster (piper and the system voice).
	Speed float64 `json:"speed,omitempty"`
	// Voice is the piper voice or the system voice; Narrator the piper
	// voice of the game master.
	Voice    string `json:"voice,omitempty"`
	Narrator string `json:"narrator,omitempty"`
	// Device is where Chatterbox runs: auto, cuda or cpu.
	Device string `json:"device,omitempty"`
}

// Choices are what the player can pick from on this machine.
type Choices struct {
	Engines      []string
	PiperVoices  []string
	SystemVoices []string
}

// Engines is the game's voice: it speaks with whichever engine the
// options name and switches engines while the game runs. Switching away
// from Chatterbox stops its server, which frees the graphics card.
type Engines struct {
	// TTSPython is the interpreter of the Chatterbox environment, Samples
	// its folder of voice samples, Settings its voices.json, URL where it
	// listens and Log where it writes.
	TTSPython string
	Samples   string
	Settings  string
	URL       string
	Log       string
	// PiperDir holds piper and its voices (see PiperExe).
	PiperDir string
	// Command, when set, is a command that speaks instead of the system
	// voice (NEURO_TTS_COMMAND).
	Command string

	mu      sync.Mutex
	options Options
	speaker *Speaker
	server  *Server
	piper   *Piper
}

// Choices lists what this machine can speak with.
func (e *Engines) Choices() Choices {
	choices := Choices{Engines: []string{EngineOff}}

	if info, err := os.Stat(e.TTSPython); err == nil && !info.IsDir() {
		choices.Engines = append(choices.Engines, EngineChatterbox)
	}

	if _, err := os.Stat(PiperExe(e.PiperDir)); err == nil {
		choices.PiperVoices = PiperVoices(e.PiperDir)
		if len(choices.PiperVoices) > 0 {
			choices.Engines = append(choices.Engines, EnginePiper)
		}
	}

	choices.SystemVoices = SystemVoices()
	if len(choices.SystemVoices) > 0 || e.Command != "" {
		choices.Engines = append(choices.Engines, EngineSystem)
	}

	return choices
}

// Options returns the options in use.
func (e *Engines) Options() Options {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.options
}

// ErrNoEngine reports an engine this machine does not have.
var ErrNoEngine = errors.New("ten głos nie jest zainstalowany")

// Apply switches to the engine of the options. The old engine stops; a
// Chatterbox server the game started is shut down.
func (e *Engines) Apply(options Options) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var (
		speaker *Speaker
		server  *Server
		piper   *Piper
	)

	switch options.Engine {
	case EngineOff, "":
	case EngineChatterbox:
		server = &Server{Python: e.TTSPython, Voices: e.Samples, URL: e.URL, Log: e.Log, Device: options.Device}
		if !server.Available() && !server.Running() {
			return ErrNoEngine
		}

		// The same server is kept when only the voice's mood settings
		// could have changed.
		if e.server != nil && e.options.Engine == EngineChatterbox && e.options.Device == options.Device {
			server = e.server
		} else {
			e.stopServer()
			if err := server.Start(); err != nil {
				return err
			}
		}

		speaker = NewChatterbox(e.URL, e.Settings)
	case EnginePiper:
		voices := PiperVoices(e.PiperDir)
		if len(voices) == 0 {
			return ErrNoEngine
		}

		if !contains(voices, options.Voice) {
			options.Voice = preferred(voices)
		}

		if options.Narrator != "" && !contains(voices, options.Narrator) {
			options.Narrator = ""
		}

		piper = &Piper{Exe: PiperExe(e.PiperDir), Voices: filepath.Join(e.PiperDir, "voices"), Voice: options.Voice, Narrator: options.Narrator, Speed: options.Speed}
		speaker = NewPiper(piper)
	case EngineSystem:
		if e.Command != "" {
			speaker = NewCommand(e.Command)
		} else {
			speaker = NewSystem(options.Voice, options.Speed)
		}

		if speaker == nil {
			return ErrNoEngine
		}
	default:
		return ErrNoEngine
	}

	if e.speaker != nil {
		e.speaker.Stop()
	}

	if e.piper != nil {
		e.piper.Close()
	}

	if server == nil {
		e.stopServer()
	}

	e.options, e.speaker, e.server, e.piper = options, speaker, server, piper

	return nil
}

// stopServer shuts down the Chatterbox server the game started.
func (e *Engines) stopServer() {
	if e.server != nil {
		e.server.Stop()
		e.server = nil
	}
}

// Close stops whatever speaks.
func (e *Engines) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.speaker != nil {
		e.speaker.Stop()
	}

	if e.piper != nil {
		e.piper.Close()
	}

	e.stopServer()
}

func (e *Engines) current() *Speaker {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.speaker
}

// Say reads a line aloud with the current engine.
func (e *Engines) Say(text string, mood Emotion) {
	e.current().Say(text, mood)
}

// SayAs reads a line as a named character with the current engine.
func (e *Engines) SayAs(name, kind, text string, mood Emotion) {
	e.current().SayAs(name, kind, text, mood)
}

// StreamAs speaks a line while it is written, with the current engine.
func (e *Engines) StreamAs(name, kind string) *Stream {
	return e.current().StreamAs(name, kind)
}

// Chatterbox reports whether the current engine is Chatterbox.
func (e *Engines) Chatterbox() bool {
	return e.Options().Engine == EngineChatterbox
}

func contains(list []string, wanted string) bool {
	for _, item := range list {
		if item == wanted {
			return true
		}
	}

	return false
}

// preferred is the voice used when none is picked: gosia, which reads
// Polish most clearly, or the first one.
func preferred(voices []string) string {
	for _, voice := range voices {
		if PiperLabel(voice) == "gosia" {
			return voice
		}
	}

	return voices[0]
}

// LoadOptions reads the voice options from the game's settings file;
// fallback is used when there are none.
func LoadOptions(path string, fallback Options) Options {
	var settings struct {
		Voice *Options `json:"voice"`
	}

	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &settings) != nil || settings.Voice == nil {
		return fallback
	}

	return *settings.Voice
}

// SaveOptions writes the voice options into the game's settings file,
// keeping whatever else is there.
func SaveOptions(path string, options Options) error {
	settings := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}

	settings["voice"] = options

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
