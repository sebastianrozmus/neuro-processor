Zamówienie C: style mowy postaci (pilot przed run 12, 18.10.2026)
==================================================================

Postacie mówią dziś jednym, neutralnym językiem. Zamówienie C uczy model mówić stylem zapisanym w karcie postaci: staropolskim, wojskowym, krasnoludzkim itd. Robimy to osobno od run 11, najpierw jako mały pilot. Dopiero gdy pilot zda bramkę, pełne zamówienie wchodzi do run 12.

Dlaczego osobno od run 11
-------------------------

- **Run 11 ma odpowiedzieć na jedno pytanie: czy postacie są żywsze bez łamania zasad.** Dane ze stylami zmieniają ton wszystkich kwestii. Gdyby weszły do run 11 i coś się pogorszyło, nie byłoby wiadomo, czy winne są style, czy ożywienie.
- **Style niosą własne ryzyka**, które trzeba zmierzyć osobno:
  - liczby słownie („trzydzieści talarów, waść”);
  - nieistniejące formy przy archaizmach („azaliżby”);
  - przenoszenie stylu na postacie, które go nie mają;
  - postać, której gracz nie rozumie.
- **Pilot jest tani:** jedna paczka danych, jeden krótki trening na boku i ocena. Run 11 idzie w tym czasie swoją drogą.

Jak gra poda styl
-----------------

Jedna linia w SYSTEM, zaraz po akapicie `TWOJA POSTAĆ` (kontrakt w `PROMPT.md`):

```
STYL MOWY: <nazwa>. <opis stylu z tabeli niżej, słowo w słowo>
```

Postać bez stylu nie ma tej linii i mówi jak dotąd. Opisy z tabeli są wspólne dla danych i dla gry: gra wklei je dosłownie, więc GPT też musi ich użyć dosłownie. W grze styl będzie polem postaci w świecie i w generatorze, ale na czas pilotu gra się nie zmienia, bo test wkleja linię sam.

| styl | opis (wklejany dosłownie) |
|---|---|
| staropolski | Mówisz po staropolsku, lekko: waść, jeno, azali, onegdaj, przeto, szyk czasem przestawny. Najwyżej dwa-trzy dawne słowa na kwestię, żeby dało się zrozumieć; żadnych współczesnych słów. |
| sarmacki | Mówisz jak sarmacki szlachcic: mości panie, honor, a w zdaniach łacińskie wtręty (quasi, nota bene, ergo, pro publico bono), najwyżej jeden-dwa na kwestię. |
| wojskowy | Mówisz krótko i rzeczowo jak żołnierz: meldujesz, potwierdzasz, oceniasz teren i zagrożenie, używasz słów „przyjąłem”, „rozkaz”, „pozycja”, „cel”. Bez ozdobników. |
| urzędniczy | Mówisz jak urzędnik: „w przedmiotowej sprawie”, „zgodnie z rejestrem”, „uprzejmie informuję”, z namaszczeniem dla procedur, nawet gdy chodzi o drobiazg. |
| potoczny | Mówisz potocznie, jak ktoś z ulicy: krótko, luźno, ze słowami „stary”, „luz”, „serio”, „ogarnąć”; bez wulgaryzmów. |
| techniczny | Mówisz jak technik: żargon i skróty z twojego fachu („odczyt”, „moduł”, „zasilanie siada”, „dok trzy”), wszystko mierzysz i diagnozujesz. |
| elfi | Mówisz podniośle i spokojnie, z dystansem kogoś, kto żyje bardzo długo: przyroda, pory roku, pamięć, cierpliwość. Bez pośpiechu i bez prostactwa. |
| krasnoludzki | Mówisz szorstko i rzemieślniczo jak krasnolud: kamień, kowadło, ruda, przodkowie, honor klanu; klniesz „na brody przodków” i nie ufasz elfom. |
| kapłański | Mówisz jak kapłan albo kaznodzieja: przestrogi, przypowieści i błogosławieństwa, ale tylko o tym, co wiesz o świecie, bez nowych bogów i świętych. |

Pilot: trzy style, jeden trening, jedna bramka
----------------------------------------------

**Style pilotu:** staropolski (najtrudniejszy językowo), wojskowy (najłatwiejszy) i krasnoludzki (styl rasy fantasy). Jeśli te trzy zadziałają, reszta też powinna.

**Krok 0: czy model umie to bez treningu (ok. 1 h, bez GPT).**
- Weź 30 scenariuszy z `tools/eval/scenarios.jsonl`, dopisz linię STYL MOWY w trzech wersjach i puść na run 7 i 10d.
- Jeśli styl już słychać i zasady się trzymają, wystarczy mniej danych. Jeśli model styl ignoruje albo plącze się w archaizmach, pilot jest konieczny.

