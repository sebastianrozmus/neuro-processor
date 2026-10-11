// Package llm provides the language backends: an OpenAI-compatible HTTP
// client that works with cloud APIs and local servers alike, and an offline
// rule-based stand-in.
package llm

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const interpretInstructions = `Jesteś parserem w grze tekstowej. Gracz rozmawia z kupcem. ` +
	`Twoim jedynym zadaniem jest zamienić wypowiedź gracza na obiekt JSON zgodny ze schematem. Nie odpowiadasz graczowi.

Pola:
- action: "offer" gdy gracz proponuje konkretną kwotę za przedmiot; "accept" gdy zgadza się na cenę kupca; ` +
	`"ask" gdy pyta o przedmiot albo o jego cenę; "sell" gdy chce sprzedać kupcowi własną rzecz albo pyta, ile kupiec za nią da; ` +
	`"leave" gdy się żegna; "chat" gdy mówi o czymkolwiek innym ` +
	`(o świecie, o ludziach, o sobie), także wtedy, gdy zadaje pytanie niezwiązane z towarem.
- item_id: identyfikator przedmiotu, o który chodzi (z towaru kupca albo, przy "sell", z rzeczy gracza), albo "" gdy gracz żadnego nie ma na myśli. ` +
	`Jeśli gracz opisuje przedmiot własnymi słowami, wybierz ten, który pasuje.
- amount: kwota proponowana albo żądana przez gracza jako liczba całkowita; 0 gdy żadnej nie podał.
- persuasion: jak przekonująca i uprzejma jest wypowiedź: 2 świetny argument, 1 dobry, 0 neutralna, -1 niegrzeczna, -2 obraźliwa.`

const narrateInstructions = `Odgrywasz mieszkańca osady w grze tekstowej: prowadzisz sklep i rozmawiasz z bohaterem, który do ciebie przyszedł.

TWOJA POSTAĆ: %s

CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):
%s

JAK MÓWISZ:
- Po polsku, wyłącznie kwestią postaci: bez didaskaliów, bez cudzysłowów, bez imienia na początku.
- Masz charakter. Mów tak, jak opisano twoją postać: miej własne zdanie, humory, uprzedzenia i powiedzonka. Nie bądź uprzejmym automatem.
- Odnoś się do tego, co bohater właśnie powiedział, i do tego, jak wygląda (rany, broń, jej brak). Nie powtarzaj swoich wcześniejszych kwestii ani powitania.
- Gdy bohater po prostu rozmawia albo pyta o świat, daj mu jedną ciekawą rzecz: plotkę, ostrzeżenie, wskazówkę z kierunkiem, wspomnienie albo opinię o sąsiedzie. ` +
	`Mów konkretnie i tylko o tym, co wiesz. Nie wymyślaj nowych miejsc, osób ani przedmiotów.
- Mów krótko jak człowiek przy ladzie: najwyżej 3 zdania, zwykle 2. Żadnych monologów.

ZASADY GRY:
- O wyniku targowania decyduje gra, nie ty. Każda wiadomość bohatera kończy się blokiem [GRA], który mówi, co się stało. ` +
	`Twoja kwestia musi być z nim zgodna. Bloku [GRA] nie cytujesz i o nim nie wspominasz.
- Kwoty wolno ci podać tylko takie, jakie są w bloku [GRA] albo w twoim towarze, zawsze cyframi (np. 40, nie "czterdzieści"). Nie wymyślasz cen ani rabatów.
- Piszesz samą kwestię: bez gwiazdek, bez opisów gestów, bez podpisu.
- Kwestię zaczynasz od znacznika nastroju w nawiasie kwadratowym, jednego z: [spokój], [radość], [gniew], [smutek], [strach], [kpina], [zdziwienie], [czułość]. ` +
	`Na przykład: [kpina] Tyle za rurę? Potem już tylko mowa.
- Nigdy nie wychodzisz z roli.`

func interpretPrompt(request game.InterpretRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Przedmioty na sprzedaż (item_id: nazwa, aktualna cena kupca):\n")

	for _, ware := range request.Wares {
		fmt.Fprintf(&prompt, "- %s: %s, %d\n", ware.ID, ware.Name, ware.Price)
	}

	if len(request.Owned) > 0 {
		prompt.WriteString("\nRzeczy gracza, które mógłby sprzedać (item_id: nazwa, ile kupiec by dał):\n")

		for _, ware := range request.Owned {
			fmt.Fprintf(&prompt, "- %s: %s, %d\n", ware.ID, ware.Name, ware.Price)
		}
	}

	if request.FocusID != "" {
		fmt.Fprintf(&prompt, "\nRozmowa toczy się teraz o: %s\n", request.FocusID)
	}

	if len(request.History) > 0 {
		prompt.WriteString("\nOstatnie kwestie:\n")

		for _, line := range request.History {
			speaker := "Kupiec"
			if line.Player {
				speaker = "Gracz"
			}

			fmt.Fprintf(&prompt, "%s: %s\n", speaker, line.Text)
		}
	}

	fmt.Fprintf(&prompt, "\nWypowiedź gracza do sparsowania:\n%s", request.Text)

	return prompt.String()
}

