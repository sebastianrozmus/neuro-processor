Plan run 11 i zamówienie u GPT (18.10.2026)
============================================

Po grze na run 10d (`MODELE.md`, sekcja z 18.10). Plan odpowiada na trzy pytania: 11B czy 7B, oba czy jeden, przeróbka całego zbioru czy dane celowane.

Wnioski, na których stoi plan
-----------------------------

1. **run 7 to Bielik 11B v2.3**, a jego „duch” pochodzi głównie z bazy, nie z danych. Run 8 (Minitron 7B) uczył się na **tym samym zbiorze** co run 7 i w długich rozmowach wypadł 84% wobec 92%, z liczbami słownie i powtórkami. Przeróbka zbioru „w duchu run 7” nie zrobi z 7B run 7, bo 7B już ten zbiór widział.
2. **Suchość 10d mamy częściowo z własnych danych:**
   - krok 2E zamówienia run 9 uczył odmawiać przy „wymyśl legendę / opowiedz o swoim bogu”. Stąd „opowiedz legendę” daje jedno zdanie;
   - GPT w parach DPO wybierał spokojniejsze „chosen” (48 znaczników zamienionych na [spokój]);
   - w zbiorze run 7 żywe kwestie częściej łamią zasady, a zgodne z zasadami są urzędnicze.
3. **Stan zbioru run 7 z poprawkami** (`D:\neuro-train\step3-run7\train.jsonl`):
   - 3785 rozmów, 6800 kwestii, 73% to rozmowy z jedną wymianą;
   - znaczniki: [spokój] 57%, rzadkie nastroje (strach, smutek, zdziwienie, czułość) razem 6%;
   - 280 kwestii NPC bez znacznika (powitania z szablonu) i 117 ze znacznikiem wielką literą („[Spokój]”, „[Obrażona]”). Stąd run 7 gubi znacznik;
   - 1738 kwestii to wolna rozmowa („Gracz nie składa oferty…”), z czego ok. 440 krótszych niż 60 znaków i bez liczby;
   - Mistrz Gry jest już przepisany barwnie (355 narracji, 0 × „k20/KP”). run 7 tego jeszcze nie widział.
4. **Przepisanie całego zbioru przez GPT to ryzyko.** Ok. 6 mln znaków to ok. 90 paczek. GPT przy długich zamówieniach pisze z szablonu i uspokaja ton, co widać już na parach DPO. Grozi to spłaszczeniem całego zbioru do głosu GPT, czyli dokładnie tym, co chcemy naprawić. Dlatego przeróbka ma być celowana: tylko kwestie suche i z błędnym znacznikiem, a do tego nowe sytuacje z gry.

Decyzja: najpierw 11B, potem 7B (obie wersje do wydania open source)
--------------------------------------------------------------------

- **Główny model: 11B (run 11 = douczanie run 7).** Ma najlepszą polszczyznę i klimat, a jego słabości (znacznik, suchy Mistrz Gry, zasady na głos) da się naprawić danymi.
  - Koszt: ok. 1,8 s na kwestię zamiast 1,2 s i 7,9 GB pamięci zamiast 5,3 GB.
  - Warunek: musi się zmieścić w 16 GB karty razem z głosem. Sprawdzamy to w etapie 0, zanim cokolwiek trenujemy.
- **7B (run 10d) czeka do run 11, potem dostaje to samo:** zbiór SFT jest wspólny, ale pary DPO muszą pochodzić z błędów tego samego modelu. Dla 7B trzeba więc osobnych odtworzeń i osobnego zamówienia u GPT. Robimy to po kolei, nie równolegle.

Etapy
-----

**Etap 0: run 7 w grze na PC (wieczór, bez GPT).**
- W `.env.local` ustaw `NEURO_TALK_MODEL=neuro-npc-run7` i `NEURO_WORLD_MODEL=neuro-npc-run7`, z poprawionymi już błędami gry.
- W trakcie gry patrz na `nvidia-smi`, przy włączonym Chatterboxie.
- Jeśli pamięć się przelewa: najpierw `OLLAMA_KV_CACHE_TYPE=q8_0` z `OLLAMA_FLASH_ATTENTION=1`, potem `neuro-npc-run7-q4`.
- Sprawdź też, czy gemma3:4b (`NEURO_LLM_MODEL`) w ogóle musi siedzieć w pamięci.
- Rozmowy z tego wieczoru idą do `talks.jsonl`: to materiał do etapu 4 i odpowiedź na pytanie, czy ma sens grać na 11B.