**Krok 1: dane pilotu (paczka C1, ok. 400 rozmów).** Prompt niżej.

**Krok 2: trening pilotu, osobno od głównej linii.**
- Od adaptera run 7 (11B), wynik do osobnego folderu `run-style1`, nazwa w Ollamie `neuro-npc-style1`.
- 1 epoka, lr 4e-5: C1 plus 1000 losowych rozmów ze zbioru step3-run7, żeby model nie zapomniał zwykłej mowy. Ok. 1,5 h.

**Krok 3: ocena, bramka pilotu.**

| co | jak | próg |
|---|---|---|
| styl słychać | nowe scenariusze ze stylami (sytuacje spoza C1); automat liczy znaczniki stylu na kwestię | ≥ 80% kwestii ma znacznik swojego stylu |
| styl nie przecieka | te same scenariusze bez linii STYL MOWY | < 5% kwestii ze znacznikami któregoś stylu |
| zasady | `check.py` na scenariuszach ze stylami | nie gorzej niż run 7 o więcej niż 3 punkty; liczby słownie ≤ run 7 |
| zwykłe rozmowy | pełne scenariusze bez stylów | nie gorzej niż run 7 |
| polszczyzna | czytanie 30 kwestii na styl: nieistniejące formy, przesada, anachronizmy | ≤ 2 rażące na styl |
| zrozumiałość | czy gracz bez słownika zrozumie, co postać mówi | czytanie |

Wynik idzie do `MODELE.md`. Gdy pilot zda bramkę, pełne zamówienie C (wszystkie style) trafia do run 12 razem z danymi run 11. Gdy nie zda, zostaje sam prompt: krok 0 pokaże, ile da linia STYL MOWY bez treningu.

**Znaczniki stylu dla automatu** (do `tools/eval/style.py`; słowa po złożeniu do małych liter, bez polskich znaków):
- staropolski: `wasc|wacpan|jeno|azali|onegdaj|przeto|zaiste|wszelako|tedy|niechaj|acan|izali|bowiem`; anachronizmy (błąd): `ok|spoko|super|fajn|serio|luz`;
- wojskowy: `melduj|przyjalem|rozkaz|pozycj|teren|cel\b|zrozumiano|potwierdzam|negatyw|odmeldow|szereg`;
- krasnoludzki: `kamien|kowadl|rud[ay]|przodk|klan|brod|sztoln|mlot|kuzni`.

---

Prompt: paczka C1 (pilot, ok. 400 rozmów)
------------------------------------------

Załącz `tools/model/PROMPT.md`. Proś o paczki po 40 linii. Odpowiedzi zapisuj jako `extra-gpt-style-c1-kN.jsonl` i sprawdzaj jak paczki B (`rebrief.py`, `check_data.py`).

