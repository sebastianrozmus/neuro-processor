package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// krypty returns a talk with the armourer of the first built-in world in
// its first settlement, where item-1 is a weapon listed at 40 with a hidden
// minimum of 26 and item-5 is the tier 2 armor.
func krypty(t *testing.T) *game.Session {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	return game.NewSession(game.MerchantFor(presets[0], 0, 2), 60)
}

func serve(t *testing.T, status int, content string, seen *map[string]any) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		if seen != nil {
			_ = json.Unmarshal(body, seen)
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}},
		})
	}))
	t.Cleanup(server.Close)

	return &Client{BaseURL: server.URL + "/v1/", APIKey: "secret", Model: "test-model"}
}

func TestClientInterpretAsksTheModelAboutLongMessages(t *testing.T) {
	seen := map[string]any{}
	client := serve(t, http.StatusOK, "```json\n{\"action\":\"accept\",\"item_id\":\"item-1\",\"amount\":999,\"persuasion\":1}\n```", &seen)

	session := krypty(t)
	text := "dam 60 za sztylet, bo ma wyszczerbione ostrze i nikt inny go nie weźmie"

	intent, err := client.Interpret(context.Background(), session.InterpretRequest(text))
	if err != nil {
		t.Fatal(err)
	}

	// The model judges the argument; the item and the amount come from the text.
	want := game.Intent{Action: game.ActionOffer, ItemID: "item-2", Amount: 60, Persuasion: 1}
	if intent != want {
		t.Errorf("intent = %+v, want %+v", intent, want)
	}

	if seen["model"] != "test-model" {
		t.Errorf("model = %v", seen["model"])
	}

	format, _ := json.Marshal(seen["response_format"])
	for _, fragment := range []string{`"json_schema"`, `"strict":true`, `"item-5"`, `"offer"`} {
		if !strings.Contains(string(format), fragment) {
			t.Errorf("response_format misses %s: %s", fragment, format)
		}
	}
}

func TestClientInterpretTrustsTheRulesFirst(t *testing.T) {
	session := krypty(t)
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := &Client{BaseURL: server.URL, Model: "test-model"}

	intent, err := client.Interpret(context.Background(), session.InterpretRequest("dam 30 za kord"))
	if want := (game.Intent{Action: game.ActionOffer, ItemID: "item-1", Amount: 30}); err != nil || intent != want || calls != 0 {
		t.Errorf("a clear short offer: %+v, %v after %d calls", intent, err, calls)
	}

	intent, err = client.Interpret(context.Background(), session.InterpretRequest("dam 30 za kord, bo jest cały zardzewiały i krzywy"))
	if want := (game.Intent{Action: game.ActionOffer, ItemID: "item-1", Amount: 30}); err != nil || intent != want || calls != 1 {
		t.Errorf("a clear long offer with the model down: %+v, %v after %d calls", intent, err, calls)
	}

	// Plain talk is not read by the model at all.
	intent, err = client.Interpret(context.Background(), session.InterpretRequest("co słychać na bagnach, stara znajoma z dawnych lat?"))
	if err != nil || intent.Action != game.ActionChat || calls != 1 {
		t.Errorf("plain talk: %+v, %v after %d calls", intent, err, calls)
	}

	// A bare number is the one thing only the model can settle.
	if _, err := client.Interpret(context.Background(), session.InterpretRequest("podobno w zeszłym roku było ich tu ze 12")); err == nil || calls != 2 {
		t.Errorf("a bare number with the model down should report the failure: %v after %d calls", err, calls)
	}
}

func TestClientNeverSendsMinPrice(t *testing.T) {
	seen := map[string]any{}
	client := serve(t, http.StatusOK, "Daj więcej.", &seen)

	session := krypty(t)
	decision := session.Apply(game.Intent{Action: game.ActionOffer, ItemID: "item-1", Amount: 20})

	client.MaxTokens = 150

	if _, err := client.Narrate(context.Background(), session.NarrateRequest("dam 20", decision)); err != nil {
		t.Fatal(err)
	}

	if seen["max_tokens"] != float64(150) {
		t.Errorf("max_tokens = %v", seen["max_tokens"])
	}

	body, _ := json.Marshal(seen)
	if decision.Outcome != game.OutcomeCounter || strings.Contains(string(body), "26") {
		t.Errorf("request leaks the minimum price: %s", body)
	}

	if !strings.Contains(string(body), "[GRA]") {
		t.Errorf("request misses the game verdict: %s", body)
	}
}

