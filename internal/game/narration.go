package game

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	narrateAttempts = 2
	// knownPrefix is how much of a spelled-out number must appear in what
	// the speaker knows for it to pass as a quotation.
	knownPrefix    = 6
	tradeSentences = 3
	chatSentences  = 4
)

// VerdictMark opens the note that tells a narrator what the rules decided.
const VerdictMark = "[GRA]"

// nameCorrection is what a narrator hears after mangling a name it had
// to pass on.
const nameCorrection = "Twoja poprzednia odpowiedź była zła: przekręciła albo pominęła imię osoby, do której odsyłasz gracza. " +
	"Napisz je dokładnie tak, bez odmiany: "

// correction is what a narrator hears after quoting a number it should not.
const correction = "Twoja poprzednia odpowiedź była zła: podała liczbę słownie albo taką, której nie ma w bloku [GRA]. " +
	"Napisz ją jeszcze raz. Liczby wolno ci podać tylko te z bloku [GRA] i tylko cyframi."

var (
	digits = regexp.MustCompile(`\d+`)
	markup = regexp.MustCompile(`<[^>]*>`)

	// spelledStems are folded pieces of Polish words for the numbers from
	// ten up, in any grammatical form ("szescdziesieciu", "dwunastu").
	spelledStems = []string{"dziesi", "dziest", "nasci", "nastu", "nasto", "tysia", "tysie"}
	spelledWords = []string{"sto", "stu", "setki", "setka", "dwiescie", "dwustu", "trzysta", "trzystu", "czterysta", "czterystu"}
)

// InterpretRequest is what is needed to read one player message.
type InterpretRequest struct {
	Text  string
	Wares []Ware
	// Owned are the player's own things, which they may want to sell.
	Owned   []Ware
	FocusID string
	History []Line
	// Currency is the phrase put after an amount, e.g. "kapsli".
	Currency string
}

// NarrateRequest is what a model needs to word the merchant's reply.
type NarrateRequest struct {
	Merchant Merchant
	Text     string
	Decision Decision
	History  []Line
	// Lore is what the merchant knows about the world and the customer,
	// so that there is something to talk about besides prices.
	Lore string
	// Hint is the piece of lore the player's words touch, or one worth
	// bringing up, so that a small model has something concrete to say.
	Hint string
	// Correction tells the narrator what was wrong with its last try.
	Correction string
	// Role, when set, replaces the opening line of the narrator's
	// instructions: who the speaker is instead of a shopkeeper.
	Role string
	// Forbidden are amounts the reply must not quote although the lore
	// holds them: a mercenary's fee in talk that is not about the hire.
	Forbidden []int
	// Required are names the reply must get right: whom the player is
	// sent to. A mangled name earns the model a second try.
	Required []string
	// Guarded are names that need not be mentioned but must be intact when
	// they are: the heroes of a fight the master describes.
	Guarded []string
	// Lead is the journal line of a referral in this reply ("Trop: …"),
	// for the screen to show; a reply with a lead teaches the world no
	// new fact.
	Lead string
}

// Interpreter turns free player text into a structured Intent.
type Interpreter interface {
	Interpret(ctx context.Context, request InterpretRequest) (Intent, error)
}

// Narrator words the merchant's reply to a decision already made.
type Narrator interface {
	Narrate(ctx context.Context, request NarrateRequest) (string, error)
}

// Streamer is a narrator that can hand its reply over piece by piece while
// it is still being written.
type Streamer interface {
	Stream(ctx context.Context, request NarrateRequest, deliver func(piece string)) (string, error)
}

// Brain is a language backend able to both interpret and narrate.
type Brain interface {
	Interpreter
	Narrator
}

// InterpretRequest builds the request for reading the given player message.
func (s *Session) InterpretRequest(text string) InterpretRequest {
	return InterpretRequest{Text: text, Wares: s.Wares(), Owned: s.OwnedWares(), FocusID: s.focus, History: s.History(), Currency: s.Merchant.Currency}
}

