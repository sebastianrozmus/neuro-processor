# Recenzja 2026-10-24: sprzeczne znaczniki nastroju — co z tym w run20

Pytanie: run19 (Bielik 11B v3) ma w test18 12 sprzecznych znaczników na 240 (run18/v2.3: 0, run19h: 6, run16dh: 2). Czy to naprawiać w danych, w silniku, osłoną, czy DPO?

Źródła: `D:\neuro-train\run18\data\{train,valid}.jsonl`, `test18\t18-*-s7/s11.jsonl`, `test18\joint3v2`, `joint4v2` (pole `znacznik`), silnik `internal/game/types.go`, `shop.go`, `session.go`, `reading.go`, `stings.go`, `internal/speech/speech.go`. Liczenie CPU, skrypty w scratchpadzie sesji (`zn_analyze.py`, `zn_test.py`, `zn_train2.py`, `zn_shield.py`).

## 1. Dane treningowe run18/run19

- 4005 rozmów, 12 681 tur asystenta; 3369 tur (train 3198 + valid 171) ma w [GRA] „Nastrój twojej postaci”. Nastroje: spokojna 2879, zadowolona 177, zniecierpliwiona 164, obrażona 113, poirytowana 36. **[dowód]**
- **Sprzecznych wg tabeli sędziów: 0 / 3369.** [gniew] przy „spokojna”: **0 / 2879** (dokładnie zero; stara pula przeszła przez `check.py` z tą samą tabelą `CLASH`, nowe paczki pisane pod reguły). **[dowód]**
- Rozkład przy „spokojna” (train): spokój 29 %, kpina 17 %, radość 16 %, zdziwienie 14 %, smutek 9 %, czułość 8 %, strach 7 %, gniew 0 %. Przy „obrażona”: gniew 87 %, kpina 13 %. „zniecierpliwiona”: kpina 55 %, gniew 20 %, strach 10 %, spokój 8 %, smutek 5 %, radość 0 %. „zadowolona”: radość 95 %, czułość 4 %. „poirytowana”: gniew 88 %, kpina 9 %. **[dowód]**
- Przekleństwo/zaczepka gracza przy „spokojna” (32 tury, luźny regex łącznie z „dziadku”): kpina 12, czułość 8, spokój 4, zdziwienie 3, strach 2, radość 2, smutek 1, **gniew 0**. Reprymenda za przekleństwo istnieje w danych, ale bez [gniew]: Siostra Klara (karta: „nie znosi przekleństw”) na „cholera, ale tu zimno” → „[zdziwienie] Przy mnie się nie klnie, młodzieńcze, nawet z zimna…”; na „kurde ale mnie boli” → „[spokój] Przeklinaniem rany nie zszyjesz…”; „auuu boli jak cholera” → „[kpina] Skoro klniesz, to żyjesz, ale ledwo”. Obelgi wprost („ty stary skurwielu”, „ty szarlatanie”, „spadaj dziadu”) mają w [GRA] już „obrażona” (silnik) i wtedy [gniew]. **[dowód]**
- Flirt przy „spokojna” (22 tury): zdziwienie 5, spokój 5, kpina 4, radość 4, czułość 2, strach 1, smutek 1, gniew 0 („ladna jestes, skoczymy na kumys?” → „[kpina] Ładna? Orlica też jest ładna, a dziobie w oko”). Treść „gniewna” przy spokojnej („nie znoszę / nienawidzę / milcz”, 40 tur): spokój 16, smutek 7, radość 6, zdziwienie 4, strach 3, czułość 2, kpina 2, gniew 0. **[dowód]**
- Wniosek: dane niczego nie uczą „źle”; uczą wyłącznie przez brak — 2879 przykładów „spokojna” bez ani jednego [gniew]. SFT taki zakaz utrwala słabo, bo nie ma przykładu negatywnego, a przy silnej treści (przekleństwo, „czego nie znosisz”) prior bazy wygrywa. **[poszlaka]**

## 2. Co robi run19 i dlaczego częściej niż run18

Moje zliczenie z surowych odpowiedzi (355 tur rozmów na ziarno; nastroje w teście: spokojna 309, zniecierpliwiona 18, obrażona 17, zadowolona 11) zgadza się co do kwestii z polem `znacznik` sędziów; „12 na 240” to 6 kwestii × 2 rundy sędziów. **[dowód]**

