package game

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestGuardReport runs the engine's guards over replies a model gave to
// scripted conversations (tools/eval/run.py output), to measure them
// offline before they go into the game: the echo of four and of three
// words, and the contradiction of the bio in the card. Run with
// NEURO_GUARD_REPLIES=t18-run21-s7.jsonl NEURO_GUARD_OUT=report.jsonl;
// NEURO_GUARD_SCENARIOS (the scenarios the replies answer, with their
// system prompts) gives the bio in the card, without it only the speaker's
// earlier replies in the talk are the bio.
func TestGuardReport(t *testing.T) {
	in, out := os.Getenv("NEURO_GUARD_REPLIES"), os.Getenv("NEURO_GUARD_OUT")
	if in == "" || out == "" {
		t.Skip("NEURO_GUARD_REPLIES and NEURO_GUARD_OUT not set")
	}

	systems := map[string]string{}
	if path := os.Getenv("NEURO_GUARD_SCENARIOS"); path != "" {
		for _, row := range readRows(t, path) {
			id, _ := row["id"].(string)
			system, _ := row["system"].(string)
			systems[id] = system
		}
	}

	// NEURO_GUARD_PRIOR holds the replies of the first talks of pairs:
	// what the speaker said in "x-p1" is the bio of "x-p2-dyn", as the
	// game would keep it, for the lines that contradict it.
	prior := map[string][]string{}
	if path := os.Getenv("NEURO_GUARD_PRIOR"); path != "" {
		for _, row := range readRows(t, path) {
			id, _ := row["id"].(string)
			replies, _ := row["replies"].([]any)
			for _, item := range replies {
				turn, _ := item.(map[string]any)
				reply, _ := turn["reply"].(string)
				prior[id] = append(prior[id], reply)
			}
		}
	}

	// NEURO_GUARD_BIOS is a JSON file of {id: [facts]}: the bio the Mind
	// read from the first talk of a pair, as the game would keep it for
	// the second.
	bios := map[string][]string{}
	if path := os.Getenv("NEURO_GUARD_BIOS"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(data, &bios); err != nil {
			t.Fatal(err)
		}
	}

	sink, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	told := regexp.MustCompile(`Co już o sobie mówił[ae]ś \(to prawda, trzymaj się tego\): ([^\n]*)`)
	sum := regexp.MustCompile(`\d+ ([a-ząęółśżźćń]+)[.,]`)
	keeper := regexp.MustCompile(`(?m)^(?:TWOJA POSTAĆ|Jesteś): ([^,.]+)`)

	for _, row := range readRows(t, in) {
		id, _ := row["id"].(string)
		replies, _ := row["replies"].([]any)
		system := systems[id]

		var bio []string
		if match := told.FindStringSubmatch(system); match != nil {
			bio = sentences(match[1])
		}

		speaker := ""
		if match := keeper.FindStringSubmatch(system); match != nil {
			speaker = strings.TrimSpace(match[1])
		}

		var history []Line
		var earlier []string

		// The currency is the word after a sum in the [GRA] blocks of the
		// talk ("36 korków."), for the price nobody asked for.
		currency := ""
		for _, item := range replies {
			turn, _ := item.(map[string]any)
			gra, _ := turn["gra"].(string)
			if match := sum.FindStringSubmatch(gra); match != nil {
				currency = match[1]

				break
			}
		}

		for k, item := range replies {
			turn, _ := item.(map[string]any)
			reply, _ := turn["reply"].(string)
			gra, _ := turn["gra"].(string)
			said, _ := turn["player"].(string)

			report := map[string]any{"id": id, "k": k, "reply": reply, "gm": strings.HasPrefix(gra, "Przebieg rundy"), "has_bio": len(bio) > 0}
			report["echo4"] = echoN(reply, history, 4)
			report["echo3"] = echoN(reply, history, 3)
			// The question and the price guards: "asks_any" is the old
			// rule (any question mark), "asks" the one that lets an echo of
			// the player's words pass; "unasked" a price in a line that was
			// to name none.
			report["asks_any"] = strings.Contains(gra, "nie pytaj") && strings.Contains(reply, "?")
			report["asks"] = strings.Contains(gra, "nie pytaj") && asks(reply, said)
			report["unasked"] = unasked(gra, false) && priceIn(reply, currency)
			report["currency"] = currency
			report["denies"] = contradicts(reply, bio, speaker)
			report["denies_self"] = contradicts(reply, earlier, speaker)
			if first, ok := strings.CutSuffix(id, "-p2-dyn"); ok {
				report["denies_prior"] = contradicts(reply, prior[first+"-p1"], speaker)
				report["denies_mind"] = contradicts(reply, bios[first+"-p1"], speaker)
			}

			data, _ := json.Marshal(report)
			sink.Write(append(data, '\n'))

			history = append(history, Line{Player: true, Text: ""}, Line{Player: false, Text: reply})
			earlier = append(earlier, reply)
		}
	}
}

func readRows(t *testing.T, path string) []map[string]any {
	t.Helper()

	source, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	var rows []map[string]any

	scanner := bufio.NewScanner(source)
	scanner.Buffer(nil, 1<<22)

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}

		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		rows = append(rows, row)
	}

	return rows
}
