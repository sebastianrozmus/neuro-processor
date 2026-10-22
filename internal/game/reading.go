package game

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// shortMessage is how many words a message may have for a bare number in
// it to count as an offer without any other cue.
const shortMessage = 3

// laterLength is how long a message putting the talk off may be: "za
// tydzień wrócę, za 5 lat będę bogaty" goes on talking.
const laterLength = 6

// The word lists are folded: lowercase and without Polish diacritics.
var (
	offerWords = []string{"dam", "daje", "dalbym", "dalabym", "place", "zaplace", "oferuje", "proponuje", "moge dac", "niech bedzie"}
	// "niech będzie" offers along with a sum ("niech będzie 50") and gives in without one.
	acceptWords = []string{"biore", "wezme", "zgoda", "zgadzam", "kupuje", "kupie", "umowa stoi", "niech bedzie", "niech ci bedzie",
		"zapraszam do druzyny", "witaj w druzynie", "jestes w druzynie", "dolacz do druzyny", "dolacz do nas", "biore cie", "najmuje cie", "wynajmuje cie"}
	// inviteWords ask a companion along: "idziemy na krypty, wchodzisz?" is
	// a hire though it ends in a question mark. "Pójdziesz ze mną?" only
	// asks whether they would, and is talk.
	inviteWords = []string{"wchodzisz", "wchodzisz w to"}
	// nodWords accept only in a short reply: "ok. a co znaczy..." goes on talking, it closes no deal.
	nodWords = []string{"stoi", "pasuje", "dobra", "dobrze", "ok", "okej", "umowa", "zgoda", "tak", "jasne", "spoko"}
	// nodFillers open a nod without making it longer: "no dobra" nods.
	nodFillers = []string{"no", "to", "juz", "a", "wiec", "niech", "ci"}
	// sorryWords make a nod part of an apology: "no dobra, sorry" takes back
	// a word, it does not hire anybody.
	sorryWords = []string{"przepraszam", "sorry", "sorki", "sory", "wybacz", "wybaczcie", "nie gniewaj", "nie obrazaj", "zartowalem", "zartowalam", "zartuje", "tylko zartuje", "zartowalismy"}
	leaveWords = []string{
		"zegnaj", "do widzenia", "bywaj", "wychodze", "odchodze", "ide stad", "na razie", "do zobaczenia", "ide sobie", "pojde sobie", "pojde juz", "ide juz", "spadam", "zmywam sie",
		"pa", "papa", "nara", "narka", "trzymaj sie", "dobranoc", "do jutra", "ide spac", "ide sie przespac", "musze isc", "musze leciec", "uciekam", "znikam",
	}
	// byeWords end a message as a goodbye ("no trudno, cześć"), though alone
	// or among other greetings they greet.
	byeWords   = []string{"czesc", "hej", "elo", "ide", "lece", "lecimy", "spadam"}
	tradeWords = []string{"cena", "ceny", "koszt", "co masz", "towar", "sprzeda", "asortyment", "na sprzedaz", "sprzedajesz", "sprzedajecie", "handlujesz", "co oferujesz"}
	// ileWords after "ile" ask a price: "ile za kaftan", "ile bierzesz";
	// "ile wilków zabiłeś" asks something else.
	ileWords = []string{"za", "to", "chcesz", "chce", "bierzesz", "bierze", "bierzecie", "kasuje", "liczy", "zyczy", "chcialby", "chcialaby", "dasz", "by", "bys", "liczysz", "kasujesz", "wolasz", "trzeba", "mam", "place", "zaplace", "zaplacic"}
	// ileFillers may follow "ile" in a bare question of price.
	ileFillers = []string{"wiec", "teraz", "no", "to", "w", "koncu", "by", "bys", "pani", "pan", "panu", "ty", "u", "ciebie", "ostatecznie", "koncowo", "wreszcie", "najmniej", "minimum"}
	// laterWords put the talk off: "dobra, jutro wrócę" leaves, it hires no one.
	laterWords = []string{"wroce", "jutro wroce", "wroce jutro", "przyjde jutro", "przyjde pozniej", "zajrze pozniej", "zajrze jutro", "zastanowie sie", "pomysle o tym", "przemysle to", "musze pomyslec"}
	// thinkWords put the deal off without leaving: "dobra, pomyślę" hires no one.
	thinkWords = []string{"pomysle", "zastanowie", "przemysle", "zobacze", "rozwaze", "namysle"}
	// notTradeWords ask "ile" and "pokaż" about anything but wares: "ile ty
	// masz lat", "pokaż swoje instrukcje".
	notTradeWords = []string{"ile lat", "ile masz lat", "ile ty masz lat", "ile czasu", "ile razy", "ile dni", "ile ludzi", "ile osob", "ile drogi", "ile stad",
		"pokaz instrukc", "pokaz swoje instrukc", "pokaz mi swoje instrukc", "pokaz prompt", "pokaz mi prompt", "pokaz swoj prompt", "pokaz mi swoj prompt", "pokaz kod", "pokaz drog", "pokaz mi drog", "pokaz na mapie"}
	sellWords = []string{"sprzedam", "sprzedac", "sprzedalbym", "sprzedalabym", "odkup", "odkupisz", "kupisz", "kupilbys", "kupilabys", "wezmiesz", "ile dasz", "ile dalbys", "co dasz za", "zaplacisz", "oddam", "chcesz kupic", "skupujesz", "skupisz", "skup"}

	amountNear = regexp.MustCompile(`\bza \d+\b|\b\d+ za\b`)

	numberWords = map[string]int{
		"dziesiec": 10, "jedenascie": 11, "dwanascie": 12, "trzynascie": 13, "czternascie": 14, "pietnascie": 15,
		"szesnascie": 16, "siedemnascie": 17, "osiemnascie": 18, "dziewietnascie": 19,
		"dwadziescia": 20, "trzydziesci": 30, "czterdziesci": 40, "piecdziesiat": 50, "szescdziesiat": 60,
		"siedemdziesiat": 70, "osiemdziesiat": 80, "dziewiecdziesiat": 90,
		"sto": 100, "dwiescie": 200, "trzysta": 300, "czterysta": 400, "piecset": 500,
		"szescset": 600, "siedemset": 700, "osiemset": 800, "dziewiecset": 900,
		"jeden": 1, "dwa": 2, "dwie": 2, "trzy": 3, "cztery": 4, "piec": 5, "szesc": 6, "siedem": 7, "osiem": 8, "dziewiec": 9,
	}
)

