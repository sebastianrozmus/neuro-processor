package game

import (
	"slices"
	"strings"
	"unicode"
)

// A companion who told of a brother Józek yesterday has no business
// saying today that they never had one, nor calling him Michał: on the
// run21 pairs (D:\neuro-train\run22\PLAN.md) every contradiction the
// judges counted was of kin: a husband dead and then never married,
// children denied and then a son, a brother renamed. What the bio says of
// kin is structured enough to check a line against: which family, whether
// they are, are not, live or are dead, and their name. Where one comes
// from, the scar and the trade are left to the model.

// kinClaim is what a text says of one of the speaker's kin.
type kinClaim struct {
	// family indexes kinFamilies; kind tells a brother from a sister
	// within it, "" for the family in general ("rodzeństwo", "dzieci").
	family int
	kind   string
	// state is one of kinHas, kinNone, kinAlive, kinDead, kinLost.
	state string
	name  string
}

const (
	kinHas   = "jest"
	kinNone  = "nie ma"
	kinAlive = "żyje"
	kinDead  = "nie żyje"
	// kinLost is "nie mam już": gone, as a dead one is.
	kinLost = "stracił"
)

var (
	// kinGeneral are the words of a family in general, folded beginnings.
	kinGeneral = []string{"rodzenstw", "dzieci", "dzieck", "potomk", "potomstw", "rodzic", "malzon", "malzen"}
	// deathStems begin the words of dying, folded.
	deathStems = []string{"zmarl", "umarl", "zginal", "zginel", "zabil", "zabit", "pochowal", "pochowan", "utonal", "utonel", "polegl", "nieboszcz", "stracil", "zabral", "zabrala"}
	// hadVerbs follow "nie" in a denial of kin: "nie mam dzieci", "dzieci
	// nie było", "nie doczekałam się syna".
	hadVerbs = []string{"mam", "mialam", "mialem", "miala", "mial", "mialo", "mialy", "mieli", "bylo", "byla", "byl", "ma", "posiadam", "doczekalam", "doczekalem", "dochowalam", "dochowalem", "starczylo", "zostalo", "urodzilam", "urodzil", "zalozylam", "zalozylem", "wzielam", "wzialem"}
	// noneWords deny kin on their own: "żadnego brata", "brak dzieci".
	noneWords = []string{"zadnego", "zadnej", "zadnych", "zadna", "zaden", "zadne", "brak", "ani"}
	// ifWords open a sentence that supposes rather than tells.
	ifWords = []string{"gdyby", "gdybym", "gdybys", "jakby", "jakbym", "chcialabym", "chcialbym", "wolalabym", "wolalbym", "moglabym", "moglbym", "marze", "marzylam", "marzylem"}
	// denyIdioms open an answer that denies the kin asked about: "Żonę?
	// A skąd!", "Syna? Ani jednego."
	denyIdioms = []string{"skad", "skadze", "ani", "gdzie"}
	// likeWords before a word of kin make it a simile or someone else's:
	// "traktuję cię jak syna", "twój brat".
	likeWords = []string{"jak", "niczym", "niby", "twoj", "twoja", "twoje", "twojego", "twojej", "twoim", "twoich", "wasz", "wasza", "wasze", "panski", "panska"}
)

// kinKind tells a brother from a sister within a family: the folded
// word's stem, "" for the family in general.
func kinKind(word string) string {
	if slices.ContainsFunc(kinGeneral, func(stem string) bool { return strings.HasPrefix(word, stem) }) {
		return ""
	}

	switch {
	case word == "maz" || strings.HasPrefix(word, "mez") || strings.HasPrefix(word, "wdow") || strings.HasPrefix(word, "zamezn"):
		return "maz"
	case strings.HasPrefix(word, "zon"):
		return "zon"
	case strings.HasPrefix(word, "ojc") || strings.HasPrefix(word, "tat"):
		return "ojc"
	case strings.HasPrefix(word, "matk") || strings.HasPrefix(word, "mam"):
		return "mat"
	case strings.HasPrefix(word, "brac") || strings.HasPrefix(word, "brat"):
		return "bra"
	}

	return word[:min(3, len(word))]
}

