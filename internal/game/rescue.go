package game

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// A shopkeeper who tells of someone dear gone missing, a daughter near
// the faceless diver or a brother lost on the marsh, gives the hero who
// heard it a job. The Mind reads the line; the game puts the missing one
// on a wild tile under guards that do not roam, the named monster among
// them when the world has it. The hero who clears the tile frees them and
// the keeper pays on their return, and the whole settlement hears of it.
//
// Only shopkeepers lose people this way: a companion walks with the hero,
// so there is nobody waiting at home to pay, and the people of a case and
// the caravan's merchant make nothing up.

const (
	// maxRescues is how many rescues may be open in the world at once.
	maxRescues = 3
	// rescueSteps is how far from the settlement a captive is best held;
	// rescueGuards how many guards a captive has at least.
	rescueSteps  = 4
	rescueGuards = 2
	// rescueGold is paid for each tier of the guards, rescueXP for each
	// danger of the place; rescueStanding is what the settlement grants.
	rescueGold     = 15
	rescueXP       = 25
	rescueStanding = 3
	// maxPerson and maxRelation keep the names the Mind gives short.
	maxPerson   = 40
	maxRelation = 30
)

// QuestHookRequest asks a model whether a keeper's line tells of someone
// dear to them gone missing or in danger.
type QuestHookRequest struct {
	// Speaker is the keeper's name, Card who they are, Known what they
	// said of themselves before.
	Speaker string
	Card    string
	Known   []string
	// Said is the player's line, Reply the keeper's answer to it.
	Said  string
	Reply string
	// Monsters and Places are the names the model may pick from.
	Monsters []string
	Places   []string
}

// QuestHook is what a model read in the line: who is missing, who they
// are to the keeper, the monster and the place named with them, and the
// words that said so.
type QuestHook struct {
	Found    bool   `json:"found"`
	Person   string `json:"person"`
	Relation string `json:"relation"`
	Monster  string `json:"monster"`
	Place    string `json:"place"`
	Quote    string `json:"quote"`
}

// QuestHooker is a model able to hear a call for help in a keeper's line.
type QuestHooker interface {
	QuestHook(ctx context.Context, request QuestHookRequest) (QuestHook, error)
}

// Folded beginnings of the words fear and loss are told with: "boję się",
// "zaginęła", "nie wróciła", "porwali".
var (
	dangerStems   = []string{"boje", "boimy", "obaw", "strach", "martwi", "zagin", "przepad", "znikn", "porwa", "porwal", "uprowadz", "pojma", "uwiez", "niewol", "potwor", "niebezpiecz"}
	dangerPhrases = []string{"boi sie", "nie wrocil", "nie wrocila", "nie wrocili", "nie wraca", "nie ma jej", "nie ma go", "slad po"}
)

// danger reports whether a line speaks of fear or loss, or names a
// monster or a dangerous place of the world.
func (r *Realm) danger(line string) bool {
	words := strings.Fields(Fold(line))
	padded := " " + strings.Join(words, " ") + " "

	if slices.ContainsFunc(words, func(word string) bool {
		return slices.ContainsFunc(dangerStems, func(stem string) bool { return strings.HasPrefix(word, stem) })
	}) || slices.ContainsFunc(dangerPhrases, func(phrase string) bool { return strings.Contains(padded, " "+phrase+" ") }) {
		return true
	}

	return slices.ContainsFunc(append(r.monsterNames(), r.wildNames()...), func(name string) bool { return whole(words, name) })
}

// monsterNames are the kinds of monster of the world.
func (r *Realm) monsterNames() []string {
	names := make([]string, 0, len(r.world.Monsters))
	for _, monster := range r.world.Monsters {
		names = append(names, monster.Name)
	}

	return names
}

// wildNames are the places where a captive may be held: those with any
// danger, once each.
func (r *Realm) wildNames() []string {
	var names []string

	for _, place := range r.places {
		if place.Danger > 0 && !slices.Contains(names, place.Name) {
			names = append(names, place.Name)
		}
	}

	return names
}

