package game

import (
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Verb is what a typed command asks for.
type Verb int

const (
	VerbUnknown Verb = iota
	VerbGo
	VerbLook
	VerbTake
	VerbDrop
	VerbUse
	VerbAttack
	VerbCast
	VerbHeal
	VerbFlee
	VerbRest
	VerbVisit
	VerbHire
	VerbDismiss
	VerbGive
	VerbTakeBack
	VerbShow
	VerbAuto
	VerbSay
	VerbOpen
	VerbJournal
	VerbQuest
	VerbAccept
	VerbBag
	VerbBestiary
	VerbSave
	VerbHelp
	VerbQuit
)

// Help lists the commands the game understands.
const Help = "Polecenia: idź na północ / południe / wschód / zachód (albo n, s, e, w) · rozejrzyj się · podnieś X · upuść X · " +
	"użyj X (załóż, wypij, naucz się) · atakuj X · rzuć X na Y · ulecz (najsilniejsze leczenie, jakie znasz) · uciekaj · odpocznij · idź do X (budynek w osadzie; w środku także «sprzedam X za N») · najmij X · odpraw X · " +
	"daj X Y (rzecz towarzyszowi) · odbierz X od Y · pokaż Y (karta towarzysza, także F1-F3) · autowalka · " +
	"Imię, ulecz / rzuć X / atakuj X / wypij X (rozkaz dla towarzysza, np. «Lyrasa, ulecz załogę») · " +
	"powiedz X · plecak · bestiariusz (także F4) · dziennik (co wiesz) · zlecenia · przyjmij (zlecenie kupca, u którego jesteś) · handluj z karawaną (gdy stoi obok) · zapisz · pomoc · wyjdź. Strzałki też chodzą, Tab otwiera plecak, a pusty Enter powtarza ostatnie polecenie."

// Command is a parsed line typed by a player.
type Command struct {
	Verb Verb
	// Dir is the step to take for VerbGo; zero when no direction was given.
	Dir world.Point
	// Arg names what the verb applies to, in the player's words.
	Arg string
	// Target names whom a spell is cast at.
	Target string
	// Index is the monster picked in the interface, used when no name is given.
	Index int
}

type verbWord struct {
	stem string
	verb Verb
}

// verbWords maps the folded beginnings of words to verbs. Stems of one or
// two letters must match the whole word.
var verbWords = []verbWord{
	{"rozejrz", VerbLook}, {"rozglad", VerbLook}, {"patrz", VerbLook}, {"spojrz", VerbLook}, {"opisz", VerbLook}, {"obejrz", VerbLook}, {"zbadaj", VerbLook}, {"l", VerbLook},
	{"podnies", VerbTake}, {"wez", VerbTake}, {"zabierz", VerbTake}, {"zbierz", VerbTake}, {"bierz", VerbTake}, {"lap", VerbTake},
	{"upusc", VerbDrop}, {"wyrzuc", VerbDrop}, {"zostaw", VerbDrop}, {"odloz", VerbDrop}, {"porzuc", VerbDrop},
	{"uzyj", VerbUse}, {"wypij", VerbUse}, {"zaloz", VerbUse}, {"ubierz", VerbUse}, {"dobadz", VerbUse}, {"zjedz", VerbUse}, {"naucz", VerbUse}, {"przeczytaj", VerbUse}, {"wyposaz", VerbUse},
	{"atak", VerbAttack}, {"bij", VerbAttack}, {"walcz", VerbAttack}, {"uderz", VerbAttack}, {"zabij", VerbAttack}, {"tnij", VerbAttack}, {"a", VerbAttack},
	{"rzuc", VerbCast}, {"czaruj", VerbCast}, {"zaklecie", VerbCast}, {"wyczaruj", VerbCast},
	{"ulecz", VerbHeal}, {"wylecz", VerbHeal}, {"lecz", VerbHeal}, {"uzdrow", VerbHeal}, {"opatrz", VerbHeal}, {"uleczyc", VerbHeal},
	{"uciek", VerbFlee}, {"wiej", VerbFlee}, {"zwiewaj", VerbFlee}, {"ucieczka", VerbFlee},
	{"odpocz", VerbRest}, {"spij", VerbRest}, {"przespij", VerbRest}, {"nocuj", VerbRest}, {"karczma", VerbRest},
	{"rozmawiaj", VerbVisit}, {"porozmawiaj", VerbVisit}, {"pogadaj", VerbVisit}, {"handluj", VerbVisit}, {"kup", VerbVisit}, {"targuj", VerbVisit}, {"wejdz", VerbVisit}, {"odwiedz", VerbVisit},
	{"najmij", VerbHire}, {"najem", VerbHire}, {"zwerbuj", VerbHire}, {"werbuj", VerbHire}, {"zatrudnij", VerbHire}, {"wynajmij", VerbHire},
	{"odpraw", VerbDismiss}, {"zwolnij", VerbDismiss}, {"rozstan", VerbDismiss},
	{"daj", VerbGive}, {"przekaz", VerbGive}, {"podaruj", VerbGive},
	{"odbierz", VerbTakeBack}, {"zabierz od", VerbTakeBack},
	{"pokaz", VerbShow}, {"karta", VerbShow},
	{"autowalka", VerbAuto}, {"automat", VerbAuto},
	{"powiedz", VerbSay}, {"mow", VerbSay}, {"krzyknij", VerbSay},
	{"ekwipunek", VerbBag}, {"plecak", VerbBag}, {"inwentarz", VerbBag}, {"ekw", VerbBag}, {"i", VerbBag},
	{"dziennik", VerbJournal}, {"notatki", VerbJournal}, {"wiem", VerbJournal}, {"plotki", VerbJournal},
	{"zlecen", VerbQuest}, {"zadani", VerbQuest}, {"misje", VerbQuest}, {"questy", VerbQuest},
	{"przyjm", VerbAccept}, {"podejm", VerbAccept}, {"biore", VerbAccept}, {"zgadzam", VerbAccept},
	{"otworz", VerbOpen}, {"otwieraj", VerbOpen}, {"odkrec", VerbOpen}, {"wylam", VerbOpen}, {"odsun", VerbOpen}, {"podwaz", VerbOpen},
	{"bestiariusz", VerbBestiary}, {"bestiarz", VerbBestiary}, {"bestie", VerbBestiary}, {"potwory", VerbBestiary},
	{"zapisz", VerbSave}, {"save", VerbSave},
	{"pomoc", VerbHelp}, {"help", VerbHelp}, {"komendy", VerbHelp}, {"polecenia", VerbHelp},
	{"wyjdz", VerbQuit}, {"koniec", VerbQuit}, {"zakoncz", VerbQuit}, {"quit", VerbQuit}, {"exit", VerbQuit}, {"q", VerbQuit},
}

var (
	walkWords = []string{"idz", "pojdz", "ide", "rusz", "biegnij", "go"}
	// fillers are dropped from the start of what a verb applies to.
	fillers = []string{"z", "ze", "do", "u", "na", "w", "sie", "to", "ten", "te"}

	north = world.Point{Y: -1}
	south = world.Point{Y: 1}
	east  = world.Point{X: 1}
	west  = world.Point{X: -1}
)

// Parse reads a line typed by a player.
func Parse(text string) Command {
	if strings.TrimSpace(text) == "?" {
		return Command{Verb: VerbHelp}
	}

	words := strings.Fields(Fold(text))
	if len(words) == 0 {
		return Command{}
	}

	first, rest := words[0], words[1:]

	if contains(walkWords, first) {
		for _, word := range rest {
			if dir, ok := direction(word); ok {
				return Command{Verb: VerbGo, Dir: dir}
			}
		}

		if arg := argument(rest); arg != "" {
			return Command{Verb: VerbVisit, Arg: arg}
		}

		return Command{Verb: VerbGo}
	}

	if first == "na" && len(rest) > 0 {
		first, rest = rest[0], rest[1:]
	}

	if dir, ok := direction(first); ok && len(rest) == 0 {
		return Command{Verb: VerbGo, Dir: dir}
	}

	switch verb := verbOf(first); verb {
	case VerbUnknown:
		// A bare name may still be a building to visit.
		return Command{Arg: strings.Join(words, " ")}
	case VerbSay:
		_, said, _ := strings.Cut(strings.TrimSpace(text), " ")

		return Command{Verb: verb, Arg: strings.TrimSpace(said)}
	case VerbCast:
		spell, target, _ := strings.Cut(" "+strings.Join(rest, " ")+" ", " na ")

		return Command{Verb: verb, Arg: argument(strings.Fields(spell)), Target: strings.TrimSpace(target)}
	default:
		return Command{Verb: verb, Arg: argument(rest)}
	}
}

func verbOf(word string) Verb {
	for _, known := range verbWords {
		if word == known.stem || (len(known.stem) > 2 && strings.HasPrefix(word, known.stem)) {
			return known.verb
		}
	}

	return VerbUnknown
}

func direction(word string) (world.Point, bool) {
	switch {
	case word == "n" || word == "pn" || strings.HasPrefix(word, "poln"):
		return north, true
	case word == "s" || word == "pd" || strings.HasPrefix(word, "polud"):
		return south, true
	case word == "e" || strings.HasPrefix(word, "wsch"):
		return east, true
	case word == "w" || strings.HasPrefix(word, "zach"):
		return west, true
	}

	return world.Point{}, false
}

// argument joins the words a verb applies to, without leading fillers.
func argument(words []string) string {
	for len(words) > 0 && contains(fillers, words[0]) {
		words = words[1:]
	}

	return strings.Join(words, " ")
}

func contains(words []string, word string) bool {
	for _, known := range words {
		if known == word {
			return true
		}
	}

	return false
}