func TestClientErrors(t *testing.T) {
	session := krypty(t)
	bare := "podobno w zeszłym roku było ich tu ze 12"

	client := serve(t, http.StatusUnauthorized, "", nil)
	if _, err := client.Interpret(context.Background(), session.InterpretRequest(bare)); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("unauthorized: err = %v", err)
	}

	client = serve(t, http.StatusOK, "nie wiem", nil)
	if _, err := client.Interpret(context.Background(), session.InterpretRequest(bare)); err == nil {
		t.Error("prose instead of JSON should be an error")
	}
}

func TestMockInterpret(t *testing.T) {
	session := krypty(t)
	tests := []struct {
		text string
		want game.Intent
	}{
		{"Dam 120 za ten sztylet", game.Intent{Action: game.ActionOffer, ItemID: "item-2", Amount: 120}},
		{"a może 90?", game.Intent{Action: game.ActionOffer, Amount: 90}},
		{"Opowiedz o kolczudze", game.Intent{Action: game.ActionAsk, ItemID: "item-5"}},
		{"Co masz na sprzedaż?", game.Intent{Action: game.ActionAsk}},
		{"Dobra, biorę", game.Intent{Action: game.ActionAccept}},
		{"ZGODA, KUPUJĘ KORD", game.Intent{Action: game.ActionAccept, ItemID: "item-1"}},
		{"ile za miecz", game.Intent{Action: game.ActionAsk}},
		{"Żegnaj", game.Intent{Action: game.ActionLeave}},
		{"Ładna dziś pogoda", game.Intent{Action: game.ActionChat}},
	}

	for _, test := range tests {
		got, err := Mock{}.Interpret(context.Background(), session.InterpretRequest(test.text))
		if err != nil || got != test.want {
			t.Errorf("Interpret(%q) = %+v, %v; want %+v", test.text, got, err, test.want)
		}
	}
}

func TestGenerateWorld(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	reply, _ := json.Marshal(presets[1])
	seen := map[string]any{}
	client := serve(t, http.StatusOK, string(reply), &seen)

	spec, err := client.GenerateWorld(context.Background(), world.ModeImagination, "pustkowia po wojnie")
	if err != nil {
		t.Fatal(err)
	}

	if spec.Name != presets[1].Name || len(spec.Items) != 17 || spec.Items[0].ID != "item-1" || len(spec.Shops) != 3 || len(spec.Backdrops) != 0 {
		t.Errorf("spec = %q with %d items", spec.Name, len(spec.Items))
	}

	if _, err := world.Build(spec); err != nil {
		t.Errorf("generated world does not build: %v", err)
	}

	format, _ := json.Marshal(seen["response_format"])
	for _, fragment := range []string{`"strict":true`, `"wasteland"`, `"charm"`, `"blast"`, `"serpent"`, `"shrine"`, `"alchemist"`, `"greeting"`} {
		if !strings.Contains(string(format), fragment) {
			t.Errorf("response_format misses %s", fragment)
		}
	}

	client = serve(t, http.StatusOK, `{"name":"Pustka","places":[],"monsters":[],"items":[]}`, nil)
	if _, err := client.GenerateWorld(context.Background(), world.ModeImagination, "nic"); err == nil {
		t.Error("an unplayable world should be rejected")
	}
}

func TestLocalAndWarm(t *testing.T) {
	for address, want := range map[string]bool{
		"http://localhost:11434/v1":  true,
		"http://127.0.0.1:1234/v1":   true,
		"https://api.openai.com/v1":  false,
		"http://192.168.1.20:11434/": false,
		"::not a url::":              false,
	} {
		if got := (&Client{BaseURL: address}).Local(); got != want {
			t.Errorf("Local(%q) = %v, want %v", address, got, want)
		}
	}

	seen := map[string]any{}
	client := serve(t, http.StatusOK, "ok", &seen)
	client.Warm(context.Background())

	if seen["model"] != "test-model" || seen["max_tokens"] != float64(1) || !client.Local() {
		t.Errorf("warm-up request = %v", seen)
	}
}

