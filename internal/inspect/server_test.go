package inspect

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// polisher rewords whatever it is given, predictably.
type polisher struct{ asked game.RewriteRequest }

func (p *polisher) Rewrite(_ context.Context, request game.RewriteRequest) (string, error) {
	p.asked = request

	return "Przepisane: " + request.Name + ".", nil
}

func testServer(t *testing.T) (*httptest.Server, *game.Realm, *polisher) {
	t.Helper()

	presets, err := world.Presets()
	if err != nil || len(presets) == 0 {
		t.Fatalf("presets: %v", err)
	}

	var spec world.Spec
	for _, candidate := range presets {
		if !candidate.Mode.Rooms() {
			spec = candidate

			break
		}
	}

	built, err := world.Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	realm := game.NewRealm(built, 1)
	realm.Join("Ala", "")

	server := New()
	brain := &polisher{}
	server.Watch(realm, brain)

	web := httptest.NewServer(server.Handler())
	t.Cleanup(web.Close)
	servers[web] = server

	return web, realm, brain
}

// servers remembers the Server behind each test web server.
var servers = map[*httptest.Server]*Server{}

func fromHandler(t *testing.T, web *httptest.Server) *Server {
	t.Helper()

	server, ok := servers[web]
	if !ok {
		t.Fatal("unknown server")
	}

	return server
}

func get(t *testing.T, url string, into any) *http.Response {
	t.Helper()

	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if into != nil && response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(into); err != nil {
			t.Fatal(err)
		}
	}

	return response
}

func post(t *testing.T, url string, body any, into any) *http.Response {
	t.Helper()

	data, _ := json.Marshal(body)
	response, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if into != nil && response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(into); err != nil {
			t.Fatal(err)
		}
	}

	return response
}

func TestInspectorServesTheWorld(t *testing.T) {
	web, _, _ := testServer(t)

	var report game.Inspection
	if response := get(t, web.URL+"/api/world", &report); response.StatusCode != http.StatusOK || report.Name == "" || len(report.Facts) == 0 || !report.Rewrites {
		t.Fatalf("world: %d, %+v", response.StatusCode, report)
	}

	if response := get(t, web.URL+"/", nil); response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
		t.Errorf("page: %d %s", response.StatusCode, response.Header.Get("Content-Type"))
	}

	for _, path := range []string{"/api/picture?kind=item&index=0", "/api/picture?kind=monster&index=0", "/api/picture?kind=boss", "/api/picture?kind=companion&index=0", "/api/picture?kind=hero&avatar=x"} {
		if response := get(t, web.URL+path, nil); response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "image/png" {
			t.Errorf("%s: %d %s", path, response.StatusCode, response.Header.Get("Content-Type"))
		}
	}

	if response := get(t, web.URL+"/api/picture?kind=item&index=999", nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("a missing picture gave %d", response.StatusCode)
	}
}

