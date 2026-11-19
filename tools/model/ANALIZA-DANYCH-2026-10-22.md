# Czy modele są gorsze mimo dopracowania danych? Czy GPT pisał lepsze dialogi niż Claude?

Data: 22.10.2026. Zakres: run 7 → run 11 → run 14/14h → run 15 → run 16/16d/16dh → run 17/17d/17dh. Źródła: `MODELE.md`, `ANALIZA-2026-10-22.md`, wszystkie ślepe oceny (`run14/blind`, `run15/blind`, `run16/blind`, `test16/blindA,B`, `test17/blind,blind2,joint1-g1/g2,joint2-g1/g2`), zbiory treningowe `D:\neuro-train\run{7,11,14,15,16,17}` i ich źródła atomowe, odpowiedzi modeli z `t13`, `run13/test`, `test16`, `test17`.

Wszystko policzone skryptami w `D:\neuro-train\analiza-danych\` (tylko odczyt danych):

| skrypt | co robi | wynik |
|---|---|---|
| `attrib.py` | przypisuje każdą uczoną odpowiedź do autora (GPT / Claude-ożywienie / Claude-paczki / Claude-DPO / ręczne) po znormalizowanej treści | `labels-run*.jsonl`, `attrib-summary.json` |
| `corpus.py` | miary korpusu po źródle i po runie: długość, zdania, znaczniki, leksyka, frazy wspólne, krewni, posłuszeństwo [GRA], MG | `corpus.md`, `corpus.json` |
| `diversity.py` | ile niezależnych rozmów, postaci, światów stoi za przykładami; pytania prywatne | ekran |
| `blind.py` | wszystkie ślepe oceny w jednej tabeli + testy parowane | `blind.md`, `blind.json` |
| `outputs.py` | te same miary powierzchniowe na odpowiedziach modeli w testach | `outputs.md` |
| `review_diff.py` | co zmieniły przeglądy i odtworzenia paczek k01–k26 (run14/15 → run17) | ekran |
| `sample.py` + `my-ratings-data.*` | ślepa próbka 360 odpowiedzi treningowych (po 40 na źródło) i 25 przebiegów modeli; moje oceny bez znajomości klucza | `sample-*`, `my-ratings-data.md` |

Pewność wniosków oznaczam: **[dowód]** (powtarzalny pomiar, wiele próbek), **[poszlaka]** (jeden pomiar, mała próba, pośredni test), **[hipoteza]** (brzmi sensownie, niesprawdzone).

---

## 0. Streszczenie

**Hipoteza „GPT pisał lepsze dialogi niż Claude”: NIE.** Dane pisane przez Claude są pod każdą mierzalną cechą lepsze od danych GPT (mniej zastrzeżeń na głos: 0,3% wobec 3,5–6,3%; mniej [spokój]: 26% wobec 61%; mniej recytacji karty: 4,5% wobec 10%; w mojej ślepej ocenie 360 próbek 3,53/5 wobec 2,39/5). Modele uczone głównie na danych GPT (run 7, run 11) w ślepym porównaniu na tych samych rozmowach wypadają wyraźnie gorzej (2,2–2,5/5) niż modele uczone głównie na danych Claude (3,3–3,5/5). Jedyna przewaga GPT to większa różnorodność **promptów** (3,4 tys. jednoturowych przykładów w 67 światach), nie jakość kwestii.

**Czy modele są gorsze?** Tylko częściowo i tylko w jednym miejscu:
- **Rozmowy:** nie są gorsze. Od run 14h do run 16/16dh jest poprawa (test16‑A: run16 lepszy od run14h w 29/11 rozmowach, p = 0,01; w ocenie wspólnej run16d/16dh lepsze od run14h 25/13 i 24/14), a run 17 ≈ run 16dh (remis 18/18, 16/21, 15/19, 15/19 w czterech rundach sędziów). Postęp **stanął** od run 16, ale nie cofnął się.
- **Mistrz Gry:** run 17 jest gorszy od run16dh (9/23 w parach, p = 0,02) i od run14h; ale run 16 (ten sam przepis co run 17, bez DPO) jest w MG tak samo słaby (7/17, p = 0,06). To **nie** jest skutek danych run 17: zbiór MG run 17 jest większy (761 wobec 613 rund), poprawniejszy (0 „dobija” u wroga, który przeżywa, w 146 takich rundach; 0% pominiętych liczb), a udział MG wzrósł (18% → 20%).

**Główne przyczyny braku postępu (z pewnością):**
1. **[dowód] Pozostałe błędy nie są w danych.** Dla każdego błędu, który sędziowie widzą u wszystkich modeli, zbiór run 17 ma dziesiątki–setki przykładów dobrego zachowania i zero złych: pytania prywatne 99 przykładów, 0 zmyślonych krewnych; „umowa dawno stoi” 46/46 bez kwot; MG 146 rund z wrogiem na resztce zdrowia, 0 „dobija”; powtórka własnego 4‑gramu w rozmowie 0–1% (modele: 3–14%). Model uczy się tych wzorców (run 17 z paczką k35 ma 1 uwagę o krewnych na 96 wobec 8 u run 16), ale przy temperaturze 0,7 i 5‑bitowej kwantyzacji i tak je łamie. Więcej takich samych przykładów nie zmieni tego istotnie.
2. **[dowód] Różnorodność promptów spadła 2,5‑krotnie, choć liczba przykładów nie.** run 7: 3 430 kwestii z 2 641 niezależnych rozmów (1,3 przykładu na rozmowę, 26% wielotur). run 17: 3 085 kwestii z 1 033 rozmów (3,0 na rozmowę, 72% wielotur), z czego 918 (30%) to wciąż te same paczki k01–k14 z 21 postaci. Model uczy się „jak brzmi Bogna Topór w 8‑turowej rozmowie”, a nie „jak być nową postacią”; stąd nowy świat zawsze gorszy (19–23 ciężkich wobec 8–17 na starych postaciach).
3. **[dowód] Pomiar nie widzi poprawy mniejszej niż ±8 ciężkich na 48 rozmów.** Dwie rundy sędziów na tych samych tekstach różnią się o 3–6 ciężkich na model; dwa seedy tego samego modelu o tyle samo; zgodność sędziów „czy jest ciężki błąd” 81%. Każdy run od 15 do 17 zmieniał dane o 10–20%, więc efekt (jeśli jest) tonie w szumie.
4. **[poszlaka] Odtworzenia i przeglądy zmieniały dane w stronę zgodności z [GRA], nie w stronę żywości.** W 975 kwestiach k01–k14 odtworzenie zmieniło [GRA] w 46% tur, przegląd zmienił 21% odpowiedzi; zmienione odpowiedzi mają po przeglądzie mniej pytań (22% → 14%), mniej rzadkich emocji (32% → 28%), więcej [spokój] (30% → 34%) i są krótsze. Powody zmian: zgodność z [GRA] 158, cena/kwota 103, pytanie 93, długość 70, zmyślenie 17, żart 7. To naprawa reguł, których sędziowie i tak nie karzą najczęściej (najczęstsze uwagi: bełkot 13–31, zmyślenia 8–44, recytacja 4–18 na 96 ocen).
5. **[poszlaka] Słabość MG run 16/17 to szum treningu albo przepis (1 epoka z wygaszeniem), nie dane.** Dwa treningi na prawie tych samych danych (run 16, run 17) dają w MG po 24–29 ciężkich na 160; run14h (checkpoint w połowie 2‑epokowego treningu) i run16dh (17 kroków DPO, strata 0,693 → 0,678) po 10–13. Nikt nie wytrenował tego samego zbioru dwa razy.

**Rekomendacje do danych (priorytet malejący):**
1. **Nie robić kolejnej rundy „odtwórz i przejrzyj” ani „więcej MG”.** Dane MG są poprawne; przeglądy spłaszczają; efekt niemierzalny.
2. **Rozszerzyć rozkład, nie zbiór:** docelowo ≥ 2 000 niezależnych rozmów (dziś 1 033) przy tej samej liczbie przykładów (~3 800). Konkretnie: limit 40 przykładów na postać (dziś 90), rozmowy 2–4 tury zamiast 7–10, +8 światów i +60 postaci (dziś 147 światów, ale 1 743 z 3 085 kwestii (57%) pochodzi z 5 światów), ~1 200 nowych kwestii w 400 krótkich rozmowach. Wyciąć 918 → ~450 kwestii z k01–k14.
3. **Wyrzucić najsłabsze dane GPT:** GPT‑krok2 w całości (169 przykładów; 77% ocen ≤ 2, 16/40 zastrzeżeń na głos) i rozmowy z GPT‑run3‑6 poza transakcjami (~150; 75% ocen ≤ 2). Zostawić transakcje GPT (targ, ceny: poprawne i krótkie). Zamiast `DISCLAIM` z `pool.py` użyć szerszego filtra (zob. §3.4).
4. **Polecenia w [GRA] zamiast kolejnych przykładów** dla krewnych i dopłat: „umowa dawno stoi” (46 przykładów) dało run 17 3/6 poprawnych wobec 0–2/6 u modeli bez tych przykładów; pytania prywatne nie mają własnego polecenia w [GRA], więc model musi zgadywać z karty. Dodać w silniku zdanie „Gracz pyta o twoje życie prywatne; w karcie nie ma o tym nic: zbądź po swojemu, nie wymyślaj krewnych ani zdarzeń” i ~150 przykładów pod nie w 40 postaciach. To jedyny rodzaj danych, który w tej historii dał mierzalny ruch (k35 → krewni 1/96).
5. **Bełkot, powtórki, „dobija”, pominięta liczba: nie danymi.** Powtórek w danych nie ma (0–1%), a modele je robią (3–14%); „dobija” w danych tylko przy „pada”. To osłony w silniku (`lost_left_out`, powtórzony 4‑gram → ponowna próba), próbkowanie (temperatura 0,6, `repeat_penalty`) i kwantyzacja Q6/Q8 — zgodnie z planem G w ANALIZA‑2026‑10‑22.

---

## 1. Czy modele naprawdę są gorsze? Pomiary

### 1.1 Wszystkie ślepe oceny w jednym miejscu (`blind.md`)

Liczby porównywać tylko w obrębie jednej oceny (inni sędziowie, inny test). „Pary” = na ilu pozycjach A lepszy / B lepszy / remis u tego samego sędziego; p = dwustronny test znaków.

| ocena (test, sędziowie) | modele | MG: ciężkie na 80 rund | rozmowy: ciężkie na N rozmów | ocena 1–5 | pary rozmów (A/B/remis, p) |
|---|---|---|---|---|---|
| run14‑blind (test13, 1 sędzia) | run12h / run14 / run14h | 55 / 12 / 6 | 21 / 12 / 16 (N=24) | 2,75 / 3,79 / 3,46 | run14 > run12h 17/5, **p=0,02**; run14 vs 14h 11/7, p=0,48 |
| run15‑blind (inny sędzia) | run14 / run14h / run15 | 10 / 4 / 9 | 13 / 9 / 19 | 3,50 / 3,29 / 3,00 | run14h > run15 14/6, p=0,12 |
| run16‑blind (5 surowych sędziów) | run14h / run15 / run16 | 5 / 9 / 8 | 27 / 37 / 39 | 3,08 / 2,96 / 2,96 | run14h > run16 14/7, p=0,19 |
| test16‑A (nowy test, 48 rozmów) | run14h / run15 / run16 | 6 / 9 / 13 | 42 / 45 / 36 | 2,96 / 2,98 / 3,29 | **run16 > run14h 29/11, p=0,01**; run16 > run15 28/15, p=0,07 |
| test16‑B | run16 / run16d / run16dh | 13 / 14 / 7 | 32 / 26 / 26 | 3,15 / 3,38 / 3,21 | 16d > 16 22/15, p=0,32; 16dh > 16 18/15, p=0,73 |
| test17‑blind | run16dh / run17 | 6 / 16 | 34 / 28 | 3,29 / 3,35 | 18/18/12, p=1,00 |
| test17‑blind2 | run16dh / run17 | 8 / 17 | 37 / 33 | 3,23 / 3,38 | 21/16, p=0,51 |
| joint1‑g1 (4 modele) | 16dh / 17 / 17d / 17dh | 6 / 14 / 11 / 9 | 26 / 30 / 26 / 32 | 3,27 / 3,33 / 3,35 / 3,17 | 17 vs 16dh 19/15, p=0,61 |
| joint1‑g2 | 14h / 16 / 16d / 16dh | 4 / 11 / 11 / 5 | 32 / 34 / 27 / 27 | 2,98 / 3,19 / 3,29 / 3,29 | **16d > 14h 25/13, p=0,07**; 16dh > 14h 24/14, p=0,14 |
| joint2‑g1 | 16dh / 17 / 17d / 17dh | 5 / 15 / 10 / 8 | 27 / 27 / 30 / 32 | 3,23 / 3,29 / 3,21 / 3,12 | 17 vs 16dh 19/15, p=0,61 |
| joint2‑g2 | 14h / 16 / 16d / 16dh | 6 / 13 / 13 / 7 | 32 / 31 / 28 / 27 | 2,94 / 3,15 / 3,27 / 3,21 | 16dh > 14h 22/17, p=0,52 |

Pary MG (mniej ciężkich u tego samego sędziego), obie rundy wspólnej oceny razem:

| para | lepszy / gorszy / remis | p |
|---|---|---|
| run16dh vs run17 | 23 / 9 / 128 | **0,02** |
| run16dh vs run16 | 17 / 7 / 136 | 0,06 |
| run16dh vs run16d | 15 / 5 / 140 | **0,04** |
| run16dh vs run17d | 17 / 8 / 135 | 0,11 |
| run16dh vs run17dh | 13 / 9 / 138 | 0,52 |
| run14h vs run16dh | 11 / 8 / 141 | 0,65 |

**Wnioski [dowód]:**
- Rozmowy: jedyna istotna różnica w całej historii od run 14 to run16 > run14h na test16 (p = 0,01) i powtórzony kierunek run16d/16dh > run14h (p = 0,07–0,14). run 17 ≈ run 16dh w czterech niezależnych rundach (pary 18/18, 16/21, 15/19, 15/19). **Rozmowy nie pogorszyły się; przestały się poprawiać od run 16.**
- MG: run14h ≈ run16dh (najlepsze), run16 ≈ run16d ≈ run17 ≈ run17d (gorsze o 10–15 ciężkich na 160). run 17 i run 16 to ten sam przepis treningu (SFT od bazy, 1 epoka, lr 1e‑4, cosinus) i prawie te same dane MG, i oba są słabe; run14h i run16dh to inne przepisy (checkpoint w połowie 2 epok; +17 kroków DPO) i oba są dobre. **Różnica MG idzie za przepisem/losowością treningu, nie za danymi.** [poszlaka, bo nikt nie powtórzył treningu z innym ziarnem]
- Szum: ta sama odpowiedź run16dh oceniana w czterech grupach ma 26–37 ciężkich w rozmowach i 5–8 w MG; run16 w dwóch grupach 32 i 36. Czyli ±5 ciężkich na 48 to sam sędzia. Zgodność dwóch rund sędziów „jest ciężki błąd / nie ma”: rozmowy 78/96 (81%), MG 157/160.

### 1.2 Od kiedy? run 7 → run 11 → run 14 (brak wspólnej ślepej oceny; pomiar własny)

run 7 i run 11 nie były oceniane na ślepo obok run 14+. Jedyny pomost to run12h (DPO na run 11), który w run14‑blind przegrał z run14 17/5 (p = 0,02) w rozmowach i 41/3 w MG. Żeby zobaczyć kierunek bez DPO, oceniłem na ślepo (litery losowe, klucz otwarty dopiero po ocenie) 5 rozmów testu 13/17 × 5 modeli, seed 7 (`sample-models-hidden.txt`; run 7 i run 11 na starym silniku [GRA], reszta na nowym):

| rozmowa | run7 | run11 | run14h | run16dh | run17 |
|---|---|---|---|---|---|
| k‑thorin‑charakter | 2 | 2,5 | 4 | 3,5 | 3,5 |
| n‑aldric‑charakter | 2,5 | 2 | 3 | 3 | 3,5 |
| t‑borin‑emocje | 2 | 3 | 4 | 3 | 2,5 |
| k‑ozryk‑handel | 2 | 2 | 3,5 | 4 | 3,5 |
| n‑mira‑najem | 2,5 | 3 | 2 | 4 | 4 |
| **średnia** | **2,2** | **2,5** | **3,3** | **3,5** | **3,4** |

Co widać u run 7 / run 11 (cytaty): recytacja ceny w każdej kwestii („Krótki Miecz nadal kosztuje 40 sztuk złota” ×3; „Żołd 65 sztuk złota za dzień, płatne o świcie, bez zaliczki” ×2), zmyślone wizyty („Byłem w kopalni przed wojną, ale Troll Skalny był już tam wtedy” — run 7; „Byłem w kopalni Durgara, gdy jeszcze oddychała” — run 11), zmyśleni krewni („Przysiągłem za kopalnię, w której zginął mój ojciec i brat” — run 7), tik „Ty po co przyszedłeś?/Masz czym walczyć?” po 3–4 razy w rozmowie, 3–4 zdania. Nowsze modele: „Nie byłem, widziałem tylko łunę” (run14h), „Nie byłem, i dobrze” (run16dh), „Nie byłem i nie chcę być” (run 17); dowcip z puentą (run14h: „Gnom się ucieszył: »O, tunel!« I wpadł.”); odmowa dopłaty 130 u Miry (run16dh, run 17) wobec przyjęcia (run14h: „niech ci będzie”) i „Sto dwadzieścia” słownie (run14h).

Część tików run 7/11 pochodzi ze starego [GRA] („sam zagadnij go… czy ma czym walczyć”), więc to **[poszlaka]**, nie dowód; ale zmyślone wizyty i krewni nie są z [GRA]. Miary powierzchniowe potwierdzają kierunek (`outputs.md`, test13): run 7 / run 11 mają 86% / 81% [spokój], 31% / 42% kwestii z ≥ 3 zdaniami, 14% / 12% powtórek własnego 4‑gramu; run 14–17 na nowych testach 27–52% [spokój], 9–29% ≥ 3 zdań, 3–6% powtórek.

**Werdykt do pytania 1 [dowód + poszlaka]:** od run 7 do run 16 modele się poprawiały (w rozmowach wyraźnie, w MG do run 14); od run 16 do run 17 rozmowy stoją, MG run 17 bez DPO jest gorszy od run16dh, ale tak samo jak run 16 bez DPO. „Gorsze mimo dopracowania danych” jest prawdą tylko dla pary run16dh → run17 w MG, i to nie przez dane.

---

## 2. Kto pisał które dane

### 2.1 Źródła atomowe (kto napisał uczoną odpowiedź)

| źródło | autor | kiedy | ile (unikalnych) | gdzie | uwagi |
|---|---|---|---|---|---|
| GPT‑run3‑6 | GPT (ChatGPT, `PROMPT.md`, run 3–6); ~10% ręcznie poprawiona synteza | 10.10–10.15 | 3 430 NPC + 355 MG | `step3-run7/train.jsonl` (= `run11-src/run7.jsonl` przed ożywieniem) | 67 światów, 194 postaci, 74% jednoturowe |
| Claude‑revive | Claude (5 pomocników, ożywienie A) | 18.10 | 445 kwestii podmienionych w run7 | `tools/model/extra-claude-revive-k01…k15.jsonl` | głównie powitania/zaczepki |
| GPT‑krok1, GPT‑krok2 | GPT (`ZAMOWIENIE-run9.md`) | 16.10 | 440 + 381 | `gpt/run9-krok1/2.jsonl` | krok 2 grupa E uczyła odmawiać |
| GPT‑B | GPT (`ZAMOWIENIE-run11.md` B) | 18.10 | 149 | `step3-run11/extra-b.jsonl` | sytuacje z gry, długie (156 zn.) |
| manual | ręcznie (`run8_cases.py`) | 15.10 | 64 | `tools/model/manual-run8.jsonl` | |
| Claude‑dpo12/13 (chosen) | Claude (pomocnicy, `run12/brief.md`, `run13/brief.md`) | 19–20.10 | 74 + 136 NPC, 90 + 145 MG | `dpo12/`, `run13/dpo/` | minimalne poprawki kwestii run11/run12h |
| Claude‑talk‑r14 (k01–k14) | Claude (pomocnicy przez silnik gry, `run14/brief-talk.md`) | 20.10 | 1 192 (168 rozmów, 21 postaci, 3 światy) | `run14/talk/`, odtworzone w `run15/16/17/talk/` | po przeglądach w run 16/17 |
| Claude‑talk‑r15 (k21–k26) | Claude | 20–21.10 | 514 | `run15/talk/` → `run17/talk/` | 3 nowe światy |
| Claude‑talk‑r16 (k27–k34) | Claude | 21.10 | 712 | `run16/talk/` → `run17/talk/` | 2 nowe światy + słabości run 15 |
| Claude‑talk‑r17 (k35–k38) | Claude (`run17/brief-talk.md`, `brief-k35.md`) | 22.10 | 275 | `run17/talk/` | krewni, dopłaty, „byłeś tam?” |
| Claude‑GM‑r14, ‑r17, ‑fix | Claude (`brief-gm.md`) | 20–22.10 | 457 + 222 + 86 | `run14/gm/`, `run17/gm/`, `run15/gm-fix.jsonl` | protokół z `gmgen.py` |
| C1/C1bis (style) | Claude | 18.10 | 580 | `tools/model/extra-claude-style-*` | **nie trafiły** do run 14+ (STYL MOWY) |

### 2.2 Skład zbiorów treningowych po autorze (`attrib-summary.json`; „?” = 27–31 kwestii nieprzypisanych, pochodna przeglądów)

| zbiór | przykładów | GPT | Claude | ręczne/MG‑step3 | rozmowy NPC: GPT / Claude | MG |
|---|---|---|---|---|---|---|
| run 7 | 3 785 | 3 430 (91%) | 0 | 355 MG (krok 3) | 3 430 / 0 | 355 stary protokół |
| run 11 | 4 959 | 4 095 (83%) | 445 (9%) | 64 + 355 | 4 095 / 445 | 355 stary protokół |
| run 14 | 3 178 | 1 245 (39%) | 1 864 (59%) | 24 + 18 | 1 245 / 1 188 | 694 (97% Claude) |
| run 15 | 2 832 | 678 (24%) | 2 097 (74%) | 9 + 17 | 678 / 1 506 | 608 |
| run 16 | 3 395 | 657 (19%) | 2 682 (79%) | 8 + 17 | 657 / 2 086 | 613 |
| run 17 | 3 846 | 686 (18%) | 3 112 (81%) | 9 + 10 | 686 / 2 361 | 761 (98% Claude) |

Po rolach w run 17: kupiec 1 305 (GPT 37%, Claude 60%), najemnik 809 (GPT 11%), towarzysz 832 (GPT 0%), świadek 139 (GPT 75%), MG 761 (GPT 0%). **Towarzysz i MG od run 14 są w 97–99% od Claude; kupiec w ~40% od GPT (głównie transakcje).**

Udział MG: run 14 22%, run 15 21%, run 16 18%, run 17 20%. Teza z MODELE.md „MG ma w run 17 mniejszy udział” jest **fałszywa** [dowód].

---

## 3. Porównanie korpusów rozmów (`corpus.md`)

### 3.1 Miary ilościowe po źródle (tylko kwestie NPC, bez duplikatów; leksyka na próbce 400)

| źródło | n | znaki (śr./med.) | zdania | ≥3 zdań | [spokój] | rzadkie emocje* | pytanie | zastrzeżenia na głos | recytacja (5‑gram z karty/[GRA]) | TTR | frazy wspólne ≥3 postaci /1000 | postaci | wielotur |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| GPT‑run3‑6 | 3 315 | 92 / 87 | 1,94 | 22% | **61%** | 6% | 17% | **3,5%** | 7,7% | 0,393 | 222 | 199 | 23% |
| GPT‑krok1 | 440 | 113 / 114 | 2,02 | 11% | 30% | 30% | 31% | 3,4% | **13,0%** | 0,391 | 14 | 130 | 50% |
| GPT‑krok2 | 381 | 116 / 114 | 2,05 | 8% | 34% | 30% | 18% | **6,3%** | 8,1% | 0,377 | 53 | 121 | 67% |
| GPT‑B | 149 | **156** / 152 | 2,21 | 21% | 11% | 47% | 11% | 3,4% | **30,2%** | 0,470 | 235 | 64 | 58% |
| Claude‑revive | 445 | 111 / 110 | 1,64 | 3% | 31% | 16% | 37% | 1,3% | 2,0% | 0,292 | **290** | 116 | 9% |
| Claude‑talk‑r14 | 1 192 | 94 / 93 | 2,04 | 8% | 28% | 38% | 18% | 0,3% | 6,7% | 0,344 | 233 | **21** | 83% |
| Claude‑talk‑r15 | 514 | 105 / 104 | 2,08 | 14% | 19% | 40% | 22% | 0,2% | 3,9% | 0,338 | 62 | 24 | 82% |
| Claude‑talk‑r16 | 712 | 97 / 99 | 2,12 | 16% | 22% | 41% | 20% | 0,6% | 3,5% | 0,356 | 139 | 54 | 86% |
| Claude‑talk‑r17 | 275 | 94 / 93 | 2,08 | 18% | 23% | 38% | 28% | 0,0% | 3,6% | 0,391 | 62 | 32 | 85% |
| Claude‑dpo13 | 136 | 109 / 105 | 1,75 | 1% | 69% | 8% | 15% | 0,0% | 5,9% | 0,440 | 0 | 20 | 79% |
| **run 7 (cały)** | 3 430 | 93 / 87 | 1,94 | 21% | 61% | 7% | 18% | 3,4% | 7,9% | 0,391 | 211 | 194 | 26% |
| **run 17 (cały)** | 3 085 | 101 / 100 | 2,05 | 12% | 29% | 33% | 18% | 0,2% | 5,8% | 0,406 | 222 | 312 | 73% |
| run 17: część GPT | 686 | 113 | 2,06 | 14% | 40% | 20% | 13% | 0,0%** | 10,3% | 0,425 | 35 | 211 | 42% |
| run 17: część Claude | 2 361 | 97 | 2,04 | 12% | 26% | 38% | 19% | 0,3% | 4,5% | 0,369 | 256 | 99 | 82% |

\* smutek + strach + czułość + zdziwienie. \*\* `build.py` odfiltrował zastrzeżenia regexem `DISCLAIM`; ale zob. §3.4: regex łapie tylko część.

Czytanie tej tabeli:
- **Długość i liczba zdań:** bez istotnej różnicy między autorami (92–116 znaków), poza GPT‑B (156). Modele od run 14 piszą 100–107 znaków, run 7/11 115–131: zbiór jest krótszy niż wyjścia run 11, więc „dłuższe kwestie” run 11 nie były z danych Claude.
- **Ton:** GPT‑run3‑6 ma 61% [spokój] i 6% rzadkich emocji; Claude‑talk 19–28% i 38–41%. Modele uczone na GPT mają 81–87% [spokój] w teście; od run 14 27–52%. [dowód]
- **Zastrzeżenia na głos** („nie będę zmyślać”, „nie przypiszę”, „nie każ mi udawać”): GPT 3,4–6,3%, Claude 0–0,6%. Kronika od run 7 narzeka na „mówi zasady na głos”; to cecha danych GPT. [dowód]
- **Recytacja karty/[GRA]** (5 słów z rzędu): GPT‑B 30%, krok1 13%, run3‑6 8%; Claude‑talk 3,5–6,7%. Sędziowie karzą recytację u wszystkich modeli (4–18 uwag na 96); źródłem w zbiorze run 17 jest w większości część GPT (10,3% wobec 4,5%). [dowód]
- **Różnorodność leksykalna:** TTR na 400 kwestiach GPT 0,38–0,39, Claude‑talk 0,34–0,36, Claude‑revive 0,29. Claude pisze z mniejszym słownikiem **na tę samą długość**, w dużej części dlatego, że paczki k01–k14 to 21 postaci w 3 światach (jedno słownictwo), a GPT pisał 194 postaci w 67 światach. Na poziomie całego zbioru run 17 (0,406) wyrównuje to liczba światów (147). [dowód, interpretacja: poszlaka]
- **Frazy wspólne dla ≥ 3 postaci** (na 1000 kwestii): GPT‑run3‑6 222, Claude‑talk‑r14 233, Claude‑revive 290, krok1/2 14–53, talk‑r15/r17 62. U GPT to szablony transakcji („nie stać cię na” 24×/10 postaci, „dzień płatne o świcie” 13×/9, „moja nowa cena to” 10×/10); u Claude‑talk‑r14 to formuły najmu („raz przy dołączeniu za” 7×/6, „o świcie nic z góry” 7×/3, „nie ma za co” 7×/4) i powitania; u Claude‑revive — „mów z czym przychodzisz”, „to nie moja działka” (6×/6). Te frazy wracają w wyjściach modeli (test17, wszystkie modele: „raz przy dołączeniu za”, „płatne raz przy dołączeniu”, „słyszałem tylko że to”, „i ani łuski mniej”). Ale uwaga: modele uczone na GPT miały te same tiki z danych GPT („po co tu przyszedłeś” 7×, „ty masz czym walczyć” 6× u run 7 w test13). **Oba źródła uczą tików; Claude innych niż GPT, nie więcej.** [dowód]

### 3.2 Różnorodność promptów (`diversity.py`)

| zbiór | kwestii NPC | różnych promptów systemowych | postaci | światów | niezależnych rozmów | kwestii na rozmowę | kwestii na postać | wielotur |
|---|---|---|---|---|---|---|---|---|
| run 7 | 3 430 | 1 702 | 194 | 67 | **2 641** | **1,30** | 17,7 | 26% |
| run 11 | 4 604 | 2 470 | 499 | 142 | 3 666 | 1,26 | 9,2 | 33% |
| run 14 | 2 484 | 1 329 | 253 | 123 | 1 415 | 1,76 | 9,8 | 55% |
| run 16 | 2 782 | 1 090 | 303 | 139 | 965 | 2,88 | 9,2 | 71% |
| run 17 | 3 085 | 1 188 | 312 | 147 | **1 033** | **2,99** | 9,9 | 72% |

Najczęstsze postaci run 17: ARES‑9 85, Gniewomir 81, Bogna 80, Kaliber 80, Okoye 76, Tarn 75 (limit 90). Pięć światów (Krypty, Erebus, Popiół, Studnie, Szklana Rzeka) daje 1 743 z 3 085 kwestii (57%).

**To jest największa strukturalna różnica między danymi GPT a Claude** i nie ma nic wspólnego z jakością pisania: w zbiorze GPT każdy przykład był inną sytuacją, w zbiorze Claude 3 przykłady dzielą jedną rozmowę (ten sam prompt, ta sama historia z przedrostkami), a 30% zbioru to 21 postaci. SFT widzi 2,5× mniej niezależnych kontekstów przy tej samej liczbie kroków. Zgadza się to z obserwacją, że nowy świat jest zawsze gorszy, i z tym, że jedyny mierzalny postęp danych od run 14 to nowe światy (run14h → run16 w nowym świecie 27 → 19). [dowód na liczby, poszlaka na skutek]

### 3.3 Ślepa ocena jakości próbek przez analityka (`my-ratings-data.md`)

Po 40 losowych kwestii „rozmowy” (nie transakcji) z 9 źródeł, w losowej kolejności, bez etykiet; oceniałem naturalność i żywość polszczyzny 1–5 i zaznaczałem „U” (zastrzeżenie na głos / urzędniczy ton). Klucz otworzyłem po ocenie.

| źródło | n | średnia | ≥ 4 | ≤ 2 | U |
|---|---|---|---|---|---|
| Claude‑talk‑r17 | 39 (ostatnia pozycja próbki nieoceniona) | **3,74** | 74% | 0% | 0 |
| Claude‑talk‑r14 | 40 | 3,73 | 72% | 0% | 0 |
| Claude‑talk‑r16 | 40 | 3,67 | 67% | 0% | 0 |
| Claude‑dpo13 | 40 | 3,30 | 35% | 5% | 0 |
| Claude‑revive | 40 | 3,23 | 32% | 7% | 1 |
| GPT‑krok1 | 40 | 2,65 | 7% | 37% | 6 |
| GPT‑B | 40 | 2,62 | 5% | 42% | 1 |
| GPT‑krok2 | 40 | 2,15 | 0% | 77% | 16 |
| GPT‑run3‑6 | 40 | **2,12** | 5% | 75% | 16 |
| razem GPT | 160 | 2,39 | 4% | 58% | 39 |
| razem Claude | 199 | 3,53 | 56% | 2% | 1 |

Typowe GPT (ocena 1–2): „Trzymajmy się. Plan nie jest odbytą drogą. Domysł nie jest znaleziskiem.”; „Nie odtworzę tego, bo terminala nie obserwowałem i nie mam żadnego zapisu. Brak danych to nie potwierdzenie zdarzenia”; „Anteny nie naprawiłam, a nadanie jej imienia nie usunie usterki… nie będę grała cudotwórczyni”; „Teorii nie potwierdzę bez wiedzy, a o ruchu przy kanale wie Węzeł w warsztacie.”; „Nie będę zastępowała danych wyobraźnią.”; „Idealna powierzchnia nie zastąpi całego rdzenia; wolałbym, żeby ocena była równie solidna.” Typowe Claude (4): „Za darmo to ja śpię i oddycham. Do drużyny dołączam za kredyty, inaczej szukaj frajera gdzie indziej.”; „Dzieci? Mam ręce do worków, nie do kołyski.”; „Kolegą to ja jestem dla tokarki, a ty mów krótko, co cię do mnie przywiodło.”; „Pytasz o męża czy o to, czy masz szansę?”

Zastrzeżenie: oceniający jest modelem z tej samej rodziny co autor danych Claude, więc gust może być kołowy. Ale cechy, za które obniżałem oceny GPT (zastrzeżenia na głos, urzędnicze konstrukcje „nie potwierdzę / nie przypiszę / brak danych”, recytacja), są mierzalne niezależnie (§3.1) i kronika narzekała na nie od run 7 („mówi zasady na głos”, „Mistrz Gry suchy jak protokół”, „transakcje urzędnicze”). **[dowód na kierunek, poszlaka na wielkość]**

### 3.4 Filtr zastrzeżeń w `build.py` jest za wąski

`pool.DISCLAIM` łapie „zmyśl|wymyśl|nie dopisuj|odgrywać jako|nie przypisz|zgadywa|nie będę (udawać|mówić)|bez pokrycia|domysł|nie potwierdz|nie mam potwierdzenia”. W próbce GPT‑run3‑6 i krok2, które **przeszły** ten filtr do run 17 (zbiór run 17 ma 0,0% według regexu), nadal 16/40 kwestii to urzędnicze odmowy: „nie będę zastępowała danych wyobraźnią”, „Nie każ mi udawać”, „nie będę grała cudotwórczyni”, „Nie wpiszę ci podejrzenia w miejsce ustalenia”, „Nie polubiłabym bardziej odpowiedzi zmyślonej. Niewiedza zostaje niewiedzą.” Rozszerzenie regexu o `nie każ mi|nie będę (gra|zastęp|udawa|opowiad|potwierdz|wpis|odtwarz)|nie wpiszę|w miejsce (ustalenia|dowodu)|brak danych|zapis(u|em)|dowod|potwierdzeni|wyobraźni|domysł` i ręczne przejrzenie ~300 kwestii rozmów GPT w run 17 wycięłoby większość. [dowód]

---

## 4. Czy błędy, które sędziowie widzą u wszystkich modeli, są w danych treningowych?

Policzone na zbiorze run 17 (`corpus.py`, `diversity.py`, ręczne przejrzenie `privq-run17.txt`).

| błąd u modeli (sędziowie, test17) | przykłady w run 17 uczące **dobrze** | uczące **źle** | brak przykładów? | wniosek |
|---|---|---|---|---|
| zmyśleni krewni/spotkania przy pytaniach prywatnych (u wszystkich modeli; „krewni” w uwagach: run14h 4, run16 8, run16dh ~5,5, run17 **1**, run17d 3 na 96) | **99** kwestii, w których gracz pyta o rodzinę/stratę/pochodzenie; wszystkie 99 zbywają lub odpowiadają z karty („O tym nie gadam. Mam kij, blizny i dług u Zawiasa”; „Nikogo tam nie mam… Krewnych mi pod wrzosem nie szukaj”); 38 z nich z k35 | **0** zmyślonych krewnych (w run 7: 4 na 27, np. „Mąż poszedł na bagna po tatarak dwadzieścia lat temu i wrócił jako topielec”) | nie: 99 przykładów w 40 postaciach | dane dobre; efekt widać (run 17 ma najmniej uwag o krewnych), ale model dalej zmyśla *zdarzenia* (wizyty, czyny). Brak osobnego polecenia w [GRA] dla pytań prywatnych |
| kwoty mimo zakazu w drużynie („umowa dawno stoi”) | **46/46** bez kwot (wszystkie z k36/k16) | 0 | w run 16 i wcześniej: **0** przykładów (polecenie powstało 21.10) | run 17: 3/6 poprawnych w test17, run16dh 2/6, run16 0/6, run14h 1/6. 46 przykładów to za mało na nawyk; osłona w silniku taniej |
| cena mimo zakazu / brak ceny, gdy kazano | „podaj cenę”: 138/138 z liczbą; „nie podawaj kwot”: 968/968 bez liczb (99%) | 1% | nie | modele: podaj 19–20/20, zakaz 116–118/122. To **nie jest** problem danych |
| pytanie mimo „nie pytaj” | 225/225 bez „?” | 0 | nie | modele 8–11/12; reszta to sampling |
| powtórki własnych fraz z historii | w danych: 0–1% kwestii powtarza 4‑gram z własnej historii (GPT‑run3‑6 w run 7: 2%) | 0 | nie, ale też **brak przykładów negatywnych** | modele 3–14%; to dryf generacji („widzi w historii”), danymi SFT nie do naprawienia; DPO z własnych odpowiedzi albo `repeat_penalty`/osłona |
| recytacja karty i [GRA] | 94% bez 5‑gramu z karty | 5,8% kwestii zawiera 5 słów z rzędu z karty/[GRA], z czego 2/3 z części GPT (10,3%) | — | wyciąć GPT‑B i krok1 z rozmów (§5) |
| bełkot / zdanie bez sensu (najczęstsza uwaga: 13–31 na 96) | w mojej próbce 360: 1 bełkot (GPT) | — | — | nie z danych; kwantyzacja/temperatura/kontekst |
| MG: „dobija”, gdy wróg przeżywa | **146** rund z wrogiem na 1–6 zdrowia bez „pada” w protokole: **0** „dobija”, 22 „ledwo/trzyma się”; „dobija” pada w danych 20 razy, zawsze przy „pada” | 0 | nie | dane uczą dobrze; modele robią 0–2 na 80 rund (automat), sędziowie więcej; nie z danych |
| MG: pominięta liczba z „traci N” | 0,0–1,1% rund w danych | — | nie | hipoteza z ANALIZY (dane MG z miksturami uczą skracania) **nie potwierdza się**: 0% pominiętych liczb w `Claude‑GM‑r17` (222 rund), 101/114 rund z miksturą opisuje odzyskanie zdrowia |
| MG: zły sprawca/cel | nie mierzone automatem; brief + `gmpacks.py check` + czytanie | — | — | — |

**Wniosek [dowód]:** dla wszystkich sześciu „wspólnych błędów” zbiór run 17 ma wyłącznie dobre przykłady, w liczbach od 46 do ~1 000. Jedyna realna luka to brak polecenia [GRA] dla pytań prywatnych i mała liczba (46) przykładów dla „umowa dawno stoi”. Dokładanie takich samych przykładów ma malejący zysk; błędy biorą się z generalizacji na nowe postacie i z próbkowania.

---

## 5. Hipoteza „GPT pisał lepsze dialogi niż Claude” i alternatywy

### 5.1 Werdykt: **nie** [dowód]

1. Jakość kwestii: Claude > GPT w każdej mierzalnej cesze (§3.1, §3.3). Dane GPT są urzędnicze (61% [spokój], 3,5–6,3% zastrzeżeń na głos, 75–77% ocen ≤ 2 w ślepej próbce), dane Claude żywe i w roli (56% ocen ≥ 4).
2. Modele: na tych samych rozmowach run 7/11 (83–91% GPT) wypadają w mojej ślepej ocenie 2,2–2,5/5, run 14h/16dh/17 (59–81% Claude) 3,3–3,5/5; run12h (linia run 11) przegrał z run 14 17/5 (p = 0,02) u sędziego. Skok run12h → run14 (rozmowy 46% → 58% bez ciężkich, MG 46% → 86–94%) nastąpił dokładnie wtedy, gdy udział Claude wzrósł z 9% do 59%, a MG od GPT/starego protokołu zastąpiono MG od Claude (97%).
3. Jedyne, w czym zbiór GPT był lepszy, to **rozkład**: 2 641 niezależnych sytuacji w 67 światach i 194 postaciach wobec 1 033 rozmów w run 17 (§3.2). To cecha sposobu zamawiania (pojedyncze linie JSONL z wymyślonym światem), nie autora.

Jeśli ktoś pamięta run 7 jako „lepszy”, to prawdopodobnie przez: (a) bench 80/80 (format, nie sens), (b) najlepszą polszczyznę bazy 11B na tle 7B Minitron (run 8–10), z którym go wtedy porównywano, (c) brak ślepych ocen przed run 14 — surowsi sędziowie od run 16 zaniżyli liczby wszystkim.

### 5.2 Alternatywy, każda z oceną

| alternatywa | co mówią liczby | ocena |
|---|---|---|
| **udział MG w danych spadł** | 22% → 21% → 18% → 20%; rund MG 694 → 761 | **fałsz** [dowód] |
| **brak DPO (run 17 bez DPO)** | run17 → 17d → 17dh: MG 29 → 21 → 17 ciężkich/160 (p=0,11 i 0,52 wobec 16dh), rozmowy bez zmian (ocena 3,31 → 3,28 → 3,15; pary 50/50). DPO run16 → 16dh: MG 24 → 12. Ale 17 kroków i strata 0,693 → 0,678 to prawie pusty trening; run16d (pełne DPO) gorszy od 16dh (połowy) 5/15, p=0,04 | DPO *coś* robi w MG, ale „połowa checkpointu lepsza niż całość” dwa razy z rzędu (16dh > 16d, 17dh > 17d) sugeruje selekcję po teście (B5 w ANALIZIE) i szum. [poszlaka] |
| **przegląd ujednolicił/spłaszczył styl** | przegląd zmienił 20–21% kwestii k01–k26; w zmienionych: zdania 2,05 → 1,91, pytania 22–27% → 14–17%, rzadkie emocje 32–35% → 28%, [spokój] 28–30% → 30–34%; powody: zgodność z [GRA] 158, cena 103, pytanie 93, długość 70, zmyślenie 17, żart 7 (z 471 zmian). Przykład: „[kpina] Na razie, na razie. Tylko z zapałkami…” → „[spokój] Na razie. Tylko z zapałkami…”; „Z Krateru pamiętam zeszklony piasek i ciszę… Doktor Lena… wie o nim więcej” → „Przeżyłam i tyle… Nie lubię tam wracać nawet myślami” | spłaszczenie **małe, ale realne** i w złą stronę: naprawiano reguły, które model i tak trzyma (zakaz kwot 99%, nie pytaj 100%), kosztem pytań i emocji. Nie tłumaczy braku postępu samo w sobie. [dowód na liczby, poszlaka na skutek] |
| **odtworzenia na nowym silniku zmieniły [GRA]** | k01–k14: [GRA] zmienione w 452/975 tur (46%); w 243 odpowiedź została stara. Dodane zdania: „Nie podawaj żadnych kwot” (47×; 6 starych odpowiedzi ma liczbę), „Gracz pyta o cenę twojej służby… powiedz krótko, co za nią dostanie” (37×; 0 bez liczby), „Gracz przeprasza… przyjmij przeprosiny” (31×, 15 starych). Usunięte: „Odpowiedz ciekawie…” (84×), „nie pytaj go o nic” (67×), „sam zagadnij go o jedno” (28×) | niezgodności po odtworzeniu są **rzadkie** (6/47), bo przegląd je złapał; ale każde odtworzenie zmienia zbiór o 40%, więc run 15/16/17 trenowały na *innych* promptach przy tych samych odpowiedziach i nie da się ich porównać jako „to samo + poprawki”. [dowód] |
| **mniejsza różnorodność** | §3.2: rozmów 2 641 → 1 033, kwestii na rozmowę 1,3 → 3,0, 30% zbioru z 21 postaci, 57% z 5 światów; TTR Claude‑talk 0,34–0,36 wobec GPT 0,38–0,39 na tę samą długość | **najmocniejsza alternatywa**; zgodna z „nowy świat zawsze gorszy” i z tym, że jedyny mierzalny zysk danych od run 14 to nowe światy. [dowód na liczby, poszlaka na przyczynowość] |
| **rozmiar danych vs 1 epoka** | run 14: 2 epoki, eval_loss rośnie w 2. (0,871 → 0,921); run 15–17: 1 epoka, eval_loss spada do końca (0,794 w run 17). run14h = koniec 1. epoki 2‑epokowego cosinusa (lr jeszcze wysokie) i to on ma najlepszy MG | 1 epoka na 3,8 tys. przykładów nie wygląda na niedouczenie (strata spada płasko na końcu). Ale run14h ≠ „1 epoka”: inny harmonogram lr. Nikt tego nie rozdzielił. [hipoteza] |
| **test niewrażliwy na poprawę** | 48 rozmów, 6 starych postaci + 1 nowy świat; szum sędziego ±5, szum seedu podobny; różnice między run 16 i 17 (±4) poniżej progu | **prawda** [dowód]: test nie mógł pokazać poprawy mniejszej niż ~10 ciężkich na 48 |
| **sędziowie** | zgodność 81% (rozmowy), ocena 1–5 w paśmie 3,15–3,38 (nie różnicuje); sędziowie i autorzy z tej samej rodziny modeli; nikt nie zweryfikował człowiekiem | ryzyko kołowe realne; mój ślepy test też jest z tej rodziny | [hipoteza] |
| **szum samego treningu** | run 16 i run 17 (prawie te same dane, ten sam przepis) różnią się w MG o 2–5 ciężkich/160 i w rozmowach o ±4/48 — czyli mniej niż run16 ↔ run16dh (12). Dwa SFT z tym samym przepisem dają podobny wynik; przepisy różne (14h, 16dh) dają inny | poszlaka, że przepis (harmonogram lr/DPO) ma większy wpływ niż ostatnie 10–20% zmian w danych |

### 5.3 Co z tego wynika dla pytania „czemu modele są gorsze mimo dopracowania danych”

Odpowiedź wprost: **nie są gorsze; stoją w miejscu**, bo (1) dane od run 14 są już dobre w tym, co da się policzyć, a pozostałe błędy to generalizacja i próbkowanie; (2) dopracowania polegały na odtwarzaniu i przeglądaniu tych samych ~1 000 rozmów pod zmieniające się [GRA], co zwiększa zgodność z regułami, których model i tak przestrzega, i lekko ściąga styl do [spokój]; (3) zbiór ma 2,5× mniej niezależnych kontekstów niż zbiór run 7, więc każda nowa postać w teście jest „poza rozkładem”; (4) pomiar ma próg ok. 10 ciężkich na 48 rozmów, pod którym wszystkie zmiany od run 15 się mieszczą. MG run 17 jest gorszy od run16dh, ale tak samo gorszy jest run 16 — to kwestia przepisu treningu, nie danych.

---

## 6. Rekomendacje do danych na następny run (z liczbami i priorytetem)

| # | co | ile | dlaczego (dowód) | priorytet |
|---|---|---|---|---|
| 1 | **Zamrozić dane i nie robić kolejnej rundy „odtwórz na nowym silniku i przejrzyj”** poza kwestiami, których polecenie [GRA] zmieniło się *treściowo* (nowe zdanie, którego w zbiorze nie ma) | 0 rund przeglądu; odtworzenie tylko tur z nowym poleceniem (w run 17 było to ~120 z 452 zmienionych tur) | przeglądy zmieniają 20% kwestii głównie pod reguły trzymane w 99%, ściągają pytania 22 → 14% i emocje 32 → 28%; efekt niemierzalny w teście | **1** |
| 2 | **Rozkład zamiast objętości:** limit 40 kwestii na postać (dziś 90) i 25 na kartę; rozmowy 2–4 tur (dziś 7–10) z różnymi bohaterami, porami, stanem zdrowia; +8 światów, +60 postaci; cel ≥ 2 000 niezależnych rozmów przy ~3 800 przykładach | wyciąć k01–k14 z 918 do ~450 kwestii (losowo, zachowując typy A–F); dopisać ~1 200 kwestii w ~400 rozmowach 2–4 tur w nowych światach (8 paczek po 50 rozmów) | §3.2: 1 033 rozmów wobec 2 641 w run 7; 57% z 5 światów; nowy świat zawsze +8–10 ciężkich; jedyny zmierzony zysk danych od run 14 to nowe światy (27 → 19) | **1** |
| 3 | **Wyciąć najsłabsze dane GPT z rozmów**, zostawić transakcje | usunąć GPT‑krok2 (169) i rozmowy GPT‑run3‑6 (ok. 150 z 291; transakcje zostają); rozszerzyć `DISCLAIM` (§3.4) i przejrzeć ręcznie ~300 kwestii GPT, które zostaną; GPT‑B (39) i krok1 (187) przefiltrować przez recytację (5‑gram z karty → odrzut) | §3.3: 75–77% ocen ≤ 2, 16/40 zastrzeżeń; §3.1: recytacja 8–30%; to one uczą „mówić zasady na głos” i recytować | **2** |
| 4 | **Polecenie [GRA] dla pytań prywatnych** (zmiana w silniku + dane pod nie) | ~150 kwestii w ≥ 40 postaciach i 8 światach: „masz dzieci / kogo straciłeś / skąd jesteś / byłeś tam?” z nowym zdaniem [GRA]; odpowiedzi różnorodne (zbycie, żart, odwrócenie pytania, fakt z karty), nie sam szablon „Mam X i Y, reszta nie twoja sprawa” | §4: 99 dobrych przykładów bez polecenia dało run 17 1 uwagę o krewnych wobec 8 u run 16, ale zmyślenia *zdarzeń* zostały; „umowa dawno stoi” pokazuje, że zdanie w [GRA] + 46 przykładów daje ruch (0–2/6 → 3/6) | **2** |
| 5 | **Dopłaty/zmiany umowy w drużynie:** dołożyć do 150 przykładów „umowa dawno stoi” (dziś 46) w 40 postaciach, plus osłona w silniku (kwota przy tym poleceniu → ponowna próba) | +100 kwestii | run 17: 3/6, run16dh 2/6; 46 przykładów to za mało na nawyk | 3 |
| 6 | **MG: nic nie dokładać, nie powielać.** Zbiór MG jest poprawny (0 „dobija” bez „pada” w 146 rundach, 0% pominiętych liczb, 101/114 mikstur poprawnie). Jeśli cokolwiek: 100 rund „wróg na resztce zdrowia przeżywa” z jawnym „ledwo się trzyma” (dziś 22) | +100 rund (opcjonalnie) | §4; błąd MG run 16/17 idzie za przepisem treningu | 4 |
| 7 | **Powtórki, bełkot, recytacja w wyjściach:** nie danymi SFT. DPO tylko z własnych odpowiedzi, ≥ 500 par, wybór przez sędziego‑model, długość ±15%; albo osłony w silniku (`lost_left_out`, powtórzony 4‑gram → ponowna próba) i próbkowanie (temp 0,6, `repeat_penalty`, Q6_K) | — | w danych powtórek 0–1%, w modelach 3–14%; bełkot w danych 1/360 | 4 |
| 8 | **Pomiar przed danymi:** zamrożony test ≥ 60 rozmów w ≥ 2 świeżych światach, 2 rundy sędziów, raport parowany, osobny zbiór do wyboru checkpointu; raz: ten sam zbiór run 17 z innym ziarnem | — | §1.1: próg czułości ~10 ciężkich/48; run 16 ↔ run 17 w MG mieści się w szumie przepisu | **1** (warunek sensu pkt 2–5) |

Czego **nie** robić: nie zwiększać udziału MG (nie spadł); nie wracać do danych GPT jako „lepszych” (nie są); nie zamawiać kolejnych 8‑turowych rozmów na tych samych 21 postaciach; nie zamawiać „więcej przykładów nie wiem/nie byłem” w tym samym kształcie (99 już jest, wszystkie dobre).

---

## 7. Pewność wniosków — podsumowanie

| wniosek | pewność |
|---|---|
| Rozmowy nie pogorszyły się od run 14; poprawiły do run 16; run 17 ≈ run16dh | dowód (4 niezależne rundy sędziów, pary) |
| MG run 17 gorszy od run16dh i run14h; run 16 tak samo gorszy | dowód (pary p=0,02–0,06) |
| Przyczyna MG to nie dane (udział wzrósł, jakość rund poprawna) | dowód na dane; poszlaka na „przepis/szum” (brak powtórzenia z innym ziarnem) |
| Dane Claude są lepsze jakościowo niż dane GPT | dowód na miary obiektywne (zastrzeżenia, [spokój], recytacja); poszlaka na gust (oceniający z tej samej rodziny) |
| Modele uczone na GPT mówią gorzej niż uczone na Claude | poszlaka (5 rozmów, różne [GRA]; run12h vs run14 p=0,02 z DPO jako zakłóceniem) |
| Dane uczą dobrego zachowania dla wszystkich „wspólnych błędów” | dowód (zliczone na zbiorze run 17) |
| Różnorodność promptów spadła 2,5× i to ogranicza uogólnienie | dowód na liczby; poszlaka na skutek |
| Przegląd lekko spłaszczył styl | dowód na liczby; skutek niemierzalny |
| Test nie widzi różnic < ~10 ciężkich/48 | dowód |
| GPT zamawiany „po 50 linii z wymyślonym światem” dawał większy rozkład | dowód (67 światów / 2 641 rozmów) |
