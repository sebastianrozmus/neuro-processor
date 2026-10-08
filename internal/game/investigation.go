package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The investigation: an adventure with a hidden truth. The people of a
// town are the keepers of the knowledge graph, each knowing a piece and
// telling it only when its conditions hold; things hold clues found by
// examining them; everybody follows a day's schedule, so a detective
// who waits or follows sees who goes where; and the case ends with an
// accusation that has to be backed by the facts found.

const (
	// maxMistakes is how many wrong accusations the client puts up with.
	maxMistakes    = 3
	personPatience = 8
)

func (r *Realm) investigation() bool {
	return r.world.Mode == world.ModeInvestigation
}

// personSpec describes a person of the town as a keeper to talk to.
func (r *Realm) personSpec(index int) world.ShopSpec {
	person := r.world.People[index]

	return world.ShopSpec{Name: person.Name, Keeper: person.Name, Persona: person.Description + " " + person.Talk, Greeting: person.Talk}
}

// personMerchant is a person as a talking partner: nothing to sell, a
// character and some patience.
func (r *Realm) personMerchant(index int) Merchant {
	spec := r.personSpec(index)

	return Merchant{Name: spec.Keeper, Persona: spec.Persona, Greeting: spec.Greeting, Patience: personPatience, Currency: r.world.Currency}
}

// personIndex finds a person by name among everyone in the town.
func (r *Realm) personIndex(name string) int {
	for index, person := range r.world.People {
		if Matches(person.Name, name) {
			return index
		}
	}

	return -1
}

// personRoom is where a person is at a hero's time of day.
func (r *Realm) personRoom(person world.PersonSpec, hero *player) string {
	if hero != nil && person.Schedule != nil {
		if where, ok := person.Schedule[r.hourName(hero)]; ok {
			return where
		}
	}

	return person.Room
}

// roomPos is where a room lies on the map.
func (r *Realm) roomPos(name string) (world.Point, bool) {
	for _, room := range r.world.Rooms {
		if room.Name == name {
			return world.Point{X: room.X, Y: room.Y}, true
		}
	}

	return world.Point{}, false
}

// talkTo opens a conversation with a person of the town, run by the model
// like a shop talk, with the person's clues as what they know.
func (r *Realm) talkTo(speaker *player, index int) {
	person := r.world.People[index]

	if r.story.flags[refusal(person.Name)] {
		r.tell(speaker, person.Name+" odwraca się plecami: po tym oskarżeniu nie zamieni z tobą słowa.")

		return
	}

	if speaker.talks == nil {
		speaker.talks = map[int]*talk{}
	}

	if speaker.talks[index] == nil {
		r.talks++
		merchant := r.personMerchant(index)
		speaker.talks[index] = &talk{session: NewSession(merchant, speaker.gold), serial: r.talks}
		speaker.talks[index].session.Favor(Modifier(speaker.abilities[Charisma]))
		speaker.talks[index].session.Remember(false, merchant.Greeting)
	}

	speaker.shop = index
	r.tell(speaker, fmt.Sprintf("Rozmawiasz z: %s. %s", person.Name, person.Description))
}

// refusal is the flag of a person who was wrongly accused.
func refusal(name string) string {
	return "odmowa:" + name
}

// caseLore words what a person of the town knows for the model.
func (r *Realm) caseLore(customer *player, index int) string {
	built := r.world
	person := built.People[index]

	var lore strings.Builder

	fmt.Fprintf(&lore, "Nie prowadzisz sklepu. Jesteś: %s. %s\n", person.Name, person.Description)
	fmt.Fprintf(&lore, "Miasteczko: %s. %s\n", built.Name, built.Intro)
	fmt.Fprintf(&lore, "Jesteś teraz w miejscu: %s. Dzień %d, pora: %s.\n", r.placeAt(customer.pos).Name, customer.day, r.hourName(customer))

	if built.Case != nil {
		fmt.Fprintf(&lore, "Twój rozmówca to %s, detektyw; wynajął go %s. Sprawa: %s\n", customer.name, built.Case.Client, built.Goal)
	}

	var others []string
	for other, someone := range built.People {
		if other != index {
			others = append(others, fmt.Sprintf("%s (%s)", someone.Name, r.personRoom(someone, customer)))
		}
	}

	if len(others) > 0 {
		fmt.Fprintf(&lore, "Inni w miasteczku i gdzie są teraz: %s.\n", join(others))
	}

	for _, fact := range r.openTo(customer, index) {
		fmt.Fprintf(&lore, "Wiesz: %s. %s\n", fact.Subject, fact.Text)
	}

	if gaps := r.gapsFor(customer, index); len(gaps) > 0 {
		fmt.Fprintf(&lore, "Nie wiesz nic o: %s. Jeśli detektyw o to pyta, odeślij go do tej osoby.\n", join(gaps[:min(len(gaps), referrals)]))
	}

	lore.WriteString("Nie wymyślaj nazwisk, adresów, numerów ani zdarzeń, których nie ma wyżej; czego nie wiesz, tego nie wiesz. O tym, co wiesz, mówisz tylko zapytany albo gdy pasuje do rozmowy.")

	return lore.String()
}