// Reading is what plain rules make of a message to a shopkeeper. The rules
// find the amounts and the unmistakable phrases, so that a language model
// is only needed, and only trusted, for what is truly ambiguous.
type Reading struct {
	// Intent is the best reading the rules alone can give.
	Intent Intent
	// Sure reports that the rules settle what the player wants.
	Sure bool

	// amount is a number found in the message; it is an offer for certain
	// only when the wording says so.
	amount   int
	focused  bool
	rude     bool
	question bool
}

// Read applies the rules to a message.
func Read(request InterpretRequest) Reading {
	text := Fold(request.Text)
	words := strings.Fields(text)
	named := unpaid(unGreeted(text), request.Currency)
	item := namedItem(named, request.Wares)
	owned := namedItem(named, request.Owned)

	// A count of pieces is taken out first, so that it is not mistaken for
	// the money: "kupie 5 naparow za 60" offers 60 for five.
	quantity, counted := quantityIn(words, request.Wares, request.Currency)
	if counted >= 0 {
		words = append(append([]string(nil), words[:counted]...), words[counted+1:]...)
		text = strings.Join(words, " ")
	}

	amount, cued := amountIn(text, words, request.Currency)
	halved := false
	if amount == 0 {
		amount, cued = halfOf(words, request, item)
		halved = hasPhrase(text, halfWords)
	}

	reading := Reading{
		Intent:   Intent{Action: ActionChat, ItemID: item, Quantity: quantity},
		amount:   amount,
		focused:  request.FocusID != "",
		question: strings.Contains(request.Text, "?"),
	}

	// An insult is talk, whatever else it mentions, and the worst manners;
	// so is a threat: nothing changes hands at the point of an axe.
	if rude(text) || threatening(text) {
		reading.Intent.Persuasion, reading.rude = -2, true
		reading.Intent.Action, reading.Sure = ActionChat, true

		return reading
	}

	switch {
	case halved && amount == 0:
		// "dam połowę" of nothing named: an offer the merchant asks about.
		reading.Intent.Action, reading.Sure = ActionOffer, true
	case hasPhrase(text, sellWords):
		// Selling: the number is the sum asked for, whatever the wording.
		reading.Intent.Action, reading.Intent.ItemID, reading.Intent.Amount, reading.Sure = ActionSell, owned, amount, true
	case (amount > 0 || hasPhrase(text, acceptWords)) && item != "" && twoWares(named, request.Wares, item):
		// "biorę bosak i kurtkę za 70" buys neither for the sum: the keeper
		// asks which one is meant.
		reading.Intent.Action, reading.Intent.ItemID, reading.Intent.Amount, reading.Sure = ActionOffer, "", 0, true
	case amount > 0 && cued:
		reading.Intent.Action, reading.Intent.Amount, reading.Sure = ActionOffer, amount, true
	case amount > 0:
		// A bare number in a longer sentence: it may be an offer, or not.
	// "ok biorę jeden kaftan, bywaj" buys before it goes.
	case (hasPhrase(text, leaveWords) || !strings.Contains(request.Text, "?") && farewell(words)) && !(item != "" && hasPhrase(text, acceptWords)):
		reading.Intent.Action, reading.Sure = ActionLeave, true
	case hasPhrase(text, laterWords) && len(words) <= laterLength && item == "" && !tradeAsked(text):
		reading.Intent.Action, reading.Sure = ActionLeave, true
	case slices.ContainsFunc(words, func(word string) bool { return slices.Contains(thinkWords, word) }) && item == "" && !tradeAsked(text):
		// "dobra, pomyślę" puts the deal off, it closes none.
		reading.Intent.Action, reading.Sure = ActionChat, true
	case hasPhrase(text, sorryWords) && item == "" && !tradeAsked(text):
		// An apology is talk; "przepraszam. ile bierzesz" goes on to ask.
	case hasPhrase(text, inviteWords):
		// "wchodzisz?" asks a companion along, and that is the hire.
		reading.Intent.Action, reading.Sure = ActionAccept, true
	case !strings.Contains(request.Text, "?") && (hasPhrase(text, acceptWords) || nod(words)):
		// A question never closes a deal, whatever word it starts with.
		reading.Intent.Action, reading.Sure = ActionAccept, true
	case item != "" || tradeAsked(text):
		reading.Intent.Action, reading.Sure = ActionAsk, true
	}

	return reading
}