// NarrateRequest builds the request for wording the reply to a decision.
func (s *Session) NarrateRequest(text string, decision Decision) NarrateRequest {
	return NarrateRequest{Merchant: s.Merchant, Text: text, Decision: decision, History: s.History()}
}

// Sanitize makes an intent safe to apply whatever the model returned:
// unknown actions become small talk and unknown items are dropped.
func (r InterpretRequest) Sanitize(intent Intent) Intent {
	known := false
	for _, action := range Actions {
		known = known || action == intent.Action
	}

	if !known {
		intent.Action = ActionChat
	}

	known = false
	for _, ware := range append(append([]Ware(nil), r.Wares...), r.Owned...) {
		known = known || ware.ID == intent.ItemID
	}

	if !known {
		intent.ItemID = ""
	}

	intent.Persuasion = clamp(intent.Persuasion, -2, 2)

	return intent
}

// ErrInconsistent reports that a narrator kept quoting amounts the rules
// never produced.
var ErrInconsistent = errors.New("narrator contradicted the rules")

// Attempt is one line a model offered and why it was refused.
type Attempt struct {
	Text   string `json:"text"`
	Reason string `json:"reason"`
}

// Rejection is the error of a reply that was refused on every attempt:
// it says what the model wrote and what was wrong with it, so that the
// screen and the inspector can show why a canned line was spoken. It is
// an ErrInconsistent.
type Rejection struct {
	Attempts []Attempt
	// Kept is set when the first attempt was spoken anyway: a line that
	// only got a name wrong is still a better line than a canned one, and
	// the journal has the name right.
	Kept bool
}

func (r *Rejection) Error() string {
	reasons := make([]string, 0, len(r.Attempts))
	for _, attempt := range r.Attempts {
		reasons = append(reasons, attempt.Reason)
	}

	return "narrator contradicted the rules: " + strings.Join(reasons, "; ")
}

func (r *Rejection) Unwrap() error {
	return ErrInconsistent
}

// Reasons a line is refused for.
const (
	reasonEmpty   = "pusta odpowiedź"
	reasonLeak    = "cytuje notę gry"
	reasonNumbers = "liczba spoza bloku [GRA] albo napisana słownie"
	reasonNames   = "imię przekręcone albo pominięte"
)

// Narrate asks the narrator for a reply and checks it against the decision.
// The reply is cleaned up and cut to a few sentences. If it quotes amounts
// the rules never produced, the narrator is told so and asked again; if
// that fails too, or the narrator cannot be reached, a canned line is
// returned together with the reason.
func Narrate(ctx context.Context, narrator Narrator, request NarrateRequest) (string, error) {
	return NarrateLive(ctx, narrator, request, nil)
}

