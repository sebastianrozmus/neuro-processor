package game_test

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// A script is a scripted conversation played through the game's own rules:
// TestScripts turns each line into the prompt the model would get, for
// tools/eval/run.py. Run with NEURO_SCRIPT=in.jsonl NEURO_SCENARIOS=out.jsonl.
type script struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Note     string   `json:"note"`
	World    string   `json:"world"`
	Place    string   `json:"place"`
	Talk     string   `json:"talk"`
	Hire     []string `json:"hire"`
	Gold     int      `json:"gold"`
	Health   int      `json:"hp"`
	Standing int      `json:"standing"`
	Lines    []line   `json:"lines"`
}

type line struct {
	Say    string       `json:"say"`
	Intent *game.Intent `json:"intent"`
}

func (l *line) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &l.Say)
	}

	type plain line

	return json.Unmarshal(data, (*plain)(l))
}

func TestScripts(t *testing.T) {
	in, out := os.Getenv("NEURO_SCRIPT"), os.Getenv("NEURO_SCENARIOS")
	if in == "" || out == "" {
		t.Skip("NEURO_SCRIPT and NEURO_SCENARIOS not set")
	}

	presets, err := world.Presets()
	if err != nil {
		t.Fatal(err)
	}

	source, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	sink, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	scanner := bufio.NewScanner(source)
	scanner.Buffer(nil, 1<<20)

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}

		var s script
		if err := json.Unmarshal(scanner.Bytes(), &s); err != nil {
			t.Fatal(err)
		}

		scenario := play(t, presets, s)
		data, _ := json.Marshal(scenario)
		sink.Write(append(data, '\n'))
	}
}

func play(t *testing.T, presets []world.Spec, s script) map[string]any {
	t.Helper()

	var spec world.Spec
	for _, preset := range presets {
		if preset.Name == s.World || (s.World == "" && preset.Name == "Smocza Góra") {
			spec = preset
		}
	}

	built, err := world.Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	realm := game.NewRealm(built, 7)
	hero := realm.Join("Radomir", "")
	must(t, s.ID, realm.Teleport(hero, s.Place))
	realm.SetGold(hero, s.Gold)
	if s.Health > 0 {
		realm.SetHealth(hero, s.Health)
	}
	if s.Standing != 0 {
		must(t, s.ID, realm.SetStanding(hero, s.Place, s.Standing))
	}
	for _, mate := range s.Hire {
		must(t, s.ID, realm.Recruit(hero, mate))
	}

	realm.Execute(hero, game.Parse("rozmawiaj z "+s.Talk))

	system := ""
	turns := []map[string]any{}

	for _, said := range s.Lines {
		request, ok := realm.HaggleRequest(hero, said.Say)
		if !ok {
			t.Fatalf("%s: no conversation at %q: %v", s.ID, said.Say, realm.View(hero).Log)
		}

		reading := game.Read(request)
		intent := reading.Alone()
		open := reading.Open(len(strings.Fields(said.Say)) >= 6)
		if said.Intent != nil {
			intent = request.Sanitize(*said.Intent)
		}

		decision, narrate, _ := realm.Haggle(hero, said.Say, intent)
		messages := llm.Transcript(narrate, "")
		if system == "" {
			system = messages[0]["content"]
		}

		user := messages[len(messages)-2]["content"]
		player, gra, _ := strings.Cut(user, "\n\n"+game.VerdictMark+" ")
		if !strings.Contains(user, "\n\n"+game.VerdictMark) {
			player, gra = "", strings.TrimPrefix(user, game.VerdictMark+" ")
		}

		if open && said.Intent == nil {
			t.Logf("%s: %q read by rules only as %+v", s.ID, said.Say, intent)
		}

		turns = append(turns, map[string]any{"player": player, "gra": gra, "outcome": decision.Outcome})
		realm.HaggleRemember(hero, said.Say, "[spokój] Dobrze.")
	}

	return map[string]any{"id": s.ID, "kind": s.Kind, "note": s.Note, "system": system, "turns": turns}
}

func must(t *testing.T, id string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
}
