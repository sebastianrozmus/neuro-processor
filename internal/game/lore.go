package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	nearSteps = 3
	farSteps  = 7
)

// lore words what the keeper of the shop a customer is in knows: their own
// wares, the world and its dangers, where things lie from here, who else
// trades in the settlement and what the customer looks like. It gives
// conversations something true to be about.
func (r *Realm) lore(customer *player, wares []Ware) string {
	if r.investigation() {
		return r.caseLore(customer, customer.shop)
	}

	if kind, ok := r.mateOf(customer.shop); ok {
		return r.mateLore(customer, kind, wares)
	}

	built := r.world
	here := customer.pos

	var lore strings.Builder

	stock := make([]string, 0, len(wares))
	for _, ware := range wares {
		stock = append(stock, fmt.Sprintf("«%s» %d", ware.Name, ware.Price))
	}

	fmt.Fprintf(&lore, "Twój towar i ceny za sztukę (%s): %s.\n", built.Currency, join(stock))

	fmt.Fprintf(&lore, "Świat: %s. %s\n", built.Name, built.Intro)

	if style := built.Style; !style.Empty() {
		fmt.Fprintf(&lore, "Styl świata: %s Tak tu mówią: %s\n", style.Look, style.Dialogue)
	}

	if r.caravan != nil && customer.shop == r.caravanShop() {
		fmt.Fprintf(&lore, "Jesteś wędrownym kupcem; twoja karawana stoi dziś w miejscu: %s. %s\n", r.placeAt(here).Name, r.placeAt(here).Description)
	} else {
		fmt.Fprintf(&lore, "Osada, w której mieszkasz: %s. %s\n", built.PlaceAt(here).Name, built.PlaceAt(here).Description)
	}

	fmt.Fprintf(&lore, "Dzień %d. %s\n", r.day, r.sky())

	if customer.inn {
		lore.WriteString("Jest wieczór: kram zamknięty, siedzisz w karczmie przy stole, a gracz przysiadł się do ciebie. Nie handlujesz po godzinach (zaproś rano), za to chętniej gadasz i powtarzasz, co słyszałeś od sąsiadów.\n")
	}

	if job := r.questLore(customer, customer.shop); job != "" {
		lore.WriteString(job + "\n")
	}

	var neighbours []string

	for index, shop := range built.Shops {
		if index != customer.shop {
			neighbours = append(neighbours, fmt.Sprintf("%s (%s)", shop.Keeper, shop.Name))
		}
	}

	if len(neighbours) > 0 {
		fmt.Fprintf(&lore, "Twoi sąsiedzi w osadzie: %s.\n", join(neighbours))
	}

	for index, town := range built.Towns {
		if town != here {
			fmt.Fprintf(&lore, "Inna osada: %s, %s.\n", built.Places[built.At(built.Towns[index]).Place].Name, bearing(here, town))
		}
	}

	fmt.Fprintf(&lore, "Największe zagrożenie: %s. %s Jego siedziba: %s.\n", built.Boss.Name, built.Boss.Description, bearing(here, built.Lair))

	for _, fact := range r.knownBy(customer.shop) {
		fmt.Fprintf(&lore, "Wiesz: %s. %s\n", fact.Subject, fact.Text)
	}

	if gaps := r.unknownTo(customer.shop); len(gaps) > 0 {
		fmt.Fprintf(&lore, "Nie wiesz nic o: %s. Jeśli gracz o to pyta, odeślij go do tej osoby.\n", join(gaps[:min(len(gaps), referrals)]))
	}

	fmt.Fprintf(&lore, "Twój rozmówca: %s, %s. Broń: %s. Pancerz: %s.",
		customer.name, condition(customer), orNothing(customer.weapon), orNothing(customer.armor))

	if town := r.townOf(customer); town >= 0 {
		score := r.standingOf(customer, town)
		fmt.Fprintf(&lore, " Opinia o nim w osadzie: %s (%+d); traktuj go odpowiednio.", standingLabel(score), score)
	}

	return lore.String()
}

