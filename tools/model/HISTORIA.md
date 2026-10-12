# Jak powstał model `neuro-npc` — historia, tutorial i plan

Ten dokument opisuje po kolei, jak z gotowego polskiego modelu językowego zrobiliśmy własny „głos” postaci gry, co po drodze nie wyszło, jak to powtórzyć od zera krok po kroku bez wiedzy o uczeniu maszynowym, i co zrobić dalej (także gdyby zaczynać jeszcze raz).

Oznaczenia przy krokach: **✅ działa i zostało**, **⚠️ słabe, ale częściowo użyteczne**, **❌ słabe, odrzucone**.

---

## 1. O co w ogóle chodzi (dla laika)

Gra ma dwa „mózgi”:

- **Umysł** (`NEURO_LLM_MODEL`, u nas `gemma3:4b` przez Ollamę) — czyta, co gracz napisał (czy to oferta, pytanie, obelga), generuje światy, wyłapuje nowe fakty i przedmioty z rozmów, przepisuje opisy w podglądzie. Tu liczy się rozumienie, nie styl.
- **Głos** (`NEURO_TALK_MODEL`, u nas `neuro-npc-run5`) — **tylko wymyśla kwestię postaci**. Dostaje od gry gotową decyzję („sprzedajesz kord za 40”, „odsyłasz do Jagny”), opis postaci, to, co postać wie, nastrój — i ma to powiedzieć po polsku, w charakterze, krótko, z liczbami co do grosza.

Gotowe modele robiły to źle: albo mówiły drewnianie (gemma), albo nie trzymały się zasad — wymyślały ceny, pisały liczebniki słownie, wstawiały emoji, mieszały dialekty (Bielik bez treningu). Dlatego Głos **douczyliśmy** na własnych przykładach rozmów. To się nazywa **fine-tuning**, a konkretnie **LoRA**: nie zmieniamy całego modelu (miliardy liczb), tylko dokładamy mały „nakładkowy” zestaw wag (kilkadziesiąt MB), który uczy się naszego stylu i naszych zasad.

### Słowniczek

| Pojęcie | Po ludzku |
|---|---|
| **Model bazowy** | gotowy model, od którego zaczynamy: `Bielik-4.5B-v3.0-Instruct` (polski, 4,5 mld parametrów). |
| **LoRA / adapter** | mały plik z „poprawkami” do modelu bazowego; trenujemy tylko jego. Wynik jednego treningu = jeden adapter (`runN-final.safetensors`). |
| **Dane treningowe** | plik JSONL: każda linia to jedna rozmowa — SYSTEM (kim jesteś, co wiesz), wejście gracza z blokiem `[GRA]` (decyzja gry), i wzorcowa odpowiedź postaci. Model uczy się *odpowiadać tak jak wzorzec*. |
| **Maskowanie promptu** | uczymy model tylko na odpowiedziach, nie na promptach — żeby nie uczył się „pisać promptów gry”. |
| **Krok / iteracja** | jedno poprawienie wag na porcji przykładów. 800 kroków × 4 przykłady ≈ 3200 przykładów przejrzanych ≈ jedna epoka naszego zbioru. |
| **Learning rate (lr)** | jak duże poprawki na krok. Za duży — model „zapomina” bazę; za mały — nic się nie uczy. |
| **Val loss / test loss / ppl** | miara „zaskoczenia” modelu na przykładach, których nie widział. Mniej = lepiej. Perplexity (ppl) ≈ e^loss; 3 = „model wybiera średnio spośród 3 sensownych słów”. Nie porównuj między runami, jeśli zmienił się zbiór. |
| **Fuse** | wtopienie adaptera w wagi bazowe → pełny model. |
| **GGUF / q8_0** | format i kompresja (8 bitów na wagę), w jakim Ollama trzyma model. |
| **Ollama** | lokalny serwer modeli; gra gada z nim po HTTP. |
| **Bench** | stały zestaw 80 przypadków, na którym mierzymy automatycznie, czy odpowiedź trzyma format (liczby, zdania, nastrój). |
| **Replay** | odtworzenie prawdziwej rozmowy z gry (z logu) na dwóch modelach obok siebie — najtańszy sposób, żeby *zobaczyć* różnicę. |

