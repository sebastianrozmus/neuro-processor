package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFallsBackToDataDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	for _, key := range []string{"NEURO_LLM_MODEL", "NEURO_TALK_MODEL"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}

	data := filepath.Join(dir, "neuro-processor")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(path, text string) {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(data, localEnvFile), "NEURO_LLM_MODEL=gemma3:4b\nNEURO_TALK_MODEL=neuro-npc\n")

	work := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(previous) })

	if cfg := Load(); cfg.Model != "gemma3:4b" || cfg.TalkModel != "neuro-npc" {
		t.Errorf("data directory settings not read: model %q, talk %q", cfg.Model, cfg.TalkModel)
	}

	write(filepath.Join(work, localEnvFile), "NEURO_TALK_MODEL=local\n")
	if cfg := Load(); cfg.Model != "gemma3:4b" || cfg.TalkModel != "local" {
		t.Errorf("working directory should win: model %q, talk %q", cfg.Model, cfg.TalkModel)
	}
}