// NarrateLive is Narrate that, with a narrator able to stream and a
// function to deliver to, hands the first attempt over piece by piece as it
// is written. The pieces are raw: only the returned text has been checked.
func NarrateLive(ctx context.Context, narrator Narrator, request NarrateRequest, deliver func(piece string)) (string, error) {
	// What the merchant knows may hold numbers of its own ("Schron 7").
	known := request.Lore + " " + request.Merchant.Persona
	allowed := append(request.Decision.Numbers(), numbersIn(known)...)
	allowed = slices.DeleteFunc(allowed, func(number int) bool { return slices.Contains(request.Forbidden, number) })
	known = Fold(known)
	var (
		problem           error
		refused           []Attempt
		streamer, streams = narrator.(Streamer)
	)

	for attempt := range narrateAttempts {
		var (
			text string
			err  error
		)

		if streams && deliver != nil && attempt == 0 {
			text, err = streamer.Stream(ctx, request, deliver)
		} else {
			text, err = narrator.Narrate(ctx, request)
		}

		if err != nil {
			problem = err

			continue
		}

		// A reply that quotes the game's own note to the narrator is no line
		// of dialogue at all.
		leaked := strings.Contains(text, VerdictMark)
		text = tidy(text, request.Merchant.Name, request.Decision.sentences())

		reason := ""
		switch {
		case text == "":
			reason = reasonEmpty
		case leaked:
			reason = reasonLeak
		case !numbersOK(text, allowed, known):
			reason = reasonNumbers
		case !namesOK(text, request.Required) || !namesIntact(text, request.Guarded):
			reason = reasonNames
		default:
			return fitMood(text, request.Decision.Mood), nil
		}

		refused = append(refused, Attempt{Text: text, Reason: reason})
		request.Correction = correction
		if reason == reasonNames {
			request.Correction = nameCorrection + join(append(append([]string(nil), request.Required...), request.Guarded...)) + "."
		}
	}

	if problem == nil || len(refused) > 0 && errors.Is(problem, context.DeadlineExceeded) {
		rejection := &Rejection{Attempts: refused}
		if len(refused) > 0 && namesOnly(refused) {
			rejection.Kept = true

			return fitMood(refused[0].Text, request.Decision.Mood), rejection
		}

		problem = rejection
	}

	return fallbackFor(request), problem
}

// namesOnly reports attempts that were all refused for a name alone.
func namesOnly(attempts []Attempt) bool {
	for _, attempt := range attempts {
		if attempt.Reason != reasonNames {
			return false
		}
	}

	return true
}

// Noteworthy reports whether a keeper's line could hold a new fact worth
// asking the mind about: it names something, with a capital letter not
// at the start of a sentence. "Idę na pole z kolegami" names nothing,
// and a mind asked anyway tends to note it.
func Noteworthy(text string) bool {
	if _, rest, tagged := strings.Cut(text, "] "); tagged && strings.HasPrefix(text, "[") {
		text = rest
	}

	start := true

	for _, word := range strings.Fields(text) {
		runes := []rune(strings.TrimLeft(word, "„«\"("))
		if len(runes) > 1 && unicode.IsUpper(runes[0]) && !start {
			return true
		}

		start = strings.ContainsAny(string(runes[len(runes)-1:]), ".!?;:")
	}

	return false
}

// fallbackFor is the canned line when the model fails: the decision's, or
// for small talk one that fits who is speaking, since a mercenary does
// not "handluje".
func fallbackFor(request NarrateRequest) string {
	if request.Decision.Outcome == OutcomeChat {
		switch request.Role {
		case RoleMercenary:
			return "Nie mój interes. Mów o robocie albo o drodze."
		case RoleCompanion:
			return "Może i tak. Idziemy dalej?"
		case RoleWitness:
			return "Nie wiem, co ci na to powiedzieć."
		}
	}

	return request.Decision.Fallback()
}

// tidy strips what models like to wrap a line in (quotes, emphasis marks,
// markup, the speaker's name before or after it) and keeps its first
// sentences.
func tidy(text, speaker string, sentences int) string {
	text = markup.ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(strings.ReplaceAll(text, "*", "")), " ")
	text = strings.TrimPrefix(text, speaker+":")

	for _, dash := range []string{"-", "–", "—"} {
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(text, speaker)), dash))
	}

	text = strings.Trim(text, " \"„”«»")
	kept := 0

	for index, char := range text {
		if !strings.ContainsRune(".!?…", char) {
			continue
		}

		rest := text[index+len(string(char)):]
		if rest != "" && !strings.HasPrefix(rest, " ") {
			continue
		}

		if kept++; kept == sentences {
			return text[:index+len(string(char))]
		}
	}

	return text
}

// sentences is how long the merchant may talk about this decision.
func (d Decision) sentences() int {
	if d.Outcome == OutcomeChat || d.Outcome == OutcomeScene {
		return chatSentences
	}

	return tradeSentences
}

