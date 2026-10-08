Neuro Processor
===============

Neuro Processor to gra konsolowa inspirowana klasykiem, jakim jest Mózgoprocesor. To moja osobista zabawa w tworzenie czegoś, co łączy retro klimat z nowoczesnymi możliwościami grafiki w terminalu i programowania. Gra jest napisana w Go.

O Grze
======

Neuro Processor jest generatorem gier: opisujesz w paru słowach, o czym ma być świat, a model językowy wymyśla miejsca, przeciwników, przedmioty i mieszkańców osad. Mapę, statystyki, ceny i wszystkie zasady wylicza kod, więc każdy wygenerowany świat jest od razu grywalny. Gotowe światy są też wbudowane.

Gra się jak w Mózgoprocesorze: na ekranie widać rysunek miejsca, w którym stoisz, a polecenia wpisuje się tekstem.

Przy tworzeniu świata wybiera się rodzaj gry:

| Rodzaj | Co to znaczy | Wbudowane przykłady |
|---|---|---|
| **Gra wyobraźni** | mapa, osady, kupcy, drużyna, walka rozstrzygana prostymi liczbami, narrator i rysowane sceny | Zatopione Krypty, Popiół i Rdza, Stacja Erebus |
| **D&D** | ten sam rodzaj świata, ale zasady stołowego RPG: rzut k20 przeciw klasie pancerza (KP), kości broni, trafienia krytyczne, rzuty obronne, inicjatywa i walka na siatce z ruchem, zasięgiem i atakami okazyjnymi | Smocza Góra |
| **Przygodowa** | tekstowa przygodówka jak Mózgoprocesor: pomieszczenia, przedmioty, zagadki, ludzie do pogadania i cel do osiągnięcia; bez walki i statystyk | Hipokamp |

Generator umie tworzyć każdy z tych rodzajów (przygodówkę opisuje innym schematem: pokoje, rzeczy, osoby i kroki zagadek).

Główne Funkcje
==============

*Generator światów*: Nowy świat z opisu gracza. Wygenerowane światy zapisują się na dysku i wracają w menu.

*Wbudowane światy*: Zatopione Krypty (fantasy), Popiół i Rdza (postapokalipsa), Stacja Erebus (science fiction).

*Polecenia tekstowe*: `idź na wschód`, `podnieś miecz`, `rzuć iskrę na topielca`, `idź do zbrojowni`. Odmiana i polskie znaki nie mają znaczenia.

*Widok miejsca*: Każde pole ma swój rysunek, zależny od terenu. Widać na nim osadę, miejsce szczególne, leżące przedmioty i przeciwników, z którymi walczysz.

*Mapa*: 15x9 pól, odkrywana w miarę wędrówki. Im dalej od pierwszej osady, tym groźniej; w jednym z rogów czeka główny przeciwnik.

*Osady i miejsca*: W każdym świecie są osady z budynkami (zbrojownia, zielarz lub medyk, sklep z mocami) oraz miejsca szczególne: strzeżone skrytki i miejsca, które przywracają siły. Dalsze osady sprzedają lepszy sprzęt.

*Walka*: Turowa, z grupami przeciwników, sterowana z menu: wybierasz cel, potem zwykły atak, moc, przedmiot albo ucieczkę. Z pokonanych zostają pieniądze i przedmioty na ziemi.

*Magia*: Moce, których bohater uczy się z przedmiotów i rzuca w walce, oraz noszone przedmioty magiczne. Poznane moce zostają na stałe i można ich znać dowolnie wiele.

*Doświadczenie i poziomy*: Za pokonanych przeciwników bohater dostaje doświadczenie. Każdy poziom (do dziesiątego) dodaje zdrowia, mocy i ataku oraz przywraca siły.

*Targowanie*: Z gospodarzami budynków rozmawia się zwykłym tekstem i negocjuje ceny. Gdy gospodarz w rozmowie zaproponuje coś, czego w świecie nie ma, przedmiot „wpada do świata”: pojawia się w jego sklepie z ceną i działaniem wyliczonym z opisu i można go kupić.

*Mistrz Gry*: Gdy skonfigurowany jest model, po każdej rundzie walki Mistrz Gry opisuje w dzienniku, co się stało. Opis przychodzi w tle i nie zatrzymuje walki; zasady rozstrzygają wszystko same, a Mistrz Gry dostaje tylko ich wynik do opowiedzenia.

*Mowa*: Opcjonalnie kwestie gospodarzy i Mistrza Gry są czytane na głos, a model dobiera do każdej nastrój (kpina, gniew, radość…), który słychać w głosie i widać przy kwestii. Opcjonalnie też mówisz do gry: F5 włącza mikrofon, F5 kończy, a rozpoznany tekst trafia do wiersza poleceń albo do rozmowy. Można kupować po kilka sztuk naraz, np. `kupię 5 naparów za 60`: kwota dotyczy całości, a hurtem jest taniej (do 15% na sztuce).