// bearing words where a point lies from another: the compass direction and
// roughly how far.
func bearing(from, to world.Point) string {
	dx, dy := to.X-from.X, to.Y-from.Y

	var vertical, horizontal string

	switch {
	case dy < 0 && abs(dy)*2 >= abs(dx):
		vertical = "północ"
	case dy > 0 && abs(dy)*2 >= abs(dx):
		vertical = "południe"
	}

	switch {
	case dx > 0 && abs(dx)*2 >= abs(dy):
		horizontal = "wschód"
	case dx < 0 && abs(dx)*2 >= abs(dy):
		horizontal = "zachód"
	}

	direction := vertical + horizontal

	switch {
	case vertical == "północ" && horizontal != "":
		direction = "północny " + horizontal
	case vertical == "południe" && horizontal != "":
		direction = "południowy " + horizontal
	}

	distance := "daleko"

	switch steps := abs(dx) + abs(dy); {
	case steps <= nearSteps:
		distance = "blisko"
	case steps <= farSteps:
		distance = "kawałek drogi"
	}

	return fmt.Sprintf("kierunek %s, %s", direction, distance)
}

func condition(hero *player) string {
	switch {
	case hero.hp*3 <= hero.maxHP():
		return "ciężko ranny"
	case hero.hp*3 <= hero.maxHP()*2:
		return "poobijany"
	}

	return "w dobrym zdrowiu"
}

