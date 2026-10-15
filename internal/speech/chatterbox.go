package speech

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// windowsPlayer plays a WAV through .NET's SoundPlayer: Windows has no
// afplay, and PowerShell is always there. Killing it stops the sound.
const windowsPlayer = "powershell"

// Chatterbox (Resemble AI, MIT) speaks Polish with a cloned voice and a
// dial for how expressive it is, so the moods a narrator tags its lines
// with can be heard. It runs as a small HTTP server in a Python
// environment of its own (tools/speech/setup-tts.sh); the game starts it,
// asks it for every line as WAV, and plays the WAV.

//go:embed chatterbox.py
var chatterboxScript string

const (
	DefaultChatterboxURL = "http://127.0.0.1:7790"
	ttsTimeout           = 120 * time.Second
	settingsFile         = "voices.json"
)

// Expression is how a mood is rendered: how far the voice is pushed
// (exaggeration, 0.25–1), how closely it follows the prompt (cfg weight;
// lower is slower and more dramatic) and how much it varies (temperature).
type Expression struct {
	Exaggeration float64 `json:"exaggeration"`
	CFG          float64 `json:"cfg_weight"`
	Temperature  float64 `json:"temperature"`
}

// DefaultExpressions is how each mood sounds unless the settings say
// otherwise. Chatterbox grows unstable above about 0.9 exaggeration.
var DefaultExpressions = map[Emotion]Expression{
	Calm: {0.45, 0.5, 0.8}, Joy: {0.7, 0.4, 0.8}, Anger: {0.9, 0.3, 0.8}, Sadness: {0.4, 0.6, 0.7},
	Fear: {0.8, 0.3, 0.9}, Mockery: {0.65, 0.4, 0.8}, Surprise: {0.8, 0.35, 0.9}, Tenderness: {0.5, 0.55, 0.7},
	"znużenie": {0.3, 0.6, 0.7},
}

// VoiceSettings say who speaks with which sample and how the moods sound.
// They live in voices.json next to the samples and may be changed while
// the game runs; the inspector edits them.
type VoiceSettings struct {
	// Language is the Chatterbox language code of the lines.
	Language string `json:"language"`
	// Default is the sample used when nothing more specific is set;
	// Narrator the one for the game master's lines.
	Default  string `json:"default"`
	Narrator string `json:"narrator,omitempty"`
	// ByName assigns samples to characters by name; ByKind to kinds of
	// shop (armory, alchemist, magic, caravan) and roles (wojownik,
	// uzdrowiciel, mag).
	ByName map[string]string `json:"by_name"`
	ByKind map[string]string `json:"by_kind"`
	// Moods overrides how moods are rendered.
	Moods map[Emotion]Expression `json:"moods"`
}

// LoadVoiceSettings reads the settings, with defaults for what is missing.
func LoadVoiceSettings(path string) VoiceSettings {
	settings := VoiceSettings{Language: "pl", ByName: map[string]string{}, ByKind: map[string]string{}, Moods: map[Emotion]Expression{}}

	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}

	if settings.Language == "" {
		settings.Language = "pl"
	}

	if settings.ByName == nil {
		settings.ByName = map[string]string{}
	}

	if settings.ByKind == nil {
		settings.ByKind = map[string]string{}
	}

	if settings.Moods == nil {
		settings.Moods = map[Emotion]Expression{}
	}

	return settings
}

// SaveVoiceSettings writes the settings.
func SaveVoiceSettings(path string, settings VoiceSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// Expression returns how a mood sounds under these settings.
func (s VoiceSettings) Expression(mood Emotion) Expression {
	if how, ok := s.Moods[mood]; ok {
		return how
	}

	if how, ok := DefaultExpressions[mood]; ok {
		return how
	}

	return DefaultExpressions[Calm]
}

// VoiceFor picks the sample for a speaker: by name, by kind, or the default.
func (s VoiceSettings) VoiceFor(name, kind string) string {
	if voice, ok := s.ByName[name]; ok && voice != "" {
		return voice
	}

	if voice, ok := s.ByKind[kind]; ok && voice != "" {
		return voice
	}

	if kind == "narrator" && s.Narrator != "" {
		return s.Narrator
	}

	return s.Default
}

// SpeakRequest is one line for the server.
type SpeakRequest struct {
	Text         string  `json:"text"`
	Language     string  `json:"language"`
	Voice        string  `json:"voice,omitempty"`
	Exaggeration float64 `json:"exaggeration"`
	CFG          float64 `json:"cfg_weight"`
	Temperature  float64 `json:"temperature"`
}

// Synthesize asks a Chatterbox server for a line and returns the WAV.
func Synthesize(ctx context.Context, base string, request SpeakRequest) ([]byte, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	call, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/speak", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	call.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(call)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		var problem struct {
			Error string `json:"error"`
		}

		_ = json.Unmarshal(data, &problem)
		if problem.Error == "" {
			problem.Error = response.Status
		}

		return nil, errors.New(problem.Error)
	}

	return data, nil
}

// Voices lists the samples a Chatterbox server has.
func Voices(ctx context.Context, base string) ([]string, error) {
	call, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/voices", nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(call)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var listed struct {
		Voices []string `json:"voices"`
	}

	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		return nil, err
	}

	return listed.Voices, nil
}

// Chatterbox is a Speaker on a Chatterbox server: each sentence is fetched
// as WAV in the mood's expression with the speaker's sample, and played.
type Chatterbox struct {
	URL string
	// Settings is the path of voices.json, read before every line so that
	// changes in the inspector are heard at once.
	Settings string
	// Player is the command that plays a WAV file given as its argument;
	// afplay on macOS.
	Player string
	// Dir holds the temporary WAV files.
	Dir string
}

