package game

import (
	"regexp"
	"strings"
	"unicode"
)

// Why a fact the mind noted in a keeper's line is turned away. A small mind
// asked "is there a new fact here?" says yes to nearly every line with a
// name in it: it notes what the keeper denied knowing, the prices of the
// wares, and the same well-known subject again and again in other words.
const (
	staleDenial = "zaprzeczenie"
	staleNumber = "liczba"
	staleWare   = "towar"
	staleNamed  = "imię spoza kwestii"
	staleKnown  = "znany temat"
)

// denial is what a keeper says when he does not know or will not tell;
// folded, so without Polish letters.
var denial = regexp.MustCompile(`\bnie (znam|wiem|wyznam|wyjawie|powiem|zdradze|zdradzam|slyszal\w*|mam pojecia|pamietam)\b|\bnikt (nie wie|nie zna)\b|\bnie jest znan\w*|\bnieznan\w*`)

var digit = regexp.MustCompile(`\d`)

// Known is what the world already knows, for telling news from old news:
// the subjects and texts of its facts, the wares of the keeper speaking
// and the names of its people.
type Known struct {
	Subjects []string
	Texts    []string
	Wares    []string
	Names    []string
}

// Stale reports why a fact noted in a line is no news, or "" when it is:
//   - the line denies it ("Ozryka nie znam" is not a fact about Ozryk);
//   - it has a number, which is a price or an amount the rules own;
//   - it is about a ware on the keeper's shelf;
//   - its text names somebody the line does not;
//   - its subject is already known, or the quote speaks of a known subject
//     or its text is about one, unless the quote brings a name the world
//     has not heard yet.
func Stale(spec FactSpec, line string, known Known) string {
	quote, text := strings.TrimSpace(Fold(spec.Quote)), Fold(spec.Text)
	folded := Fold(line)

	switch {
	case denial.MatchString(sentenceWith(line, quote)) || denial.MatchString(text):
		return staleDenial
	case digit.MatchString(quote) || digit.MatchString(text):
		return staleNumber
	}

	for _, ware := range known.Wares {
		if Matches(ware, spec.Subject) || Matches(spec.Subject, ware) || Matches(quote, ware) {
			return staleWare
		}
	}

	words := strings.Fields(folded)
	for _, name := range properNames(spec.Text) {
		if !heardOf(words, name) {
			return staleNamed
		}
	}

	if fresh := freshNames(line, quote, known); len(fresh) > 0 {
		return ""
	}

	for _, subject := range known.Subjects {
		if Matches(subject, spec.Subject) || Matches(spec.Subject, subject) || Matches(quote, subject) || Matches(text, subject) {
			return staleKnown
		}
	}

	return ""
}

// sentenceWith is the folded sentence of the line the folded quote is in:
// a denial elsewhere in the line ("Ozryka nie znam. Kurhany leżą na
// północy.") says nothing about the quote. A quote over more sentences
// gets the whole line.
func sentenceWith(line, quote string) string {
	for _, sentence := range stops.Split(line, -1) {
		if folded := strings.TrimSpace(Fold(sentence)); quote != "" && strings.Contains(folded, quote) {
			return folded
		}
	}

	return Fold(line)
}

var stops = regexp.MustCompile(`[.!?…]+`)

// properNames are the folded words of a text written with a capital
// letter, other than the first word of a sentence.
func properNames(text string) []string {
	var names []string

	start := true

	for _, word := range strings.Fields(text) {
		trimmed := strings.TrimFunc(word, func(r rune) bool { return !unicode.IsLetter(r) })
		runes := []rune(trimmed)

		if len(runes) >= minQueryWord && unicode.IsUpper(runes[0]) && !start {
			names = append(names, Fold(trimmed))
		}

		start = strings.ContainsAny(word, ".!?…")
	}

	return names
}

// freshNames are the names in the quoted part of the line the world has
// not heard: in none of its subjects, people or the names in its facts.
// They are read from the line, where it is clear which capital letter
// starts a sentence.
func freshNames(line, quote string, known Known) []string {
	var heard []string
	for _, name := range append(append([]string{}, known.Subjects...), known.Names...) {
		heard = append(heard, strings.Fields(Fold(name))...)
	}

	for _, text := range known.Texts {
		heard = append(heard, properNames(text)...)
	}

	var fresh []string

	quoted := strings.Fields(quote)

	for _, name := range properNames(line) {
		if namedIn(quoted, name) && !heardOf(heard, name) {
			fresh = append(fresh, name)
		}
	}

	return fresh
}

// heardOf reports whether a folded name is a form of one of the heard
// words, either way round: "kurhanach" of "kurhany", "krolow" of "krolow".
func heardOf(heard []string, name string) bool {
	if namedIn(heard, name) {
		return true
	}

	for _, word := range heard {
		if len(word) > minQueryWord && nameOK([]string{name}, word) {
			return true
		}
	}

	return false
}

// namedIn reports whether a form of the folded name is among the words.
func namedIn(words []string, name string) bool {
	return len(name) < minQueryWord || nameOK(words, name)
}

// news is what the world knows, for telling news a keeper told the named
// player from old news. The caller holds the lock.
func (r *Realm) news(name string) Known {
	var known Known

	for _, fact := range r.wisdom.facts {
		known.Subjects = append(known.Subjects, fact.Subject)
		known.Texts = append(known.Texts, fact.Text)
	}

	if session := r.session(name); session != nil {
		for _, ware := range session.Wares() {
			known.Wares = append(known.Wares, ware.Name)
		}
	}

	for _, shop := range r.world.Shops {
		known.Names = append(known.Names, shop.Keeper)
	}

	for _, person := range r.world.People {
		known.Names = append(known.Names, person.Name)
	}

	return known
}
