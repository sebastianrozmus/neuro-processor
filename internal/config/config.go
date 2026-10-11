// Package config reads the game settings from the environment.
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

const (
	localEnvFile   = ".env.local"
	defaultBaseURL = "https://api.openai.com/v1"
)

// Config holds the language backend settings.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	// MaxTokens caps the length of spoken replies; 0 leaves it to the server.
	MaxTokens int
	// TalkBaseURL, TalkAPIKey and TalkModel name a second model that only
	// words the characters' lines (the game's own fine-tuned one); the
	// main model keeps reading intents and generating worlds. Empty
	// TalkModel means one model does everything.
	TalkBaseURL string
	TalkAPIKey  string
	TalkModel   string
	// Context is the window, in tokens, a local Ollama is asked to run the
	// main model in, which it otherwise does in 4096, too few to design a
	// world; TalkContext the same for the talking model. 0 keeps the
	// server's own setting.
	Context     int
	TalkContext int
	// Speech turns reading the characters' lines aloud on; Voice picks the
	// system voice, empty meaning the Polish one; SpeechCommand, when set,
	// is a shell command that gets each line on its input instead.
	Speech        bool
	Voice         string
	SpeechCommand string
	// Hearing turns the microphone (push-to-talk) off when false.
	Hearing bool
	// Talks turns keeping a record of the model's conversations on, as
	// material for training a model of the game's own.
	Talks bool
	// Python is the interpreter of the environment with the image model
	// that draws avatars; empty means the one tools/avatar/setup.sh makes.
	Python string
	// Blocks picks the characters generated pictures are drawn with: half
	// (1x2 pixels per cell), quad (2x2) or sextant (2x3); TrueColor sends
	// their colours as 24-bit values rather than the nearest of 256.
	Blocks    string
	TrueColor bool
	// WorldArt turns painting pictures for generated worlds (backdrops,
	// scenes, keepers) with the image model on; it needs Python.
	WorldArt bool
	// Images shows the painted pictures as real images in terminals that
	// speak the kitty graphics protocol: "off" (the default), "auto" or
	// "kitty" (forced).
	Images string
	// Inspector is the address the browser inspector of the running
	// world listens on; empty keeps it off.
	Inspector string
}

// Offline reports whether no model is configured, in which case the game
// runs on the built-in rule-based backend.
func (c Config) Offline() bool {
	return c.Model == ""
}

// Load reads the settings from the environment. Variables missing from the
// environment are taken from .env.local in the working directory, if any.
func Load() Config {
	local := readEnvFile(localEnvFile)

	get := func(key, fallback string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}

		if value, ok := local[key]; ok {
			return value
		}

		return fallback
	}

	// A value that is not a number means no cap.
	maxTokens, _ := strconv.Atoi(get("NEURO_LLM_MAX_TOKENS", ""))
	window, _ := strconv.Atoi(get("NEURO_LLM_CONTEXT", "8192"))
	talkWindow, _ := strconv.Atoi(get("NEURO_TALK_CONTEXT", ""))

	return Config{
		BaseURL:       get("NEURO_LLM_BASE_URL", defaultBaseURL),
		APIKey:        get("NEURO_LLM_API_KEY", ""),
		Model:         get("NEURO_LLM_MODEL", ""),
		MaxTokens:     max(maxTokens, 0),
		TalkBaseURL:   get("NEURO_TALK_BASE_URL", get("NEURO_LLM_BASE_URL", defaultBaseURL)),
		TalkAPIKey:    get("NEURO_TALK_API_KEY", get("NEURO_LLM_API_KEY", "")),
		TalkModel:     get("NEURO_TALK_MODEL", ""),
		Context:       max(window, 0),
		TalkContext:   max(talkWindow, 0),
		Speech:        isOn(get("NEURO_TTS", "")),
		Voice:         get("NEURO_TTS_VOICE", ""),
		SpeechCommand: get("NEURO_TTS_COMMAND", ""),
		Hearing:       !isOff(get("NEURO_STT", "")),
		Talks:         isOn(get("NEURO_LOG_TALKS", "")),
		Python:        get("NEURO_AVATAR_PYTHON", ""),
		Blocks:        blocks(get("NEURO_BLOCKS", "")),
		TrueColor:     !isOff(get("NEURO_TRUECOLOR", "")),
		WorldArt:      !isOff(get("NEURO_WORLD_ART", "")),
		Images:        images(get("NEURO_IMAGES", "")),
		Inspector:     inspector(get("NEURO_DEBUG", "")),
	}
}

// blocks reads the block-character setting; anything unknown means quad.
func images(value string) string {
	switch value = strings.ToLower(strings.TrimSpace(value)); value {
	case "auto", "kitty":
		return value
	case "1", "true", "yes", "tak", "on":
		return "auto"
	}

	return "off"
}

func blocks(value string) string {
	switch value = strings.ToLower(strings.TrimSpace(value)); value {
	case "half", "sextant":
		return value
	}

	return "quad"
}

func readEnvFile(path string) map[string]string {
	values := map[string]string{}

	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}

	return values
}

// isOn reads a yes-or-no setting.
func isOn(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "tak", "on":
		return true
	}

	return false
}

// isOff reads a setting that is on unless switched off.
func isOff(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "nie", "off":
		return true
	}

	return false
}

// inspector reads the inspector setting: off by default, "on" (or "1",
// "true") for the local default address, or an address of its own.
func inspector(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "off", "false", "no":
		return ""
	case "1", "on", "true", "yes":
		return "127.0.0.1:7788"
	}

	return strings.TrimSpace(value)
}
