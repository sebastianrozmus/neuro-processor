package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// An investigation asks for too much at once for a small model: asked for
// the whole town, its people, the hidden truth, the things and the rules
// in one answer, a 4B model hands back three rooms and no case. So it is
// designed in two steps: first the town, the people and the case, then
// the things, the rules and the events around them.

const caseInstructions = `Projektujesz ŚLEDZTWO do tekstowej gry przygodowej w stylu Mózgoprocesora: miasteczko, jego mieszkańców i ukrytą prawdę, którą detektyw ma odkryć. ` +
	`Na podstawie opisu gracza zwracasz obiekt JSON zgodny ze schematem. Wszystkie teksty po polsku. Rzeczy, zagadki i zdarzenia powstaną w następnym kroku; teraz tylko miejsce, ludzie i sprawa.

Wymagania:
- name: nazwa gry, najwyżej 4 słowa. intro: 2-3 zdania, jak zaczyna się sprawa. goal: jedno zdanie, co zlecono detektywowi.
- rooms: 8 do 12 miejsc miasteczka (rynek, lokal, poczta, domy, stacja, park, cmentarz...). Każde ma name, description (2-3 zdania, co widać), ` +
	`terrain (street albo station dla ulic i wnętrz; park, cmentarz i okolice mogą mieć forest, plains, water) oraz x (0-14) i y (0-8): położenie na mapie. ` +
	`Przejść można tylko między miejscami stojącymi obok siebie (różnica 1 w x albo w y), więc układaj je w spójną sieć. Pierwsze miejsce to start.
- people: 5 do 8 osób. Każda ma name (imię i nazwisko), description (1-2 zdania), room (nazwa miejsca z rooms, gdzie zwykle jest), talk (co mówi, gdy nie ma nic ważniejszego), ` +
	`schedule: gdzie jest o każdej porze (klucze ranek, popołudnie, wieczór, noc; wartości to nazwy z rooms) i clues: 1-3 fakty, które zna (subject: krótka nazwa faktu po polsku, zwykłe słowa ze spacjami, np. "spotkanie na molo", nie identyfikator; text: 1-2 zdania). ` +
	`Warunki faktu: heard (subject innego faktu, który gracz musi już znać), hour (pora, o której osoba to powie), needs (flaga; zwykle puste).
- case: ukryta prawda. client to osoba z people, która wynajęła detektywa; answer to name osoby z people, którą trzeba wskazać; proof to 3-5 subject faktów, bez których oskarżenie nie przejdzie; ` +
	`verdict to zakończenie po udanym oskarżeniu (2-3 zdania); hint to podpowiedź, gdy nazwisko jest trafne, ale brakuje dowodów.
- Sprawca i jedna fałszywie podejrzana osoba mają w clues kłamstwo; ktoś inny zna fakt, który je obala (z heard wskazującym na kłamstwo). ` +
	`Co najmniej 2 fakty z proof ma ktoś w clues; resztę dadzą rzeczy i zdarzenia z następnego kroku.
- deadline: liczba godzin na całą sprawę (24-48; dzień ma 12 godzin: ranek 0-5, popołudnie 6-7, wieczór 8-9, noc 10-11). loss: zdanie na koniec, gdy czas minie.`