---

## 2. Jak to szło po kolei

### Krok 0 — gotowe modele bez treningu (6 X)
- `gemma3:4b` jako jedyny model: **⚠️** działa, ale kwestie są drewniane i po angielsku „myślane”; została jako Umysł.
- `bielik-4.5b-v3.0-instruct:Q8_0` z Ollamy jako Głos bez treningu: **❌ odrzucone** — żywszy polski, ale niestabilny: zmyślone ceny, liczebniki słownie, emoji, wycieki znaczników gry.
- Wniosek: to nie prompt jest problemem, tylko brak przykładów *naszego* formatu. Decyzja: douczyć Bielika.

### Krok 1 — wybór bazy i narzędzi (7 X) ✅
- Baza: `speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit` — polski model, wersja MLX (Apple Silicon), 8-bitowa, mieści się w 18 GB pamięci razem z treningiem.
- Narzędzie: `mlx-lm` (LoRA na Macu), skrypty `tools/model/prepare.py` (składa zbiór, dzieli na train/valid/test) i `tools/model/train.sh` (trening → fuse → Ollama).
- Ustawienia, które zostały do dziś: LoRA na 24 warstwach, rank 8, batch 1 z akumulacją 4, grad checkpointing (oszczędza pamięć), maksymalna długość 2048 tokenów, `--mask-prompt`.

### Krok 2 — pierwsze dane: syntetyczne z gemmy (7–8 X)
- `tools/synth` grał w sklep i walki „sam ze sobą” z gemmą i zapisywał rozmowy: 275 wierszy. **⚠️ słabe** — szablonowe wejścia gracza, odpowiedzi gemmy równie drewniane jak w grze. Uczyć się od słabego modelu = powielać słabość.
- Ratunek: **269 z 275 odpowiedzi przepisanych ręcznie** (narzędzia `digest.py`/`apply.py`: wypis → poprawki → wklejenie). Powstał `synth-fixed.jsonl` (314 wierszy) ✅ — do dziś w zbiorze, ale to ułamek.
- Do tego `manual.jsonl` (21 wierszy od GPT) ✅ drobiazg.

### Krok 3 — run 1 (8 X) ❌ słaby
- Zbiór: synth + **surowe logi gemmy z gry** (`talks.jsonl`, 320 wierszy, w tym 263 narracje walk) — logi stanowiły 44 % zbioru. Błąd: logi gemmy to nie wzorzec, tylko to, od czego chcieliśmy odejść.
- 400 kroków, lr 1e-4, ~45 min. Val loss 2,81 → 1,75, test ppl **5,9**.
- Po drodze: pierwszy start padł na błędzie w `prepare.py` (nieocytowane ścieżki ze spacją — folder „Application Support”).
- **Import do Ollamy dawał śmieci** (`[UNK_BYTE_…]`): wtopione wagi MLX nie niosą tokenizera, a importer Ollamy psuje tokenizer Bielika. Obejście: `tools/model/serve.sh` — serwer `mlx_lm.server` z adapterem na porcie 8080 i rozdział na Umysł/Głos (`llm.Split`, `NEURO_TALK_*`). **⚠️ tymczasowe** — sam rozdział na dwa modele został ✅ i okazał się jedną z najlepszych decyzji; serwer MLX odrzucony po naprawie importu.

### Krok 4 — naprawa drogi do Ollamy (9 X) ✅
- Działająca ścieżka (jest w `train.sh` i `publish.sh`): `mlx_lm.fuse --dequantize` → skopiować pliki tokenizera z cache HF do `fused/` → `llama.cpp/convert_hf_to_gguf.py --outtype q8_0` → `Modelfile` z szablonem ChatML, `stop <|im_end|>`, temperatura 0,8, `num_ctx 4096` → `ollama create`.
- Pułapki: nie uruchamiać konwertera z `python -I` (importuje własny moduł obok), ścieżki ze spacją w cudzysłowie, `ollama create` **bez** `-q` (GGUF już jest q8_0).

