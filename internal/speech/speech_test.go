package speech

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpeakerCutsThePreviousLineShort(t *testing.T) {
	var started []string
	stopped := 0

	speaker := &Speaker{start: func(text string, mood Emotion) func() {
		started = append(started, string(mood)+": "+text)

		return func() { stopped++ }
	}}

	speaker.Say("Witaj, «wędrowcze».", Calm)
	speaker.Say("   ", Joy)
	speaker.Say("\x1b[1mDaj 140 — i po sprawie.\x1b[0m", Anger)

	want := []string{"spokój: Witaj, wędrowcze.", "gniew: Daj 140 , i po sprawie."}
	if len(started) != 2 || started[0] != want[0] || started[1] != want[1] {
		t.Errorf("started = %q, want %q", started, want)
	}

	if stopped != 1 {
		t.Errorf("the first line was stopped %d times, want 1", stopped)
	}

	var silent *Speaker
	silent.Say("nic", Calm)
}

func TestSplitMood(t *testing.T) {
	for text, want := range map[string][2]string{
		"[kpina] Tyle za rurę?":   {"kpina", "Tyle za rurę?"},
		"  [GNIEW]  Won stąd!":    {"gniew", "Won stąd!"},
		"[nuda] Bez emocji.":      {"spokój", "Bez emocji."},
		"Zwykłe zdanie.":          {"spokój", "Zwykłe zdanie."},
		"[bez zamknięcia nawiasu": {"spokój", "[bez zamknięcia nawiasu"},
	} {
		mood, rest := SplitMood(text)
		if string(mood) != want[0] || rest != want[1] {
			t.Errorf("SplitMood(%q) = %q, %q; want %q, %q", text, mood, rest, want[0], want[1])
		}
	}
}

func TestCommandSpeakerPipesTheLine(t *testing.T) {
	speaker := NewCommand("cat > /dev/null")
	if speaker == nil {
		t.Fatal("no speaker for a command")
	}

	speaker.Say("próba", Mockery)

	if NewCommand("   ") != nil {
		t.Error("an empty command should give no speaker")
	}
}

// fakeRecorder stands in for the Python interpreter: it behaves like the
// recorder script, speaking the given word when stopped.
func fakeRecorder(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "python3")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestListenerWaitsUntilRecordingBeforeItCanBeStopped(t *testing.T) {
	ear := &Listener{
		Python: fakeRecorder(t, `trap 'echo "ile za kord" > "$3"; exit 0' INT
sleep 0.3
echo recording
while :; do sleep 0.05; done`),
		Dir: t.TempDir(),
	}

	if !ear.Available() || ear.Recording() {
		t.Fatal("the listener is not available or already recording")
	}

	started := time.Now()
	if err := ear.Start(); err != nil {
		t.Fatal(err)
	}

	// Start came back only once the recorder was ready, so an immediate stop is safe.
	if time.Since(started) < 250*time.Millisecond || !ear.Recording() {
		t.Error("Start did not wait for the recorder")
	}

	if err := ear.Start(); err == nil {
		t.Error("a second recording started alongside the first")
	}

	heard, err := ear.Stop()
	if err != nil || heard != "ile za kord" || ear.Recording() {
		t.Errorf("heard %q, err %v, recording %v", heard, err, ear.Recording())
	}

	if _, err := ear.Stop(); err == nil {
		t.Error("stopping twice did not fail")
	}
}

func TestListenerReportsARecorderThatDies(t *testing.T) {
	ear := &Listener{Python: fakeRecorder(t, "echo 'listen: brak mikrofonu' >&2\nexit 1"), Dir: t.TempDir()}

	if err := ear.Start(); err == nil || err.Error() != "listen: brak mikrofonu" || ear.Recording() {
		t.Errorf("err = %v, recording %v", err, ear.Recording())
	}
}