// kinClaims lists what a text claims of the speaker's kin, sentence by
// sentence. A question claims nothing ("Syna? Jakiego syna?"), nor does a
// supposition, a simile or the player's own kin.
func kinClaims(text, speaker string) []kinClaim {
	if _, rest, tagged := strings.Cut(text, "] "); tagged && strings.HasPrefix(text, "[") {
		text = rest
	}

	skip := strings.Fields(Fold(speaker))

	var claims []kinClaim

	// A one-word question is the topic of the sentence after it: "Mąż?
	// Zmarł dawno", "Syna? On nie zejdzie na dół".
	carried := ""

	for _, sentence := range sentences(text) {
		tokens := strings.Fields(dropVocatives(sentence))
		folded := make([]string, len(tokens))
		// comma marks the tokens a clause ends with: a denial does not
		// reach past one ("nie ma kogo uczyć, syn został na górze").
		comma := make([]bool, len(tokens))
		for index, token := range tokens {
			folded[index] = strings.Join(strings.Fields(Fold(token)), "")
			comma[index] = strings.ContainsAny(token[len(token)-1:], ",;:")
		}

		if strings.HasSuffix(sentence, "?") {
			carried = ""
			if len(folded) == 1 && kinFamily(folded[0]) >= 0 && kinFamily(folded[0]) < kinOnly && !slices.Contains(skip, folded[0]) {
				carried = folded[0]
			}

			continue
		}

		if slices.ContainsFunc(folded, func(word string) bool { return slices.Contains(ifWords, word) }) {
			carried = ""

			continue
		}

		found := false

		for index, word := range folded {
			family := kinFamily(word)
			if family < 0 || family >= kinOnly || slices.Contains(skip, word) {
				continue
			}

			if index > 0 && slices.Contains(likeWords, folded[index-1]) || index > 1 && slices.Contains(likeWords, folded[index-2]) && !slices.Contains(hadVerbs, folded[index-1]) {
				continue
			}

			found = true
			from, to := max(0, index-4), min(len(folded), index+5)
			state := kinState(word, folded[from:to], comma[from:to], index-from)

			// "Miałam męża, a dzieci nie." ends in a denial.
			if index+2 == len(folded) && folded[index+1] == "nie" && state == kinHas {
				state = kinNone
			}

			claims = append(claims, kinClaim{family: family, kind: kinKind(word), state: state, name: kinName(tokens, folded, index, skip)})
		}

		// "Żonę? A skąd!", "Syna? Ani jednego." deny.
		if !found && carried != "" {
			state := kinState(carried, folded[:min(len(folded), 5)], comma[:min(len(folded), 5)], -1)
			if slices.ContainsFunc(folded[:min(len(folded), 3)], func(word string) bool { return slices.Contains(denyIdioms, word) }) {
				state = kinNone
			}

			claims = append(claims, kinClaim{family: kinFamily(carried), kind: kinKind(carried), state: state})
		}

		carried = ""
	}

	return claims
}

// sentences splits a text into sentences, each keeping its final mark.
func sentences(text string) []string {
	var out []string
	start := 0

	for index, char := range text {
		if strings.ContainsRune(".!?", char) {
			if sentence := strings.TrimSpace(text[start : index+1]); sentence != "" {
				out = append(out, sentence)
			}

			start = index + 1
		}
	}

	if sentence := strings.TrimSpace(text[start:]); sentence != "" {
		out = append(out, sentence)
	}

	return out
}

// butWords cut a denial off from the kin after them: "siostry nie mam,
// tylko brata" denies no brother.
var butWords = []string{"tylko", "ale", "a", "lecz", "jedynie", "za", "oprocz", "procz"}

