package game

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sebastianrozmus/neuro-processor/internal/speech"
)

// A companion or a shopkeeper may make up a brother, a scar or a village
// they come from: the card says little of their life and a player asks.
// What they say about themselves is noted by the Mind and kept, so that
// the brother named today is the same brother tomorrow and after a save.
// The people of a case never make anything up, and the caravan's merchant
// is nobody in particular, so neither has a bio.

const (
	// maxBio is how much a companion remembers saying of themselves. The
	// oldest lines are the canon the player has heard most, so past the
	// cap new ones are dropped rather than the old ones.
	maxBio = 12
	// maxBioNoted is how many facts one line may add.
	maxBioNoted = 3
	// maxBioText keeps a fact a sentence, not a story.
	maxBioText = 200
)

// bioFact is one thing a companion said about themselves, the day they
// said it and who heard it.
type bioFact struct {
	text  string
	day   int
	heard map[string]bool
}

// SelfNoteRequest asks a model what a companion or a shopkeeper has just
// said about themselves.
type SelfNoteRequest struct {
	// Speaker is their name, Card who they are.
	Speaker string
	Card    string
	// Known is what they have said of themselves before.
	Known []string
	// Said is the player's line, Reply the speaker's answer to it.
	Said  string
	Reply string
}

// SelfNoter is a model able to read what a speaker said of themselves.
type SelfNoter interface {
	SelfNote(ctx context.Context, request SelfNoteRequest) ([]string, error)
}

// Folded beginnings of the words a speaker talks about their own life
// with: family, the body, where from, the old days. personalWords count
// only whole ("wies", not "wiesz"); personalPhrases as phrases.
var (
	personalStems = []string{
		"brat", "siostr", "matk", "matce", "mama", "mame", "ojc", "ojciec", "tata", "taty", "dziad", "babk", "babci",
		"zona", "zone", "zony", "zonie", "zonaty", "zamezn", "meza", "mezem", "mezu", "wdow", "sierot",
		"dzieck", "dzieci", "syn", "corka", "corke", "corki", "corek", "coreczk", "wnuk", "wnucz", "rodzin", "rodzic", "rodzenstw",
		"blizn", "urodz", "pochodz", "wiosk", "dawniej", "sluzyl", "wychowal", "wychowywal", "mlodosc", "terminowal",
	}
	personalWords   = []string{"maz", "wies", "wsi", "kiedys", "mlodu"}
	personalPhrases = []string{"lat temu", "w mlodosci", "za mlodu", "uczyl mnie", "nauczyl mnie", "mnie uczyl", "mnie nauczyl", "jako chlopak", "jako dziewczyna", "jako dziewczynka", "moj dom", "moja wies"}
)

// Personal reports whether a speaker's line talks about their own life,
// worth asking the Mind about. The speaker's own name is no cue: "Brat
// Jaromir" says nothing of a brother, "mój brat" does.
func Personal(line, speaker string) bool {
	name := strings.Fields(Fold(speaker))

	var words []string
	for _, word := range strings.Fields(line) {
		folded := strings.Fields(Fold(word))
		if len(folded) == 0 {
			continue
		}

		if first, _ := utf8.DecodeRuneInString(strings.TrimLeft(word, "„«\"(")); unicode.IsUpper(first) && slices.Contains(name, folded[0]) {
			continue
		}

		words = append(words, folded...)
	}

	for _, word := range words {
		if slices.Contains(personalWords, word) || slices.ContainsFunc(personalStems, func(stem string) bool { return strings.HasPrefix(word, stem) }) {
			return true
		}
	}

	padded := " " + strings.Join(words, " ") + " "

	return slices.ContainsFunc(personalPhrases, func(phrase string) bool { return strings.Contains(padded, " "+phrase) })
}

// bioPlainWords are words too common to ground a fact in a line.
var bioPlainWords = []string{
	"jest", "byla", "bylo", "byli", "byly", "bedzie", "mial", "miala", "mialo", "mieli", "miec", "masz", "mowi", "mowil", "mowila",
	"jego", "swoj", "swoja", "swoje", "swojego", "swojej", "swoim", "swoich", "ktory", "ktora", "ktore", "ktorego", "ktorej",
	"przez", "jako", "oraz", "bardzo", "tylko", "jeszcze", "tego", "temu", "sobie", "siebie", "teraz", "dawno", "nigdy", "zawsze",
	"gdzie", "kiedy", "kiedys", "jednak", "nadal", "wciaz", "wtedy", "potem", "zanim", "niego", "niej", "nich", "ktos", "cos",
}