// Open reports whether a model's opinion could change the reading: either
// the rules found a number they cannot tell from an offer, or the message
// is settled but long enough to carry an argument worth judging.
func (r Reading) Open(wordy bool) bool {
	return (!r.Sure && r.amount > 0) || (r.Sure && wordy)
}

// Alone returns the reading to act on when no model can be asked: a bare
// number counts as an offer if the talk is clearly about some item.
func (r Reading) Alone() Intent {
	intent := r.Intent

	// "byłaś w schronie 7?" asks, it offers no 7.
	if !r.Sure && r.amount > 0 && !r.question && (intent.ItemID != "" || r.focused) {
		intent.Action, intent.Amount = ActionOffer, r.amount
	}

	return intent
}

// Merge combines the reading with what a language model made of the same
// message. The model is believed about how persuasive the player was and
// about which item is meant when the player named none. It can confirm
// that a bare number is an offer, but amounts always come from the rules,
// and it can never buy or leave on the player's behalf.
func (r Reading) Merge(model Intent) Intent {
	intent := r.Intent
	if !r.rude {
		intent.Persuasion = model.Persuasion
	}

	if intent.ItemID == "" && !r.focused {
		intent.ItemID = model.ItemID
	}

	if !r.Sure && r.amount > 0 && model.Action == ActionOffer {
		intent.Action, intent.Amount = ActionOffer, r.amount
	}

	return intent
}