### Krok 5 — run 2 (9 X) ⚠️
- Wyrzucone surowe logi; zbiór kuratorowany **355** wierszy (synth-fixed 314 + manual 41), 240 kroków. Val 2,94 → 1,85, test ppl **4,59**.
- Sonda: ceny trzyma (3/3), ale polska odmiana się sypie („plombów”, „filt”) i — kluczowe — **w zbiorze było np. 10 przykładów kontroferty**. Model nie może umieć czegoś, czego prawie nie widział. Wniosek: dane są wąskim gardłem, nie liczba kroków.
- Porządek: `train.sh` zaczął zachowywać poprzedni adapter (`adapters-prev`), żeby móc porównywać; sprzątnięte 27 GB sierot w `~/.ollama`.

### Krok 6 — zamówienie danych u silnego modelu i run 3 (9 X) ✅ pierwszy grywalny
- Napisaliśmy `tools/model/PROMPT.md`: **dokładny kontrakt** promptu gry (SYSTEM co do znaku, bloki `[GRA]` dla każdej decyzji, nastroje, zasada liczb) i zamówienie dla GPT: 8 światów, ~1700 rozmów NPC (kupcy z historią, najemnicy, towarzysze, świadkowie śledztwa) + 300 narracji walk.
- Walidator (`check_new.py`) sprawdzał każdy wiersz względem szablonów z kodu gry; przeszło 1700 po jednej normalizacji. **660 szablonowych „sklepowych” wierszy odrzucone ❌** (powtarzalne jak synth) — do `spare/`.
- Run 3: 2355 wierszy, 1600 kroków, **~4 h**. Val 2,94 → 1,41, test ppl **3,85**. Ceny 6/6, nastroje są, polski dobry. Słabe: pogawędka najemnika ogólnikowa, czasem spacja w środku słowa.
- Lekcja na przyszłość: podział train/valid/test był losowy, a rozmowy z historią współdzielą fragmenty — val wychodził trochę za optymistycznie. Od run 4 podział jest po haszu wiersza (stały).

### Krok 7 — run 4: dane celowane + bench (10 X) ✅
- Najpierw **poprawka w grze**: najemnik i towarzysz dostawali brief kupca („poleć coś ze swojego towaru”) — stąd dziwne odpowiedzi. `Decision.BriefFor(role)`; stare dane przepisane skryptem `rebrief.py`.
- GPT: 400 wierszy skupionych (200 najemnik / 120 towarzysz / 80 kupiec) + **80 przypadków regresji, których nie uczymy** — to nasz stały bench (`bench_ollama.py`, sprawdza liczby, zdania, nastrój).
- Run 4 = **kontynuacja od adaptera run 3** (nie od zera), 700 kroków, lr 5e-5 (o połowę mniejszy, żeby nie zepsuć tego, co już umie), ~70 min. Bench: **71 → 78/80**.
- Lekcja: bench nie łapie wszystkiego. W grze na żywo „siema mordo” dawało bełkot, a pytanie o zbroję — przemowę o szkielecie. Przyczyny po stronie gry (brief bez nastroju, podpowiedź z jednego słowa w opisie potwora, zero powitań w danych) — naprawione w kodzie; logi pełnych promptów (`NEURO_LOG_TALKS=1`) i skrypty replay stały się podstawowym narzędziem.

### Krok 8 — temperatura 0,6 zamiast 0,8 ❌ odrzucone
- Bench 75/80 vs 78/80; słownikowa miara „nie-słów” nie była lepsza. Zostało 0,8.

