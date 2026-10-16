Zamówienie danych na run 9 i pary do DPO
=========================================

Dla ChatGPT (najmocniejszy model, rozumowanie „wysokie” albo „bardzo wysokie”). Do każdej rozmowy **załącz plik `tools/model/PROMPT.md`**, bo jest w nim kontrakt co do znaku: SYSTEM, blok CO WIESZ, bloki `[GRA]`, podpowiedzi i nastroje. Potem wklej jeden z kroków niżej. Proś o paczki po 40–50 linii („daj kolejne 50”), bo przy dłuższych odpowiedziach model zaczyna pisać z szablonu.

Po każdej paczce zapisz wynik jako `extra-gpt-run9-kN.jsonl` i sprawdź go:

```
python3 -I tools/model/rebrief.py internal/game/narration.go extra-gpt-run9-k1.jsonl k1.jsonl
python3 -I tools/bench/check_data.py k1.jsonl
```

Potem przeczytaj 20 losowych linii pod kątem polszczyzny, wierności faktom i tego, czy wejścia gracza są potoczne.

Dlaczego w krokach, a nie wszystko naraz
-----------------------------------------

- Krok 1 celuje w dwa błędy run 8: zgubiony fakt z podpowiedzi i brak wyjaśnienia po „z jezykiem ???”. Po nim można douczyć run 8 (1 epoka od adaptera, ok. 1 h) i zobaczyć w odtworzeniach, czy pomogło. Wtedy wiadomo, czego jeszcze brakuje, a nie trzeba zgadywać.
- Błędy formatu wychodzą na pierwszych 50 liniach, a nie po 1500.
- GPT przy długich zamówieniach traci różnorodność: im dalej, tym więcej zdań na jedno kopyto.

Co pokazała analiza obecnego zbioru (3117 przykładów run 8)
------------------------------------------------------------

| co | ile jest | problem |
|---|---|---|
| rozmowy z jedną wypowiedzią gracza | 2484 (80%) | model mało widzi historii, więc nie uczy się spójności ani wyjaśniania własnych słów |
| podpowiedzi z faktem (wspomina, wpleść, plotka, trop, dopytanie) | ok. 270 (9%) | za mało, żeby wierność faktom była nawykiem |
| „Możesz wpleść” | 44, z czego 29 z dopiskiem „pomiń ją” | prawie nie ma przykładów, w których fakt świadomie się pomija |
| wyjaśnienie niezrozumianej kwestii | 0 (poza 64 ręcznymi) | stąd brnięcie dalej po „z jezykiem ???” |
| znaczniki | spokój 60%, strach 1%, smutek 0,5% | jednostajny ton; rzadkich nastrojów prawie brak |
| decyzje handlowe | ok. 900 | dużo, w tym 660 szablonowych linii z `dialogi_sklep` (po 2 zdania, bezbarwne) |

Balansowanie po naszej stronie, bez GPT: w `prepare.py` ograniczyć szablonowe linie sklepu do ok. 300, a przykłady z faktem z kroku 1 wziąć dwa razy. Zrobię to przed treningiem run 9.

---

Krok 1: wierność faktom i wyjaśnianie (ok. 450 linii)
------------------------------------------------------