func TestInvent(t *testing.T) {
	seen := map[string]any{}
	client := serve(t, http.StatusOK, `{"offered":true,"name":"Hełm z Rondem","description":"Garnek z daszkiem.","kind":"armor","effect":"none","tier":2}`, &seen)

	session := krypty(t)
	item, offered, err := client.Invent(context.Background(), game.InventRequest{Reply: "Mam też hełm z rondem.", Wares: session.Wares()})
	if err != nil || !offered || item.Name != "Hełm z Rondem" || item.Kind != world.KindArmor || item.Tier != 2 {
		t.Errorf("invent = %+v, %v, %v", item, offered, err)
	}

	format, _ := json.Marshal(seen["response_format"])
	for _, fragment := range []string{`"offered"`, `"charm"`, `"focus"`, `"strict":true`} {
		if !strings.Contains(string(format), fragment) {
			t.Errorf("response_format misses %s", fragment)
		}
	}

	messages, _ := json.Marshal(seen["messages"])
	if !strings.Contains(string(messages), "Zardzewiały Kord") || !strings.Contains(string(messages), "hełm z rondem") {
		t.Errorf("the model was not shown the wares and the line: %s", messages)
	}

	client = serve(t, http.StatusOK, `{"offered":false,"name":"","description":"","kind":"","effect":"","tier":0}`, nil)
	if _, offered, err := client.Invent(context.Background(), game.InventRequest{}); offered || err != nil {
		t.Errorf("nothing offered: %v, %v", offered, err)
	}

	client = serve(t, http.StatusBadGateway, "", nil)
	if _, _, err := client.Invent(context.Background(), game.InventRequest{}); err == nil {
		t.Error("a failing server should be an error")
	}
}

func TestTranscriptMatchesThePrompt(t *testing.T) {
	session := krypty(t)
	decision := session.Apply(game.Intent{Action: game.ActionChat})
	request := session.NarrateRequest("co słychać?", decision)
	request.Lore = "Świat: Test."

	transcript := Transcript(request, "Nic nowego.")
	if len(transcript) != 3 || transcript[0]["role"] != "system" || transcript[2]["content"] != "Nic nowego." {
		t.Fatalf("transcript = %v", transcript)
	}

	if !strings.Contains(transcript[0]["content"], "Świat: Test.") || !strings.Contains(transcript[1]["content"], game.VerdictMark) {
		t.Errorf("transcript lacks the lore or the verdict: %v", transcript)
	}
}

func TestStream(t *testing.T) {
	streamed := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"stream":true`) {
			// The retry after a bad reply is an ordinary request.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "Daj 140."}}},
			})

			return
		}

		streamed++
		w.Header().Set("Content-Type", "text/event-stream")
		for _, piece := range []string{"Daj ", "140", "."} {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", piece)
		}

		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(server.Close)

	client := &Client{BaseURL: server.URL, Model: "test-model"}

	var pieces []string
	text, err := client.Stream(context.Background(), game.NarrateRequest{}, func(piece string) { pieces = append(pieces, piece) })
	if err != nil || text != "Daj 140." || len(pieces) != 3 {
		t.Errorf("stream = %q, %v, pieces %q", text, err, pieces)
	}

	// The game's narrator uses the stream for its first try and checks the whole.
	session := krypty(t)
	decision := session.Apply(game.Intent{Action: game.ActionOffer, ItemID: "item-1", Amount: 20})
	pieces = nil

	text, err = game.NarrateLive(context.Background(), client, session.NarrateRequest("dam 20", decision), func(piece string) { pieces = append(pieces, piece) })
	if err == nil || text != decision.Fallback() || len(pieces) != 3 || streamed != 2 {
		t.Errorf("a streamed reply with a wrong price: %q, %v, pieces %q, %d streamed requests", text, err, pieces, streamed)
	}
}

func TestTranslate(t *testing.T) {
	client := serve(t, http.StatusOK, "  \"an old witch with a raven on her shoulder\"\n", nil)

	if text, err := client.Translate(context.Background(), "stara wiedźma z krukiem na ramieniu"); err != nil || text != "an old witch with a raven on her shoulder" {
		t.Errorf("translate = %q, %v", text, err)
	}
}