// open reports whether a fact's conditions hold for a hero: the flag it
// needs, the fact they must have heard first, the time of day.
func (r *Realm) open(fact *Fact, hero *player) bool {
	switch {
	case fact.needs != "" && !r.story.flags[fact.needs]:
		return false
	case fact.heardFirst != "" && !r.heardOf(hero, fact.heardFirst):
		return false
	case fact.hour != "" && r.hourName(hero) != fact.hour:
		return false
	}

	return true
}

// heardOf reports whether a hero has come across a fact about a subject.
func (r *Realm) heardOf(hero *player, subject string) bool {
	for _, fact := range r.wisdom.facts {
		if fact.heard[hero.name] && (strings.EqualFold(fact.Subject, subject) || Matches(fact.Subject, subject)) {
			return true
		}
	}

	return false
}

// openTo lists the facts a keeper knows and would tell this hero now.
func (r *Realm) openTo(hero *player, keeper int) []*Fact {
	var open []*Fact
	for _, fact := range r.knownBy(keeper) {
		if r.open(fact, hero) {
			open = append(open, fact)
		}
	}

	return open
}

// gapsFor lists what a keeper does not know but another would tell this
// hero, with who to ask.
func (r *Realm) gapsFor(hero *player, keeper int) []string {
	var gaps []string

	for _, fact := range r.wisdom.facts {
		if fact.known[keeper] || !r.open(fact, hero) {
			continue
		}

		for other := range fact.known {
			if other != keeper {
				gaps = append(gaps, fmt.Sprintf("%s (wie o tym %s)", fact.Subject, r.keeperName(other)))

				break
			}
		}
	}

	return gaps
}

// seedClues puts the clues of people and things into the graph.
func (r *Realm) seedClues() {
	for index, person := range r.world.People {
		for _, clue := range person.Clues {
			r.addClue(clue, "", index)
		}
	}

	for _, thing := range r.world.Things {
		for _, clue := range thing.Clues {
			r.addClue(clue, thing.Name)
		}
	}
}

// addClue adds a gated fact, held by a thing or known by keepers.
func (r *Realm) addClue(clue world.ClueSpec, thing string, keepers ...int) *Fact {
	fact := r.add(clue.Subject, clue.Text, "", keepers...)
	if fact != nil {
		fact.needs, fact.heardFirst, fact.hour, fact.thing = clue.Needs, clue.Heard, clue.Hour, thing
	}

	return fact
}

// inspect gives a hero the clues a thing holds, when their conditions
// hold; it reports whether anything was found.
func (r *Realm) inspect(viewer *player, thing string) bool {
	found := false

	for _, fact := range r.wisdom.facts {
		if fact.thing == "" || fact.thing != thing || !r.open(fact, viewer) {
			continue
		}

		if !fact.heard[viewer.name] {
			r.tell(viewer, "Zauważasz: "+fact.Text+" (dziennik: "+fact.Subject+")")
		}

		r.learn(viewer, fact)
		found = true
	}

	return found
}

// passHour is what an hour's passing means in a town: people move by
// their schedules, a detective following somebody goes with them, and
// the steps that wait on hours run.
func (r *Realm) passHour(hero *player, before string) {
	now := r.hourName(hero)
	if now == before {
		return
	}

	room := r.roomOf(hero).Name
	r.tell(hero, "Nastaje "+now+".")

	var gone, came []string
	followed := ""

	for _, person := range r.world.People {
		was := r.personRoomAt(person, before)
		is := r.personRoom(person, hero)

		switch {
		case was == room && is != room:
			gone = append(gone, fmt.Sprintf("%s (do: %s)", person.Name, is))
			if hero.following == person.Name {
				followed = is
			}
		case was != room && is == room:
			came = append(came, person.Name)
		}
	}

	if len(gone) > 0 {
		r.tell(hero, "Wychodzi: "+join(gone)+".")
	}

	if len(came) > 0 {
		r.tell(hero, "Przychodzi: "+join(came)+".")
	}

	if followed != "" {
		if at, ok := r.roomPos(followed); ok {
			r.tell(hero, "Idziesz za "+hero.following+".")
			r.place(hero, at, r.world.At(hero.pos).Place)
		}
	}

	r.act(hero, world.StepWait, "", r.roomOf(hero).Name)
	r.deadline(hero)
}

// personRoomAt is where a person is at a named time of day.
func (r *Realm) personRoomAt(person world.PersonSpec, hour string) string {
	if where, ok := person.Schedule[hour]; ok {
		return where
	}

	return person.Room
}

// wait lets an hour pass where the hero stands.
func (r *Realm) wait(hero *player) {
	r.tell(hero, "Czekasz.")
	r.hour(hero)
}