### Krok 9 — run 5: luki z gry (10 X) ✅ obecny model
- Zamówienie z tego, czego brakowało *w logach gry*: 160 powitań, 120 zaczepek, 100 potwierdzeń najmu (model mylił kierunek zapłaty: „wypłaciłem **ci** 85”), 60 pytań o rodzaj towaru, 60 odesłań, 50 odmów towarzyszy; potoczne wejścia bez polskich znaków. GPT na ustawieniu „bardzo wysoki” — czystsze niż poprzednie paczki (0 błędów kierunku zapłaty).
- Kontynuacja od run 4, 800 kroków, lr 5e-5, 3070/171/64. Val 1,22 → 1,09, test ppl 2,99.
- Bench w dwóch seedach: run 4 **78, 79**; run 5 **72, 78** — różnica w szumie. Replay prawdziwych rozmów: powitanie 3/3 (było 0/3), odesłanie 3/3 (było 1/3), kierunek zapłaty **12/12** (było ~połowa źle). Przełączony.
- Wpadka: przy wtapianiu adaptera zabrakło pamięci (proces ubity). Trening był skończony; końcówkę robi osobno `publish.sh`. Przed fuse zamknij przeglądarkę.

### Równolegle — poprawki gry, które dały więcej niż niejeden trening ✅
Podpowiedzi w `[GRA]` (fakt, towar wg rodzaju, powitanie, zaczepka), **trop** z okruchem i imieniem znawcy, sprawdzanie liczb (tylko z bloku `[GRA]`, cyframi) i imion (po rdzeniu, z odmianą) z jedną poprawką, cena służby jako liczba zakazana w pogawędce, blokada wymyślania przedmiotów u najemników, odrzucone kwestie **z powodem** (zakładka Model w podglądzie). Zasada: **zanim douczysz model, sprawdź w logu, co dokładnie dostał** — w większości „bełkotów” winny był prompt.

### Co było słabe — lista do zapamiętania
1. ❌ Uczenie na surowych logach słabego modelu (run 1).
2. ❌ Szablonowe dane (synth, 660 „sklepowych”) — model uczy się szablonu, nie języka.
3. ❌ Za mały zbiór z dziurami (run 2) — policz przykłady *na każdy rodzaj decyzji* zanim ruszysz trening.
4. ⚠️ Losowy podział train/val przy rozmowach z historią — val zawyżony.
5. ⚠️ Mierzenie jednym przebiegiem przy temperaturze 0,8 — ±4 punkty szumu; zawsze dwa seedy.
6. ⚠️ Bench formalny ≠ gra; dopiero replay logów i granie pokazują bełkot.
7. ⚠️ Import do Ollamy „na skróty” — zawsze przez llama.cpp.
8. ❌ Temperatura 0,6.
9. ⚠️ Fuse z dekwantyzacją przy otwartej przeglądarce — brak pamięci.

---

## 3. Tutorial: jak to powtórzyć od zera (Mac z Apple Silicon, 18 GB)

Czas: przygotowanie ~1 h, każdy trening 1–4 h, w trakcie gra wyłączona. Miejsce: ~25 GB (baza, wtopione wagi, GGUF, kopie w Ollamie).

**0. Narzędzia.** Zainstaluj Ollamę (ollama.com) i Pythona 3.11+. Z katalogu gry: `tools/avatar/setup.sh` tworzy środowisko (`venv`) w folderze danych gry (`~/Library/Application Support/neuro-processor/`) i instaluje w nim `mlx-lm`. Sprawdź: `"$DATA/venv/bin/python3" -m mlx_lm.lora --help`.

**1. Zbierz dane.** Wszystko leży w folderze danych gry jako pliki JSONL (jedna rozmowa na linię, pole `messages` jak w API czatu + `source`):
- `extra-gpt-npc*.jsonl` — rozmowy zamówione u silnego modelu według `tools/model/PROMPT.md` (to jest serce zbioru; zamawiaj po 300–600 na raz, z listą braków, każ pisać potoczne wejścia gracza);
- `synth-fixed.jsonl`, `manual*.jsonl` — ręcznie poprawione i własne;
- `talks.jsonl` — logi z gry; **domyślnie pomijane** (wchodzą tylko z `NEURO_RAW_TALKS=1`), bo to zapis obecnego modelu, nie wzorzec.
Każdą nową paczkę przepuść przez walidator (`scratchpad`/`check_new.py` z kompilacji szablonów z `internal/llm/prompt.go`) i przeczytaj losowe 30 linii oczami: kierunek zapłaty, liczby, polszczyzna.