// namesOK reports whether every required name is in the text: its first
// word (the given name) in some case, so that "u Wędrownego Kupca" and
// "idź do Thorina" pass and "u Wędronowego Kupuca" does not. The rest of
// a name ("Żelazna Broda") a small model may drop; the journal has it.
func namesOK(text string, required []string) bool {
	words := strings.Fields(Fold(text))

	for _, name := range required {
		parts := strings.Fields(Fold(name))
		if len(parts) == 0 {
			continue
		}

		if len(parts[0]) >= minQueryWord && !nameOK(words, parts[0]) {
			return false
		}

		// A one-letter or two-letter given name is told by the rest.
		if len(parts[0]) < minQueryWord && len(parts) > 1 && !nameOK(words, parts[1]) {
			return false
		}
	}

	return true
}

// namesIntact reports that no guarded name is mangled in the text: a word
// that begins like the name ("sebascianrzmusa" for "sebastianrozmus") but
// is not the name in some case gives it away. Names need not appear.
func namesIntact(text string, guarded []string) bool {
	words := strings.Fields(Fold(text))

	for _, name := range guarded {
		parts := strings.Fields(Fold(name))
		if len(parts) == 0 || len(parts[0]) < 5 {
			continue
		}

		part := parts[0]
		for _, word := range words {
			word = strings.Trim(word, ".,!?;:")
			if strings.HasPrefix(word, part[:4]) && len(word) >= len(part)-2 && !nameOK([]string{word}, part) {
				return false
			}
		}
	}

	return true
}

// nameOK reports whether one word of a name is among the words in some
// case: a word starts with the name less its ending, allowing for a
// fleeting e ("Kupiec" → "Kupca", "Kowalec" → "Kowalca").
func nameOK(words []string, part string) bool {
	// A long name loses two letters to its ending, a short one only one:
	// "Thorin" must not pass as "Thoraina".
	cut := 1
	if len(part) >= 7 {
		cut = 2
	}

	stems := []string{part[:max(minQueryWord, len(part)-cut)]}

	switch fled := fleeting(part); {
	case strings.HasSuffix(part, "iec"):
		stems = append(stems, part[:len(part)-3]+"c")
	case fled != part:
		stems = append(stems, fled)
	}

	for _, word := range words {
		for _, stem := range stems {
			if strings.HasPrefix(word, stem) {
				return true
			}
		}
	}

	return false
}

// NumbersOK reports whether every number written in text is one of
// allowed. Numbers from ten up must be written in digits: spelled out they
// cannot be checked, so they are not accepted at all.
func NumbersOK(text string, allowed []int) bool {
	return numbersOK(text, allowed, "")
}

// numbersOK is NumbersOK that lets through spelled-out numbers which are
// part of what the speaker knows, given as folded text ("czterdziesci dni").
func numbersOK(text string, allowed []int, known string) bool {
	for _, word := range strings.Fields(Fold(text)) {
		if spelled(word) && (known == "" || !strings.Contains(known, word[:min(len(word), knownPrefix)])) {
			return false
		}
	}

	for _, match := range digits.FindAllString(text, -1) {
		number, err := strconv.Atoi(match)
		if err != nil {
			return false
		}

		found := false
		for _, value := range allowed {
			found = found || value == number
		}

		if !found {
			return false
		}
	}

	return true
}

// spelled reports whether a folded word is a number from ten up, in any
// grammatical form.
func spelled(word string) bool {
	if contains(spelledWords, word) || numberWords[word] >= 10 {
		return true
	}

	for _, stem := range spelledStems {
		if strings.Contains(word, stem) {
			return true
		}
	}

	return false
}

// numbersIn returns the numbers written in a text.
func numbersIn(text string) []int {
	var numbers []int

	for _, match := range digits.FindAllString(text, -1) {
		if number, err := strconv.Atoi(match); err == nil {
			numbers = append(numbers, number)
		}
	}

	return numbers
}

