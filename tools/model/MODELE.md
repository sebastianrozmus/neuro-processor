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
- **Serwer na PC:** run 6 zarejestrowany w Ollamie na PC (`neuro-npc-run6`, Q8_0 z Maca) do porównań. Raport obok siebie: `tools/eval/out/report.html`.

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