// named finds a name a model gave among a list, also in another form:
// "Nurka bez Twarzy" is the "Nurek bez Twarzy". It returns -1 for none.
func named(names []string, text string) int {
	if strings.TrimSpace(text) == "" {
		return -1
	}

	for index, name := range names {
		if strings.EqualFold(name, strings.TrimSpace(text)) {
			return index
		}
	}

	words := strings.Fields(Fold(text))
	for index, name := range names {
		if whole(words, name) && whole(strings.Fields(Fold(name)), text) {
			return index
		}
	}

	return -1
}

// openRescue is a keeper's rescue not yet paid for.
func (r *Realm) openRescue(keeper int) *quest {
	for _, job := range r.quests {
		if job.kind == questRescue && job.keeper == keeper && !job.paid {
			return job
		}
	}

	return nil
}

// rescues counts the rescues open in the world.
func (r *Realm) rescues() int {
	count := 0

	for _, job := range r.quests {
		if job.kind == questRescue && !job.paid {
			count++
		}
	}

	return count
}

// rescuable reports whether the keeper of a talk may ask for a rescue
// now: a shopkeeper with none open, in a world with monsters and room
// for one more.
func (r *Realm) rescuable(keeper int) bool {
	return !r.investigation() && !r.adventure() && len(r.world.Monsters) > 0 && keeper >= 0 && keeper < len(r.world.Shops) &&
		r.openRescue(keeper) == nil && r.rescues() < maxRescues
}

// QuestHookRequest is the request to read whether the shopkeeper a player
// talks with has just told of someone dear gone missing: a line about
// their own life, or one that answers a private question or follows what
// they said of themselves, that also speaks of fear, loss or a monster.
// It reports false for any other talk.
func (r *Realm) QuestHookRequest(name, said, reply string) (QuestHookRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer := r.active(name)
	if customer == nil || !r.rescuable(customer.shop) {
		return QuestHookRequest{}, false
	}

	speaker := r.world.Shops[customer.shop].Keeper
	_, line := speech.SplitMood(reply)

	var known []string
	if bio, ok := r.bioName(customer.shop); ok {
		known = r.bioTexts(bio)
	}

	// Kin the keeper's card names count as kin said: "Wit zszedł do
	// Gnilnych Komór" from Stara Ruda, whose card has her son Wit.
	kin := cardKin(r.world.Shops[customer.shop].Persona, speaker)
	personal := Personal(line, speaker) || Private(said) || len(known) > 0 ||
		slices.ContainsFunc(kin, func(name string) bool { return personNamed(line+" "+said, name) })
	if strings.TrimSpace(line) == "" || !personal || !r.danger(line) {
		return QuestHookRequest{}, false
	}

	return QuestHookRequest{
		Speaker: speaker, Card: r.bioCard(customer.shop), Known: known, Said: said, Reply: line,
		Monsters: r.monsterNames(), Places: r.wildNames(),
	}, true
}

// RescueGrounded reports whether a hook the Mind read is in what the
// keeper said: the person named in the line or in what they said of
// themselves before, and the quote taken from the line.
func RescueGrounded(hook QuestHook, line string, known []string) bool {
	quote := bioKey(hook.Quote)

	return quote != "" && strings.Contains(" "+bioKey(line)+" ", " "+quote+" ") &&
		personNamed(line+" "+strings.Join(known, " "), hook.Person)
}

// personNamed reports whether a text names a person in any form: every
// word of the name is in it, maybe declined, "Antka" and "Antkiem" for
// "Antek", "Pawła" for "Paweł", "Rafiego" for "Rafi".
func personNamed(text, person string) bool {
	words := strings.Fields(Fold(text))
	matched := false

	for _, part := range strings.Fields(Fold(person)) {
		if len(part) < minQueryWord {
			continue
		}

		if !slices.ContainsFunc(words, func(word string) bool { return word == part || formOf(word, part) }) {
			return false
		}

		matched = true
	}

	return matched
}