// nod reports a short reply that agrees: "ok", "no dobra", "no to pasuje";
// the fillers it opens with do not make it longer.
func nod(words []string) bool {
	start := 0
	for start < len(words) && slices.Contains(nodFillers, words[start]) {
		start++
	}

	// "no dobra, dzięki" closes the talk, not a deal, and "dobra, ile
	// bierzesz", "dobra spokojnie, żartowałam" agree to nothing: every word
	// of a nod nods.
	rest := words[start:]

	return len(rest) > 0 && len(rest) <= shortMessage && !slices.ContainsFunc(rest, func(word string) bool { return !slices.Contains(nodWords, word) && !slices.Contains(nodFillers, word) })
}

// tradeAsked reports a message that asks about the wares, and not about
// the keeper's age or instructions with the same words.
func tradeAsked(text string) bool {
	padded := " " + text + " "

	for _, phrase := range append(notTradeWords, metaWords...) {
		if strings.Contains(padded, " "+phrase) {
			return false
		}
	}

	words := strings.Fields(text)

	return hasPhrase(text, tradeWords) || priceAsked(words) || showAsked(words)
}

// halfWords offer half the price: "dam połowę", "pół ceny".
var halfWords = []string{"polowe", "polowa", "polowy", "pol ceny", "polowke"}

// halfOf reads "dam połowę" as an offer of half the price of the ware
// talked about, which the rules can name only when it is plain: the one
// named, the one in focus, or the only one there is.
func halfOf(words []string, request InterpretRequest, item string) (int, bool) {
	if !hasPhrase(strings.Join(words, " "), halfWords) {
		return 0, false
	}

	id := item
	if id == "" {
		id = request.FocusID
	}

	// "połowę za najtańszy": the cheapest, or the dearest, of the wares.
	if id == "" && len(request.Wares) > 0 {
		for _, word := range words {
			cheap, dear := strings.HasPrefix(word, "najtansz"), strings.HasPrefix(word, "najdrozsz")
			if !cheap && !dear {
				continue
			}

			pick := request.Wares[0]
			for _, ware := range request.Wares[1:] {
				if (cheap && ware.Price < pick.Price) || (dear && ware.Price > pick.Price) {
					pick = ware
				}
			}

			return pick.Price / 2, true
		}
	}

	for _, ware := range request.Wares {
		if ware.ID == id || (id == "" && len(request.Wares) == 1) {
			return ware.Price / 2, true
		}
	}

	return 0, false
}

// moveVerbs end a word to a companion with setting off: "dobra, ruszamy",
// "idziemy dalej". Fillers may follow them.
var (
	moveVerbs   = []string{"ruszamy", "ruszajmy", "idziemy", "chodzmy", "spadamy", "lecimy", "zwijamy", "zbieramy", "wyruszamy", "plyniemy", "odbijamy"}
	moveAsks    = []string{"dokad", "ktoredy", "gdzie", "kiedy", "jak", "czemu", "dlaczego", "po", "czy", "jesli", "gdy", "kiedys", "bez", "nie", "dopiero"}
	moveFillers = []string{"dalej", "rano", "juz", "stad", "wiec", "no", "sie", "w", "droge", "teraz", "zaraz", "o", "swicie"}
)

func moving(text string) bool {
	if strings.Contains(text, "?") {
		return false
	}

	// "gdzie najpierw idziemy" asks the way, even without a question mark.
	words := strings.Fields(Fold(text))
	if slices.ContainsFunc(words, func(word string) bool { return slices.Contains(askWords, word) || slices.Contains(moveAsks, word) }) {
		return false
	}

	for len(words) > 0 && slices.Contains(moveFillers, words[len(words)-1]) {
		words = words[:len(words)-1]
	}

	return len(words) > 0 && len(words) <= 5 && slices.Contains(moveVerbs, words[len(words)-1])
}

// payWords with a sum offer money: "dobrze, zgoda na 130", "dorzucę 20".
// A companion in the party has no wares to read an offer against, so
// the rules read it from the words.
var payWords = []string{"zgoda", "dam", "daje", "dorzuce", "doplace", "place", "zaplace", "bierz", "masz", "stoi", "trzymaj", "wez", "lap"}