// bioContent lists the words of a folded text that carry meaning: long
// enough, not plain and not the speaker's name.
func bioContent(text string, skip []string) []string {
	var words []string

	for _, word := range strings.Fields(Fold(text)) {
		if (len(word) > minQueryWord || isDigits(word)) && !slices.Contains(bioPlainWords, word) && !slices.Contains(skip, word) {
			words = append(words, word)
		}
	}

	return words
}

func isDigits(word string) bool {
	return word != "" && strings.Trim(word, "0123456789") == ""
}

// shares reports whether any meaningful word of a text begins, by its
// stem, some word of another: "brata Radka" and "mam brata, Radka".
func shares(text, other string, skip []string) bool {
	words := strings.Fields(Fold(other))

	for _, part := range bioContent(text, skip) {
		stem := Stem(fleeting(part))
		if isDigits(part) {
			stem = part
		}

		for _, word := range words {
			if strings.HasPrefix(word, stem) || strings.HasPrefix(fleeting(word), stem) {
				return true
			}
		}
	}

	return false
}

// BioGrounded reports whether a fact the Mind read from a speaker's line
// is in it at all: a word of it, other than the speaker's name, is in the
// line. A Mind asked about a plain line otherwise gives the speaker a
// sister they never mentioned.
func BioGrounded(fact, line, speaker string) bool {
	return shares(fact, line, strings.Fields(Fold(speaker)))
}

// bioKey is a fact as compared with others: folded, words only.
func bioKey(text string) string {
	return strings.Join(strings.Fields(Fold(text)), " ")
}

// bioSentence tidies a fact into one sentence.
func bioSentence(text string) string {
	text = strings.Trim(strings.TrimSpace(strings.Join(strings.Fields(text), " ")), "\"„”«»")
	if text == "" {
		return ""
	}

	if !strings.ContainsAny(text[len(text)-1:], ".!?") {
		text += "."
	}

	return capitalized(text)
}

// bioSpeakers lists who may have a bio, in a fixed order: the companions,
// then the shopkeepers whose name is nobody else's, so that a keeper and a
// companion of one name never share a past.
func (r *Realm) bioSpeakers() []string {
	speakers := make([]string, 0, len(r.world.Companions)+len(r.world.Shops))
	for _, spec := range r.world.Companions {
		speakers = append(speakers, spec.Name)
	}

	for _, shop := range r.world.Shops {
		alike := 0
		for _, other := range r.world.Shops {
			if other.Keeper == shop.Keeper {
				alike++
			}
		}

		if alike == 1 && !slices.Contains(speakers, shop.Keeper) {
			speakers = append(speakers, shop.Keeper)
		}
	}

	return speakers
}

// bioName is the name the keeper of a talk keeps a bio under: a companion
// or a shopkeeper, never the caravan's merchant or the people of a case.
func (r *Realm) bioName(index int) (string, bool) {
	if r.investigation() || index < 0 {
		return "", false
	}

	if kind, ok := r.mateOf(index); ok {
		return r.world.Companions[kind].Name, true
	}

	if index >= len(r.world.Shops) {
		return "", false
	}

	// A keeper's place on the list is past the companions, or they have
	// none.
	name := r.world.Shops[index].Keeper

	return name, slices.Index(r.bioSpeakers(), name) >= len(r.world.Companions)
}

// bioCard is who the keeper of a talk is, as the Mind is shown it.
func (r *Realm) bioCard(index int) string {
	if kind, ok := r.mateOf(index); ok {
		spec := r.world.Companions[kind]

		return fmt.Sprintf("%s, %s. %s", spec.Name, roleName(spec.Role), spec.Description)
	}

	shop := r.world.Shops[index]

	return fmt.Sprintf("%s, %s. %s", shop.Keeper, shop.Name, shop.Persona)
}

// bioTexts are what a speaker has said of themselves, oldest first.
func (r *Realm) bioTexts(speaker string) []string {
	texts := make([]string, 0, len(r.bios[speaker]))
	for _, fact := range r.bios[speaker] {
		texts = append(texts, fact.text)
	}

	return texts
}

// remember keeps a fact a speaker said of themselves, heard by a
// player. It reports whether the fact is new; one said before is only
// heard again.
func (r *Realm) remember(speaker, text string, day int, heard ...string) bool {
	key := bioKey(text)
	if key == "" {
		return false
	}

	for _, fact := range r.bios[speaker] {
		if bioKey(fact.text) == key {
			for _, name := range heard {
				fact.heard[name] = true
			}

			return false
		}
	}

	if len(r.bios[speaker]) >= maxBio {
		return false
	}

	fact := &bioFact{text: text, day: day, heard: map[string]bool{}}
	for _, name := range heard {
		fact.heard[name] = true
	}

	r.bios[speaker] = append(r.bios[speaker], fact)

	return true
}