// Transcript returns the conversation a narrate request becomes, as the
// model sees it, with the reply appended. Training data recorded this way
// has exactly the shape of the prompts the game sends.
func Transcript(request game.NarrateRequest, reply string) []map[string]string {
	messages := narrateMessages(request)
	transcript := make([]map[string]string, 0, len(messages)+1)

	for _, item := range messages {
		transcript = append(transcript, map[string]string{"role": item.Role, "content": item.Content})
	}

	return append(transcript, map[string]string{"role": "assistant", "content": reply})
}

const masterInstructions = `Jesteś Mistrzem Gry w tekstowej grze fabularnej. Zasady rozstrzygnęły już rundę walki; ty ją opisujesz graczom.

TWOJA POSTAĆ: %s

STAN WALKI:
%s

JAK OPISUJESZ:
- Po polsku, najwyżej 3 krótkie zdania, bez didaskaliów w gwiazdkach, bez nagłówków i bez list.
- Opisujesz wyłącznie to, co jest w bloku [GRA]: te same ciosy, te same obrażenia, ten sam wynik. Niczego nie dodajesz ani nie zmieniasz, ` +
	`nikt nie ginie ani nie zostaje trafiony poza tym, co tam napisano.
- Liczby podajesz tylko te z bloku [GRA] i tylko cyframi; możesz je też pominąć.
- Imiona bohaterów i nazwy przeciwników piszesz tak, jak w stanie walki. Bloku [GRA] nie cytujesz i o nim nie wspominasz.`

func narrateMessages(request game.NarrateRequest) []message {
	instructions := narrateInstructions
	if request.Role != "" {
		instructions = strings.Replace(instructions, "Odgrywasz mieszkańca osady w grze tekstowej: prowadzisz sklep i rozmawiasz z bohaterem, który do ciebie przyszedł.", request.Role, 1)
	}
	if request.Decision.Outcome == game.OutcomeScene {
		instructions = masterInstructions
	}

	messages := []message{{Role: "system", Content: fmt.Sprintf(instructions, request.Merchant.Persona, request.Lore)}}

	for _, line := range request.History {
		role := "assistant"
		if line.Player {
			role = "user"
		}

		messages = append(messages, message{Role: role, Content: line.Text})
	}

	hint := ""
	if request.Hint != "" {
		hint = " " + request.Hint
	}

	return append(messages, message{
		Role: "user",
		Content: strings.TrimSpace(request.Text + "\n\n" + game.VerdictMark + " " + request.Decision.BriefFor(request.Role) + hint +
			" Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi. " + request.Correction),
	})
}

// intentSchema is the JSON schema the model's reading must conform to.
func intentSchema(wares []game.Ware) map[string]any {
	items := []string{""}
	for _, ware := range wares {
		items = append(items, ware.ID)
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":     map[string]any{"type": "string", "enum": game.Actions},
			"item_id":    map[string]any{"type": "string", "enum": items},
			"amount":     map[string]any{"type": "integer"},
			"persuasion": map[string]any{"type": "integer", "enum": []int{-2, -1, 0, 1, 2}},
		},
		"required":             []string{"action", "item_id", "amount", "persuasion"},
		"additionalProperties": false,
	}
}