*Postać*: Przed wejściem do świata wybierasz imię i awatar: jeden z sześciu wbudowanych albo własny, narysowany przez mały lokalny model obrazkowy z twojego opisu i zapisany pod wybraną nazwą.

*Zapis gry*: Polecenie `zapisz` zapisuje stan świata; zapisane gry pojawiają się w menu.

*Towarzysze*: W karczmie każdej osady czekają bohaterowie do wynajęcia: wojownik, uzdrowiciel albo mag. `najmij X` bierze ich do drużyny (najwyżej dwoje), `odpraw X` odsyła. Towarzysz jest pełną postacią: ma poziom i doświadczenie, zdrowie, moc, broń, pancerz, amulet, moce i awatar. Rzeczy dajesz mu poleceniem `daj kord bognie`, odbierasz `odbierz kord od bogny`. Lewy panel ma karty: F1 to twoja postać, F2 i F3 towarzysze (albo `pokaż bognę`), a Tab pokazuje plecak i moce wybranej karty.

*Karta postaci*: Tab otwiera kartę wybranej postaci (F1 ty, F2 i F3 towarzysze): sześć cech jak w D&D (Siła, Zręczność, Kondycja, Inteligencja, Mądrość, Charyzma) z modyfikatorami, liczby bojowe, sprzęt, plecak i moce. Cechy działają na zasady: Siła dodaje do ataku, Zręczność do szybkości i ucieczki, Kondycja do zdrowia z każdym poziomem, Inteligencja do mocy, Mądrość wzmacnia leczenie, a Charyzma zjednuje kupców (niższe ceny) i obniża koszt najmu. Twoja postać dostaje cechy według wybranego awatara (wojownik jest silny, czarodziej bystry), a własny awatar oznacza losowanie czterema kośćmi.

*Legenda na żołdzie*: W każdym świecie jest jeden najemnik najwyższej klasy, dwa razy twardszy od reszty i z obiema wrodzonymi mocami. Nie płaci się za niego z góry, tylko żołd o każdym świcie; dzień mija po dwunastu krokach albo po odpoczynku. Odprawiony przestaje pobierać, a gdy zabraknie na żołd, odchodzi sam i oddaje twoje rzeczy.

*Inicjatywa*: Każda postać i każdy potwór ma szybkość. Bohater zyskuje jej z poziomem i traci pod ciężkim pancerzem, towarzysze zależą od roli (mag najszybszy, uzdrowiciel najwolniejszy), potwory od rodzaju (bestie i duchy szybkie, nieumarli i galarety powolne). Kolejność tur w rundzie wynika z szybkości i jest pokazana w panelu walki. Gdy przychodzi tura towarzysza, decydujesz za niego tym samym menu (atak, moc, przedmiot) albo wybierasz „Auto”; `autowalka` oddaje wszystkich towarzyszy automatowi na stałe. Potwory mogą atakować towarzyszy; nieprzytomnych stawia na nogi odpoczynek w osadzie. Gracz, który nie podejmie decyzji w 20 sekund, po prostu atakuje.

*Gra w kilka osób*: Do tego samego świata mogą dołączyć kolejni gracze i wspólnie walczyć.

Uruchomienie
============

Wymagane jest Go 1.22 lub nowsze i terminal o rozmiarze co najmniej 118x36 z obsługą 256 kolorów.

```
go run ./cmd/neuro
```

| Flaga | Znaczenie |
|---|---|
| `-name IMIĘ` | proponowane imię twojej postaci, domyślnie nazwa użytkownika systemu |
| `-listen ADRES` | otwiera świat dla innych graczy, np. `-listen :2222` |

Polecenia
=========

| Polecenie | Co robi |
|---|---|
| `idź na północ` / `południe` / `wschód` / `zachód`, albo `n`, `s`, `e`, `w` | ruch o jedno pole |
| `rozejrzyj się` | opis miejsca, drogi w cztery strony, leżące przedmioty |
| `podnieś X`, `podnieś wszystko` | zabiera przedmioty z ziemi |
| `upuść X` | odkłada przedmiot na ziemię (tak można go komuś przekazać) |
| `użyj X`, `załóż X`, `wypij X`, `naucz się X` | zakłada broń, pancerz lub amulet, wypija miksturę, uczy mocy |
| `atakuj`, `atakuj X` | atak na wybranego albo wskazanego przeciwnika |
| `rzuć X`, `rzuć X na Y` | rzuca poznaną moc |
| `uciekaj` | próba ucieczki z walki na poprzednie pole |
| `odpocznij` | w osadzie przywraca zdrowie i moc |
| `idź do X`, `rozmawiaj z X` | wejście do budynku w osadzie, np. `idź do zbrojowni` |
| `najmij X`, `odpraw X` | wynajęcie towarzysza z karczmy albo odesłanie go |
| `daj X Y`, `odbierz X od Y` | przekazanie rzeczy towarzyszowi (sprzęt zakłada od razu, moce uczy się, mikstury nosi) i zabranie jej |
| `pokaż Y`, `autowalka` | karta towarzysza w lewym panelu (także F1-F3); towarzysze walczą sami |
| `ulecz` | rzuca najsilniejsze leczenie, na jakie masz moc, na całą drużynę (w walce i poza nią) |
| `Imię, ulecz` / `rzuć X na Y` / `atakuj X` / `wypij X` / `automat` | rozkaz dla towarzysza, np. `Lyrasa, ulecz siebie i załogę`; wystarczy pierwsze słowo imienia |
| `powiedz X` | wiadomość do wszystkich graczy |
| `plecak` | karta postaci: cechy, ekwipunek i poznane moce |
| `bestiariusz` | księga potworów (także F4): kogo już spotkałeś, ile ma zdrowia, jak bije i co potrafi |
| `dziennik` | co wiesz i od kogo, oraz twoje zlecenia |
| `zlecenia`, `przyjmij` | zlecenia kupców: lista i przyjęcie tego, które proponuje gospodarz, u którego jesteś |
| `handluj z karawaną` | targ z wędrownym kupcem, gdy jego karawana stoi na twoim polu (`$` na mapie) |
| `zapisz` | zapisuje grę (tylko gospodarz świata) |
| `pomoc`, `wyjdź` | lista poleceń, wyjście z gry |