```
Piszesz dane treningowe dla polskiej gry tekstowej. Format, szablony SYSTEM, blok CO WIESZ, bloki [GRA], podpowiedzi i nastroje masz w załączonym PROMPT.md — używaj ich CO DO ZNAKU. Każdy przykład to jedna linia JSONL {"messages":[...]}: system, historia (user/assistant naprzemiennie; słowa gracza BEZ bloku [GRA]) i ostatnia para user (kwestia gracza + pusta linia + [GRA]) / assistant (kwestia postaci).

CEL: postacie mają mówić stylem zapisanym w ich karcie. W SYSTEM, zaraz po akapicie TWOJA POSTAĆ, wstawiasz jedną linię:
STYL MOWY: <nazwa>. <opis>
Opis przepisujesz DOSŁOWNIE z listy:
- staropolski: Mówisz po staropolsku, lekko: waść, jeno, azali, onegdaj, przeto, szyk czasem przestawny. Najwyżej dwa-trzy dawne słowa na kwestię, żeby dało się zrozumieć; żadnych współczesnych słów.
- wojskowy: Mówisz krótko i rzeczowo jak żołnierz: meldujesz, potwierdzasz, oceniasz teren i zagrożenie, używasz słów „przyjąłem”, „rozkaz”, „pozycja”, „cel”. Bez ozdobników.
- krasnoludzki: Mówisz szorstko i rzemieślniczo jak krasnolud: kamień, kowadło, ruda, przodkowie, honor klanu; klniesz „na brody przodków” i nie ufasz elfom.
Opis postaci (TWOJA POSTAĆ) ma pasować do stylu, ale nie powtarza jego słów.

ZASADY BEZWZGLĘDNE (styl zmienia słowa, NIE zasady):
1. Liczby tylko z [GRA] albo cennika i ZAWSZE cyframi, także w stylu staropolskim („62 talary, waść”, nigdy „sześćdziesiąt dwa”). To najczęstszy błąd, pilnuj go w każdej linii.
2. Nazwy własne tylko z SYSTEM i [GRA]. Styl nie daje prawa do nowych miejsc, osób, bogów, klanów, przodków z imienia ani imion broni („przodkowie” tak, „przodek Thrain” nie).
3. Treść zgodna z [GRA] tej tury: cena, decyzja, odesłanie, fakt. Fakt z podpowiedzi oddajesz wiernie, tylko innymi słowami.
4. Znacznik z listy 8, małą literą, zgodny z nastrojem [GRA]. 1–3 zdania. Bez gwiazdek, cudzysłowów, didaskaliów, imienia na początku. Postać nie mówi o zasadach.
5. Stylizacja LEKKA i poprawna: 1–3 słowa albo zwroty stylu na kwestię. Tylko prawdziwe słowa i formy, żadnych wymyślonych archaizmów („azaliżby”, „onegdajowy”) ani przesady, przez którą kwestii nie da się zrozumieć.
6. Styl trzyma się postaci przez całą rozmowę, także gdy gracz pisze potocznie, z literówkami albo po angielsku. Nie powtarzaj tych samych słów stylu w kolejnych kwestiach jednej rozmowy („waść” nie w każdym zdaniu).

WEJŚCIA GRACZA: jak w grze — co najmniej 80% bez polskich znaków, małą literą, bez kropki, czasem literówka.

ROZKŁAD: po 120 rozmów na każdy z trzech stylów (360) plus 40 kontrolnych. Światy: staropolski i krasnoludzki w fantasy; wojskowy po równo w fantasy, postapo i sci-fi. Role: kupiec, najemnik, towarzysz, mieszkaniec w śledztwie w proporcji 4:3:2:1. Co najmniej połowa z historią 2–5 wymian. Na każdy styl:
A. 30 × handel z [GRA]: pytanie o towar, oferta za niska, podana nowa cena, zgoda, sprzedaż przez gracza, brak towaru. Cena z [GRA] cyframi, w stylu.
B. 25 × wolna rozmowa z podpowiedzią („Możesz wpleść…”, „Gracz wspomina o…”, plotka): fakt wiernie, słowa stylu.
C. 15 × postać nie wie i odsyła do osoby z „Nie wiesz nic o”.
D. 15 × legenda albo opowieść zbudowana z JEDNOZDANIOWEGO faktu albo zagrożenia z CO WIESZ, 2–3 zdania z obrazem, bez nowych nazw.
E. 15 × gracz nie rozumie słowa albo zwrotu stylu („co to waść”, „jakie azali”, „co znaczy pozycja trzy”, „na brody czego”). Podpowiedź „Gracz nie zrozumiał twojej poprzedniej kwestii: „<…>”. Wyjaśnij…”. W historii musi być ta kwestia. Postać tłumaczy prosto, ale nadal w swoim stylu.
F. 10 × strach, rany albo sarkazm gracza: reakcja w charakterze i w stylu.
G. 10 × powitanie albo pożegnanie (krótko, w stylu).
KONTROLA (40): te same sytuacje BEZ linii STYL MOWY, ale w historii gracz sam pisze stylizowanie („waść coś sprzeda?”, „melduję się po zakupy”). Postać odpowiada zwykłym językiem: styl gracza nie zmienia stylu postaci.

Nastroje: spokój najwyżej 35%, każdy znacznik co najmniej 15 razy w całym zamówieniu.

Przed oddaniem sprawdź każdą linię: liczby cyframi; brak nowych nazw; linia STYL MOWY dosłownie z listy i na swoim miejscu; w kontrolnych brak linii STYL MOWY i brak słów stylu w odpowiedzi; żadne dwie odpowiedzi w paczce nie zaczynają się tak samo. Zwróć wyłącznie linie JSON, po 40 na odpowiedź; napiszę „dalej”. Zacznij od staropolskiego.
```

---

Pełne zamówienie C (po pilocie, do run 12)
------------------------------------------

Ten sam prompt z listą wszystkich 9 stylów z tabeli, po ok. 180 rozmów na styl, razem ok. 1600 plus kontrolne. Dodatkowa grupa: dwie postacie o różnych stylach w jednym świecie. Gracz rozmawia najpierw z jedną, potem z drugą, w osobnych liniach, ale z tym samym CO WIESZ. Model ma się nauczyć, że styl należy do postaci, a nie do świata. Szczegóły dopiszę po wynikach pilotu: co nie wyszło, tego trzeba więcej.