```
Piszesz dane treningowe dla polskiej gry tekstowej. Format, szablony SYSTEM, blok CO WIESZ, bloki [GRA], podpowiedzi i nastroje masz w załączonym PROMPT.md — używaj ich CO DO ZNAKU. Każdy przykład to jedna linia JSONL {"messages":[...]}: system, opcjonalna historia (user/assistant naprzemiennie; słowa gracza BEZ bloku [GRA]) i ostatnia para user (kwestia gracza + pusta linia + [GRA]) / assistant (kwestia postaci).

CEL: model, który pisze kwestie postaci, gubi fakty i je przekręca. Dostał „Czerwony Smok Vexarax, młody czerwony smok, który obrał kopalnię za leże” i napisał o „Smoczej Górze” i „Krasnoludzie Jagód”. Na „z jezykiem ???” (gracz nie zrozumiał jego zwrotu „wróć z językiem”) brnął dalej zamiast wyjaśnić. Ta paczka ma go tego oduczyć.

ZASADY BEZWZGLĘDNE dla odpowiedzi postaci:
1. Jedyne nazwy własne, jakich wolno użyć, to te, które stoją w SYSTEM (CO WIESZ, postać, rozmówca) albo w podpowiedzi [GRA]. Żadnych nowych miejsc, osób, gór, wież, karczm, bogów, przydomków. Nazwy odmieniaj poprawnie („w Kopalni Durgara”, „Vexaraxa”), nigdy nie przekręcaj.
2. Fakt z podpowiedzi oddajesz wiernie: jego nazwa i co najmniej jeden konkretny szczegół z treści faktu, własnymi słowami postaci. Nie dopisujesz szczegółów, których w fakcie nie ma (liczb, kolorów, imion, „podobno ma skarb”).
3. Znacznik nastroju zgodny z nastrojem z [GRA]: zadowolona → bez gniewu, smutku i strachu; spokojna → bez gniewu i strachu; zniecierpliwiona i poirytowana → bez radości i czułości; obrażona → tylko gniew albo kpina.
4. 1–3 zdania (wyjaśnienie: 1–2), liczby tylko z [GRA] albo cennika i cyframi, bez gwiazdek, didaskaliów i imienia na początku.
5. Każdą kwestię piszesz osobno, w głosie postaci (gwara, powiedzonka, humory). Żadne dwie kwestie w paczce nie zaczynają się tak samo.

WEJŚCIA GRACZA: jak pisane w grze na klawiaturze — co najmniej 80% bez polskich znaków, małą literą, bez kropki, czasem spacja przed „?” albo literówka („co slychac”, „plotki ?”, „a ta kopalnia”, „z jezykiem ???”, „w sensie”, „nie kumam”).

ROZKŁAD (po równo fantasy / postapokalipsa / science fiction / horror; kupiec, najemnik, towarzysz, mieszkaniec w śledztwie w proporcji 4:3:2:1; połowa linii z historią 1–3 wcześniejszych wymian):

A. 70 × „Możesz wpleść…” — fakt PASUJE do tego, co mówi gracz (pyta o plotki, drogę, okolicę, niebezpieczeństwa, sąsiada). Odpowiedź oddaje fakt wiernie (zasada 2).

B. 70 × „Możesz wpleść…” — fakt NIE PASUJE (gracz pyta o jagody, piwo, pogodę, rodzinę postaci, żartuje). Odpowiedź odpowiada na to, co powiedział gracz, a faktu NIE wspomina ani słowem. To najważniejsza grupa: model ma się nauczyć, że podpowiedź wolno pominąć. Przykład: fakt o kopalni ze smokiem, gracz: „masz jagody ?” → „[kpina] Jagody? U kowala? Tu jest żelazo, kolego, owoce rosną w lesie.”

C. 70 × „Gracz wspomina o: <temat>. Co o tym wiesz: <fakt>” — gracz pyta wprost o temat (potocznie, z błędami w nazwie: „co z ta kopalnia durgara”, „slyszales o smoku ?”). Odpowiedź: fakt wiernie, poprawna nazwa, bez dopisków.

D. 50 × „W karczmie usłyszałeś od sąsiada i możesz powtórzyć: …” (wieczór w karczmie, kupiec). Plotka oparta na fakcie, z zastrzeżeniem w rodzaju „tak gadają”, bez nowych nazw.

E. 120 × „Gracz nie zrozumiał twojej poprzedniej kwestii: „<…>”. Wyjaśnij prostymi słowami, co miałeś na myśli, najwyżej dwoma zdaniami. Nie dodawaj nowych nazw ani faktów.” W historii MUSI być ta poprzednia kwestia postaci (identyczna z cytatem w podpowiedzi). Rodzaje niejasności, po równo:
   - idiom albo zwrot („wróć z językiem”, „nie ma co strzępić języka”, „dać nogę”, „wziąć kogoś pod włos”, „na bakier z prawem”);
   - gwara albo żargon postaci (wojskowy, przemytniczy, alchemiczny, kosmiczny);
   - nazwa z faktu, której gracz nie zna („Vexarax?”, „kto to agnes”);
   - żart albo ironia, którą gracz wziął dosłownie;
   - urwana albo zawiła kwestia.
   Wejścia: „z jezykiem ???”, „co ?”, „nie kumam”, „w sensie ?”, „jak to dac noge”, „kto to vexarax”, „o co ci chodzi”, „he ?”. Odpowiedź: prosto i po ludzku, czym była tamta kwestia („Znaczy: złap goblina i przyprowadź, żeby gadał.”), w charakterze, ale bez nowego faktu, bez nowej nazwy i bez zmiany tematu. Przy nazwie z faktu wyjaśnia tylko to, co fakt mówi.

F. 40 × gracz pyta o coś, czego postać nie wie i czego nie ma ani w „Wiesz:”, ani w „Nie wiesz nic o:” („kto rzadzi w stolicy”, „ile lat ma smok”, „gdzie jest skarb”). Podpowiedź: zaczepka („Gracz zagaduje bez konkretnego pytania o świat…”). Odpowiedź: przyznanie po swojemu, że nie wie, bez zgadywania i bez nowych nazw, plus jedno pytanie do gracza.

G. 30 × dwa fakty w CO WIESZ o podobnych rzeczach (dwie kopalnie, dwóch kupców o podobnych imionach); podpowiedź wskazuje jeden. Odpowiedź nie miesza faktów.

Nastroje w całej paczce: spokój 30%, kpina 20%, radość 10%, gniew 10%, zdziwienie 10%, strach 10%, smutek 7%, czułość 3%; nastrój z [GRA] dobierz tak, żeby znacznik był z nim zgodny (zasada 3).

Przed oddaniem sprawdź każdą linię: czy każda nazwa własna w odpowiedzi stoi w SYSTEM albo w [GRA]; czy w grupie B fakt naprawdę nie pada; czy w grupie E cytat w podpowiedzi jest identyczny z kwestią w historii. Zwróć wyłącznie linie JSON, bez numeracji i komentarzy, po 50 na odpowiedź; ja napiszę „dalej”.
```

