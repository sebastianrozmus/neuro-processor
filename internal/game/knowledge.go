package game

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The world's knowledge is a small graph of facts: what is true about
// places, monsters, people and things, and who knows it. Keepers talk
// only from what they know, send the player to whoever knows the rest,
// and pick up what others said with time. Facts a keeper says for the
// first time join the graph, so the world remembers what was made up in
// it, and events add facts of their own.

// names lists the names of the world: places, people, monsters, things,
// the subjects of what is known and the goods keepers talked into it.
func (r *Realm) names() []string {
	built := r.world
	names := []string{built.Name, built.Currency, built.Boss.Name}

	for _, place := range built.Places {
		names = append(names, place.Name)
	}

	for _, site := range built.Sites {
		names = append(names, site.Name)
	}

	for _, monster := range built.Monsters {
		names = append(names, monster.Name)
	}

	for _, item := range built.Items {
		names = append(names, item.Name)
	}

	for _, shop := range built.Shops {
		names = append(names, shop.Keeper, shop.Name)
	}

	for _, companion := range built.Companions {
		names = append(names, companion.Name)
	}

	for _, person := range built.People {
		names = append(names, person.Name)
	}

	for _, room := range built.Rooms {
		names = append(names, room.Name)
	}

	for _, thing := range built.Things {
		names = append(names, thing.Name)
	}

	for _, fact := range r.wisdom.facts {
		names = append(names, fact.Subject)
	}

	for _, added := range r.extras {
		names = append(names, added.item.Name)
	}

	for _, item := range r.world.Artifacts {
		names = append(names, item.Name)
	}

	return names
}

// Fact is one thing somebody may know.
type Fact struct {
	ID      int
	Subject string
	Text    string
	// Source names who brought the fact into the world: a keeper, an
	// event, or empty for what the world began with.
	Source string
	Day    int
	// known holds the keepers (by shop index, or by person in a town)
	// who know the fact; heard the players who have come across it.
	known map[int]bool
	heard map[string]bool
	// pointed holds the players who were sent to somebody about the fact
	// before hearing it, and whom they were sent to: a lead in their
	// journal until the fact itself is heard.
	pointed map[string]string
	// A clue of an investigation may be held by a thing rather than
	// known by anyone, and may wait for a flag, for another fact to have
	// been heard, or for a time of day.
	thing      string
	needs      string
	heardFirst string
	hour       string
}

// knowers lists the keepers who know the fact in a fixed order: walking
// the map sent the player to a different keeper on every run.
func (fact *Fact) knowers() []int {
	keepers := make([]int, 0, len(fact.known))
	for keeper := range fact.known {
		keepers = append(keepers, keeper)
	}

	slices.Sort(keepers)

	return keepers
}

// FactSpec is a fact as a model or an event states it. Quote, from a
// model, is the words of the line the fact was read from: a fact without
// its source in the line is not taken.
type FactSpec struct {
	Subject string `json:"subject"`
	Text    string `json:"fact"`
	Quote   string `json:"quote,omitempty"`
}

// FactView is a fact as a player's journal shows it.
type FactView struct {
	Subject string
	Text    string
	Source  string
	Day     int
	// Lead names who to ask about a fact the player has only been
	// pointed at; such a view has no Text.
	Lead string
}

// knowledge is the graph of a realm.
type knowledge struct {
	facts []*Fact
}

const (
	// spreadPercent is the chance per day that a keeper picks up a fact a
	// neighbour knows.
	spreadPercent = 45
	maxFacts      = 200
	// referrals is how many things a keeper does not know but could point
	// the player to.
	referrals = 2
	// crumbWords is how much of a fact a keeper who lacks it has heard.
	crumbWords = 12
	// leadWord is how long a player's word must be to point at a fact
	// the keeper lacks: its stem is then four letters, not three.
	leadWord = 5
)