// Numbers lists the numbers the merchant may mention for this decision:
// the sums, and for several pieces also their count and the price of one.
func (d Decision) Numbers() []int {
	var numbers []int

	switch d.Outcome {
	case OutcomeSold, OutcomeCounter, OutcomeInsulted, OutcomeWalkedAway:
		numbers = []int{d.Offer, d.Price}
	case OutcomeNoGold:
		numbers = []int{d.Offer, d.Price, d.Gold}
	case OutcomeShort:
		numbers = []int{d.Price, 1}
	case OutcomeBought, OutcomeBid:
		numbers = []int{d.Offer, d.Price, d.Gold}
	case OutcomeInfo:
		numbers = append([]int{d.Price, d.Item.Price}, numbersIn(d.Item.Description)...)
	case OutcomeWares:
		for _, ware := range d.Wares {
			numbers = append(numbers, ware.Price)
		}
	}

	if d.Outcome == OutcomeScene {
		return numbersIn(d.Scene)
	}

	if d.Quantity > 1 && len(numbers) > 0 {
		numbers = append(numbers, d.Quantity, d.Item.Price, d.Price/d.Quantity, (d.Price+d.Quantity-1)/d.Quantity)
	}

	return numbers
}

// what names the goods the decision is about, with their count when there
// are several.
func (d Decision) what() string {
	if d.Quantity > 1 {
		return fmt.Sprintf("%d × «%s»", d.Quantity, d.Item.Name)
	}

	return "«" + d.Item.Name + "»"
}

// Brief describes the decision to the narrating model, in Polish.
// The opening sentences of the narrator's instructions for those who are
// not shopkeepers; empty Role in a NarrateRequest means a shopkeeper.
const (
	RoleMercenary = "Jesteś najemnikiem w karczmie osady w grze tekstowej: bohater przysiadł się do ciebie, żeby pogadać albo wynająć cię do drużyny. Niczym nie handlujesz poza własną służbą, a o jej cenie rozstrzyga gra."
	RoleCompanion = "Jesteś towarzyszem bohatera w grze tekstowej: siedzicie razem i rozmawiacie o drodze, walce i świecie. Niczym nie handlujesz."
	RoleWitness   = "Odgrywasz mieszkańca miasteczka w grze detektywistycznej: rozmawiasz z detektywem, który cię zagadnął. Niczym nie handlujesz."
)

// chatBriefs are the briefs of plain talk for those who are not
// shopkeepers. The shopkeeper's brief invites them to recommend wares
// with prices, which sends a mercenary straight back to his wage when he
// was asked about his axe.
var chatBriefs = map[string]string{
	RoleMercenary: "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział: o swojej broni, rzemiośle, drodze albo świecie. " +
		"O cenie służby mów tylko, gdy o nią pyta. W tej kwestii niczego nie sprzedajesz.",
	RoleCompanion: "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. " +
		"Jesteś już w drużynie: nie proponujesz najmu ani zapłaty i nie opisujesz czynów, których nie było; tylko rozmawiasz.",
	RoleWitness: "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz w swojej roli na to, co powiedział. " +
		"O sprawie mówisz tylko to, co masz w CO WIESZ; niczego nie dopowiadasz i niczym nie handlujesz.",
}

// BriefFor is the brief for somebody in a role: the same as Brief, except
// that plain talk is briefed by the role.
func (d Decision) BriefFor(role string) string {
	if brief, ok := chatBriefs[role]; ok && d.Outcome == OutcomeChat {
		return brief + " Nastrój twojej postaci: " + string(d.Mood) + "."
	}

	// A mercenary's deal is a hire, not a sale: told to "potwierdź
	// sprzedaż", models confirmed a sale and paid the player.
	if role == RoleMercenary && d.Outcome == OutcomeSold {
		brief := fmt.Sprintf("Gracz przyjmuje twoją cenę i płaci ci %s za %s. Umowa stoi: potwierdź, że dołączasz do drużyny. Pieniądze bierzesz ty, nie dajesz.", d.amount(d.Price), d.what())
		if d.Item.Name == WageName {
			// A legend's wage is settled at dawn: nothing changes hands now.
			brief = fmt.Sprintf("Gracz przyjmuje twoją stawkę %s %s za każdy dzień, płatne o świcie; nic z góry. Umowa stoi: potwierdź, że dołączasz do drużyny. Teraz nie bierzesz pieniędzy.", d.what(), d.amount(d.Price))
		}

		return brief + " Nastrój twojej postaci: " + string(d.Mood) + "."
	}

	return d.Brief()
}