| model | s7 | s11 | razem | gdzie |
|---|---|---|---|---|
| run19 | 2 | 4 | 6 | Agata kw5 ×2, Mira kw4 ×2, Rataj kw5, Bronisława kw2 |
| run19h | 2 | 1 | 3 | Agata kw5 ×2, Rataj kw5 |
| run16dh / q4 | 1 / 1 | 0 / 1 | 1 / 2 | Kuba kw5 zniecierpliwiona→[radość] |
| run18, run18h | 0 | 0 | 0 | — |
| run17 / run14h | 1 / 1 | 1 / 2 | 2 / 3 | pojedyncze |

Wszystkie przypadki run19/run19h to „spokojna” → [gniew] i wszystkie są **zgodne z treścią**:
- `t2-agata-rodzina` kw5, gracz „kurwa, nie wierze”, karta „Nie znosi przekleństw”: run19 „[gniew] Przy mnie nie przeklinaj, córko, nawet z niedowierzania…”; run18 ta sama treść z innym znacznikiem: „[kpina] Przeklinasz przy szarytce? Język na wodzy…”; run16dh „[spokój] Nie musisz wierzyć…” (bez reakcji na przekleństwo, sędzia to też wytknął).
- `t2-mira-flirt` kw4 to nie flirt, tylko pytanie „czego najbardziej nie znosisz”: run19 „[gniew] Ludzi, co obiecują i nie płacą. Tego nienawidzę najbardziej”; run18 „[smutek] Najbardziej nie znoszę, jak ktoś nie płaci…”, run16dh „[kpina] …”.
- `t1-rataj-druzyna` kw5 „gwizdne sobie na lodce” (karta: nie pozwala gwizdać na wodzie): run19 s11 „[gniew] Ani na łódce, ani nigdzie…”, run19 s7 i run18 „[strach] Ani mi się waż!…”.
- `t1-bronislawa-ai` kw2 „kurwa jak boli”: run19 s11 „[gniew] Kurwa, a niech to, przy mnie przeklinać nie będziesz!”; run19 s7 „[kpina] Kurwa to brzydko, a boli, bo żyjesz”.

Dlaczego v3 częściej:
- **Ekspresywność bazy** (poszlaka): przy „spokojna” run19 daje kpina 19–20 % (run18 12–13 %, dane 17 %), zdziwienie 7 % (run18 13 %), spokój 30–31 % (run18 32–36 %). Przy „zniecierpliwiona” run19 wybiera [gniew] 5–6/18, run18 2–3/18 (dane 20 % ≈ 3,6/18). Łącznie [gniew] 24–26 u run19 wobec 19–20 u run18 (17 to obrażona, identycznie u wszystkich). v3 przesuwa się ku ostrzejszym znacznikom i tam, gdzie treść jest gniewna, przekracza zakaz. **[poszlaka — jeden trening, dwa ziarna]**
- **Znacznik z treści, nie z [GRA]**: wszystkie 6 kwestii to reprymenda/tabu/„nienawidzę”; run18 pisze to samo pod [kpina]/[strach]/[smutek], czyli trzyma znacznik przy nastroju z [GRA], a v3 przy treści. **[dowód na opis, hipoteza na mechanizm]** Inne priory Instruct v3 nie są mierzalne bez GPU (bazy bez treningu nie znają formatu). run19h (środek wygaszania) ma 3, run19 6, więc to nie kwestia „za mało epok”.
- **Czy model, czy reguła?** Po połowie. Przy „kurwa jak boli” / „kurwa, nie wierze” silnik świadomie zostawia „spokojna”: `rudeWords` ma „kurwo”, „suko”, „chuj”, „skurw”, ale nie wykrzyknik „kurwa”/„cholera”; `stings.go` czyta z karty tylko „nazywa go X”, nie „nie znosi przekleństw”. Dane są z tym spójne (reprymenda pod kpiną/zdziwieniem). Dla Miry i Rataja [gniew] jest naturalny, ale reguła „spokojna ≠ gniew” jest założeniem gry (nastrój = stan z silnika, znacznik = ekspresja) i działa, dopóki silnik zmienia nastrój, gdy coś się dzieje. **[ocena]**

## 3. Osłona w silniku: już jest, ale ma skutek uboczny