// questionWords are what a question is asked with, never what it is
// about.
var questionWords = []string{
	"jaki", "jaka", "jakie", "jakiego", "jakich", "ktory", "ktora", "ktore", "skad", "dokad", "gdzie", "kiedy", "czemu", "dlaczego",
	"ile", "czego", "komu", "czym", "kogo", "kto", "coś", "cos", "powiedz", "opowiedz", "wiesz", "slyszales", "slyszalas", "znasz",
	"masz", "macie", "maja", "jest", "tutaj", "karczmie", "karczma", "dobre", "dobry", "dobra", "moze", "chyba", "wlasciwie",
}

// questionWord reports a folded word that is part of asking, not of the
// subject asked about.
func questionWord(word string) bool {
	word = strings.Trim(word, ".,!?;:")
	for _, candidate := range questionWords {
		if word == candidate {
			return true
		}
	}

	return false
}

// add puts a fact into the graph, known by the given keepers.
func (r *Realm) add(subject, text, source string, keepers ...int) *Fact {
	if len(r.wisdom.facts) >= maxFacts {
		return nil
	}

	fact := &Fact{ID: len(r.wisdom.facts) + 1, Subject: strings.TrimSpace(subject), Text: strings.TrimSpace(text), Source: source, Day: r.day, known: map[int]bool{}, heard: map[string]bool{}, pointed: map[string]string{}}
	for _, keeper := range keepers {
		fact.known[keeper] = true
	}

	r.wisdom.facts = append(r.wisdom.facts, fact)

	return fact
}

// everyone lists every keeper: the shopkeepers and the caravan, or the
// people of a town.
func (r *Realm) everyone() []int {
	if r.investigation() {
		all := make([]int, 0, len(r.world.People))
		for index := range r.world.People {
			all = append(all, index)
		}

		return all
	}

	all := make([]int, 0, len(r.world.Shops)+1)
	for index := range r.world.Shops {
		all = append(all, index)
	}

	if r.caravan != nil {
		all = append(all, r.caravanShop())
	}

	return all
}

// some picks a few keepers at random, at least one.
func (r *Realm) some(count int) []int {
	all := r.everyone()
	if len(all) == 0 {
		return nil
	}

	r.rng.Shuffle(len(all), func(a, b int) { all[a], all[b] = all[b], all[a] })

	return all[:min(max(count, 1), len(all))]
}

// keeperName names a keeper, or returns "" for an index that is none.
func (r *Realm) keeperName(index int) string {
	switch {
	case index < 0:
		return ""
	case r.investigation():
		if index < len(r.world.People) {
			return r.world.People[index].Name
		}
	case r.caravan != nil && index == r.caravanShop():
		return caravanKeeper
	case index < len(r.world.Shops):
		return r.world.Shops[index].Keeper
	}

	if kind, ok := r.mateOf(index); ok {
		return r.world.Companions[kind].Name
	}

	return ""
}

// shopOfKind returns the keeper of a kind of shop, or -1.
func (r *Realm) shopOfKind(kind world.ShopKind) int {
	for index, shop := range r.world.Shops {
		if shop.Kind == kind {
			return index
		}
	}

	return -1
}

