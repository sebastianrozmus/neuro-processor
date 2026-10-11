package inspect

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
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

	return web, realm, brain
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

func itoa(value int) string {
	data, _ := json.Marshal(value)

	return string(data)
}