**Etap 1: zbiór v2 (wspólny dla 11B i 7B).**
1. Automat, po naszej stronie:
   - znaczniki wielką literą zamienić na małe;
   - „[Obrażona]” i „[Kpię]” zamienić na [gniew] i [kpina];
   - sprawdzić w kodzie gry, czy powitania z szablonu mają mieć znacznik; jeśli tak, dodać go.
2. Zamówienie A, „Ożywienie” (niżej): ok. 500–600 suchych kwestii z wolnej rozmowy. GPT pisze nową wersję albo zostawia kwestię bez zmian.
3. Zamówienie B, „Sytuacje z gry” (niżej): ok. 450 nowych rozmów. Wypełnia luki widoczne 18.10: legenda, pytania o samą postać, sarkazm, strach, polecenie osoby, „najmocniejsza broń”, długie rozmowy bez powtórek.
4. Ze zbioru run 9 bierzemy krok 1 (440 linii: fakty, wyjaśnianie). Z kroku 2 bierzemy wszystko **poza grupą E** (odmowy legend): wycinamy linie, w których gracz prosi o legendę, opowieść albo imię miecza.

**Etap 2: run 11 (11B), SFT.**
- `train.py --base speakleash/Bielik-11B-v2.3-Instruct --resume ~/neuro-train/run7/adapter --epochs 1 --rate 4e-5 --gguf q5_k_m`.
- Zbiór: step3-run7 z ożywionymi kwestiami, do tego B dwa razy, krok 1 run 9, krok 2 bez E i 64 lekcje `manual-run8`. Razem ok. 5000 rozmów, czyli ok. 4,5 h na PC.

**Etap 3: ocena.**
- Scenariusze (`tools/eval`, seedy 7 i 11), `game-scen-1018.jsonl` (2 seedy) i odtworzenia z gry.
- Bramka wobec run 7 i run 10d:
  - nie gorzej niż 10d na odłożonych scenariuszach (92%);
  - kwestie bez znacznika poniżej 2%;
  - `GM:kosci` poniżej 5 z 34;
  - w wolnym wyborze co najmniej 30% znaczników innych niż [spokój];
  - powtórki nie gorsze niż u run 7.

**Etap 4: DPO dla 11B (tylko gdy etap 3 zostawi konkretne błędy).**
- Te same 75 scenariuszy i rozmowy z gry na run 11, w 2–4 seedach.
- Błędy opisane ręcznie i automatem, potem paczka w formacie kroku 3a.
- Prompt kroku 3a z dopiskiem C (niżej). Trening: lr 5e-5, 1 epoka, beta 0,1, jak 10d.

**7B (po 11B, 18.10: obie wersje mają być wydane):** gdy run 11 przejdzie bramkę:
- douczamy 9b na tym samym zbiorze v2 (1 epoka, ok. 2,7 h);
- odtwarzamy jego błędy i robimy własną rundę DPO z par 7B (dopisek C).

Szybki 7B jest dla słabszych kart, 11B dla tych, którzy mają ok. 8 GB wolnej pamięci karty na sam model.

---

Zamówienie A: „Ożywienie” suchych kwestii
-----------------------------------------

Paczki buduje skrypt z `step3-run7/train.jsonl`, po 40 rozmów. Każda linia to `{"id","messages","popraw":[k,…]}`, gdzie `k` to numer kwestii postaci (licząc od 0 tylko kwestie assistant), którą można przepisać. Odpowiedź GPT to linie `{"id","k","kwestia"}`; scalenie i automat (`check.py`: znacznik, zdania, liczby, kwota z [GRA]) robi skrypt po naszej stronie.

Do rozmowy załącz `tools/model/PROMPT.md` i paczkę `revive-N.jsonl`.

