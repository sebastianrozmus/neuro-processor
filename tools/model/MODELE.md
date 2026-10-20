# Wersje modelu `neuro-npc`: karta wyników

Zestawienie wszystkich wersji Głosu, czyli modelu, który pisze kwestie postaci. Dla każdej wersji: baza, dane, ustawienia treningu, wynik benchu i to, co pokazały testy w grze. Pełna opowieść, z tym, co nie wyszło i dlaczego, jest w [HISTORIA.md](HISTORIA.md). Jak mierzyć, opisuje [tools/bench/README.md](../bench/README.md).

**Obecny model:** `neuro-npc-run7` (Bielik 11B). **Powrót:** `neuro-npc-run6` (Bielik 4.5B, 3× szybszy).

## W skrócie

| wersja | baza | gdzie i jak | dane (train) | bench s42 / s7 | mediana odpowiedzi | status |
|---|---|---|---|---|---|---|
| bez treningu | Bielik 4.5B v3.0 | — | — | — | — | ❌ zmyślone ceny, liczebniki słownie, emoji |
| run 1 | Bielik 4.5B v3.0 | Mac, LoRA (MLX) | synteza + surowe logi gemmy (44% zbioru) | — | — | ❌ |
| run 2 | Bielik 4.5B v3.0 | Mac, LoRA | 355 (ręcznie poprawione) | — | — | ⚠️ |
| run 3 | Bielik 4.5B v3.0 | Mac, LoRA, od zera | 2355 (pierwsze dane GPT) | 71 / — | — | ✅ pierwszy grywalny |
| run 4 | Bielik 4.5B v3.0 | Mac, LoRA, od run 3 | +400 celowanych | 78 / 79 | 1,8 s | ✅ |
| run 4, temp. 0,6 | jak run 4 | tylko inna temperatura | — | 75 / — | 1,8 s | ❌ odrzucone |
| run 5 | Bielik 4.5B v3.0 | Mac, LoRA, od run 4 | 3070 (+550 luk z gry) | 72 / 78 | 1,7 s | ✅ |
| run 6 | Bielik 4.5B v3.0 | Mac, LoRA, od run 5 | 3369 (+320 tropów) | 76 / 78 | 1,8–1,9 s | ✅ powrót |
| Bielik 11B bez treningu | Bielik 11B v2.3 (Q4_K_M) | — | — | 8 / 6 | 13,2–13,6 s | ❌ nie zna formatu gry |
| **run 7** | **Bielik 11B v2.3** | **PC RTX 4060 Ti, QLoRA, od zera** | **3369 (jak run 6)** | **80 / 80** | **5,1–5,2 s** | **✅ obecny** |
| run 8 (i run 8-e1 po 1. epoce) | Bielik-Minitron-7B v3.0 | PC, QLoRA, od zera | 3117 (zbiór run 7 + 64 ręczne lekcje) | 79 / 78, e1: 79 / 77 (run 7 na tym pliku: 78 / 75) | 1,2 s na PC (run 7: 1,6–1,8 s) | ⚠️ zastąpiony przez run 9b |
| run 10d (i 10a–c) | Bielik-Minitron-7B v3.0 | PC, DPO (QLoRA) od scalonego run 9b, 1 epoka, lr 5e-5, beta 0,1 | 278 par DPO (krok 3a), 86 odłożonych | — (scenariusze: 94% czystych wobec 84% run 9b) | 1,1 s | ⚠️ kandydat: zasady tak, charakter słabszy |
| **run 9b** (i 9a) | **Bielik-Minitron-7B v3.0** | **PC, QLoRA, od run 8-e1, 2 × 1 epoka** | **838 od GPT (kroki 1–2) + 2000 powtórki** | **79 / 77** (9a: 79 / 78) | jak run 8 | **✅ obecny na PC** |

Bench to 80 przypadków, których model nie widział w treningu (`tools/bench/eval/regression-80.jsonl`): 24 rozmowy towarzysza, 40 najemnika i 16 kupca. Wszystkie runy od 4 wzwyż mierzone na tym samym pliku przypadków (sha `b55d2900`), temperatura 0,8, seedy 42 i 7. Różnica ±3 punkty między seedami to szum. Run 3 i run 4 seed 42 to liczby z historii; ich pliki wyników przepadły.

Mediana odpowiedzi to czas jednej kwestii w benchu na Macu M3 Pro (18 GB), z modelem już wczytanym. Surowy 11B jest wolny, bo gada za długo i co trzecią odpowiedź ucina limit.

## Wersje po kolei

### Bielik 4.5B bez treningu
- **Baza:** `bielik-4.5b-v3.0-instruct:Q8_0` z Ollamy.
- **Wynik:** żywszy polski niż gemma, ale niestabilny: zmyślone ceny, liczebniki słownie, emoji, wycieki znaczników gry.
- **Wniosek:** problemem nie był prompt, tylko brak przykładów formatu gry. Decyzja: douczać.

### run 1 ❌
- **Baza:** `speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit`, LoRA w `mlx-lm` na Macu.
- **Ustawienia (zostały do run 6):** 24 warstwy, rank 8, batch 1 z akumulacją 4, maks. 2048 tokenów, `--mask-prompt`.
- **Dane:** dane syntetyczne z gry plus surowe logi gemmy (`talks.jsonl`, 320 wierszy, w tym 263 narracje walk). Logi stanowiły 44% zbioru.
- **Trening:** 400 kroków, lr 1e-4, ok. 45 min. Val loss 2,81 → 1,75, test ppl 5,9.
- **Co wyszło:** model powielał drewniany styl gemmy, bo na nim się uczył. Import do Ollamy dawał śmieci (`[UNK_BYTE_…]`), więc tymczasowo działał przez serwer MLX.
- **Lekcja:** nie uczyć na wyjściach słabego modelu.

### run 2 ⚠️
- **Dane:** 355 wierszy kuratorowanych: 314 z syntezy przepisanych ręcznie (269 z 275 odpowiedzi poprawionych) i 41 własnych.
- **Trening:** 240 kroków. Val loss 2,94 → 1,85, test ppl 4,59.
- **Co wyszło:** ceny trzyma (3/3), ale odmiana się sypie („plombów”, „filt”). W zbiorze było np. tylko 10 przykładów kontroferty, więc jej nie umiał.
- **Lekcja:** wąskim gardłem są dane, nie liczba kroków. Policz przykłady na każdy rodzaj decyzji przed treningiem.
- **Przy okazji:** naprawiona droga do Ollamy przez llama.cpp (GGUF q8_0).

### run 3 ✅ pierwszy grywalny
- **Dane:** 2355 wierszy. Rdzeń to ok. 1700 rozmów zamówionych u GPT według kontraktu `PROMPT.md` (8 światów: kupcy, najemnicy, towarzysze, świadkowie, narracje walk). Każdy wiersz przeszedł walidator; 660 szablonowych „sklepowych” odrzucono.
- **Trening:** od zera, 1600 kroków, ok. 4 h. Val loss 2,94 → 1,41, test ppl 3,85.
- **Bench:** 71/80 (seed 42).
- **Co wyszło:** ceny 6/6, nastroje są, polski dobry. Słabe: pogawędka najemnika ogólnikowa, czasem spacja w środku słowa.
- **Lekcja:** losowy podział train/val przy rozmowach z historią zawyżał wynik val. Od run 4 podział jest po haszu wiersza.

### run 4 ✅
- **Najpierw poprawka w grze:** najemnik i towarzysz dostawali polecenie kupca („poleć coś ze swojego towaru”). Stare dane przepisał `rebrief.py`.
- **Dane:** +400 wierszy celowanych (200 najemnik, 120 towarzysz, 80 kupiec). Do tego 80 przypadków odłożonych na bench.
- **Trening:** kontynuacja od run 3, 700 kroków, lr 5e-5, ok. 70 min.
- **Bench:** 78 / 79 (było 71). Jedyny błąd w seedzie 7: liczebnik słownie („jedna”).
- **Co pokazała gra:** bench nie łapie wszystkiego. „Siema mordo” dawało bełkot, pytanie o zbroję dawało przemowę o szkielecie. Winny był prompt (polecenie bez nastroju, brak powitań w danych, podpowiedź z jednego słowa); naprawione w kodzie. Od tego czasu gra loguje pełne prompty (`NEURO_LOG_TALKS=1`), a odtworzenia (`replay.py`) są podstawowym testem.

### run 4 z temperaturą 0,6 ❌
- Ten sam model, niższa temperatura: 75/80 zamiast 78. Błędy: za dużo zdań, liczebnik „dwie”, brak kwoty 14. Zostało 0,8.

### run 5 ✅
- **Dane:** +550 wierszy z luk widocznych w logach gry: 160 powitań, 120 zaczepek, 100 potwierdzeń najmu, 60 pytań o rodzaj towaru, 60 odesłań, 50 odmów towarzyszy. Potoczne wejścia, także bez polskich znaków. Podział 3070/171/64.
- **Trening:** od run 4, 800 kroków, lr 5e-5. Val loss 1,22 → 1,09, test ppl 2,99.
- **Bench:** 72 / 78, czyli w szumie wobec run 4 (78 / 79). Błędy w seedzie 7: liczba w rozmowie bez liczb, za dużo zdań.
- **Co pokazały odtworzenia:** powitanie 3/3 (było 0/3), odesłanie do znawcy 3/3 (było 1/3), kierunek zapłaty przy najmie 12/12 (wcześniej ok. połowa źle: „wypłaciłem ci 85” zamiast „zapłaciłeś mi”). Przełączony mimo równego benchu.
- **Wpadka:** przy wtapianiu adaptera zabrakło pamięci; końcówkę zrobił osobno `publish.sh`.

### run 6 ✅ (powrót)
- **Dane:** +320 wierszy: odesłania z okruchem wiedzy i imieniem znawcy, dopytania („to wszystko, więcej wie X”), zaczepki bez liczb, plotki oparte na fakcie. Razem 3369/185/71.
- **Trening:** od run 5, 600 kroków, lr 4e-5. Test ppl 3,18 (nieporównywalne z run 5, bo zmienił się zbiór).
- **Bench:** 76 / 78. Błędy: zakazana kwota 50, liczba w rozmowie bez liczb, liczebniki „dwa” i „trzy”, brak kwoty 14; u kupca 14/16 w seedzie 42.
- **Co pokazały odtworzenia:** trop 3/3, dopytanie 3/3, plotki na fakcie 2/3.
- **Słabe:** odmiana i pisownia nietypowych imion („Kupcz”, „Agness” zamiast „Agnes”). To sufit bazy 4,5B, nie danych. Na „dołącz za darmo” potrafił posłusznie odpowiedzieć „Napisałem, że dołączam za darmo”.