// seedKnowledge gives the world its starting facts and hands them out:
// everybody knows the threat, specialists know their monsters, places
// and secrets are known to a few, and companions to the keepers of the
// settlement they wait in.
func (r *Realm) seedKnowledge() {
	built := r.world
	if r.investigation() {
		r.seedClues()

		return
	}

	if built.Mode.Rooms() || len(built.Shops) == 0 {
		return
	}

	all := r.everyone()
	r.add(built.Boss.Name, built.Boss.Description+" Siedziba: "+bearing(built.Start(), built.Lair)+".", "", all...)

	for _, place := range built.Places {
		if place.Danger > 0 {
			r.add(place.Name, place.Description+" Niebezpieczeństwo: "+[]string{"małe", "spore", "śmiertelne"}[min(place.Danger, world.MaxTier)-1]+".", "", r.some(2)...)
		}
	}

	for _, monster := range built.Monsters {
		expert := r.shopOfKind(world.ShopArmory)

		switch monster.Shape {
		case "serpent", "insect", "blob":
			expert = r.shopOfKind(world.ShopAlchemist)
		case "spirit", "undead":
			expert = r.shopOfKind(world.ShopMagic)
		}

		text := fmt.Sprintf("%s Cecha: %s, %s", monster.Description, traits[monster.Shape].Name, strings.ToLower(traits[monster.Shape].Text))
		r.add(monster.Name, text, "", append(r.some(1), expert)...)
	}

	for index, tile := range built.Tiles {
		if tile.Site < 0 {
			continue
		}

		site := built.Sites[tile.Site]
		kind := "miejsce, które przywraca siły"
		if site.Kind == world.SiteTreasure {
			kind = "strzeżona skrytka z cennym przedmiotem"
		}

		point := world.Point{X: index % world.Width, Y: index / world.Width}
		r.add(site.Name, fmt.Sprintf("%s (%s), %s od osady startowej. %s", site.Name, kind, bearing(built.Start(), point), site.Description), "", r.some(1)...)
	}

	for index, companion := range built.Companions {
		r.add(companion.Name, fmt.Sprintf("%s Czeka na najem w karczmie w osadzie %s.", companion.Description, r.placeAt(built.Towns[r.innOf(index)]).Name), "", all...)
	}
}

// knownBy lists the facts a keeper knows, newest last.
func (r *Realm) knownBy(keeper int) []*Fact {
	var known []*Fact
	for _, fact := range r.wisdom.facts {
		if fact.known[keeper] {
			known = append(known, fact)
		}
	}

	return known
}

// unknownTo lists facts a keeper lacks that some other keeper has, with
// who to ask.
func (r *Realm) unknownTo(keeper int) []string {
	var gaps []string

	for _, fact := range r.wisdom.facts {
		if fact.known[keeper] {
			continue
		}

		for _, other := range fact.knowers() {
			if other != keeper && r.keeperName(other) != "" {
				gaps = append(gaps, fmt.Sprintf("%s (wie o tym %s)", fact.Subject, r.keeperName(other)))

				break
			}
		}
	}

	return gaps
}

// spreadKnowledge lets facts travel between keepers, one day at a time.
func (r *Realm) spreadKnowledge() {
	for _, fact := range r.wisdom.facts {
		if len(fact.known) == 0 {
			continue
		}

		for _, keeper := range r.everyone() {
			if !fact.known[keeper] && r.rng.Intn(100) < spreadPercent {
				fact.known[keeper] = true
			}
		}
	}
}

// learn marks that a player has come across a fact.
func (r *Realm) learn(hero *player, fact *Fact) {
	if fact != nil {
		fact.heard[hero.name] = true
	}
}

// gapFor finds a fact the keeper lacks, another keeper has, and the
// player's words ask about: the subject matched by a word of theirs. It
// returns the fact and who knows it, or nil.
func (r *Realm) gapFor(hero *player, keeper int, words []string) (*Fact, int) {
	for _, word := range words {
		// A short word's stem is a prefix of too much ("skad" of
		// "Skalny"), and a question word is never a subject.
		if len(word) < leadWord || questionWord(word) || slices.Contains(plainWords, word) {
			continue
		}

		for _, fact := range r.wisdom.facts {
			// Nobody is sent to ask about themselves: "Borin, dokąd idziemy?"
			// is no question about Borin Kamienna Pięść.
			if fact.known[keeper] || !r.open(fact, hero) || !leads(r.calledBy(fact.Subject), word) || !r.person(fact.Subject) && !firmLead(r.calledBy(fact.Subject), word, words) || fact.Subject == r.keeperName(keeper) {
				continue
			}

			for _, other := range fact.knowers() {
				if other != keeper && r.keeperName(other) != "" {
					return fact, other
				}
			}
		}
	}

	return nil, -1
}