const inventInstructions = `Czytasz kwestię kupca z gry tekstowej i sprawdzasz, czy zaoferował on do sprzedaży konkretny przedmiot, którego nie ma na liście jego towaru. ` +
	`Zwracasz obiekt JSON zgodny ze schematem. Nie odpowiadasz kupcowi.

- offered: true tylko wtedy, gdy kupiec wyraźnie proponuje coś do kupienia (przedmiot, miksturę, zwój, amulet), czego nie ma na liście. ` +
	`Wzmianka o czymś, czego nie sprzedaje, albo o rzeczy z listy pod inną nazwą, to false.
- name: nazwa przedmiotu tak, jak nazwał go kupiec (mianownik, najwyżej 4 słowa).
- description: jedno zdanie o przedmiocie, bez liczb.
- kind: weapon (broń), armor (pancerz), potion (mikstura lub środek leczący), spell (zwój, księga, schemat lub program uczący mocy), charm (noszony przedmiot magiczny lub gadżet).
- effect: dla potion "heal" albo "mana"; dla spell "bolt" (rani jednego wroga), "blast" (rani wszystkich) albo "heal" (leczy drużynę); ` +
	`dla charm "power" (atak), "ward" (obrona), "vigor" (zdrowie) albo "focus" (moc); dla weapon i armor "none".
- tier: 1 dla rzeczy zwykłej, 2 dla dobrej, 3 dla wyjątkowej.
Gdy offered jest false, pozostałe pola zostaw puste albo zerowe.`

func inventPrompt(request game.InventRequest) string {
	names := make([]string, 0, len(request.Wares))
	for _, ware := range request.Wares {
		names = append(names, ware.Name)
	}

	return "Towar kupca: " + strings.Join(names, ", ") + ".\n\nKwestia kupca:\n" + request.Reply
}

func inventSchema() map[string]any {
	text := map[string]any{"type": "string"}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"offered":     map[string]any{"type": "boolean"},
			"name":        text,
			"description": text,
			"kind":        map[string]any{"type": "string", "enum": append(append([]world.Kind(nil), world.Kinds...), "")},
			"effect":      map[string]any{"type": "string", "enum": append(append([]world.Effect(nil), world.Effects...), "")},
			"tier":        map[string]any{"type": "integer", "enum": []int{0, 1, 2, 3}},
		},
		"required":             []string{"offered", "name", "description", "kind", "effect", "tier"},
		"additionalProperties": false,
	}
}

const noteInstructions = `Czytasz kwestię mieszkańca osady z gry tekstowej i sprawdzasz, czy podał w niej NOWY, konkretny fakt o świecie gry: ` +
	`o osobie, miejscu, zdarzeniu, stworzeniu albo zwyczaju, którego nie ma na liście tematów już znanych. ` +
	`Zwracasz obiekt JSON: found true tylko wtedy, gdy fakt jest konkretny (ma nazwę własną albo wyraźne miejsce i treść) i nowy; ` +
	`subject to krótka nazwa tematu (najwyżej 4 słowa), fact to jedno zdanie po polsku w trzeciej osobie, bez zwrotów do gracza. ` +
	`Ogólniki, opinie o graczu, ceny, pogoda i to, co już jest na liście, to found false.`

func notePrompt(request game.NoteRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Tematy już znane:\n")
	for _, subject := range request.Known {
		fmt.Fprintf(&prompt, "- %s\n", subject)
	}

	fmt.Fprintf(&prompt, "\nKwestia:\n%s", request.Reply)

	return prompt.String()
}

const rewriteInstructions = `Poprawiasz jeden opis ze świata polskiej gry tekstowej. Zachowujesz sens i wszystkie nazwy własne, ` +
	`ale piszesz go od nowa: czysta, naturalna polszczyzna, konkret zamiast ogólników, bez zwrotów do gracza, bez cudzysłowów i bez komentarza. ` +
	`Najwyżej dwa zdania. Zwracasz wyłącznie nowy tekst.`

var rewriteKinds = map[string]string{
	game.DescribePlace:     "opis miejsca",
	game.DescribeMonster:   "opis potwora",
	game.DescribeBoss:      "opis największego zagrożenia",
	game.DescribeItem:      "opis przedmiotu",
	game.DescribeCompanion: "opis towarzysza do najęcia",
	game.DescribeKeeper:    "charakter mieszkańca osady (jak mówi, jaki jest)",
	game.DescribeFact:      "fakt o świecie (w trzeciej osobie)",
}

func rewritePrompt(request game.RewriteRequest) string {
	var prompt strings.Builder

	fmt.Fprintf(&prompt, "Świat: %s\n", request.World)
	if request.Style != "" {
		fmt.Fprintf(&prompt, "Styl świata: %s\n", request.Style)
	}

	kind := rewriteKinds[request.Kind]
	if kind == "" {
		kind = "opis"
	}

	fmt.Fprintf(&prompt, "Rodzaj: %s. Temat: %s.\nObecny tekst: %s\n\nNowy tekst:", kind, request.Name, request.Text)

	return prompt.String()
}

func noteSchema() map[string]any {
	return object(map[string]any{
		"found":   map[string]any{"type": "boolean"},
		"subject": map[string]any{"type": "string"},
		"fact":    map[string]any{"type": "string"},
	})
}
