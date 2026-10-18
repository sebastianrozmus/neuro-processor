package game

import "testing"

// The cases come from what gemma3:4b noted in a real session (18.10):
// tools/eval/out/mind-1018.md.
func TestStaleFacts(t *testing.T) {
	known := Known{
		Subjects: []string{"Kurhany Królów", "Elira", "Błędne Ognie"},
		Texts:    []string{"W Kurhanach Królów spoczywają dawni władcy.", "Elira z Kuźni zna stare runy."},
		Wares:    []string{"Kolczuga", "Krótki Miecz", "Wielka Mikstura"},
		Names:    []string{"Borin Kamienna Pięść", "Siostra Agnes", "Thorin"},
	}

	tests := []struct {
		name string
		line string
		spec FactSpec
		want string
	}{
		{
			"what the keeper does not know is no fact",
			"[spokój] Ozryka nie znam, pytaj Siostrę Agnes.",
			FactSpec{Subject: "Ozryk", Text: "Ozryk nie jest znany.", Quote: "Ozryka nie znam"},
			staleDenial,
		},
		{
			"a secret kept is not a secret told",
			"[gniew] Położenia Kurhanów Królów nie wyznam, choćbyś płacił złotem.",
			FactSpec{Subject: "Kurhany Królów", Text: "Mieszkaniec ujawnił położenie Kurhanów Królów.", Quote: "Położenia Kurhanów Królów nie wyznam"},
			staleDenial,
		},
		{
			"a fact that only says it is unknown",
			"[spokój] Ozryk to zagadka dla wszystkich.",
			FactSpec{Subject: "Ozryk", Text: "Ozryk jest nieznanym miejscem.", Quote: "Ozryk to zagadka"},
			staleDenial,
		},
		{
			"a denial in another sentence does not count",
			"[spokój] Ozryka nie znam. Na Mokradłach Harald widział nocą zielone światła.",
			FactSpec{Subject: "Mokradła", Text: "Harald widział nocą zielone światła na Mokradłach.", Quote: "Na Mokradłach Harald widział nocą zielone światła"},
			"",
		},
		{
			"a price belongs to the rules",
			"[radość] Krótki Miecz oddałem wczoraj za 25 sztuk złota.",
			FactSpec{Subject: "Sprzedaż", Text: "Wczoraj sprzedano miecz za 25 sztuk złota.", Quote: "oddałem wczoraj za 25 sztuk złota"},
			staleNumber,
		},
		{
			"a ware on the shelf is not news",
			"[spokój] Kolczugę mam prosto od kowala z gór, lepszej nie dostaniesz.",
			FactSpec{Subject: "Kolczuga", Text: "Kolczuga pochodzi od kowala z gór.", Quote: "Kolczugę mam prosto od kowala z gór"},
			staleWare,
		},
		{
			"nor is a quote about one",
			"[spokój] Wielką Miksturę warzę sama, z korzeni z Mokradeł.",
			FactSpec{Subject: "Korzenie z Mokradeł", Text: "Korzenie z Mokradeł idą do mikstur.", Quote: "Wielką Miksturę warzę sama, z korzeni z Mokradeł"},
			staleWare,
		},
		{
			"a name the line never says is made up",
			"[spokój] Na zapleczu trzymam Młot Runiczny, ale nie na sprzedaż.",
			FactSpec{Subject: "Młot Runiczny", Text: "Niespodziewanie, Grigor znajduje Młot Runiczny.", Quote: "Na zapleczu trzymam Młot Runiczny"},
			staleNamed,
		},
		{
			"a known subject again",
			"[spokój] Mówią, że w dolinie leżą Kurhany Królów.",
			FactSpec{Subject: "Kurhany Królów", Text: "W dolinie leżą Kurhany Królów.", Quote: "w dolinie leżą Kurhany Królów"},
			staleKnown,
		},
		{
			"a known subject in other words",
			"[spokój] Mówią, że w dolinie leżą Kurhany Królów i że to siedziby dawnych władców.",
			FactSpec{Subject: "Siedziby dawnych władców", Text: "Siedziby dawnych władców leżą w dolinie.", Quote: "Kurhany Królów i że to siedziby dawnych władców"},
			staleKnown,
		},
		{
			"a known subject inflected",
			"[strach] W Kurhanach Królów straszy po zmroku.",
			FactSpec{Subject: "Kurhan", Text: "W kurhanie straszy po zmroku.", Quote: "W Kurhanach Królów straszy po zmroku"},
			staleKnown,
		},
		{
			"a new name makes a known subject news",
			"[strach] W Kurhanach Królów nocuje teraz Harald Rudobrody ze swoją bandą.",
			FactSpec{Subject: "Kurhany Królów", Text: "W Kurhanach Królów nocuje Harald Rudobrody z bandą.", Quote: "W Kurhanach Królów nocuje teraz Harald Rudobrody"},
			"",
		},
		{
			"a known person is no new name",
			"[spokój] Elira z Kuźni bywa w Kurhanach Królów co pełnię.",
			FactSpec{Subject: "Kurhany Królów", Text: "Elira bywa w Kurhanach Królów co pełnię.", Quote: "Elira z Kuźni bywa w Kurhanach Królów co pełnię"},
			staleKnown,
		},
		{
			"a name in another case is still the line's",
			"[strach] W Starym Borze grasuje wilkołak, drwale wołają na niego Szary Kieł.",
			FactSpec{Subject: "Wilkołak w Starym Borze", Text: "Wilkołak nazywany Szarym Kiełem grasuje w Starym Borze.", Quote: "W Starym Borze grasuje wilkołak"},
			"",
		},
		{
			"a fact retold as about a known subject",
			"[spokój] Przy Kurhanach Królów ludzie słyszą nocą głos podróżnego.",
			FactSpec{Subject: "Głos podróżnego", Text: "Ludzie słyszą głos podróżnego w Kurhanach Królów.", Quote: "ludzie słyszą nocą głos podróżnego"},
			staleKnown,
		},
		{
			"plain news passes",
			"[spokój] Nocą w młynie nad rzeką świeci się zielone światło.",
			FactSpec{Subject: "Młyn nad rzeką", Text: "Nocą w młynie świeci się zielone światło.", Quote: "w młynie nad rzeką świeci się zielone światło"},
			"",
		},
		{
			"news naming known people passes",
			"[kpina] Thorin przegrał wczoraj w kości swój topór z Przełęczy.",
			FactSpec{Subject: "Topór Thorina", Text: "Thorin przegrał w kości swój topór.", Quote: "Thorin przegrał wczoraj w kości swój topór"},
			"",
		},
	}

	for _, test := range tests {
		if got := Stale(test.spec, test.line, known); got != test.want {
			t.Errorf("%s: Stale = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestSentenceWith(t *testing.T) {
	line := "Ozryka nie znam. Na Mokradłach Harald widział światła! Tyle wiem."

	if got := sentenceWith(line, Fold("Harald widział")); got != Fold("Na Mokradłach Harald widział światła") {
		t.Errorf("sentence %q", got)
	}

	if got := sentenceWith(line, "czego nie ma"); got != Fold(line) {
		t.Errorf("a quote not in the line should give the whole line, got %q", got)
	}
}

func TestProperNamesSkipSentenceStarts(t *testing.T) {
	got := properNames("Mówią, że Harald z Przystani wie. Kurhany leżą: Ozryk tam był.")
	want := []string{"harald", "przystani", "ozryk"}

	if len(got) != len(want) {
		t.Fatalf("names %v, want %v", got, want)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf("names %v, want %v", got, want)
		}
	}
}

func TestLearnTakesOnlyNews(t *testing.T) {
	realm := testRealm(t)
	hero := enlist(realm, "Ala")
	do(realm, hero, "idź do zbrojowni")

	realm.mu.Lock()
	ware := realm.session(hero).Wares()[0].Name
	realm.mu.Unlock()

	refused := []struct {
		why  string
		spec FactSpec
		line string
	}{
		{"denial", FactSpec{Subject: "Ozryk", Text: "Ozryk nie jest znany.", Quote: "Ozryka nie znam"}, "Ozryka nie znam, pytaj gdzie indziej."},
		{"price", FactSpec{Subject: "Targ", Text: "Na targu płacą 40 sztuk.", Quote: "na targu płacą 40 sztuk"}, "Wczoraj na targu płacą 40 sztuk, mówię ci."},
		{"ware", FactSpec{Subject: ware, Text: ware + " jest z gór.", Quote: ware + " mam z gór"}, ware + " mam z gór, prosto z kuźni."},
		{"known subject", FactSpec{Subject: "Szef", Text: "Szef śpi za dnia.", Quote: "Szef śpi za dnia"}, "Szef śpi za dnia, słyszałem."},
	}

	for _, test := range refused {
		if realm.Learn(hero, test.spec, test.line) {
			t.Errorf("%s: learnt %+v", test.why, test.spec)
		}
	}

	if !realm.Learn(hero, FactSpec{Subject: "Młyn nad rzeką", Text: "Nocą świeci się tam zielone światło.", Quote: "w młynie nad rzeką świeci się zielone światło"}, "Nocą w młynie nad rzeką świeci się zielone światło.") {
		t.Error("news was not learnt")
	}

	// Once learnt, the mill is old news in other words too.
	if realm.Learn(hero, FactSpec{Subject: "Młyn", Text: "W młynie straszy.", Quote: "w młynie nad rzeką straszy"}, "Mówią, że w młynie nad rzeką straszy.") {
		t.Error("a known subject was learnt again in other words")
	}

	if journal := realm.Journal(hero); len(journal) != 1 {
		t.Errorf("journal %+v", journal)
	}
}

// The mind quotes with the full stop; the stop must not hide which
// sentence the quote is from.
func TestStaleQuoteWithStop(t *testing.T) {
	line := "[spokój] Mówią, że w dolinie leżą Kurhany Królów. Nic poza tym nie wiem, pytaj kupca."
	spec := FactSpec{Subject: "Dolina", Text: "W dolinie leżą kurhany.", Quote: "w dolinie leżą Kurhany Królów."}

	if got := Stale(spec, line, Known{}); got != "" {
		t.Errorf("Stale = %q: a denial in the next sentence was taken for this one", got)
	}
}