Skróty: strzałki chodzą po mapie, Tab otwiera plecak (z listą poznanych mocy), F4 bestiariusz, a pusty Enter powtarza ostatnie polecenie.

Rozkazy dla towarzyszy można też wypowiedzieć (F5). Towarzysz wykonuje rozkaz od razu, gdy trwa jego tura albo gdy nie ma walki (leczenie i mikstury), a w walce poza swoją turą zapamiętuje go i wykonuje, gdy kolej przyjdzie na niego. „Ulecz” wybiera najsilniejszą moc leczącą, na którą starcza mocy: wrodzony Opatrunek uzdrowiciela jest darmowy, wyuczone zaklęcie leczy mocniej, ale kosztuje. Poza walką leczenie stawia na nogi także nieprzytomnych.

W budynku rozmawia się z gospodarzem zwykłym tekstem. Oprócz kupowania i targowania się („ile za kord?”, „dam 30”, „kupię 5 naparów za 100”) można też sprzedawać: „sprzedam kord za 40”, „ile dasz za amulet?”, „kupisz ode mnie napar?”. Kupiec skupuje tylko to, czym sam handluje (kowal broń i pancerze, zielarka mikstury, mag zwoje i amulety), daje za rzecz 40% jej ceny i da się go podciągnąć do 55% (więcej przy wysokiej Charyzmie); każde zbyt wysokie żądanie kosztuje jego cierpliwość. Odkupiona rzecz trafia na półkę i można ją kupić z powrotem po pełnej cenie.

Walka ma własne menu, obsługiwane strzałkami, gdy wiersz poleceń jest pusty: góra i dół wybierają pozycję, Enter przechodzi dalej (cel, akcja, a dla mocy i przedmiotów lista do wyboru), strzałka w lewo wraca. Menu zawsze dotyczy postaci, której tura właśnie trwa: twojej albo towarzysza. Po wykonaniu akcji menu zostaje na liście akcji, więc kolejny Enter powtarza atak. Polecenia tekstowe działają w walce tak samo jak poza nią.

Wejście na pole z przeciwnikami zaczyna walkę. Bohater, który padnie, budzi się w ostatnio odwiedzonej osadzie z połową zdrowia i traci część pieniędzy.

Każdy kształt potwora walczy inaczej i warto to znać (bestiariusz pokazuje cechę po pierwszym spotkaniu):

| Kształt | Cecha | Co robi |
|---|---|---|
| bestia | Szał | poniżej połowy zdrowia atakuje dwa razy w rundzie |
| nieumarły | Nieumarły | na początku każdej rundy odzyskuje zdrowie |
| humanoid | Tarcza | co trzeci cios bronią odbija i traci od niego o połowę mniej |
| duch | Widmo | broń rani go o połowę słabiej, moce w pełni |
| wąż | Jad | ugryzienie zatruwa na 3 rundy (3 zdrowia na rundę); napar leczniczy zdejmuje truciznę |
| owad | Rój | kąsa dwóch naraz, każdego słabiej |
| maszyna | Pancerz | każdy cios rani go mniej o grubość pancerza |
| szlam | Kwas | jego cios przeżera zbroję: obrona przed nim nie chroni |

Boss ma cechę swojego kształtu i do tego Furię: poniżej jednej trzeciej zdrowia bije o połowę mocniej. Potwory są wyraźnie twardsze niż bohater pierwszego poziomu, więc w głąb mapy lepiej iść z towarzyszem, miksturami i po kilku poziomach. Ciosy widać w scenie: trafiony przeciwnik błyska i odskakuje, nad nim pojawiają się odebrane punkty, błyskawica mocy uderza w cel, fala podmuchu rozchodzi się po polu, a gdy obrywa twoja drużyna, ekran drga i czerwienieje.

