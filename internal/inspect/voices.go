package inspect

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// TTS is the Chatterbox server the inspector lets a person tune: which
// sample speaks for whom, how each mood sounds, and a button to hear it.
type TTS struct {
	URL      string
	Settings string
	Voices   string
}

const (
	sampleLimit = 20 << 20
	sayTimeout  = 120 * time.Second
)

// voicesHandlers adds the voice settings API.
func (s *Server) voicesHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/voices", s.voices)
	mux.HandleFunc("POST /api/voices", s.saveVoices)
	mux.HandleFunc("POST /api/voice", s.uploadVoice)
	mux.HandleFunc("POST /api/say", s.say)
}

// voices answers the settings, the samples on disk, the moods and who
// there is to assign voices to.
func (s *Server) voices(w http.ResponseWriter, _ *http.Request) {
	if s.TTS == nil {
		reply(w, map[string]any{"enabled": false})

		return
	}

	settings := speech.LoadVoiceSettings(s.TTS.Settings)
	moods := map[speech.Emotion]speech.Expression{}
	for mood := range speech.DefaultExpressions {
		moods[mood] = settings.Expression(mood)
	}

	speakers := []map[string]string{}
	if realm := s.watched(); realm != nil {
		report := realm.Inspect()
		for _, keeper := range report.Keepers {
			speakers = append(speakers, map[string]string{"name": keeper.Name, "kind": keeper.Kind, "where": keeper.Building})
		}

		for _, mate := range report.Companions {
			speakers = append(speakers, map[string]string{"name": mate.Name, "kind": mate.Role, "where": "do najęcia"})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := speech.Voices(ctx, s.TTS.URL)
	alive := err == nil

	reply(w, map[string]any{
		"enabled": true, "url": s.TTS.URL, "alive": alive, "settings": settings, "moods": moods,
		"samples": samples(s.TTS.Voices), "speakers": speakers,
		"kinds": []string{string(world.ShopArmory), string(world.ShopAlchemist), string(world.ShopMagic), string(world.ShopCaravan), "wojownik", "uzdrowiciel", "mag", "towarzysz", "narrator"},
	})
}

// samples lists the voice samples in the folder.
func samples(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	var names []string
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if !entry.IsDir() && (strings.HasSuffix(name, ".wav") || strings.HasSuffix(name, ".mp3") || strings.HasSuffix(name, ".flac")) {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	return names
}

// watched returns the realm, if any, without answering the client.
func (s *Server) watched() *game.Realm {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.realm
}

func (s *Server) saveVoices(w http.ResponseWriter, r *http.Request) {
	if s.TTS == nil {
		http.Error(w, "mowa Chatterbox nie jest włączona (NEURO_TTS=chatterbox)", http.StatusNotFound)

		return
	}

	var settings speech.VoiceSettings
	if !decode(w, r, &settings) {
		return
	}

	if err := speech.SaveVoiceSettings(s.TTS.Settings, settings); err != nil {
		http.Error(w, "nie zapisano: "+err.Error(), http.StatusInternalServerError)

		return
	}

	reply(w, speech.LoadVoiceSettings(s.TTS.Settings))
}

// uploadVoice stores a sample (a few seconds of clean speech, WAV) under
// the given name.
func (s *Server) uploadVoice(w http.ResponseWriter, r *http.Request) {
	if s.TTS == nil {
		http.Error(w, "mowa Chatterbox nie jest włączona", http.StatusNotFound)

		return
	}

	if err := r.ParseMultipartForm(sampleLimit); err != nil {
		http.Error(w, "złe żądanie: "+err.Error(), http.StatusBadRequest)

		return
	}

	file, header, err := r.FormFile("sample")
	if err != nil {
		http.Error(w, "brak pliku „sample”", http.StatusBadRequest)

		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	if given := strings.TrimSpace(r.FormValue("name")); given != "" {
		name = filepath.Base(given)
		if filepath.Ext(name) == "" {
			name += filepath.Ext(header.Filename)
		}
	}

	if name == "." || name == "" || strings.HasPrefix(name, ".") {
		http.Error(w, "zła nazwa", http.StatusBadRequest)

		return
	}

	if err := os.MkdirAll(s.TTS.Voices, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	out, err := os.Create(filepath.Join(s.TTS.Voices, name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(file, sampleLimit)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	reply(w, map[string]any{"name": name, "samples": samples(s.TTS.Voices)})
}

// say synthesizes a test line in a mood with a sample and returns the WAV,
// for the browser to play.
func (s *Server) say(w http.ResponseWriter, r *http.Request) {
	if s.TTS == nil {
		http.Error(w, "mowa Chatterbox nie jest włączona", http.StatusNotFound)

		return
	}

	var body struct {
		Text  string         `json:"text"`
		Mood  speech.Emotion `json:"mood"`
		Voice string         `json:"voice"`
		// Name and Kind pick the voice from the settings when Voice is empty.
		Name string `json:"name"`
		Kind string `json:"kind"`
	}

	if !decode(w, r, &body) {
		return
	}

	settings := speech.LoadVoiceSettings(s.TTS.Settings)
	how := settings.Expression(body.Mood)
	voice := body.Voice
	if voice == "" {
		voice = settings.VoiceFor(body.Name, body.Kind)
	}

	ctx, cancel := context.WithTimeout(r.Context(), sayTimeout)
	defer cancel()

	data, err := speech.Synthesize(ctx, s.TTS.URL, speech.SpeakRequest{
		Text: speech.Clean(body.Text), Language: settings.Language, Voice: voice,
		Exaggeration: how.Exaggeration, CFG: how.CFG, Temperature: how.Temperature,
	})
	if err != nil {
		http.Error(w, "serwer mowy: "+err.Error(), http.StatusBadGateway)

		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	_, _ = w.Write(data)
}