// SelfNoteRequest is the request to read what the companion or the
// shopkeeper a player talks with has just said about themselves, when
// their line or the player's question is about their own life. It
// reports false for any other talk.
func (r *Realm) SelfNoteRequest(name, said, reply string) (SelfNoteRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer := r.active(name)
	if customer == nil {
		return SelfNoteRequest{}, false
	}

	speaker, ok := r.bioName(customer.shop)
	if !ok {
		return SelfNoteRequest{}, false
	}

	_, line := speech.SplitMood(reply)

	if strings.TrimSpace(line) == "" || len(r.bios[speaker]) >= maxBio || !Personal(line, speaker) && !Private(said) {
		return SelfNoteRequest{}, false
	}

	return SelfNoteRequest{
		Speaker: speaker, Card: r.bioCard(customer.shop),
		Known: r.bioTexts(speaker), Said: said, Reply: line,
	}, true
}

// LearnSelf keeps what a speaker said about themselves in a line, as
// the Mind read it: only facts grounded in the line, not about the
// player, money or the hire, and not said before. It returns the facts
// kept, which the player is told.
func (r *Realm) LearnSelf(name, speaker string, facts []string, line string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer := r.active(name)
	if customer == nil || !slices.Contains(r.bioSpeakers(), speaker) {
		return nil
	}

	_, line = speech.SplitMood(line)
	hero := strings.Fields(Fold(customer.name))
	money := []string{Fold(r.world.Currency), "zold", "najem", "najm", "umow", "zaplat", "cena", "cene", "ceny"}

	var kept []string

	for _, fact := range facts[:min(len(facts), maxBioNoted)] {
		text := bioSentence(fact)
		folded := strings.Fields(Fold(text))

		switch {
		case text == "" || len(text) > maxBioText || !BioGrounded(text, line, speaker):
			continue
		case slices.ContainsFunc(folded, func(word string) bool { return slices.Contains(hero, word) }):
			continue
		case slices.ContainsFunc(folded, func(word string) bool {
			return slices.ContainsFunc(money, func(stem string) bool { return stem != "" && strings.HasPrefix(word, stem) })
		}):
			continue
		}

		if r.remember(speaker, text, r.day, customer.name) {
			kept = append(kept, text)
			r.tell(customer, "Zapamiętujesz: "+speaker+" o sobie: "+text)
		}
	}

	return kept
}

// bioHint answers a companion or a shopkeeper asked about their own life:
// keep to what they said before, or, when they said nothing of it, make
// up one thing that fits their card and keep to that.
func (r *Realm) bioHint(speaker, text string) string {
	she := speech.Feminine(speaker)
	said := gendered(she, "mówiłeś", "mówiłaś")
	skip := strings.Fields(Fold(speaker))

	var touched []string

	for _, fact := range r.bios[speaker] {
		if shares(text, fact.text, skip) {
			touched = append(touched, fact.text)
		}
	}

	if len(touched) > 0 {
		return fmt.Sprintf("Gracz pyta o twoje prywatne sprawy. Już o tym %s: %s Trzymaj się tego i nie zaprzeczaj sobie; możesz dodać jeden drobny szczegół, który do tego pasuje. Mów krótko, po swojemu.",
			said, strings.Join(touched, " "))
	}

	hint := "Gracz pyta o twoje prywatne sprawy: rodzinę, bliskich, skąd jesteś, skąd blizna. Jeśli twoja karta o tym milczy, możesz wymyślić jedną konkretną rzecz, " +
		"która do niej pasuje (imię, miejsce albo zdarzenie), i od teraz to prawda o tobie. Tylko jedną, bez długiej historii; możesz też zbyć pytanie, jeśli tak każe charakter."
	if len(r.bios[speaker]) > 0 {
		hint += " Nie zaprzeczaj temu, co już o sobie " + said + "."
	}

	return hint
}

// bioJournal is what a player has heard companions and shopkeepers say
// of themselves, for the journal.
func (r *Realm) bioJournal(name string) []FactView {
	var journal []FactView

	for _, speaker := range r.bioSpeakers() {
		for _, fact := range r.bios[speaker] {
			if fact.heard[name] {
				journal = append(journal, FactView{Subject: speaker + " o sobie", Text: fact.text, Day: fact.day})
			}
		}
	}

	return journal
}

// sortedNames lists the names in a set in order, for a save.
func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}
