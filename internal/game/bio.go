package game

import (
	"context"
	"fmt"
	"maps"
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
// Jaromir" says nothing of a brother, "mój brat" does; nor is a word of
// kin the speaker calls the player by: "bierz, synu" has no son in it.
func Personal(line, speaker string) bool {
	name := strings.Fields(Fold(speaker))

	var words []string
	for _, word := range strings.Fields(dropVocatives(line)) {
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

// Words of kin that call the one spoken to, folded: "synu", "córko",
// "dziecino". Said by a keeper to the player, or by the player to a
// keeper, they say nothing of anybody's family. vocatives are such words
// unless a preposition or a possessive stands before them ("o synu",
// "po moim synu" tell of a son); commaVocatives only when set off by a
// comma, as "dziecko" and "bracie" are also a child and a brother: "masz
// dziecko?", "o bracie".
var (
	vocatives = []string{
		"synu", "synku", "synus", "synusiu", "corko", "coreczko", "coruchno", "corus", "dziecino", "dziecie", "dzieciatko", "dzieciaku",
		"siostro", "siostrzyczko", "braciszku", "bratku", "ojcze", "matko", "mamo", "mamusiu", "tato", "tatku", "tatusiu",
		"babciu", "babuniu", "babko", "dziadku", "dziadziu", "wnusiu", "wnuczku", "wnuczko",
	}
	commaVocatives = []string{"dziecko", "bracie"}
	vocativeBefore = []string{"o", "po", "przy", "w", "na", "moim", "twoim", "swoim", "naszym", "waszym", "jej", "jego", "mym"}
)

// dropVocatives leaves out of a line the words of kin it calls somebody
// by, keeping the rest as it was, their punctuation too: "masz wnuki,
// babko?" is still a question.
func dropVocatives(line string) string {
	tokens := strings.Fields(line)
	kept := make([]string, 0, len(tokens))

	for index, token := range tokens {
		folded := strings.TrimSpace(Fold(token))
		before := ""
		if index > 0 {
			before = strings.TrimSpace(Fold(tokens[index-1]))
		}

		set := index > 0 && strings.HasSuffix(tokens[index-1], ",") || index == 0 && strings.ContainsAny(token[len(token)-1:], ",!")
		if slices.Contains(vocatives, folded) && !slices.Contains(vocativeBefore, before) || slices.Contains(commaVocatives, folded) && set {
			token = strings.TrimLeftFunc(token, func(char rune) bool { return !strings.ContainsRune(".,!?;:", char) })
			if token == "" {
				continue
			}
		}

		kept = append(kept, token)
	}

	return strings.Join(kept, " ")
}

// kinFamilies gather the folded words of one side of a speaker's life,
// so that a question in one word touches a fact told in another: "masz
// rodzeństwo?" the brother, "ma pani dzieci?" the son, "mąż żyje?" the
// widow, "skąd jesteś?" the village born in. A word ending in * is a
// beginning, any other a whole word. "rodzin*" (a family) touches every
// family of kin, and "rodzinn*" (a home village) where one comes from.
var kinFamilies = [][]string{
	{"rodzenstw*", "brat", "brata", "bratem", "bratu", "bracia", "braci", "bracmi", "braciszk*", "siostr*"},
	{"dzieci", "dziecmi", "dzieciom", "dzieciach", "dzieck*", "potomk*", "potomstw*", "syn", "syna", "synem", "synu", "synowi", "synowie",
		"synow", "synami", "synek", "synka", "synkiem", "synkowi", "corka", "corki", "corke", "corce", "corek", "corkami", "corkom", "coreczk*"},
	// "maza" is how a line without diacritics writes "męża"; a bachelor
	// answers "masz żonę?" ("kawaler z wyboru")
	{"maz", "maza", "meza", "mezem", "mezu", "mezowi", "zona", "zony", "zone", "zonie", "zonka", "zonke", "zonaty", "zonata", "wdow*", "malzon*", "malzen*", "zamezn*", "kawaler*"},
	{"rodzic*", "ojciec", "ojca", "ojcu", "ojcem", "ojcze", "tata", "taty", "tate", "tacie", "tato", "tatus*", "matk*", "matce", "mama", "mamy",
		"mame", "mamie", "mamo", "mamus*"},
	// grandchildren and a sweetheart: "masz wnuki?", "masz chłopaka?" – "Nie."
	{"wnuk*", "wnucz*"},
	{"narzeczon*", "chlopak", "chlopaka", "chlopakiem", "chlopakowi", "dziewczyna", "dziewczyny", "dziewczyne", "dziewczynie"},
	{"skad", "pochodz*", "urodz*", "rodem", "rodzinn*", "wychow*", "dorast*"},
	// who taught them: "kto cię uczył?" and "uczył go mistrz Ezechiel"
	{"uczyl*", "nauczyl*", "nauczyciel*", "mistrz", "mistrza", "mistrzem", "mistrzowi", "mistrzyni*", "terminowal*", "praktykowal*"},
}

// kinOnly is how many of kinFamilies are kin; the rest are where one
// comes from and who taught them, which a family in general does not touch.
// Only the first kinChecked are checked for contradictions: "dziewczyna"
// and "chłopak" are as often anybody's lass and lad.
const (
	kinOnly    = 6
	kinChecked = 4
)

// kinFamily is the family of a folded word: an index of kinFamilies, -2
// for a word of the whole family, -1 for none.
func kinFamily(word string) int {
	if strings.HasPrefix(word, "rodzin") && !strings.HasPrefix(word, "rodzinn") {
		return -2
	}

	for index, family := range kinFamilies {
		if slices.ContainsFunc(family, func(entry string) bool {
			if stem, ok := strings.CutSuffix(entry, "*"); ok {
				return strings.HasPrefix(word, stem)
			}

			return word == entry
		}) {
			return index
		}
	}

	return -1
}

// kinShared reports whether a question and a fact speak of one side of a
// speaker's life, each in its own words. The speaker's name is skipped:
// "Brat Jaromir" has no brother in it.
func kinShared(text, other string, skip []string) bool {
	families := map[int]bool{}

	for _, word := range strings.Fields(Fold(text)) {
		if family := kinFamily(word); family != -1 && !slices.Contains(skip, word) {
			families[family] = true
		}
	}

	if len(families) == 0 {
		return false
	}

	for _, word := range strings.Fields(Fold(other)) {
		family := kinFamily(word)
		if family == -1 || slices.Contains(skip, word) {
			continue
		}

		// A family in general touches any kin, not where one comes from or
		// who taught them.
		isKin := func(f int) bool { return f >= 0 && f < kinOnly }
		kin := families[-2] || slices.ContainsFunc(slices.Collect(maps.Keys(families)), isKin)
		if families[family] || family == -2 && kin || isKin(family) && families[-2] {
			return true
		}
	}

	return false
}

// kinAsked reports whether a fact tells of the very kin a question asked
// about: "Bogna jest wdową" to "a mąż żyje?". A family in general asks
// about any kin ("masz rodzinę?"), but a fact of the family in general
// ("pochodzi z rodziny górników") answers no question about the husband.
func kinAsked(question, fact string) bool {
	families := map[int]bool{}
	for _, word := range strings.Fields(Fold(question)) {
		if family := kinFamily(word); family != -1 {
			families[family] = true
		}
	}

	for _, word := range strings.Fields(Fold(fact)) {
		if family := kinFamily(word); family >= 0 && (families[family] || families[-2] && family < kinOnly) {
			return true
		}
	}

	return false
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
		// A short word of kin means as much as a long one: "syn", "mąż".
		if (len(word) > minQueryWord || isDigits(word) || kinFamily(word) >= 0) && !slices.Contains(bioPlainWords, word) && !slices.Contains(skip, word) {
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

	// The same fact in fewer words is the same fact ("Szczepan ma brata
	// Józka, który rąbie drwa" after "… rąbie drwa pod Nadszybiem"); in
	// more words it is the same fact told better, and takes its place.
	words := strings.Fields(key)
	for _, fact := range r.bios[speaker] {
		known := strings.Fields(bioKey(fact.text))
		within := func(part, whole []string) bool {
			return len(part) > 0 && !slices.ContainsFunc(part, func(word string) bool { return !slices.Contains(whole, word) })
		}

		switch {
		case within(words, known):
			for _, name := range heard {
				fact.heard[name] = true
			}

			return false
		case within(known, words):
			fact.text = text
			for _, name := range heard {
				fact.heard[name] = true
			}

			return true
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

// privateTurns is how many lines a private question stays the topic
// for: the one right after it, and one more past a "przykro mi".
const privateTurns = 2

// privateAsked is the private question a player asked in the last line
// or two, counted as one line older now, or "" when none is that fresh.
func (r *Realm) privateAsked(customer *player) string {
	current := customer.talks[customer.shop]
	if current == nil {
		return ""
	}

	current.privateHinted = false
	if current.private == "" {
		return ""
	}

	current.privateAge++
	if current.privateAge > privateTurns {
		current.private = ""
	}

	return current.private
}

// keepPrivate remembers the private question a player has just asked.
func (r *Realm) keepPrivate(customer *player, text string) {
	if current := customer.talks[customer.shop]; current != nil {
		current.private, current.privateAge = text, 0
	}
}

// hintPrivate marks the line as briefed as a follow-up on the private
// question, so that the Mind reads the answer too, and keeps the topic
// for one more line: "i co robi?" after "a gdzie mieszka?" after "jak ma
// na imię?" is still about the sister. The line after that is not the
// line right after the question, so a ware asked for then is not read.
func (r *Realm) hintPrivate(customer *player) {
	if current := customer.talks[customer.shop]; current != nil {
		current.privateHinted = true
		current.privateAge = min(current.privateAge, 1)
	}
}

// privateWords are the words a player asks further about a keeper's own
// people and past with, folded, besides those of following up a lead and
// of kin: "jak miał na imię?", "a gdzie teraz?", "i co z nim?".
var privateWords = []string{
	"imie", "imienia", "imieniem", "nazywal", "nazywala", "nazywali", "wolali", "stalo", "zdarzylo", "umarl", "umarla", "umarli", "zmarl", "zmarla", "zginal", "zginela", "zgineli",
	"zyje", "zyja", "zyla", "zyli", "mieszka", "mieszkaja", "dlugo", "dawno", "temu", "jeszcze", "takze", "bedzie", "byl", "byla", "bylo", "byli", "jego", "jej", "oni", "one",
	"tylko", "naprawde", "dokladnie", "dlaczego", "czemu", "czego", "ktos", "nikt", "nikogo", "sam", "sama", "sami", "samotn", "wiek", "stary", "stara", "mlody", "mloda", "mlodszy", "mlodsza", "starszy", "starsza",
	"robi", "robia", "robil", "robila", "pracuje", "pracuja", "pomaga", "pomagaja", "pomagal", "pomagala", "uczy", "ucza", "chodzi", "chodza", "wrocil", "wrocila", "wroci", "wroca", "odszedl", "odeszla", "zostal", "zostala", "widziales", "widzialas", "widzieliscie",
	"pamietasz", "tesknisz", "kochales", "kochalas", "lubiles", "lubilas", "boli", "bolalo", "sie",
}

// privateFollowUp reports a short question that asks further about the
// private matter asked about a line or two ago rather than starts
// something else: every word of any length is a way of asking further, a
// word of kin or of a life, or from the keeper's own card, bio or last
// line. "ile za miód?" and "a po zmroku chodzi żubr?" are something else.
func (r *Realm) privateFollowUp(customer *player, text string, words []string, wares []Ware) bool {
	if len(words) == 0 || len(words) > 8 || !strings.Contains(text, "?") && !asked(text) {
		return false
	}

	folded := strings.Join(words, " ")
	if tradeAsked(folded) || len(waresAsked(words, wares)) > 0 || greeting(words) || thanked(words) || farewell(words) ||
		slices.ContainsFunc(words, func(word string) bool { return slices.Contains(mineWords, word) }) {
		return false
	}

	speaker, _ := r.bioName(customer.shop)
	own := strings.Fields(Fold(r.bioCard(customer.shop) + " " + strings.Join(r.bioTexts(speaker), " ") + " " + lastReply(customer)))

	for _, word := range words {
		if len(word) < minQueryWord+1 || slices.Contains(greetingWords, word) || slices.Contains(lifeVerbs, word) || slices.Contains(bodyWords, word) ||
			slices.Contains(personalWords, word) || slices.Contains(privateWords, word) || kinFamily(word) != -1 {
			continue
		}

		known := slices.ContainsFunc(followWords, func(stem string) bool { return strings.HasPrefix(word, stem) }) ||
			slices.ContainsFunc(personalStems, func(stem string) bool { return strings.HasPrefix(word, stem) }) ||
			slices.ContainsFunc(privateWords, func(stem string) bool { return strings.HasPrefix(word, stem) })
		for _, part := range own {
			known = known || len(part) >= minQueryWord+1 && (strings.HasPrefix(word, Stem(part)) || strings.HasPrefix(fleeting(word), Stem(fleeting(part))))
		}

		if !known {
			return false
		}
	}

	return true
}

// SelfNoteRequest is the request to read what the companion or the
// shopkeeper a player talks with has just said about themselves, when
// their line or the player's question is about their own life, or the
// line answers a private question asked a line or two ago. It reports
// false for any other talk.
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

	// Kin the card or the bio names count as kin said: "gdzie Rita?" to
	// one who has told of her daughter Rita.
	kin := cardKin(r.keeperCard(customer.shop)+" "+strings.Join(r.bioTexts(speaker), " "), speaker)
	named := slices.ContainsFunc(kin, func(name string) bool { return personNamed(line+" "+said, name) })

	// "Najstarszy chłopak, Mściwuj" to "któreś pomaga w kuźni?" right after
	// "ile masz dzieci?" tells of a son: the question before is the context.
	following := false
	if current := customer.talks[customer.shop]; current != nil && current.private != "" && !Private(said) && (current.privateAge <= 1 || current.privateHinted) {
		following, said = true, current.private+" "+said
	}

	if strings.TrimSpace(line) == "" || len(r.bios[speaker]) >= maxBio || !Personal(line, speaker) && !Private(said) && !named && !following {
		return SelfNoteRequest{}, false
	}

	if current := customer.talks[customer.shop]; current != nil {
		current.noted.said, current.noted.reply = said, line
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
	money := []string{"zold", "najem", "najm", "umow", "zaplat", "cena", "cene", "ceny"}

	// The question the line answered grounds a fact of the kin it asked
	// about: "Nie żyje, dawno" to "a mąż żyje?" is about the husband,
	// though the line has no word for him.
	said := ""
	if current := customer.talks[customer.shop]; current != nil && current.noted.reply == line {
		said = current.noted.said
	}

	var kept []string

	for _, fact := range facts[:min(len(facts), maxBioNoted)] {
		text := bioSentence(fact)
		folded := strings.Fields(Fold(text))

		switch {
		case text == "" || len(text) > maxBioText || !BioGrounded(text, line, speaker) && !kinAsked(said, text):
			continue
		case slices.ContainsFunc(folded, func(word string) bool { return slices.Contains(hero, word) }):
			continue
		case slices.ContainsFunc(folded, func(word string) bool {
			return slices.ContainsFunc(money, func(stem string) bool { return strings.HasPrefix(word, stem) })
		}), priced(folded, r.world.Currency):
			continue
		}

		if r.remember(speaker, text, r.day, customer.name) {
			kept = append(kept, text)
			r.tell(customer, "Zapamiętujesz: "+speaker+" o sobie: "+text)
		}
	}

	return kept
}

// priced reports a folded fact that names a sum of money: a number and the
// world's currency. The currency alone is no sum, as it is often a plain
// word: a lightning ("piorun") is no feather ("piór"), and the "Mgławica
// Zarodników" is a place, not a price in spores.
func priced(folded []string, currency string) bool {
	if !slices.ContainsFunc(folded, func(word string) bool { _, spelled := numberWords[word]; return isDigits(word) || spelled }) {
		return false
	}

	units := strings.Fields(Fold(currency))

	return slices.ContainsFunc(folded, func(word string) bool {
		return slices.ContainsFunc(units, func(unit string) bool { return len(unit) >= minQueryWord && formOf(word, unit) })
	})
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
		if shares(text, fact.text, skip) || kinShared(text, fact.text, skip) {
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