// kinCard are the folded words of kin a card names somebody's relation
// by, before their name: "jej syn Wit", "ma córkę Zosię".
var kinCard = []string{
	"syn", "syna", "synem", "synek", "synka", "corka", "corke", "corki", "coreczka", "coreczke", "brat", "brata", "bratem", "siostra", "siostre", "siostry",
	"zona", "zone", "zony", "maz", "meza", "mezem", "ojciec", "ojca", "matka", "matke", "matki", "wnuk", "wnuka", "wnuczka", "wnuczke", "dziadek", "dziadka",
	"babka", "babke", "babcia", "babcie", "bratanek", "bratanka", "bratanica", "siostrzeniec", "siostrzenca", "siostrzenica", "kuzyn", "kuzyna", "kuzynka", "kuzynke",
	"narzeczony", "narzeczonego", "narzeczona", "narzeczonej", "blizniak", "blizniaka", "blizniaczka", "blizniaczke", "wuj", "wuja", "wujek", "wujka", "ciotka", "ciotke", "przyjaciel", "przyjaciela", "przyjaciolka", "przyjaciolke",
}

// cardKin lists the names of the kin a keeper's card tells of: the word
// said with a capital right after a word of kin, "jej syn Wit kopie". The
// speaker's own name is none: "mąż Babki Smolnej" is Babka Smolna herself.
func cardKin(card, speaker string) []string {
	var names []string

	tokens := strings.Fields(card)
	for index, token := range tokens[:max(0, len(tokens)-1)] {
		folded := strings.TrimSpace(Fold(token))
		next := strings.Trim(tokens[index+1], ".,;:!?()„”\"«»")
		first, _ := utf8.DecodeRuneInString(next)

		if slices.Contains(kinCard, folded) && !strings.ContainsAny(token, ".;:!?") && unicode.IsUpper(first) && len(next) >= minQueryWord && !slices.Contains(names, next) && !personNamed(speaker, next) {
			names = append(names, next)
		}
	}

	return names
}

// SpawnRescue makes a job of the call for help a model heard in a
// keeper's line: the missing person is held on a wild tile under guards,
// and the hero who heard it has the job at once. It returns what the
// hero is told, or false when the hook does not hold.
func (r *Realm) SpawnRescue(name, speaker string, hook QuestHook, line string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	hero := r.active(name)
	keeper := slices.IndexFunc(r.world.Shops, func(shop world.ShopSpec) bool { return shop.Keeper == speaker })

	if hero == nil || !hook.Found || !r.rescuable(keeper) {
		return "", false
	}

	_, line = speech.SplitMood(line)

	var known []string
	if bio, ok := r.bioName(keeper); ok {
		known = r.bioTexts(bio)
	}

	person := strings.Trim(strings.Join(strings.Fields(hook.Person), " "), "\"„”«».,!?;:")
	relation := strings.Trim(strings.ToLower(strings.Join(strings.Fields(hook.Relation), " ")), "\"„”«».,!?;:")
	if !r.personName(person, relation, speaker, hero) || relation == "" || len(relation) > maxRelation || !RescueGrounded(hook, line, known) {
		return "", false
	}

	town := r.townOf(hero)
	if town < 0 {
		town = hero.home
	}

	monsters, places := r.monsterNames(), r.wildNames()
	kind := named(monsters, hook.Monster)

	place := -1
	if found := named(places, hook.Place); found >= 0 {
		place = slices.IndexFunc(r.places, func(spec world.PlaceSpec) bool { return spec.Name == places[found] })
	}

	at, ok := r.rescueTile(r.world.Towns[town], kind, place)
	if !ok {
		return "", false
	}

	job := &quest{
		id: len(r.quests) + 1, keeper: keeper, kind: questRescue, subject: person, relation: relation,
		monster: r.guard(at, kind), pos: at, town: town, taker: hero.name, day: r.day,
	}

	for _, foe := range r.foes(at) {
		job.reward += foe.tier * rescueGold
	}

	r.withItem(job)
	r.quests = append(r.quests, job)
	r.revealed[r.world.Index(at)] = true

	she := rescueFeminine(person, relation)
	news := fmt.Sprintf("%s martwi się o %s %s: %s w miejscu %s, %s %s. Odnaleźć %s %s %s.",
		speaker, accusative(relation, she), accusative(person, she), gendered(she, "zaginął", "zaginęła"), r.placeAt(at).Name,
		gendered(she, "strzeże go", "strzeże jej"), r.world.Monsters[job.monster].Name,
		gendered(she, "go", "ją"), gendered(speech.Feminine(hero.name), "obiecał", "obiecała"), hero.name)
	if fact := r.add(person, news, speaker, r.everyone()...); fact != nil {
		r.learn(hero, fact)
	}

	told := fmt.Sprintf("Nowe zlecenie od %s: %s. Po uwolnieniu wróć do %s po nagrodę.", speaker, r.questText(job), speaker)
	r.tell(hero, told)

	return told, true
}

