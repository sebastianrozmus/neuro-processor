// Package settings keeps what the player picks in the game's menu
// (Esc → Ustawienia): the voice and the models. They live in one JSON file
// in the game's data directory, each under its own key, and win over the
// NEURO_* variables, which only say what to start with.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Load reads the value under key into into, reporting whether there was
// one.
func Load(path, key string, into any) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var all map[string]json.RawMessage
	if json.Unmarshal(data, &all) != nil || all[key] == nil {
		return false
	}

	return json.Unmarshal(all[key], into) == nil
}

// Save writes a value under key, keeping the other keys.
func Save(path, key string, value any) error {
	all := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &all)
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	all[key] = raw

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