func (d Decision) Brief() string {
	var brief string

	switch outcome := d.Outcome; {
	case outcome == OutcomeSold:
		brief = fmt.Sprintf("Gracz kupuje %s za %s. Transakcja zawarta, potwierdź sprzedaż.", d.what(), d.amount(d.Price))
	case outcome == OutcomeCounter:
		brief = fmt.Sprintf("Gracz zaproponował %s za %s. To za mało. Odrzuć ofertę i podaj swoją nową cenę: %s.", d.amount(d.Offer), d.what(), d.amount(d.Price))
	case outcome == OutcomeInsulted:
		brief = fmt.Sprintf("Gracz zaproponował śmiesznie niską kwotę %s za %s. To cię uraża. Nie obniżasz ceny, nadal żądasz %s.", d.amount(d.Offer), d.what(), d.amount(d.Price))
	case outcome == OutcomeNoGold:
		brief = fmt.Sprintf("Gracz chce kupić %s (twoja cena: %s), ale ma przy sobie tylko %d. Wytknij mu, że go nie stać. Niczego nie sprzedajesz.", d.what(), d.amount(d.Price), d.Gold)
	case outcome == OutcomeShort:
		brief = fmt.Sprintf("Gracz chce kupić %s, ale masz tylko 1 sztukę. Powiedz mu to. Za tę jedną chcesz %s.", d.what(), d.amount(d.Price))
	case outcome == OutcomeUnclear:
		brief = "Nie jest jasne, co i za ile gracz chce kupić. Poproś, żeby powiedział wprost, o który przedmiot chodzi i ile oferuje. Nie podawaj żadnych kwot."
	case outcome == OutcomeInfo:
		brief = fmt.Sprintf("Gracz pyta o %s. Opis: %s Twoja aktualna cena: %s. Opowiedz krótko o przedmiocie i podaj cenę.", d.what(), d.Item.Description, d.amount(d.Price))
	case outcome == OutcomeWares:
		brief = "Gracz pyta, co masz na sprzedaż. " + d.waresBrief()
	case outcome == OutcomeChat:
		brief = "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. " +
			"Jeśli pyta o towar, możesz polecić coś ze swojego towaru i podać jego cenę z listy. W tej kwestii niczego nie sprzedajesz."
	case outcome == OutcomeFarewell:
		brief = "Gracz się żegna. Pożegnaj go. Nie podawaj żadnych kwot."
	case outcome == OutcomeWalkedAway:
		brief = "Masz dość tej rozmowy. Kończysz ją i odprawiasz gracza. Niczego mu już nie sprzedasz. Nie podawaj żadnych kwot."
	case outcome == OutcomeBought:
		brief = fmt.Sprintf("Gracz sprzedaje ci %s za %s. Transakcja zawarta: płacisz i potwierdzasz.", d.what(), d.amount(d.Price))
	case outcome == OutcomeBid && d.Offer > 0:
		brief = fmt.Sprintf("Gracz chce ci sprzedać %s i żąda %s. To za dużo. Odmów i zaoferuj swoją cenę: %s.", d.what(), d.amount(d.Offer), d.amount(d.Price))
	case outcome == OutcomeBid:
		brief = fmt.Sprintf("Gracz chce ci sprzedać %s. Obejrzyj rzecz i zaoferuj za nią %s; więcej nie dasz.", d.what(), d.amount(d.Price))
	case outcome == OutcomeUnwanted:
		brief = fmt.Sprintf("Gracz chce ci sprzedać %s, ale ty takimi rzeczami nie handlujesz. Odmów i powiedz, czym handlujesz. Nie podawaj żadnych kwot.", d.what())
	case outcome == OutcomeScene:
		return "Przebieg rundy: " + d.Scene
	}

	if d.Quantity > 1 && d.Outcome != OutcomeShort {
		brief += fmt.Sprintf(" Kwoty dotyczą wszystkich %d sztuk razem; pojedyncza sztuka kosztuje u ciebie %d, hurtem jest taniej.", d.Quantity, d.Item.Price)
	}

	return brief + " Nastrój twojej postaci: " + string(d.Mood) + "."
}

