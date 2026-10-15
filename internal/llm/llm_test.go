package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestGenerateWorldFollowsAStyleCard(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	preset, _ := json.Marshal(presets[1])
	style := `{"look":"Kosmiczny western.","dialogue":"Sucho, z ironią.","characters":"Łowcy nagród.","art":"neon, 90s anime"}`

	var briefs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request)

		messages := request["messages"].([]any)
		briefs = append(briefs, messages[len(messages)-1].(map[string]any)["content"].(string))

		content := string(preset)
		if len(briefs) == 1 {
			content = style
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}},
		})
	}))
	t.Cleanup(server.Close)

	client := &Client{BaseURL: server.URL + "/v1/", APIKey: "secret", Model: "test-model"}

	spec, err := client.GenerateWorld(context.Background(), world.ModeImagination, "detektywi w stylu cowboy bebop")
	if err != nil {
		t.Fatal(err)
	}

	if len(briefs) != 2 || !strings.HasSuffix(briefs[0], "Opis świata od gracza:\ndetektywi w stylu cowboy bebop") {
		t.Fatalf("briefs = %q", briefs)
	}

	for _, want := range []string{"KARTA STYLU", "Świat i estetyka: Kosmiczny western.", "Ton rozmów: Sucho, z ironią.", "Postacie: Łowcy nagród."} {
		if !strings.Contains(briefs[1], want) {
			t.Errorf("world brief misses %q:\n%s", want, briefs[1])
		}
	}

	if spec.Style == nil || spec.Style.Art != "neon, 90s anime" || spec.Style.Source != "detektywi w stylu cowboy bebop" {
		t.Errorf("style = %+v", spec.Style)
	}

	// A model that cannot describe the style still designs the world.
	client = serve(t, http.StatusOK, string(preset), nil)
	if spec, err := client.GenerateWorld(context.Background(), world.ModeImagination, "nic"); err != nil || spec.Style != nil {
		t.Errorf("without a card: %+v, %v", spec.Style, err)
	}
}