func TestInspectorWritesTheWorld(t *testing.T) {
	web, realm, brain := testServer(t)

	var report game.Inspection
	get(t, web.URL+"/api/world", &report)
	keeper := report.Keepers[0].Name

	var fact game.FactInfo
	if response := post(t, web.URL+"/api/fact", map[string]any{"id": 0, "subject": "Studnia", "text": "Woda smakuje żelazem.", "known": []string{keeper}}, &fact); response.StatusCode != http.StatusOK || fact.ID == 0 || len(fact.Known) != 1 {
		t.Fatalf("add: %d %+v", response.StatusCode, fact)
	}

	var rewritten map[string]string
	if response := post(t, web.URL+"/api/rewrite", map[string]any{"kind": game.DescribeFact, "index": fact.ID}, &rewritten); response.StatusCode != http.StatusOK || rewritten["text"] != "Przepisane: Studnia." || brain.asked.World == "" {
		t.Fatalf("rewrite: %d %+v (%+v)", response.StatusCode, rewritten, brain.asked)
	}

	if _, text, _ := realm.Description(game.DescribeFact, fact.ID); text != "Przepisane: Studnia." {
		t.Errorf("the rewrite was not kept: %q", text)
	}

	if response := post(t, web.URL+"/api/describe", map[string]any{"kind": game.DescribePlace, "index": 0, "text": "Nowe miejsce."}, nil); response.StatusCode != http.StatusOK {
		t.Errorf("describe: %d", response.StatusCode)
	}

	if response := post(t, web.URL+"/api/player", map[string]any{"name": "Ala", "field": "gold", "value": 77}, nil); response.StatusCode != http.StatusOK || realm.View("Ala").Me.Gold != 77 {
		t.Errorf("player: %d, gold %d", response.StatusCode, realm.View("Ala").Me.Gold)
	}

	request, _ := http.NewRequest(http.MethodDelete, web.URL+"/api/fact?id="+itoa(fact.ID), nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("delete: %v %d", err, response.StatusCode)
	}
	response.Body.Close()

	if _, _, ok := realm.Description(game.DescribeFact, fact.ID); ok {
		t.Error("the fact is still there")
	}

	if response := post(t, web.URL+"/api/fact", map[string]any{"id": 0, "subject": "", "text": "x"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("an empty subject gave %d", response.StatusCode)
	}

	// Without a realm there is nothing to show.
	empty := httptest.NewServer(New().Handler())
	defer empty.Close()

	if response := get(t, empty.URL+"/api/world", nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("no realm gave %d", response.StatusCode)
	}
}

func TestInspectorShowsPromptsAndTakesMarks(t *testing.T) {
	web, realm, _ := testServer(t)
	request := game.NarrateRequest{Merchant: game.Merchant{Name: "Ktoś"}, Text: "siema", Decision: game.Decision{Outcome: game.OutcomeChat, Mood: game.MoodCalm}, Hint: "Gracz tylko się wita."}
	realm.Exchange("Ala", request, "[spokój] Siema.", nil)

	var shown struct {
		Exchange game.Exchange       `json:"exchange"`
		Messages []map[string]string `json:"messages"`
	}

	if response := get(t, web.URL+"/api/exchange?id=1", &shown); response.StatusCode != http.StatusOK || shown.Exchange.Reply != "[spokój] Siema." || len(shown.Messages) < 3 || !strings.Contains(shown.Messages[1]["content"], "Gracz tylko się wita.") {
		t.Fatalf("exchange: %d %+v", response.StatusCode, shown)
	}

	if response := get(t, web.URL+"/api/exchange?id=9", nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("a missing exchange gave %d", response.StatusCode)
	}

	var note game.Note
	if response := post(t, web.URL+"/api/mark", map[string]any{"kind": "exchange", "keys": []string{"1"}, "comment": "za sucho"}, &note); response.StatusCode != http.StatusOK || note.Comment != "za sucho" {
		t.Fatalf("mark: %d %+v", response.StatusCode, note)
	}

	var report game.Inspection
	get(t, web.URL+"/api/world", &report)
	if len(report.Exchanges) != 1 || len(report.Exchanges[0].Marks) != 1 || len(report.Notes) != 1 || report.Traces == nil {
		t.Errorf("report after a mark: exchanges %+v, notes %+v", report.Exchanges, report.Notes)
	}

	if response := post(t, web.URL+"/api/mark", map[string]any{"kind": "exchange", "keys": []string{}, "comment": "x"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("an empty mark gave %d", response.StatusCode)
	}
}

func TestMarksAreWrittenDown(t *testing.T) {
	web, realm, _ := testServer(t)
	path := filepath.Join(t.TempDir(), "feedback.jsonl")
	fromHandler(t, web).Feedback = path

	realm.Exchange("Ala", game.NarrateRequest{Merchant: game.Merchant{Name: "Ktoś"}, Text: "hej", Decision: game.Decision{Outcome: game.OutcomeChat}}, "Hej.", nil)
	post(t, web.URL+"/api/mark", map[string]any{"kind": "exchange", "keys": []string{"1"}, "comment": "nijakie"}, nil)
	post(t, web.URL+"/api/mark", map[string]any{"kind": game.DescribePlace, "keys": []string{"0"}, "comment": "opis bez sensu"}, nil)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("feedback = %q", data)
	}

	var first struct {
		Comment string           `json:"comment"`
		Items   []map[string]any `json:"items"`
	}

	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil || first.Comment != "nijakie" || len(first.Items) != 1 || first.Items[0]["messages"] == nil {
		t.Errorf("first record = %+v (%v)", first, err)
	}

	if !strings.Contains(lines[1], "opis bez sensu") || !strings.Contains(lines[1], `"name"`) {
		t.Errorf("second record = %s", lines[1])
	}
}

func itoa(value int) string {
	data, _ := json.Marshal(value)

	return string(data)
}

func TestVoicesAreTunedThroughTheInspector(t *testing.T) {
	web, _, _ := testServer(t)

	var off struct {
		Enabled bool `json:"enabled"`
	}

	if response := get(t, web.URL+"/api/voices", &off); response.StatusCode != http.StatusOK || off.Enabled {
		t.Fatalf("voices without TTS: %d %+v", response.StatusCode, off)
	}

	// A fake Chatterbox: it echoes the request as the "wav".
	var asked speech.SpeakRequest
	box := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/voices" {
			_, _ = w.Write([]byte(`{"voices":["kowal.wav"]}`))

			return
		}

		_ = json.NewDecoder(r.Body).Decode(&asked)
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF"))
	}))
	defer box.Close()

	dir := t.TempDir()
	fromHandler(t, web).TTS = &TTS{URL: box.URL, Settings: filepath.Join(dir, "voices.json"), Voices: dir}
	_ = os.WriteFile(filepath.Join(dir, "kowal.wav"), []byte("x"), 0o644)

	var on struct {
		Enabled  bool                 `json:"enabled"`
		Alive    bool                 `json:"alive"`
		Samples  []string             `json:"samples"`
		Speakers []map[string]string  `json:"speakers"`
		Settings speech.VoiceSettings `json:"settings"`
	}

	if response := get(t, web.URL+"/api/voices", &on); response.StatusCode != http.StatusOK || !on.Enabled || !on.Alive || len(on.Samples) != 1 || len(on.Speakers) == 0 {
		t.Fatalf("voices: %d %+v", response.StatusCode, on)
	}

	if on.Settings.ByName == nil {
		t.Fatal("the settings came without maps")
	}

	on.Settings.ByName[on.Speakers[0]["name"]] = "kowal.wav"
	on.Settings.Moods = map[speech.Emotion]speech.Expression{speech.Anger: {Exaggeration: 0.85, CFG: 0.3, Temperature: 0.8}}
	if response := post(t, web.URL+"/api/voices", on.Settings, nil); response.StatusCode != http.StatusOK {
		t.Fatalf("save: %d", response.StatusCode)
	}

	if saved := speech.LoadVoiceSettings(filepath.Join(dir, "voices.json")); saved.ByName[on.Speakers[0]["name"]] != "kowal.wav" || saved.Expression(speech.Anger).Exaggeration != 0.85 {
		t.Errorf("saved = %+v", saved)
	}

	response := post(t, web.URL+"/api/say", map[string]any{"text": "[gniew] Precz!", "mood": "gniew", "name": on.Speakers[0]["name"], "kind": "armory"}, nil)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "audio/wav" || asked.Voice != "kowal.wav" || asked.Exaggeration != 0.85 || asked.Text != "gniew Precz!" && asked.Text != "Precz!" {
		t.Errorf("say: %d %s %+v", response.StatusCode, response.Header.Get("Content-Type"), asked)
	}
}
