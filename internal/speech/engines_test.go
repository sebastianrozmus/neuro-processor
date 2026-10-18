package speech

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionsKeepOtherSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"other": 7}`), 0o644); err != nil {
		t.Fatal(err)
	}

	fallback := Options{Engine: EngineSystem}
	if got := LoadOptions(path, fallback); got != fallback {
		t.Errorf("without voice options: %+v", got)
	}

	want := Options{Engine: EnginePiper, Speed: 1.2, Voice: "pl_PL-gosia-medium.onnx", Narrator: "pl_PL-darkman-medium.onnx"}
	if err := SaveOptions(path, want); err != nil {
		t.Fatal(err)
	}

	if got := LoadOptions(path, fallback); got != want {
		t.Errorf("read back %+v, want %+v", got, want)
	}

	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `"other": 7`) {
		t.Errorf("other settings lost: %s", data)
	}
}

func TestEnginesRefuseWhatIsNotInstalled(t *testing.T) {
	dir := t.TempDir()
	engines := &Engines{TTSPython: filepath.Join(dir, "none"), PiperDir: filepath.Join(dir, "piper"), URL: "http://127.0.0.1:1"}

	for _, engine := range []string{EnginePiper, EngineChatterbox, "espeak"} {
		if err := engines.Apply(Options{Engine: engine}); !errors.Is(err, ErrNoEngine) {
			t.Errorf("%s: err = %v", engine, err)
		}
	}

	if err := engines.Apply(Options{Engine: EngineOff}); err != nil {
		t.Fatal(err)
	}

	// Silent, but safe to speak to.
	engines.Say("cisza", Calm)
	engines.StreamAs("Borin", "wojownik").Close("[spokój] cisza")

	for _, engine := range engines.Choices().Engines {
		if engine == EnginePiper || engine == EngineChatterbox {
			t.Errorf("%s offered, but it is not installed", engine)
		}
	}
}

func TestPiperVoicesAndLabels(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "voices"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"pl_PL-mc_speech-medium.onnx", "pl_PL-gosia-medium.onnx", "pl_PL-gosia-medium.onnx.json"} {
		if err := os.WriteFile(filepath.Join(dir, "voices", name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	voices := PiperVoices(dir)
	if len(voices) != 2 || voices[0] != "pl_PL-gosia-medium.onnx" {
		t.Fatalf("voices %v", voices)
	}

	if got := PiperLabel(voices[1]); got != "mc_speech" {
		t.Errorf("label %q", got)
	}

	if got := preferred([]string{"pl_PL-darkman-medium.onnx", "pl_PL-gosia-medium.onnx"}); got != "pl_PL-gosia-medium.onnx" {
		t.Errorf("preferred %q", got)
	}
}

// TestPiperForReal runs the real piper when NEURO_PIPER_DIR names a folder
// made by setup-piper: two sentences through one process.
func TestPiperForReal(t *testing.T) {
	dir := os.Getenv("NEURO_PIPER_DIR")
	if dir == "" {
		t.Skip("NEURO_PIPER_DIR not set")
	}

	voices := PiperVoices(dir)
	if len(voices) == 0 {
		t.Fatal("no voices in", dir)
	}

	piper := &Piper{Exe: PiperExe(dir), Voices: filepath.Join(dir, "voices"), Voice: preferred(voices), Speed: 1.1, Dir: t.TempDir()}
	defer piper.Close()

	for _, text := range []string{"Za 62 plomby płaszcz jest twój.", "Zapnij kaptur przed wyjściem, wędrowcze."} {
		path, err := piper.fetch(t.Context(), "", text)
		if err != nil {
			t.Fatal(err)
		}

		if info, err := os.Stat(path); err != nil || info.Size() < 10000 {
			t.Errorf("%s: %v", path, err)
		}
	}
}