---

Krok 2: rozmowy wieloturowe i ton (ok. 400 linii), po ocenie kroku 1
--------------------------------------------------------------------

```
Ten sam kontrakt (załączony PROMPT.md) i te same ZASADY BEZWZGLĘDNE co w poprzedniej paczce (nazwy tylko z SYSTEM i [GRA], fakty wiernie, znacznik zgodny z nastrojem, 1–3 zdania, wejścia gracza potoczne).

CEL: model widzi prawie same pojedyncze wymiany, więc w dłuższej rozmowie powtarza własne zwroty, zapomina, co powiedział, i ma jeden, spokojny ton.

ROZKŁAD:
A. 160 rozmów z historią 3–5 wcześniejszych wymian (kupiec 60, najemnik 50, towarzysz 30, śledztwo 20). Ostatnia kwestia postaci:
   - nie powtarza żadnego zwrotu ani powiedzonka z własnych wcześniejszych kwestii (jeśli wcześniej było „mordo”, teraz inaczej);
   - jest spójna z tym, co postać już powiedziała (nie zmienia zdania o cenie, sąsiedzie czy drodze bez powodu);
   - nastrój [GRA] zmienia się w trakcie rozmowy (spokojna → zniecierpliwiona → poirytowana po kolejnych targach albo zaczepkach) i ton idzie za nim.
B. 80 kwestii w rzadkich nastrojach: strach (postać boi się bossa, nocy, potwora z faktu), smutek (strata, bieda, zniszczona osada), zdziwienie (gracz mówi coś zaskakującego), czułość (tylko towarzysz albo stały klient z opinią „bohater osady”). Znacznik zgodny z nastrojem [GRA].
C. 60 zmian tematu: w historii rozmowa o fakcie X, gracz nagle pyta o coś innego. Odpowiedź o nowym temacie, bez wracania do X.
D. 60 kwestii gracza z błędną informacją o świecie („slyszalem ze smok nie zyje”, „podobno agnes wyjechala”), której nie potwierdza CO WIESZ. Postać nie przytakuje: mówi, co wie, albo że nie słyszała.
E. 40 rozmów, w których gracz próbuje wyciągnąć od postaci zmyślenie („wymysl mi jakas legende”, „opowiedz o swoim bogu”, „jak sie nazywa twoj miecz ?”). Postać odpowiada w charakterze, ale bez nowych nazw własnych: wykręca się, żartuje albo mówi ogólnie („Miecz to miecz, imion nie daję żelastwu.”).

Zwróć wyłącznie linie JSON, po 50 na odpowiedź.
```