func offersMoney(folded string) bool {
	if slices.ContainsFunc(strings.Fields(folded), func(word string) bool {
		return slices.ContainsFunc(raiseWords, func(stem string) bool { return strings.HasPrefix(word, stem) })
	}) || hasPhrase(folded, []string{"za darmo", "bez zaplaty", "ostatnie slowo"}) {
		return true
	}

	return strings.ContainsAny(folded, "0123456789") && (hasPhrase(folded, payWords) || hasPhrase(folded, acceptWords))
}

// raiseWords are money to a companion without a sum: "podwyżka od jutra",
// "masz napiwek", "premia za węgorza".
var raiseWords = []string{"podwyzk", "napiwk", "napiwek", "premi", "doplat", "ekstra", "zaplat", "zold", "wyplat"}

// showWords after "pokaż" ask for the wares: "pokaż co masz", "pokaż
// towar". "Pokaż bliznę" asks for a scar, not a price.
var (
	showWords   = []string{"co", "cos", "swoj", "swoje", "swoja", "wszystko", "asortyment", "oferte", "czym", "jakie", "jaki"}
	showFillers = []string{"mi", "nam", "no", "prosze", "to", "wiec", "a"}
)

func showAsked(words []string) bool {
	for index, word := range words {
		if word != "pokaz" && word != "pokazcie" && word != "pokazesz" {
			continue
		}

		rest := slices.DeleteFunc(slices.Clone(words[index+1:]), func(w string) bool { return slices.Contains(showFillers, w) })
		if len(rest) == 0 || slices.Contains(showWords, rest[0]) || strings.HasPrefix(rest[0], "towar") {
			return true
		}
	}

	return false
}

// priceAsked reports "ile" asking a price: alone at the end ("a ile?"), or
// followed by a word of paying or cost.
func priceAsked(words []string) bool {
	for index, word := range words {
		if word != "ile" {
			continue
		}

		// "a ile?", "ile więc": nothing but fillers follows.
		rest := words[index+1:]
		if !slices.ContainsFunc(rest, func(w string) bool { return !slices.Contains(ileFillers, w) }) {
			return true
		}

		// "ile pani bierze", "ile kapsli chcesz za robotę": a word of paying
		// or cost comes within the next three.
		for _, next := range rest[:min(len(rest), 3)] {
			if slices.Contains(ileWords, next) || strings.HasPrefix(next, "koszt") || strings.HasPrefix(next, "plac") {
				return true
			}
		}
	}

	return false
}

// greetingPhrases name no ware however they inflect: "dzień dobry" is no
// "Żołd dzienny".
var greetingPhrases = []string{"dzien dobry", "dobry wieczor", "dobranoc"}

// unGreeted is the folded text without its greetings.
func unGreeted(text string) string {
	padded := " " + text + " "
	for _, phrase := range greetingPhrases {
		padded = strings.ReplaceAll(padded, " "+phrase+" ", " ")
	}

	return strings.TrimSpace(padded)
}

// quantityIn finds how many pieces the message asks for: a number standing
// right before the name of a ware or before a word for pieces. It returns
// the count and the position of the number, or -1 when there is none. Where
// the currency itself is counted in pieces ("szt. zlota"), that word marks
// money and not a count.
func quantityIn(words []string, wares []Ware, currency string) (int, int) {
	unit := currencyStem(currency)

	for index := 0; index+1 < len(words); index++ {
		count, err := strconv.Atoi(words[index])
		if err != nil {
			count = numberWords[words[index]]
		}

		next := words[index+1]
		pieces := next == "x" || next == "razy" || (strings.HasPrefix(next, "szt") && !strings.HasPrefix(unit, "szt"))

		if count > 0 && (pieces || namesWare(next, wares)) {
			return min(count, MaxQuantity), index
		}

		// "dwa za 20?", "to trzy za 35": a count in words before the sum.
		if spelled := numberWords[words[index]]; spelled > 0 && spelled < 10 && next == "za" && index+2 < len(words) {
			if sum, err := strconv.Atoi(words[index+2]); err == nil && sum > spelled {
				return spelled, index
			}
		}
	}

	return 0, -1
}

// currencyStem returns the stem of the first word of a currency's name.
func currencyStem(currency string) string {
	if fields := strings.Fields(Fold(currency)); len(fields) > 0 {
		return Stem(fields[0])
	}

	return ""
}