func orNothing(item *world.ItemSpec) string {
	if item == nil {
		return "brak"
	}

	return item.Name
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

// touches counts the player's words (long enough to mean something) that
// begin a word of a text.
func touches(words []string, text string) int {
	count := 0

	for _, word := range words {
		if len(word) >= minQueryWord+1 && Matches(text, word) {
			count++
		}
	}

	return count
}

// kindWords are folded beginnings of the words a player uses for a kind of
// ware without naming one: "jaką masz zbroję", "coś na rany".
var kindWords = map[world.Kind][]string{
	world.KindWeapon: {"bron", "orez", "miecz", "topor", "luk", "sztylet", "noz", "wloczn", "mlot", "pistolet", "karabin", "strzelb"},
	world.KindArmor:  {"zbroj", "pancerz", "kaftan", "kolczug", "oslon", "tarcz", "helm", "kamizel", "ochron"},
	world.KindPotion: {"mikstur", "napar", "eliksir", "wywar", "lekarstw", "leku", "leki", "lecz", "rany", "rane", "rann"},
	world.KindSpell:  {"zakle", "czar", "zwoj", "ksieg", "magi"},
	world.KindCharm:  {"amulet", "talizman", "urok", "pierscien", "wisior"},
}

// hint finds the fact the player's words touch among those the keeper
// knows; when they touch none and only chat, it offers one, so that the
// talk goes somewhere. Whatever is touched, the player has now heard.
func (r *Realm) hint(customer *player, text string, decision Decision, wares []Ware) (string, *lead, bool) {
	switch decision.Outcome {
	case OutcomeChat, OutcomeInfo, OutcomeWares, OutcomeFarewell, OutcomeUnclear:
	default:
		return "", nil, false
	}

	known := r.openTo(customer, customer.shop)
	words := strings.Fields(Fold(text))

	// "z językiem???" after a keeper's line asks what the line meant: the
	// keeper explains it instead of pressing on with another fact.
	if decision.Outcome == OutcomeChat || decision.Outcome == OutcomeUnclear {
		if last := lastReply(customer); last != "" && puzzled(text, last) {
			return fmt.Sprintf("Gracz nie zrozumiał twojej poprzedniej kwestii: „%s”. Wyjaśnij prostymi słowami, co miałeś na myśli, najwyżej dwoma zdaniami. Nie dodawaj nowych nazw ani faktów.", last), nil, true
		}
	}

	// Talk about the goods goes to the goods before anything else: "jaką
	// masz zbroję" is about the mail on the shelf, not about the skeleton
	// whose armour hangs on rust.
	if decision.Outcome == OutcomeChat || decision.Outcome == OutcomeUnclear {
		if fitting := waresAsked(words, wares); len(fitting) > 0 {
			return "Gracz pyta o twój towar. Pasuje: " + join(fitting) + ". Poleć jedną z tych rzeczy, powiedz, do czego jest, i podaj jej cenę z listy.", nil, false
		}
	}

	// A bare greeting is answered in one sentence: the model has never seen
	// one, and given three sentences and a random fact it drifts.
	if decision.Outcome == OutcomeChat && greeting(words) {
		return "Gracz tylko się wita. Odpowiedz jednym krótkim zdaniem: odwzajemnij powitanie po swojemu i zapytaj, czego chce. Nic więcej.", nil, false
	}

	// The fact the player's words touch most: subject words count double,
	// and a single word somewhere in a fact's body is not enough.
	var best *Fact
	bestScore := 1

	for _, fact := range known {
		if score := touches(words, fact.Subject)*2 + touches(words, fact.Text); score > bestScore {
			best, bestScore = fact, score
		}
	}

	if best != nil {
		r.learn(customer, best)

		return fmt.Sprintf("Gracz wspomina o: %s. Co o tym wiesz: %s", best.Subject, best.Text), nil, true
	}

	// Something the keeper lacks but another has is a lead: the keeper
	// passes on the little they heard and names who knows the rest.
	if fact, keeper := r.gapFor(customer, customer.shop, words); fact != nil {
		return r.referral(fact, keeper, false), &lead{fact: fact, keeper: keeper}, true
	}

	// "A co słyszałeś?" after a lead stays on that lead instead of
	// drifting to a random fact; "dobre mają piwo?" is another topic.
	if last := customer.talks[customer.shop]; last != nil && last.lead != nil && decision.Outcome == OutcomeChat && followUp(words, last.lead) {
		return r.referral(last.lead, last.leadKeeper, true), &lead{fact: last.lead, keeper: last.leadKeeper}, true
	}

	if customer.inn && decision.Outcome == OutcomeChat {
		// Evenings at the inn are when a keeper repeats the neighbours' news.
		if gossip := r.gossip(customer.shop); gossip != nil {
			r.learn(customer, gossip)

			return fmt.Sprintf("W karczmie usłyszałeś od sąsiada i możesz powtórzyć: %s. %s", gossip.Subject, gossip.Text), nil, true
		}
	}

	if decision.Outcome == OutcomeChat && len(known) > 0 {
		fact := known[r.rng.Intn(len(known))]
		r.learn(customer, fact)

		// The model was taught to always use the fact; told it may drop one
		// that does not fit, it stops sending a buyer of berries to a mine.
		return fmt.Sprintf("Możesz wpleść jedną rzecz, którą wiesz: %s. %s Jeśli nie pasuje do tego, co mówi gracz, pomiń ją.", fact.Subject, fact.Text), nil, true
	}

	// A greeting or banter with nothing to hang on: without a direction a
	// small model fills the three sentences with noise.
	if decision.Outcome == OutcomeChat {
		return "Gracz zagaduje bez konkretnego pytania o świat. Odpowiedz krótko i po swojemu na to, co powiedział, " +
			"i sam zagadnij go o jedno: po co przyszedł, dokąd idzie albo czy ma czym walczyć.", nil, false
	}

	return "", nil, false
}

// followWords are the stems of words a player asks further with: "a co
// dokładnie słyszałeś?", "opowiedz więcej", "no i?".
var followWords = []string{
	"wiec", "slysz", "powiedz", "opowi", "mow", "gada", "wtedy", "dokladn", "takie", "tego", "tym", "tamt", "tam", "jeszcze", "dalej",
	"czego", "czym", "ktor", "kto", "gdzie", "kiedy", "dlaczego", "czemu", "dobr", "czlowiek", "przyjaciel", "wiesz", "wiem", "chodzi",
	"znacz", "mial", "mysl", "pewn", "napraw", "serio", "prosz", "powtorz", "wspomn", "spraw", "rzecz", "temat", "legend", "histori", "plotk",
}

// followUp reports whether the player's words ask further about a lead
// rather than start something else: every word of any length is either
// small talk, a way of asking further, or from the lead itself.
func followUp(words []string, fact *Fact) bool {
	topic := strings.Fields(Fold(fact.Subject + " " + crumb(fact)))

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) < minQueryWord+1 || slices.Contains(greetingWords, word) {
			continue
		}

		known := false
		for _, stem := range followWords {
			known = known || strings.HasPrefix(word, stem)
		}

		for _, part := range topic {
			known = known || (len(part) >= minQueryWord+1 && strings.HasPrefix(word, Stem(part)))
		}

		if !known {
			return false
		}
	}

	return true
}

// lead is a fact a keeper sent the player to somebody else about.
type lead struct {
	fact   *Fact
	keeper int
}

// referral words the hint for a fact the keeper lacks: the crumb they
// heard and who to ask, by name, so the name can be checked in the reply.
// A follow-up repeats that nothing more is known instead of inventing it.
func (r *Realm) referral(fact *Fact, keeper int, again bool) string {
	who := r.keeperName(keeper)
	if where := r.whereIs(keeper); where != "" {
		who += " (" + where + ")"
	}

	if again {
		return fmt.Sprintf("Gracz dopytuje o to samo: %s. Nic więcej nie wiesz poza tym, co słyszałeś: „%s”. Nie zmyślaj: powtórz krótko, że to wszystko, i że więcej wie %s. Imię podaj dokładnie tak.",
			fact.Subject, crumb(fact), who)
	}

	return fmt.Sprintf("Gracz pyta o coś, czego nie znasz: %s. Słyszałeś tylko tyle: „%s”. Powiedz to jak zasłyszaną plotkę, przyznaj, że więcej nie wiesz, i odeślij go do osoby, która zna sprawę: %s. Imię podaj dokładnie tak.",
		fact.Subject, crumb(fact), who)
}