// personName reports whether a name a model gave can be a missing
// person's: a name, short, said with a capital, and nobody of the world
// already: not the keeper, the hero, a companion or a monster.
func (r *Realm) personName(person, relation, speaker string, hero *player) bool {
	first, _ := utf8.DecodeRuneInString(person)
	if person == "" || len(person) > maxPerson || len(strings.Fields(person)) > 3 || !unicode.IsUpper(first) || strings.EqualFold(person, relation) {
		return false
	}

	taken := append([]string{speaker, hero.name}, r.monsterNames()...)
	for _, spec := range r.world.Companions {
		taken = append(taken, spec.Name)
	}

	return !slices.ContainsFunc(taken, func(name string) bool { return bioKey(name) == bioKey(person) })
}

// rescueTile picks where a captive is held: a wild tile with some danger,
// never a settlement, a site, the lair, a fight or a hero; best the place
// or among the monsters the keeper named, and at a fair walk from the
// settlement.
func (r *Realm) rescueTile(from world.Point, kind, place int) (world.Point, bool) {
	best, bestScore, found := world.Point{}, 0, false

	for index, tile := range r.world.Tiles {
		at := world.Point{X: index % world.Width, Y: index / world.Width}

		if tile.Town >= 0 || tile.Site >= 0 || tile.Place < 0 || tile.Boss || at == r.world.Lair || r.world.Near(at) ||
			r.placeAt(at).Danger < 1 || r.someoneAt(at) || r.fights[at] != nil || r.held(at) {
			continue
		}

		score := -abs(abs(at.X-from.X) + abs(at.Y-from.Y) - rescueSteps)
		if place >= 0 && tile.Place == place {
			score += 20
		}

		if kind >= 0 && slices.ContainsFunc(r.foes(at), func(foe *monster) bool { return foe.kind == kind }) {
			score += 10
		}

		// Ties go to chance, not to the top of the map.
		score = score*10 + r.rng.Intn(10)
		if !found || score > bestScore {
			best, bestScore, found = at, score, true
		}
	}

	return best, found
}

// held reports whether a captive of an open rescue is held on a tile.
func (r *Realm) held(at world.Point) bool {
	return slices.ContainsFunc(r.quests, func(job *quest) bool { return job.kind == questRescue && !job.finished && job.pos == at })
}

// guard sets guards on a captive's tile: the monster the keeper named,
// when the world has it, and a pack fit for the place, so that at least
// a few stand there. Every living monster of the tile stays put from now
// on. It returns the kind that leads them.
func (r *Realm) guard(at world.Point, kind int) int {
	index := r.world.Index(at)
	want := rescueGuards + r.rng.Intn(2)

	var kinds []int
	if kind >= 0 {
		kinds = append(kinds, kind)
	}

	if missing := want - r.alive(at) - len(kinds); missing > 0 {
		kinds = append(kinds, r.world.Pack(r.rng, max(r.placeAt(at).Danger, 1), missing)...)
	}

	for _, added := range kinds {
		r.monsters[index] = append(r.monsters[index], r.newMonster(added))
	}

	leader := kind
	for _, foe := range r.foes(at) {
		foe.pinned = true
		if leader < 0 || leader != kind && foe.tier > r.world.Monsters[leader].Tier {
			leader = foe.kind
		}
	}

	return leader
}