// leadPrefix is how many first letters a player's word shares with a word
// of a subject for a lead: "przełęczy" is "Przełęcz", "przepraszam" is not.
const leadPrefix = 5

// leads reports whether a folded word of the player's names a word of the
// subject: the two share leadPrefix letters, or all but the ending of the
// shorter one ("widma" is "Widmo"), and never fewer than four: "dzień" is
// no "Dzik". A word of three letters leads nowhere: "Borin" is no "Bór".
func leads(subject, word string) bool {
	for _, part := range strings.Fields(Fold(subject)) {
		if len(part) <= minQueryWord {
			continue
		}

		shared := 0
		for shared < len(part) && shared < len(word) && part[shared] == word[shared] {
			shared++
		}

		// "chodź" leads to no "Chodniki", nor "strzygę" to the "Strzygoń".
		if shared >= max(minQueryWord+1, min(leadPrefix, len(part)-1, len(word)-1)) && formOf(word, part) {
			return true
		}
	}

	return false
}

// crumb is the little a keeper who does not know a fact has heard of it:
// its first sentence, cut short.
func crumb(fact *Fact) string {
	text := fact.Text
	for index, char := range text {
		if strings.ContainsRune(".!?", char) && (index+len(string(char)) == len(text) || strings.HasPrefix(text[index+len(string(char)):], " ")) {
			text = text[:index]

			break
		}
	}

	words := strings.Fields(text)
	if len(words) > crumbWords {
		words = append(words[:crumbWords], "…")
	}

	return strings.Join(words, " ")
}

// whereIs says where a keeper can be found, for a lead.
func (r *Realm) whereIs(keeper int) string {
	switch {
	case r.investigation():
		return ""
	case r.caravan != nil && keeper == r.caravanShop():
		return "karawana na trakcie, dziś w miejscu: " + r.placeAt(r.caravan.pos).Name
	case keeper < len(r.world.Shops):
		// Every settlement has this shop; "in every settlement" in the hint
		// was repeated word for word by the model.
		return r.world.Shops[keeper].Name
	}

	if kind, ok := r.mateOf(keeper); ok {
		return "karczma w osadzie " + r.placeAt(r.world.Towns[r.innOf(kind)]).Name
	}

	return ""
}

// point records that a player was sent to a keeper about a fact. It
// returns the journal line of the lead, which the player is told.
func (r *Realm) point(hero *player, fact *Fact, keeper int) string {
	who := r.keeperName(keeper)
	fact.pointed[hero.name] = who

	line := "Trop: " + fact.Subject + " → " + who + "."
	r.tell(hero, line)

	return line
}

// Learn records a fact a keeper just stated for the first time: it joins
// the graph, known by that keeper and heard by the player. The line it
// was noted from is kept as its origin.
func (r *Realm) Learn(name string, spec FactSpec, line string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer := r.active(name)
	if customer == nil || customer.shop < 0 || strings.TrimSpace(spec.Text) == "" || strings.TrimSpace(spec.Subject) == "" {
		return false
	}

	// A fact must be grounded in the line: the quote the mind gives must
	// be in it, and so must the subject. A mind asked about a garbled
	// line otherwise "notes" whatever it imagines ("Karczma i Woda").
	if !Grounded(spec, line) {
		return false
	}

	// And it must be news: not a denial, a price, a ware or a known
	// subject in other words.
	if Stale(spec, line, r.news(name)) != "" {
		return false
	}

	for _, fact := range r.wisdom.facts {
		if Matches(fact.Subject, spec.Subject) && Matches(spec.Subject, fact.Subject) {
			// Something more about a known subject still counts, once.
			if strings.EqualFold(fact.Text, spec.Text) {
				return false
			}
		}
	}

	fact := r.add(spec.Subject, spec.Text, r.keeperName(customer.shop), customer.shop)
	if fact == nil {
		return false
	}

	r.learn(customer, fact)
	r.trace(DescribeFact, fact.ID, Trace{How: HowNoted + " (" + r.keeperName(customer.shop) + ")", Prompt: line})
	r.tell(customer, "Zapamiętujesz: "+fact.Subject+".")

	return true
}