// lastReply is the keeper's last line in the talk the customer is in,
// without its mood tag.
func lastReply(customer *player) string {
	talk := customer.talks[customer.shop]
	if talk == nil || talk.session == nil {
		return ""
	}

	history := talk.session.History()
	for index := len(history) - 1; index >= 0; index-- {
		if !history[index].Player {
			line := history[index].Text
			if _, rest, tagged := strings.Cut(line, "] "); tagged && strings.HasPrefix(line, "[") {
				line = rest
			}

			return line
		}
	}

	return ""
}

// puzzledCues are folded phrases a player says when a line lost them.
var puzzledCues = []string{
	"nie rozumiem", "nie kumam", "nie lapie", "o co chodzi", "o co ci chodzi", "co to znaczy", "co znaczy", "co masz na mysli",
	"jak to", "ze co", "w sensie", "co ty gadasz", "co ty mowisz", "slucham",
}

// echoFillers are the short words a player may put around the keeper's own
// words when asking what they meant ("z językiem?", "a ta łuna?").
var echoFillers = []string{"a", "i", "z", "ze", "w", "we", "na", "do", "o", "po", "ta", "ten", "to", "te", "no", "ale"}

// puzzled reports a short message that asks what the keeper's last line
// meant: one that says so, a bare "co?", or a question made of nothing but
// the keeper's own words.
func puzzled(text, last string) bool {
	words := strings.Fields(Fold(text))
	if len(words) == 0 || len(words) > 6 {
		return false
	}

	joined := " " + strings.Join(words, " ") + " "
	for _, cue := range puzzledCues {
		if strings.Contains(joined, " "+cue+" ") {
			return true
		}
	}

	if !strings.Contains(text, "?") || len(words) > 3 {
		return false
	}

	if len(words) == 1 && slices.Contains([]string{"co", "he", "hm", "hmm", "ze", "jak"}, words[0]) {
		return true
	}

	said := strings.Fields(Fold(last))
	echoed := false

	for _, word := range words {
		if slices.Contains(echoFillers, word) {
			continue
		}

		if len(word) < minQueryWord+1 || !slices.ContainsFunc(said, func(other string) bool { return strings.HasPrefix(other, Stem(word)) }) {
			return false
		}

		echoed = true
	}

	return echoed
}

// greetingWords are folded words a greeting is made of, with the fillers
// that come with it ("siema mordo", "no hej", "dzień dobry panu").
var greetingWords = []string{
	"siema", "siemka", "siemano", "hej", "hejka", "czesc", "czolem", "witam", "witaj", "witajcie", "dzien", "dobry", "wieczor",
	"elo", "yo", "hello", "hi", "halo", "ej", "no", "mordo", "stary", "ziom", "ziomek", "ziomku", "bracie", "brachu", "panu", "pani", "panie", "kolego", "mistrzu",
}

// greeting reports words that are a greeting and nothing else.
func greeting(words []string) bool {
	if len(words) == 0 || len(words) > 4 {
		return false
	}

	for _, word := range words {
		if !slices.Contains(greetingWords, strings.Trim(word, ".,!?")) {
			return false
		}
	}

	return true
}

// waresAsked lists the wares the player's words point at: by the kind of
// thing they ask for, by name or by what the description says; three at
// most, with prices, as the brief wants them.
func waresAsked(words []string, wares []Ware) []string {
	var kinds []world.Kind

	for kind, stems := range kindWords {
		for _, word := range words {
			if len(word) < minQueryWord || slices.Contains(kinds, kind) {
				continue
			}

			for _, stem := range stems {
				if strings.HasPrefix(word, stem) {
					kinds = append(kinds, kind)

					break
				}
			}
		}
	}

	var fitting []string

	for _, ware := range wares {
		score := touches(words, ware.Name)*2 + touches(words, ware.Description)
		if slices.Contains(kinds, ware.Kind) {
			score += 2
		}

		if score == 0 || len(fitting) >= referrals {
			continue
		}

		fitting = append(fitting, fmt.Sprintf("«%s» %d (%s)", ware.Name, ware.Price, strings.TrimSuffix(ware.Description, ".")))
	}

	return fitting
}
