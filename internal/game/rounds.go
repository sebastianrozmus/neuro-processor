package game

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// PartedRound is how many events a round needs to be told in parts: on
	// test18 a master telling five or more in one go missed or flipped
	// one about three times as often as with two (15% heavy errors
	// against 4.8%).
	PartedRound = 5
	// partEvents is how many events one part of a round holds at most,
	// unless the round is so long that maxParts parts must take more.
	partEvents = 3
	// maxParts is how many calls a round may cost the master at most.
	maxParts = 3

	// RoundPrefix opens the master's brief of a round.
	RoundPrefix = "Przebieg rundy: "
	// ReplyLimit closes every brief: how long the answer may be.
	ReplyLimit = "Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi."
	// PartLimit closes the brief of one part of a round told in parts.
	PartLimit = "Odpowiedz najwyżej dwoma krótkimi zdaniami. Liczby pisz cyframi."
)

var (
	// sequel are the sentences of a round that tell what came of the one
	// before rather than something new: a blow's damage, a fall, a miss,
	// a note on a hit, a spell's effect.
	sequel = regexp.MustCompile(`^(Pudło\b|Odpoczynek w osadzie|Kwas przeżera|Trafienie krytyczne|Jad działa!|Jad przestaje|Rzut obronny udany|Drużyna odzyskuje)|` +
		` pada(\.| nieprzytomn| bez sił)`)
	// bareBlow is damage with no one dealing it in the same sentence: one
	// of a spell's victims ("Goblin traci 9 zdrowia.").
	bareBlow = regexp.MustCompile(`^[^:]* traci \d+ zdrowia`)
	// herald are sentences announcing a blow that the next sentence, when
	// it is the same attacker's, tells: "Czas na decyzję minął: Ala
	// atakuje." then "Ala atakuje: Goblin traci 4 zdrowia.".
	herald = regexp.MustCompile(`^Czas na decyzję minął: (.+) atakuje\.$|^(.+) wyprowadza atak okazyjny`)
	// ending opens the close of a fight, told after every event: a boss's
	// artifact, the victory, the loot on the ground.
	ending = regexp.MustCompile(`^(Zwycięstwo!|Na ziemi leży: )| zostawia artefakt: `)
)

// RoundEvents cuts the log of a round into its events, in order: an
// action with what came of it ("Ala atakuje: Goblin traci 5 zdrowia.
// Goblin pada."). The close of a won fight, from the victory or a boss's
// artifact on, is one event more, always the last. The events joined with
// spaces give the log back. It reads the text the rules wrote, with dice or
// without, so the game and the evaluation of frozen rounds cut alike.
func RoundEvents(scene string) []string {
	var (
		events   []string
		heralded string
		closing  bool
		opens    bool
	)

	for _, sentence := range roundSentences(scene) {
		switch {
		case closing:
			opens = false
		case ending.MatchString(sentence):
			closing, opens = true, true
		default:
			opens = (heralded == "" || !strings.HasPrefix(sentence, heralded+" atakuje")) &&
				!sequel.MatchString(sentence) && !bareBlow.MatchString(sentence)
		}

		if opens || len(events) == 0 {
			events = append(events, sentence)
		} else {
			events[len(events)-1] += " " + sentence
		}

		heralded = ""
		if match := herald.FindStringSubmatch(sentence); match != nil {
			heralded = match[1] + match[2]
		}
	}

	return events
}

// RoundParts is a round as the master tells it: the whole log when it holds
// fewer than PartedRound events, otherwise consecutive runs of whole
// events, at most partEvents in each, as even as can be, longer ones first,
// and never more than maxParts runs (then they take more each: 10 events go
// 4+3+3). The close of a fight stays in the last part.
func RoundParts(scene string) []string {
	events := RoundEvents(scene)
	if len(events) < PartedRound {
		return []string{scene}
	}

	count := min(maxParts, (len(events)+partEvents-1)/partEvents)
	parts := make([]string, 0, count)

	for index, start := 0, 0; index < count; index++ {
		size := len(events) / count
		if index < len(events)%count {
			size++
		}

		parts = append(parts, strings.Join(events[start:start+size], " "))
		start += size
	}

	return parts
}

// roundSentences splits a round's log after every full stop or exclamation
// mark that a space and a capital follow, outside brackets: the dice
// "(k20: 20, trafienie krytyczne!)" and names in «» stay whole.
func roundSentences(scene string) []string {
	var (
		sentences []string
		depth     int
		start     int
	)

	for index, char := range scene {
		switch char {
		case '(', '«':
			depth++
		case ')', '»':
			depth = max(0, depth-1)
		case '.', '!':
			rest := scene[index+1:]
			if depth > 0 || !strings.HasPrefix(rest, " ") {
				continue
			}

			next, _ := utf8.DecodeRuneInString(strings.TrimLeft(rest, " "))
			if !unicode.IsUpper(next) {
				continue
			}

			sentences = append(sentences, strings.TrimSpace(scene[start:index+1]))
			start = index + 1
		}
	}

	if last := strings.TrimSpace(scene[start:]); last != "" {
		sentences = append(sentences, last)
	}

	return sentences
}

// Limit is the closing line of a brief: how long the answer may be.
func (d Decision) Limit() string {
	if d.Part {
		return PartLimit
	}

	return ReplyLimit
}