// replies serves one scripted answer per request and keeps the requests.
func replies(t *testing.T, answers ...string) (*Client, *[][]map[string]any) {
	t.Helper()

	var seen [][]map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []map[string]any `json:"messages"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request)
		seen = append(seen, request.Messages)

		content := answers[len(answers)-1]
		if len(seen) <= len(answers) {
			content = answers[len(seen)-1]
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}},
		})
	}))
	t.Cleanup(server.Close)

	return &Client{BaseURL: server.URL + "/v1/", APIKey: "secret", Model: "test-model"}, &seen
}

func lastUser(messages []map[string]any) string {
	return messages[len(messages)-1]["content"].(string)
}

func TestDesignLetsTheModelMendARejectedWorld(t *testing.T) {
	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	good, _ := json.Marshal(presets[1])
	client, seen := replies(t, `{}`, `{"name":"Pustka","places":[],"monsters":[],"items":[]}`, string(good))

	spec, err := client.GenerateWorld(context.Background(), world.ModeImagination, "pustka")
	if err != nil || spec.Name != presets[1].Name {
		t.Fatalf("spec %q, err %v", spec.Name, err)
	}

	// Style, the rejected design, the mended one.
	if len(*seen) != 3 {
		t.Fatalf("%d requests", len(*seen))
	}

	retry := (*seen)[2]
	if len(retry) != 2 || !strings.Contains(lastUser(retry), "Opis świata od gracza:\npustka") || !strings.Contains(lastUser(retry), "Gra odrzuciła poprzedni projekt: ") {
		t.Errorf("retry conversation = %v", retry)
	}

	client, seen = replies(t, `{}`, `{"name":"Pustka","places":[],"monsters":[],"items":[]}`)
	if _, err := client.GenerateWorld(context.Background(), world.ModeImagination, "pustka"); err == nil || len(*seen) != 3 {
		t.Errorf("a world rejected twice should fail after %d requests: %v", len(*seen), err)
	}
}

func investigationHalves(t *testing.T) (world.Spec, string, string) {
	t.Helper()

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	index := slices.IndexFunc(presets, func(spec world.Spec) bool { return spec.Mode == world.ModeInvestigation })
	if index < 0 {
		t.Fatal("no investigation preset")
	}

	preset := presets[index]

	first := caseDraft{Name: preset.Name, Intro: preset.Intro, Goal: preset.Goal, People: preset.People, Case: preset.Case, Deadline: preset.Deadline, Loss: preset.Loss}
	for _, room := range preset.Rooms {
		room.Things, room.Doors = nil, nil
		first.Rooms = append(first.Rooms, room)
	}

	type placed struct {
		world.ThingSpec
		Room string `json:"room"`
	}

	second := map[string]any{"steps": preset.Steps, "events": preset.Events}
	things := make([]placed, 0, len(preset.Things))
	for _, thing := range preset.Things {
		where := ""
		for _, room := range preset.Rooms {
			if slices.Contains(room.Things, thing.Name) {
				where = room.Name
			}
		}

		things = append(things, placed{thing, where})
	}

	second["things"] = things

	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)

	return preset, string(a), string(b)
}

func TestGenerateInvestigationInTwoSteps(t *testing.T) {
	preset, first, second := investigationHalves(t)
	client, seen := replies(t, `{"look":"Mgła.","dialogue":"Cicho.","characters":"Sąsiedzi.","art":"fog"}`, first, second)

	spec, err := client.GenerateWorld(context.Background(), world.ModeInvestigation, "zaginięcie nad jeziorem")
	if err != nil {
		t.Fatal(err)
	}

	if len(*seen) != 3 {
		t.Fatalf("%d requests", len(*seen))
	}

	if sys := (*seen)[1][0]["content"].(string); !strings.HasPrefix(sys, caseInstructions) {
		t.Errorf("first step asked with %q", sys[:40])
	}

	if sys, brief := (*seen)[2][0]["content"].(string), lastUser((*seen)[2]); !strings.HasPrefix(sys, townInstructions) || !strings.Contains(brief, "USTALONA SPRAWA") || !strings.Contains(brief, preset.Case.Answer) || !strings.Contains(brief, "KARTA STYLU") {
		t.Errorf("second step: %q / %q", sys[:40], brief)
	}

	if spec.Mode != world.ModeInvestigation || spec.Case == nil || spec.Case.Answer != preset.Case.Answer || spec.Style == nil || spec.Style.Art != "fog" {
		t.Errorf("spec = %+v", spec)
	}

	if len(spec.Rooms) != len(preset.Rooms) || len(spec.People) != len(preset.People) || len(spec.Things) != len(preset.Things) || len(spec.Steps) != len(preset.Steps) {
		t.Errorf("merged %d rooms, %d people, %d things, %d steps; preset has %d, %d, %d, %d",
			len(spec.Rooms), len(spec.People), len(spec.Things), len(spec.Steps), len(preset.Rooms), len(preset.People), len(preset.Things), len(preset.Steps))
	}

	for index, room := range preset.Rooms {
		if !slices.Equal(spec.Rooms[index].Things, room.Things) {
			t.Errorf("room %s got things %v, want %v", room.Name, spec.Rooms[index].Things, room.Things)
		}
	}

	if _, err := world.Build(spec); err != nil {
		t.Errorf("merged investigation does not build: %v", err)
	}
}

func TestGenerateInvestigationChecksTheCase(t *testing.T) {
	_, first, second := investigationHalves(t)

	var draft caseDraft
	_ = json.Unmarshal([]byte(first), &draft)
	draft.Case.Answer = "Nikt Taki"
	draft.People[0].Room = "Nieistniejący Pokój"
	broken, _ := json.Marshal(draft)

	client, seen := replies(t, `{}`, string(broken), first, second)

	spec, err := client.GenerateWorld(context.Background(), world.ModeInvestigation, "zaginięcie")
	if err != nil {
		t.Fatal(err)
	}

	if len(*seen) != 4 || !strings.Contains(lastUser((*seen)[2]), `case.answer "Nikt Taki" nie jest name żadnej osoby`) {
		t.Errorf("%d requests; retry asked: %q", len(*seen), lastUser((*seen)[2]))
	}

	if spec.Case == nil || len(spec.People) == 0 {
		t.Errorf("spec = %+v", spec)
	}

	// A culprit named by surname alone is matched to the person.
	draft = caseDraft{}
	_ = json.Unmarshal([]byte(first), &draft)
	full := draft.Case.Answer
	draft.Case.Answer = strings.Fields(full)[len(strings.Fields(full))-1]
	draft.mend()
	if draft.Case.Answer != full {
		t.Errorf("answer mended to %q, want %q", draft.Case.Answer, full)
	}

	// A person in a room the town lacks moves to the first room instead
	// of vanishing.
	draft = caseDraft{}
	_ = json.Unmarshal([]byte(first), &draft)
	draft.People[0].Room = "Nieistniejący Pokój"
	moved := draft.merge(townDraft{})
	if moved.People[0].Room != draft.Rooms[0].Name {
		t.Errorf("person moved to %q", moved.People[0].Room)
	}
}

func TestPlayableNeedsASourceForEveryProof(t *testing.T) {
	preset, _, _ := investigationHalves(t)
	if err := playable(preset); err != nil {
		t.Fatalf("the preset should be playable: %v", err)
	}

	short := preset
	short.Things = short.Things[:2]
	if err := playable(short); err == nil || !strings.Contains(err.Error(), "jest tylko 2 rzeczy") {
		t.Errorf("too few things passed: %v", err)
	}

	orphan := preset
	orphan.Case = &world.CaseSpec{Answer: preset.Case.Answer, Proof: append([]string{"Fakt Znikąd"}, preset.Case.Proof...)}
	if err := playable(orphan); err == nil || !strings.Contains(err.Error(), "Fakt Znikąd") {
		t.Errorf("an unobtainable proof passed: %v", err)
	}
}

// ollamaServer answers like an Ollama: it has a version and takes chat
// requests on its own API, each remembered in seen.
func ollamaServer(t *testing.T, seen *[]map[string]any, answer func(w http.ResponseWriter, request map[string]any)) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "0.18.2"})
		case "/api/chat":
			request := map[string]any{}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &request)
			*seen = append(*seen, request)
			answer(w, request)
		default:
			t.Errorf("path = %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return &Client{BaseURL: server.URL + "/v1", Model: "gemma", Context: 8192}
}

func TestOllamaIsAskedForTheWindowOnItsOwnAPI(t *testing.T) {
	var seen []map[string]any
	client := ollamaServer(t, &seen, func(w http.ResponseWriter, request map[string]any) {
		if request["stream"] == true {
			for _, piece := range []string{"Daj ", "140."} {
				_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": piece}, "done": false})
			}

			_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": ""}, "done": true, "done_reason": "stop"})

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": `{"action":"offer","amount":30}`}, "done": true, "done_reason": "stop"})
	})

	content, err := client.chat(context.Background(), chatRequest{
		Model:          "gemma",
		Messages:       []message{{Role: "user", Content: "dam 30 za plaszcz, mam 30"}},
		ResponseFormat: map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "x", "strict": true, "schema": map[string]any{"type": "object"}}},
	})
	if err != nil || content != `{"action":"offer","amount":30}` {
		t.Fatalf("chat = %q, %v", content, err)
	}

	client.Warm(context.Background())

	var pieces []string
	text, err := client.Stream(context.Background(), game.NarrateRequest{}, func(piece string) { pieces = append(pieces, piece) })
	if err != nil || text != "Daj 140." || len(pieces) != 2 {
		t.Errorf("stream = %q, %v, pieces %q", text, err, pieces)
	}

	if len(seen) != 3 {
		t.Fatalf("%d requests: %v", len(seen), seen)
	}

	options := seen[0]["options"].(map[string]any)
	if options["num_ctx"] != float64(8192) || options["num_predict"] != nil || seen[0]["stream"] != false {
		t.Errorf("design request = %v", seen[0])
	}

	if format, ok := seen[0]["format"].(map[string]any); !ok || format["type"] != "object" || seen[0]["response_format"] != nil {
		t.Errorf("schema was not passed as the format: %v", seen[0])
	}

	if options := seen[1]["options"].(map[string]any); options["num_predict"] != float64(1) || options["num_ctx"] != float64(8192) {
		t.Errorf("warm-up request = %v", seen[1])
	}

	for i, request := range seen {
		if request["keep_alive"] != keepAlive {
			t.Errorf("request %d keeps the model for %v, want %s", i, request["keep_alive"], keepAlive)
		}
	}

	if seen[2]["stream"] != true || seen[2]["format"] != nil {
		t.Errorf("stream request = %v", seen[2])
	}
}

func TestOtherServersKeepTheOpenAIPath(t *testing.T) {
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "ok"}, "finish_reason": "stop"}},
		})
	}))
	t.Cleanup(server.Close)

	client := &Client{BaseURL: server.URL + "/v1", Model: "gpt", Context: 8192}
	for range 2 {
		if content, err := client.chat(context.Background(), chatRequest{Model: "gpt", Messages: []message{{Role: "user", Content: "ok"}}}); err != nil || content != "ok" {
			t.Fatalf("chat = %q, %v", content, err)
		}
	}

	// The server is asked what it is once, not before every request.
	if asked != 3 {
		t.Errorf("%d requests to a plain server", asked)
	}

	// Without a window to ask for, nobody is probed.
	plain := &Client{BaseURL: server.URL + "/v1", Model: "gpt"}
	if plain.ollama(context.Background()) || asked != 3 {
		t.Errorf("a client without a window probed the server")
	}
}

func TestACutAnswerIsRedoneShorter(t *testing.T) {
	var seen []map[string]any
	client := ollamaServer(t, &seen, func(w http.ResponseWriter, request map[string]any) {
		if len(seen) == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": `{"name":"Mia`}, "done": true, "done_reason": "length"})

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": `{"name":"Miasto"}`}, "done": true, "done_reason": "stop"})
	})

	_, err := client.chat(context.Background(), chatRequest{Model: "gemma", Messages: []message{{Role: "user", Content: "świat"}}})
	if !errors.Is(err, ErrCut) {
		t.Fatalf("a cut answer came back as %v", err)
	}

	// A capped reply may end at the cap: that is not a cut.
	seen = nil
	client.server = serverUnknown
	if _, err := client.chat(context.Background(), chatRequest{Model: "gemma", Messages: []message{{Role: "user", Content: "mów"}}, MaxTokens: 5}); err != nil {
		t.Errorf("a capped reply was refused: %v", err)
	}

	seen = nil
	content, err := client.design(context.Background(), "projektuj", map[string]any{"type": "object"}, "świat", func(content string) error {
		if !strings.HasSuffix(content, "}") {
			return errors.New("ucięte")
		}

		return nil
	})
	if err != nil || content != `{"name":"Miasto"}` || len(seen) != 2 {
		t.Fatalf("design = %q, %v after %d requests", content, err, len(seen))
	}

	retry := seen[1]["messages"].([]any)[1].(map[string]any)["content"].(string)
	if !strings.Contains(retry, "została ucięta") {
		t.Errorf("the retry did not ask for a shorter answer: %q", retry)
	}

	// Designs are capped, so that a rambling model cannot run for a quarter
	// of an hour; a design cut at the cap is still a cut.
	if options := seen[0]["options"].(map[string]any); options["num_predict"] != float64(designBudget) || options["temperature"] != designTemperature {
		t.Errorf("design request = %v", seen[0])
	}

	seen = nil
	if _, err := client.chat(context.Background(), chatRequest{Model: "gemma", Messages: []message{{Role: "user", Content: "świat"}}, MaxTokens: 5, ResponseFormat: map[string]any{"type": "json_object"}}); !errors.Is(err, ErrCut) {
		t.Errorf("a cut JSON answer came back as %v", err)
	}
}