**2. Złóż zbiór.** `python3 tools/model/prepare.py` → `model/data/{train,valid,test}.jsonl`. Podział po haszu wiersza jest stały, więc kolejne runy są porównywalne między sobą. Spójrz na wypis „train/valid/test” i na liczbę przykładów na rodzaj decyzji.

**3. Trenuj.**
```
M="$HOME/Library/Application Support/neuro-processor/model"
# od zera (pierwszy raz): ~1500 kroków, lr 1e-4
NEURO_MODEL_NAME=neuro-npc-run1 tools/model/train.sh 1500 > "$M/train-1.out" 2>&1
# douczanie od poprzedniego adaptera: 600–800 kroków, lr 4e-5–5e-5
NEURO_RESUME="$M/run5-final.safetensors" NEURO_MODEL_NAME=neuro-npc-run6 NEURO_RATE=4e-5 tools/model/train.sh 600 > "$M/train-6.out" 2>&1
```
W logu co 100 kroków jest `val` — powinien spadać (z szumem). Na końcu `Test loss … ppl …`, potem fuse, GGUF i `ollama create`. Po treningu **skopiuj adapter**: `cp "$M/adapters/adapters.safetensors" "$M/run6-final.safetensors"` — to Twój punkt powrotu. Jeśli maszynie zabraknie pamięci przy wtapianiu: `NEURO_MODEL_NAME=neuro-npc-run6 tools/model/publish.sh`.

**4. Zmierz, zanim przełączysz.**
```
cd scratchpad/run4/zip/neuro_run4_focus   # (bez python3 -I)
python3 tools/bench_ollama.py --model neuro-npc-run6 --cases ../../bench/regression-80-game.jsonl --out ../../bench/run6 --temperature 0.8 --num-predict 220 --num-ctx 4096 --timeout 300
python3 tools/bench_ollama.py ... --seed 7 --out ../../bench/run6-s7
python3 scratchpad/replay4.py "$DATA/talks.jsonl" neuro-npc-run5,neuro-npc-run6   # Twoje prawdziwe rozmowy na obu modelach
```
Patrz na trzy rzeczy: liczba OK/80 w dwóch seedach, replay oczami, i to, czy błędy z poprzedniej wersji zniknęły. Lepszy model = taki, który naprawia to, co widziałeś w grze, bez psucia benchu o więcej niż szum (±3).

**5. Przełącz.** W `.env.local`: `NEURO_TALK_MODEL=neuro-npc-run6`. Powrót to zmiana jednej linijki. Graj z `NEURO_DEBUG=on` i patrz na zakładkę **Model** w podglądzie — każda odrzucona kwestia to albo błąd modelu (materiał na następne zamówienie), albo za ostra zasada w grze.

**6. Pętla.** Zbieraj z logów to, co wychodzi źle → spisz jako punkty zamówienia (jak w PROMPT.md: *ile, jaka podpowiedź co do znaku, jak ma wyglądać dobra odpowiedź, czego nie wolno*) → 300–600 nowych wierszy → krok 2–5. Jedna pętla to pół dnia.

---

## 4. Plan rozwoju

