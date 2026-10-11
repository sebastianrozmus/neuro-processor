package game

import (
	"context"
	"fmt"
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

// FactSpec is a fact as a model or an event states it.
type FactSpec struct {
	Subject string `json:"subject"`
	Text    string `json:"fact"`
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

		for other := range fact.known {
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
		if len(word) < leadWord || questionWord(word) {
			continue
		}

		for _, fact := range r.wisdom.facts {
			if fact.known[keeper] || !r.open(fact, hero) || !Matches(fact.Subject, word) {
				continue
			}

			for other := range fact.known {
				if other != keeper && r.keeperName(other) != "" {
					return fact, other
				}
			}
		}
	}

	return nil, -1
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
		return r.world.Shops[keeper].Name + " w każdej osadzie"
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

	return journal
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