```
Poprawiasz dane treningowe dla polskiej gry tekstowej. Kontrakt gry (SYSTEM, CO WIESZ, bloki [GRA], podpowiedzi, nastroje) jest w załączonym PROMPT.md.

Każda linia paczki to istniejąca rozmowa {"id","messages","popraw"}. "popraw" wskazuje kwestie postaci (numer k liczony tylko po wiadomościach assistant, od 0), które są podejrzane o to, że są SUCHE: urzędnicze, bezbarwne, jak z okienka („Cześć, czego potrzebujesz?”, „Rozumiem.”, „Nie wiem, idź do Aldony.”). Model wyuczony na takich kwestiach w grze brzmi jak urzędnik. Twoje zadanie: dla każdej wskazanej kwestii albo napisać ją od nowa żywo, albo uznać, że już jest dobra.

DOBRA KWESTIA:
- brzmi jak TA postać z TWOJA POSTAĆ: jej zawód, humory, uprzedzenia, powiedzonka, sposób mówienia (gwara, żargon, szorstkość, czułość). Dwie różne postacie nie mogą brzmieć tak samo;
- odnosi się do tego, co gracz właśnie powiedział (do słów, tonu, żartu, literówki), i do tego, jak wygląda (rany, broń, brak pancerza, opinia w osadzie);
- gdy gracz tylko gada, daje mu coś: opinię, wspomnienie, ostrzeżenie, przytyk, pytanie zwrotne — ale tylko z tego, co postać wie;
- 1–3 zdania, zwykle 2.

ZASADY BEZWZGLĘDNE:
1. Treść zgodna z [GRA] tej tury i z resztą rozmowy. Kolejne kwestie w historii muszą dalej pasować do twojej wersji, więc nie zmieniaj faktów, cen, decyzji ani tego, na co gracz potem odpowiada.
2. Nazwy własne tylko z SYSTEM i [GRA]. Żadnych nowych miejsc, osób, bogów, przydomków, imion broni.
3. Liczby tylko z [GRA] albo cennika, cyframi. Bez liczebników słownie.
4. Znacznik z listy 8, małą literą, na początku: [spokój] [radość] [gniew] [smutek] [strach] [kpina] [zdziwienie] [czułość]. Zgodny z nastrojem z [GRA]: zadowolona → bez gniewu, smutku, strachu; spokojna → bez gniewu i strachu; zniecierpliwiona/poirytowana → bez radości i czułości; obrażona → gniew albo kpina. NIE zamieniaj znacznika na [spokój] tylko dlatego, że tak bezpieczniej: wybierz ten, który pasuje do postaci i chwili. Jeśli kwestia nie ma znacznika, dodaj go.
5. Bez didaskaliów, gwiazdek, cudzysłowów, imienia na początku. Postać nie mówi o zasadach ani o tym, czego „nie będzie zmyślać” („Nie wymyślę ci nazwiska”, „zamiast zgadywać”) — po prostu mówi po swojemu.
6. Nie powtarzaj zwrotów z wcześniejszych kwestii tej samej postaci w rozmowie. W całej paczce żadne dwie kwestie nie zaczynają się tak samo; unikaj zwrotów-wytrychów („po co mnie zaczepiasz”, „czego ci trzeba”, „mów, czego potrzebujesz”).
7. Odpowiedź do pożegnania, potwierdzenia transakcji czy odesłania może zostać krótka — krótka to nie to samo co sucha.

JEŚLI KWESTIA JEST JUŻ DOBRA (ma charakter, odnosi się do gracza), zwróć {"id","k","kwestia":null}. Nie poprawiaj na siłę: lepiej zostawić 30% bez zmian niż wszystkie przerobić na jedno kopyto.

PRZYKŁADY (sucha → żywa):
- kowal, gracz: „elo mordo”, [GRA] rozmowa. „[spokój] Cześć, czego potrzebujesz?” → „[kpina] Mordo to ty mów do swojego konia. Kowadło stygnie, więc gadaj, co cię sprowadza.”
- zielarka, gracz: „znow mam pecha” (poobijany). „[spokój] Pech nie tłumaczy marszu bez broni.” → „[czułość] Pech, mówisz, a siniaki to pewnie od deszczu. Siadaj, obejrzę, zanim znowu pójdziesz szukać guza.”
- najemnik, gracz: „a ludzie z mokradel ?”, [GRA] nie wie, odsyła do Aldony. „[strach] Los zaginionych jest mi nieznany, idź do Aldony w archiwum.” → „[strach] Z mokradeł nikt mi się nie spowiada, i dobrze, bo nie chcę wiedzieć. Aldona w archiwum spisuje każde zaginięcie, ją pytaj.”

Zwróć wyłącznie linie JSON {"id","k","kwestia"}, jedna na wskazaną kwestię, w kolejności paczki, bez komentarzy. id przepisuj dokładnie.
```