const townInstructions = `Dokańczasz ŚLEDZTWO do tekstowej gry przygodowej w stylu Mózgoprocesora. Miasteczko, ludzie i sprawa są już ustalone (dostajesz je w treści); ty dodajesz rzeczy, zagadki i zdarzenia. ` +
	`Zwracasz obiekt JSON zgodny ze schematem. Wszystkie teksty po polsku. Nazwy miejsc i osób piszesz dokładnie tak, jak w ustalonej sprawie.

Wymagania:
- things: 8 do 12 rzeczy. name po polsku, zwykłe słowa ze spacjami (np. "książka telefoniczna", nie identyfikator), description (1-2 zdania), room (nazwa miejsca, w którym leży na początku), fixed true dla tych, których nie da się zabrać (szafka, rejestr, automat). ` +
	`Część ma clues (kartka, książka telefoniczna, rejestr, zdjęcie): subject, text i warunki heard/hour/needs jak u ludzi; tak powstaje łańcuch poszlak od ludzi do rzeczy i adresów ` +
	`(np. numer na kartce w szafce dostępnej wieczorem, książka telefoniczna zamieniająca numer na nazwisko i adres). Fakty z proof, których nie zna żaden człowiek, muszą być w clues rzeczy albo w fact kroków.
- steps: 4 do 10 reguł. verb: "use" (użyj rzeczy; "użyj X na Y": thing X, on Y), "give" (daj thing osobie on), "talk" (rozmowa z osobą on), "open" (otwórz on), "examine" (zbadaj on), ` +
	`"enter" (wejście do miejsca on), "wait" (gracz czeka godzinę w miejscu on) z with (osoby, które muszą tam być) i hour: tak powstaje dowód naoczny (fact: subject i text, np. zdjęcie ze spotkania). ` +
	`room ogranicza krok do miejsca; needs to flaga; heard to subject faktu, który gracz musi już znać; once true dla jednorazowych; text to odpowiedź gry (1-2 zdania); gives daje rzecz, reveals ujawnia rzecz w miejscu; pola, których krok nie używa, zostają puste. ` +
	`Co najmniej jeden fakt z proof pochodzi z kroku wait albo enter z with i hour. Krok win nie jest potrzebny: sprawę kończy oskarżenie.
- events: 2-4 zdarzenia dziwności w stylu analog horror (when.day 2 lub 3, global true, text, opcjonalnie transform miejsca: room i description): miasteczko robi się dziwniejsze, im więcej gracz wie.`

// caseDraft is the first half of an investigation.
type caseDraft struct {
	Name     string             `json:"name"`
	Intro    string             `json:"intro"`
	Goal     string             `json:"goal"`
	Rooms    []world.RoomSpec   `json:"rooms"`
	People   []world.PersonSpec `json:"people"`
	Case     *world.CaseSpec    `json:"case"`
	Deadline int                `json:"deadline"`
	Loss     string             `json:"loss"`
}

// townDraft is the second half: the things come with the room they lie in.
type townDraft struct {
	Things []struct {
		world.ThingSpec
		Room string `json:"room"`
	} `json:"things"`
	Steps  []world.StepSpec  `json:"steps"`
	Events []world.EventSpec `json:"events"`
}

func caseSchema() map[string]any {
	full := investigationSchema()["properties"].(map[string]any)
	rooms := full["rooms"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)

	return object(map[string]any{
		"name":  full["name"],
		"intro": full["intro"],
		"goal":  full["goal"],
		"rooms": list(object(map[string]any{
			"name": rooms["name"], "description": rooms["description"], "terrain": rooms["terrain"], "x": rooms["x"], "y": rooms["y"],
		})),
		"people":   full["people"],
		"case":     full["case"],
		"deadline": full["deadline"],
		"loss":     full["loss"],
	})
}