// Grounded reports whether a fact the mind read from a line is really in
// it: the quoted words appear in the line (allowing for case and
// diacritics) and the subject's words do too, by stem.
func Grounded(spec FactSpec, line string) bool {
	folded := Fold(line)
	quote := strings.TrimSpace(Fold(spec.Quote))
	if len(strings.Fields(quote)) < 2 || !strings.Contains(folded, quote) {
		return false
	}

	words := strings.Fields(folded)
	for _, part := range strings.Fields(Fold(spec.Subject)) {
		if len(part) < minQueryWord {
			continue
		}

		if !nameOK(words, part) {
			return false
		}
	}

	return true
}

// Journal returns what a player has learnt, oldest first.
func (r *Realm) Journal(name string) []FactView {
	r.mu.Lock()
	defer r.mu.Unlock()

	var journal []FactView
	for _, fact := range r.wisdom.facts {
		switch {
		case fact.heard[name]:
			journal = append(journal, FactView{Subject: fact.Subject, Text: fact.Text, Source: fact.Source, Day: fact.Day})
		case fact.pointed[name] != "":
			journal = append(journal, FactView{Subject: fact.Subject, Lead: fact.pointed[name], Day: fact.Day})
		}
	}

	// What companions said of themselves comes after the world's news.
	return append(journal, r.bioJournal(name)...)
}

// JournalLines words a journal for the log.
func JournalLines(journal []FactView) []string {
	if len(journal) == 0 {
		return []string{"Dziennik jest pusty. Rozmawiaj z ludźmi, a zapiszesz, czego się dowiesz."}
	}

	lines := []string{fmt.Sprintf("Dziennik (%d):", len(journal))}
	for _, fact := range journal {
		if fact.Lead != "" {
			lines = append(lines, fmt.Sprintf("• Trop: %s — pytaj: %s", fact.Subject, fact.Lead))

			continue
		}

		source := ""
		if fact.Source != "" {
			source = " (od: " + fact.Source + ")"
		}

		lines = append(lines, fmt.Sprintf("• %s: %s%s", fact.Subject, fact.Text, source))
	}

	return lines
}

// NoteRequest asks a model whether a keeper's line brought a new fact
// into the world.
type NoteRequest struct {
	Reply string
	// Known are the subjects already in the graph, so that old news is
	// not noted twice.
	Known []string
}

// Noter is a model able to spot new facts in a line.
type Noter interface {
	Note(ctx context.Context, request NoteRequest) (FactSpec, bool, error)
}

// Subjects lists what the graph already knows about, for a Noter.
func (r *Realm) Subjects() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	seen := map[string]bool{}
	var subjects []string

	for _, fact := range r.wisdom.facts {
		if !seen[fact.Subject] {
			seen[fact.Subject] = true
			subjects = append(subjects, fact.Subject)
		}
	}

	sort.Strings(subjects)

	return subjects
}

// told is a fact as a keeper tells it now: a companion someone has hired
// no longer waits at the inn, so Borin "czeka na najem" while he walks
// with the hero no more.
func (r *Realm) told(fact *Fact) string {
	for kind, companion := range r.world.Companions {
		if companion.Name != fact.Subject || r.hired[kind] == "" {
			continue
		}

		if text, _, found := strings.Cut(fact.Text, " Czeka na najem"); found {
			return fmt.Sprintf("%s Jest teraz w drużynie bohatera %s.", text, capitalized(r.hired[kind]))
		}
	}

	return fact.Text
}

// firmLead reports whether a word that leads to a subject is not just its
// adjective before another noun: "popielnego upiora" leads to no Popielne
// Zbocza, "popielne zbocza" do.
func firmLead(subject, word string, words []string) bool {
	index := slices.Index(words, word)

	return index < 0 || !lone(subject, words, index)
}