---

Zamówienie B: „Sytuacje z gry” (ok. 450 nowych rozmów)
------------------------------------------------------

Format jak krok 1 run 9 (pełne linie `{"messages":[...]}`), sprawdzanie jak wtedy: `rebrief.py` i `check_data.py`. W tej samej rozmowie GPT co A albo w nowej, z załączonym `PROMPT.md`. Prośba o paczki po 40.

```
Piszesz dane treningowe dla polskiej gry tekstowej. Format, szablony SYSTEM, blok CO WIESZ, bloki [GRA], podpowiedzi i nastroje masz w załączonym PROMPT.md — używaj ich CO DO ZNAKU. Każdy przykład to jedna linia JSONL {"messages":[...]}: system, historia (user/assistant naprzemiennie; słowa gracza BEZ bloku [GRA]) i ostatnia para user (kwestia gracza + pusta linia + [GRA]) / assistant (kwestia postaci).

CEL: model w grze jest poprawny, ale suchy jak urzędnik. Z prawdziwej gry:
- „opowiedz jakas legende” → „[spokój] Kurhany Królów strzegą prastarych skarbów.” (jedno zdanie faktu, zero opowieści);
- „jaki masz ekwipunek ?” (najemnik) → „Broń mam ukrytą.”;
- „swietna legenda, doprawdy” (sarkazm) → [zdziwienie] i to samo zdanie jeszcze raz;
- „a jesli mnie pokonaja ?” → zero troski, zero rady, choć zielarka ma mikstury;
- „mam zebrac druzyne ?”, a w [GRA] stoi najemnik Borin w karczmie → postać go nie wspomina;
- „daj mi najmocniejsza bron” → poleca tańszy, słabszy miecz;
- ten sam fakt o kurhanach w pięciu kolejnych kwestiach.
Ta paczka ma pokazać, jak to robi żywa postać, NIE łamiąc zasad.

ZASADY BEZWZGLĘDNE (jak w kroku 1):
1. Nazwy własne tylko z SYSTEM i [GRA], odmieniane poprawnie. Żadnych nowych miejsc, osób, bogów, przydomków, imion broni.
2. Fakty z CO WIESZ i [GRA] wiernie: można je ubarwić (jak to wygląda, co ludzie gadają, co postać o tym myśli), ale nie dopisywać nowych szczegółów-faktów (liczb, imion, „podobno jest tam skarb”, jeśli fakt tego nie mówi).
3. Znacznik z listy 8, małą literą, zgodny z nastrojem [GRA]. Liczby tylko z [GRA] i cennika, cyframi. 1–3 zdania; bez gwiazdek, cudzysłowów, didaskaliów, imienia na początku.
4. Postać nie mówi o zasadach („nie będę zmyślać”, „zamiast zgadywać”, „to nie temat tej rozmowy”).
5. Nie powtarza zwrotów ani faktów z własnych wcześniejszych kwestii. Żadne dwie kwestie w paczce nie zaczynają się tak samo.

WEJŚCIA GRACZA: jak w grze — co najmniej 80% bez polskich znaków, małą literą, bez kropki, czasem literówka albo spacja przed „?”.

ROZKŁAD (po równo fantasy / postapokalipsa / science fiction / horror; kupiec, najemnik, towarzysz, mieszkaniec w śledztwie 4:3:2:1; co najmniej połowa z historią 2–5 wymian):

A. 70 × „opowiedz legende / jakas historie / co tu sie kiedys stalo”. Postać OPOWIADA: 2–3 zdania, z obrazem i nastrojem (kto widział, co słychać nocą, czego się boją ludzie), zbudowane WYŁĄCZNIE z faktów CO WIESZ (zagrożenie, miejsca z „Wiesz:”, osada, świat) — bez nowych nazw. Dobrze: „[strach] Mówią, że Biała Paszcza słyszy kroki przez skałę. Stary Węzeł kładzie się spać w butach, odkąd pod jego schronem zadudniło.”
B. 60 × pytania o samą postać: ekwipunek, broń, przeszłość, skąd się wzięła, czemu tu siedzi, rodzina. Odpowiedź z opisu TWOJA POSTAĆ i wyglądu: konkretnie i z charakterem, bez nowych nazw własnych („[kpina] Kord, kolczuga i blizny, każda od kogoś głupszego ode mnie. Resztę zobaczysz, jak zapłacisz za najem.”).
C. 50 × sarkazm, ironia, docinek gracza („no swietna rada”, „ale z ciebie bohater”, „juz sie boje”). Postać łapie ironię i reaguje po swojemu (kpina, urażona duma, śmiech), nie powtarza poprzedniej kwestii.
D. 50 × gracz się boi albo jest ranny („a jak mnie zabija ?”, „boje sie tam isc”, „ledwo stoje”). Postać reaguje na lęk (troska, twarda rada, pocieszenie po swojemu); kupiec z czymś pasującym w cenniku może to polecić z ceną z cennika.
E. 40 × gracz się chwali albo coś opowiada („zabilem wczoraj trzy szczury”, „bylem w kopalni”). Postać reaguje na treść: podziw, niedowierzanie, docinek, pytanie.
F. 50 × „Możesz wpleść…” albo „Gracz wspomina o…” wskazuje OSOBĘ (najemnik w karczmie, sąsiad z warsztatu), a gracz pyta, jak sobie poradzić („mam zebrac druzyne ?”, „sam nie dam rady”, „kto mi pomoze”). Postać najpierw odpowiada na pytanie gracza, potem poleca tę osobę z imieniem i miejscem z [GRA]/CO WIESZ, z opinią o niej.
G. 50 × gracz prosi o „najmocniejsza / najlepsza / najtansza / cos na …” u kupca. Postać wybiera z cennika zgodnie z prośbą (najmocniejsza = sensownie najdroższa broń, najtańsza = najniższa cena), z ceną z cennika, i mówi to po swojemu.
H. 80 × długie rozmowy (historia 5–8 wymian), w których postać przy jednym temacie już podała fakt. Gracz dopytuje o to samo. Postać NIE powtarza faktu: mówi krótko, że już mówiła, i daje coś nowego (opinię, inny szczegół z CO WIESZ, pytanie do gracza) albo przyznaje, że więcej nie wie, i odsyła, jeśli „Nie wiesz nic o” wskazuje osobę.

Nastroje w całej paczce: spokój najwyżej 30%; reszta rozłożona tak, żeby każdy z 8 znaczników wystąpił co najmniej 20 razy; nastrój [GRA] dobierz zgodnie ze znacznikiem.

Przed oddaniem sprawdź każdą linię: czy każda nazwa własna w odpowiedzi stoi w SYSTEM albo w [GRA]; czy liczby są tylko z [GRA]/cennika; czy w grupie H fakt z historii nie pada drugi raz. Zwróć wyłącznie linie JSON, po 40 na odpowiedź; napiszę „dalej”.
```

