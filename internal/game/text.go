package game

import (
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
			found = found || strings.HasPrefix(candidate, stem)
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

func join(names []string) string {
	return strings.Join(names, ", ")
}