- `fitMood`/`Mood.FitTag` w `types.go` od commitu f1936f6 (15.10): działa w `narration.go` dla każdej odpowiedzi (kupiec, najemnik, towarzysz), także w głosie czytanym w trakcie pisania (`shop.go: speaking.Fit`); do historii targu trafia znacznik po podmianie (`ui/game.go:222`). **[dowód]**
- Tabela silnika jest **ostrzejsza niż sędziów i niż dane**: `MoodCalm: wrong {gniew, strach} → spokój`, `MoodOffended: wrong {radość, czułość, spokój} → gniew`. Efekt: **spokojna+[strach]** jest podmieniane, a to 203 / 3369 tur danych (6 %) i u run18 **24–25 kwestii na 355** na ziarno (run19 19–23, run16dh 13–18) — wobec 0–4 kwestii faktycznie sprzecznych. Przykłady, które osłona zamienia na [spokój]: Hanna o córce „[strach] Boję się, że kręci się koło Nurka bez Twarzy”; Ignac „[strach] Matko, ledwo się trzymasz na nogach”; Rataj „[strach] Ani mi się waż! Na wodzie nie gwiżdż” (sędziowie uznali to za dobre). Brief v2 sędziów mówi wprost: „inne znaczniki przy spokojna są dozwolone”. **[dowód]**
- Skutki osłony poza tym: TTS dostaje tempo/wysokość „spokój” (175/48) zamiast „gniew” (205/42) przy gniewnym zdaniu — słyszalny, ale mały zgrzyt; w UI znika szara etykieta „(gniew)”. Treść zostaje gniewna, więc niespójność znacznik–treść jest ceną osłony, nie da się jej uniknąć bez ponownej próby (koszt czasu odpowiedzi) albo zmiany nastroju w silniku. **[dowód na mechanikę, ocena na wagę]**

## 4. Opcje na run20 i koszt

| opcja | co | koszt | efekt | ocena |
|---|---|---|---|---|
| (c) osłona | jest; **poprawić tabelę**: usunąć `strach` z `MoodCalm.wrong` (zostawić gniew), rozważyć podmianę spokojna+[gniew] → **[kpina]** zamiast [spokój] (w danych reprymendy przy spokojnej to głównie kpina; „Przy mnie nie przeklinaj, córko” pod [kpina] brzmi jak run18, pod [spokój] gaśnie) | 10–30 min + test | usuwa 100 % klasy u każdego modelu; poprawka tabeli oddaje ~7 % legalnych [strach] | **zrobić najpierw** |
| (b) silnik: nastrój reaguje | z karty cue „nie znosi/nie cierpi przekleństw” (jak `stingCue`) + wykrzykniki „kurwa, cholera, kurde, chuj…” → `MoodOffended` + podpowiedź „Gracz przeklina, a ty tego nie znosisz”; tylko dla postaci z taką kartą | 1–2 h Go + test | 3 z 6 przypadków run19 (Agata ×2, Bronisława) stają się legalnym [gniew] z zachowaną emocją; reszta postaci dalej „spokojna” jak w danych | warto, niezależnie od modelu |
| (a) dane | nie ma czego czyścić (0 sprzecznych). Kontrprzykłady: ~30–60 kwestii do paczek run20 (wykrzyknik przy postaci „nie znosi przekleństw” → [kpina]/[zdziwienie]; „czego nie znosisz” → [kpina]/[smutek]; tabu → [strach]/[kpina]) | ~1 h przy pisaniu paczek, 0 dodatkowego GPU | niepewny: 2879 przykładów bez [gniew] nie wystarczyło, 50 więcej nie zmieni prioru; co najwyżej ustabilizuje reprymendę | dołożyć tylko „przy okazji” |
| (d) DPO na znacznik | pary chosen [kpina] / rejected [gniew] | ~300 par syntetycznych + 1 epoka DPO (jak run10: minuty) | 6 naturalnych przypadków na 355 to za mało na pary; run10 pokazał, że DPO na znaczniki spłaszcza emocje (48 par GPT → [spokój]); ryzyko stłumienia [gniew] tam, gdzie jest słuszny (obrażona 17/17) | **nie** |

**Rekomendacja:** (1) poprawić tabelę osłony (strach zostaje przy spokojnej; spokojna+gniew → kpina lub spokój — decyzja smaku, ja bym dał kpinę) i uznać klasę za zamkniętą w ocenie — sędziowie już liczą ją osobno w polu `znacznik`, w `check.py` warto liczyć ją po osłonie. (2) Dołożyć w silniku reakcję na przekleństwo u postaci z taką kartą — to poprawia grę (Agata faktycznie powinna się obrazić), nie tylko metrykę. (3) W danych run20 nic nie ruszać; ewentualnie kilkadziesiąt kontrprzykładów w nowych paczkach. (4) DPO nie. Po (1) różnica run18/run19 w rozmowach z tej klasy znika (0 vs 0), co nie zmienia wniosku „remis” z recenzji 24.10.
