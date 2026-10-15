# Pomiar modelu rozmów (bench i odtworzenia)

Tu mierzymy każdą nową wersję głosu gry (`neuro-npc-runN`), zanim gra się na nią przełączy. Mierzymy na dwa sposoby, bo jeden nie wystarcza:

1. **Bench**: 80 stałych przypadków, których model nigdy nie widział w treningu. Automat sprawdza format: liczby cyframi i tylko te z gry, wymaganą kwotę, liczbę zdań, znacznik nastroju, brak cudzysłowów i dopisków. **Nie** ocenia sensu.
2. **Odtworzenia**: prawdziwe rozmowy z Twojej gry, zalogowane z pełnym promptem, puszczone na stary i nowy model. Tu oceniasz oczami, czy odpowiedź jest na temat, po polsku i w roli.

Model wygrywa, gdy naprawia to, co widziałeś w grze, a bench nie spada bardziej niż szum, czyli o ±3 na 80.

| plik | co robi |
|---|---|
| `bench.sh` | bench modelu w dwóch seedach (42 i 7); wyniki do `results/<model>-s<seed>/` |
| `compare.py` | tabela wszystkich wyników: OK/80, wynik według ról, najczęstsze błędy |
| `replay.py` | odtwarza zalogowane rozmowy na kilku modelach |
| `check_data.py` | sprawdza nową paczkę danych (np. zamówienie z `tools/model/PROMPT.md`) przed treningiem, tymi samymi zasadami co bench; narracje Mistrza Gry bez znacznika nastroju |
| `bench_ollama.py`, `npc_checks.py` | silnik benchu i sprawdzanie formatu (sama biblioteka standardowa Pythona) |
| `test_tools.py` | testy sprawdzania (`python3 -m unittest test_tools` w tym folderze) |
| `eval/regression-80.jsonl` | 80 przypadków: 40 najemników, 24 towarzyszy, 16 kupców, 4 gatunki, światy spoza treningu. **Nigdy nie dodawaj ich do danych treningowych.** |
| `eval/regression-80-game.jsonl` | te same przypadki z poleceniami gry w obecnym brzmieniu; tworzy go `bench.sh`, nie edytuj ręcznie |
| `results/` | dotychczasowe pomiary (opis niżej) |

## Krok po kroku

**0. Przed treningiem: sprawdź nowe dane**

```sh
python3 tools/bench/check_data.py ~/Library/Application\ Support/neuro-processor/extra-gpt-npc-run7.jsonl
```

Pokazuje wiersze z błędami i ich numery linii. Pojedyncze fałszywe alarmy się zdarzają (np. „od siedmiu boleści” liczy się jako liczba słowem). Popraw albo usuń resztę i przeczytaj losowe 30 linii oczami: kierunek zapłaty, rola, polszczyzna.


Potrzebne: Ollama z modelem (po treningu na Macu `tools/model/train.sh` i `publish.sh`, po treningu na PC `tools/model/import.sh`, patrz `tools/pc/README.md`). Python 3.10+ bez dodatkowych pakietów. Na Macu z 18 GB **nie** odpalaj gry ani drugiego pomiaru w tym samym czasie: dwa modele naraz się nie zmieszczą i wyniki zrobią się wolne.

**1. Bench (po treningu)**

```sh
tools/bench/bench.sh neuro-npc-run7
```

4,5B potrzebuje ok. 5 minut na seed, 11B ok. 10–15 minut. Na końcu dostajesz tabelę, którą możesz potem wywołać sam:

```sh
python3 tools/bench/compare.py run6 run7
```

Jeśli zmieniły się polecenia gry w `internal/game/narration.go`, zmienia się też skrót pliku przypadków (ostatnia kolumna tabeli). Wtedy zmierz od nowa także poprzedni model: usuń jego folder w `results/` i uruchom `bench.sh` jeszcze raz. Porównujemy tylko wyniki z tym samym skrótem.

Szczegóły każdej odpowiedzi są w `results/<model>-s<seed>/responses.jsonl`: pytanie, odpowiedź, wzorzec (tylko do porównania, nie trafia do modelu) i błędy.

**2. Odtworzenia**

Rozmowy loguje gra, gdy w `.env.local` jest `NEURO_LOG_TALKS=1`. Trafiają do `~/Library/Application Support/neuro-processor/talks.jsonl`. Pograj chwilę, zwłaszcza w tych miejscach, gdzie poprzedni model był słaby, i:

```sh
python3 tools/bench/replay.py neuro-npc-run6,neuro-npc-run7 > tools/bench/results/replays/replay-run7.out
```

Dla każdego otwarcia z listy (`--keys`, np. „co wiesz o krolu”, „plotki”, „skad te piwo”) bierze ostatnią taką rozmowę z logu. Pokazuje, co gra wtedy powiedziała, a potem po 3 odpowiedzi każdego modelu. Własne pytania podajesz tak: `--keys "jaka masz zbroje,ile za miecz"`.

**3. Decyzja**

Jeśli nowy wygrywa, w `.env.local` ustaw `NEURO_TALK_MODEL=neuro-npc-run7`. Poprzedni zostaje w Ollamie jako punkt powrotu. Wynik dopisz w `tools/model/PROMPT.md` (sekcja wyników danego runu) i w `tools/model/HISTORIA.md`.

## Co znaczą błędy

| błąd | znaczy |
|---|---|
| `sentence_count` | za dużo albo za mało zdań względem polecenia |
| `spelled_numerals:trzy` | liczba słowem zamiast cyfrą (gra sprawdza cyfry, więc „trzy” przemyca kwotę) |
| `forbidden_numbers:50` | kwota, której gra zabroniła (np. stara cena po targu) |
| `missing_required_numbers:28` | brak kwoty, którą gra kazała podać |
| `numbers_in_non_numeric_case` | liczba w rozmowie, w której ma jej nie być |
| `markup_or_quotes_in_speech` | cudzysłowy, gwiazdki, dopiski typu „Postać:” |
| `generation_truncated` | odpowiedź urwana na limicie, zwykle gadulstwo |

## Dotychczasowe wyniki

| model | seed 42 | seed 7 | uwagi |
|---|---|---|---|
| run 3 | 71 | — | baza przed danymi celowanymi |
| run 4 | 78 | 79 | po 400 wierszach celowanych |
| run 4, temp. 0,6 | 75 | — | niższa temperatura nie pomogła |
| run 5 | 72 | 78 | w szumie; odtworzenia wyraźnie lepsze, więc przełączony |
| run 6 | 76 | 78 | tropy i dopytania 3/3; przełączony |
| Bielik 11B bez treningu | 8 | 6 | nie zna formatu gry, ale pisze lepszą polszczyzną; baza run 7 |

Z run 3, run 4 (seed 42) i run 5 (seed 42) zostały tylko liczby z historii. Pliki tych pomiarów zginęły razem z katalogiem tymczasowym, więc `compare.py` ich nie pokazuje. W `results/replays/` są odtworzenia z run 5, run 6 i 11B. Pliki `replay5*` pochodzą z jednorazowej próby nowych podpowiedzi tropu (stare i nowe brzmienie polecenia), pozostałe z `replay.py`.
