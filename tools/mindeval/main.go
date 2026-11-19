// Command mindeval runs the Mind's calls (Interpret, Note, Invent,
// Translate) through llm.Client on real lines from a game session plus a
// few crafted ones, and writes each call with its latency as JSONL, for
// judging the Mind model and timing it on GPU against CPU.
//
//	go run ./tools/mindeval -model gemma3:4b -out tools/eval/out/mind-gpu.jsonl
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
)

type talk struct {
	Keeper   string `json:"keeper"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type result struct {
	Type    string  `json:"type"`
	ID      string  `json:"id"`
	Model   string  `json:"model"`
	Ms      float64 `json:"ms"`
	Input   any     `json:"input"`
	Output  any     `json:"output"`
	Expect  string  `json:"expect,omitempty"`
	Error   string  `json:"error,omitempty"`
	Comment string  `json:"comment,omitempty"`
}

var (
	wareRe    = regexp.MustCompile(`«([^»]+)»\s+(\d+)`)
	knownRe   = regexp.MustCompile(`(?m)^Wiesz: ([^.]+)\.`)
	unknownRe = regexp.MustCompile(`(?m)^Nie wiesz nic o: (.+)\. Jeśli`)
)

func main() {
	base := flag.String("base", "http://localhost:11434/v1", "OpenAI-compatible base URL")
	model := flag.String("model", "gemma3:4b", "Mind model")
	window := flag.Int("ctx", 8192, "context window (NEURO_LLM_CONTEXT default)")
	talks := flag.String("talks", "tools/eval/talks/talks-1018.jsonl", "talk log")
	out := flag.String("out", "tools/eval/out/mind.jsonl", "results JSONL")
	unload := flag.Bool("unload", true, "unload every model before starting")
	dry := flag.Bool("dry", false, "list the cases and exit")
	rescore := flag.String("rescore", "", "results JSONL to pass through the game's fact filters again, without asking any model")
	flag.Parse()

	ctx := context.Background()
	root := strings.TrimSuffix(strings.TrimRight(*base, "/"), "/v1")
	client := &llm.Client{BaseURL: *base, Model: *model, Context: *window}

	entries := load(*talks)
	known := knownSubjects(entries)

	if *rescore != "" {
		rescoreNotes(*rescore, entries, known)

		return
	}

	if *dry {
		fmt.Println("known:", strings.Join(known, "; "))
		for _, line := range keeperLines(entries) {
			fmt.Printf("line %s %v noteworthy=%v %.90s\n", line.id, names(line.wares), game.Noteworthy(line.text), line.text)
		}
		for _, message := range playerMessages(entries) {
			reading := game.Read(message.request)
			open := reading.Open(len(strings.Fields(message.request.Text)) >= 6)
			fmt.Printf("msg %s open=%v sure=%v rules=%+v | %s\n", message.id, open, reading.Sure, reading.Intent, message.request.Text)
		}
		return
	}

	file, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	emit := func(r result) {
		r.Model = *model
		data, _ := json.Marshal(r)
		writer.Write(append(data, '\n'))
		writer.Flush()
	}

	if *unload {
		unloadAll(root)
	}

	vramIdle := vram()
	start := time.Now()
	client.Warm(ctx)
	load := time.Since(start)
	vramLoaded := vram()
	emit(result{Type: "load", ID: "warm", Ms: ms(load), Output: map[string]any{"vram_idle_mib": vramIdle, "vram_loaded_mib": vramLoaded, "ps": ps(root)}})
	fmt.Printf("load %.1fs, VRAM %d -> %d MiB\n", load.Seconds(), vramIdle, vramLoaded)

	latencies := map[string][]float64{}
	timed := func(kind string, call func() (any, error)) (any, float64, string) {
		begin := time.Now()
		output, err := call()
		elapsed := ms(time.Since(begin))
		latencies[kind] = append(latencies[kind], elapsed)
		message := ""
		if err != nil {
			message = err.Error()
		}
		return output, elapsed, message
	}

	// Note and Invent for every keeper's line, as the game asks after it.
	for _, line := range keeperLines(entries) {
		request := game.NoteRequest{Reply: line.text, Known: known}
		output, elapsed, problem := timed("note", func() (any, error) {
			fact, found, err := client.Note(ctx, request)
			return map[string]any{"found": found, "fact": fact, "noteworthy": game.Noteworthy(line.text), "grounded": found && game.Grounded(fact, line.text)}, err
		})
		emit(result{Type: "note", ID: line.id, Ms: elapsed, Input: map[string]any{"keeper": line.keeper, "reply": line.text}, Output: output, Expect: line.fact, Error: problem})

		invent := game.InventRequest{Reply: line.text, Wares: line.wares}
		output, elapsed, problem = timed("invent", func() (any, error) {
			item, offered, err := client.Invent(ctx, invent)
			return map[string]any{"offered": offered, "item": item, "kept": offered && game.Offered(line.text, item.Name)}, err
		})
		emit(result{Type: "invent", ID: line.id, Ms: elapsed, Input: map[string]any{"keeper": line.keeper, "reply": line.text, "wares": names(line.wares)}, Output: output, Expect: line.item, Error: problem})
	}

	// Interpret for player messages the rules leave open.
	asked, skipped := 0, 0
	for _, message := range playerMessages(entries) {
		reading := game.Read(message.request)
		if !reading.Open(len(strings.Fields(message.request.Text)) >= 6) {
			skipped++
			continue
		}
		asked++
		output, elapsed, problem := timed("interpret", func() (any, error) {
			intent, err := client.Interpret(ctx, message.request)
			return map[string]any{"intent": intent, "rules": reading.Intent, "rules_sure": reading.Sure}, err
		})
		emit(result{Type: "interpret", ID: message.id, Ms: elapsed, Input: map[string]any{"text": message.request.Text, "focus": message.request.FocusID, "wares": names(message.request.Wares)}, Output: output, Expect: message.expect, Error: problem})
	}
	fmt.Printf("interpret: %d asked, %d settled by rules\n", asked, skipped)

	for i, text := range translations {
		output, elapsed, problem := timed("translate", func() (any, error) { return client.Translate(ctx, text) })
		emit(result{Type: "translate", ID: fmt.Sprintf("t%02d", i+1), Ms: elapsed, Input: text, Output: output, Error: problem})
	}

	vramEnd := vram()
	emit(result{Type: "load", ID: "end", Output: map[string]any{"vram_mib": vramEnd, "ps": ps(root)}})

	kinds := make([]string, 0, len(latencies))
	for kind := range latencies {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		values := latencies[kind]
		fmt.Printf("%-10s n=%2d median %6.0f ms  p90 %6.0f ms\n", kind, len(values), quantile(values, 0.5), quantile(values, 0.9))
	}
	fmt.Printf("VRAM at end: %d MiB\n", vramEnd)
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func quantile(values []float64, q float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 0 {
		return 0
	}
	index := int(q*float64(len(sorted)-1) + 0.5)
	return sorted[index]
}

func load(path string) []talk {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var entries []talk
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<24)
	for scanner.Scan() {
		var entry talk
		if json.Unmarshal(scanner.Bytes(), &entry) == nil && len(entry.Messages) > 0 {
			entries = append(entries, entry)
		}
	}
	return entries
}

// knownSubjects approximates Realm.Subjects: every subject the keepers'
// lore names as a fact or a referral.
func knownSubjects(entries []talk) []string {
	seen := map[string]bool{}
	var subjects []string
	add := func(subject string) {
		subject = strings.TrimSpace(subject)
		if subject != "" && !seen[subject] {
			seen[subject] = true
			subjects = append(subjects, subject)
		}
	}
	for _, entry := range entries {
		system := entry.Messages[0].Content
		for _, match := range knownRe.FindAllStringSubmatch(system, -1) {
			add(match[1])
		}
		for _, match := range unknownRe.FindAllStringSubmatch(system, -1) {
			for _, part := range strings.Split(match[1], "), ") {
				if name, _, ok := strings.Cut(part, " ("); ok {
					add(name)
				}
			}
		}
	}
	sort.Strings(subjects)
	return subjects
}

func wares(system string) []game.Ware {
	var list []game.Ware
	for _, match := range wareRe.FindAllStringSubmatch(system, -1) {
		var price int
		fmt.Sscan(match[2], &price)
		list = append(list, game.Ware{ID: slug(match[1]), Name: match[1], Price: price})
	}
	return list
}

func slug(name string) string {
	return strings.ReplaceAll(game.Fold(name), " ", "-")
}

func names(list []game.Ware) []string {
	out := make([]string, 0, len(list))
	for _, ware := range list {
		out = append(out, ware.Name)
	}
	return out
}

type keeperLine struct {
	id, keeper, text string
	wares            []game.Ware
	fact, item       string
}

var (
	thorinWares = []game.Ware{{ID: "krotki-miecz", Name: "Krótki Miecz", Price: 40}, {ID: "krasnoludzki-topor", Name: "Krasnoludzki Topór", Price: 110}, {ID: "skorzany-kaftan", Name: "Skórzany Kaftan", Price: 35}, {ID: "kolczuga", Name: "Kolczuga", Price: 95}}
	agnesWares  = []game.Ware{{ID: "mikstura-leczenia", Name: "Mikstura Leczenia", Price: 15}, {ID: "wielka-mikstura-leczenia", Name: "Wielka Mikstura Leczenia", Price: 35}, {ID: "eliksir-jasnosci", Name: "Eliksir Jasności", Price: 15}}
)

// crafted are keeper lines with a known answer, to see misses as well as
// false alarms: the session's own lines hardly ever offer or tell anything new.
var crafted = []keeperLine{
	{id: "c-inv1", keeper: "Siostra Agnes", text: "[spokój] Mam też Maść z Arniki na rany po goblińskich nożach. Kosztuje 12 sztuk złota.", wares: agnesWares, item: "TAK: Maść z Arniki, potion heal", fact: "nie (towar)"},
	{id: "c-inv2", keeper: "Thorin Żelazna Broda", text: "[kpina] Elfie badziewie? Na zapleczu trzymam Młot Runiczny dla kogoś, kto go udźwignie, 150 sztuk.", wares: thorinWares, item: "TAK: Młot Runiczny, weapon", fact: "nie"},
	{id: "c-inv3", keeper: "Thorin Żelazna Broda", text: "[spokój] Kolczuga 95 i ani grosza mniej. Kowal z Przystani kuje gorsze.", wares: thorinWares, item: "NIE (z listy)", fact: "nie"},
	{id: "c-inv4", keeper: "Siostra Agnes", text: "[smutek] Amuletów nie sprzedaję, to robota Ozryka z jego wieży.", wares: agnesWares, item: "NIE (odmowa)", fact: "nie / ewentualnie Ozryk robi amulety"},
	{id: "c-inv5", keeper: "Siostra Agnes", text: "[czułość] Weź Wielką Miksturę Leczenia, za 35 postawi cię na nogi po każdej bójce.", wares: agnesWares, item: "NIE (z listy)", fact: "nie"},
	{id: "c-inv6", keeper: "Thorin Żelazna Broda", text: "[radość] Mogę ci sprzedać Dębową Tarczę, lekka i mocna, 30 sztuk złota.", wares: thorinWares, item: "TAK: Dębowa Tarcza, armor", fact: "nie"},
	{id: "c-inv7", keeper: "Siostra Agnes", text: "[spokój] Elira Cieniokrąg nosi podobno Pierścień Płomieni, ale ja takich rzeczy nie mam.", wares: agnesWares, item: "NIE (wzmianka)", fact: "może: Elira nosi Pierścień Płomieni (Elira znana)"},
	{id: "c-not1", keeper: "Thorin Żelazna Broda", text: "[strach] W Starym Borze grasuje wilkołak, drwale wołają na niego Szary Kieł. Dwóch już nie wróciło.", wares: thorinWares, item: "NIE", fact: "TAK: Szary Kieł, wilkołak w Starym Borze"},
	{id: "c-not2", keeper: "Siostra Agnes", text: "[spokój] Wędrowny Kupiec zjeżdża do Karwina co dziesięć dni, zawsze od strony Przystani.", wares: agnesWares, item: "NIE", fact: "TAK: Wędrowny Kupiec przyjeżdża co 10 dni"},
	{id: "c-not3", keeper: "Thorin Żelazna Broda", text: "[kpina] Pogoda dziś piękna, a ty i tak wyglądasz jak zmokła kura.", wares: thorinWares, item: "NIE", fact: "NIE (pogoda, opinia)"},
	{id: "c-not4", keeper: "Thorin Żelazna Broda", text: "[gniew] Krótki Miecz 40, bierz albo idź.", wares: thorinWares, item: "NIE", fact: "NIE (cena)"},
	{id: "c-not5", keeper: "Siostra Agnes", text: "[gniew] Z Kopalni Durgara bije czerwona łuna, wszyscy to wiedzą.", wares: agnesWares, item: "NIE", fact: "NIE (znane)"},
	{id: "c-not6", keeper: "Thorin Żelazna Broda", text: "[smutek] Mój brat Harald zginął na Przełęczy Wichrów, troll zepchnął go w przepaść.", wares: thorinWares, item: "NIE", fact: "TAK: Harald, brat Thorina, zginął na Przełęczy"},
}

func keeperLines(entries []talk) []keeperLine {
	var lines []keeperLine
	seen := map[string]bool{}
	for i, entry := range entries {
		if entry.Keeper == "Mistrz Gry" {
			continue
		}
		last := entry.Messages[len(entry.Messages)-1]
		if last.Role != "assistant" || seen[last.Content] {
			continue
		}
		seen[last.Content] = true
		lines = append(lines, keeperLine{id: fmt.Sprintf("log%02d", i), keeper: entry.Keeper, text: last.Content, wares: wares(entry.Messages[0].Content)})
	}
	return append(lines, crafted...)
}

type playerMessage struct {
	id      string
	request game.InterpretRequest
	expect  string
}

func playerMessages(entries []talk) []playerMessage {
	var messages []playerMessage
	seen := map[string]bool{}
	for i, entry := range entries {
		if entry.Keeper == "Mistrz Gry" {
			continue
		}
		var history []game.Line
		for j, message := range entry.Messages[1:] {
			text, _, _ := strings.Cut(message.Content, "\n\n"+game.VerdictMark)
			if message.Role == "user" && j == len(entry.Messages)-3 && !seen[text] {
				seen[text] = true
				messages = append(messages, playerMessage{
					id:      fmt.Sprintf("log%02d", i),
					request: game.InterpretRequest{Text: text, Wares: wares(entry.Messages[0].Content), History: tail(history, 6), Currency: "szt. złota"},
				})
			}
			history = append(history, game.Line{Player: message.Role == "user", Text: text})
		}
	}

	thorinHistory := []game.Line{{Text: "Czego? Broń jest na ścianie, ceny na tabliczce."}, {Player: true, Text: "ile za krotki miecz"}, {Text: "Krótki Miecz. Prosty, z Karwina, zaostrzony. 40 w monetach."}}
	agnesHistory := []game.Line{{Text: "Znowu ranny? Siadaj, pokaż. Co ci sprzedać, żebyś tu nie wracał?"}, {Player: true, Text: "ile kosztuje mikstura leczenia"}, {Text: "Mikstura Leczenia 15 sztuk złota. Wielka 35."}}
	thorin := func(id, text, focus, expect string) playerMessage {
		return playerMessage{id: id, request: game.InterpretRequest{Text: text, Wares: thorinWares, FocusID: focus, History: thorinHistory, Currency: "szt. złota"}, expect: expect}
	}
	agnes := func(id, text, focus, expect string) playerMessage {
		return playerMessage{id: id, request: game.InterpretRequest{Text: text, Wares: agnesWares, FocusID: focus, History: agnesHistory, Currency: "szt. złota"}, expect: expect}
	}

	return append(messages,
		thorin("c-i01", "30", "krotki-miecz", "offer 30"),
		thorin("c-i02", "moze 32?", "krotki-miecz", "offer 32"),
		thorin("c-i03", "co powiesz na 25", "krotki-miecz", "offer 25"),
		thorin("c-i04", "mam przy sobie tylko 28 zlota", "krotki-miecz", "offer 28 (lub chat, dopuszczalne)"),
		thorin("c-i05", "zabilem dzis 3 gobliny w starym borze", "", "chat, NIE offer"),
		thorin("c-i06", "mam 2 rece i obie puste, potrzebuje jakiejs broni", "", "chat/ask, NIE offer"),
		thorin("c-i07", "dzien 1 a ja juz bez broni, co polecasz", "", "chat/ask, NIE offer"),
		thorin("c-i08", "Szanowny mistrzu, gobliny ukradły mi wszystko, a idę bić się ze smokiem za całą dolinę. Opuść trochę cenę topora, proszę.", "", "ask/chat, item topor, persuasion 1-2"),
		thorin("c-i09", "ty stary skapy krasnoludzie, twoje miecze sa gorsze od elfich, daj taniej", "krotki-miecz", "persuasion -1/-2"),
		thorin("c-i10", "dam 50 za kolczuge bo i tak nikt jej nie kupi w tej zabitej dechami dziurze", "", "offer 50 kolczuga, persuasion -1"),
		thorin("c-i11", "biore kaftan, ale dorzuc cos ekstra bo jestem twoim pierwszym klientem dzisiaj", "", "accept kaftan, persuasion 0/1"),
		thorin("c-i12", "a gdybym przyniosl ci 5 skor szczurow, opuscisz cos z ceny?", "krotki-miecz", "chat/ask, NIE offer 5"),
		thorin("c-i13", "ten topor 90 i mamy umowe", "", "offer 90 topor"),
		thorin("c-i14", "slyszalem ze Borin bije sie za dwoch, ty tez kiedys walczyles z goblinami?", "", "chat"),
		thorin("c-i15", "krasnoludzka stal jest najlepsza na swiecie, a twoja kuznia najlepsza w dolinie. sprzedasz mi miecz za 35?", "krotki-miecz", "offer 35, persuasion 1-2"),
		agnes("c-i16", "dziecko mi choruje w przystani, potrzebuje mikstury leczenia, ale mam tylko 12", "", "offer 12 (mikstura), persuasion 1"),
		agnes("c-i17", "prosze siostry, zakonnica nie powinna zdzierac z rannych, 10 za miksture to uczciwa cena", "mikstura-leczenia", "offer 10, persuasion 0/1"),
		agnes("c-i18", "12?", "mikstura-leczenia", "offer 12"),
		agnes("c-i19", "wroce za 2 dni po wielka", "", "chat, NIE offer 2"),
		agnes("c-i20", "Siostro, uratowałem wczoraj trzech chłopów z Pól Karw przed goblinami i jestem cały w ranach, czy dasz mi miksturę taniej?", "mikstura-leczenia", "ask/chat, persuasion 2"),
	)
}

func tail(lines []game.Line, n int) []game.Line {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return append([]game.Line(nil), lines...)
}

var translations = []string{
	"Omszałe kopce dawnych władców doliny. Nocą płoną nad nimi błędne ognie.",
	"Szczur wielkości psa, z zębami pożółkłymi od padliny.",
	"Krasnoludzki kowal, dumny i małomówny, z siwą brodą zaplecioną w warkocze.",
	"Elfia czarodziejka w ciemnozielonym płaszczu, z płomieniem tańczącym na dłoni.",
	"Zakonnica-zielarka o surowym spojrzeniu i dobrym sercu.",
	"Oblodzona przełęcz, gdzie wiatr zrywa z nóg, a każdy kamień może być trollem.",
	"Krasnoludzki Topór, wyważony tak, że sam szuka karku.",
	"Kupiecka osada u stóp góry, z palisadą, karczmą i tablicą pełną listów gończych.",
	"Młody czerwony smok, który obrał kopalnię za leże i goblinów za służbę.",
	"Duch spalonego górnika, który wciąż szuka wyjścia z kopalni.",
	"wysoki krasnolud z ruda broda, blizna przez oko, skorzana zbroja",
	"elfka lucznik, srebrne wlosy, zielony kaptur, luk z cisu",
}

func unloadAll(root string) {
	for _, name := range loaded(root) {
		body, _ := json.Marshal(map[string]any{"model": name, "keep_alive": 0})
		if response, err := http.Post(root+"/api/generate", "application/json", bytes.NewReader(body)); err == nil {
			response.Body.Close()
		}
	}
	for i := 0; i < 30 && len(loaded(root)) > 0; i++ {
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(time.Second)
}

func loaded(root string) []string {
	var listing struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	response, err := http.Get(root + "/api/ps")
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&listing)
	var names []string
	for _, model := range listing.Models {
		names = append(names, model.Name)
	}
	return names
}

func ps(root string) any {
	response, err := http.Get(root + "/api/ps")
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	var listing any
	json.NewDecoder(response.Body).Decode(&listing)
	return listing
}

func vram() int {
	output, err := exec.Command("nvidia-smi", "--query-gpu=memory.used", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return -1
	}
	var used int
	fmt.Sscan(strings.TrimSpace(string(output)), &used)
	return used
}

// rescoreNotes passes the facts a run noted through the game's filters as
// they are now (Noteworthy, Grounded, Stale) and prints what each did with
// them, so that a change of the filters is judged on the same answers.
func rescoreNotes(path string, entries []talk, known []string) {
	lines := map[string]keeperLine{}
	keepers := map[string]bool{}
	for _, line := range append(keeperLines(entries), crafted...) {
		lines[line.id] = line
		keepers[line.keeper] = true
	}

	names := []string{}
	for keeper := range keepers {
		names = append(names, keeper)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	counts := map[string]int{}

	for _, raw := range bytes.Split(data, []byte("\n")) {
		var entry struct {
			Type   string `json:"type"`
			ID     string `json:"id"`
			Output struct {
				Fact struct {
					Subject string `json:"subject"`
					Fact    string `json:"fact"`
					Quote   string `json:"quote"`
				} `json:"fact"`
				Found bool `json:"found"`
			} `json:"output"`
		}

		if json.Unmarshal(raw, &entry) != nil || entry.Type != "note" || !entry.Output.Found {
			continue
		}

		line := lines[entry.ID]
		spec := game.FactSpec{Subject: entry.Output.Fact.Subject, Text: entry.Output.Fact.Fact, Quote: entry.Output.Fact.Quote}
		verdict := "PRZYJĘTY"

		switch {
		case !game.Noteworthy(line.text):
			verdict = "bez nazwy"
		case !game.Grounded(spec, line.text):
			verdict = "bez pokrycia"
		default:
			if why := game.Stale(spec, line.text, game.Known{Subjects: known, Wares: names2(line.wares), Names: names}); why != "" {
				verdict = why
			}
		}

		counts[verdict]++
		fmt.Printf("%-8s %-18s %s: %s  [%s]\n", entry.ID, verdict, spec.Subject, spec.Text, line.fact)
	}

	fmt.Println(counts)
}

func names2(wares []game.Ware) []string {
	out := []string{}
	for _, ware := range wares {
		out = append(out, ware.Name)
	}

	return out
}
