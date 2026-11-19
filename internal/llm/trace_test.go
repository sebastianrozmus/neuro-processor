package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/trace"
)

// ollamaStub answers like Ollama's own API, with its counts of tokens and
// times, and keeps the options of the last request.
func ollamaStub(t *testing.T, options *map[string]any) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_, _ = io.WriteString(w, `{"version":"0.40.4"}`)
		case "/api/chat":
			var body struct {
				Options map[string]any `json:"options"`
			}

			_ = json.NewDecoder(r.Body).Decode(&body)
			*options = body.Options
			_, _ = io.WriteString(w, `{"message":{"role":"assistant","content":"[spokój] Witaj."},"done":true,"prompt_eval_count":812,"eval_count":9,"load_duration":2000000,"prompt_eval_duration":150000000,"eval_duration":300000000}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func TestCallsAreTracedWithTokensAndTimes(t *testing.T) {
	var options map[string]any
	server := ollamaStub(t, &options)
	client := &Client{BaseURL: server.URL + "/v1", Model: "neuro-npc", Context: 4096, Role: RoleTalk}

	calls, _ := trace.Default.Since(0)
	last := int64(0)
	if len(calls) > 0 {
		last = calls[len(calls)-1].ID
	}

	request := game.NarrateRequest{Merchant: game.Merchant{Name: "Borin"}, Text: "elo"}
	if _, err := client.Narrate(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	calls, _ = trace.Default.Since(last)
	if len(calls) != 1 {
		t.Fatalf("%d calls traced", len(calls))
	}

	call := calls[0]
	if call.Role != RoleTalk || call.Op != "narrate" || call.Who != "Borin" || call.Said != "elo" || call.Model != "neuro-npc" {
		t.Errorf("call %+v", call)
	}

	if call.PromptTokens != 812 || call.ReplyTokens != 9 || call.GenMs != 300 || call.PromptMs != 150 {
		t.Errorf("counts %+v", call)
	}

	if len(call.Messages) == 0 || call.Reply != "[spokój] Witaj." {
		t.Errorf("prompt or reply missing: %+v", call)
	}

	if _, cpu := options["num_gpu"]; cpu {
		t.Error("a model on the graphics card was sent to the processor")
	}

	client.Use("gemma3:4b", true)
	if _, err := client.Narrate(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	if options["num_gpu"] != float64(0) {
		t.Errorf("options %v, want num_gpu 0", options)
	}
}

func TestSplitRolesSwitchModels(t *testing.T) {
	split := Split{
		Mind: &Client{Model: "gemma3:4b"}, Voice: &Client{Model: "run10d"}, Master: &Client{Model: "run10d"},
		World: &Client{Model: "gemma3:4b"}, Images: &Client{Model: "gemma3:4b"},
	}

	split.Use(map[string]Choice{RoleFight: {Model: "run7"}, RoleMind: {Model: "gemma3:4b", CPU: true}, RoleWorlds: {}})

	choices := split.Choices()
	if choices[RoleFight].Model != "run7" || choices[RoleTalk].Model != "run10d" || !choices[RoleMind].CPU || choices[RoleWorlds].Model != "gemma3:4b" {
		t.Errorf("choices %+v", choices)
	}

	if split.Narrator() != split.Master {
		t.Error("fights are not narrated by the Master")
	}
}