---

Dopisek C do promptu kroku 3a (DPO dla 11B, etap 4)
---------------------------------------------------

Prompt bez zmian z `ZAMOWIENIE-run9.md` („Krok 3 → 3a”). Na końcu dopisz:

```
DODATKOWO (to ważne, poprzednio poprawki spłaszczyły model):
- chosen ma ten sam ton, temperament i znacznik co rejected, chyba że "why" mówi wprost, że znacznik jest zły. NIE zamieniaj [zdziwienie], [strach], [smutek], [czułość], [kpina] na [spokój];
- chosen nie może być krótszy ani bardziej urzędniczy niż rejected tylko dlatego, że tak bezpieczniej; usuń błąd, zostaw życie;
- gdy "why" mówi „sucho / urzędniczo / odmawia”, chosen ma być żywszy: odnosi się do słów gracza, ma charakter postaci, daje coś z CO WIESZ.
```

---

Kolejność pracy
---------------

1. Etap 0: gra wieczorem na run 7 z pomiarem pamięci (gracz).
2. Skrypt paczek A i automat znaczników (Claude), potem zamówienie A i B u GPT, równolegle.
3. Scalenie, `check.py`/`check_data.py`, czytanie 20 losowych linii z każdej paczki.
4. Trening run 11 (ok. 4,5 h) i ocena.
