package game_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// A script is a scripted conversation played through the game's own rules:
// TestScripts turns each line into the prompt the model would get, for
// tools/eval/run.py. Run with NEURO_SCRIPT=in.jsonl NEURO_SCENARIOS=out.jsonl.
// A line may carry the reply to remember in place of a stub: the next prompts
// then follow from it, and the turn gets its full messages, as training data.
//
// A script may seed a long game's memory first:
//
//	"day": 3                                   the calendar
//	"bio": {"Bogna Topór": ["Bogna ma brata Radka, kowala."]}
//	                                           what speakers said of themselves before
//	"rescue": {"keeper": "Halszka Nitownica", "person": "Zosia", "relation": "córka",
//	           "monster": "...", "place": "...", "taker": "", "state": "open|freed|paid"}
//
// and a line with a reply may carry the Mind's labels:
//
//	"self": ["Bogna ma siostrę Wierę."]        what SelfNote should read; kept with LearnSelf
//	"hook": {"found": true, "person": ...}     QuestHook label; a found hook opens the rescue
//
// A turn with a reply gets, when the game would ask the Mind (ui/game.go),
// "selfnote": {"system", "user"} and "questhook": {"system", "user",
// "monsters", "places"}, and "learned" (facts LearnSelf kept) and "spawned"
// (what SpawnRescue told, "ODRZUCONE" when it refused).
type script struct {
	ID       string              `json:"id"`
	Kind     string              `json:"kind"`
	Note     string              `json:"note"`
	World    string              `json:"world"`
	Place    string              `json:"place"`
	Hero     string              `json:"hero"`
	Hour     string              `json:"hour"`
	Talk     string              `json:"talk"`
	Hire     []string            `json:"hire"`
	Gold     int                 `json:"gold"`
	Health   int                 `json:"hp"`
	Standing int                 `json:"standing"`
	Day      int                 `json:"day"`
	Bio      map[string][]string `json:"bio"`
	Rescue   *rescueSeed         `json:"rescue"`
	Lines    []line              `json:"lines"`
}

type rescueSeed struct {
	Keeper   string `json:"keeper"`
	Person   string `json:"person"`
	Relation string `json:"relation"`
	Monster  string `json:"monster"`
	Place    string `json:"place"`
	Taker    string `json:"taker"`
	State    string `json:"state"`
}

type line struct {
	Say    string          `json:"say"`
	Reply  string          `json:"reply"`
	Intent *game.Intent    `json:"intent"`
	Self   []string        `json:"self"`
	Hook   *game.QuestHook `json:"hook"`
}

// extraWorlds are the worlds in NEURO_WORLDS (a folder of world files, as
// the game saves them), for scripts in worlds the game does not ship. A file
// that does not load fails the run instead of being skipped.
func extraWorlds(t *testing.T) []world.Spec {
	t.Helper()

	dir := os.Getenv("NEURO_WORLDS")
	if dir == "" {
		return nil
	}

	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	specs := make([]world.Spec, 0, len(paths))

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		var spec world.Spec
		if err := json.Unmarshal(data, &spec); err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		if err := spec.Normalize(); err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		specs = append(specs, spec)
	}

	return specs
}

// TestWorldFiles checks the worlds in NEURO_WORLDS load and that each place,
// keeper and companion can be talked to, as scripts need.
func TestWorldFiles(t *testing.T) {
	if os.Getenv("NEURO_WORLDS") == "" {
		t.Skip("NEURO_WORLDS not set")
	}

	for _, spec := range extraWorlds(t) {
		// Places are the settlements and the dangerous places both; sites
		// are the special tiles.
		towns := len(spec.TownPlaces())
		t.Logf("%s (%s): %d osad, %d miejsc groźnych, %d miejsc specjalnych, %d sklepów, %d towarzyszy, %d przedmiotów, %d artefaktów, %d potworów",
			spec.Name, spec.Mode, towns, len(spec.Places)-towns, len(spec.Sites), len(spec.Shops), len(spec.Companions), len(spec.Items), len(spec.Artifacts), len(spec.Monsters))

		built, err := world.Build(spec)
		if err != nil {
			t.Errorf("%s: %v", spec.Name, err)

			continue
		}

		realm := game.NewRealm(built, 7)
		hero := realm.Join("Radomir", "")

		for _, place := range spec.Places {
			if err := realm.Teleport(hero, place.Name); err != nil {
				t.Errorf("%s: %s: %v", spec.Name, place.Name, err)
			}
		}
	}
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

	presets = append(presets, extraWorlds(t)...)

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
			t.Fatalf("%v: %s", err, scanner.Text()[:min(80, len(scanner.Text()))])
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

	if spec.Name == "" {
		t.Fatalf("%s: no world %q", s.ID, s.World)
	}

	built, err := world.Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	realm := game.NewRealm(built, 7)
	name := s.Hero
	if name == "" {
		name = "Radomir"
	}

	hero := realm.Join(name, "")
	must(t, s.ID, realm.Teleport(hero, s.Place))
	realm.SetDay(s.Day)

	for speaker, facts := range s.Bio {
		kept, err := realm.SeedBio(hero, speaker, facts)
		must(t, s.ID, err)

		if kept != len(facts) {
			t.Fatalf("%s: bio of %s: kept %d of %d", s.ID, speaker, kept, len(facts))
		}
	}

	if r := s.Rescue; r != nil {
		must(t, s.ID, realm.SeedRescue(hero, r.Keeper, r.Person, r.Relation, r.Monster, r.Place, r.Taker, r.State))
	}

	realm.SetGold(hero, s.Gold)
	realm.SetHour(hero, s.Hour)
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

		turn := map[string]any{"player": player, "gra": gra, "outcome": decision.Outcome}
		reply := "[spokój] Dobrze."
		if said.Reply != "" {
			turn["messages"], reply = messages[:len(messages)-1], said.Reply
		}

		turns = append(turns, turn)
		realm.HaggleRemember(hero, said.Say, reply)

		if said.Reply != "" {
			mind(realm, hero, said, turn)
		}
	}

	return map[string]any{"id": s.ID, "kind": s.Kind, "note": s.Note, "system": system, "turns": turns}
}

// mind asks the Mind's tasks as the game asks them after a reply is
// remembered (ui/game.go), and plays the labels the script gives.
func mind(realm *game.Realm, hero string, said line, turn map[string]any) {
	if note, ok := realm.SelfNoteRequest(hero, said.Say, said.Reply); ok {
		system, user := llm.SelfNotePrompt(note)
		turn["selfnote"] = map[string]string{"system": system, "user": user}

		if said.Self != nil {
			turn["learned"] = realm.LearnSelf(hero, note.Speaker, said.Self, said.Reply)
		}
	}

	if ask, ok := realm.QuestHookRequest(hero, said.Say, said.Reply); ok {
		system, user := llm.QuestHookPrompt(ask)
		turn["questhook"] = map[string]any{"system": system, "user": user, "monsters": ask.Monsters, "places": ask.Places}

		if said.Hook != nil && said.Hook.Found {
			told, ok := realm.SpawnRescue(hero, ask.Speaker, *said.Hook, said.Reply)
			turn["spawned"] = told

			if !ok {
				turn["spawned"] = "ODRZUCONE"
			}
		}
	}
}

func must(t *testing.T, id string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
}