// NewChatterbox returns a speaker that talks through a Chatterbox server.
func NewChatterbox(base, settings string) *Speaker {
	if base == "" {
		base = DefaultChatterboxURL
	}

	player := "afplay"
	if runtime.GOOS == "windows" {
		player = windowsPlayer
	} else if _, err := exec.LookPath(player); err != nil {
		player = "aplay"
	}

	box := &Chatterbox{URL: base, Settings: settings, Player: player, Dir: os.TempDir()}

	return &Speaker{
		start:   func(text string, mood Emotion) func() { return box.start("", "", text, mood) },
		startAs: box.start,
		queueAs: box.queue,
	}
}

// start speaks one whole line, sentence by sentence, and returns what
// stops it.
func (b *Chatterbox) start(name, kind, text string, mood Emotion) func() {
	add, stop := b.queue(name, kind)

	chunks, _ := Sentences(text, true)
	for _, chunk := range chunks {
		add(chunk, mood)
	}

	return stop
}

// queued is a sentence waiting for the voice.
type queued struct {
	text string
	mood Emotion
}

// queue starts a line whose sentences come one by one: each is fetched as
// soon as the one before it is, and played when the one before it ends, so
// the next sentence is being synthesized while the previous one is heard.
// It returns what adds a sentence and what stops it all: the fetches are
// abandoned and the player killed.
func (b *Chatterbox) queue(name, kind string) (func(text string, mood Emotion), func()) {
	ctx, cancel := context.WithCancel(context.Background())
	lines := make(chan queued, 32)
	sounds := make(chan string, 32)

	var (
		mu     sync.Mutex
		cmd    *exec.Cmd
		closed bool
	)

	go func() {
		defer close(sounds)

		for line := range lines {
			if ctx.Err() != nil {
				return
			}

			path, err := b.fetch(ctx, name, kind, line)
			if err != nil {
				continue
			}

			sounds <- path
		}
	}()

	go func() {
		for path := range sounds {
			mu.Lock()
			if ctx.Err() == nil {
				cmd = b.play(ctx, path)
				if cmd.Start() != nil {
					cmd = nil
				}
			}
			playing := cmd
			mu.Unlock()

			if playing != nil {
				_ = playing.Wait()
			}

			_ = os.Remove(path)
		}
	}()

	add := func(text string, mood Emotion) {
		mu.Lock()
		defer mu.Unlock()

		if closed {
			return
		}

		select {
		case lines <- queued{text, mood}:
		default: // a line this long is not worth hearing to the end
		}
	}

	stop := func() {
		mu.Lock()
		defer mu.Unlock()

		cancel()
		if !closed {
			closed = true
			close(lines)
		}

		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}

	return add, stop
}

// fetch synthesizes one sentence into a WAV file and returns its path.
func (b *Chatterbox) fetch(ctx context.Context, name, kind string, line queued) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ttsTimeout)
	defer cancel()

	settings := LoadVoiceSettings(b.Settings)
	how := settings.Expression(line.mood)
	data, err := Synthesize(ctx, b.URL, SpeakRequest{
		Text: line.text, Language: settings.Language, Voice: settings.VoiceFor(name, kind),
		Exaggeration: how.Exaggeration, CFG: how.CFG, Temperature: how.Temperature,
	})
	if err != nil {
		return "", err
	}

	path := filepath.Join(b.Dir, fmt.Sprintf("neuro-line-%d.wav", time.Now().UnixNano()))

	return path, os.WriteFile(path, data, 0o644)
}

// play is the command that plays the WAV at path.
func (b *Chatterbox) play(ctx context.Context, path string) *exec.Cmd {
	if b.Player == windowsPlayer {
		script := "(New-Object Media.SoundPlayer '" + strings.ReplaceAll(path, "'", "''") + "').PlaySync()"

		return exec.CommandContext(ctx, b.Player, "-NoProfile", "-NonInteractive", "-Command", script)
	}

	return exec.CommandContext(ctx, b.Player, path)
}

// VenvPython is the interpreter of the Python environment in dir:
// Scripts\python.exe on Windows, bin/python3 elsewhere.
func VenvPython(dir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, "Scripts", "python.exe")
	}

	return filepath.Join(dir, "bin", "python3")
}

// Server runs the embedded Chatterbox server in the TTS environment.
type Server struct {
	Python string
	Voices string
	URL    string
	Log    string

	cmd *exec.Cmd
}

// Available reports whether the TTS environment exists.
func (s *Server) Available() bool {
	info, err := os.Stat(s.Python)

	return err == nil && !info.IsDir()
}

// Running reports whether something already answers at the URL.
func (s *Server) Running() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	call, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.URL, "/")+"/health", nil)
	if err != nil {
		return false
	}

	response, err := http.DefaultClient.Do(call)
	if err != nil {
		return false
	}

	response.Body.Close()

	return response.StatusCode == http.StatusOK
}

// Start launches the server unless one already answers. It returns at
// once; the model loads in the background and lines before that are
// dropped.
func (s *Server) Start() error {
	if s.Running() {
		return nil
	}

	parsed, err := url.Parse(s.URL)
	if err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.Voices, 0o755); err != nil {
		return err
	}

	cmd := exec.Command(s.Python, "-", "--host", host, "--port", port, "--voices", s.Voices)
	cmd.Stdin = strings.NewReader(chatterboxScript)

	if s.Log != "" {
		if file, err := os.OpenFile(s.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			cmd.Stdout, cmd.Stderr = file, file
		}
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	s.cmd = cmd

	go func() { _ = cmd.Wait() }()

	return nil
}

// Stop ends the server the game started.
func (s *Server) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}