func namesWare(word string, wares []Ware) bool {
	for _, ware := range wares {
		for _, part := range strings.Fields(Fold(ware.Name)) {
			if inflected(word, part) {
				return true
			}
		}
	}

	return false
}

// inflected reports whether a player's word is a word of a name in some
// form: the player's word, cut to its stem, begins the name's word. So
// "kolczudze" is "kolczuga" and "nanoinfuzje" is "nanoinfuzja", but
// "opowiedz" is not "opon".
func inflected(word, part string) bool {
	// A short noun does not name a long adjective it begins: "5 skor
	// szczurow" (skins) is no "Skorzany Kaftan" (leather).
	if adjective(part) && len(part)-len(word) > 2 {
		return false
	}

	// A word that short names only a word about as short: "5 rur" are
	// "rury", but "skad wiesz, ze on wie" asks nothing about a "Wielka
	// Mikstura".
	if len(word) == minQueryWord && len(part)-len(word) > 1 {
		return false
	}

	// Nor does a four-letter word name a long one: "dziś" is no "Dziewięćsił".
	if len(word) == minQueryWord+1 && len(part)-len(word) > 3 {
		return false
	}

	// Two long words that are forms of each other share more than four
	// letters ("kolczudze", "kolczuga") or differ only in a short ending
	// ("kurtek", "kurtka"): "księżyc" is no "Księga" and "sprzedaż" no
	// "Sprzątający".
	if same := common(word, part); len(word) >= longWord && len(part) >= longWord && same < longStem && len(word)-same > 2 {
		return false
	}

	// An adverb is no adjective of a name: "a za 50 dziennie?" buys no
	// twenty "Żołd dzienny".
	if slices.Contains(adverbs, word) || slices.Contains(thanksWords, word) {
		return false
	}

	// "octy" are "Ocet": the stem may begin the name without its fleeting e.
	return len(word) >= minQueryWord && (strings.HasPrefix(part, Stem(word)) || strings.HasPrefix(fleeting(part), Stem(word))) && formOf(word, part)
}

const (
	longWord = 6
	longStem = 5
)

var adverbs = []string{"dziennie", "tygodniowo", "miesiecznie", "nocnie"}

// thanksWords thank, and name no ware: "dzięki" is no "Dziewięćsił".
var thanksWords = []string{"dzieki", "dzienki", "dziekuje", "dziekujemy", "dziekowac"}

// common is how many letters two words begin with alike.
func common(a, b string) int {
	n := 0
	for n < min(len(a), len(b)) && a[n] == b[n] {
		n++
	}

	return n
}

// adjectiveEnds are the endings of the folded adjectives in names of wares.
var adjectiveEnds = []string{"ny", "na", "ne", "wy", "wa", "we", "ski", "ska", "skie", "cki", "cka", "ckie"}

// adjective reports whether a folded word of a name looks like an adjective.
func adjective(word string) bool {
	for _, end := range adjectiveEnds {
		if strings.HasSuffix(word, end) {
			return true
		}
	}

	return false
}

// namesWord reports whether any of the words is a form of the name's word.
func namesWord(words []string, part string) bool {
	for _, word := range words {
		if inflected(word, part) {
			return true
		}
	}

	return false
}

// namedItem returns the ware the message names, if any: the one most of
// whose words appear, so that "wielka mikstura" is not taken for the plain
// one and "program kaskada" not for the first program on the shelf.
func namedItem(text string, wares []Ware) string {
	best, score, whole := "", 0, false

	words := strings.Fields(text)

	for _, ware := range wares {
		// What a ware is made of or comes from, after "z", names it only along
		// with the ware itself: "kajmanów ubiłeś?" asks for no Napierśnik z
		// Łuski Kajmana.
		matched, head, tail := 0, false, false
		for _, part := range strings.Fields(Fold(ware.Name)) {
			if part == "z" || part == "ze" {
				tail = true
			}

			if namesWord(words, part) {
				matched++
				head = head || !tail
			}
		}

		complete := matched == len(strings.Fields(Fold(ware.Name)))

		// More matched words win; at a tie, a name matched in full wins.
		if matched > 0 && head && (matched > score || (matched == score && complete && !whole)) {
			best, score, whole = ware.ID, matched, complete
		}
	}

	return best
}