// Fallback is the merchant's reply used when no model can provide one.
func (d Decision) Fallback() string {
	switch d.Outcome {
	case OutcomeSold:
		return fmt.Sprintf("Niech będzie. %s za %s i jesteśmy kwita.", d.what(), d.amount(d.Price))
	case OutcomeCounter:
		return fmt.Sprintf("%d? Za mało. Mogę zejść do %s za %s i ani trochę niżej.", d.Offer, d.amount(d.Price), d.what())
	case OutcomeInsulted:
		return fmt.Sprintf("%s? Nie obrażaj mnie. Cena to nadal %d.", d.amount(d.Offer), d.Price)
	case OutcomeNoGold:
		return fmt.Sprintf("Masz przy sobie %s, a %s kosztuje %d. Wróć, gdy będzie cię stać.", d.amount(d.Gold), d.what(), d.Price)
	case OutcomeShort:
		return fmt.Sprintf("«%s» mam tylko 1 sztukę. Za nią chcę %s.", d.Item.Name, d.amount(d.Price))
	case OutcomeUnclear:
		return "Mów jaśniej: który przedmiot i ile za niego dajesz?"
	case OutcomeInfo:
		return fmt.Sprintf("%s. %s Cena: %s.", d.what(), d.Item.Description, d.amount(d.Price))
	case OutcomeWares:
		return d.waresBrief()
	case OutcomeFarewell:
		return "Bywaj. Wiesz, gdzie mnie szukać."
	case OutcomeWalkedAway:
		return "Dość tego. Nie będę tracić na ciebie czasu, zabieraj się stąd."
	case OutcomeBought:
		return fmt.Sprintf("Zgoda. %s za %s, masz tu swoje pieniądze.", d.what(), d.amount(d.Price))
	case OutcomeBid:
		if d.Offer > 0 {
			return fmt.Sprintf("%d? Nie te czasy. Za %s dam %s i to moje ostatnie słowo.", d.Offer, d.what(), d.amount(d.Price))
		}

		return fmt.Sprintf("Pokaż. Hm, za %s mogę dać %s.", d.what(), d.amount(d.Price))
	case OutcomeUnwanted:
		return fmt.Sprintf("%s? Nie skupuję takich rzeczy. Poszukaj kogoś, kto się na tym zna.", d.what())
	case OutcomeScene:
		// A round needs no canned narration: the rules' own lines are shown anyway.
		return ""
	}

	return "Ciekawe rzeczy opowiadasz, ale ja tu handluję."
}

// amount words a sum in the world's currency.
func (d Decision) amount(value int) string {
	return fmt.Sprintf("%d %s", value, d.Currency)
}

func (d Decision) waresBrief() string {
	if len(d.Wares) == 0 {
		return "Nie masz już nic na sprzedaż."
	}

	wares := make([]string, 0, len(d.Wares))
	for _, ware := range d.Wares {
		wares = append(wares, fmt.Sprintf("«%s» za %s", ware.Name, d.amount(ware.Price)))
	}

	return "Na sprzedaż: " + strings.Join(wares, ", ") + "."
}