// kinState reads, around a word of kin, whether they are, are not, live
// or are dead: "Nie żyje, dawno", "dzieci nie było", "żadnego brata",
// "nie mam już męża".
func kinState(word string, window []string, comma []bool, at int) string {
	if strings.HasPrefix(word, "wdow") {
		return kinDead
	}

	dead, alive, none, lost := false, false, false, false

	for index, other := range window {
		next := ""
		if index+1 < len(window) {
			next = window[index+1]
		}

		// A denial reaches the kin before it, and the kin right after the
		// verb unless a "tylko", an "ale" or a comma stands between.
		denies := func() bool {
			if at < index {
				return true
			}

			return !slices.ContainsFunc(window[min(index+2, at):at], func(between string) bool { return slices.Contains(butWords, between) }) &&
				!slices.Contains(comma[index+1:at], true)
		}

		switch {
		case slices.ContainsFunc(deathStems, func(stem string) bool { return strings.HasPrefix(other, stem) }):
			dead = true
		case other == "nie" && (strings.HasPrefix(next, "zyj") || next == "zyl" || next == "zyla"):
			dead = true
		case strings.HasPrefix(other, "zyj") && (index == 0 || window[index-1] != "nie"):
			alive = true
		case slices.Contains(noneWords, other) && (index < at && at-index <= 2 || index > at && index-at <= 2), other == "bez" && index+1 == at:
			none = true
		case other == "nie" && slices.Contains(hadVerbs, next) && denies():
			none = true
		}

		lost = lost || other == "juz"
	}

	switch {
	case dead:
		return kinDead
	case none && lost:
		return kinLost
	case none:
		return kinNone
	case alive:
		return kinAlive
	}

	return kinHas
}

// kinName is the name given right after a word of kin, folded: "brata,
// Józka", "syn Jan". The speaker's own name and the polite capitals are
// no name of kin.
func kinName(tokens, folded []string, at int, skip []string) string {
	for index := at + 1; index < min(len(tokens), at+3); index++ {
		raw := strings.TrimLeft(tokens[index], "„«\"(")
		if raw == "" {
			continue
		}

		if first := []rune(raw)[0]; !unicode.IsUpper(first) {
			if strings.Trim(tokens[index], ".,!?;:") == "" {
				continue
			}

			return ""
		}

		word := folded[index]
		if len(word) < minQueryWord || slices.Contains(skip, word) || slices.Contains(commonCapitals, word) || kinFamily(word) != -1 {
			return ""
		}

		return word
	}

	return ""
}

// sameName reports two folded names that are one name in two cases:
// "jozek" and "jozka", "jan" and "jana".
func sameName(a, b string) bool {
	a, b = fleeting(a), fleeting(b)
	short := min(len(a), len(b))
	if short < minQueryWord {
		return a == b
	}

	return strings.HasPrefix(a, b[:short]) || strings.HasPrefix(b, a[:short]) || strings.HasPrefix(a, Stem(b)) && strings.HasPrefix(b, Stem(a))
}

// contradicts returns the fact of the bio a line contradicts about the
// speaker's kin, or "": the bio has them and the line says there were
// none, the bio has none and the line has them, the bio has them alive
// and the line dead or the other way round, or the line names them
// otherwise. A brother and a sister are each their own; "rodzeństwo"
// and "dzieci" are any of theirs.
func contradicts(line string, bio []string, speaker string) string {
	said := kinClaims(line, speaker)
	if len(said) == 0 {
		return ""
	}

	for _, fact := range bio {
		for _, known := range kinClaims(fact, speaker) {
			for _, claim := range said {
				if claim.family != known.family || claim.kind != "" && known.kind != "" && claim.kind != known.kind {
					continue
				}

				switch {
				case claim.state == kinNone && known.state != kinNone && known.state != kinLost,
					known.state == kinNone && claim.state != kinNone && claim.state != kinLost,
					known.state == kinAlive && (claim.state == kinDead || claim.state == kinLost),
					known.state == kinDead && claim.state == kinAlive,
					known.name != "" && claim.name != "" && !sameName(known.name, claim.name):
					return fact
				}
			}
		}
	}

	return ""
}