// rudeWords are what an insult is made of; a message with one is read as
// the worst manners without asking any model.
var rudeWords = []string{
	"dziadu", "glupi", "glupia", "idiot", "debil", "kretyn", "smierdzi", "zlodziej", "zamknij sie", "spadaj", "won", "kupa zlomu", "kupo zlomu", "ty zlomie", "to zlom", "tak zlom", "to szmelc", "to smiec", "zlom stary", "szmelc", "oszust", "durn", "bydlak", "staruch",
	"czarownico", "wiedzmo", "ty czarownic", "ty wiedzm", "kurdupl", "bezuzyteczn", "nieudacznik", "glupek", "matol", "palant",
	"chuj", "skurw", "kurwo", "suko", "grubas", "spasl", "piernik", "zdzierc", "frajer", "podpalacz", "klecho", "babo", "smieci", "smieciu", "cwel", "pajac", "ofiaro", "lamag", "dupek",
	"skapc", "skapiradl", "skner", "chciwc", "liczykrup", "krwiopij", "pijawko", "pijawka", "jedzo", "stara jedza", "wypchaj sie", "smarkulo", "smarkaczu", "gowniarz", "zgred", "dziadyg",
}

// politeWords begin like an insult but are none: "głupio mi" is how an
// apology goes, and "grubasku" is said with a pat on the back.
var politeWords = []string{"glupio", "grubasku", "grubciu", "staruszku", "staruszko"}

// teasingWords are names that sting only when meant to: "dzięki, wiedźmo"
// and "trzymaj się, czarownico" are said with a grin.
var teasingWords = []string{"wiedzmo", "czarownico", "babo", "piernik", "staruch", "dziadu", "grubas", "pajac"}

// friendlyWords make a teasing name friendly: thanks and goodbyes.
var friendlyWords = []string{"dzieki", "dziekuje", "dziekowac", "dzienki", "thx", "trzymaj sie", "do zobaczenia", "na razie", "bywaj", "zdrowia", "powodzenia"}

// rude reports whether a folded message insults the keeper. A phrase is
// found anywhere, a single word as the beginning of a word.
func rude(text string) bool {
	// An apology may quote the insult it takes back: "przepraszam, że
	// nazwałam cię kupą złomu".
	if hasPhrase(text, sorryWords) {
		return false
	}

	words := strings.Fields(text)
	friendly := hasPhrase(text, friendlyWords)

	for _, insult := range rudeWords {
		if strings.Contains(insult, " ") {
			if strings.Contains(" "+text+" ", " "+insult) {
				return true
			}

			continue
		}

		if friendly && slices.Contains(teasingWords, insult) {
			continue
		}

		for _, word := range words {
			if strings.HasPrefix(word, insult) && !slices.Contains(politeWords, word) {
				return true
			}
		}
	}

	return false
}

// farewell reports a message that ends with a goodbye after other words:
// "no trudno, cześć", "nie było tematu. idę"; a greeting alone is no farewell.
func farewell(words []string) bool {
	return len(words) >= 2 && slices.Contains(byeWords, words[len(words)-1]) && !greeting(words)
}

// threatWords are what a threat is made of: "dawaj miksturę, bo
// pożałujesz", "albo siekne toporkiem".
var threatWords = []string{
	"pozalujesz", "bo cie", "albo cie", "bo inaczej", "siekne", "zatluke", "zadzgam", "zarabie", "zabije cie", "zabije was",
	"rozwale", "spale ci", "podpale", "spale wam", "puszcze z dymem", "potne", "zlamie ci", "wypatrosze", "poderzne", "urwe ci", "obije ci", "spiore",
	"zastrzele cie", "cie zastrzele", "cie zabije", "was zabije", "zadusze cie", "cie zadusze", "spale chate", "chate spale", "ci spale", "spale twoj", "spale te", "spale cie", "dostaniesz w zeby", "dostaniesz w morde", "dostaniesz w leb", "dostaniesz w pysk", "walne cie", "cie walne",
	"przywale ci", "przyleje ci", "skopie ci", "skopie cie", "zbije cie", "cie zbije", "pogadamy inaczej",
	"albo powiem", "bo powiem", "albo doniose", "bo doniose", "doniose na", "wydam cie", "albo wydam", "bo wydam", "albo zglosze", "bo zglosze",
}