### Najbliżej (tygodnie)
1. **Run 6** z zamówienia w PROMPT.md: tropy, dopytania, zmiany tematu, zaczepki bez liczb, powitania z nastrojem, plotki na fakcie.
2. **Zasady liczb w pogawędce** — po zebraniu odrzuceń z zakładki Model zdecydować, czy cyfry 1–9 bez waluty puszczać (dziś każda cyfra = odrzucenie).
3. **Większy bench** (200–300 przypadków z gry, nie od GPT) + **sędzia**: Umysł ocenia „czy odpowiedź jest na temat i po polsku” w skali 1–3; dziś mierzymy tylko format.
4. **Pary dobre/złe z odrzuceń** — gra już zapisuje, co model napisał i co było źle. To gotowy materiał do *uczenia preferencji* (DPO): „ta odpowiedź lepsza od tamtej”. `mlx-lm` ma `mlx_lm.dpo`; 300–500 par wystarczy, żeby ukrócić konkretne nawyki (liczby słownie, powrót do ceny) skuteczniej niż kolejne 500 przykładów SFT.

Trening na PC z NVIDIA (Bielik 11B, QLoRA): instrukcja i skrypty w `tools/pc/`. Surowy 11B bez treningu wypadł w benchu 8 i 6/80 (format), ale pisze wyraźnie lepszą polszczyzną — dobry kandydat na bazę run 7.

### Dalej (miesiące)
5. **Większa baza**: Bielik 11B v2.3 w 4 bitach (LoRA ~9–10 GB pamięci — na styk na 18 GB, trening 3–4× wolniejszy) albo Qwen2.5‑7B/Gemma 3 12B z dobrym polskim. 4,5B nie przeskoczy pewnego pułapu odmiany i składni; każdy „Wędronowy Kupuc” to limit rozmiaru, nie danych. Alternatywa: **GPU w chmurze na 2–3 h** (A100 za kilkanaście dolarów) — pełna LoRA rank 32 na 11B, a lokalnie tylko GGUF q5/q6 do grania.
6. **Dane z historią**: dziś większość przykładów to jedna wymiana. Zamówić rozmowy 3–5 tur z narastającym kontekstem (targ → zgoda → pogawędka → odesłanie), bo w grze model widzi historię i czasem się nią myli.
7. **Modele per rola?** Nie. Jeden Głos z briefami ról wystarcza; za to osobny mały adapter dla **Mistrza Gry** (narracja walk) ma sens — inny styl, inny format.

### Gdyby zaczynać od nowa — co zrobić inaczej
- **Najpierw kontrakt i test, potem dane.** Spisać prompt co do znaku (dziś PROMPT.md) i zbiór 100 przypadków testowych *zanim* powstanie pierwszy przykład treningowy. My zrobiliśmy to dopiero przy run 3–4 i część wcześniejszej pracy poszła do kosza.
- **Pominąć dane syntetyczne ze słabego modelu i surowe logi** — od razu zamawiać u silnego modelu z walidatorem w pętli (odrzuca wiersz → model poprawia). Oszczędza dwa runy.
- **Od razu rozdzielić Umysł i Głos** i logować pełne prompty — to dało największy skok diagnostyczny.
- **Od razu podział po haszu i dwa seedy w pomiarze.**
- **Wybrać większą bazę na starcie**, jeśli sprzęt pozwala (albo chmurę na trening). 4,5B to dobry wybór na laptop i szybkie pętle, ale sufit jakości polszczyzny widać już teraz.
- **Format odpowiedzi**: rozważyć, żeby Głos zwracał `[nastrój]` + kwestię jak dziś, ale uczyć go też *krótkich odmów poza rolą* („nie wiem”, „nie mój interes”) zamiast zmyślania — brak takich przykładów to źródło połowy bełkotu.
- **Uczyć preferencji wcześnie** (punkt 4): odrzucenia zbierają się same od pierwszego dnia grania.

### Czego nie robić
- Nie uczyć na własnych wyjściach modelu bez ręcznej selekcji.
- Nie zmieniać kilku rzeczy naraz (dane + lr + temperatura) — nie będzie wiadomo, co pomogło.
- Nie przełączać gry na nowy model bez replayu prawdziwych rozmów.
- Nie trenować, gdy Ollama trzyma model w pamięci (gra wyłączona), i nie wtapiać adaptera przy otwartej przeglądarce.
