package speech

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakeBox answers like the Chatterbox server and remembers what it was asked.
type fakeBox struct {
	mu       sync.Mutex
	requests []SpeakRequest
}

func (f *fakeBox) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /speak", func(w http.ResponseWriter, r *http.Request) {
		var request SpeakRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		f.mu.Lock()
		f.requests = append(f.requests, request)
		f.mu.Unlock()

		if request.Text == "" {
			http.Error(w, `{"error":"pusty tekst"}`, http.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF....WAVE"))
	})
	mux.HandleFunc("GET /voices", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"voices":["kowal.wav"]}`))
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	return mux
}

func TestVoiceSettingsPickVoicesAndMoods(t *testing.T) {
	path := filepath.Join(t.TempDir(), "voices.json")
	settings := LoadVoiceSettings(path)
	if settings.Language != "pl" || settings.Expression(Anger) != DefaultExpressions[Anger] || settings.VoiceFor("x", "y") != "" {
		t.Fatalf("defaults = %+v", settings)
	}

	settings.Default = "ogolny.wav"
	settings.Narrator = "narrator.wav"
	settings.ByKind["armory"] = "kowal.wav"
	settings.ByName["Borin"] = "krasnolud.wav"
	settings.Moods[Mockery] = Expression{0.6, 0.45, 0.8}
	if err := SaveVoiceSettings(path, settings); err != nil {
		t.Fatal(err)
	}

	loaded := LoadVoiceSettings(path)
	for who, want := range map[[2]string]string{{"Borin", "armory"}: "krasnolud.wav", {"Kowal", "armory"}: "kowal.wav", {"Ktoś", "magic"}: "ogolny.wav", {"", "narrator"}: "narrator.wav"} {
		if got := loaded.VoiceFor(who[0], who[1]); got != want {
			t.Errorf("VoiceFor(%q, %q) = %q, want %q", who[0], who[1], got, want)
		}
	}

	if loaded.Expression(Mockery) != (Expression{0.6, 0.45, 0.8}) || loaded.Expression(Joy) != DefaultExpressions[Joy] || loaded.Expression("nieznany") != DefaultExpressions[Calm] {
		t.Errorf("expressions = %+v", loaded.Moods)
	}
}

func TestChatterboxSpeaksInTheMoodWithTheRightVoice(t *testing.T) {
	box := &fakeBox{}
	web := httptest.NewServer(box.handler())
	defer web.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "voices.json")
	_ = SaveVoiceSettings(path, VoiceSettings{Language: "pl", Default: "ogolny.wav", ByName: map[string]string{"Borin": "krasnolud.wav"}})

	speaker := NewChatterbox(web.URL, path)
	speaker.startAs = (&Chatterbox{URL: web.URL, Settings: path, Player: "true", Dir: dir}).start
	speaker.SayAs("Borin", "wojownik", "[gniew] Dwie korony? Za topór?", Anger)

	deadline := time.Now().Add(5 * time.Second)
	for {
		box.mu.Lock()
		count := len(box.requests)
		box.mu.Unlock()

		if count > 0 || time.Now().After(deadline) {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	box.mu.Lock()
	requests := append([]SpeakRequest(nil), box.requests...)
	box.mu.Unlock()

	if len(requests) != 1 {
		t.Fatalf("requests = %+v", requests)
	}

	got := requests[0]
	if got.Voice != "krasnolud.wav" || got.Language != "pl" || got.Exaggeration != DefaultExpressions[Anger].Exaggeration || got.CFG != DefaultExpressions[Anger].CFG || got.Text != "gniew Dwie korony? Za topór?" && got.Text != "Dwie korony? Za topór?" {
		t.Errorf("request = %+v", got)
	}

	// A sound file was written for the player and taken away afterwards.
	time.Sleep(100 * time.Millisecond)
	if left, _ := filepath.Glob(filepath.Join(dir, "neuro-line-*.wav")); len(left) != 0 {
		t.Errorf("sound files left behind: %v", left)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if voices, err := Voices(ctx, web.URL); err != nil || len(voices) != 1 || voices[0] != "kowal.wav" {
		t.Errorf("voices = %v, %v", voices, err)
	}

	if _, err := Synthesize(ctx, web.URL, SpeakRequest{}); err == nil || err.Error() != "pusty tekst" {
		t.Errorf("an empty line gave %v", err)
	}

	server := &Server{Python: filepath.Join(dir, "none"), URL: web.URL}
	if server.Available() || !server.Running() {
		t.Error("server availability is off")
	}

	if err := os.WriteFile(filepath.Join(dir, "none"), nil, 0o755); err == nil && !server.Available() {
		t.Error("an existing python is not available")
	}
}