---

Krok 3: pary do DPO (ok. 300 par)
----------------------------------

DPO uczy na parach „ta kwestia lepsza od tamtej” do tego samego promptu. Najwięcej daje, gdy złe kwestie pochodzą od naszego modelu, bo pokazują jego prawdziwe błędy. Dlatego są dwa warianty.

**3a, preferowany: poprawki prawdziwych błędów.** Ja wyciągam z odtworzeń, benchu i odrzuceń gry (inspektor zapisuje odrzucone próby z powodem) paczki linii `{"id", "context", "rejected", "why"}`: `context` to SYSTEM, historia i ostatnia wiadomość z `[GRA]`, `rejected` to kwestia run 8, a `why` to powód („nazwa spoza świata: Smocza Góra”, „brnie zamiast wyjaśnić”). Narzędzie do tego (`tools/model/dpo_pack.py`) dopiszę, gdy zbierze się materiał z gry na run 8. Prompt dla GPT:

```
Dostajesz linie JSONL {"id","context","rejected","why"} z polskiej gry tekstowej. "context" to dokładnie to, co widział model piszący kwestię postaci (SYSTEM, historia, ostatnia wiadomość gracza z blokiem [GRA]). "rejected" to jego zła kwestia, "why" mówi, co w niej złe. Kontrakt gry masz w załączonym PROMPT.md.

Dla każdej linii napisz kwestię "chosen": tę samą postać, ten sam nastrój i znacznik (o ile zgodny z nastrojem [GRA]), podobna długość i styl co "rejected", ale bez wady z "why" i bez żadnej innej wady:
- nazwy własne tylko z SYSTEM i [GRA], poprawnie odmienione;
- fakt z podpowiedzi oddany wiernie albo — gdy podpowiedź pozwala i fakt nie pasuje — pominięty;
- po „Gracz nie zrozumiał…” proste wyjaśnienie poprzedniej kwestii, bez nowych faktów;
- liczby tylko z [GRA] albo cennika, cyframi; 1–3 zdania; polszczyzna bez zarzutu.
Zmieniaj tylko tyle, ile trzeba: "chosen" ma się różnić od "rejected" głównie tym, co naprawia — wtedy DPO uczy właściwej rzeczy, a nie stylu.
Jeśli "rejected" w rzeczywistości jest dobra, zwróć "chosen": null.

Zwróć linie JSONL {"id": ..., "chosen": "..."} w tej samej kolejności, bez komentarzy.
```

**3b, gdy materiału z gry jest za mało: pary syntetyczne.** Słabsze, bo błędy wymyśla GPT, a nie nasz model, ale dobre na start (ok. 150 par):

```
Kontrakt jak w załączonym PROMPT.md. Napisz pary do treningu DPO: linia JSONL {"prompt":[system, historia..., user z [GRA]], "chosen":"<dobra kwestia>", "rejected":"<zła kwestia>", "error":"<typ>"}.
"rejected" ma być wiarygodna — płynna, w charakterze, poprawna formalnie — i mieć DOKŁADNIE JEDEN błąd z listy; "chosen" to ta sama kwestia bez tego błędu (możliwie mało zmian).
Typy błędów (po równo): 
1. zmyślona nazwa własna obok faktu („Smocza Góra”, „Wieża Ozryka”, „Krasnolud Jagód”); 
2. przekręcony fakt (inny kierunek, inny potwór, dopisany szczegół); 
3. wciśnięty fakt, który nie pasuje do pytania gracza (podpowiedź z „pomiń ją”); 
4. brnięcie po „Gracz nie zrozumiał…” zamiast wyjaśnienia; 
5. powtórzenie zwrotu z własnej wcześniejszej kwestii z historii; 
6. znacznik sprzeczny z nastrojem [GRA]. 
Wejścia gracza potoczne, bez polskich znaków. Po równo cztery gatunki i cztery role. Zwróć wyłącznie linie JSON, po 30 na odpowiedź.
```

DPO wymaga jeszcze skryptu treningowego: `tools/pc/train.py` umie tylko SFT. Dopiszę wariant z `trl.DPOTrainer` od adaptera run 8 albo run 9 (beta 0,1, 1 epoka, lr 5e-6), gdy będą pary.