Żywy świat
==========

Świat nie stoi w miejscu:

- **Wiedza i plotki.** Gra prowadzi graf faktów: co jest prawdą o miejscach, potworach, skrytkach i ludziach oraz **kto to wie**. Każdy gospodarz zna swoją działkę (zielarka jady i owady, kowal pancerne bestie, mag duchy), o skrytkach wie jeden czy dwóch, a o tym, czego nie wie, mówi wprost i odsyła do sąsiada. Plotki rozchodzą się między gospodarzami z dnia na dzień. Gdy gospodarz powie w rozmowie coś nowego i konkretnego (model to wychwytuje tak jak wymyślone przedmioty), fakt trafia do grafu z jego nazwiskiem jako źródłem, zaczyna krążyć, a ty zapisujesz go w **dzienniku** (`dziennik`). Wydarzenia dopisują własne fakty.
- **Wydarzenia.** Każdy świat z mapą ma scenariusz zdarzeń (`events` w presecie): warunek (wejście do miejsca lub skrytki, zabicie n potworów danego rodzaju, nadejście dnia, rozmowa z gospodarzem, posiadanie rzeczy, flagi) i skutki: zmiana miejsca (opis i teren, czyli spalony las na mapie i w scenie), nowe potwory (tu, w całej krainie albo pod osadą), rzeczy, złoto, fakt, zmiana wszystkich cen, flaga, zwycięstwo; zdarzenie może być jednorazowe, losowe (`chance`) i ogłaszane wszystkim. Wbudowane światy mają po sześć: najazdy pod osady, pożary po zabiciu podpalaczy, skarby w skrytkach, nagrody od gospodarzy za odwagę, przebudzenie bossa po wejściu do jego siedziby, powódź, odwilż, deszcz.
- **Puls dnia.** Potwory wędrują po okolicy i wracają na opustoszałe ziemie; ceny dryfują i rosną po każdym zakupie, a potem wracają do normy; po ósmej godzinie dnia zapada **noc** (scena ciemnieje, w dziczy łatwiej o zasadzkę); na drogach zdarzają się spotkania: wędrowiec z plotką, zgubiona sakiewka, sidła.
- **Sława w osadach.** Każda osada ma o tobie zdanie (od wyklętego przez obcego do bohatera osady; `rozejrzyj się` w osadzie i karta postaci je pokazują). Zakup lub sprzedaż to punkt, obelga w kramie to dwa punkty w dół, każdy potwór ubity w promieniu dwóch pól od palisady to punkt, wykonane zlecenie dwa, wydarzenia mogą dodać swoje (`standing` w presecie). Lubiani płacą mniej (3% za punkt), wyklętym (−5) kupcy zamykają drzwi, a gospodarz w rozmowie wie, za kogo cię mają.
- **Zlecenia z plotek.** Fakt, który gospodarz zna od dwóch dni, może zamienić się w zlecenie: o potworach w polowanie („ubij 3 sztuki”), o miejscu lub skrytce w wyprawę („dotrzyj i wróć”). Zlecenie słyszysz wchodząc do kramu, bierzesz poleceniem `przyjmij`, postęp liczy się sam, nagrodę (złoto, czasem rzecz z półki) odbierasz u zleceniodawcy, a o wykonanej robocie dowiadują się wszyscy gospodarze. Każdy kupiec ma naraz jedno zlecenie.
- **Pory roku i pogoda.** Rok ma cztery pory po dziesięć dni. Co rano losowana jest pogoda: **mgła** odsłania tylko pole, na którym stoisz, i pozwala na zasadzki za dnia; **deszcz** i **śnieżyca** podwajają czas każdego kroku; **upał** zatrzymuje odnawianie mocy w marszu. Zimą równiny, lasy i bagna leżą pod śniegiem (na mapie i w scenie), a ceny osiadają wyżej.
- **Karawana.** Po drogach między osadami jeździ wędrowny kupiec (`$` na mapie, dwa pola dziennie, omija potwory i czeka, gdy zagrodzą drogę). Ma wszystko, co świat w ogóle sprzedaje, także rzeczy najwyższego poziomu, których nie ma w pierwszej osadzie, o 30% drożej; zna plotki ze wszystkich osad, przez które przejeżdżał.
- **Godziny gospodarzy.** Za dnia gospodarz stoi w kramie, wieczorem (od ósmej godziny) siedzi w karczmie: wtedy nie handluje, ale rozmawia chętniej i powtarza, co usłyszał od sąsiadów (to najlepsza pora na plotki), a w głęboką noc (od dziesiątej) osada śpi i zostaje tylko odpoczynek do rana.
- **Świat nie kończy się z bossem.** Po jego upadku ceny spadają, najgroźniejsze ziemie cichną o stopień, każda osada dodaje ci sławy, a najliczniejszy ocalały gatunek ściąga do pustego leża: po trzech dniach ma przywódcę (np. „Goblin Łupieżca Alfa”) i świat znów czeka na ratunek. Cykl powtarza się, a świat z każdym razem robi się spokojniejszy.
- **Towarzysze z własnym zdaniem.** Nikt nie idzie z kimś, kogo w jego osadzie mają za podejrzanego, legenda tylko z kimś znanym (sława ≥ 1). Słaby towarzysz (poziom + stopień poniżej czterech) wprowadzony w śmiertelne strony najpierw ostrzega, a za drugim razem zawraca do karczmy i oddaje twoje rzeczy; towarzysz odchodzi też o świcie, gdy w jego rodzinnej osadzie zostaniesz wyklęty.

