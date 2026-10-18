package settings

import (
	"path/filepath"
	"testing"
)

func TestKeysAreKeptApart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	var missing map[string]string
	if Load(path, "models", &missing) {
		t.Fatal("loaded from a file that does not exist")
	}

	if err := Save(path, "voice", map[string]string{"engine": "piper"}); err != nil {
		t.Fatal(err)
	}

	if err := Save(path, "models", map[string]string{"umysł": "gemma3:4b"}); err != nil {
		t.Fatal(err)
	}

	var voice, models map[string]string
	if !Load(path, "voice", &voice) || voice["engine"] != "piper" {
		t.Errorf("voice %v", voice)
	}

	if !Load(path, "models", &models) || models["umysł"] != "gemma3:4b" {
		t.Errorf("models %v", models)
	}
}
