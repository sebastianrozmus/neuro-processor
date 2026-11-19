package game

import (
	"regexp"
	"slices"
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

var (
	// struck is an attack in a round's log, dice already dropped: who, how
	// and what follows the colon ("Ala atakuje: Goblin traci 5 zdrowia.").
	struck = regexp.MustCompile(`^(.+?) (atakuje|rzuca)(?: \(trafienie krytyczne!\))?: (.+)$`)
	// struckDown is the one who loses health in a sentence.
	struckDown = regexp.MustCompile(`^(?:[^:]*: )?(.+?) traci \d+ zdrowia`)
	// missed are the words a narration tells a miss with.
	missed = regexp.MustCompile(`(?i)(pudł|chybi)`)
	// clauses cut a narration where its subject may change: after
	// punctuation and before "a", "ale" and their kin.
	clauses = regexp.MustCompile(`[.,;:!?…]|\s(?:a|ale|lecz|zaś|za to|choć|gdy|zanim)\s`)
)

// subjectWords may stand between the names a clause opens with: "Rak i
// Nurek", "Pirat też".
var subjectWords = []string{"i", "oraz", "a", "też", "także", "również", "sam", "sama", "znów", "znowu", "nawet", "potem"}

// flipped returns the one a narration has missing although in the round
// it hit and never missed, or did not attack at all: "Rak, Szczur i
// Nurek pudłują" after Nurek drew blood. Only the names a clause opens
// with count, before any other word ("Ola zamachuje się na rój i chybia"
// is Ola's miss, not the swarm's), and only a clause with no number in it
// ("pudłuje, ale za 4" may be another's blow). Names are matched word by
// word as written in the round, so only in the nominative; words two
// fighters share ("Wilk"), words after a preposition ("z Tratwy") and
// genitive ones ("Komarów") are no one's. On test18 this caught 18 of 149
// rounds both judges found heavily wrong and none of 1450 clean ones.
func flipped(text, scene string) string {
	hit, miss, attackers, names := strikes(scene)

	owner := map[string]string{}
	shared := map[string]bool{}

	for _, name := range names {
		words := strings.Fields(name)
		for index, word := range words {
			if utf8.RuneCountInString(word) < 3 || !capital(word) || index > 0 && !capital(words[index-1]) {
				continue
			}

			word = strings.ToLower(word)
			if strings.HasSuffix(word, "ów") || strings.HasSuffix(word, "ego") {
				continue
			}

			if other, ok := owner[word]; ok && other != name {
				shared[word] = true
			}

			owner[word] = name
		}
	}

	for _, clause := range clauses.Split(text, -1) {
		at := missed.FindStringIndex(clause)
		if at == nil || digits.MatchString(clause) || strings.Contains(clause, "traci") {
			continue
		}

		for _, word := range strings.Fields(clause[:at[0]]) {
			word = strings.ToLower(word)

			name, ok := owner[word]
			if !ok {
				if slices.Contains(subjectWords, word) {
					continue
				}

				break
			}

			if !shared[word] && (hit[name] && !miss[name] || !slices.Contains(attackers, name)) {
				return name
			}
		}
	}

	return ""
}

// strikes reads a round's attacks: who hit at least once, who missed at
// least once, who attacked, and every name in a blow, struck or striking.
// A spell hits those who lose health in the sentences after it.
func strikes(scene string) (hit, miss map[string]bool, attackers, names []string) {
	hit, miss = map[string]bool{}, map[string]bool{}
	sentences := roundSentences(withoutDice(scene))

	for index, sentence := range sentences {
		if down := struckDown.FindStringSubmatch(sentence); down != nil && !slices.Contains(names, down[1]) {
			names = append(names, down[1])
		}

		match := struck.FindStringSubmatch(sentence)
		if match == nil {
			continue
		}

		who := match[1]
		if !slices.Contains(attackers, who) {
			attackers = append(attackers, who)
			names = append(names, who)
		}

		next := ""
		if index+1 < len(sentences) {
			next = sentences[index+1]
		}

		switch {
		case strings.Contains(match[3], " traci "), bareBlow.MatchString(next):
			hit[who] = true
		case strings.HasPrefix(next, "Pudło"):
			miss[who] = true

			// The one missed is a name too; what is cast is not.
			if target := strings.TrimSuffix(match[3], "."); match[2] == "atakuje" && !slices.Contains(names, target) {
				names = append(names, target)
			}
		}
	}

	return hit, miss, attackers, names
}

// capital reports whether a word begins with a capital letter.
func capital(word string) bool {
	first, _ := utf8.DecodeRuneInString(word)

	return unicode.IsUpper(first)
}