Tryb D&D
========

W świecie D&D postać ma te same cechy, poziomy, sprzęt i moce, ale walka idzie po stołowemu:

- **Trafienie**: k20 + biegłość (2, od 5. poziomu 3) + modyfikator Siły przeciw **KP** celu. Naturalna 1 to zawsze pudło, 20 to trafienie krytyczne (podwójne kości). Moce rzuca się k20 + biegłość + Intelekt (pocisk) albo wymuszają rzut obronny Zręczności przeciw ST 8 + biegłość + Intelekt (podmuch: udany rzut to połowa obrażeń).
- **KP** bohatera = 10 + modyfikator Zręczności + pancerz (1/3/5 za tier) + amulet obrony. Potwory: 10 + 2 × tier (maszyna +2, humanoid z tarczą +1, szlam −2, boss +2).
- **Obrażenia**: kość broni k6/k8/k10 (tier 1/2/3, bez broni k4) + Siła; potwory k6+2, k8+3, 2k6+4, boss 2k8+6.
- **Rzuty obronne**: jad węża pozwala na rzut Kondycji (ST 10 + 2 × tier); kwas szlamu omija pancerz (liczy się tylko Zręczność).
- **Inicjatywa**: k20 + Zręczność, losowana raz na walkę.
- **Pole bitwy**: 8 × 3 pola rysowane w scenie jak na stole: twoja drużyna stoi po lewej jako figurki (każdy wbudowany awatar ma własną, towarzysze mają figurki roli), potwory po prawej jako swoje sprite’y, tylny rząd wyżej, pod każdym pasek zdrowia, nad tym, czyja tura, znacznik. Cios to wypad w stronę celu i błysk trafionego, pocisk leci przez pole, podmuch rozchodzi się falą, polegli zostają na polu jako cienie. Atak wręcz wymaga stania obok. Co turę masz ruch (bohater 4 pola, towarzysze 2–4, potwory 1–4 wedle kształtu): akcja «Ruch» podświetla pola w zasięgu, strzałki przesuwają (albo `idź na wschód`). **Atak okazyjny**: kto odchodzi spod wroga, dostaje od niego darmowy cios; ucieczka spod wroga też. Pocisk sięga 5 pól, podmuch wybucha wokół celu (promień 1), leczenie działa wszędzie. Potwory i towarzysze na automacie same podchodzą.

Tryb przygodowy
===============

Przygodówka nie ma walki, drużyny ani sklepów; ma pomieszczenia ułożone na mapie (widać tylko odwiedzone), rzeczy, osoby i scenariusz zapisany jako „kroki”: co się dzieje, gdy użyjesz X na Y, dasz coś komuś, pogadasz, otworzysz albo wejdziesz dokądś, pod jakimi warunkami (flagi) i co to zmienia (otwarte drzwi, nowa rzecz, zmieniony opis albo wygląd pomieszczenia, zwycięstwo). Polecenia: `rozejrzyj się`, `zbadaj X`, `podnieś X`, `upuść X`, `użyj X`, `użyj X na Y`, `otwórz X`, `daj X Y`, `rozmawiaj z Y`; Tab pokazuje rzeczy, Enter na rzeczy ją opisuje. Rzeczy przytwierdzone (łóżko, terminal) bada się, ale nie zabiera. Zapis gry pamięta flagi i zmienione pomieszczenia.

Gra w kilka osób
================

Gospodarz uruchamia grę z flagą `-listen` i wybiera świat:

```
go run ./cmd/neuro -listen :2222
```

Pozostali gracze nie instalują niczego, dołączają zwykłym klientem SSH:

```
ssh -p 2222 IMIĘ@ADRES-GOSPODARZA
```

Dokładne polecenie pojawia się w dzienniku gospodarza po wejściu do świata. Każdy gość wybiera własne imię i awatar. Wszyscy widzą się na mapie; kto wejdzie na pole, na którym trwa walka, dołącza do niej. Runda rozstrzyga się, gdy każdy wybierze akcję albo po 20 sekundach. Pieniądze dzielone są po równo, doświadczenie każdy dostaje w całości, a przedmioty leżą na ziemi dla tego, kto je podniesie. Ponowne połączenie pod tym samym imieniem przywraca postać.

Serwer nie ma uwierzytelniania: do świata dołączy każdy, kto ma dostęp do tego portu, więc otwieraj go tylko w zaufanej sieci. Rozmowy gości z gospodarzami budynków obsługuje model językowy gospodarza gry.