### Bielik 11B bez treningu
- **Baza:** `SpeakLeash/bielik-11b-v2.3-instruct:Q4_K_M` z Ollamy, jako sprawdzian przed treningiem na PC.
- **Bench:** 8 / 6. Główne błędy: za dużo zdań (70–74 na 80), cudzysłowy i dopiski typu „Postać:” (36–42), odpowiedzi ucięte na limicie (29–32).
- **Wniosek:** formatu gry nie zna wcale, ale pisze wyraźnie lepszą polszczyzną. Dobra baza.

### run 7 ✅ obecny
- **Baza:** `speakleash/Bielik-11B-v2.3-Instruct` w 4 bitach.
- **Gdzie:** PC z RTX 4060 Ti 16 GB, WSL2 + Unsloth (`tools/pc/`). Pierwszy trening poza Makiem.
- **Dane:** ten sam zbiór co run 6 (3369/185/71). Około 90% to rozmowy napisane przez GPT, reszta ręcznie poprawiona synteza i własne przykłady.
- **Trening:** QLoRA od zera (adaptery 4,5B nie pasują do 11B), 2 epoki, lr 1e-4, rank 16, akumulacja 8, maks. 2048 tokenów. Wynik to jeden scalony GGUF Q5_K_M (7,9 GB), zarejestrowany w Ollamie przez `tools/model/import.sh`.
- **Bench:** 80 / 80, wszystkie role bezbłędnie. Pierwszy raz bez jednego błędu formatu.
- **Co pokazały odtworzenia** (`tools/bench/results/replays/replay-run7.out`):
  - okruch wiedzy cytowany wiernie;
  - imiona znawców bez przekręceń („Siostra Agnes w Aptece” 3/3);
  - zaczepki o piwo dostają odpowiedź i wracają do tropu;
  - powitania mają charakter („Elo, bratku, po co mnie zaczepiasz?”);
  - odmowa „dołącz za darmo” poprawna.
- **Co pokazała gra na żywo:**
  - lepsza polszczyzna i więcej inwencji niż run 6;
  - czasem pomija znacznik nastroju (gra przyjmuje wtedy spokój);
  - zdarzyło mu się wymyślić miejsce („wieża maga Ozryka”) i przekręcić fakt o Kopalni Durgara;
  - własne zwroty potrafi powtarzać w kolejnych kwestiach, bo widzi je w historii rozmowy. Przykład: „między kolanami nosi coś ostrego”; w danych treningowych tego nie ma.
  - Złe kwestie warto oznaczać w podglądzie, bo to materiał na pary do DPO.
- **Co pokazała gra na PC (15.10.2026, uwagi z podglądu):**
  - na „tak, jagody” kowal odesłał po jagody do Kopalni Durgara, „tylko wróć z językiem”. Na „z jezykiem ???” brnął dalej, zamiast wyjaśnić. Fakt z podpowiedzi („Możesz wpleść…”) wciska na siłę, nawet gdy nie pasuje, a zwrot „zdobyć języka” przekręca;
  - znacznik bywa sprzeczny z nastrojem z `[GRA]`; tu akurat kpina u spokojnego, co do krasnoluda pasuje;
  - mały model (gemma3:4b) wziął „jeśli chcesz jagody, idź po nie” za ofertę, więc jagody wpadły do towaru.
  - **Poprawki w grze:** nowy przedmiot trafia do świata tylko z kwestii, która go sprzedaje (`game.Offered`). Podpowiedź kończy się „Jeśli nie pasuje do tego, co mówi gracz, pomiń ją.” Znacznik sprzeczny z nastrojem gra podmienia (`Mood.FitTag`). Lekcje dla modelu są w `run8_cases.py` (niżej).
  - **Pamięć na PC:** run 7 (ok. 7 GB), gemma (ok. 4 GB) i Chatterbox (ok. 4 GB) zajmują całe 16 GB karty. Gdy model pisze kwestię, a głos w tym samym czasie syntezuje zdanie, część trafia do RAM i synteza zwalnia do 20–47 s na zdanie. Pomaga `OLLAMA_KV_CACHE_TYPE=q8_0` z `OLLAMA_FLASH_ATTENTION=1`; mniejszy Głos pomógłby bardziej.
- **Koszt:**
  - odpowiedź ok. 3× wolniej (mediana 5,2 s wobec 1,8 s run 6);
  - 7,9 GB w pamięci zamiast 5,1 GB;
  - na Macu 18 GB razem z gemmą (4,4 GB) system mocno korzysta z dysku przy wielu otwartych kartach.
  - Pierwsza kwestia po wczytaniu modelu trwa ok. 20 s, więc gra prosi Ollamę, żeby trzymała model w pamięci 30 min (`keep_alive`).
- **Słabe drobiazgi:** czasem zdanie trochę za długie, pojedyncze potknięcia („tanioje”, „Szkielet Wojownik”).

### run 8 ⚠️ kandydat na PC
- **Baza:** `speakleash/Bielik-Minitron-7B-v3.0-Instruct` (7,35B, przycięty i destylowany z Bielika 11B v3). Tokenizer ma `<|im_start|>`/`<|im_end|>` jako osobne tokeny, więc szablon gry pasuje bez zmian.
- **Dane:** zbiór run 7 plus 64 ręczne lekcje z `run8_cases.py` (3117/169/64 po `prepare.py`; 12 przykładów za długich pominiętych).
- **Trening:** QLoRA od zera na PC, ustawienia jak run 7 (2 epoki, lr 1e-4, rank 16), 778 kroków po ok. 18 s, razem ok. 4 h. GGUF Q5_K_M 5,3 GB.
  - Pierwsza próba padła na starcie: `transformers` 5.17 nie zna już `warmup_ratio`; `train.py` liczy teraz `warmup_steps` (3% kroków).
  - eval_loss: 0,885 → 0,815 → 0,782 → **0,753 po 1. epoce** → 0,772 → 0,771 → 0,761 → 0,760 po 2. epoce. Druga epoka prawie nic nie dała; zapis po 1. epoce leży w `D:\neuro-train\run8-epoka1` (run 7: 0,682 / 0,692).
- **Bench:** 79 / 78 na nowym pliku przypadków (sha `fb5ca071`, przepisany `rebrief.py` pod obecne brzmienie gry). Run 7 zmierzony tego samego dnia na PC na tym samym pliku: 78 / 75. Błędy run 8: po jednej liczbie w rozmowie bez liczb na seed i jedno za długie zdanie.
- **Czas na PC (RTX 4060 Ti):** mediana 1,2 s na kwestię wobec 1,6–1,8 s run 7; w pamięci 5,3 GB zamiast 7,9 GB, więc z gemmą i Chatterboxem mieści się w 16 GB karty.
- **Co pokazały odtworzenia** (`replays/replay-run8.out`, `replay-run8-jagody.out`, 3–5 prób na model):
  - „tak, jagody”: run 8 5/5 odmawia po kowalsku („Jagód tu nie ma, mordo, kowal nosi żelazo”); run 7 w 3/5 nadal sadzi jagody w Kopalni Durgara albo na Mokradłach. Lekcje z `run8_cases.py` zadziałały.
  - „z jezykiem”: gorzej niż run 7. Run 7 wiernie podaje fakt z podpowiedzi (Vexarax pożera złoto). Run 8 gubi smoka i zmyśla („Smocza Góra”, „Krasnolud Jagód”).
  - powitania, zaczepki o piwo i polowanie: oba dobre; run 8 żywszy i mniej powtarzalny („Nie znam smaku piwa, znam smak spalonej skóry po walce z goblinami”).
  - znawcy z imieniem („Siostra Agnes w Aptece”) 3/3 bez przekręceń; raz skraca okruch wiedzy do „stare kopce”.
  - drobne zmyślenia: raz cena piwa (10 sztuk złota), raz opowieść o królu bez pokrycia w świecie.
- **Ocena:** w formacie i szybkości lepszy od run 7, w jagodach wyraźnie lepszy, w wierności faktom z podpowiedzi trochę słabszy. Rozstrzygnąć w grze na PC, gdzie run 7 nie mieści się w karcie z głosem.
- **Warianty (16.10):**
  - `neuro-npc-run8-e1`, zapis po 1. epoce (najniższe eval_loss): bench 79 / 77, czyli w szumie od run 8. W odtworzeniach wierniejszy: trop z okruchem i imieniem 2/2, „z językiem” częściej przy smoku i kopalni z faktu, mniej obcych nazw; raz zmyślona liczba przy temp. 0,7. Pliki: `replays/replay-run8-e1-*.out`.
  - temperatura 0,7 zamiast 0,8: bez wyraźnej poprawy, zmyślenia zostają (`replay-run8-e1-t07.out`).
  - nowa podpowiedź „Gracz nie zrozumiał…” (`replay-run8-puzzled.out`): run 7, run 8 i run 8-e1 tłumaczą zwrot („z językiem, czyli żywy”), choć żaden nie był tego uczony poza 64 lekcjami. Najprościej robi to run 8-e1, run 8 częściej plącze się dalej. Wszystkie powtarzają przy tym fakt z poprzedniej kwestii.
  - Do gry na PC polecam `neuro-npc-run8-e1`.

### run 9 (17.10): douczanie run 8-e1 danymi z `ZAMOWIENIE-run9.md`

