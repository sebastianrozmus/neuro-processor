window.GUIDE = {
 "runs": [
  {
   "id": "bielik45-raw",
   "date": "2026-10-06",
   "title": "Gotowe modele bez treningu",
   "base": "gemma3:4b oraz Bielik 4.5B v3.0 (Q8_0)",
   "method": "inne",
   "changes": [
    "Próba grania na gotowych modelach, bez douczania",
    "gemma3:4b jako jedyny model, potem Bielik 4.5B jako Głos"
   ],
   "result": "gemma: kwestie drewniane. Bielik 4.5B: żywszy polski, ale zmyślone ceny, liczebniki słownie, emoji, wycieki znaczników gry. Liczbowego wyniku brak (bench powstał później).",
   "verdict": "nie_pomoglo",
   "in_game": true,
   "note": "gemma3:4b została w grze jako Umysł aż do run21 (04.11). Bielik bez treningu odrzucony.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 0",
    "D:/dev/neuro-processor/tools/model/MODELE.md#Bielik 4.5B bez treningu"
   ]
  },
  {
   "id": "run1",
   "date": "2026-10-08",
   "title": "Pierwsza LoRA na Macu, na logach gemmy",
   "base": "Bielik 4.5B v3.0 (MLX 8-bit)",
   "method": "SFT",
   "changes": [
    "Pierwszy trening LoRA w mlx-lm na Macu (24 warstwy, rank 8)",
    "Dane: synteza z gry + surowe logi gemmy (44% zbioru, 320 wierszy)",
    "400 kroków, lr 1e-4, ok. 45 min"
   ],
   "result": "Val loss 2,81 → 1,75, test ppl 5,9. Model powielał drewniany styl gemmy. Import do Ollamy dawał śmieci ([UNK_BYTE_…]).",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Działał tylko tymczasowo przez serwer MLX. Przy okazji powstał podział na Umysł i Głos, który został.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 1",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 3"
   ]
  },
  {
   "id": "run2",
   "date": "2026-10-09",
   "title": "Mały zbiór poprawiony ręcznie",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "Wyrzucone surowe logi gemmy",
    "355 wierszy: 314 z syntezy przepisanych ręcznie (269 z 275 odpowiedzi) + 41 własnych",
    "240 kroków"
   ],
   "result": "Val loss 2,94 → 1,85, test ppl 4,59. Ceny trzyma (3/3), ale odmiana się sypie („plombów”, „filt”). Kontroferty nie umiał: w zbiorze było jej tylko ok. 10 przykładów.",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Przy okazji naprawiona droga do Ollamy przez llama.cpp (GGUF q8_0).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 2",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 5"
   ]
  },
  {
   "id": "run3",
   "date": "2026-10-09",
   "title": "Pierwszy grywalny: dane zamówione u GPT",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "Kontrakt promptu gry spisany co do znaku (PROMPT.md)",
    "Ok. 1700 rozmów od GPT, każda przez walidator; 660 szablonowych odrzuconych",
    "Zbiór 2355 wierszy, od zera, 1600 kroków, ok. 4 h"
   ],
   "result": "Val loss 2,94 → 1,41, test ppl 3,85. Bench 71/80 (seed 42). Ceny 6/6, nastroje są, polski dobry; słabe pogawędki najemnika.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Pierwsza wersja nadająca się do gry (Mac). Dokładna data przełączenia: brak danych.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 3",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 6"
   ]
  },
  {
   "id": "run4",
   "date": "2026-10-10",
   "title": "Dane celowane i powstanie benchu",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "Najpierw poprawka gry: najemnik i towarzysz dostawali polecenie kupca",
    "+400 wierszy celowanych (200 najemnik, 120 towarzysz, 80 kupiec)",
    "80 przypadków odłożonych na stały bench",
    "Douczanie od run 3, 700 kroków, lr 5e-5, ok. 70 min"
   ],
   "result": "Bench 78 / 79 (było 71). W grze „siema mordo” dawało bełkot; winny był prompt gry, naprawiony w kodzie.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Od tego runu gra loguje pełne prompty, a odtworzenia (replay) są podstawowym testem.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 4",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 7"
   ]
  },
  {
   "id": "run4-t0.6",
   "date": "2026-10-10",
   "title": "Ten sam model, temperatura 0,6",
   "base": "jak run4",
   "method": "inne",
   "changes": [
    "Tylko niższa temperatura próbkowania (0,6 zamiast 0,8)"
   ],
   "result": "Bench 75/80 wobec 78. Błędy: za dużo zdań, liczebnik „dwie”, brak kwoty 14.",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Zostało 0,8.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 4 z temperaturą 0,6"
   ]
  },
  {
   "id": "run5",
   "date": "2026-10-10",
   "title": "Luki z logów gry",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "+550 wierszy z luk w logach: 160 powitań, 120 zaczepek, 100 potwierdzeń najmu, 60 pytań o towar, 60 odesłań, 50 odmów",
    "Potoczne wejścia, także bez polskich znaków",
    "Od run 4, 800 kroków, lr 5e-5; podział 3070/171/64"
   ],
   "result": "Val loss 1,22 → 1,09, test ppl 2,99. Bench 72 / 78 (w szumie wobec 78/79). Odtworzenia: powitanie 3/3 (było 0/3), odesłanie 3/3 (było 1/3), kierunek zapłaty 12/12 (było ok. połowa źle).",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Przełączony mimo równego benchu, bo odtworzenia wyraźnie lepsze. Przy wtapianiu zabrakło pamięci.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 5",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 9"
   ]
  },
  {
   "id": "run6",
   "date": "2026-10-11",
   "title": "Tropy i dopytania",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "+320 wierszy: odesłania z okruchem wiedzy i imieniem znawcy, dopytania, zaczepki bez liczb, plotki na fakcie",
    "Od run 5, 600 kroków, lr 4e-5; zbiór 3369/185/71"
   ],
   "result": "Test ppl 3,18. Bench 76 / 78. Odtworzenia: trop 3/3, dopytanie 3/3, plotki na fakcie 2/3. Słabe: odmiana imion („Agness”, „Kupcz”) — sufit bazy 4,5B.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Był w grze na Macu, potem został jako model zapasowy (powrót).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 6",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 10"
   ]
  },
  {
   "id": "bielik11b-raw",
   "date": "2026-10-15",
   "title": "Bielik 11B bez treningu (sprawdzian)",
   "base": "Bielik 11B v2.3 (Q4_K_M)",
   "method": "inne",
   "changes": [
    "Sprawdzenie większej bazy przed treningiem na PC"
   ],
   "result": "Bench 8 / 6. Za dużo zdań (70–74 na 80), cudzysłowy i dopiski (36–42), odpowiedzi ucięte (29–32). Ale wyraźnie lepsza polszczyzna. Mediana 13,2–13,6 s.",
   "verdict": "niejasne",
   "in_game": false,
   "note": "Formatu gry nie znał wcale, ale uznany za dobrą bazę do treningu.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#Bielik 11B bez treningu"
   ]
  },
  {
   "id": "run7",
   "date": "2026-10-15",
   "title": "Przesiadka na Bielika 11B i PC",
   "base": "Bielik 11B v2.3 (4 bity)",
   "method": "SFT",
   "changes": [
    "Pierwszy trening na PC: RTX 4060 Ti 16 GB, WSL2 + Unsloth, QLoRA",
    "Od zera, ten sam zbiór co run 6 (3369/185/71)",
    "2 epoki, lr 1e-4, rank 16; GGUF Q5_K_M 7,9 GB"
   ],
   "result": "Bench 80 / 80 — pierwszy raz bez błędu formatu. Imiona bez przekręceń (3/3). Ale ok. 3× wolniej na Macu (5,2 s wobec 1,8 s) i 7,9 GB pamięci.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Model gry od 15.10 (Mac i PC). Na PC z gemmą i Chatterboxem zajmował całą kartę 16 GB.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 7",
    "D:/dev/neuro-processor/tools/model/HISTORIA.md#Krok 11"
   ]
  },
  {
   "id": "run8",
   "date": "2026-10-16",
   "title": "Mniejszy Minitron 7B (+ run8-e1 po 1. epoce)",
   "base": "Bielik-Minitron-7B v3.0",
   "method": "SFT",
   "changes": [
    "Mniejsza baza, żeby zmieścić się w karcie z głosem",
    "Zbiór run 7 + 64 ręczne lekcje z błędów gry (3117/169/64)",
    "2 epoki, lr 1e-4, r16, ok. 4 h; GGUF 5,3 GB",
    "Wariant run8-e1 = zapis po 1. epoce"
   ],
   "result": "Bench 79 / 78 (run8-e1: 79 / 77; run 7 na tym pliku 78 / 75). Mediana 1,2 s wobec 1,6–1,8 s run 7. „Tak, jagody” 5/5 dobrze (run 7 3/5 źle), ale słabsza wierność faktom. Później w długich rozmowach: 84–86% czystych wobec 92% run 7.",
   "verdict": "mieszane",
   "in_game": true,
   "note": "run8-e1 polecony do gry na PC 16.10; zastąpiony przez run9b 17.10.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 8",
    "D:/dev/neuro-processor/tools/model/MODELE.md#Ocena na scenariuszach (17.10)"
   ]
  },
  {
   "id": "run10d",
   "date": "2026-10-17",
   "title": "Pierwsze DPO (10a–10d) na błędach run9b",
   "base": "Minitron 7B (scalony run9b)",
   "method": "DPO",
   "changes": [
    "Nowy tryb DPO w train.py (trl.DPOTrainer)",
    "364 pary (poprawki GPT do błędów run9b): 278 treningowych, 86 odłożonych",
    "Warianty lr: 10a 5e-6, 10b 5e-5 × 2 epoki, 10c 5e-5 × 1, 10d jak 10c z przywróconym znacznikiem nastroju"
   ],
   "result": "10a: trafność 57% (nic). 10b: przestrzelony. 10d: scenariusze 94% czystych (9b 84%), liczby słownie 41 → 2. Ale emocje prawie znikły (zdziwienie/smutek/strach/czułość 25/16/13/13 → 3/1/1/5 w 10d, 0/0/0/0 w 10b).",
   "verdict": "mieszane",
   "in_game": true,
   "note": "Gra na PC na run10d od 17.10 wieczorem. Zasady lepsze, charakter urzędniczy.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 10 (17.10)",
    "D:/dev/neuro-processor/tools/model/MODELE.md#Gra na run 10d (18.10)"
   ]
  },
  {
   "id": "run9b",
   "date": "2026-10-17",
   "title": "Douczanie Minitrona danymi GPT (9a, 9b)",
   "base": "Bielik-Minitron-7B v3.0 (od run8-e1)",
   "method": "SFT",
   "changes": [
    "838 nowych linii od GPT (kroki 1–2 zamówienia) + 2000 linii powtórki starego zbioru",
    "9a: 1 epoka, lr 4e-5, 200 kroków; 9b: od 9a, 223 kroki",
    "Naprawione --resume, które wychodziło poza 16 GB"
   ],
   "result": "Bench: 9a 79 / 78, 9b 79 / 77. Odłożone linie: czyste 9b 92/102 (e1 90). Scenariusze długie: 9b 85% (run 7 92%), liczby słownie 20.",
   "verdict": "mieszane",
   "in_game": true,
   "note": "Gra na PC na run9b od 17.10 (krótko, tego samego wieczoru run10d).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 9 (17.10)",
    "D:/dev/neuro-processor/tools/model/MODELE.md#Ocena na scenariuszach (17.10)"
   ]
  },
  {
   "id": "style1",
   "date": "2026-10-18",
   "title": "Pilot stylów mowy (style1, style2)",
   "base": "Bielik 11B v2.3 (od run7)",
   "method": "SFT",
   "changes": [
    "Linia STYL MOWY w karcie postaci: staropolski, wojskowy, krasnoludzki",
    "style1: 356 linii stylu (Claude) + 1000 starych, 1 epoka, lr 4e-5",
    "style2: dane stylu ×2 + 180 długich rozmów, lr 1e-4, 1 epoka"
   ],
   "result": "Słowa stylu: style1 28/54/37%, style2 46/57/37% (bramka ≥80%). style1 zasady ze stylem 82% (run 7 91%); style2 91%, ale 2× więcej powtórek.",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Bramka niezaliczona dwa razy, style odłożone. neuro-npc-style2 został w Ollamie do porównań.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#Style mowy: decyzja o pilocie (18.10)"
   ]
  },
  {
   "id": "run11",
   "date": "2026-10-19",
   "title": "Douczanie run7 ożywionym zbiorem v2",
   "base": "Bielik 11B v2.3 (od run7)",
   "method": "SFT",
   "changes": [
    "597 suchych kwestii przepisanych przez Claude (445 zmienionych), bo GPT miał limit",
    "Zbiór 4959 rozmów (4835 w treningu), poprawione znaczniki wielką literą",
    "1 epoka, lr 4e-5, 605 kroków, 295 min"
   ],
   "result": "eval_loss 0,709 → 0,698. Scenariusze 90/92% (run 7 92%), z gry 18.10: 96/89% (run 7 82%). Brak znacznika 0, k20 w MG 0. Ale dłuższy (115–117 znaków wobec 97) i więcej powtórek (10–14 wobec 6).",
   "verdict": "mieszane",
   "in_game": true,
   "note": "Grany w sesjach 19.10; 21.10 w .env.local nadal był run11.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 11: zbiór v2 i trening (18.10, wieczór)"
   ]
  },
  {
   "id": "run12",
   "date": "2026-10-19",
   "title": "DPO na run11 z sesji gry (run12, run12h)",
   "base": "Bielik 11B v2.3 (scalony run11)",
   "method": "DPO",
   "changes": [
    "542 pary z sesji gry 19.10, poprawki pomocników",
    "Dobre odpowiedzi nie spokojniejsze od złych, za to krótsze",
    "1 epoka, lr 5e-5, beta 0,1, 119 min; run12h = checkpoint w połowie epoki"
   ],
   "result": "Automat: scenariusze 90/92% → 98/98%, powtórki 14/10 → 2/1. Trafność na parach 97%. Ale kpina tam, gdzie nie pasuje, i dziwne słowa. Później ślepa ocena (run14): run12h tylko 46% rund MG bez ciężkich błędów.",
   "verdict": "mieszane",
   "in_game": false,
   "note": "W MODELE.md: „w grze zostaje run12h” (20.10), ale 21.10 .env.local miał run11 — faktyczne przełączenie niepewne.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 12 (19.10, DPO na run 11)",
    "D:/dev/neuro-processor/tools/model/MODELE.md#Test 13"
   ]
  },
  {
   "id": "run13",
   "date": "2026-10-20",
   "title": "DPO na run12h: MG wierniejszy, rozmowy przegadane",
   "base": "Bielik 11B v2.3 (scalony run12h)",
   "method": "DPO",
   "changes": [
    "575 par (302 MG, 273 rozmowy) z nowych skryptów i losowych walk",
    "1 epoka, lr 3e-5, beta 0,1; run13h = połowa"
   ],
   "result": "MG na ślepo: rundy bez ciężkich błędów 79% (12h) → 88% (13). Ale automat 88/94% → 62/60%, długość 111 → 238 znaków; wyliczanki i bełkot.",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "DPO nauczyło się długości, bo dobre odpowiedzi były dłuższe. Awaria: dysk WSL urósł do 151 GB.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 13 (20.10, DPO na run 12h)"
   ]
  },
  {
   "id": "run14",
   "date": "2026-10-20",
   "title": "Reset: SFT od bazy na przebranym zbiorze (run14, run14h)",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "Trening od czystej bazy zamiast łańcucha DPO",
    "Przebrana pula + 168 nowych rozmów przez silnik gry + ~460 rund MG; 3155 przykładów",
    "2 epoki, lr 1e-4, r16, 391 min; run14h = koniec 1. epoki"
   ],
   "result": "eval_loss najniżej 0,871 po 1. epoce, potem rośnie. Ślepa ocena MG: run14h 94%, run14 86%, run12h 46% rund bez ciężkich błędów; ocena rozmów 3,46 / 3,79 / 2,75.",
   "verdict": "pomoglo",
   "in_game": false,
   "note": "Wynik zawyżony wyciekiem testu (169 przykładów ze świata testu). Polecany do gry, przełączenia nie odnotowano.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 14 (20.10)"
   ]
  },
  {
   "id": "run15",
   "date": "2026-10-21",
   "title": "SFT 1 epoka, bez świata testu",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "Usunięty świat testu (Smocza Góra)",
    "Limity: 70 przykładów na postać, 12 na kartę; 3 nowe światy, 24 postacie",
    "Poprawiona tokenizacja „▁[”; 2832 przykłady, 1 epoka, 204 min"
   ],
   "result": "eval_loss 0,863. Automat 88/91% (jak run14h). Ślepo: MG 89% rund bez ciężkich (run14h 95%), rozmowy bez ciężkich 42% (run14h 71%).",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Uczciwy poziom na nieznanych postaciach: walka się uogólnia, rozmowy słabiej.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 15 (20–21.10)"
   ]
  },
  {
   "id": "run16",
   "date": "2026-10-21",
   "title": "SFT po poprawkach czytania gracza w silniku",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "Silnik: wiele poprawek czytania gracza (fałszywe nazwy, ceny, pożegnania, prośby spoza gry)",
    "Dane odtworzone na nowym silniku i przejrzane w 7 rundach",
    "2 nowe światy, paczki na słabości run15; 3395 przykładów, 1 epoka, 4 h 25 min"
   ],
   "result": "Automat 91/93% (najlepszy dotąd), powtórki 7/4. Ślepo na starym teście: remis z run15, run14h dalej lepszy. Na nowym teście test16 w nowym świecie run16 19 ciężkich, run14h 27.",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Stary test okazał się przestarzały (zamrożone stare polecenia gry).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 16 (21.10)"
   ]
  },
  {
   "id": "run16dh",
   "date": "2026-10-21",
   "title": "Małe DPO z własnych odpowiedzi (run16d, run16dh)",
   "base": "Bielik 11B v2.3 (run16)",
   "method": "SFT+DPO",
   "changes": [
    "Pary z własnych odpowiedzi run16 o podobnej długości: tylko 135",
    "1 epoka, lr 3e-5, beta 0,1, 17 kroków; run16dh = checkpoint-15",
    "Nowy test test16 (nowy świat Zgasłe Szkiery)"
   ],
   "result": "Strata 0,693 → 0,678 (prawie bez ruchu). Ślepo: MG run16dh 7 ciężkich (run16 13), rozmowy ciężkie 26 (run16 32). W kolejnych ocenach (22.10) nic go nie pobiło.",
   "verdict": "niejasne",
   "in_game": true,
   "note": "Model gry ok. 22–23.10 („gra zostaje na run16dh”), zastąpiony przez run18 23.10. Analiza z 22.10: efekt może być szumem.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 16d / run16dh (21.10)",
    "D:/dev/neuro-processor/tools/model/ANALIZA-2026-10-22.md#B8"
   ]
  },
  {
   "id": "b45gm",
   "date": "2026-10-22",
   "title": "Bielik 4.5B uczony tylko opisów walk",
   "base": "Bielik 4.5B v3.0",
   "method": "SFT",
   "changes": [
    "LoRA tylko na 761 rundach MG z run17",
    "2 epoki, lr 2e-4, r16, ok. 0,5 h"
   ],
   "result": "eval_loss 1,068 → 0,874. Ślepo (80 rund): ciężkie b45gm 13, run16dh 7, run14h 6; lekkich 93 wobec 35–39 (przekręcone nazwy, odmiana).",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Fakty niewiele gorzej od 11B, język wyraźnie gorzej.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#Małe Bieliki jako Mind i MG (22.10)"
   ]
  },
  {
   "id": "run17",
   "date": "2026-10-22",
   "title": "SFT na danych przejrzanych pod silnik b0b79a3",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "Dane odtworzone i przejrzane w 9 częściach",
    "Paczki k35–k38 (krewni, towarzysze, najemnicy, kupcy), 227 nowych rund MG",
    "3846 przykładów, 1 epoka, lr 1e-4, ok. 5 h"
   ],
   "result": "eval_loss 0,794. Ślepo vs run16dh: rozmowy 28 vs 34 ciężkie (remis w teście parowym), MG 16 vs 6 ciężkich (gorzej). Druga ocena: MG 17 vs 8.",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Słaby MG okazał się później pechem ziarna (run17s).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run 17 (22.10)"
   ]
  },
  {
   "id": "run17dh",
   "date": "2026-10-22",
   "title": "DPO na run17 i ocena siedmiu modeli",
   "base": "Bielik 11B v2.3 (run17)",
   "method": "SFT+DPO",
   "changes": [
    "239 par z samoodtworzenia (45 MG, reguła „pominięta liczba”)",
    "lr 3e-5, beta 0,1; run17dh = połowa",
    "Wspólna ślepa ocena 7 modeli z kotwicą run16dh i testem znaków"
   ],
   "result": "MG ciężkie na 160: run17 29 → run17d 21 → run17dh 17, run16dh ok. 10–12. Rozmowy wszystkie w szumie (ocena 3,15–3,31).",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Nic nie pobiło run16dh.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run17d / run17dh"
   ]
  },
  {
   "id": "run18",
   "date": "2026-10-23",
   "title": "Różnorodność zamiast objętości",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "8 nowych światów i 400 krótkich rozmów; limit 40 kwestii na postać (było 90)",
    "Wycięte najsłabsze dane GPT, odrzut kwestii recytujących kartę",
    "Silnik po ok. 108 poprawkach odczytu; 4005 przykładów (807 rund MG)",
    "2 epoki, lr 1e-4, r16, ok. 10 h"
   ],
   "result": "Nowy test test18, 2 rundy po 12 sędziów: MG ciężkie 25/26 (run16dh 53/47), rozmowy 28/35 (run16dh 69/72), ocena 3,72 (3,20). Pary z run16dh: MG 84/42, rozmowy 133/41, p<0,001.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Model gry od 23.10 do 03.11. Wariant run18h przez błąd skryptu = ten sam model.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run18 (23.10)"
   ]
  },
  {
   "id": "run16dhq4",
   "date": "2026-10-24",
   "title": "run16dh w mocniejszej kompresji Q4_K_M",
   "base": "Bielik 11B v2.3 (run16dh)",
   "method": "inne",
   "changes": [
    "Ten sam model zapisany w 4 bitach (Q4_K_M) zamiast Q5_K_M"
   ],
   "result": "6,7 GB zamiast 7,9 GB, 1,9 s zamiast 2,1 s. MG bez straty (pary 56/61, p=0,71), rozmowy nieco gorsze (76/103, p=0,05; ciężkie 162 vs 143).",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Q4 nadaje się co najwyżej na osobny model MG.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run19 (24.10)"
   ]
  },
  {
   "id": "run17s",
   "date": "2026-10-24",
   "title": "Pomiar szumu: run17 z innym ziarnem",
   "base": "Bielik 11B v2.3",
   "method": "SFT",
   "changes": [
    "Identyczne dane i przepis co run17, tylko ziarno losowe 18"
   ],
   "result": "MG ciężkie na 160: run17s 12 (6/6), run17 30 (16/14) — istotnie (pary 25/10, p=0,017). Rozmowy bez różnicy (53 vs 54).",
   "verdict": "pomoglo",
   "in_game": false,
   "note": "Eksperyment pomiarowy: pokazał, że samo ziarno zmienia błędy MG 2,5×.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run17s (24.10)"
   ]
  },
  {
   "id": "run19",
   "date": "2026-10-24",
   "title": "Nowsza baza Bielik 11B v3.0 (run19, run19h)",
   "base": "Bielik 11B v3.0",
   "method": "SFT",
   "changes": [
    "Te same dane i przepis co run18, tylko inna baza",
    "run19h = checkpoint po 1. epoce"
   ],
   "result": "eval_loss o 0,02–0,04 niżej niż run18. Ślepo: remis z run18 (MG pary 45/37 p=0,44; rozmowy 74/73 p=1,0). run19h w rozmowach gorszy (60/104, p=0,001).",
   "verdict": "niejasne",
   "in_game": false,
   "note": "Baza v3 nic istotnego nie dała, ale nie została porzucona (decyzja w run20).",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#run19 (24.10)",
    "D:/dev/neuro-processor/tools/model/RECENZJA-2026-10-24.md#0"
   ]
  },
  {
   "id": "m7mind",
   "date": "2026-10-25",
   "title": "Umysł zamiast gemmy: b45mind i m7mind",
   "base": "Bielik 4.5B v3.0 / Minitron 7B v3.0",
   "method": "SFT",
   "changes": [
    "LoRA na zadaniach Umysłu (Note, Invent, Interpret, SelfNote, QuestHook): 1255 / 192 przykładów",
    "2 epoki, lr 2e-4, r16, ok. 1 h każdy",
    "Później m7mindq4 = m7mind w Q4_K_M (4,5 GB)"
   ],
   "result": "m7mind: zmyślone fakty w SelfNote 0% (gemma 52%), fałszywe alarmy QuestHook 0/15 (gemma 15/15), Interpret akcja 95% (gemma 40%). b45mind dobry, ale nigdy nie wymyśla towaru (Invent 0/6). m7mindq4 = jakość m7mind.",
   "verdict": "pomoglo",
   "in_game": false,
   "note": "Nie mieścił się w karcie obok run18 Q5 (Ollama wyrzucała model rozmów). Stąd pomysł run21.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#25.10 wieczór",
    "D:/dev/neuro-processor/tools/model/MODELE.md#27.10"
   ]
  },
  {
   "id": "run20v3",
   "date": "2026-10-27",
   "title": "Biografie postaci i zlecenia; v2.3 kontra v3 (run20v2, run20v3)",
   "base": "Bielik 11B v2.3 i v3.0",
   "method": "SFT",
   "changes": [
    "Dane pod nowe funkcje gry: biografie towarzyszy i kupców, zlecenia ratunkowe, przekleństwa",
    "Wszystko przegrane na silniku bb9e0c1; 4429 / 208 przykładów",
    "Dwa treningi (v2.3 i v3), ten sam przepis, ziarno 7; reguła decyzji ustalona z góry"
   ],
   "result": "Rozmowy lepsze od run18 na obu testach (test18 ciężkie 42 → 26–27; test20 14 → 9–11). MG bez zmian. v3 vs v2: remis (kierunki przeciwne, p=0,34 i 0,56) → wybrany v3. Słabość: pary rozmów 6–7 ciężkich na 16 (run18: 2).",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "run20v3 w grze od 03.11 (z gemmą jako Umysłem), dzień później zastąpiony przez run21.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#27.10: run20v2 vs run20v3",
    "D:/neuro-train/run20/DECYZJA.md",
    "D:/neuro-train/run21/PLAN.md#03.11"
   ]
  },
  {
   "id": "run21",
   "date": "2026-11-04",
   "title": "Jeden model do wszystkiego: rozmowy, MG i Umysł",
   "base": "Bielik 11B v3.0",
   "method": "SFT",
   "changes": [
    "Dane run20 (z poprawkami k61–k69, +120 par rozmów pod biografię) + 1536 przykładów Umysłu (24%)",
    "6324 / 475 przykładów, 2 epoki, lr 1e-4, r16, 13 h 55 min",
    "Gemma przestaje być potrzebna"
   ],
   "result": "Ślepo vs run20v3: nigdzie istotnie gorszy (test18 MG 15 vs 20, rozmowy 46 vs 44; test20 MG 4 vs 6, rozmowy 8 vs 10). Umysł ≈ m7mind (Interpret 95% wobec 40% gemmy). Pamięć 8,7 GB zamiast 12,4 GB.",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Model gry od 04.11 (później zastąpiony przez run23). Cel nieosiągnięty: spójność biografii — 4 sprzeczności na 16 par.",
   "sources": [
    "D:/dev/neuro-processor/tools/model/MODELE.md#04.11: run21",
    "D:/neuro-train/run21/eval/DECYZJA.md"
   ]
  },
  {
   "id": "run22",
   "date": "2026-11-05",
   "title": "Polski NPC także dla innych gier i modów",
   "base": "Bielik-11B-v3.0-Instruct",
   "method": "SFT",
   "changes": [
    "19% danych w obcych formatach (1705 przykładów): narrator, JSON z progami, limity długości, karty postaci, próby wyłamania z roli, mody z głosem, handlarze",
    "209 nowych przykładów Umysłu na krótkie i przeczące odpowiedzi („Nie żyje, dawno”, „kawaler z wyboru”)",
    "323 nowe kwestie rozmów i 269 nowych rund MG celowane w słabości run21; wyrzucone wzorce z kwotami pod zakazem, „znów” w MG i powtarzanymi frazami",
    "Przepis bez zmian (2 epoki, lr 1e-4, r16, max-len 4096, ziarno 7); zbiór 8550 / 595, 2138 kroków, 16 h 52 min"
   ],
   "result": "W grze Neuro remis z run21 (np. MG t18: 7 vs 11 ciężkich, rozmowy t18: 58 vs 50, różnice nieistotne). Wielka poprawa limitów długości (1 vs 12 ciężkich, p=0,001) i pamięci między rozmowami (sprzeczne pary 0 vs 3), ale gorszy narrator (15 vs 8) i więcej fałszywych notatek Umysłu (sq20: 7/11 vs 4/11).",
   "verdict": "mieszane",
   "in_game": false,
   "note": "Reguła z góry nie spełniona w 3 punktach (B1, B2, M2), więc gra została na run21. Wcześniej, 04.11, ta sama seria prac poprawiła silnik (lepki temat, prompt Umysłu v2, strażnik sprzeczności).",
   "sources": [
    "D:\\neuro-train\\run22\\PLAN22.md#1a. Liczby po build22.py",
    "D:\\neuro-train\\run22\\eval\\DECYZJA22.md",
    "D:\\neuro-train\\run22\\train22.log",
    "D:\\neuro-train\\run23\\PLAN23.md#Diagnoza"
   ]
  },
  {
   "id": "run23s",
   "date": "2026-11-06",
   "title": "SFT run23: nowy narrator, kupiec ze skryptu, negatywy Umysłu",
   "base": "Bielik-11B-v3.0-Instruct",
   "method": "SFT",
   "changes": [
    "Narrator napisany od nowa (338 przykładów): osoba narracji brana z promptu, 1–3 zdania, jawne wyniki („ucieka” to nie „ginie”), automatyczny sprawdzacz zmyślonych bytów",
    "Kupiec liczony skryptem (97 tur), 53 trudne przypadki JSON, 158 kwestii Neuro na zakazy, 89 negatywów Umysłu „fakt już jest w karcie”",
    "Checkpointy co 0,5 epoki i wybór na osobnym zestawie dev (nie testowym)",
    "Zbiór 8975 / 572 (rozmowy Neuro 50,1%), 2244 kroki, ok. 17,5 h"
   ],
   "result": "Na dev: 1,0 ep. D=19,6, 1,5 ep. 17,3, 2,0 ep. 17,9 (niżej = lepiej). Różnica 0,6 pkt była mniejsza niż wymagany 1,0 pkt, więc wzięto 2,0 ep. Ten model sam nigdy nie przeszedł ślepej oceny.",
   "verdict": "niejasne",
   "in_game": false,
   "note": "Baza, od której zrobiono DPO run23 „b”. Brak ślepej oceny samego SFT utrudnił później ocenę run24.",
   "sources": [
    "D:\\neuro-train\\run23\\DANE23.md",
    "D:\\neuro-train\\run23\\stats23.json",
    "D:\\neuro-train\\run23\\eval\\dev\\out\\dev-score-sft.md",
    "D:\\neuro-train\\run23\\train23.log"
   ]
  },
  {
   "id": "run23",
   "date": "2026-11-08",
   "title": "run23 „b”: SFT + DPO na parach wzorzec kontra błąd",
   "base": "Bielik-11B-v3.0-Instruct (od run23s, 2,0 ep.)",
   "method": "SFT+DPO",
   "changes": [
    "DPO na ok. 670 parach: 390 par z własnych odpowiedzi modelu (przejrzane przez sędziów i użytkownika) + ok. 280 par „wzorcowa odpowiedź kontra zła odpowiedź modelu”",
    "lr 5e-5, 1 epoka; spośród 7 wariantów (d, dh, e, eh, b, c, k) tylko „b” poprawił dev ponad szum",
    "Ocena na silniku z nowymi osłonami (d33d36b), dwóch sędziów (Opus + Sonnet) na każdą pozycję"
   ],
   "result": "Ślepo vs run21: rozmowy 92 vs 148 ciężkich błędów na 420 (p<0,0001), obce formaty 61 vs 141 na 320 (p<0,0001), narrator 11 vs 75, MG 26 vs 40 (nieistotne). EQ-Bench 76,4 (baza 75,9), MT-Bench PL 5,98 (baza 5,96), PingPong-PL 4,05 (baza 3,34).",
   "verdict": "pomoglo",
   "in_game": true,
   "note": "Wg reguły tylko remis (niespełnione M1, M4 i B3 – sklep), ale użytkownik 09.11 świadomie przepiął grę na run23. 11.11 opublikowany na Hugging Face: rozmus/neuro-npc-bielik-11b-v3-GGUF.",
   "sources": [
    "D:\\neuro-train\\run23\\eval\\DECYZJA23.md",
    "D:\\neuro-train\\run23\\eval\\dev\\out\\dev-score.md",
    "D:\\neuro-train\\run23\\WNIOSKI-DPO23.md",
    "C:\\Users\\prow13r\\.claude\\projects\\D--dev-neuro-processor\\memory\\run24-gpt-orders.md#11.11 HF"
   ]
  },
  {
   "id": "run24",
   "date": "2026-11-10",
   "title": "run24: dane run23 poprawione przez GPT/Codex",
   "base": "Bielik-11B-v3.0-Instruct",
   "method": "SFT",
   "changes": [
    "Do danych run23 dołożono poprawki GPT/Codex: Z5 (499 poprawionych + 608 „dobrych” kwestii run23), Z6 (150 zmienionych starych wierszy, 28 usuniętych), Z3 (192 notatki Umysłu), Z8 (220 odczytów intencji)",
    "Zbiór 10 466 / 678; 129 najdłuższych rozmów odpadło po cichu przez limit 4096 tokenów",
    "Wybrany checkpoint 1,5 epoki (dev 14,1 vs 18,8 po 2,0 ep.); DPO odrzucone na dev",
    "Trening przerwany dwa razy (restart WSL, restart Windows), wznowiony z checkpointu"
   ],
   "result": "Ślepo vs run23: MG istotnie gorzej (49,5 vs 25 ciężkich na 669 rund, p=0,0001), rozmowy remis (136,5 vs 110,5, p=0,61), obce formaty i G23 remis (G23 57 vs 65). Umysł podobny lub lepszy (surowe fałszywe notatki 5/90 vs 8/90).",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Analiza 13.11: główna przyczyna to porównanie samego SFT z modelem po SFT+DPO. Profil błędów MG run24 = run22 (też samo SFT). Realne szkody poprawek: 11 wierszy MG bez liczb, powtórki z Z5, 129 utraconych wierszy.",
   "sources": [
    "D:\\neuro-train\\run24\\DECYZJA24.md",
    "D:\\neuro-train\\run24\\DECYZJA24.md#Analiza przyczyn (13.11.2026)",
    "D:\\neuro-train\\run24\\dev-score.md#SFT run24s",
    "D:\\neuro-train\\run24\\analiza\\opus\\RAPORT.md",
    "D:\\neuro-train\\run24\\analiza\\fable\\RAPORT.md"
   ]
  },
  {
   "id": "run24d",
   "date": "2026-11-12",
   "title": "DPO na run24 bez par MG",
   "base": "run24s, checkpoint 1,5 ep.",
   "method": "DPO",
   "changes": [
    "495 par treningowych / 95 walidacyjnych: rozmowy i 55 par Umysłu, 0 par MG, 0 obcych formatów",
    "lr 5e-5, beta 0,1, nowy dodatek rpo_alpha 0,2 (kara SFT na dobrej odpowiedzi), 1 epoka (62 kroki)",
    "Złe odpowiedzi w parach pochodziły z run23, a nie z trenowanego modelu"
   ],
   "result": "Trening wyglądał świetnie (margines 0,72, trafność 92%), ale na dev ΔD +0,7 [−3,2; +4,5], kwestie krótsze o 17%, [spokój] +10 pkt, fałszywe notatki Umysłu 0/15 → 4/15. MG się poprawiło (27,7% → 19,1%).",
   "verdict": "nie_pomoglo",
   "in_game": false,
   "note": "Prawdopodobieństwo dobrej odpowiedzi spadło o ok. 9 natów (−27,6 → −36,5), w „b” tylko o 1,6. Silny margines nie oznacza lepszego modelu.",
   "sources": [
    "D:\\neuro-train\\run24\\dev-score.md#DPO run24d",
    "D:\\neuro-train\\run24\\analiza\\fable\\RAPORT.md#B. Hipotezy"
   ]
  },
  {
   "id": "run24v31",
   "date": "2026-11-12",
   "title": "Te same dane run24 na nowej bazie Bielik 11B v3.1",
   "base": "Bielik-11B-v3.1-Instruct",
   "method": "SFT",
   "changes": [
    "Zmieniona tylko baza (decyzja: jedna zmienna naraz); dane i przepis jak run24",
    "v3.1 ma inny tokenizer i config niż v3.0; trening używa własnego szablonu rozmowy, nie szablonu v3.1 z długim promptem dostawcy i <think>",
    "Plan: checkpointy 1,5 i 2,0 ep. → dev, porównanie z run24 SFT (nie z run23)"
   ],
   "result": "W toku: 13.11 o 12:21 krok 1905 z 2586 (74%). eval_loss po 1,0 ep. 0,519 wobec 0,503 dla run24 na tym samym zbiorze walidacyjnym (po 0,25 ep. 0,580 vs 0,566). Brak wyników dev i ślepej oceny.",
   "verdict": "w_toku",
   "in_game": false,
   "note": "Nieco wyższy eval_loss to tylko wstępny sygnał, nie wynik. Bazę pobrano 12.11 (GGUF 7,9 GB + safetensors).",
   "sources": [
    "D:\\neuro-train\\run24\\train24v31.sh",
    "D:\\neuro-train\\run24\\train24v31.log",
    "D:\\neuro-train\\run24\\get_v31.log",
    "D:\\neuro-train\\run24\\train24.log"
   ]
  },
  {
   "id": "run25",
   "date": "2026-11-13",
   "title": "Plan: naprawa MG danymi gm25 na zwycięskiej bazie",
   "base": "v3.0 albo v3.1 (zależnie od wyniku run24v31)",
   "method": "SFT+DPO",
   "changes": [
    "579 nowych rund MG (gm25 v2.1) pisanych przez Codex: regeneracja, drużyny 3+, bliźniacze potwory z liczebnikami („drugi Topielec”)",
    "Propozycja: wymienić 185 najsłabszych starych wierszy MG; udział MG 12,7% wierszy (run24: 9,6%)",
    "Lekcje z run24: DPO na stałe w przepisie, poprawki MG przez walidator, bramki jakości dla nowych wierszy, przycinanie długich rozmów"
   ],
   "result": "Brak – jeszcze nie trenowany. Dane gm25 przeszły bramki (validate_gm22: 0 błędów, kontrola przecieku OK), w losowej próbce 60 wierszy 0 ciężkich i 3 lekkie błędy.",
   "verdict": "w_toku",
   "in_game": false,
   "note": "Ślepa ocena ma być tylko vs run23.",
   "sources": [
    "D:\\neuro-train\\run25\\BRIEF-GM25.md",
    "D:\\neuro-train\\run25\\BRIEF-GM25B.md",
    "D:\\neuro-train\\run25\\gm\\DANE-GM25.md#Propozycja dla run25",
    "D:\\neuro-train\\run24\\DECYZJA24.md#Analiza przyczyn (13.11.2026)"
   ]
  }
 ],
 "experiments": [
  {
   "topic": "Większy model bazowy (4,5B → 11B)",
   "category": "baza",
   "runs": [
    "run6",
    "run7"
   ],
   "what": "Ten sam zbiór danych, ale baza Bielik 11B zamiast 4,5B, trening na PC z kartą graficzną.",
   "effect": "Bench 76/78 → 80/80. Imiona bez przekręceń. W długich rozmowach 92% (7B na tych samych danych 84%). Koszt: 3× wolniej na Macu, 7,9 GB zamiast 5,1 GB.",
   "verdict": "pomoglo",
   "why": "Fakt: rozmiar bazy to sufit języka; 4,5B mylił odmianę imion niezależnie od danych.",
   "worth": "tak",
   "worth_note": "Brać największą bazę, jaką mieści sprzęt."
  },
  {
   "topic": "Mniejszy, szybszy model Minitron 7B",
   "category": "baza",
   "runs": [
    "run8",
    "run9b",
    "run10d"
   ],
   "what": "Przycięty Bielik 7B, żeby zmieścić Głos w karcie razem z gemmą i syntezą mowy.",
   "effect": "Szybszy (1,2 s wobec 1,6–1,8 s) i mniejszy (5,3 GB), bench w szumie, ale w długich rozmowach 84–86% wobec 92% run 7 i więcej liczb słownie.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: na tych samych danych słabszy. Domysł: „klimat” run 7 pochodzi z bazy 11B, nie z danych.",
   "worth": "nie",
   "worth_note": "Jako Głos nie; jako Umysł po treningu tak (m7mind)."
  },
  {
   "topic": "Nowsza baza Bielik 11B v3.0",
   "category": "baza",
   "runs": [
    "run19",
    "run20v3"
   ],
   "what": "Te same dane i przepis na v3.0 zamiast v2.3.",
   "effect": "Strata niższa o 0,02–0,04 w każdym punkcie. Ślepo: remis (run19 vs run18 p=0,44 / 1,0; run20v3 vs v2 p=0,34 / 0,56). Inny profil błędów: v3 mniej bełkotu, więcej powtórek.",
   "verdict": "niejasne",
   "why": "Fakt: niższa strata nie przełożyła się na oceny. Domysł: przepis dobrany pod v2.3.",
   "worth": "warunkowo",
   "worth_note": "Wybrano v3 przy remisie (narzędzia, długi kontekst)."
  },
  {
   "topic": "Nowa baza Bielik 11B v3.1",
   "category": "baza",
   "runs": [
    "run24v31"
   ],
   "what": "Ten sam zbiór i przepis co run24, zmieniona tylko baza na v3.1 (wydana 10.11, trenowana dodatkowo RL).",
   "effect": "W toku. eval_loss na tym samym zbiorze walidacyjnym nieco wyższy niż na v3.0: 0,580 vs 0,566 (0,25 ep.), 0,519 vs 0,503 (1,0 ep.).",
   "verdict": "w_toku",
   "why": "Fakt: inny tokenizer i config niż v3.0. Trening nie używa szablonu v3.1 (z długim promptem dostawcy i <think>). Wyższy eval_loss to tylko wstępny sygnał; decyduje dev i porównanie z run24 SFT.",
   "worth": "warunkowo",
   "worth_note": "Porównywać z run24 SFT, nie z run23; gorsze MG od run23 nie będzie dowodem przeciw v3.1."
  },
  {
   "topic": "Zamówienie danych u silnego modelu z kontraktem i walidatorem",
   "category": "dane",
   "runs": [
    "run3",
    "run4",
    "run5",
    "run6"
   ],
   "what": "Spisaliśmy prompt gry co do znaku i zamówiliśmy u GPT ok. 1700 rozmów. Każdy wiersz sprawdzał automat.",
   "effect": "Test ppl 4,59 → 3,85, pierwszy grywalny model, bench 71/80, ceny 6/6.",
   "verdict": "pomoglo",
   "why": "Fakt: dane były zgodne z formatem gry i różnorodne. To jeden z dwóch największych skoków w projekcie.",
   "worth": "tak",
   "worth_note": "Zawsze: najpierw kontrakt i test, potem dane, z walidatorem w pętli."
  },
  {
   "topic": "Dane celowane w luki widoczne w logach gry",
   "category": "dane",
   "runs": [
    "run4",
    "run5",
    "run6"
   ],
   "what": "Czytaliśmy logi gry i zamawialiśmy dokładnie to, czego brakowało: powitania, zaczepki, potwierdzenia najmu, tropy.",
   "effect": "Bench 71 → 78/79. Odtworzenia: powitanie 0/3 → 3/3, kierunek zapłaty ok. połowa → 12/12, trop 3/3.",
   "verdict": "pomoglo",
   "why": "Fakt: każda paczka naprawiała konkretny, zmierzony błąd.",
   "worth": "tak",
   "worth_note": "Tak, jeśli luka jest widoczna w logach i da się ją zmierzyć."
  },
  {
   "topic": "Więcej światów i postaci zamiast więcej przykładów na postać",
   "category": "dane",
   "runs": [
    "run15",
    "run16",
    "run18",
    "run20v3"
   ],
   "what": "Nowe światy, limit przykładów na postać (90 → 40), krótkie rozmowy, wycięte słabe dane GPT.",
   "effect": "Nowy świat: run14h 27 → run16 19 ciężkich. run18 vs run16dh: rozmowy 69/72 → 28/35 ciężkich, MG 53/47 → 25/26 (p<0,001).",
   "verdict": "pomoglo",
   "why": "Fakt: model uczy się „jak być kupcem w nieznanym świecie”, a nie „jak brzmi Thorin”.",
   "worth": "tak",
   "worth_note": "Różnorodność ważniejsza niż objętość."
  },
  {
   "topic": "Kto pisał lepsze dane: GPT czy Claude",
   "category": "dane",
   "runs": [
    "run18"
   ],
   "what": "Analiza danych 22.10 porównała paczki od GPT i od Claude.",
   "effect": "Dane Claude lepsze w każdej mierze (ślepa ocena 360 kwestii 3,53 vs 2,39). Przewaga GPT to tylko rozkład: 2641 sytuacji w 67 światach.",
   "verdict": "pomoglo",
   "why": "Fakt: liczy się różnorodność sytuacji, nie autor.",
   "worth": "tak",
   "worth_note": "Zamawiać szeroko (wiele światów), sprawdzać jakość niezależnie od autora."
  },
  {
   "topic": "Narrator napisany od nowa z walidatorem",
   "category": "dane",
   "runs": [
    "run23"
   ],
   "what": "Nowy narrator: osoba narracji z promptu („ty” albo imię), 1–3 zdania, jawne wyniki rzutu, skrypt odrzuca zmyślone byty spoza sceny.",
   "effect": "Narrator w ślepej ocenie: 11,3 ciężkich wobec 74,8 (run21) i 44,0 (run22). Zła osoba narracji tylko w 4 z 88 odpowiedzi (4,5%).",
   "verdict": "pomoglo",
   "why": "Fakt: próbka 10 promptów na run21/run22 pokazała dokładnie te błędy, które potem celowano. Zastrzeżenie: model run23 miał też DPO z wieloma parami obcych formatów, więc nie cały zysk to dane SFT.",
   "worth": "tak",
   "worth_note": "Wzór do kopiowania: najpierw próbka błędów, potem dane i automatyczny sprawdzacz."
  },
  {
   "topic": "Ożywianie suchych kwestii i douczanie gotowego modelu",
   "category": "dane",
   "runs": [
    "run11"
   ],
   "what": "Claude przepisał 445 suchych kwestii, run 7 douczony 1 epoką.",
   "effect": "Więcej emocji (32–39% znaczników innych niż spokój wobec 23%), z gry 96/89% czystych (run 7 82%), ale kwestie o ok. 20% dłuższe i więcej powtórek (10–14 vs 6).",
   "verdict": "mieszane",
   "why": "Fakt: żywsze dane dały żywszy model, ale też gadatliwszy.",
   "worth": "warunkowo",
   "worth_note": "Pilnować długości odpowiedzi w nowych danych."
  },
  {
   "topic": "Biografie postaci i pamięć między rozmowami",
   "category": "dane",
   "runs": [
    "run20v3",
    "run21"
   ],
   "what": "Gra zapisuje, co postać powiedziała o sobie; dane uczą trzymać się tego w kolejnej rozmowie.",
   "effect": "Rozmowy ogólnie lepsze, ale pary rozmów: run20 6–7 ciężkich na 16 (run18 2), run21 4 sprzeczności na 16 („mąż nie żyje” → „żadnego męża nie było”).",
   "verdict": "mieszane",
   "why": "Domysł: dane uczyły dopowiadania kilku faktów naraz w pierwszej rozmowie.",
   "worth": "warunkowo",
   "worth_note": "Jeden nowy fakt na kwestię i więcej pytań sprawdzających; cel wciąż otwarty."
  },
  {
   "topic": "Dane w obcych formatach (polski NPC ogólnie)",
   "category": "dane",
   "runs": [
    "run22",
    "run23"
   ],
   "what": "Dołożono 1705 przykładów spoza gry Neuro: narrator RPG, JSON, limity długości, karty postaci, jailbreaki, mody, sklepy.",
   "effect": "run22: limity długości 1 vs 12 ciężkich (p=0,001), lepszy JSON, ale narrator gorszy (15 vs 8) i łącznie obce formaty tylko 31 vs 36 (p=0,37). Po przepisaniu narratora w run23: obce formaty 61 vs 141 vs run21 (p<0,0001).",
   "verdict": "mieszane",
   "why": "Fakt: w run22 narrator miał wejścia prawie zawsze w 1. osobie i zawsze 3 zdania, więc model uczył się złych nawyków. Fakt: w Neuro nic się nie pogorszyło (wtręty Neuro w obcych formatach 0).",
   "worth": "tak",
   "worth_note": "Warto, ale każdy format musi mieć własny walidator i dobre dane, inaczej uczy błędów."
  },
  {
   "topic": "Uczenie na wyjściach słabszego modelu",
   "category": "dane",
   "runs": [
    "run1"
   ],
   "what": "Do treningu wrzuciliśmy surowe logi rozmów gemmy z gry (44% zbioru).",
   "effect": "Model przejął drewniany styl gemmy. Test ppl 5,9 — najgorszy ze wszystkich runów.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: model uczy się naśladować wzorzec. Logi gemmy były tym, od czego chcieliśmy odejść.",
   "worth": "nie",
   "worth_note": "Nie uczyć na własnych lub cudzych wyjściach bez ręcznej selekcji."
  },
  {
   "topic": "Dane szablonowe i za mały zbiór",
   "category": "dane",
   "runs": [
    "run2",
    "run3"
   ],
   "what": "Najpierw dane z gry „sam ze sobą” (synteza) i 355 ręcznie poprawionych wierszy; potem odrzucenie 660 szablonowych „sklepowych” wierszy od GPT.",
   "effect": "run2: ceny dobre (3/3), ale kontroferty nie umiał — miał jej ok. 10 przykładów. Odmiana się sypała.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: model nie umie tego, czego prawie nie widział. Szablony uczą szablonu, nie języka.",
   "worth": "nie",
   "worth_note": "Przed treningiem policzyć przykłady na każdy rodzaj sytuacji."
  },
  {
   "topic": "Pilot stylów mowy",
   "category": "dane",
   "runs": [
    "style1"
   ],
   "what": "Jedna linia „STYL MOWY” w karcie postaci i 356–716 przykładów stylu (staropolski, wojskowy, krasnoludzki).",
   "effect": "Słowa stylu: 0–25% bez treningu → style1 28–54% → style2 37–57%. Bramka ≥80% niezaliczona dwa razy.",
   "verdict": "nie_pomoglo",
   "why": "Domysł: model traktuje linię stylu jako słabą wskazówkę obok mocnego wzorca run 7.",
   "worth": "warunkowo",
   "worth_note": "Ewentualnie osobny adapter od bazy albo przykładowe zwroty w karcie zamiast opisu."
  },
  {
   "topic": "Odtwarzanie i przeglądanie danych pod każdy nowy silnik",
   "category": "dane",
   "runs": [
    "run15",
    "run16",
    "run17"
   ],
   "what": "Po każdej zmianie gry wszystkie rozmowy treningowe przegrane i przejrzane (run16: 7 rund, run17: 9 rund).",
   "effect": "Rozmowy w ślepej ocenie: 42% → 46% → 52/46% bez ciężkich przy szumie ±8 pkt; MG 89 → 85 → 84%.",
   "verdict": "nie_pomoglo",
   "why": "Domysł z analizy 22.10: pozostałe błędy (bełkot, zmyślenia) to nie brak przykładów, a sędziowie karzą co innego niż to, co poprawiano.",
   "worth": "nie",
   "worth_note": "Odtwarzać tylko kwestie z poleceniem, którego w zbiorze w ogóle nie ma."
  },
  {
   "topic": "Kupiec, który liczy sumy na głos (shop23)",
   "category": "dane",
   "runs": [
    "run23"
   ],
   "what": "Liczby (sumy, rabaty, reszty) liczył skrypt, a autor pisał tylko tekst. 97 tur kupca.",
   "effect": "Automat sklepu w teście B v2: 24 błędy wobec 18 u run21 – warunek B3 niespełniony. Liczby spoza promptu 11 vs 6.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: dane były poprawne, ale uczyły liczyć na głos. Domysł z obu planów run24: model 11B liczy źle, więc lepiej, by w ogóle nie podawał liczb pochodnych. Część różnicy to szum (mała próbka).",
   "worth": "warunkowo",
   "worth_note": "Planowany kupiec v3 bez liczenia na głos nie powstał."
  },
  {
   "topic": "158 kwestii celowanych w zakazy i długie rozmowy",
   "category": "dane",
   "runs": [
    "run23"
   ],
   "what": "Nowe kwestie Neuro: pożegnanie z kwotą, zakaz pytań, najemnik bez ceny, dopytywanie o rodzinę, rozmowy bez powtórek.",
   "effect": "W teście wielotur t23 run23 = run22 (49 lepszych / 44 gorszych, p=0,68).",
   "verdict": "nie_pomoglo",
   "why": "Fakt: rozmowy Neuro stały od run20, a takich błędów prawie nie było w danych. Domysł: 158 przykładów to za mało; to, co poprawiło rozmowy run23, to głównie DPO „b”.",
   "worth": "warunkowo",
   "worth_note": "Tylko w dużo większej skali albo jako pary DPO."
  },
  {
   "topic": "Z5: poprawki odpowiedzi run23 przez Codex (złote dane i pary)",
   "category": "dane",
   "runs": [
    "run24"
   ],
   "what": "GPT napisał 6 światów i 150 długich skryptów gracza. run23 je zagrał, a Codex ocenił 1762 tury i poprawiał błędy, zmieniając najwyżej 40% słów. Do SFT weszło 499 poprawek i 608 tur uznanych za dobre.",
   "effect": "Rozmowy w ślepej ocenie bez poprawy: 153 lepsze / 163 gorsze (48,4%, p=0,61). Ciężkie powtórki u sędziów 22 → 33 (wg innej analizy 29 → 42,5). W głębokich turach (7+) lekko lepiej: 31 vs 40 błędów automatu na 291.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: nowego sygnału było ok. 10% wierszy. Fakt: wiersze Z5 nie przeszły bramek jakości (powtórki 4,3–4,8% wobec 0,1%). Fakt: 39% poprawek dotyczyło zmyśleń, a sędziowie liczą głównie kwoty, pytania pod zakazem, sprzeczność z [GRA] i powtórki. Fakt: poprawki często były unikami (20,7% vs 3% w starych danych).",
   "worth": "warunkowo",
   "worth_note": "Tylko z bramkami build23, celowane w błędy liczone przez sędziów i z testem wielotur t24."
  },
  {
   "topic": "Z6: audyt starych danych przez GPT",
   "category": "dane",
   "runs": [
    "run24"
   ],
   "what": "GPT sprawdził 560 starych wierszy i poprawił błędne. Przyjęto tylko poprawki do 40% zmian: 150 wierszy zmienionych, 28 usuniętych.",
   "effect": "W 11 wierszach MG zniknęły wszystkie liczby obrażeń (25 liczb) – w run21–23 takich wierszy było 0. W teście run24 pomija liczby częściej o 1,2 pkt proc. Poprawki usunęły też 6 zmyślonych liczb.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: Z6 dostało wstęp pisany pod rozmowy („mechanika nie pada w ustach postaci”), a build24 nie puszczał poprawek MG przez walidator MG. Fakt: GPT widział błędy w ok. 50% sprawdzonych wierszy (MG 86%), ale poprawiono tylko 1,7% zbioru.",
   "worth": "warunkowo",
   "worth_note": "Osobny brief dla MG i walidator po każdej poprawce; lepiej pełny przegląd MG niż próbka."
  },
  {
   "topic": "gm25: nowe rundy MG pisane przez Codex",
   "category": "dane",
   "runs": [
    "run25"
   ],
   "what": "Generator rund zgodny z silnikiem (nacisk na regenerację, drużyny 3+, ucieczki), narracje Codexa w 2 wariantach, wybór walidatorem, sędzia Codex, ręczna próbka Claude.",
   "effect": "v1: 554 wiersze, sędzia Codex dał 0 ciężkich, ale niezależny przegląd 60 wierszy znalazł 8 (13%) z ciężkim błędem – zawsze dopisany „drugi/inny X” przy powtórzonej nazwie potwora. v2.1: 579 wierszy, suchy styl 39% → 17,8%, próbka 60: 0 ciężkich, 3 lekkie (5%).",
   "verdict": "w_toku",
   "why": "Fakt: autor i sędzia to ten sam model (czułość sędziego 19/20 na sztucznie zepsutych), więc błędu bliźniaków nie widział. Fakt: rundy pochodzą z symulatora w Pythonie, nie z gry.",
   "worth": "tak",
   "worth_note": "Przed treningiem ręczny przegląd; efekt dopiero po run25."
  },
  {
   "topic": "DPO „b”: pary wzorzec kontra błąd modelu",
   "category": "dpo",
   "runs": [
    "run23"
   ],
   "what": "390 par z przeglądu + ok. 280 par, w których dobra odpowiedź jest wzorcowa, a zła to prawdziwa odpowiedź modelu z ciężkim błędem. lr 5e-5, 1 epoka.",
   "effect": "Margines 0,38, trafność 0,82. Dev 13,9 (ΔD −3,9 [−7,7; −0,2]) – jedyny wariant z poprawą ponad szum. W ślepej ocenie rozmowy 92 vs 148 ciężkich (vs run21), MG 26. Opis MG skraca się (ok. 160 → 142–145 znaków).",
   "verdict": "pomoglo",
   "why": "Fakt: w parach było 75–84 par MG z krótszą dobrą odpowiedzią i kilkaset par obcych formatów (dwie analizy liczą skład nieco inaczej). Fakt: logp dobrej odpowiedzi spadło tylko o 1,6 nata.",
   "worth": "tak",
   "worth_note": "Stała część przepisu: złe odpowiedzi z własnego SFT, ≥ 80–100 par MG, pary obcych formatów, Umysł poza DPO."
  },
  {
   "topic": "DPO z poprawkami pisanymi przez GPT",
   "category": "dpo",
   "runs": [
    "run10d"
   ],
   "what": "Pary „zła odpowiedź modelu / poprawka GPT”, 278 par, różne lr.",
   "effect": "lr 5e-6: nic (trafność 57%). lr 5e-5 × 2 epoki: przestrzelenie. 10d: czyste 84% → 94%, liczby słownie 41 → 2, ale emocje prawie znikły (25/16/13/13 → 3/1/1/5).",
   "verdict": "mieszane",
   "why": "Fakt: GPT w 48 parach zamienił znacznik na [spokój], a jego poprawki były ogólnie spokojniejsze. Model nauczył się „być spokojnym”.",
   "worth": "warunkowo",
   "worth_note": "Tylko gdy poprawka różni się od złej odpowiedzi samym błędem (ten sam ton, znacznik i długość)."
  },
  {
   "topic": "DPO na błędach z sesji gry",
   "category": "dpo",
   "runs": [
    "run12"
   ],
   "what": "542 pary z prawdziwych rozmów, dobre odpowiedzi krótsze i nie spokojniejsze od złych.",
   "effect": "Automat 90/92% → 98/98%, powtórki 14/10 → 2/1, [spokój] 68% → 55%. Ale kpina w złych miejscach, dziwne słowa; w ślepej ocenie MG run12h tylko 46% rund bez ciężkich błędów.",
   "verdict": "mieszane",
   "why": "Fakt: automat pokazał sukces, sędziowie nie. Domysł: DPO przechyliło charakter w stronę kpiny.",
   "worth": "warunkowo",
   "worth_note": "Oceniać DPO ślepą oceną, nie automatem."
  },
  {
   "topic": "DPO z dłuższymi dobrymi odpowiedziami",
   "category": "dpo",
   "runs": [
    "run13"
   ],
   "what": "Pary, w których poprawka dopisywała brakujące fakty walki.",
   "effect": "MG wierniejszy (79% → 88% rund bez ciężkich), ale długość 111 → 238 znaków, automat 88/94% → 62/60%.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: dobre odpowiedzi były systematycznie dłuższe, więc DPO nauczyło się „pisz dłużej”.",
   "worth": "nie",
   "worth_note": "Pary muszą mieć podobną długość."
  },
  {
   "topic": "DPO na subtelnych parach z własnego modelu (run23d)",
   "category": "dpo",
   "runs": [
    "run23"
   ],
   "what": "390 par: dwie odpowiedzi tego samego modelu o podobnej długości, lepsza i gorsza. lr 2e-5, 1 epoka.",
   "effect": "Trening prawie bez efektu: margines 0,03, eval loss 0,693 → 0,678. Dev 16,0 vs SFT 17,9, ale przedział [−4,5; +0,7] obejmuje zero.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: pary były za subtelne, a lr za niskie. Fakt: pary trafiały nie tam, gdzie były błędy (rozmowy 32% par przy 6% błędów, MG 12% par przy 23% błędów, Umysł 0 par).",
   "worth": "nie",
   "worth_note": "Pary dobierać wg policzonych błędów, lr 5e-5."
  },
  {
   "topic": "Inne kroki po SFT: rejection sampling i KTO",
   "category": "dpo",
   "runs": [
    "run23"
   ],
   "what": "Wariant c: dodatkowe SFT na najlepszych własnych odpowiedziach. Wariant k: KTO (uczenie z ocen dobre/złe bez par).",
   "effect": "c: dev 17,4 (ΔD −0,4, szum). k: dev 15,9, ale odpada, bo udział nastroju [spokój] wzrósł o 7 pkt (styl spłaszczony).",
   "verdict": "nie_pomoglo",
   "why": "Fakt: żaden nie przeszedł reguły dev.",
   "worth": "nie",
   "worth_note": "Brak powodu, by wracać przed lepszym DPO."
  },
  {
   "topic": "DPO run24d na cudzych parach, bez MG",
   "category": "dpo",
   "runs": [
    "run24"
   ],
   "what": "DPO na 495 parach: złe odpowiedzi pochodziły z run23, nie z trenowanego modelu; 0 par MG i obcych formatów, 55 par Umysłu; dodano rpo_alpha 0,2.",
   "effect": "Margines 0,72 i trafność 92%, ale dev gorszy (ΔD +0,7), styl spłaszczony (−17% długości, [spokój] +10 pkt), Umysł zepsuty (0/15 → 4/15). MG na dev lepsze (27,7% → 19,1%).",
   "verdict": "nie_pomoglo",
   "why": "Fakt: logp dobrej odpowiedzi spadło o 9 natów (w „b” 1,6). Domysł (hipoteza obu analiz): pary spoza rozkładu modelu – przed treningiem model wolał złą odpowiedź (−25,2) od dobrej (−27,6).",
   "worth": "nie",
   "worth_note": "Robić DPO jak „b” i przerywać, gdy logp dobrej odpowiedzi spada o więcej niż ok. 2 naty."
  },
  {
   "topic": "Małe DPO z własnych odpowiedzi modelu",
   "category": "dpo",
   "runs": [
    "run16dh",
    "run17dh"
   ],
   "what": "Model odgrywał scenariusze, automat wybierał lepszą i gorszą odpowiedź. Tylko 135–239 par, 17–29 kroków.",
   "effect": "Strata prawie bez ruchu (0,693 → 0,678). run16dh: MG 13 → 7 ciężkich. run17: 29 → 21 → 17 ciężkich na 160.",
   "verdict": "niejasne",
   "why": "Domysł: tak mały trening prawie nie zmienia wag; część efektu to szum. Kierunek w MG był jednak spójny.",
   "worth": "warunkowo",
   "worth_note": "Analiza 22.10: dopiero od ok. 500 par wybranych przez sędziego, nie automat. Późniejsze runy (poza zakresem) wróciły do DPO."
  },
  {
   "topic": "Wyższe lr i więcej epok DPO (wariant e)",
   "category": "dpo",
   "runs": [
    "run23"
   ],
   "what": "Te same 390 par, lr 5e-5, 2 epoki.",
   "effect": "Margines 0,19; dev 15,5 (ΔD −2,4 [−5,8; +1,2]) – poprawa w granicach szumu. MG na dev najlepsze (16,0%).",
   "verdict": "niejasne",
   "why": "Fakt: sam wyższy lr nie wystarcza, gdy pary są słabe.",
   "worth": "nie",
   "worth_note": "Lepszy był wariant z parami wzorcowymi."
  },
  {
   "topic": "Szum pomiaru: dwa ziarna zamiast jednego",
   "category": "ocena",
   "runs": [
    "run4",
    "run5"
   ],
   "what": "Każdy bench puszczaliśmy dwa razy z innym ziarnem losowania (42 i 7).",
   "effect": "Ten sam run5 dał 72 i 78 na 80. Różnica ±3–4 punkty to sam przypadek.",
   "verdict": "pomoglo",
   "why": "Fakt: przy temperaturze 0,8 wynik pojedynczego przebiegu skacze.",
   "worth": "tak",
   "worth_note": "Zawsze co najmniej dwa ziarna, a decyzje tylko przy różnicy większej niż szum."
  },
  {
   "topic": "Stały podział danych po haszu",
   "category": "ocena",
   "runs": [
    "run3",
    "run4"
   ],
   "what": "Losowy podział na trening i walidację zastąpiliśmy stałym, liczonym z treści wiersza.",
   "effect": "Liczby brak. Wcześniej val był zawyżony, bo rozmowy z historią dzieliły fragmenty.",
   "verdict": "pomoglo",
   "why": "Fakt: podobne przykłady trafiały do obu części. Stały podział pozwala porównywać runy.",
   "worth": "tak",
   "worth_note": "Od początku każdego projektu."
  },
  {
   "topic": "Bench formatu kontra prawdziwe rozmowy",
   "category": "ocena",
   "runs": [
    "run4",
    "run5",
    "run8",
    "run12"
   ],
   "what": "Bench sprawdzał tylko format (liczby, zdania, nastrój). Obok odtwarzaliśmy prawdziwe rozmowy z gry.",
   "effect": "run5: bench w szumie, a odtworzenia wyraźnie lepsze. run12: 98% w automacie, ale tylko 46% rund MG bez ciężkich błędów na ślepo.",
   "verdict": "pomoglo",
   "why": "Fakt: automat nie widzi sensu, bełkotu ani zmyśleń. Korelacja automatu z sędziami w rozmowach: 0,28.",
   "worth": "tak",
   "worth_note": "Automat tylko jako filtr formatu; decyzje na podstawie odtworzeń i ślepej oceny."
  },
  {
   "topic": "Ślepa ocena przez sędziów z kluczem i testem znaków",
   "category": "ocena",
   "runs": [
    "run13",
    "run14",
    "run17",
    "run18"
   ],
   "what": "Odpowiedzi modeli pod losowymi literami ocenia kilku pomocników-sędziów; dwie niezależne rundy, model-kotwica, liczenie par „lepszy/gorszy”.",
   "effect": "Zgodność sędziów: MG 157/160, rozmowy 78/96. run18 pod dwiema literami dostał niemal identyczne oceny. Pokazała, że automat „kłamie”.",
   "verdict": "pomoglo",
   "why": "Fakt: jedyna miara, która widzi bełkot, zmyślenia i złego sprawcę w walce.",
   "worth": "tak",
   "worth_note": "Ryzyko: sędziowie i autorzy danych to te same modele; zgodności z człowiekiem nie sprawdzono (brak danych)."
  },
  {
   "topic": "Szum samego treningu (ziarno)",
   "category": "ocena",
   "runs": [
    "run17",
    "run17s"
   ],
   "what": "Ten sam zbiór i przepis, tylko inne ziarno losowe.",
   "effect": "MG: 30 → 12 ciężkich na 160 (p=0,017). Rozmowy bez zmian (53 vs 54). Szum próbkowania: run18 MG ziarno 7 → 17, ziarno 11 → 40 ciężkich na 324.",
   "verdict": "pomoglo",
   "why": "Fakt: słaby MG run17 był pechem treningu, nie winą danych. Różnice ±15 ciężkich na 160 nic nie znaczą.",
   "worth": "tak",
   "worth_note": "Przy ważnych treningach 2 ziarna i wybór lepszego (koszt ok. 5 h GPU)."
  },
  {
   "topic": "Reguła decyzji ustalona przed oceną",
   "category": "ocena",
   "runs": [
    "run20v3",
    "run21"
   ],
   "what": "Przed ślepą oceną zapisywaliśmy, kiedy nowy model wygrywa (np. ten sam kierunek na obu testach i p<0,05).",
   "effect": "run20: remis → v3 zgodnie z regułą. run21: nigdzie istotnie gorszy → dopuszczony.",
   "verdict": "pomoglo",
   "why": "Fakt: chroni przed dopasowaniem wniosku do wyniku.",
   "worth": "tak",
   "worth_note": "Stosować przy każdej decyzji o zmianie modelu."
  },
  {
   "topic": "Zestaw dev i reguła wyboru checkpointu",
   "category": "ocena",
   "runs": [
    "run23",
    "run24"
   ],
   "what": "Osobny mały zestaw (świat Gliniane Wzgórza: 20 rozmów, 47 rund MG, 34 prompty obce, 27 Umysłu) z oceną automatyczną D. Wybór checkpointu tylko na nim, nigdy na teście. Od run23 z bootstrapem.",
   "effect": "run23: e15 17,3 vs e20 17,9 → wzięto e20 (różnica < 1 pkt). run24: e15 14,1 vs e20 18,8 (ΔD −4,7 [−8,5; −1,1]) → wzięto e15; po 2,0 ep. MG miało 39,4% błędów. Wyłowił też DPO „b” (ΔD −3,9).",
   "verdict": "pomoglo",
   "why": "Fakt: chroni przed przeuczeniem i przed testowaniem na zestawie testowym. Słabość: samo SFT różni się między ziarnami o 3,7 pkt, a dev nie ma żadnej rundy z regeneracją, która jest największym źródłem błędów MG.",
   "worth": "tak",
   "worth_note": "Powiększyć dev (MG ≥ 150 promptów, rundy z regeneracją, ≥ 60 negatywów Umysłu)."
  },
  {
   "topic": "Dwóch sędziów na pozycję, trzeci przy sporze, kappa",
   "category": "ocena",
   "runs": [
    "run23",
    "run24"
   ],
   "what": "Każdą pozycję oceniali ślepo Opus i Sonnet (litery losowe). Przy sprzecznym kierunku – trzeci sędzia. Zgodność mierzona kappą.",
   "effect": "run23: kappa ciężkich 0,65 (rozmowy) do 0,86 (obce formaty). run24: MG 0,75 bez sporów, G23 0,91. Słabo w parach (0,38–0,39) i w części A run24 (0,35).",
   "verdict": "pomoglo",
   "why": "Fakt: przy kappie 0,75 i braku sporów regres MG run24 był pewny; każdy sędzia osobno go widział. Domysł: części z kappą < 0,4 mają zbyt nieostry brief.",
   "worth": "tak",
   "worth_note": "Do wykrycia małej poprawy (55% wygranych) potrzeba ok. 800 rozstrzygniętych par."
  },
  {
   "topic": "Zestaw testowy od niezależnego autora (G23, ChatGPT)",
   "category": "ocena",
   "runs": [
    "run23",
    "run24"
   ],
   "what": "120 promptów napisał ChatGPT, żeby test nie miał stylu danych pisanych przez Claude. Wybór bez wglądu w odpowiedzi modeli.",
   "effect": "run23: 57,5 ciężkich vs 86,5 run21 (p=0,0097). run24 vs run23: 57 vs 65 (p=0,88, remis).",
   "verdict": "pomoglo",
   "why": "Fakt: potwierdził przewagę run23 na obcym stylu promptów.",
   "worth": "tak",
   "worth_note": "Dobry strażnik; planowane t24 (wielotur) i SN24 jeszcze nie powstały."
  },
  {
   "topic": "Strażnicy zewnętrzni: EQ-Bench, MT-Bench PL, PingPong-PL",
   "category": "ocena",
   "runs": [
    "run23"
   ],
   "what": "Publiczne testy sprawdzające, czy model nie stracił ogólnych umiejętności bazy.",
   "effect": "EQ-Bench 76,4 vs baza 75,9; MT-Bench PL 5,98 vs 5,96; opóźnienie 1,86 s vs 1,73 s (+7,5%). PingPong-PL: baza 3,34, run21 3,91, run22 3,96, run23 4,05.",
   "verdict": "pomoglo",
   "why": "Fakt: brak regresu, ale zapas minimalny. MT-Bench ma kilkanaście błędnych odpowiedzi wzorcowych.",
   "worth": "tak",
   "worth_note": "Zostawić jako bezpiecznik; PingPong jako raport."
  },
  {
   "topic": "Wdrożenie run23 mimo remisu wg reguły",
   "category": "ocena",
   "runs": [
    "run23"
   ],
   "what": "Reguła ustalona przed treningiem dała remis (M1, M4, B3 niespełnione), ale run23 był lepszy od run21 we wszystkich częściach. Użytkownik przepiął grę.",
   "effect": "Granie 09.11: krótkie, w roli, odporne na „czy jesteś AI?”, targ zgodny z regułami, odpowiedź 1,5–3,5 s. Słabe: notatki Umysłu, pojedyncze sprzeczności w jednej kwestii.",
   "verdict": "pomoglo",
   "why": "Fakt: progi bezwzględne Umysłu nie spełniał też run21. Domysł: progi były za ostre względem modelu w grze.",
   "worth": "warunkowo",
   "worth_note": "Odstępstwa tylko świadomie i opisane; regułę i tak zapisywać przed treningiem."
  },
  {
   "topic": "Wyciek testu do danych treningowych",
   "category": "ocena",
   "runs": [
    "run14",
    "run15"
   ],
   "what": "W zbiorze run14 było 169 przykładów ze świata testowego (Smocza Góra), 4 z 6 postaci testu.",
   "effect": "Wyniki run14 i wcześniejszych zawyżone. run15 bez wycieku: rozmowy bez ciężkich 42% wobec 71% run14h.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: model zapamiętał postacie testu.",
   "worth": "tak",
   "worth_note": "Lekcja: świat i postacie testu trzymać poza katalogiem treningowym (od test16 tak jest)."
  },
  {
   "topic": "Zmiana wielu rzeczy naraz i wybór checkpointu po teście",
   "category": "ocena",
   "runs": [
    "run12",
    "run14",
    "run16dh",
    "run17dh"
   ],
   "what": "W runach 14–17 zmieniały się jednocześnie silnik, dane, test i przepis; połówki „h” wybierano po wyniku na tym samym teście.",
   "effect": "Analiza 22.10: brak jednego ciągu liczb porównywalnego od run7 do run17; wyniki „h” optymistycznie zawyżone.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: nie da się powiedzieć, co pomogło. Domysł: część „zwycięstw” to szum.",
   "worth": "nie",
   "worth_note": "Jedna zmiana na raz, zamrożony test, osobny zbiór do wyboru checkpointu."
  },
  {
   "topic": "Porównanie samego SFT z modelem po SFT+DPO",
   "category": "ocena",
   "runs": [
    "run24"
   ],
   "what": "Kandydat run24 (samo SFT) poszedł do ślepej oceny przeciw run23 (SFT + DPO). Potem dwie niezależne analizy (Fable i Opus) szukały przyczyny regresu MG.",
   "effect": "Profil błędów MG run24 prawie jak run22 (też samo SFT): 49,5 vs 42 ciężkich, dopowiedziane zdarzenia 12 vs 11,5, słowa spoza protokołu 4,51 vs 4,46 (różnica +0,05 [−0,12; +0,22]). run23 po DPO: 26 / 2 / 3,5. SFT vs SFT na dev: run24 lekko lepszy (ΔD −3,2 [−7,2; +0,6]).",
   "verdict": "nie_pomoglo",
   "why": "Fakt: porównanie nierówne – run24 nie miał „szlifu” DPO. Hipoteza (nieudowodniona): ok. 16 z 24,5 dodatkowych ciężkich błędów MG to brak DPO. run23 po samym SFT nigdy nie był oceniony ślepo.",
   "worth": "tak",
   "worth_note": "Lekcja: porównywać SFT z SFT i DPO z DPO; tani pomiar run23s vs run24s-e15 rozstrzygnąłby sprawę."
  },
  {
   "topic": "Reset: SFT od czystej bazy zamiast łańcucha DPO",
   "category": "przepis",
   "runs": [
    "run14"
   ],
   "what": "Zamiast nakładać kolejne DPO, trening od bazy na przebranym, czystszym zbiorze.",
   "effect": "MG: 46% (run12h) → 86–94% rund bez ciężkich błędów; ocena rozmów 2,75 → 3,46–3,79.",
   "verdict": "pomoglo",
   "why": "Fakt: kierunek pewny. Wielkość zawyżona wyciekiem testu.",
   "worth": "tak",
   "worth_note": "Gdy łańcuch poprawek się plącze, wrócić do czystego SFT."
  },
  {
   "topic": "1,5 epoki zamiast 2",
   "category": "przepis",
   "runs": [
    "run24"
   ],
   "what": "Wybór checkpointu z połowy drugiej epoki, gdy dev pokazał, że koniec treningu jest gorszy.",
   "effect": "run24: dev D 14,1 (1,5 ep.) vs 18,8 (2,0 ep.); MG 27,7% vs 39,4%. eval_loss najniższy po 1,0 ep. (0,503), potem 0,512–0,517.",
   "verdict": "pomoglo",
   "why": "Fakt: na danych run24 druga epoka psuje MG (vs run23 e20: p=0,004). Domysł: kandydat 1,5 ep. może być w połowie tego zjawiska.",
   "worth": "tak",
   "worth_note": "Zostać przy 1,0–1,5 ep. i wybierać na dev."
  },
  {
   "topic": "Druga epoka i checkpoint po pierwszej („h”)",
   "category": "przepis",
   "runs": [
    "run8",
    "run14",
    "run18",
    "run19"
   ],
   "what": "Trenowaliśmy 2 epoki i porównywaliśmy z zapisem po pierwszej.",
   "effect": "eval_loss zawsze najniżej po 1. epoce (run14 0,871, run18 0,817, run19 0,798), w 2. rośnie o 0,03–0,05. Ale run19h w rozmowach istotnie gorszy (ocena 3,42 vs 3,70, p=0,001); run14h lepszy w MG.",
   "verdict": "mieszane",
   "why": "Fakt: niższa strata ≠ lepsze rozmowy. Domysł: checkpoint w połowie cosinusa to inny obiekt niż trening 1-epokowy.",
   "worth": "warunkowo",
   "worth_note": "Wybierać checkpoint na osobnym zbiorze, nie po stracie i nie na teście porównawczym."
  },
  {
   "topic": "Kara za powtórzenia (repeat_penalty 1.1)",
   "category": "przepis",
   "runs": [
    "run23"
   ],
   "what": "Sprawdzono, czy ustawienie generowania „kara za powtórki” zmniejszy powtarzanie fraz przez run21.",
   "effect": "Powtórki bez zmian (echo4 9 = 9), ale liczby słownie 37 vs 2, pominięte ceny 36 vs 1, odpowiedzi dłuższe o 40–45%, odpowiedzi bez uwag 78% vs 91%.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: kara obejmuje ostatnie 64 tokeny całego kontekstu, także blok [GRA] z cenami, więc model unika cyfr z polecenia. Odkrycie: gra dotąd działała bez kary.",
   "worth": "nie",
   "worth_note": "Powtórki zostawić DPO i osłonie silnika."
  },
  {
   "topic": "Limit 4096 tokenów w treningu",
   "category": "przepis",
   "runs": [
    "run24",
    "run24v31"
   ],
   "what": "Skrypt treningu odrzuca przykłady dłuższe niż 4096 tokenów zamiast je przycinać.",
   "effect": "W run24 po cichu odpadło 127–129 wierszy, wszystkie z Z5 (11,7% wierszy Z5), w tym 81 najgłębszych tur z 7 kwestiami gracza – dokładnie te, dla których Z5 powstało. W run22 i run23: 0.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: długie karty kupców + długa historia przekraczały limit. run24v31 traci te same 129 wierszy.",
   "worth": "tak",
   "worth_note": "Przycinać historię (np. do 4–5 wymian) i sprawdzać tokenizerem w buildzie, że nic nie odpada."
  },
  {
   "topic": "Błąd tokenizacji „▁[”",
   "category": "przepis",
   "runs": [
    "run14",
    "run15"
   ],
   "what": "Odpowiedź tokenizowana osobno zaczynała się od „▁[” zamiast „[”. Dotyczyło wszystkich SFT do run14.",
   "effect": "Poprawione od run15. Wpływu na jakość nie zmierzono (run14 przed poprawką miał najlepszy MG).",
   "verdict": "niejasne",
   "why": "Domysł: zbieg okoliczności albo nie — nikt nie sprawdził na tym samym zbiorze.",
   "worth": "nie",
   "worth_note": "Brak danych o wpływie; poprawka zostaje."
  },
  {
   "topic": "Zupa adapterów run21 + run22",
   "category": "przepis",
   "runs": [
    "run23"
   ],
   "what": "Uśrednienie dwóch wytrenowanych adapterów (0,5 + 0,5) bez nowego treningu, jako tani zapas.",
   "effect": "Brak wyniku. Adaptery były wyraźnie różne (podobieństwo 0,60), ale przy zapisie GGUF dysk WSL zapełnił dysk E: do zera i WSL przestał startować.",
   "verdict": "niejasne",
   "why": "Fakt: pomiaru nie dokończono. Nie wiadomo, czy zupa byłaby lepsza.",
   "worth": "warunkowo",
   "worth_note": "Tanie, ale tylko z plikami pośrednimi na dysku z miejscem (np. D:)."
  },
  {
   "topic": "Poprawki w grze zamiast kolejnego treningu",
   "category": "silnik",
   "runs": [
    "run4",
    "run7",
    "run10d",
    "run11",
    "run16",
    "run18"
   ],
   "what": "Gdy model „bełkotał”, sprawdzaliśmy w logu, co dokładnie dostał. Często winne było polecenie gry: zła rola, brak nastroju, „ok” czytane jako zgoda, „starucho” jako Stary Bór.",
   "effect": "run11 sesja 2: brak znacznika 14/48 → 0/32, k20 w MG 0/113. Analiza 22.10: najlepszy stosunek zysku do kosztu w całym projekcie.",
   "verdict": "pomoglo",
   "why": "Fakt: wiele błędów wspólnych wszystkim modelom wynikało z poleceń gry.",
   "worth": "tak",
   "worth_note": "Zanim douczysz model, sprawdź w logu, co dostał."
  },
  {
   "topic": "Podział długich rund walki",
   "category": "silnik",
   "runs": [
    "run18",
    "run19"
   ],
   "what": "Silnik dzieli rundy z ≥5 zdarzeniami na części po ≤3 i prosi o najwyżej dwa krótkie zdania.",
   "effect": "Ciężkie błędy w długich rundach: run18 16,0% → 2,1% (p<0,001), run19 17,0% → 7,4%. Lekkich 2× więcej.",
   "verdict": "pomoglo",
   "why": "Fakt: model gubił się przy wielu zdarzeniach naraz (2% błędów przy 1–2 zdarzeniach, 12–19% przy ≥5).",
   "worth": "tak",
   "worth_note": "Uznana za najlepszą decyzję tamtych dni; zostaje w silniku."
  },
  {
   "topic": "Osłona: fakt z karty nie trafia do biografii",
   "category": "silnik",
   "runs": [
    "run23"
   ],
   "what": "Silnik odrzuca notatkę Umysłu, jeśli wszystkie jej słowa (albo wiek) już są w karcie postaci.",
   "effect": "Fałszywe notatki na S: run22 8/60 → 0/60, run21 4/60 → 0/60. Trafienia bez zmian (168/175 i 117/175).",
   "verdict": "pomoglo",
   "why": "Fakt: zmierzone offline na zapisanych odpowiedziach. Później okazało się, że reguła łapie tylko dosłowne powtórzenia, a nie parafrazy karty (8/90 w SN23).",
   "worth": "tak",
   "worth_note": "Rozwijać w stronę parafraz (zrobione częściowo w etapie 0 run24)."
  },
  {
   "topic": "Osłony: cena bez pytania i pytanie-echo",
   "category": "silnik",
   "runs": [
    "run23"
   ],
   "what": "Silnik prosi o poprawkę, gdy postać podaje kwotę, choć nie wolno. Krótkie powtórzenie słów gracza ze znakiem zapytania („Na szczury? Bosak.”) nie jest już liczone jako zakazane pytanie.",
   "effect": "Offline na t18: osłona ceny złapała 10 (run21) i 13 (run22) kwestii na 1065; wszystkie 23 to prawdziwe błędy, 0 fałszywych alarmów. 3 niepotrzebne powtórki mniej przy pytaniach-echo.",
   "verdict": "pomoglo",
   "why": "Fakt: trafienia przejrzane ręcznie. Osłona jest miękka: po 2 próbach mówi się ostatnią wersję.",
   "worth": "tak",
   "worth_note": "W grze od d33d36b."
  },
  {
   "topic": "Lepki temat prywatny i prompt Umysłu v2 w silniku",
   "category": "silnik",
   "runs": [
    "run22"
   ],
   "what": "Silnik pamięta pytanie prywatne przez 2 kwestie i pyta Umysł przy każdej kwestii. Prompt Umysłu uczy czytać krótkie odpowiedzi razem z pytaniem.",
   "effect": "Umysł zapisuje więcej: trafienia na krótkich odpowiedziach 45% → 70% (9/20 → 16/23). Puste biografie 18/24 → 12/24. Ale w teście 24 par sprzeczności 16/72 → 18/72, czyli remis (p=0,46). Po poprawce runda bis też remis (11/48 vs 11/48).",
   "verdict": "mieszane",
   "why": "Fakt: tam, gdzie fakt trafił do pamięci, sprzeczności znikały. Fakt: przy pustej pamięci model i tak wymyślał krewnych na podchwytliwe pytania. Domysł: 24 pary to za mało, by zobaczyć zysk netto.",
   "worth": "tak",
   "worth_note": "Zmiany weszły do gry. Warto mierzyć na większej próbie (≥ 40 par, nowe światy)."
  },
  {
   "topic": "Strażnik sprzeczności z biografią",
   "category": "silnik",
   "runs": [
    "run22"
   ],
   "what": "Silnik sprawdza, czy kwestia postaci przeczy temu, co już o sobie powiedziała, i wtedy prosi o poprawkę.",
   "effect": "Fałszywe alarmy 1 na 305 kwestii (0,3%). Łapie jednak tylko 4 z 9 sprzeczności (próg był 6/9).",
   "verdict": "mieszane",
   "why": "Fakt: 5 chybionych to sprawy spoza zakresu reguły (np. zmiana zawodu brata, literówka). Strażnik jest miękki: po dwóch złych próbach kwestia i tak idzie.",
   "worth": "warunkowo",
   "worth_note": "Tani w utrzymaniu; więcej da, gdy Umysł będzie lepiej zapisywał biografię."
  },
  {
   "topic": "Etap 0: kolejność pól JSON Umysłu i inne poprawki silnika",
   "category": "silnik",
   "runs": [
    "run23",
    "run24"
   ],
   "what": "Silnik wysyłał schemat JSON z polami alfabetycznie (z mapy Go), a dane treningowe miały inną kolejność. Poprawiono kolejność, filtr śmieci w notatkach i rozmyty filtr „fakt z karty” (commity d390e13, 985238d).",
   "effect": "run23 bez zmian modelu: fałszywe notatki SN23 8/90 → 4/90, w teście krótkich odpowiedzi 3/29 → 1/29. W grze zniknęły śmieci typu \"fact\":\",\". Ale 12.11 trafienia w krótkich odpowiedziach spadły do 16/23 (było 20/23) u run23 i run24.",
   "verdict": "mieszane",
   "why": "Fakt: błąd był niewidoczny w ocenach Pythona, bo miały własne schematy. Zastrzeżenie: osłonę strojono na tych samych danych SN23, więc wynik jest optymistyczny. Spadek trafień przypisano zmianie 985238d.",
   "worth": "tak",
   "worth_note": "Ocena musi używać dokładnie tych samych schematów co gra."
  },
  {
   "topic": "Niższa temperatura próbkowania",
   "category": "silnik",
   "runs": [
    "run4-t0.6",
    "run8"
   ],
   "what": "Ten sam model, ale ostrożniejsze losowanie słów: 0,6 zamiast 0,8 (run4), 0,7 (run8-e1).",
   "effect": "run4: bench 75 zamiast 78. run8-e1 przy 0,7: bez wyraźnej poprawy, zmyślenia zostały.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: format się nie poprawił. Domysł: błędy siedzą w samym modelu, nie w losowaniu.",
   "worth": "warunkowo",
   "worth_note": "Analiza z 22.10 proponowała 0,6 + min_p + repeat_penalty na bełkot i powtórki; tego testu w źródłach nie ma (brak danych)."
  },
  {
   "topic": "Numeracja bliźniaczych potworów w silniku",
   "category": "silnik",
   "runs": [
    "run25"
   ],
   "what": "Gdy na polu walki są dwa potwory o tej samej nazwie, silnik podaje MG „Topielec 1”, „Topielec 2” (commity 50b852c, f015c1c). Gracz widzi zwykłe nazwy, a MG pisze „pierwszy/drugi Topielec”.",
   "effect": "W danych gm25 v2.1: bliźniaki w 244 z 579 wierszy, 0 niezgodnych liczebników, 0 cyfr po nazwie w narracji. W modelu jeszcze niezmierzone.",
   "verdict": "w_toku",
   "why": "Fakt: przyczyną błędu „drugi X” był protokół, z którego nie dało się poznać, czy potwory są dwa. Poprawka u źródła zamiast w danych.",
   "worth": "tak",
   "worth_note": "Sprawdzić na dev/teście po run25."
  },
  {
   "topic": "Droga z treningu do Ollamy",
   "category": "sprzet",
   "runs": [
    "run1",
    "run2",
    "run7"
   ],
   "what": "Wtopiony model z Maca importowany wprost do Ollamy psuł tokenizer Bielika.",
   "effect": "run1: śmieci [UNK_BYTE_…]. Naprawione w run2: fuse → pliki tokenizera → llama.cpp → GGUF q8_0 → ollama create.",
   "verdict": "pomoglo",
   "why": "Fakt: wtopione wagi MLX nie niosą tokenizera.",
   "worth": "tak",
   "worth_note": "Zawsze przez llama.cpp, bez skrótów."
  },
  {
   "topic": "Wznawianie treningu z checkpointu",
   "category": "sprzet",
   "runs": [
    "run24"
   ],
   "what": "Dodano wznowienie pełnego stanu treningu (--resume-from) po awariach: restart WSL 10.11 o 18:37 (krok 175, bez checkpointu) i restart Windows (aktualizacja) 11.11 o 2:30 (krok 815).",
   "effect": "Trening wznowiony z checkpointu 646; eval_loss przebiegał jak w run23 (1,0 ep.: 0,5031 vs 0,5025). Śladów szkody brak.",
   "verdict": "pomoglo",
   "why": "Fakt: checkpointy co 646 kroków uratowały ok. 5 h pracy GPU. Zastrzeżenie: to odstępstwo od „przepisu bez zmian”, nieopisane w DECYZJA.",
   "worth": "tak",
   "worth_note": "Checkpointy częściej niż co ćwierć epoki i wyłączone automatyczne restarty Windows na czas treningu."
  },
  {
   "topic": "Pamięć karty graficznej 16 GB",
   "category": "sprzet",
   "runs": [
    "run7",
    "m7mind",
    "run21"
   ],
   "what": "W karcie musiały się zmieścić: model rozmów, Umysł i synteza mowy (Chatterbox).",
   "effect": "run7 (7 GB) + gemma (4 GB) + Chatterbox (4 GB) = cała karta; synteza zwalniała do 20–47 s na zdanie. run18 Q5 + m7mind się nie mieścił. run21 sam: 8,7 GB zamiast 12,4 GB.",
   "verdict": "mieszane",
   "why": "Fakt: ograniczenie sprzętowe kierowało wyborem modeli (Minitron, Q4, jeden model).",
   "worth": "tak",
   "worth_note": "Mierzyć pamięć całego zestawu, nie pojedynczego modelu. Pomaga OLLAMA_KV_CACHE_TYPE=q8_0 z flash attention."
  },
  {
   "topic": "Kompresja Q4 zamiast Q5",
   "category": "sprzet",
   "runs": [
    "run7",
    "run16dhq4",
    "m7mind"
   ],
   "what": "Zapis modelu w 4 bitach, żeby oszczędzić pamięć.",
   "effect": "run7 Q4: 89% vs 92% czystych. run16dhq4: MG bez straty (p=0,71), rozmowy gorsze (p=0,05). m7mindq4: jakość jak m7mind przy 4,5 GB.",
   "verdict": "mieszane",
   "why": "Fakt: kompresja kosztuje w rozmowach, mniej w prostszych zadaniach (MG, Umysł).",
   "worth": "warunkowo",
   "worth_note": "Q4 dla Umysłu lub MG tak; dla rozmów raczej nie. Q6/Q8 nie testowane (brak danych)."
  },
  {
   "topic": "Wspólne GPU z innymi programami",
   "category": "sprzet",
   "runs": [
    "run24"
   ],
   "what": "W trakcie treningu na tej samej karcie działały: Minecraft, syntezator mowy (TTS) i Unreal Editor.",
   "effect": "TTS i Unreal spowolniły trening z ok. 25 do 70 s na krok (pamięć karty przelała się do pamięci współdzielonej); po ich zamknięciu prędkość wróciła sama. Na granie trening był pauzowany (pkill -STOP/-CONT).",
   "verdict": "mieszane",
   "why": "Fakt: brak awarii, tylko spowolnienie. Pauza działa bez utraty postępu.",
   "worth": "warunkowo",
   "worth_note": "Pauzować trening zamiast dzielić kartę."
  },
  {
   "topic": "Brak miejsca i pamięci przy zapisie modelu",
   "category": "sprzet",
   "runs": [
    "run5",
    "style1",
    "run13",
    "run14",
    "run20v3"
   ],
   "what": "Końcowe kroki (wtapianie, konwersja do GGUF, ollama create) potrzebują dużo RAM i dysku.",
   "effect": "run5: zabrakło pamięci przy wtapianiu. style1: zapchany dysk C:, WSL padł. run13: dysk WSL urósł do 151 GB. run14: pełny D:. run20v3: pełny E:. GGUF na wolnym dysku G: dokładał 1 h 15 min.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: konwersja 11B potrzebuje szczytowo ok. 52 GB; dysk WSL (vhdx) nie oddaje miejsca zwolnionego w środku.",
   "worth": "warunkowo",
   "worth_note": "Lekcja infrastrukturalna: przed treningiem sprawdzić wolne miejsce, skrypty wznawiania (resume) od razu."
  },
  {
   "topic": "Pułapki narzędzi na PC (WSL, biblioteki, skrypty)",
   "category": "sprzet",
   "runs": [
    "run8",
    "run9b",
    "run18",
    "run21"
   ],
   "what": "Drobne awarie infrastruktury w trakcie przesiadki na PC.",
   "effect": "run8: transformers 5.17 nie znał warmup_ratio. run9: --resume wychodził poza 16 GB. run18: zły sort i limit 3 checkpointów — model po 1. epoce przepadł. run21: Ollama w Windows nie działała, testy zgubiły serwer po zmianie IP. Pendrive 2 TB okazał się fałszywy.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: każda awaria kosztowała godziny, ale nic nie przepadło na stałe poza checkpointem run18.",
   "worth": "warunkowo",
   "worth_note": "Lekcja: trzymać więcej checkpointów (teraz 6), mieć skrypty wznawiania i porównywać kopie przed usunięciem."
  },
  {
   "topic": "Dysk i pamięć WSL",
   "category": "sprzet",
   "runs": [
    "run23",
    "run24"
   ],
   "what": "Pliki pośrednie (scalone wagi 22 GB + GGUF 22 GB) zapisywane w wirtualnym dysku WSL; wysyłka na HF przez WSL.",
   "effect": "06.11 dysk WSL urósł do 231 GB i zapełnił dysk E: do zera; WSL nie startował, padły procesy innych agentów, zupa adapterów przepadła. 11.11 wysyłkę na HF dwa razy zabiło zwalnianie pamięci (cache stron WSL); pomogło okresowe czyszczenie cache (dropcache.timer co 10 min).",
   "verdict": "nie_pomoglo",
   "why": "Fakt: df w WSL pokazuje rozmiar wirtualny (1 TB), a nie miejsce na dysku Windows. Plik vhdx sam nie maleje.",
   "worth": "nie",
   "worth_note": "Duże pliki pośrednie trzymać na D:, przed zapisem sprawdzać wolne miejsce z Windows."
  },
  {
   "topic": "Douczony Umysł (b45mind, m7mind)",
   "category": "umysl",
   "runs": [
    "m7mind"
   ],
   "what": "LoRA na 1255 przykładach zadań Umysłu.",
   "effect": "m7mind: zmyślenia 13% → 0%, fałszywe haczyki 11/15 → 0/15, Interpret 80% → 95%, notatka 0,5 s. b45mind: Invent 0/6 — zadanie z małą liczbą przykładów znikło.",
   "verdict": "pomoglo",
   "why": "Fakt: douczenie bardzo pomaga. Rzadkie zadanie w danych model „zapomina”.",
   "worth": "tak",
   "worth_note": "Pilnować proporcji zadań w danych."
  },
  {
   "topic": "Jeden model do wszystkiego",
   "category": "umysl",
   "runs": [
    "run21"
   ],
   "what": "Rozmowy, MG i Umysł w jednym 11B; Umysł 24% danych.",
   "effect": "Nie gorszy od run20v3 w żadnej części (p 0,37–1,0); Umysł ≈ m7mind; pamięć 8,7 GB zamiast 12,4 GB. Umysł wolniejszy niż gemma (zrozumienie gracza 1,0 s vs 0,65 s).",
   "verdict": "pomoglo",
   "why": "Fakt: dodanie zadań Umysłu nie zepsuło rozmów ani walk.",
   "worth": "tak",
   "worth_note": "Pilnować proporcji Umysłu (plan: 20–25%)."
  },
  {
   "topic": "Z3 i Z8: nowe etykiety Umysłu od GPT",
   "category": "umysl",
   "runs": [
    "run24"
   ],
   "what": "192 notatki SelfNote (co postać powiedziała o sobie) i 220 przykładów odczytu intencji gracza i faktów o świecie.",
   "effect": "Umysł na dev 0/15 fałszywych vs 4/15 (różnica −14,8 pkt, istotna). SN23 surowe fałszywe notatki 5/90 vs 8/90, sq20 2/11 vs 4/11. Trafienia w krótkich odpowiedziach bez zmian (16/23).",
   "verdict": "pomoglo",
   "why": "Fakt: to jedyna część run24 wyraźnie lepsza od run23. Zastrzeżenie: Codex czasem łapał drobny fakt i gubił główny.",
   "worth": "tak",
   "worth_note": "Potrzebny nowy test SN24 i większy test krótkich odpowiedzi."
  },
  {
   "topic": "Umysł w jednym modelu z Głosem i MG",
   "category": "umysl",
   "runs": [
    "run22",
    "run23",
    "run24"
   ],
   "what": "Od run21 jeden model pisze kwestie, opisuje walki i robi notatki Umysłu (ok. 20% danych to Umysł).",
   "effect": "Pamięć karty 8,7 GB zamiast 12,4 GB (dwa modele). run23: opóźnienie +7,5% vs run21; w grze krok z Umysłem 5–9 s. Ale DPO bez par Umysłu psuje notatki (run24d: 0/15 → 4/15).",
   "verdict": "pomoglo",
   "why": "Fakt: osobny model Umysłu odrzucono, bo nie mieści się w 16 GB. Fakt: notatki idą w tle w kolejce za Głosem.",
   "worth": "tak",
   "worth_note": "Pilnować Umysłu przy każdym DPO (trzymać go poza DPO albo dodawać powtórkę jego danych)."
  },
  {
   "topic": "Filtry w grze zamiast treningu gemmy",
   "category": "umysl",
   "runs": [],
   "what": "18.10 zmierzyliśmy gemmę na 115 wywołaniach i zamiast ją trenować dodaliśmy filtry w grze.",
   "effect": "Note: fakt w 36 z 43 kwestii, 82% fałszywych alarmów; po filtrach ok. 22 fałszywych. Na procesorze 10× wolniej (Interpret 6,8 s vs 0,65 s).",
   "verdict": "mieszane",
   "why": "Fakt: filtry odcięły część błędów, ale nie wszystkie (8 faktów z odwróconą treścią).",
   "worth": "warunkowo",
   "worth_note": "Trening ma sens od ok. 500 przykładów Note i 200 Invent — tak też potem zrobiono."
  },
  {
   "topic": "Mały model tylko do opisów walk",
   "category": "umysl",
   "runs": [
    "b45gm"
   ],
   "what": "Bielik 4.5B uczony wyłącznie na rundach MG.",
   "effect": "Ciężkie 13 vs 6–7 u 11B (różnica nieistotna, p=0,45), lekkie 93 vs 35–39.",
   "verdict": "mieszane",
   "why": "Fakt: fakty prawie dobrze, język i nazwy wyraźnie gorzej.",
   "worth": "warunkowo",
   "worth_note": "Tylko ze wspólnym treningiem MG + Umysł i osłoną nazw w silniku."
  },
  {
   "topic": "Krótkie i przeczące odpowiedzi w danych Umysłu",
   "category": "umysl",
   "runs": [
    "run22"
   ],
   "what": "209 przykładów uczących Umysł, że „Nie. Od zimy.” na pytanie o męża to też fakt.",
   "effect": "Trafienia na krótkich odpowiedziach 21/23 (run21: 16/23). Ale więcej fałszywych notatek: sq20 7/11 (run21 4/11), S 8/60 (run21 4/60).",
   "verdict": "mieszane",
   "why": "Fakt: wszystkie 8 fałszywych notatek na S to fakty, które już stały w karcie postaci. To naprawiono w silniku (osobny eksperyment).",
   "worth": "tak",
   "worth_note": "Razem z filtrem „fakt z karty” w silniku."
  },
  {
   "topic": "Gotowe małe modele jako Umysł (bez treningu)",
   "category": "umysl",
   "runs": [
    "m7mind"
   ],
   "what": "Bielik 4.5B, Minitron 7B, goły 11B v3 i qwen2.5:3b porównane z gemmą na zadaniach Umysłu.",
   "effect": "Bielik 4.5B: fałszywe notatki 97–100% (gemma 82%), perswazja 6–7/17 (gemma 13/17). qwen gorszy we wszystkim. Gemma też słaba: zmyśla 52% faktów w SelfNote.",
   "verdict": "nie_pomoglo",
   "why": "Fakt: bez treningu żaden nie zastąpił gemmy.",
   "worth": "nie",
   "worth_note": "Umysł wymaga douczenia."
  }
 ],
 "lessons": [
  "DPO na parach „wzorzec kontra prawdziwy błąd modelu” dało run23 największy skok (rozmowy 92 vs 148 ciężkich), a subtelne pary z tego samego modelu prawie nic.",
  "DPO to stała część przepisu: model po samym SFT trzeba porównywać z SFT, a model po DPO z DPO, inaczej regres można przypisać złej przyczynie (run24).",
  "Pary DPO dobiera się według policzonych błędów, ze złymi odpowiedziami z własnego modelu, i przerywa, gdy prawdopodobieństwo dobrej odpowiedzi spada o więcej niż ok. 2 naty.",
  "Każda poprawka danych musi przejść te same bramki co reszta (walidator MG, powtórki, długość); bez nich GPT usunął liczby z 11 rund MG, a Z5 wniosło powtórki.",
  "Dużo „poprawionych danych” to często mało nowego sygnału: w run24 tylko ok. 10% wierszy niosło coś nowego, a poprawki celowały w inne błędy niż te, które liczą sędziowie.",
  "Wybór checkpointu na osobnym zestawie dev działa (run24: 1,5 ep. zamiast przeuczonego 2,0 ep.), ale dev jest mały i głośny, a ostatnie słowo ma ślepa ocena.",
  "Tanie poprawki silnika (fakt z karty, cena bez pytania, kolejność pól JSON, numeracja bliźniaków) często dają więcej niż trening i od razu działają w grze.",
  "Reguła decyzji zapisana przed treningiem chroni przed oszukiwaniem samego siebie; odstępstwo (wdrożenie run23 mimo remisu) było świadome i opisane.",
  "Zmieniać jedną rzecz naraz: run24v31 to tylko nowa baza, a naprawa MG idzie osobno w run25.",
  "Duże pliki trzymać poza dyskiem WSL, robić częste checkpointy i sprawdzać tokenizerem, że limit 4096 nic po cichu nie wycina."
 ]
};