Model językowy
==============

Gra łączy się z dowolnym serwerem zgodnym z API OpenAI (`/chat/completions`), więc ten sam kod obsługuje modele w chmurze i lokalne. Ustawienia można podać jako zmienne środowiskowe albo wpisać do pliku `.env.local`:

| Zmienna | Znaczenie |
|---|---|
| `NEURO_LLM_MODEL` | nazwa modelu; pusta oznacza tryb offline |
| `NEURO_LLM_BASE_URL` | adres API, domyślnie `https://api.openai.com/v1` |
| `NEURO_LLM_API_KEY` | klucz API; lokalne serwery zwykle go nie wymagają |
| `NEURO_LLM_MAX_TOKENS` | limit długości kwestii; przyspiesza małe modele lokalne, np. `220` |
| `NEURO_TTS` | `1` włącza czytanie kwestii na głos; domyślnie wyłączone |
| `NEURO_TTS_VOICE` | nazwa głosu systemowego (macOS `say`); domyślnie polski głos „Zosia”, jeśli jest zainstalowany |
| `NEURO_TTS_COMMAND` | zamiast głosu systemowego: polecenie powłoki, które dostaje kwestię na wejściu i samo ją odtwarza (np. Piper) |
| `NEURO_STT` | `0` wyłącza mikrofon; domyślnie włączony, gdy jest środowisko z `tools/avatar/setup.sh` |
| `NEURO_AVATAR_PYTHON` | interpreter Pythona ze środowiska, w którym jest model obrazkowy; domyślnie ten, który tworzy `tools/avatar/setup.sh` |
| `NEURO_LOG_TALKS` | `1` zapisuje udane rozmowy z modelem do `talks.jsonl` w katalogu danych, w kształcie promptów gry, jako materiał do douczenia własnego modelu |

Lokalny model przez Ollamę:

```
ollama pull gemma3:4b
```

i w pliku `.env.local`:

```
NEURO_LLM_BASE_URL=http://localhost:11434/v1
NEURO_LLM_MODEL=gemma3:4b
NEURO_LLM_MAX_TOKENS=220
```

Ollama musi działać w tle. Gra przy starcie sama prosi lokalny serwer o załadowanie modelu, żeby pierwsza odpowiedź nie czekała na to kilkanaście sekund. Mały model (około 4 miliardów parametrów) wystarcza do rozmów; światy generuje poprawne, ale wyraźnie słabsze niż duży model.

Bez modelu gra działa w trybie offline: wbudowane światy, polecenia, walka, magia, zapis i gra w kilka osób działają normalnie, bo nie korzystają z modelu. Rozmowy w budynkach czyta wtedy prosty parser słów kluczowych, a gospodarze odpowiadają gotowymi kwestiami. Generator światów wymaga modelu.

Jak to działa
=============

Model nigdy nie decyduje o wyniku i niczego nie liczy.

*Generowanie świata*: model zwraca opis świata zgodny ze schematem JSON: nazwy, opisy, rodzaje terenu i kształty przeciwników z zamkniętych list oraz poziomy trudności w skali 1-3. Kod w `internal/world` sprawdza i naprawia ten opis, a potem wylicza z niego mapę, statystyki przeciwników oraz działanie i ceny przedmiotów. Wbudowane światy to takie same opisy, zapisane w `internal/world/presets`.

*Polecenia*: tekst gracza czyta zwykły parser w `internal/game`, bez modelu, więc odpowiedź jest natychmiastowa i zawsze taka sama.

*Rozmowy w budynkach*: każda tura ma trzy kroki.

1. Wypowiedź gracza czytają zwykłe reguły: wyłapują kwoty (cyframi i słownie), liczbę sztuk, nazwy przedmiotów oraz jednoznaczne zwroty, takie jak "biorę" czy "żegnaj". Model jest pytany tylko w dwóch przypadkach: gdy w zdaniu jest liczba, o której nie wiadomo, czy jest ofertą, oraz gdy wypowiedź jest na tyle długa, że warto ocenić siłę argumentu. Kwota i liczba sztuk zawsze pochodzą z tekstu gracza, a model nie może za niego niczego kupić ani zakończyć rozmowy. Zwykła rozmowa nie wymaga czytania przez model, więc odpowiedź przychodzi szybciej.
2. Reguły gry liczą wynik: sprzedaż, kontrofertę, obrazę albo koniec cierpliwości. Cena minimalna nigdy nie trafia do modelu.
3. Model pisze kwestię gospodarza do gotowej decyzji. Dostaje jego charakter oraz to, co gospodarz wie: własny towar z cenami, plotki o miejscach z kierunkami, sąsiadów, zagrożenia, stan i ekwipunek rozmówcy. Kwestia jest czyszczona i przycinana do kilku zdań. Jeśli zawiera kwotę, której reguły nie wyliczyły, albo kwotę zapisaną słownie, model dostaje jedną poprawkę, a gdy i ona zawiedzie, pada gotowa kwestia.