// townSchema asks for the things, the rules and the events. Its rules
// carry only the fields an investigation uses: every field of the schema
// is one the model must fill, and a small model kept to a long schema
// spends its answer on empty fields and the whitespace between them.
func townSchema() map[string]any {
	full := investigationSchema()["properties"].(map[string]any)
	thing := full["things"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	step := full["steps"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)

	rule := map[string]any{}
	for _, name := range []string{"verb", "thing", "on", "room", "with", "hour", "heard", "needs", "once", "text", "gives", "reveals", "fact"} {
		rule[name] = step[name]
	}

	return object(map[string]any{
		"things": list(object(map[string]any{
			"name": thing["name"], "description": thing["description"], "room": map[string]any{"type": "string"}, "fixed": thing["fixed"], "clues": thing["clues"],
		})),
		"steps":  list(object(rule)),
		"events": full["events"],
	})
}

// generateInvestigation designs an investigation in two steps and merges
// them into one playable description.
func (c *Client) generateInvestigation(ctx context.Context, brief string, style *world.StyleSpec) (world.Spec, error) {
	var draft caseDraft

	_, err := c.design(ctx, caseInstructions, caseSchema(), brief, func(content string) error {
		dump(content, ".case")

		draft = caseDraft{}
		if err := json.Unmarshal([]byte(extractJSON(content)), &draft); err != nil {
			return fmt.Errorf("to nie jest poprawny JSON: %w", err)
		}

		draft.mend()

		return draft.check()
	})
	if err != nil {
		return world.Spec{}, err
	}

	// The second step gets the case as a short list, not the whole JSON:
	// a local model runs in a window of a few thousand tokens, and the
	// first answer alone would fill most of it.
	brief += "\n\nUSTALONA SPRAWA (nie zmieniaj jej; używaj dokładnie tych nazw miejsc, osób i faktów):\n" + draft.summary()

	var spec world.Spec

	_, err = c.design(ctx, townInstructions, townSchema(), brief, func(content string) error {
		dump(content, "")

		var town townDraft
		if err := json.Unmarshal([]byte(extractJSON(content)), &town); err != nil {
			return fmt.Errorf("to nie jest poprawny JSON: %w", err)
		}

		spec = draft.merge(town)
		spec.Mode, spec.Style = world.ModeInvestigation, style

		if err := spec.Normalize(); err != nil {
			return err
		}

		source(&spec)

		return playable(spec)
	})
	if err != nil {
		return world.Spec{}, fmt.Errorf("model returned an unplayable world: %w", err)
	}

	return spec, nil
}

// source gives every proof of the case a clue that yields it. A small
// model names the same fact a little differently in proof and in clues
// ("przekupstwo urzędników" against "przekupstwa urzędnika"), so a clue
// about the same words is renamed to the proof; a proof nothing comes
// close to is put on a thing as a plain clue rather than left unprovable.
func source(spec *world.Spec) {
	if spec.Case == nil {
		return
	}

	var clues []*world.ClueSpec
	for index := range spec.People {
		for at := range spec.People[index].Clues {
			clues = append(clues, &spec.People[index].Clues[at])
		}
	}

	for index := range spec.Things {
		for at := range spec.Things[index].Clues {
			clues = append(clues, &spec.Things[index].Clues[at])
		}
	}

	known := func(subject string) bool {
		for _, clue := range clues {
			if strings.EqualFold(clue.Subject, subject) {
				return true
			}
		}

		for _, step := range spec.Steps {
			if step.Fact != nil && strings.EqualFold(step.Fact.Subject, subject) {
				return true
			}
		}

		return false
	}

	proofs := map[string]bool{}
	for _, proof := range spec.Case.Proof {
		proofs[strings.ToLower(proof)] = true
	}

	for _, proof := range spec.Case.Proof {
		if known(proof) {
			continue
		}

		var best *world.ClueSpec
		score := 0
		for _, clue := range clues {
			// A clue that already yields another proof stays as it is.
			if proofs[strings.ToLower(clue.Subject)] {
				continue
			}

			if shared := alike(clue.Subject, proof); shared > score {
				best, score = clue, shared
			}
		}

		if best != nil {
			best.Subject = proof

			continue
		}

		if at := clueHolder(spec.Things); at >= 0 {
			spec.Things[at].Clues = append(spec.Things[at].Clues, world.ClueSpec{Subject: proof, Text: "Wszystko tu wskazuje na jedno: " + proof + "."})
		}
	}
}

// alike counts the word stems two subjects share: the first five letters
// of every word of at least four, which rides over Polish endings.
func alike(a, b string) int {
	stems := func(text string) map[string]bool {
		found := map[string]bool{}
		for _, word := range strings.Fields(strings.ToLower(text)) {
			if runes := []rune(word); len(runes) >= 4 {
				found[string(runes[:min(5, len(runes))])] = true
			}
		}

		return found
	}

	shared := 0
	other := stems(b)
	for stem := range stems(a) {
		if other[stem] {
			shared++
		}
	}

	return shared
}

// clueHolder picks the thing to put a clue on: a fixed one without clues
// (a register, a cabinet), else any without, else the first.
func clueHolder(things []world.ThingSpec) int {
	for pass := 0; pass < 2; pass++ {
		for index, thing := range things {
			if len(thing.Clues) == 0 && (thing.Fixed || pass == 1) {
				return index
			}
		}
	}

	if len(things) > 0 {
		return 0
	}

	return -1
}

// playable tells what the second step left out: a case whose proof nobody
// and nothing can give cannot be solved.
func playable(spec world.Spec) error {
	if len(spec.Things) < 4 {
		return fmt.Errorf("jest tylko %d rzeczy w things, ma być 8 do 12", len(spec.Things))
	}

	known := map[string]bool{}
	for _, person := range spec.People {
		for _, clue := range person.Clues {
			known[strings.ToLower(clue.Subject)] = true
		}
	}

	for _, thing := range spec.Things {
		for _, clue := range thing.Clues {
			known[strings.ToLower(clue.Subject)] = true
		}
	}

	for _, step := range spec.Steps {
		if step.Fact != nil {
			known[strings.ToLower(step.Fact.Subject)] = true
		}
	}

	var missing []string
	for _, subject := range spec.Case.Proof {
		if !known[strings.ToLower(subject)] {
			missing = append(missing, subject)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("dowody z case.proof, których nikt i nic nie daje: %s; dodaj je jako clues rzeczy (subject dokładnie taki sam) albo fact kroku wait", strings.Join(missing, ", "))
	}

	return nil
}

// summary lists the first step's names for the second: rooms, people with
// the facts they hold, and the case.
func (d caseDraft) summary() string {
	var out strings.Builder

	fmt.Fprintf(&out, "Gra: %s. %s\nMiejsca (rooms): ", d.Name, d.Goal)

	names := make([]string, 0, len(d.Rooms))
	for _, room := range d.Rooms {
		names = append(names, room.Name)
	}

	out.WriteString(strings.Join(names, "; ") + "\nOsoby (people) i fakty (subject), które znają:\n")

	for _, person := range d.People {
		subjects := make([]string, 0, len(person.Clues))
		for _, clue := range person.Clues {
			subjects = append(subjects, clue.Subject)
		}

		fmt.Fprintf(&out, "- %s, zwykle w: %s; wie: %s\n", person.Name, person.Room, strings.Join(subjects, "; "))
	}

	if d.Case != nil {
		fmt.Fprintf(&out, "Sprawa: zleca %s, wskazać trzeba: %s. Dowody (proof), które gracz musi zdobyć: %s.", d.Case.Client, d.Case.Answer, strings.Join(d.Case.Proof, "; "))
	}

	return out.String()
}

// check tells what the model got wrong in the first step, in words it can
// act on.
func (d caseDraft) check() error {
	switch {
	case strings.TrimSpace(d.Name) == "":
		return errors.New("brakuje name")
	case len(d.Rooms) < 4:
		return fmt.Errorf("jest tylko %d miejsc w rooms, ma być 8 do 12", len(d.Rooms))
	case len(d.People) < 3:
		return fmt.Errorf("jest tylko %d osób w people, ma być 5 do 8", len(d.People))
	case d.Case == nil || strings.TrimSpace(d.Case.Answer) == "":
		return errors.New("brakuje case z answer: name osoby z people, którą trzeba wskazać")
	case !d.person(d.Case.Answer):
		return fmt.Errorf("case.answer %q nie jest name żadnej osoby z people", d.Case.Answer)
	case len(d.Case.Proof) < 2:
		return errors.New("case.proof ma mieć 3-5 subject faktów z clues")
	}

	return nil
}

// mend fixes what can be fixed without asking again: a culprit or client
// named a little differently than in people ("Kowalski" for "Jan
// Kowalski") is matched by the words of the name.
func (d *caseDraft) mend() {
	if d.Case == nil {
		return
	}

	for _, field := range []*string{&d.Case.Answer, &d.Case.Client} {
		if *field == "" || d.person(*field) {
			continue
		}

		if match, ok := d.closest(*field); ok {
			*field = match
		}
	}
}

// closest finds the one person who shares a word of at least three
// letters with the name.
func (d caseDraft) closest(name string) (string, bool) {
	var found []string

	for _, person := range d.People {
		for _, word := range strings.Fields(strings.ToLower(name)) {
			if len([]rune(word)) >= 3 && strings.Contains(strings.ToLower(person.Name), word) {
				found = append(found, person.Name)

				break
			}
		}
	}

	if len(found) == 1 {
		return found[0], true
	}

	return "", false
}

func (d caseDraft) person(name string) bool {
	for _, person := range d.People {
		if strings.EqualFold(strings.TrimSpace(person.Name), strings.TrimSpace(name)) {
			return true
		}
	}

	return false
}

// merge puts the two halves together. People and things that name a room
// the town lacks go to the first room rather than being lost.
func (d caseDraft) merge(town townDraft) world.Spec {
	rooms := append([]world.RoomSpec(nil), d.Rooms...)
	people := append([]world.PersonSpec(nil), d.People...)

	roomIndex := func(name string) int {
		for index, room := range rooms {
			if strings.EqualFold(strings.TrimSpace(room.Name), strings.TrimSpace(name)) {
				return index
			}
		}

		return -1
	}

	for index := range people {
		if at := roomIndex(people[index].Room); at >= 0 {
			people[index].Room = rooms[at].Name
		} else if len(rooms) > 0 {
			people[index].Room = rooms[0].Name
		}

		for hour, where := range people[index].Schedule {
			if at := roomIndex(where); at >= 0 {
				people[index].Schedule[hour] = rooms[at].Name
			} else {
				delete(people[index].Schedule, hour)
			}
		}
	}

	// An investigation is won by the accusation alone: a model that marks
	// its steps and events as winning would end the case at the first
	// conversation.
	steps := append([]world.StepSpec(nil), town.Steps...)
	for index := range steps {
		steps[index].Win = false
	}

	events := append([]world.EventSpec(nil), town.Events...)
	for index := range events {
		events[index].Win = false
	}

	things := make([]world.ThingSpec, 0, len(town.Things))
	for _, thing := range town.Things {
		things = append(things, thing.ThingSpec)

		at := roomIndex(thing.Room)
		if at < 0 && len(rooms) > 0 {
			at = 0
		}

		if at >= 0 {
			rooms[at].Things = append(rooms[at].Things, thing.Name)
		}
	}

	return world.Spec{
		Name:     d.Name,
		Intro:    d.Intro,
		Goal:     d.Goal,
		Rooms:    rooms,
		Things:   things,
		People:   people,
		Steps:    steps,
		Events:   events,
		Case:     d.Case,
		Deadline: d.Deadline,
		Loss:     d.Loss,
	}
}

// dump keeps the model's raw answer in the file NEURO_WORLD_DUMP names
// (with a suffix for a step of a two-step design), for looking into worlds
// that come out unplayable.
func dump(content, suffix string) {
	if path := os.Getenv("NEURO_WORLD_DUMP"); path != "" {
		_ = os.WriteFile(path+suffix, []byte(content), 0o644)
	}
}

// designBudget caps a design answer in tokens. A sound world needs fewer;
// a model that runs on past it is repeating itself, and without the cap a
// local one would do so for a quarter of an hour until the window is full.
const designBudget = 3000

// designTemperature is the sampling temperature of a design. At the
// model's default of 1 a small model, kept to the schema's grammar, falls
// into runs of tabs and spaces between the fields (the grammar allows
// them) and spends the budget on whitespace; cooler sampling keeps it on
// the words.
const designTemperature = 0.6

// design asks for an object by schema and, when the game rejects it, shows
// the model what was wrong and lets it mend the answer once.
func (c *Client) design(ctx context.Context, instructions string, schema map[string]any, brief string, accept func(content string) error) (string, error) {
	var rejected error

	for attempt := 0; attempt < 2; attempt++ {
		// The retry starts over with the complaint in the brief rather
		// than carrying the rejected answer along: a local model's window
		// is small, and its own long answer would crowd out the task.
		asked := brief
		if rejected != nil {
			asked += "\n\nGra odrzuciła poprzedni projekt: " + rejected.Error() + ". Zrób go od nowa bez tego błędu, kompletny i zgodny ze schematem."
		}

		content, err := c.chat(ctx, chatRequest{
			Model: c.Model,
			Messages: []message{
				{Role: "system", Content: instructions},
				{Role: "user", Content: asked},
			},
			MaxTokens:   designBudget,
			Temperature: designTemperature,
			ResponseFormat: map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "world",
					"strict": true,
					"schema": schema,
				},
			},
		})
		if err != nil {
			// An answer that did not fit the window is a design to redo
			// shorter, not a broken server.
			if !errors.Is(err, ErrCut) {
				return "", err
			}

			dump(content, ".cut")
			rejected = errors.New("odpowiedź była za długa i została ucięta; pisz zwięźle, opisy po jednym krótkim zdaniu")

			continue
		}

		if rejected = accept(content); rejected == nil {
			return content, nil
		}
	}

	return "", rejected
}
