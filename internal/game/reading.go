package game

import (
	"regexp"
	"strconv"
	"strings"
)

// shortMessage is how many words a message may have for a bare number in
// it to count as an offer without any other cue.
const shortMessage = 3

// The word lists are folded: lowercase and without Polish diacritics.
var (
	offerWords  = []string{"dam", "daje", "dalbym", "dalabym", "place", "zaplace", "oferuje", "proponuje", "moge dac", "niech bedzie"}
	acceptWords = []string{"biore", "wezme", "zgoda", "zgadzam", "kupuje", "kupie", "umowa stoi", "stoi", "pasuje", "dobra", "ok"}
	leaveWords  = []string{"zegnaj", "do widzenia", "bywaj", "wychodze", "odchodze", "ide stad", "na razie", "do zobaczenia"}
	tradeWords  = []string{"ile", "cena", "ceny", "koszt", "co masz", "pokaz", "towar", "sprzeda", "asortyment", "na sprzedaz"}
	sellWords   = []string{"sprzedam", "sprzedac", "sprzedalbym", "sprzedalabym", "odkup", "odkupisz", "kupisz", "kupilbys", "kupilabys", "wezmiesz", "ile dasz", "ile dalbys", "co dasz za", "zaplacisz", "oddam", "chcesz kupic", "skupujesz", "skupisz", "skup"}

	amountNear = regexp.MustCompile(`\bza \d+\b|\b\d+ za\b`)

	numberWords = map[string]int{
		"dziesiec": 10, "jedenascie": 11, "dwanascie": 12, "trzynascie": 13, "czternascie": 14, "pietnascie": 15,
		"szesnascie": 16, "siedemnascie": 17, "osiemnascie": 18, "dziewietnascie": 19,
		"dwadziescia": 20, "trzydziesci": 30, "czterdziesci": 40, "piecdziesiat": 50, "szescdziesiat": 60,
		"siedemdziesiat": 70, "osiemdziesiat": 80, "dziewiecdziesiat": 90,
		"sto": 100, "dwiescie": 200, "trzysta": 300, "czterysta": 400, "piecset": 500,
		"szescset": 600, "siedemset": 700, "osiemset": 800, "dziewiecset": 900,
		"jeden": 1, "dwa": 2, "trzy": 3, "cztery": 4, "piec": 5, "szesc": 6, "siedem": 7, "osiem": 8, "dziewiec": 9,
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
	amount  int
	focused bool
	rude    bool
}

// Read applies the rules to a message.
func Read(request InterpretRequest) Reading {
	text := Fold(request.Text)
	words := strings.Fields(text)
	item := namedItem(text, request.Wares)
	owned := namedItem(text, request.Owned)

	// A count of pieces is taken out first, so that it is not mistaken for
	// the money: "kupie 5 naparow za 60" offers 60 for five.
	quantity, counted := quantityIn(words, request.Wares, request.Currency)
	if counted >= 0 {
		words = append(append([]string(nil), words[:counted]...), words[counted+1:]...)
		text = strings.Join(words, " ")
	}

	amount, cued := amountIn(text, words, request.Currency)

	reading := Reading{
		Intent:  Intent{Action: ActionChat, ItemID: item, Quantity: quantity},
		amount:  amount,
		focused: request.FocusID != "",
	}

	// An insult is talk, whatever else it mentions, and the worst manners.
	if rude(text) {
		reading.Intent.Persuasion, reading.rude = -2, true
		reading.Intent.Action, reading.Sure = ActionChat, true

		return reading
	}

	switch {
	case hasPhrase(text, sellWords):
		// Selling: the number is the sum asked for, whatever the wording.
		reading.Intent.Action, reading.Intent.ItemID, reading.Intent.Amount, reading.Sure = ActionSell, owned, amount, true
	case amount > 0 && cued:
		reading.Intent.Action, reading.Intent.Amount, reading.Sure = ActionOffer, amount, true
	case amount > 0:
		// A bare number in a longer sentence: it may be an offer, or not.
	case hasPhrase(text, leaveWords):
		reading.Intent.Action, reading.Sure = ActionLeave, true
	case hasPhrase(text, acceptWords):
		reading.Intent.Action, reading.Sure = ActionAccept, true
	case item != "" || hasPhrase(text, tradeWords):
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

	if !r.Sure && r.amount > 0 && (intent.ItemID != "" || r.focused) {
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
	return len(word) >= minQueryWord && strings.HasPrefix(part, Stem(word))
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
		matched := 0
		for _, part := range strings.Fields(Fold(ware.Name)) {
			if namesWord(words, part) {
				matched++
			}
		}

		complete := matched == len(strings.Fields(Fold(ware.Name)))

		// More matched words win; at a tie, a name matched in full wins.
		if matched > 0 && (matched > score || (matched == score && complete && !whole)) {
			best, score, whole = ware.ID, matched, complete
		}
	}

	return best
}

// rudeWords are what an insult is made of; a message with one is read as
// the worst manners without asking any model.
var rudeWords = []string{"dziadu", "glupi", "glupia", "idiot", "debil", "kretyn", "smierdzi", "zlodziej", "zamknij sie", "spadaj", "won", "zlom", "szmelc", "oszust", "durn", "bydlak"}

// rude reports whether a folded message insults the keeper.
func rude(text string) bool {
	for _, word := range rudeWords {
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

	for _, word := range words {
		paid = paid || (unit != "" && strings.HasPrefix(word, unit))

		if number, err := strconv.Atoi(word); err == nil {
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