*Przedmioty z rozmowy*: po zwykłej pogawędce model dostaje kwestię gospodarza i listę jego towaru i odpowiada według schematu, czy padła w niej propozycja czegoś nowego i jakiego rodzaju to rzecz (broń, pancerz, mikstura, moc, amulet, z działaniem i poziomem 1-3). Cenę i liczby wylicza kod z poziomu, tak jak dla reszty przedmiotów. Rzeczy o nazwie podobnej do istniejących nie są dodawane. Wymyślone przedmioty trafiają do zapisu gry.

*Wspólny świat*: cały stan gry trzyma jeden obiekt (`game.Realm`), a każdy gracz, lokalny czy połączony przez SSH, ma tylko własny ekran. Dzięki temu gra jednoosobowa i wieloosobowa to ten sam kod.

*Grafika*: widoki miejsc i przeciwnicy to małe obrazki o 256 kolorach w `assets/px`. Gra składa z nich scenę i rysuje ją znakami półbloków, po dwa piksele na znak.

Rysowanie awatarów
==================

Własne awatary rysuje lokalnie mały model obrazkowy SD-Turbo (ok. 2,5 GB), a gra zamienia obraz na pixel art 40x40 w 256 kolorach, taki sam jak wbudowane portrety. Przygotowanie, jednorazowo:

```
tools/avatar/setup.sh
```

Skrypt tworzy środowisko Pythona z biblioteką diffusers w katalogu danych gry i pobiera model. Potem w ekranie wyboru postaci, za ostatnim awatarem, jest pozycja „Dodaj nowy…”: opisujesz postać, model ją rysuje (na Macu z układem Apple kilka sekund, pierwszy raz dłużej, bo model się ładuje), nadajesz nazwę i zapisujesz. Zapisane awatary leżą w podkatalogu `avatars` katalogu danych i są dostępne także gościom przez SSH, choć goście sami nowych nie rysują.

Mowa: głos i mikrofon
=====================

Obie rzeczy są opcjonalne i gra bez nich działa tak samo.