// rescueFeminine tells a missing person's gender by who they are to the
// keeper, "córka" or "brat", and by the name when that says nothing.
func rescueFeminine(person, relation string) bool {
	switch word := Fold(strings.Fields(relation + " x")[0]); {
	case slices.Contains([]string{"tata", "kolega", "sluga", "wujek"}, word):
		return false
	case strings.HasSuffix(word, "a"):
		return true
	case word != "x" && word != "":
		return false
	}

	return speech.Feminine(person)
}

// accusative puts the first word of a name or a word of kin into the
// form a verb takes it in: "Zosia" is freed as "Zosię", "Radek" as
// "Radka", "córka" sought as "córkę". Other words are left as they are.
func accusative(name string, feminine bool) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return name
	}

	word := []rune(words[0])
	lower := strings.ToLower(words[0])
	last := len(word) - 1

	// A soft consonant before a fleeting e softens what is left: "Maniek"
	// is sought as "Mańka", "Czesiek" as "Cześka".
	soft := map[rune]rune{'s': 'ś', 'z': 'ź', 'c': 'ć', 'n': 'ń'}

	switch {
	case lower == "mąż":
		word = append(word[:last-1], 'ę', 'ż', 'a')
	case lower == "ojciec":
		word = append(word[:3], 'a')
	case lower == "uczeń":
		word = append(word[:3], 'n', 'i', 'a')
	case lower == "szwagier":
		word = append(word[:5], 'r', 'a')
	case strings.HasSuffix(lower, "a"):
		word[last] = 'ę'
	case feminine:
	case strings.HasSuffix(lower, "o"):
		word[last] = 'a'
	case strings.HasSuffix(lower, "iec") && last > 3:
		word = append(word[:last-2], 'c', 'a')
	case strings.HasSuffix(lower, "iek") && last > 3 && soft[word[last-3]] != 0:
		word = append(word[:last-3], soft[word[last-3]], 'k', 'a')
	case strings.HasSuffix(lower, "ek") && last > 2:
		word = append(word[:last-1], 'k', 'a')
	// "Paweł" and "Gaweł" lose their e: "Pawła".
	case strings.HasSuffix(lower, "eł") && last > 2:
		word = append(word[:last-1], 'ł', 'a')
	// A man's name in -i is declined like an adjective: "Rafiego",
	// "Antoniego"; after an l the i goes: "Alego".
	case strings.HasSuffix(lower, "li") && last > 1:
		word = append(word[:last], []rune("ego")...)
	case strings.HasSuffix(lower, "i") && last > 1:
		word = append(word, []rune("ego")...)
	// "Staś" and "Jaś" are sought as "Stasia" and "Jasia".
	case strings.ContainsRune("śćńź", unicode.ToLower(word[last])) && last > 0:
		hard := map[rune]rune{'ś': 's', 'ć': 'c', 'ń': 'n', 'ź': 'z'}
		word = append(word[:last], hard[unicode.ToLower(word[last])], 'i', 'a')
	case !strings.ContainsRune("aąeęiouóy", unicode.ToLower(word[last])):
		word = append(word, 'a')
	}

	words[0] = string(word)

	return strings.Join(words, " ")
}

// rescueText words a rescue for the journal and the keeper.
func (r *Realm) rescueText(job *quest) string {
	she := rescueFeminine(job.subject, job.relation)
	guard := "strażnicy"

	if job.monster >= 0 && job.monster < len(r.world.Monsters) {
		guard = r.world.Monsters[job.monster].Name
	}

	return fmt.Sprintf("odnajdź i uwolnij %s (%s): %s %s; miejsce: %s, od osady %s %s",
		accusative(job.subject, she), job.relation, gendered(she, "strzeże go", "strzeże jej"), guard,
		r.placeAt(job.pos).Name, r.townName(job.town), bearing(r.world.Towns[job.town], job.pos))
}

