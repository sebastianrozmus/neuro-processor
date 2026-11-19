// Command gmsplit prepares frozen Game Master rounds for measuring the
// telling of long rounds in parts, with no game and no model: it cuts every
// round of a scenario file (the tools/eval/run.py format) exactly as the
// game does, with game.RoundParts.
//
//	go run ./tools/gmsplit -in test18.jsonl -out out/gmsplit
//
// It writes three files to -out:
//   - split.jsonl: all scenarios, every GM round of game.PartedRound events
//     or more replaced by its parts;
//   - long.jsonl: GM scenarios with only those long rounds, whole (baseline);
//   - long-split.jsonl: the same long rounds, in parts.
//
// Every GM turn written gets "round" (its index among the scenario's
// original turns) and "part" ([k, n], 1-based; [1, 1] for a whole round),
// so replies, which run.py saves in turn order, can be joined back per
// round. Other scenarios are copied as they are.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
)

type scenario = map[string]json.RawMessage

func main() {
	in := flag.String("in", "", "scenario JSONL to read")
	out := flag.String("out", "", "directory to write split.jsonl, long.jsonl and long-split.jsonl to")
	flag.Parse()

	if *in == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "gmsplit:", err)
		os.Exit(1)
	}
}

// stats counts what was cut.
type stats struct {
	rounds, long, foreign int
	events, parts         map[int]int
}

func run(in, out string) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	var (
		split, long, longSplit bytes.Buffer
		counts                 = stats{events: map[int]int{}, parts: map[int]int{}}
	)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1<<20), 64<<20)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var item scenario
		if err := json.Unmarshal(line, &item); err != nil {
			return err
		}

		var gm bool
		_ = json.Unmarshal(item["gm"], &gm)

		if !gm {
			split.Write(line)
			split.WriteByte('\n')

			continue
		}

		all, whole, parted, err := cut(item["turns"], &counts)
		if err != nil {
			return fmt.Errorf("%s: %w", item["id"], err)
		}

		if err := write(&split, item, all); err != nil {
			return err
		}

		if len(whole) == 0 {
			continue
		}

		if err := write(&long, item, whole); err != nil {
			return err
		}

		if err := write(&longSplit, item, parted); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	for name, buffer := range map[string]*bytes.Buffer{"split.jsonl": &split, "long.jsonl": &long, "long-split.jsonl": &longSplit} {
		if err := os.WriteFile(filepath.Join(out, name), buffer.Bytes(), 0o644); err != nil {
			return err
		}
	}

	report(counts)

	return nil
}

// cut goes through the rounds of one GM scenario: all of them with the long
// ones in parts, the long ones whole, and the long ones in parts.
func cut(raw json.RawMessage, counts *stats) (all, whole, parted []map[string]any, err error) {
	var turns []map[string]any
	if err := json.Unmarshal(raw, &turns); err != nil {
		return nil, nil, nil, err
	}

	for index, turn := range turns {
		gra, _ := turn["gra"].(string)
		scene, ok := sceneOf(gra)

		if !ok {
			counts.foreign++
			all = append(all, marked(turn, gra, index, 1, 1))

			continue
		}

		counts.rounds++
		counts.events[len(game.RoundEvents(scene))]++

		parts := game.RoundParts(scene)
		if len(parts) == 1 {
			all = append(all, marked(turn, gra, index, 1, 1))

			continue
		}

		counts.long++
		counts.parts[len(parts)]++
		whole = append(whole, marked(turn, gra, index, 1, 1))

		for number, part := range parts {
			piece := marked(turn, game.RoundPrefix+part+" "+game.PartLimit, index, number+1, len(parts))
			all, parted = append(all, piece), append(parted, piece)
		}
	}

	return all, whole, parted, nil
}

// sceneOf is the log of a round inside a brief the game wrote whole.
func sceneOf(gra string) (string, bool) {
	gra = strings.TrimSpace(gra)
	if !strings.HasPrefix(gra, game.RoundPrefix) || !strings.HasSuffix(gra, game.ReplyLimit) {
		return "", false
	}

	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(gra, game.RoundPrefix), game.ReplyLimit)), true
}

// marked is a copy of a turn with its brief and its place in the round.
func marked(turn map[string]any, gra string, round, part, parts int) map[string]any {
	copied := make(map[string]any, len(turn)+2)
	for key, value := range turn {
		copied[key] = value
	}

	copied["gra"], copied["round"], copied["part"] = gra, round, []int{part, parts}

	return copied
}

// write adds a scenario with the given turns as one line.
func write(buffer *bytes.Buffer, item scenario, turns []map[string]any) error {
	copied := make(map[string]any, len(item))
	for key, value := range item {
		copied[key] = value
	}

	copied["turns"] = turns

	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	return encoder.Encode(copied)
}

func report(counts stats) {
	fmt.Printf("rundy MG: %d, w częściach (od %d zdarzeń): %d\n", counts.rounds, game.PartedRound, counts.long)

	if counts.foreign > 0 {
		fmt.Printf("tury MG bez „%s…%s”: %d (bez zmian)\n", game.RoundPrefix, game.ReplyLimit, counts.foreign)
	}

	fmt.Println("zdarzeń w rundzie: " + histogram(counts.events))
	fmt.Println("części na długą rundę: " + histogram(counts.parts))
}

func histogram(counts map[int]int) string {
	keys := make([]int, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}

	sort.Ints(keys)

	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, fmt.Sprintf("%d→%d", key, counts[key]))
	}

	return strings.Join(fields, ", ")
}