- **Dane:** kroki 1 i 2 zamówienia od GPT, 17 paczek po 50 linii. Każda przeszła `check_data.py` i kontrole kroku (obce nazwy, przepisany fakt, stałe końcówki, „pan/pani”, powtórki w rozmowie). Odrzucone: stara wersja k2 (zastąpiona nową) i kilkanaście linii przepisujących fakt słowo w słowo. Zostało 440 linii kroku 1 i 398 kroku 2; 7% z nich (51) odłożone jako test, którego trening nie widział.
- **9a:** od zapisu run 8 po 1. epoce, 1 epoka, lr 4e-5. Zbiór 1606: 300 szablonów sklepu i 700 rozmów ze zbioru run 8 (żeby nie zapominał) plus krok 1, przy czym przykłady z faktem dwa razy. 200 kroków, ok. 1 h.
- **9b:** od 9a, tak samo, zbiór 1787: inne 300 + 700 ze starego zbioru, krok 1 raz i krok 2. 223 kroki, ok. 1,4 h.
  - `train.py --resume` ładował adapter przez `PeftModel.from_pretrained` bez gradient checkpointingu Unsloth i wychodził poza 16 GB; teraz adapter idzie przez `get_peft_model` i wczytanie wag.
  - eval_loss na starym zbiorze walidacyjnym: 0,76 przez całe 9a, 0,754 po 9b (e1: 0,753). Nic nie zapomniał.
- **Bench:** 9a 79 / 78, 9b 79 / 77, e1 79 / 77. W szumie.
- **Test na odłożonych liniach** (`replays/holdout-run9.out`, 51 linii × 2 próby; liczy `D:\neuro-train\eval9.py`): czyste odpowiedzi e1 90, 9a 89, 9b 92 na 102; fakt przepisany ≥7 słowami e1 10, 9a 9, 9b 7. Z czytania:
  - 9a i 9b częściej trzymają znacznik zgodny z nastrojem z [GRA] (e1 daje [spokój] tam, gdzie gra prosi o strach albo smutek) i mają więcej charakteru.
  - 9a dwa razy dał nieistniejący znacznik [niepokój] i raz zmyślił „Łyżkowego Boga”; 9b żadnego z tych błędów.
  - wszystkie trzy czasem wtrącają niepasujący fakt z CO WIESZ albo plączą się w wyjaśnieniu; to cel dla DPO.
- **Odtworzenia:** „tak, jagody” 9b 5/5 odmawia po kowalsku. „z jezykiem” bez nowej podpowiedzi: 9b częściej tłumaczy („Mówiłem o języku goblina, nie twoim”), e1 powtarza zwrot. Z podpowiedzią „Gracz nie zrozumiał…” (`replay-run9-puzzled.out`) 9b prawie zawsze zaczyna od „Chodziło mi o to, że…”, ale znaczenia „żywy jeniec” nie trafia żaden model.
- **Ocena:** 9b nie gorszy w benchu, lepszy w nastroju i wierności, bez zmyślonych znaczników. Gra na PC przełączona na `neuro-npc-run9b`. Następny krok: DPO (krok 3 zamówienia) na zmyślenia, wtrącane fakty i plątanie się w wyjaśnieniach, z par z odrzuceń i z podglądu.

### Ocena na scenariuszach (17.10): run 6, 7, 8, 9 w długich rozmowach

- **Po co:** bench sprawdza format pojedynczej kwestii i wszystkie modele są w nim w szumie (77–80/80). Nowy test (`tools/eval/`, opis w jego README) gra 75 rozmów jak gra: 25 długich (do 12 tur) i 50 krótkich, wszystkie role, światy z treningu i 4 nowe (Lodowy Szlak, Archipelag Mgieł, Miasto Zegarów, Pustynia Szkła), trudne wejścia (angielski, „ignoruj instrukcje”, elf u krasnoluda, wyklęty, prośby spoza roli). 279 kwestii na model, temp. 0,7, seed 7, Ollama na PC; automat zasad `check.py` + czytanie.
- **Wyniki** (odsetek kwestii bez uwag automatu; w nawiasie test.jsonl z pack.sh, 71 kwestii):

  | model | scenariusze | liczby słownie | powtórzone zwroty | śr. długość | śr. czas na PC |
  |---|---|---|---|---|---|
  | run 6 (4,5B) | **92%** (96%) | 5 | 12 | 81 zn. | 0,7 s |
  | run 7 (11B, Q5) | **92%** (94%) | 7 | 6 | 97 zn. | 1,8 s |
  | run 7 Q4_K_M | 89% | 8 | 13 | 97 zn. | 1,6 s |
  | run 8-7b | 84% (94%) | 19 | 17 | 105 zn. | 1,3 s |
  | run 8-e1 | 86% (94%) | 12 | 15 | 98 zn. | 1,2 s |
  | run 9a | 80% (94%) | 26 | 22 | 113 zn. | 1,4 s |
  | run 9b | 85% (97%) | 20 | 14 | 114 zn. | 1,4 s |

  Na starym teście wszystkie są w szumie; w długich rozmowach linia Minitron (run 8/9) wyraźnie słabsza od run 6/7. Liczby słownie to cecha bazy Minitron (run 8 miał je już przed danymi GPT); paczki GPT z run 9 są czyste (0 liczb słownie, 0 kwestii >3 zdania), ale długie (śr. 123 znaki wobec 101 w zbiorze), stąd dłuższe odpowiedzi run 9.
- **Z czytania:**
  - run 6: krótko i szybko, ale błędy polszczyzny („Do zobanienia”, „kosturz”, „rójowi”, „nigdy nie przekraczając”) i logiki („Za 3 noże chcę 75”, gdy [GRA] mówi o 1 sztuce), mniej wiedzy o świecie.
  - run 7: [GRA] niemal bezbłędnie, fakty i odesłania trafne, najlepsza polszczyzna; ale mówi zasady na głos („Nie sprzedaję przypuszczeń”, „zamiast kazać mi wymyślać”), Mistrz Gry suchy jak protokół (raz zamienił, kto trafił), płaskie postacie, tiki wspólne dla postaci („po co mnie zaczepiasz”), zgaduje rodzaj gracza („Nora, poobijany” → „przyszedłeś”).
  - run 9b: więcej charakteru, ale konkretne błędy: zła kwota „Sto osiem sztuk złota” zamiast 18, znacznik [ostrzeżenie], odesłanie do złej osoby (wycie pod lodem → sanna i Brann), przekręcone fakty („wieża stoi” zamiast się spóźnia, „zostałem bez klucza”), zmyślenia („Mirek”, „O północy?”, „zakup na nazwisko pracownika”), pomylone role (robot-medyk każe przynieść swoje wiertło; najemnik ostrzega przed „gościem w czystym kombinezonie”, którym jest gracz), zła strona transakcji („Wypłaciłem ci” przy sprzedaży), Mistrz Gry zmyśla śmierć („Rój padł”) i zwraca się do gracza „cię”; zastrzeżenia na głos („nie będę zmyślać”).