// free lets a captive go with the hero who came for them.
func (r *Realm) free(hero *player, job *quest) {
	job.finished = true
	r.tell(hero, fmt.Sprintf("Uwalniasz %s. Wraca z tobą do osady %s. Wróć do %s po nagrodę.",
		accusative(job.subject, rescueFeminine(job.subject, job.relation)), r.townName(job.town), r.shopSpec(job.keeper).Keeper))
}

// rescued settles a rescue once paid: experience, the settlement's
// gratitude, news everyone tells and the keeper's own memory of it.
func (r *Realm) rescued(hero *player, job *quest) {
	keeper := r.shopSpec(job.keeper).Keeper
	she := rescueFeminine(job.subject, job.relation)
	xp := rescueXP * max(r.placeAt(job.pos).Danger, 1)

	r.tell(hero, fmt.Sprintf("%s: %s wraca do domu cał%s. Doświadczenie: +%d.", keeper, job.subject, gendered(she, "y", "a"), xp))
	r.reward(hero, xp)
	r.esteem(hero, job.town, rescueStanding, "Cała osada o tym mówi.")

	// The keeper's own past now has the rescue in it, so that they speak
	// of the child as safe, and so does the news.
	kept := fmt.Sprintf("%s odzyskał%s %s %s: %s %s %s.", keeper, gendered(speech.Feminine(keeper), "", "a"),
		accusative(job.relation, she), accusative(job.subject, she),
		gendered(speech.Feminine(hero.name), "uwolnił", "uwolniła"), gendered(she, "go", "ją"), hero.name)
	r.remember(keeper, kept, r.day, hero.name)

	// The news of the missing one turns into news of the rescue: left as
	// it was, it had the keepers still worrying.
	for _, fact := range r.wisdom.facts {
		if fact.Subject == job.subject && fact.Source == keeper {
			fact.Text, fact.Day = kept, r.day
			r.learn(hero, fact)

			return
		}
	}

	if fact := r.add(job.subject, kept, keeper, r.everyone()...); fact != nil {
		r.learn(hero, fact)
	}
}

// rescueLore tells a keeper of their missing one: still held, freed, and
// who went looking.
func (r *Realm) rescueLore(customer *player, job *quest) string {
	she := rescueFeminine(job.subject, job.relation)
	yours := gendered(she, "Twój", "Twoja")
	where := fmt.Sprintf("%s %s w miejscu %s (%s)", gendered(she, "strzeże go", "strzeże jej"), r.world.Monsters[job.monster].Name,
		r.placeAt(job.pos).Name, bearing(customer.pos, job.pos))

	switch {
	case job.finished && job.taker == customer.name:
		hero := speech.Feminine(customer.name)

		return fmt.Sprintf("Gracz uwolnił%s %s %s %s i odprowadził%s %s do domu. Dziękujesz z całego serca.",
			gendered(hero, "", "a"), gendered(she, "twojego", "twoją"), accusative(job.relation, she), accusative(job.subject, she),
			gendered(hero, "", "a"), gendered(she, "go", "ją"))
	case job.taker == customer.name:
		return fmt.Sprintf("%s %s %s %s; %s. Gracz obiecał %s odnaleźć i uwolnić. Martwisz się i czekasz na wieści; nie mów, że już %s.",
			yours, job.relation, job.subject, gendered(she, "zaginął", "zaginęła"), where, gendered(she, "go", "ją"), gendered(she, "wrócił", "wróciła"))
	}

	return fmt.Sprintf("%s %s %s %s; %s. Szuka %s już %s.", yours, job.relation, job.subject, gendered(she, "zaginął", "zaginęła"), where,
		gendered(she, "go", "jej"), job.taker)
}