func TestAnInvestigationIsWonByTheAccusationAlone(t *testing.T) {
	draft := caseDraft{Name: "Mgła", Rooms: []world.RoomSpec{{Name: "Port"}}}
	spec := draft.merge(townDraft{
		Steps:  []world.StepSpec{{Verb: "talk", Thing: "Elias", Room: "Port", Win: true}},
		Events: []world.EventSpec{{Text: "Syrena.", Win: true}},
	})

	if spec.Steps[0].Win || spec.Events[0].Win {
		t.Errorf("a step or event of the case still wins the game: %+v %+v", spec.Steps[0], spec.Events[0])
	}
}

func TestSourceGivesEveryProofAClue(t *testing.T) {
	spec := world.Spec{
		Case: &world.CaseSpec{Answer: "Elias", Proof: []string{"spotkanie na molo", "przekupstwo urzędników", "zaginiony skaner"}},
		People: []world.PersonSpec{
			{Name: "Glitch", Clues: []world.ClueSpec{{Subject: "Spotkanie na molo", Text: "Widziałem ich."}, {Subject: "przekupstwa urzędnika", Text: "Brał kopertę."}}},
		},
		Things: []world.ThingSpec{
			{Name: "lampa"},
			{Name: "rejestr", Fixed: true, Clues: []world.ClueSpec{{Subject: "długi w porcie", Text: "Spis dłużników."}}},
			{Name: "szafka", Fixed: true},
		},
	}

	source(&spec)

	if err := playable(world.Spec{Case: spec.Case, People: spec.People, Things: append(spec.Things, world.ThingSpec{Name: "x"})}); err != nil {
		t.Fatalf("after sourcing: %v", err)
	}

	// The same words with another ending are the same fact, renamed to
	// match the proof exactly; the exact one is left alone.
	if clues := spec.People[0].Clues; clues[0].Subject != "Spotkanie na molo" || clues[1].Subject != "przekupstwo urzędników" || clues[1].Text != "Brał kopertę." {
		t.Errorf("person clues = %+v", clues)
	}

	// A proof nothing resembles goes on a fixed thing without clues.
	if clues := spec.Things[2].Clues; len(clues) != 1 || clues[0].Subject != "zaginiony skaner" || !strings.Contains(clues[0].Text, "zaginiony skaner") {
		t.Errorf("szafka clues = %+v", clues)
	}

	if len(spec.Things[0].Clues)+len(spec.Things[1].Clues) != 1 {
		t.Errorf("other things changed: %+v", spec.Things)
	}

	if alike("znikające fundusze z kościelnego rachunku", "fundusze kościelne") != 2 || alike("molo", "port") != 0 {
		t.Errorf("alike = %d, %d", alike("znikające fundusze z kościelnego rachunku", "fundusze kościelne"), alike("molo", "port"))
	}
}