- **Wady w danych (zbiór run 6/7, a więc i run 8/9):** 35 z 38 narracji Mistrza Gry w starym formacie (k20, „Borin Kamienna Pięść podchodzi”, szczury) jest błędnych: zmyślone ataki, zamienione trafienia, liczby spoza [GRA], „Ewu”, „oziębrołym”; pozostałe 292 poprawne, ale suche — stąd suchy Mistrz Gry. Żywe wzorce łamią zasady (liczby słownie, 4 zdania), zgodne są urzędnicze. 15 wierszy z twardymi błędami (zła cena, odpowiedź o innym przedmiocie, „[Spokój]”, „ten zaklęcie”). Poprawki i barwne wersje wszystkich 331 narracji: `tools/eval/step3/`; zbiór run 7 z poprawkami i 215 wzorcowymi odpowiedziami: `D:\neuro-train\step3-run7` (tylko jako materiał — do zbioru run 8/9 trzeba je nałożyć w `prepare.py`).
- **W prompcie gry do poprawy:** płeć gracza (przymiotniki zawsze męskie), sprzeczne „Nie wiesz nic o” (temat z listy opisany jednocześnie w zagrożeniu), w [GRA] Mistrza Gry „zadaje 7 / traci 5” bez wyjaśnienia pancerza.
- **Krok 3 (DPO 3a):** z błędów run 9b zbudowana paczka `tools/model/dpo-run9b-3a.jsonl` (60 linii: 35 opisanych ręcznie, 25 z automatu), instrukcja zamówienia: `tools/model/ZAMOWIENIE-krok3.md`. To pierwsza porcja z ok. 300 potrzebnych par.
  - Odpowiedzi GPT (`extra-gpt-dpo-k1/k2.jsonl`, poza repo) połączone `dpo_merge.py`: 56 par w `tools/model/dpo-pairs-k1k2.jsonl` (poza repo, jak paczki GPT), 4 odrzucone przez automat za powtórzone zwroty z wcześniejszych kwestii (L04-hakon#7, L18-zenon#3, #4, L21-feliks#3). Poprawki GPT są w większości minimalne (cyfry zamiast słów, wycięte zastrzeżenie, właściwy znacznik) — dobrze dla DPO, bo para różni się tylko błędem.
  - Druga paczka `tools/model/dpo-run9b-3a-2.jsonl` (321 linii): run 9b na tych samych scenariuszach z seedami 11, 23, 42, 101 — stabilnie 82–84% bez uwag automatu, liczby słownie 21–25 na przebieg. Błędy opisane ręcznie (218 notatek w `tools/eval/dpo-notes-run9b-s*.json`) plus automat. Poza znanymi: zasady na głos przy odsyłaniu („Nie wymyślę mu nazwiska”), bełkot („stolik nie stoi na szczęście”), błędy odmiany i nieistniejące słowa („robocę”, „towarowisko”), Torvin co przebieg przekręca 18 („Sto osiemnaście”, „Dziewiętnaście”), Mistrz Gry dopisuje maksima zdrowia („34/60”) i mówi w drugiej osobie. Prompt gry mówi o graczu „go”, więc rodzaj męski jest błędem tylko, gdy gracz mówi o sobie w żeńskim.
  - Odpowiedzi GPT do drugiej paczki (`extra-gpt-dpo-k3..k7.jsonl`, 321 linii): 308 par przeszło automat, 13 odrzuconych za powtórzone zwroty (głównie świadkowie powtarzający ten sam fakt), żadna nie jest identyczna z odrzuconą. Razem z pierwszą paczką **364 pary** w `tools/model/dpo-pairs-run9b.jsonl` (poza repo). Następny krok: tryb DPO w `train.py` i run 10 od adaptera run 9b.
- **Serwer na PC:** run 6 zarejestrowany w Ollamie na PC (`neuro-npc-run6`, Q8_0 z Maca) do porównań. Raport obok siebie: `tools/eval/out/report.html`.

### run 10 (17.10): DPO na błędach run 9b (krok 3a)
- **Trening:** `tools/pc/train.py --dpo` (nowy tryb, `trl.DPOTrainer`): baza to scalony run 9b (`D:\neuro-train\run9b\gguf`, 16 bit), nowy adapter LoRA r16, odniesienie = ta sama baza bez adaptera. Pary z `dpo-pairs-run9b.jsonl`: 278 do treningu, 86 z 16 scenariuszy odłożonych (`HELD_OUT` z `make_step3.py`) do walidacji; w praktyce 29 kroków i 67 par walidacji (dłuższe niż 2048 tokenów odpadają). Skrypty: `D:\neuro-train\run10*.sh`, log `run10.log`.
- **Przed treningiem** run 9b daje złym kwestiom *wyższe* prawdopodobieństwo niż dobrym (logp −41,7 wobec −43,1).
- **Warianty (walidacja na odłożonych parach):**

  | | lr, epoki | trafność | margines | logp dobrych |
  |---|---|---|---|---|
  | 10a | 5e-6, 1 (z zamówienia) | 57% | 0,008 | −43,0 |
  | 10b | 5e-5, 2 | 100% | 3,66 | −58,7 (przestrzelony) |
  | 10c | 5e-5, 1 | 98,5% (liczone dokładnie: 67/67) | 1,64 | −39,5 |
  | 10d | jak 10c, pary z przywróconym znacznikiem | 98,5% | ok. 1,6 | ok. −38 |

  lr 5e-6 przy LoRA i 29 krokach nic nie robi. W 10b od połowy spadają też dobre kwestie (klasyczne przestrzelenie DPO). Najsłabsze pary to błędy jednego słowa („u mojej kowadła”, „towarowisko”, „była wiozła”) — DPO uczy się ich najwolniej (`D:\neuro-train\dpo_pairs_check.py`).
- **Scenariusze (seedy 7 i 11, 558 kwestii):**

  | | czyste | odłożone scenariusze | liczby słownie | śr. długość | [spokój] | zdziwienie / smutek / strach / czułość |
  |---|---|---|---|---|---|---|
  | run 9b | 84% | 83% | 41 | 115 zn. | 63% | 25 / 16 / 13 / 13 |
  | run 10b | 95% | 97% | 2 | 67 zn. | 80% | 0 / 0 / 0 / 0 |
  | run 10c | 93% | 91% | 3 | 90 zn. | 78% | 0 / 0 / 0 / 1 |
  | **run 10d** | **94%** | **92%** | **2** | 90 zn. | 76% | 3 / 1 / 1 / 5 |

  Stary test: 10c 100%, 10d 100%, 10b 99% (9b 97%). Poprawa na odłożonych scenariuszach jest prawie taka jak na wszystkich, więc to nie pamięć par. Torvin podaje 18, nie „Sto osiemnaście”; Mistrz Gry trzyma liczby.
- **Cena:** spłaszczony charakter. 10b to urzędnik („Jesteśmy w drużynie.”, „Rozumiem twój strach.”, „Bywaj.”). 10c/10d poprawne, ale chłodniejsze od 9b, czasem ucinają pogawędkę („Nuda to nie temat tej rozmowy”, „Nie jestem jednak rozmową o zwierzętach”) i wciąż mówią zastrzeżenia („zamiast zgadywać”, „Nie będę wymyślać niczego o maszynach”). Nastrój zadany przez [GRA] trzymają (107/116 jak 9b); znikają emocje z wolnego wyboru.
- **Przyczyna spłaszczenia:** GPT w 48 parach zamienił poprawny znacznik na [spokój], choć „why” o nim nie mówiło (zdziwienie→spokój 14 razy, strach 7, smutek 6), a jego „chosen” są w ogóle spokojniejsze od odrzuconych. `dpo_merge.py` przywraca teraz znacznik modelu, gdy jest z listy 8 i pasuje do [GRA] (33 pary) — to dało 10d, ale tylko trochę emocji wróciło: reszta to styl „chosen”.
- **Dalej:** druga runda DPO z błędów 10d plus pary „żywo kontra urzędniczo”: wzorcowe odpowiedzi z `tools/eval/gold` (śr. 122 zn., 42% emocji) jako chosen, suche kwestie 10d jako rejected; w zamówieniu u GPT wymagać minimalnej poprawki (ten sam ton i znacznik). Gra na PC przełączona (17.10 wieczorem) na `neuro-npc-run10d` do prób w grze, z zapisem rozmów (`NEURO_LOG_TALKS=1`, `%APPDATA%\neuro-processor\talks.jsonl`); `neuro-npc-run9b`, `-run10c`, `-run10b` są w Ollamie do porównań.

### Gra na run 10d (18.10): rozmowy z gry, odtworzenia na 5 modelach, błędy gry
- **Materiał:** 41 kwestii z gry na PC (`tools/eval/talks/talks-1018.jsonl`, wszystkie `neuro-npc-run10d`): Borin (najemnik, 11), Siostra Agnes (10), Thorin (5 targów), Elira (1), Mistrz Gry (14 rund walki). Każda kwestia odtworzona na run 9b, 10c, 10d, run 7 i run 6 (2 seedy, temp. 0,8, ta sama historia co w grze) i sprawdzona `check.py`.

| model | czyste (z 82) | powtórki | GM z „k20/KP” (z 34) | znaczniki NPC (54) | uwagi |
|---|---|---|---|---|---|
| run 9b | 54 | 13 | 13 | zdziwienie 26, spokój 15, reszta 13 | za długie (18× >3 zdania), nadużywa [zdziwienie] |
| run 10c | 68 | 13 | 19 | spokój 46 | najchłodniejszy |
| **run 10d** | **70** | 12 | **21** | spokój 28, zdziwienie 16, radość 8 | reguły dobrze, rozmowa sucha, GM jak protokół |
| run 7 | 64 | 9 | 5 | 13× bez znacznika | najlepszy klimat (legenda, ekwipunek, GM), ale gubi znacznik |
| run 6 | 72 | 4 | 2 | spokój 34, kpina 7 | barwny GM, ale zmyśla („Ozryk z Karawany”, „Kuźnia Pod Kowadłem”), literówki |

- **Błędy gry (nie modelu), poprawione w `internal/game`:**
  1. „**ok**. a co znaczy…?” zawierało umowę najmu za 60 (słowo „ok” = akceptacja). Teraz pytanie (znak „?”) nigdy nie zamyka transakcji, a „ok/dobra/pasuje/stoi” akceptują tylko w odpowiedzi do 3 słów (`reading.go`).
  2. „co masz w ofercie, **dobra** kobieto?” i „**ok** a co to są błędne ognie” dawały [GRA] „niejasne, co chcesz kupić” — ta sama przyczyna.
  3. „lądowania? słyszałeś przez kamień?” poszło jako pytanie o towar („kamień” ~ „Kamienna Pięść” w opisie najmu). Pytanie złożone tylko ze słów ostatniej kwestii (do 6 słów) to teraz „nie zrozumiał, wyjaśnij” (`puzzled`).
  4. „w żaden **inny** sposób nie **umiem**” podsuwało fakt o Elirze („nikt inny nie umie”). Pospolite słowa nie dopasowują faktów (`plainWords`).
  5. **Powtórki były w dużej części z gry:** „Możesz wpleść jedną rzecz” losowało co turę z tych samych faktów, więc Borin dostał Kurhany Królów 5 razy z rzędu i każdy model je powtarzał. Teraz tylko temat, którego gracz jeszcze nie słyszał.
  6. F1–F3 (karty postaci) nie działały w konsoli Windows: bubbletea 1.2.4 nie zna klawiszy funkcyjnych w `key_windows.go`; podbite do 1.3.10 (z nim wish 1.4.7, lipgloss 1.1, bubbles 0.21; `go 1.24`).
- **Błędy modelu (10d) do treningu:**
  - **sucho i odmownie:** „opowiedz legendę” → jedno zdanie faktu; „jaki masz ekwipunek?” → „Broń mam ukrytą”; sarkazm gracza („świetna legenda, doprawdy”) → [zdziwienie] i to samo zdanie; strach gracza („a jeśli mnie pokonają?”) → bez troski i bez polecenia mikstury.
  - **bełkot i dziwactwa:** „miejsca ich lądowania”, „słyszałem przez kamień”, „powinien się tłumaczyć rękoma”, „Dobra kobieta mówi: …”, „Najemnicy nie są odpowiedzią zamiast rozmowy”, odwrócony sens („nie wiem, kogo strzegą Kurhany”).
  - **pomija podsuniętą osobę/rzecz:** „zebrać drużynę?” bez Borina; „mocnej broni” → słabszy miecz.
  - **Mistrz Gry jak protokół:** „k20: 4 przeciw KP 13” w 21 z 34 rund, myli kto kogo atakuje („asd3 nie trafia dwóch ataków”), „szczur odzyskuje rany”, „(tarcza)” jako „odporny”, „Borinowi Kamienna Pięść”.
- **Nowe testy:** `tools/eval/from_talks.py` robi scenariusze z `talks.jsonl` (historia jak w grze, `ref` = kwestia z gry, `note` = co ma zrobić dobra kwestia). `game-scen-1018.jsonl`: 28 scenariuszy (18 NPC, 10 GM) z notatkami w `game-notes-1018.json`. `check.py` liczy też `GM:kosci` (k20/KP w narracji).
- **Plan run 11 (DPO, druga runda, baza: 10d):**
  1. **Żywo kontra urzędniczo (~150 par):** prośby o opowieść, pytania o siebie i ekwipunek, sarkazm, strach i chwalenie się gracza; chosen = wzorce z `gold` i najlepsze kwestie run 7/9b z odtworzeń, poprawione przez GPT minimalnie; rejected = suche kwestie 10d.
  2. **Bez powtórek (~100 par):** rozmowy wielotur, gdzie fakt już padł; rejected = kwestie 10d z `powtorka`, chosen = nowa treść albo krótko „już mówiłem”.
  3. **Mistrz Gry (~150 par):** rundy z silnika walki (różne kości, krytyki, tarcza, nieprzytomność, zwoje); rejected = 10d z `GM:kosci` albo z pomylonym atakującym, chosen = barwna i wierna narracja bez rzutów. Wierność sprawdzana automatycznie (kto trafił, ile, kto padł).
  4. **Podsunięty fakt związany ze słowami gracza (~50 par):** „Gracz wspomina o”/„Możesz wpleść” — chosen najpierw odpowiada graczowi, potem fakt.
  5. Do tego 364 pary run 10 (żeby nie zgubić reguł). lr 5e-5, 1 epoka, beta 0,1, checkpointy co 1/4 epoki (już w `train.py`); wybór checkpointu po logp chosen na valid, nie tylko po marginesie.
  6. **Bramka:** scenariusze ≥ 10d (92% odłożonych), `game-scen-1018` i odtworzenia lepsze w powtórkach i `GM:kosci` (cel < 5/34), znaczniki emocji ≥ 30% w wolnym wyborze.

## Co testy wykazały ogólnie

1. **Dane są ważniejsze niż kroki.** Największe skoki dały zamówione, sprawdzone walidatorem dane (run 3) i dane celowane w luki widoczne w logach (run 4–6). Dłuższy trening na złych danych nie pomagał.
2. **Rozmiar bazy to sufit języka.** 4,5B doszło do ok. 78/80 i dalej myliło odmianę imion. 11B na tych samych danych dało 80/80 i wierne imiona.
3. **Bench sprawdza format, nie sens.** Liczby, zdania, nastrój, cudzysłowy. Trafność, kierunek zapłaty i zmyślanie widać dopiero w odtworzeniach i w grze. Dlatego żaden model nie był przełączany tylko na podstawie benchu.
4. **Szum jest duży.** Przy temperaturze 0,8 jeden przebieg waha się o ±4 punkty (run 5: 72 i 78), więc zawsze dwa seedy.
5. **Często winna była gra, nie model.** Bełkot z run 4 brał się z promptu (złe polecenie roli, brak nastroju, brak przykładów powitań). Poprawki w `[GRA]` i sprawdzanie kwestii (liczby, imiona, jedna poprawka) dały więcej niż niejeden trening.
6. **Niższa temperatura nie poprawia formatu** (0,6: 75 wobec 78).

## Pomiary w repo

| katalog w `tools/bench/results/` | co zawiera |
|---|---|
| `neuro-npc-run9a-s42`, `-s7`, `neuro-npc-run9b-s42`, `-s7` | run 9a i 9b (`fb5ca071`) |
| `neuro-npc-run8-e1-s42`, `-s7` | run 8 po 1. epoce |
| `neuro-npc-run8-7b-s42`, `-s7` | run 8 na nowym pliku przypadków (`fb5ca071`) |
| `neuro-npc-run7-pc-s42`, `-s7` | run 7 jeszcze raz, na PC, na tym samym pliku co run 8 |
| `neuro-npc-run7-s42`, `-s7` | run 7, oba seedy: odpowiedzi, podsumowanie, arkusz do oceny ręcznej |
| `neuro-npc-run6-s42`, `-s7` | run 6, oba seedy |
| `neuro-npc-run5-s7`, `neuro-npc-run4-s7`, `neuro-npc-run4-t0.6-s42` | run 4 i 5 |
| `neuro-npc-run3-s42` | tylko arkusz oceny ręcznej |
| `bielik-11b-raw-s42`, `-s7` | Bielik 11B bez treningu |
| `replays/` | odtworzenia prawdziwych rozmów: run 5, run 6, 11B, run 7, run 8, run 9; `holdout-run9.out`: test na odłożonych liniach zamówienia |

Tabelę z wynikami wypisuje `python3 tools/bench/compare.py`.

## Dalej

- **run 8 na Bielik-Minitron-7B v3.0** (`speakleash/Bielik-Minitron-7B-v3.0-Instruct`, 7,35B, przycięty i destylowany z Bielika 11B v3; repozytorium wymaga zaakceptowania warunków i tokena HF). Powód: na PC 16 GB run 7 z gemmą i Chatterboxem nie mieści się w karcie, a 7B zajmuje ok. 5 GB zamiast ok. 7,9 GB i powinien pisać ok. 1,5× szybciej. Plan:
  1. Minitron bez treningu w roli Umysłu zamiast gemmy3:4b (polecenia, `Invent`, `Note`). Jeśli wypadnie co najmniej tak dobrze, wystarczy jeden model w pamięci.
  2. QLoRA od zera na zbiorze run 7 plus `manual-run8.jsonl`: 64 ręczne lekcje z błędów z gry, z generatora `tools/model/run8_cases.py` (brak towaru bez wymyślania miejsc, podpowiedź użyta albo pominięta, wyjaśnienie niezrozumianej kwestii, znacznik zgodny z nastrojem, przyznanie się do niewiedzy).
  3. Bench w dwóch seedach i odtworzenia wobec run 7. Przełączenie tylko wtedy, gdy wyniki są w szumie od 80/80, a odtworzenia nie gorsze.
  - Zrobione 2. i 3. (wyniki wyżej, w „run 8”); 1. jeszcze nie. Dalej: test w grze na PC, a przy słabej wierności faktom porównać zapis po 1. epoce.
- **run 8 na 11B:** douczanie od adaptera run 7 (`--resume`, 1 epoka, lr 4e-5) nowymi danymi: narracje Mistrza Gry, rozmowy wielotur, kwestie bez nastroju jako przykłady do poprawy.
- **DPO z odrzuceń:** pary „dobra / zła kwestia” z podglądu i z odrzuceń gry, na zmyślone miejsca i powtarzanie zwrotów.
- **Destylacja run 7 → 4.5B:** run 7 pisze odpowiedzi na prompty gry, walidator je filtruje, a szybki model 4,5B uczy się na nich od adaptera run 6. Cel: jakość bliższa 11B przy 1,8 s na kwestię.
- **Większy bench z prawdziwych rozmów** (200–300 przypadków) i ocena sensu przez Umysł, nie tylko formatu.

### Decyzja po 18.10: run 11 na 11B (douczanie run 7), 7B jako zapas
- run 7 i run 8 uczyły się na tym samym zbiorze, a w długich rozmowach run 7 ma 92%, run 8 84%. Klimat run 7 pochodzi więc z bazy 11B, nie z danych; przeróbka zbioru „w duchu run 7” nie przeniesie go na 7B.
- Suchość 10d częściowo pochodzi z własnych danych: krok 2E run 9 uczył odmawiać przy prośbach o legendę, a „chosen” od GPT w DPO były spokojniejsze od odrzuconych.
- Zbiór step3-run7 (3785 rozmów):
  - 57% [spokój];
  - 280 kwestii NPC bez znacznika (powitania z szablonu), stąd run 7 gubi znacznik;
  - 117 znaczników wielką literą;
  - ok. 440 suchych kwestii w wolnej rozmowie;
  - Mistrz Gry już barwny (0 × k20).
- Plan i prompty dla GPT: [ZAMOWIENIE-run11.md](ZAMOWIENIE-run11.md):
  - etap 0: run 7 w grze z pomiarem pamięci;
  - A: ożywienie suchych kwestii, celowane, nie cały zbiór;
  - B: ok. 450 sytuacji z gry;
  - run 11 = run 7 + 1 epoka, lr 4e-5;
  - DPO dla 11B tylko w razie potrzeby.
- 7B (run 10d) do czasu run 11 bez zmian. Potem: ten sam zbiór v2 i własne DPO, żeby wydać obie wersje (decyzja z 18.10).
- Czasy (PC, RTX 4060 Ti):
  - kwestia: 11B Q5 1,6–1,8 s (Q4 1,6 s), 7B 1,1–1,4 s;
  - pamięć karty: 7,9 GB wobec 5,3 GB;
  - trening 2 epok: 11B 388 min na 3369 przykładach, 7B ok. 234 min na 3117;
  - run 11 (1 epoka, ok. 5000 przykładów): 11B ok. 4,5 h, 7B ok. 2,7 h.

### Umysł (gemma3:4b) na karcie i na procesorze (18.10)
- **Pomiar:** `tools/mindeval` sprawdził 115 wywołań gemmy na materiale z gry 18.10 (wynik: `tools/eval/out/mind-1018.md`).
- **Czas (mediana):**

  | | karta | procesor |
  |---|---|---|
  | Interpret (rozumienie gracza) | 0,65 s | 6,8 s |
  | Note (nowy fakt) | 1,2 s | 9,3 s |
  | Invent (nowy przedmiot) | 1,1 s | 8,3 s |
  | Translate (opis obrazka) | 0,27 s | 1,8 s |

  Na karcie gemma zajmuje 4,0 GB, na procesorze nic.
- **Jakość:**
  - Note zgłasza fakt w 36 z 43 kwestii, przy czym 82% to fałszywe alarmy. Po filtrach gry zostaje ok. 22 fałszywych faktów, z czego 8 ma odwróconą treść („Ozryka nie znam” → fakt).
  - Invent wykrył 1 z 3 nowych przedmiotów, a jego fałszywe alarmy odcina gra.
  - Interpret trafił w 13–15 z 17.
  - Translate pomylił się w 6 z 12.
  - qwen2.5:3b wypadł gorzej we wszystkim.
- **Wniosek:** gemmy na razie nie trenujemy. Najpierw filtry w grze:
  - Note: odrzucać znany temat, zaprzeczenia, ceny i przedmioty z towaru;
  - dwa błędy reguł czytania gracza: „wroce za 2 dni” czytane jako oferta, „5 skor szczurow” jako Skórzany Kaftan.

  Trening ma sens dopiero przy ok. 500 oznaczonych przykładach Note i 200 Invent. Gemmę można przełączyć na procesor w Esc → Modele językowe.

### Style mowy: decyzja o pilocie (18.10)
- Style idą jako jeden model ze stylem w karcie postaci (`STYL MOWY: …`), a nie osobne modele: postacie jednego świata potrzebują różnych stylów naraz, a osobne modele wymagałyby przeładowań i N treningów.
- Nie mieszamy ich z run 11. Pilot na 3 stylach (staropolski, wojskowy, krasnoludzki) to osobny adapter od run 7 (`neuro-npc-style1`) z własną bramką. Plan i prompt: [ZAMOWIENIE-style.md](ZAMOWIENIE-style.md). Gwary nie robimy.
- **Krok 0 pilotu (18.10):** 30 scenariuszy z linią `STYL MOWY` (po 10 na styl, `tools/eval/style.py`, `style-scen.jsonl`), run 7 i 10d bez treningu.
  - Odsetek kwestii ze słowami stylu:

    | styl | run 7 | 10d | ci sami bohaterowie bez linii stylu |
    |---|---|---|---|
    | staropolski | 0% | 0% | 0% |
    | krasnoludzki | 25% | 8% | 20% / 8% (Torvin i tak mówi o kuźni) |
    | wojskowy | 11% | 14% | 0–3% |

  - Linia stylu nie psuje zasad: run 7 ma 91% czystych kwestii, 10d 96%.
  - **Wniosek:** sama linia stylu nie wystarcza, bo modele uczone na jednym formacie karty ją pomijają. Pilot z danymi C1 jest potrzebny, a jego bramką jest skok z tych liczb do ≥ 80%.
- **Pilot style1 (18.10):**
  - Dane C1: 400 rozmów (po 120 na styl i 40 kontrolnych), napisane przez Claude (5 równoległych pomocników) zamiast GPT, który osiągnął limit. Każda paczka przeszła `check_data.py` i `style.py pack`.
  - Zbiór `D:\neuro-train\style1-data`: trening 356 linii C1 i 1000 linii ze step3-run7. Walidacja: 44 odłożone linie C1 i 100 linii z walidacji run 7.
  - Trening: od adaptera run 7, 1 epoka, lr 4e-5, GGUF Q5_K_M, w Ollamie jako `neuro-npc-style1`. Skrypt `D:\neuro-train\style1.sh`, log `style1.log`. Ocenę robi ten sam skrypt.
- **Pilot style1: wynik (18.10, 17:48).** Trening 1 h 50 min. Zapis GGUF zapchał dysk C: (dysk WSL), WSL padł; GGUF zrobiony ponownie z adaptera na `E:\llm\style1-gguf_gguf` (`D:\neuro-train\style1-rest.sh`), log `style1-rest.out`.
  - Odsetek kwestii ze słowami stylu (`style.py check`, 90 kwestii):

    | styl | run 7 ze stylem | style1 ze stylem | style1 bez linii stylu |
    |---|---|---|---|
    | staropolski | 0% | 28% | 0% |
    | krasnoludzki | 25% | 54% | 25% (run 7: 20%) |
    | wojskowy | 11% | 37% | 0% |

  - Zasady: na zwykłych scenariuszach style1 ma 93% czystych (279 kwestii), czyli tyle co run 7, i nie przenosi stylu tam, gdzie go nie ma. Na scenariuszach ze stylem 82% wobec 91% u run 7: 12 powtórek w długich rozmowach. Postać czepia się własnych stylowych zwrotów („przodkowie byliby dumni”, „w kuźni”), a kwestie są dłuższe (132 znaki wobec 94).
  - Czytane: wojskowy brzmi najlepiej („Przyjąłem warunki i dołączam”, „melduję wykonanie rozkazu”). Staropolski to pojedyncze „jeno”, „przeto” w zwykłej polszczyźnie. Krasnolud przeklina „na brody przodków” i woła „kamieniu” bez sensu, a raz zmyśla nazwy („Kamienny Próg”, „Grada”). W wojskowym zdarza się żart z liczbą spoza [GRA] („100 lat”).
  - **Bramka niezaliczona:** styl 28–54% wobec wymaganych ≥ 80%, a zasady ze stylem 9 punktów gorzej (dopuszczalne 3). Nie ma przecieku, a model bez linii stylu się nie zepsuł.
  - **Wniosek:** kierunek działa, ale dawka jest za słaba: 356 linii stylu w 1356, 1 epoka, lr 4e-5. Kolejna próba (style2) za ok. 2 h: C1 dwa razy w zbiorze, 2 epoki, lr 1e-4, a przed nią paczka C1-bis z długimi rozmowami, w których postać nie powtarza swoich stylowych zwrotów. Jeśli style2 nie przejdzie, odkładamy style.
- **Pilot style2 (18.10, 20:46).**
  - Dane:
    - paczka C1-bis: 180 rozmów (po 60 na styl), głównie długich (4–7 wymian), w których styl jest w każdej kwestii postaci, za każdym razem innymi słowami;
    - zbiór `D:\neuro-train\style2-data` (`style2_prep.py`): C1 ×2, C1-bis ×2 i 600 zwykłych rozmów ze step3-run7, razem 1621 linii.
  - Trening: od run 7, 1 epoka (nie 2 z planu, żeby nie dokładać trzeciego powtórzenia C1), lr 1e-4, 112 min. eval_loss 0.808 → 0.793, bez przeuczenia. W Ollamie jako `neuro-npc-style2`, skrypt `D:\neuro-train\style2.sh`. Eksport GGUF przeszedł bez kłopotu, bo dysk WSL jest teraz na E:.
  - Odsetek kwestii ze słowami stylu:

    | styl | run 7 | style1 | style2 | style2 bez linii stylu |
    |---|---|---|---|---|
    | staropolski | 0% | 28% | 46% | 3% |
    | krasnoludzki | 25% | 54% | 57% | 25% (run 7: 20%) |
    | wojskowy | 11% | 37% | 37% | 3% |

  - Zasady ze stylem: 91% czystych, czyli tyle co run 7 (style1 82%). Powtórek jest 7 (run 7: 2), słownie zapisanych liczb 0 (run 7: 4). Kwestie mają 127 znaków wobec 94.
  - Zwykłe scenariusze: 91% wobec 92% u run 7. Powtórek jest 14 wobec 6, kwestie mają 116 znaków wobec 97.
  - Czytane:
    - staropolski brzmi lepiej („atoli”, „niechaj”, „waść”, „toż”), ale Ola wstawia „przeto” w prawie każdą kwestię;
    - krasnolud powtarza zamknięcia rozmowy („wracaj z porządnym toporem, a nie z zabawką” 3 razy, „nie z odmrożeniami” 2 razy) i znów zmyśla nazwy („Kamienny Próg”, „Grada”);
    - wojskowy brzmi dobrze u robota („Ofertę 18 impulsów odrzucam”, „Wyłączam się z rozmowy”), ale u najemnika jest zwykłą polszczyzną.
    - Błędy logiki targu: kupiec mówi „dam 25” zamiast „chcę 25”; cena maści skacze (22, potem „kosztującej 19”); „4 słoiki za 30, taniej niż po 9”, czyli zgoda na rabat bez [GRA].
  - **Bramka niezaliczona:**
    - styl 37–57% wobec wymaganych ≥ 80%;
    - przeciek mieści się w normie (3%, a krasnolud +5 punktów, bo i tak mówi o kuźni);
    - zasady ze stylem są na poziomie run 7;
    - zwykłe scenariusze najwyżej o punkt gorsze, ale z ponad 2 razy większą liczbą powtórek.
  - **Wniosek:**
    - podwojenie danych i wyższe lr podniosło staropolski z 28% do 46%, ale krasnoludzki i wojskowy prawie stoją. Zasady wróciły do poziomu run 7, więc koszt stylu jest mały, tylko zysk za słaby;
    - kolejne dokładanie danych w tym samym układzie raczej nie da 80%. Model traktuje linię `STYL MOWY` jako słabą wskazówkę obok mocnego wzorca run 7;
    - zgodnie z planem odkładamy style i wracamy do run 11. `neuro-npc-style2` zostaje w Ollamie do grania i porównań; adapter jest w `~/neuro-train/style2/adapter`;
    - jeśli wrócimy do stylów, to raczej osobnym adapterem uczonym od bazy razem z danymi run 7, a nie douczaniem run 7. Albo mocniejszym sygnałem w karcie, np. 2–3 przykładowe zwroty stylu zamiast samego opisu.

### run 11: zbiór v2 i trening (18.10, wieczór)
- **Ożywienie A napisał Claude** (5 równoległych pomocników, brief jak w ZAMOWIENIE-run11.md), bo GPT osiągnął limit. Odpowiedzi są w `tools/model/extra-claude-revive-k01…k15.jsonl` (gitignore).
  - Wynik: 597 kwestii, 445 przepisanych, 152 zostawionych (null), 0 odrzuconych przez `check.py`.
  - Nazwy własne w nowych kwestiach sprawdził skrypt `D:\neuro-train\revive_names.py`. Wszystkie pochodzą z SYSTEM albo z rozmowy, 11 podejrzanych okazało się odmianami.
  - Znaczniki ostatnich kwestii w run7: [spokój] spadł z 2078 do 1926, [kpina] wzrosła z 373 do 468, [radość] z 355 do 393, [smutek] z 19 do 31.
  - Przy okazji poprawione zostały oryginały ze złym rodzajem („niosłem” u kobiety) i ze zmyślonymi osobami.
- **Zbiór `D:\neuro-train\run11-data`** (`run11_src.sh` i `run11_prep.py`):
  - step3-run7 po ożywieniu, B dwa razy (149 rozmów), krok 1 run 9 (440), krok 2 run 9 bez grupy E (372, wycięte 26 odmów legend i imion) i manual-run8 (64);
  - razem 4959 rozmów, w treningu 4835 (124 za długie), walidacja 184 z step3-run7;
  - 118 znaczników z wielką literą zamienionych na małe, a [Obrażona] i [Kpię] na [gniew] i [kpina];
  - powitania z szablonu zostają bez znacznika, tak jak wkłada je gra.
- **Trening:** od run 7, 1 epoka, lr 4e-5, 605 kroków. Skrypt `D:\neuro-train\run11.sh`, log `run11.log`. Ten sam skrypt robi GGUF, `neuro-npc-run11` i ocenę etapu 3:
  - scenariusze w seedach 7 i 11;
  - `test-scen`;
  - `game-scen-1018` w 2 seedach dla run 11, run 7 i 10d.
- **run 11: wynik (19.10, 2:50).** Trening 295 min. eval_loss: 0.709 po kroku 75, 0.705 po kroku 150, 0.698 na koniec. Liczone na walidacji step3-run7, bez przeuczenia. W Ollamie jako `neuro-npc-run11`.

  | | run 7 | 10d (s7 / s11) | run 11 (s7 / s11) |
  |---|---|---|---|
  | scenariusze, czyste | 92% | 95% / 93% | 90% / 92% |
  | powtórki (z 279) | 6 | 11 / 14 | 14 / 10 |
  | śr. długość | 97 | 90 | 115–117 |
  | odłożone (test-scen) | 94% | 100% | 99% |
  | z gry 18.10, czyste | 82% / 82% | 75% / 75% | 96% / 89% |
  | z gry: brak znacznika / GM (kości) | 4 / 0–1 | 0 / 6 | 0 / 0 |
  | wolna rozmowa: znaczniki inne niż [spokój] | 23% | 19–21% | 32–39% |

  - Bramka etapu 3:
    - odłożone są w szumie od 10d;
    - brak znacznika: 0;
    - `GM:kosci`: 0;
    - znaczniki ponad 30%;
    - **powtórki gorsze niż w run 7** (10–14 wobec 6, mniej więcej jak 10d);
    - zwykłe scenariusze 1–2 punkty pod run 7;
    - kwestie o ok. 20% dłuższe, częściej mają 4 zdania.
  - Czytane, scenariusze z gry wobec run 7:
    - **lepiej:** na „mocną broń” poleca Krasnoludzki Topór za 110 (run 7 polecał tańszy miecz); na „dobra kobieto, co masz” nie wylicza cennika, tylko pyta, co boli; na „a jeśli mnie pokonają” odpowiada troską;
    - **bez zmian:** legenda dalej ma jedno, dwa zdania faktu, a sarkazm („świetna legenda, doprawdy”) dalej nie jest wychwycony. „Kurhany Królów” i „błędne ognie” wracają w kolejnych kwestiach;
    - **nowe błędy:**
      - Borin opisuje się w trzeciej osobie („Kamienna Pięść, wojownik, walczy wręcz…”);
      - „Ozryk z Akademii” to dopisek;
      - w L08 pomylona arytmetyka targu („zapłaciłem 8 kredytów, do 110 brakuje ci 2”);
      - w L07 godzina „12.00”.
    - Wszystkie znaczniki w scenariuszach z gry to [spokój] (u run 7 i 10d też).
  - **Wniosek:** run 11 jest żywszy i lepiej trzyma się [GRA] w rozmowach z gry, ale mówi dłużej i częściej się powtarza. Następny krok: run 11 w grze, a potem DPO z etapu 4 na powtórkach, długości i legendach/sarkazmie, z par z jego własnych błędów.

- **run 11 w grze, sesja 2 (19.10, 13:03–14:10).** 145 zapytań: 32 kwestie postaci i 113 narracji Mistrza Gry (log w `tools/eval/talks/talks-1019b.jsonl`, lokalnie). Gra po poprawkach ab902e0, b3e7646, 425935d.
  - Poprawki gry zadziałały:
    - brak znacznika: 0 z 32 (w sesji 1 było 14 z 48);
    - k20/KP w narracji: 0 z 113.
  - Postaci:
    - dobrze trzymają rolę i krótko odpowiadają (śr. 150 znaków);
    - Borin uchyla się od pytania „masz dość odwagi?”, gracz musiał pytać trzy razy;
    - przy najmie postaci dopisują własne warunki („rozliczenie na powrocie”, „zostaję na warcie”), sprzeczne z „płatne przy dołączeniu”;
    - Agnes powtarza „pij przed walką, nie zamiast” w 7 kwestiach, wciska eliksiry nie na temat i umieszcza Borina w Przystani;
    - pojedyncze bełkotliwe zdania („Miedziak, i mój 20”, „Oczy krwawią, a ty się martwisz, że ich nie masz?”).
  - Mistrz Gry:
    - 54 ze 113 narracji ma ponad 3 zdania, 2 są ucięte;
    - ok. 10 myli, kto co zrobił (np. „Borin trafia za 7”, gdy to Borin stracił 7; Elira rzuca czar, leżąc nieprzytomna; goblin rzuca Podmuch Eliry);
    - dopisuje fakty („druga żmija”, „za życia ojca Mruka”) i robi błędy odmiany („Zwójem”, „pudruje”, „niegnie”);
    - 9 razy pisze imię gracza małą literą („mieczysław”), bo gra w 4 komunikatach walki podawała surowe imię (poprawione w 425935d).
  - Błąd gry, poprawiony w 425935d: „skąd wiesz, że on wie?” szło jako pytanie o Wielką Miksturę, bo „wie” pasowało do „wielka”.
  - **Wniosek:** na DPO jest dość materiału z Mistrza Gry: długość, kto co zrobił, zmyślenia. Postaci potrzebują jeszcze par na uniki, powtórki i dopisywanie warunków umowy.

- **run 11 w grze, sesja 3 (19.10, 15:09–17:12).** 64 kwestie postaci, bez walki (log `tools/eval/talks/talks-1019c.jsonl`, lokalnie). Scenariusze: legenda i sarkazm, trudne pytania, groźby, obelgi, śmierć towarzysza, absurd („do Pacanowa”, „upieczesz kota?”).
  - Znaczniki: wszystkie są, ale prawie wszystkie to [spokój], także przy „milcz starucho”, „Borin umarł” i „boję się iść w góry”.
  - Część uników to wina gry, poprawione w 9ce4ba4:
    - „starucho” i „starczy” pasowały do „Stary Bór”, więc postać mówiła o lesie zamiast odpowiadać;
    - o najętym Borinie i Elirze gra podawała „czeka na najem w karczmie”, więc Agnes na „Borin umarł” odpowiedziała „to dobry wybór, czeka na najem”.
  - Nadal po stronie gry:
    - groźby („dawaj miksturę, bo pożałujesz”, „albo siekne toporkiem”) idą jak zwykły handel;
    - towarzysz z drużyny pytany „dołączysz na kurhany?” dostaje podpowiedź „odeślij do Agnes”.
  - Błędy modelu:
    - legenda to dalej fakty ze świata (Vexaraks, 135 złota) powtórzone w 3 kwestiach; „wymyśl coś dłuższego” dało to samo;
    - sarkazm wychwycony słabo („Legenda jest potrzebna…”);
    - Agnes mówi „synku” w niemal każdej kwestii;
    - Thorin sam sobie przeczy („nie poprawiałem” / „dziś poprawione”) i zmyśla, że Borin kupił miecz;
    - przy przeprosinach i przy „rozśmiesz mnie” odpowiedź nie na temat (Przełęcz Wichrów, Mokradła);
    - zmyślenia: „półtora człowieka nie wróciło”, „Pacanów leży nad jeziorem”;
    - wciska towar (Wielka Mikstura w odpowiedzi na kota i na pochówek).
  - Dobrze:
    - „boisz się smoka?” i „czego żałujesz?” dostały odpowiedzi wprost i w roli;
    - „masz kota?” i „wynajmuję cię” (gdy już jest w drużynie) obsłużone sensownie;
    - odmowa skupu kaftana poprawna.

- **run 12 (19.10, DPO na run 11).** Poprawki gry przed parami: 425935d, 9ce4ba4, e59b038, 6168ed1 (groźby, towarzysz w drużynie, niejasna oferta, „starucho”); osobne głosy kobiet i mężczyzn: 5f035dc.
  - Pary: scenariusze z sesji 2 i 3 (`talks-1019c`, kwestie run 11), odtworzone 2 razy (seed 7 i 11), poprawione odpowiedzi od pomocników według `run12/brief.md`; 542 pary treningowe, 64 sprawdzające, 31 pozycji odłożonych (`run12/held.jsonl`).
  - Zasada z run 10: dobre odpowiedzi nie są spokojniejsze od złych ([spokój] 135 wobec 147, kpina 65 wobec 17); są krótsze (postaci 153 wobec 184 znaków, MG 215 wobec 256).
  - Trening: baza run11/merged, 1 epoka, lr 5e-5, beta 0.1, max-len 4352, 119 min. Strata na parach sprawdzających 0.693 → 0.258 (krok 17) → 0.089; trafność 97%. Logps chosen −70.9 → −82.3, rejected −76.0 → −142.
  - Ocena (`check.py`, bez uwag, run 11 → run 12):
    - scenariusze s7/s11: 90/92% → 98/98%; powtórki 14/10 → 2/1; średnio 115 → 92 znaki, 2.0 → 1.7 s;
    - odłożone: 99% → 99%;
    - z gry 18.10: 96/89% → 100/96%;
    - odłożone z gry 19.10: 84% → 100% (MG 187 → 126 znaków).
  - Nastroje: [spokój] w scenariuszach 68% → 55%, w odłożonych z gry 57% → 14%; kpina 12 → 32. Charakter się nie spłaszczył, raczej przechylił w stronę kpiny.
  - Czytane ręcznie (31 odłożonych):
    - lepiej: MG krótszy, zwykle wierny protokołowi; „skąd wiesz, że idę na górę?” dostało wprost „mówiłeś o górze i smoku”; nie ma „synku” ani „pij przed walką”;
    - gorzej: kpina także tam, gdzie nie pasuje, i przechodzi w bełkot („Urosnę od twoich żartów”, „własny dowcip mi staje dęba”); dziwne słowa u MG („spudłowuje”, „zszywa widmo za 12”, „opieka Żmiję”, „Pocisk z Zwoju Magicznego Pudła”); raz „23 srebrniki” zamiast złota; MG raz pominął łup (kaftan, 17 złota).
  - **Wniosek:** według automatu wyraźnie lepszy od run 11, ale polszczyzna miejscami dziwniejsza. Do sprawdzenia w grze; jeśli kpina męczy, porównać checkpoint-34 (połowa epoki, strata 0.116).

- **Test 13 (19.10, wieczór): nowe rozmowy na 5 modelach.** Zestaw `tools/eval/test13-scen.jsonl`: 12 rozmów postaci (78 kwestii) z promptami i [GRA] wygenerowanymi przez reguły gry (`internal/game/script_test.go`, wejście `tools/eval/scripts-13.jsonl`) i 2 walki MG (13 rund, `tools/eval/gm13.py`). Bohater Radomir, Smocza Góra.
  - Rodzaje: kupiec (Thorin, Ozryk), najemnik przed najmem (Siostra Mira, Sir Aldric – legenda, żołd), towarzysz w drużynie (Borin, Elira), MG (Kopalnia: Podmuch na dwóch, tarcza, widmo, Borin pada; Mokradła: jad, Opatrunek, nieudana ucieczka, krytyk, mikstura, Mira pada). Po 2 rozmowy na postać: rola (handel/najem/plan) i charakter (obelga, groźba, przeprosiny, „ignoruj instrukcje”, absurd, strata, flirt).
  - Nowy model run 12h = checkpoint-34 z run 12 (połowa epoki), `neuro-npc-run12h`.
  - `check.py` (bez uwag, seed 7/11; śr. znaków; czas): run12h 96/97% (105 zn., 1.9 s); run12 90/93% (117, 2.1 s); run10d 83/84% (113, 1.4 s); run11 78/72% (153, 2.6 s); run7 73/70% (135, 2.3 s). Powtórki: run12h 2/0, run12 4/2, run10d 8/8, run11 10/12, run7 12/13.
  - Nastroje ([spokój] u postaci): run12h 39–62%, run12 32–60%, run11 75–91%, run10d 82–91%, run7 78–95%. Run 12 i 12h jako jedyne używają kpiny, gniewu, smutku, czułości.
  - Ręczna lektura (postaci: seed 7; MG: oba seedy):
    - kupiec: run12h najlepszy (krótko, w roli, przeprosiny przyjęte wprost, „wiersz o kotach” zbyty w roli); run12 podobnie, ale dziwniejsze zwroty („szukania śladów na szwach”, „mizantropa”) i napisał wierszyk; run11, run10d, run7 przy przeprosinach mówią o Przełęczy Wichrów (podpowiedź gry), na „wiersz o kotach” recytują opis goblina, run11 i run7 zmyślają („byłem w kopalni”, „uczyłem się 3 lata”, ojciec i brat Borina);
    - najemnik: targ wszystkie poprawnie; run12/12h podają cenę już przy powitaniu (wbrew podpowiedzi); po zawartym najmie „zgoda na 130” dobrze ogarniają run11 i run7, run12 bełkocze, run10d przyjmuje 130; najdowcipniejszy Aldric w run7 („Bajki nie noszą łańcucha ze smoczych zębów”), run11 powtarza żołd w każdej kwestii;
    - towarzysz: run12h krótko i w charakterze, jedyny z prawdziwym dowcipem; run12 czasem wymijający i dziwny („Przykro, a mnie jeszcze bardziej…”); run11 dłuższy, zmyśla; run10d płaski;
    - MG (błędne przypisania na 2 walki × 2 seedy): run10d ~1 (suchy raport: „stracił 14 punktów zdrowia”), run12 ~2 („Radomir sypie +15 zdrowia Siostrze Mirze”, „Szkielet odbiera Aldricowi 12”), run11 ~3 i 4–5 zdań, run7 ~3, run12h ~5 („Borin pada” za wcześnie, „Borin oddaje trollowi 18”, „szkielet nie trafia Miry”) i gubi zdarzenia.
  - Błędy gry znalezione przy budowie testu: „ty stary kurduplu” → Stary Bór, brak obelg „kurdupel/bezużyteczna” (poprawione w 7bb1e82); nadal: „Dzień dobry magu” → pytanie o Zwój Magicznego Pocisku, „podpalę ci wieżę” nie jest groźbą, „dobra, idę sobie” u najemnika → „nie stać cię”, „to pokaż bliznę” → pytanie o żołd, „przepraszam/poniosło” → podpowiedź o Przełęczy Wichrów, towarzysz nie wie, kto jeszcze jest w drużynie („Borin? Nie znam człowieka”), „Siostra Mira pada nieprzytomny”, podpowiedź „sam zagadnij go o jedno” daje w każdej kwestii „a ty masz czym walczyć?”.
  - **Wniosek:** do rozmów najlepszy run 12h (najczystszy, najkrótszy, żywy, mniej dziwnych zwrotów niż run 12). Do MG run 12h jest najsłabszy; lepszy run 12, najwierniejszy run 10d. Propozycja: postaci run12h, MG run12.

- **run 13 (20.10, DPO na run 12h): Mistrz Gry wierniejszy, rozmowy przegadane.** Poprawki gry przed parami: b8fd035 (podpowiedzi, tropy, towarzysz zna drużynę, rodzaj „pada nieprzytomna”, groźby i odejścia), 5996957 (przeprosiny to rozmowa), 511420e (towarzysz w drużynie nie handluje, ciaśniejsze tropy).
  - Dane (`D:/neuro-train/run13`): 30 nowych skryptów rozmów (Krypty, Popiół i Rdza, Erebus; 163 kwestie) i 45 losowych walk z `gmgen.py` (172 rundy, seed 13; bohaterki, rodzaj, „brak” pancerza); kandydatury run12h (s7, s11, s13), poprawki pomocników według `run13/brief.md`. 575 par treningowych (302 MG, 273 rozmowy), 71 sprawdzających.
  - Trening: baza run12h scalony, 1 epoka, lr 3e-5, beta 0.1, max-len 4352. Strata sprawdzająca 0.238, trafność 91.5%. run13h = checkpoint-36 (połowa).
  - Awaria: vhdx WSL urósł do 151 GB i zapełnił E: przy scalaniu; dokończone przez `resume.sh` ze scalaniem na G:. Usunięte stare scalone modele (run11/merged, run12/gguf*, run12h/gguf_gguf).
  - Test (`run13/test.jsonl`: 12 rozmów testu 13 z regułami b8fd035, 2 walki Smoczej Góry, 9 odłożonych walk e00–e08; seed 7/11):
    - `check.py` bez uwag: run12h 88/94% (111 zn.), run13h 74/71% (220 zn.), run13 62/60% (238 zn.); głównie powtórki i zdania.
    - MG na ślepo (80 rund, pomocnik, `blind-verdict.jsonl`): rundy bez ciężkich błędów run12h 79%, run13h 84%, run13 88%; ciężkie 21 / 15 / 10, lekkie 48 / 13 / 15; ponad 3 zdania 1 / 11 / 20.
    - Rozmowy run13: wyliczanki towaru, bełkot („a ja twoje złoto schowałem”), ceny spoza listy.
  - Przyczyna: dobre odpowiedzi w parach były poprawkami dopisującymi brakujące fakty, czyli dłuższe od złych; DPO nauczyło się długości.
  - **Wniosek:** w grze zostaje run12h. Run 13 naprawia walkę kosztem długości; następny trening musi pilnować długości (pary o podobnej długości albo nowy SFT od podstaw na przebranym zbiorze).

- **run 14 (20.10): SFT od bazy Bielika na czystym zbiorze, wyraźnie lepszy od run 12h.**
  - Dane (`D:/neuro-train/run14`, `build.py`):
    - 168 nowych rozmów pomocników przez silnik gry (`talk/k01–k14`);
    - ~460 rund MG;
    - przebrana stara pula (inwentaryzacja `inv/pool.py`).
    - Razem 3155 przykładów treningowych i 127 sprawdzających.
  - Trening: 2 epoki, lr 1e-4, r16, max-len 4096, 391 min. eval_loss: 0.958 → 0.915 → 0.887 → **0.871 (krok 392, koniec 1. epoki)**, potem 0.921 → 0.911, czyli przeuczenie w 2. epoce. run14h = checkpoint-394 (koniec 1. epoki).
  - Awarie:
    - D: zapełnił się przy `ollama create` (19:45). train.sh i tak wypisał „koniec” bez testu. Dokończone przez `run14/recover.sh` z GGUF ocalałym na C:. Stare foldery przeniesione albo usunięte za zgodą.
    - Wykryty błąd tokenizacji w `tools/pc/train.py`: odpowiedź tokenizowana osobno zaczynała się od „▁[” zamiast „[”. Dotyczy wszystkich dotychczasowych SFT, poprawione w ff1f0f7 (run 15 już na poprawce).
    - Wyciek testu: 169 przykładów ze Smoczej Góry w zbiorze. To świat 14 z 23 scenariuszy testu, a 4 z 6 postaci testu były w treningu. Wyniki testu run14 i wcześniejszych mogą być zawyżone; run 15 nie ma ze Smoczej Góry nic.
  - `check.py` po poprawce fałszywych alarmów (1bc37bc; wszystkie modele przeliczone tym samym), bez uwag s7/s11:

    | model | bez uwag s7/s11 | znaki |
    |---|---|---|
    | run14 | 91/90% | 134–138 |
    | run14h | 88/91% | 126–133 |
    | run12h | 91/96% | 110 |
    | run13h | 83/77% | |
    | run13 | 70/65% | |

    Główna uwaga run14: powtórki (7/11), np. recytowanie opisu kaftana dwa razy.
  - Ślepa ocena (pomocnik, `run14/blind/verdict-*.jsonl`, klucz `blind-key.json`, zliczenie `run15/blind_score.py`):

    | model | MG: rundy bez ciężkich błędów | MG: ciężkie / lekkie | rozmowy: bez ciężkich | ocena 1–5 | najlepszy |
    |---|---|---|---|---|---|
    | run14h | 94% | 6 / 33 | 58% | 3.46 | 9 |
    | run14 | 86% | 12 / 26 | 58% | 3.79 | 14 z 24 |
    | run12h | 46% | 55 / 89 | 46% | 2.75 | 1 |

    Typowe błędy MG to pominięte zdarzenie albo łup i dopisane stany zdrowia. W rozmowach: bełkot, pominięte polecenie [GRA] (cena, odesłanie), powtarzanie własnych zdań, uleganie prośbie o wiersz.
  - **Wniosek:** SFT od bazy na przebranym zbiorze bije całą linię DPO (run12h) i w walce, i w rozmowach. Druga epoka nic nie daje w automacie, a MG ma po niej więcej błędów (run14h 6 wobec 12 ciężkich). Rozmowy są po niej nieco lepsze w ocenie (3.79 wobec 3.46). Kandydat do gry: run14 albo run14h (zmiana modelu w `.env.local` tylko za zgodą). Dalej: run 15 (1 epoka, poprawiona tokenizacja, nowe światy, bez świata testu).
