package game

import (
	"slices"
	"strings"
	"unicode"
)

// A typed word matches a name by its stem: the word without its last
// letter, but three to four letters long. That is short enough to survive
// Polish declension ("mieczem" finds "Miecz", "falę" finds "Fali").
const (
	minStem      = 3
	maxStem      = 4
	minQueryWord = 3
)

var folding = strings.NewReplacer(
	"ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n", "ó", "o", "ś", "s", "ź", "z", "ż", "z",
)

// Fold lowercases text, strips Polish diacritics and turns punctuation into
// spaces, so that typed text can be compared with names loosely.
func Fold(text string) string {
	folded := folding.Replace(strings.ToLower(text))

	return strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			return char
		}

		return ' '
	}, folded)
}

// fleeting drops the vowel that Polish loses in declension (Marek, Marka;
// Paweł, Pawła): the e before a final consonant. After a soft i it goes
// too (kupiec, kupca; ojciec, ojca).
func fleeting(word string) string {
	if base, found := strings.CutSuffix(word, "iec"); found && len(base) >= 2 {
		return strings.TrimSuffix(base, "c") + "c"
	}

	runes := []rune(word)
	if n := len(runes); n >= 4 && runes[n-2] == 'e' && !strings.ContainsRune("aeiouy", runes[n-1]) {
		return string(runes[:n-2]) + string(runes[n-1])
	}

	return word
}

// Matches reports whether every meaningful word of the query begins some
// word of the name. Short words such as prepositions are ignored.
func Matches(name, query string) bool {
	nameWords := strings.Fields(Fold(name))
	matched := false

	for _, word := range strings.Fields(Fold(query)) {
		if len(word) < minQueryWord {
			continue
		}

		stem, found := Stem(word), false
		for _, candidate := range nameWords {
			found = found || strings.HasPrefix(candidate, stem) || strings.HasPrefix(fleeting(candidate), stem)
		}

		if !found {
			return false
		}

		matched = true
	}

	return matched
}

// Stem returns the beginning of a folded word that its other grammatical
// forms share.
func Stem(word string) string {
	return word[:min(len(word), max(minStem, min(maxStem, len(word)-1)))]
}

// inflections are what Polish puts after a stem to inflect a noun or an
// adjective, folded: "krypty", "kryptach", "Krateru", "popielnego".
var inflections = []string{
	"", "a", "e", "i", "y", "o", "u", "ie", "ia", "iu", "io", "ii", "ach", "ami", "om", "ow", "owi", "owie", "em", "ej", "ego", "emu", "ym", "ymi", "ych",
	"iach", "iami", "iom", "iow", "iem", "iego", "iemu", "im", "imi", "ich", "iej",
}

// alternations are the consonants a stem swaps before an ending:
// "kolczuga" and "kolczudze", "krater" and "kraterze".
var alternations = [][2]string{{"g", "dz"}, {"k", "c"}, {"ch", "sz"}, {"r", "rz"}, {"t", "c"}, {"d", "dz"}, {"st", "sc"}}

// formOf reports whether a player's word is a form of a word of a name:
// the two are one stem of three letters or more with an ending each, or
// with consonants that alternate before the ending. A word that only
// begins like the name is none: "kawał" is no "Kawa", "służbę" no
// "Śluza", "dzieci" no "dzienny" and "strzygę" no "Strzygoń", while
// "kurtek" is "kurtka" and "kolczudze" is "Kolczuga".
func formOf(word, part string) bool {
	for _, w := range []string{word, fleeting(word)} {
		for _, p := range []string{part, fleeting(part)} {
			for same := common(w, p); same >= minStem; same-- {
				if inflects(w[same:], p[same:]) {
					return true
				}
			}
		}
	}

	return false
}

// inflects reports whether two remainders of one stem are endings, maybe
// after consonants that alternate.
func inflects(a, b string) bool {
	if slices.Contains(inflections, a) && slices.Contains(inflections, b) {
		return true
	}

	for _, pair := range alternations {
		for _, swap := range [][2]string{pair, {pair[1], pair[0]}} {
			if strings.HasPrefix(a, swap[0]) && strings.HasPrefix(b, swap[1]) && slices.Contains(inflections, a[len(swap[0]):]) && slices.Contains(inflections, b[len(swap[1]):]) {
				return true
			}
		}
	}

	return false
}

func join(names []string) string {
	return strings.Join(names, ", ")
}

// bullets lists names one under another, each after a dot, the way a
// list is easier to read than a long sentence.
func bullets(names []string) string {
	return "\n  • " + strings.Join(names, "\n  • ")
}