*Czytanie na głos.* Domyślnie gra używa systemowego `say` z macOS (polski głos „Zosia” jest w systemie), a nastrój kwestii, który model podaje w znaczniku (`[kpina] Tyle za rurę?`), zamienia na tempo i wysokość głosu. Można podpiąć dowolną inną syntezę przez `NEURO_TTS_COMMAND`: polecenie dostaje tekst na wejściu, a nastrój w zmiennej `NEURO_TTS_EMOTION`. Dobre otwarte opcje z polskim głosem: [Piper](https://github.com/rhasspy/piper) (szybki, lekki, głosy `pl_PL-gosia-medium` i `pl_PL-darkman-medium`) oraz XTTS v2 z projektu Coqui (ten z żabą; firma zamknęła się w 2024, projekt żyje jako [społecznościowy fork](https://github.com/idiap/coqui-ai-TTS)), który klonuje głos z kilkusekundowej próbki, ale jest wolniejszy. Przykład z Piperem:

```
NEURO_TTS=1
NEURO_TTS_COMMAND=piper -m ~/piper/pl_PL-gosia-medium.onnx -f /tmp/neuro.wav && afplay /tmp/neuro.wav
```

*Mówienie do gry.* F5 włącza nagrywanie, drugie F5 kończy je i rozpoznaje mowę lokalnie modelem Whisper (`mlx-community/whisper-large-v3-turbo`, ok. 1,6 GB, działa na układach Apple). Rozpoznany tekst ląduje w wierszu poleceń albo w rozmowie z gospodarzem, do poprawienia i zatwierdzenia Enterem. Przy pierwszym użyciu macOS zapyta o zgodę na mikrofon dla Terminala. Terminal nie widzi przytrzymania samego Shifta, dlatego mikrofon przełącza się klawiszem, a nie trzyma. Środowisko przygotowuje `tools/avatar/setup.sh`.

Tech demo
=========

`go run ./tools/demo -out /tmp/klatki` nagrywa pokaz gry bez terminala i bez modelu: trzy rodzaje gry po kolei i generowanie świata (świat „wygenerowany” wczytuje się z `tools/demo/world.json`, żeby pokaz nie zależał od modelu). `python3 tools/demo/render.py /tmp/klatki demo.mp4` zamienia klatki w film (potrzebne PIL i ffmpeg).

Dane do własnego modelu
=======================

Trzy źródła danych i jeden skrypt treningu:

1. `NEURO_LOG_TALKS=1` zapisuje każdą dobrą wymianę z gry do `talks.jsonl` w folderze danych gry.
2. `go run ./tools/synth -n 200 -fights 20` gra sam ze sobą: losuje świat, sklep i scenariusz rozmowy (pytania, targi, obelgi, hurt, sprzedaż), zasady gry rozstrzygają jak zwykle, a skonfigurowany model (`.env.local`) ubiera odpowiedzi w słowa; zostają tylko odpowiedzi, które przechodzą kontrole gry. `-fights` dokłada narracje Mistrza Gry z automatycznych walk. Wynik: `synth.jsonl`. Jakość danych jest taka jak modelu, który je generuje: lokalna gemma daje poprawny format, model chmurowy ciekawsze kwestie.
3. `tools/model/PROMPT.md` to prompt do wklejenia w dowolny czat z silnym modelem, gdy chcesz generować dane ręcznie.

Odpowiedzi małego modelu bywają poprawne, ale nijakie; dane można poprawić silniejszym modelem (albo ręcznie) bez ruszania promptów, które są prawdziwymi sytuacjami z gry: `tools/model/digest.py synth.jsonl digest.jsonl` skraca każdą wymianę do charakteru gospodarza, ostatnich kwestii, słów gracza i bloku `[GRA]`; na tej podstawie pisze się nowe kwestie jako linie `{"id": N, "reply": "[nastrój] ..."}`, a `tools/model/apply.py synth.jsonl poprawki.jsonl synth-fixed.jsonl` wstawia je w miejsce starych (kwestie bez znacznika nastroju odrzuca).

`tools/model/train.sh [kroki]` łączy wszystko (`prepare.py` czyści, usuwa duplikaty i dzieli na train/valid/test), trenuje adapter LoRA przez `mlx-lm` na `speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit` (zmienna `NEURO_BASE_MODEL` wybiera inny), wtapia adapter w wagi i rejestruje model w Ollamie jako `neuro-npc` (kwantyzacja q8_0, szablon ChatML); potem wystarczy `NEURO_LLM_MODEL=neuro-npc` w `.env.local`. Jeśli Ollama nie przyjmie wag wprost, skrypt eksportuje je do GGUF i próbuje jeszcze raz. Strata liczona jest tylko z odpowiedzi gospodarza (`--mask-prompt`), nie z reguł w prompcie, a ustawienia (jedna sekwencja naraz, akumulacja gradientu po czterech, przeliczanie aktywacji, 400 kroków, czyli około trzech epok na pięciuset wymianach) mieszczą się w 18 GB pamięci; `NEURO_BATCH`, `NEURO_ACCUMULATE`, `NEURO_LAYERS` i `NEURO_RATE` je zmieniają. Na czas treningu zamknij grę: każda rozmowa ładuje do Ollamy drugi model. Log leży w `model/train.log` w folderze danych.

Pierwszy trening (469 wymian, 400 kroków, ok. 45 minut na M3 Pro) zszedł ze straty walidacyjnej 2,81 do 1,75 (perpleksja testowa 5,9). Uwaga: Ollama importuje te wagi, ale psuje przy tym tokenizer Bielika i model bełkocze, więc do grania służy `tools/model/serve.sh [port]`: podnosi `mlx_lm.server` z bazowym modelem i adapterem na `http://localhost:8080/v1`. Ten serwer nie wymusza schematu JSON, dlatego własny model podpina się jako **drugi głos**, a odczyt intencji, wymyślanie rzeczy, notowanie faktów i generowanie światów zostają przy głównym modelu:

```
NEURO_TALK_BASE_URL=http://localhost:8080/v1
NEURO_TALK_MODEL=speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit
```

(`NEURO_TALK_API_KEY` opcjonalnie). Z ustawionym `NEURO_TALK_MODEL` gra pokazuje w menu oba modele; bez niego jeden model robi wszystko jak dotąd.


Katalog `tools/data` zawiera skrypt `fireball_to_neuro.py`, który zamienia tury ze zbioru FIREBALL (Hugging Face `lara-martin/FIREBALL`, CC-BY-4.0) na rekordy w tym samym kształcie, w jakim gra zapisuje własne rozmowy (`NEURO_LOG_TALKS`): komunikat systemowy z rolą i wiedzą, wypowiedź z blokiem `[GRA]` i kwestia do nauczenia. Oba źródła razem są punktem wyjścia do douczenia małego modelu pod rozmowy w grze.

Dane gry
========

Wygenerowane światy, zapisane gry i klucz serwera SSH leżą w katalogu konfiguracyjnym użytkownika, w podkatalogu `neuro-processor` (na macOS `~/Library/Application Support/neuro-processor`). Zapis jest jeden na świat: kolejne `zapisz` nadpisuje poprzedni.

Technologia i Rozwój
====================

Interfejs korzysta z biblioteki Bubble Tea, a serwer dla kolejnych graczy z biblioteki Wish. Pierwsza wersja projektu, pisana w PHP i Symfony, jest zachowana na gałęzi `php-legacy`. Projekt traktuję jako hobbystyczny eksperyment i formę zabawy, a nie przedsięwzięcie komercyjne.

```
go test ./...
```

Screenshots
===========

Zrzuty ekranu pochodzą z pierwszej wersji projektu.

![forest](./images/screen1.png)
![dragon](./images/screen2.png)
![npc](./images/screen3.png)