// shadow sets a hero to follow somebody who is in the room.
func (r *Realm) shadow(hero *player, name string) {
	if name == "" || Matches("nikogo stop przestan koniec", name) {
		hero.following = ""
		r.tell(hero, "Nikogo nie śledzisz.")

		return
	}

	person, ok := r.personAt(hero, name)
	if !ok {
		r.tell(hero, "Nie ma tu takiej osoby. Śledzić można kogoś, kto jest w tym samym miejscu.")

		return
	}

	hero.following = person.Name
	r.tell(hero, fmt.Sprintf("Śledzisz: %s. Gdy wyjdzie, pójdziesz za nim; «czekaj» przesuwa czas o godzinę.", person.Name))
}

// accuse ends the case, or spoils it a little.
func (r *Realm) accuse(hero *player, name string) {
	spec := r.world.Case
	if spec == nil {
		r.tell(hero, "Nie ma tu sprawy do rozstrzygnięcia.")

		return
	}

	if name == "" {
		r.tell(hero, "Kogo oskarżasz? Napisz np. «oskarżam "+strings.Fields(spec.Answer)[0]+"».")

		return
	}

	if !Matches(spec.Answer, name) {
		r.story.mistakes++
		accused := name
		if index := r.personIndex(name); index >= 0 {
			accused = r.world.People[index].Name
			r.story.flags[refusal(accused)] = true
		}

		r.tell(hero, fmt.Sprintf("Oskarżasz: %s. Nikt ci nie wierzy, a %s nie odezwie się już do ciebie. Pomyłka %d z %d, na jakie pozwala zleceniodawca.", accused, accused, r.story.mistakes, maxMistakes))

		if r.story.mistakes >= maxMistakes {
			r.lose(hero, fmt.Sprintf("%s wycofuje sprawę: «Dość pomówień.»", spec.Client))
		}

		return
	}

	var missing []string
	for _, subject := range spec.Proof {
		if !r.heardOf(hero, subject) {
			missing = append(missing, subject)
		}
	}

	if len(missing) > 0 {
		hint := spec.Hint
		if hint == "" {
			hint = "Szukaj dalej: rozmawiaj, badaj rzeczy, przychodź o innych porach."
		}

		r.tell(hero, fmt.Sprintf("Wskazujesz: %s. Może i masz rację, ale bez dowodów nikt ci nie uwierzy: brakuje %d z %d poszlak. %s", spec.Answer, len(missing), len(spec.Proof), hint))

		return
	}

	r.won = true
	r.tell(hero, fmt.Sprintf("Oskarżasz: %s, i kładziesz na stole dowody: %s.", spec.Answer, join(spec.Proof)))

	for _, name := range r.order {
		r.tell(r.players[name], spec.Verdict)
		r.tell(r.players[name], fmt.Sprintf("Sprawa «%s» rozwiązana!", r.world.Name))
	}
}

// deadline checks the clock of a game with a time limit.
func (r *Realm) deadline(hero *player) {
	if r.world.Deadline == 0 || r.lost || r.won {
		return
	}

	elapsed := (hero.day-1)*stepsPerDay + hero.steps
	left := r.world.Deadline - elapsed

	switch {
	case left <= 0:
		loss := r.world.Loss
		if loss == "" {
			loss = "Czas minął."
		}

		r.lose(hero, loss)
	case left <= 3:
		r.tell(hero, fmt.Sprintf("Zostały ci %d godziny.", left))
	}
}

// lose ends the game in failure.
func (r *Realm) lose(hero *player, text string) {
	if r.lost {
		return
	}

	r.lost = true

	for _, name := range r.order {
		r.tell(r.players[name], text)
		r.tell(r.players[name], fmt.Sprintf("Gra «%s» przegrana. Zacznij od nowa albo wczytaj zapis.", r.world.Name))
	}
}

// hoursLeft is how many hours a hero has before the deadline, or -1.
func (r *Realm) hoursLeft(hero *player) int {
	if r.world.Deadline == 0 {
		return -1
	}

	return max(0, r.world.Deadline-((hero.day-1)*stepsPerDay+hero.steps))
}

// InvestigationHelp lists the commands of an investigation.
const InvestigationHelp = "Polecenia: idź na północ / południe / wschód / zachód (albo n, s, e, w) · rozejrzyj się · zbadaj X (rzeczy kryją poszlaki) · podnieś X · użyj X na Y · otwórz X · daj X Y · " +
	"rozmawiaj z Y (rozmowa prowadzona przez model; Esc kończy) · czekaj (mija godzina, ludzie zmieniają miejsca) · śledź Y (pójdziesz za nim, gdy wyjdzie) · " +
	"dziennik (zebrane poszlaki) · oskarżam Y (koniec sprawy, gdy masz dowody; trzy pomyłki i zleceniodawca rezygnuje) · plecak · zapisz · pomoc · wyjdź."