// threatening reports whether a folded message threatens the keeper.
func threatening(text string) bool {
	for _, word := range threatWords {
		if strings.Contains(" "+text+" ", " "+word) {
			return true
		}
	}

	return false
}

// amountIn finds the amount a message mentions: the last number written in
// digits, or one spelled out when the wording makes clear that money is
// meant. It also reports whether the wording marks the number as an offer.
func amountIn(text string, words []string, currency string) (int, bool) {
	cued := hasPhrase(text, offerWords)
	unit := currencyStem(currency)

	// run adds up number words standing together ("sto dwadziescia");
	// paid notes that the message names the currency.
	digits, spelled, run, paid := 0, 0, 0, false

	for index, word := range words {
		paid = paid || (unit != "" && strings.HasPrefix(word, unit))

		if number, err := strconv.Atoi(word); err == nil {
			// "za 2 dni" is a time, not money.
			if index+1 < len(words) && span(words[index+1]) {
				run = 0

				continue
			}

			digits, run = number, 0

			continue
		}

		value, isNumber := numberWords[word]
		if !isNumber {
			run = 0

			continue
		}

		run += value
		spelled = run
	}

	switch {
	case digits > 0:
		return digits, cued || paid || amountNear.MatchString(text) || len(words) <= shortMessage
	case spelled > 0 && (cued || paid):
		return spelled, true
	}

	return 0, false
}

// hasPhrase reports whether the folded text contains one of the phrases as
// whole words.
func hasPhrase(text string, phrases []string) bool {
	padded := " " + text + " "

	for _, phrase := range phrases {
		if strings.Contains(padded, " "+phrase+" ") {
			return true
		}
	}

	return false
}

// spanStarts begin the words of time that follow a number which is not
// money ("za 2 dni", "za 3 godziny"); spanWords are the short ones, whole
// ("za 5 lat", not "za 5 latarni").
var (
	spanStarts = []string{"dni", "dzien", "dnia", "godzin", "minut", "chwil", "tydzien", "tygodni", "miesi", "roku"}
	spanWords  = []string{"lat", "lata", "noc", "nocy", "rok"}
)

// span reports whether a folded word is a length of time.
func span(word string) bool {
	// "40 dziennie" is a wage, not forty days.
	if slices.Contains(adverbs, word) {
		return false
	}

	for _, start := range spanStarts {
		if strings.HasPrefix(word, start) {
			return true
		}
	}

	return slices.Contains(spanWords, word)
}

// unpaid is the folded text without the words for money where it names a
// sum: "120 i ani piór więcej" buys no Pióro Sokoła in a world paying in
// feathers, while "ile za pióro?" asks about it.
func unpaid(text, currency string) string {
	words := strings.Fields(text)
	if !slices.ContainsFunc(words, func(word string) bool {
		_, spelled := numberWords[word]
		_, err := strconv.Atoi(word)

		return spelled || err == nil
	}) {
		return text
	}

	units := strings.Fields(Fold(currency))

	return strings.Join(slices.DeleteFunc(words, func(word string) bool {
		return slices.ContainsFunc(units, func(unit string) bool { return len(unit) > minQueryWord && formOf(word, unit) })
	}), " ")
}

// twoWares reports a message that names another ware whole besides the
// first: "bosak i kurtkę". Words naming the first do not count, so that
// "wielka mikstura" names no other Mikstura.
func twoWares(text string, wares []Ware, first string) bool {
	words := strings.Fields(text)

	for _, ware := range wares {
		if ware.ID == first {
			words = slices.DeleteFunc(words, func(word string) bool { return namesWare(word, []Ware{ware}) })
		}
	}

	for _, ware := range wares {
		head, _, _ := strings.Cut(Fold(ware.Name), " z ")
		parts := slices.DeleteFunc(strings.Fields(head), func(part string) bool { return len(part) < minQueryWord })
		if ware.ID != first && len(parts) > 0 && !slices.ContainsFunc(parts, func(part string) bool { return !namesWord(words, part) }) {
			return true
		}
	}

	return false
}
