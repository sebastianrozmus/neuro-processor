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

	return Config{
		BaseURL:       get("NEURO_LLM_BASE_URL", defaultBaseURL),
		APIKey:        get("NEURO_LLM_API_KEY", ""),
		Model:         get("NEURO_LLM_MODEL", ""),
		MaxTokens:     max(maxTokens, 0),
		TalkBaseURL:   get("NEURO_TALK_BASE_URL", get("NEURO_LLM_BASE_URL", defaultBaseURL)),
		TalkAPIKey:    get("NEURO_TALK_API_KEY", get("NEURO_LLM_API_KEY", "")),
		TalkModel:     get("NEURO_TALK_MODEL", ""),
		Speech:        isOn(get("NEURO_TTS", "")),
		Voice:         get("NEURO_TTS_VOICE", ""),
		SpeechCommand: get("NEURO_TTS_COMMAND", ""),
		Hearing:       !isOff(get("NEURO_STT", "")),
		Talks:         isOn(get("NEURO_LOG_TALKS", "")),
		Python:        get("NEURO_AVATAR_PYTHON", ""),
	}
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
