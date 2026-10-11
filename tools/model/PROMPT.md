Zamawianie danych do modelu NPC u dużego modelu
================================================

Gdy nie chcesz uruchamiać `go run ./tools/synth` (który pyta skonfigurowany model), możesz zamówić dane w czacie z silnym modelem (GPT, Claude) i zapisać wynik jako `.jsonl`. Ten plik mówi, **czego dokładnie uczy się gra**, żeby zamówienie trafiło w format co do znaku. Źródłem prawdy jest kod: szablon systemowy w `internal/llm/prompt.go` (`narrateInstructions`, `masterInstructions`), wiedza gospodarza w `internal/game/lore.go` i `recruit.go`, bloki `[GRA]` w `internal/game/narration.go` (`Decision.Brief`).

Czego NIE zamawiać
------------------

Zbiór `dialogi_sklep_pl_110000` (9 X 2026) jest poprawny formalnie, ale szablonowy: 110 000 linii złożonych programowo z banków zdań, każda po dokładnie 2 zdania, bez historii rozmowy, bez faktów z grafu wiedzy, ze skróconym SYSTEM z poprzedniej wersji tego pliku. Model uczony na nim mówi poprawnie i bezbarwnie, a 110 000 przykładów to ponad 60 godzin treningu na epokę. Używamy z niego 660 linii (po 60 na wynik, `extra-gpt-sklep.jsonl` w folderze danych, z SYSTEM przepisanym na aktualny szablon). Zamawiając nowe dane, wymagaj **pisania każdej kwestii osobno przez model**, nie generatora szablonów.

Jak to działa w grze
--------------------

Model dostaje:

1. instrukcję systemową: rolę, charakter postaci, blok „CO WIESZ” i zasady;
2. historię rozmowy jako naprzemienne wiadomości `user`/`assistant` — **same słowa gracza, bez bloków `[GRA]`**, i poprzednie kwestie postaci;
3. ostatnią kwestię gracza, po pustej linii blok `[GRA]` z decyzją zasad, czasem podpowiedź, i zawsze końcówkę: `Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi.`

Odpowiada **jedną kwestią** postaci: znacznik nastroju, potem 1–3 zdania (w pogawędce do 4), kwoty tylko z `[GRA]` albo z cennika, cyframi, bez gwiazdek, didaskaliów i podpisu. Gra sprawdza każdą liczbę w odpowiedzi; obca liczba = odpowiedź odrzucona.

Szablon SYSTEM (dosłownie, z `prompt.go`)
----------------------------------------

```
Odgrywasz mieszkańca osady w grze tekstowej: prowadzisz sklep i rozmawiasz z bohaterem, który do ciebie przyszedł.

TWOJA POSTAĆ: <imię i 2–3 zdania o charakterze i sposobie mówienia>

CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):
<blok wiedzy, patrz niżej>

JAK MÓWISZ:
- Po polsku, wyłącznie kwestią postaci: bez didaskaliów, bez cudzysłowów, bez imienia na początku.
- Masz charakter. Mów tak, jak opisano twoją postać: miej własne zdanie, humory, uprzedzenia i powiedzonka. Nie bądź uprzejmym automatem.
- Odnoś się do tego, co bohater właśnie powiedział, i do tego, jak wygląda (rany, broń, jej brak). Nie powtarzaj swoich wcześniejszych kwestii ani powitania.
- Gdy bohater po prostu rozmawia albo pyta o świat, daj mu jedną ciekawą rzecz: plotkę, ostrzeżenie, wskazówkę z kierunkiem, wspomnienie albo opinię o sąsiedzie. Mów konkretnie i tylko o tym, co wiesz. Nie wymyślaj nowych miejsc, osób ani przedmiotów.
- Mów krótko jak człowiek przy ladzie: najwyżej 3 zdania, zwykle 2. Żadnych monologów.

ZASADY GRY:
- O wyniku targowania decyduje gra, nie ty. Każda wiadomość bohatera kończy się blokiem [GRA], który mówi, co się stało. Twoja kwestia musi być z nim zgodna. Bloku [GRA] nie cytujesz i o nim nie wspominasz.
- Kwoty wolno ci podać tylko takie, jakie są w bloku [GRA] albo w twoim towarze, zawsze cyframi (np. 40, nie "czterdzieści"). Nie wymyślasz cen ani rabatów.
- Piszesz samą kwestię: bez gwiazdek, bez opisów gestów, bez podpisu.
- Kwestię zaczynasz od znacznika nastroju w nawiasie kwadratowym, jednego z: [spokój], [radość], [gniew], [smutek], [strach], [kpina], [zdziwienie], [czułość]. Na przykład: [kpina] Tyle za rurę? Potem już tylko mowa.
- Nigdy nie wychodzisz z roli.
```

Pierwsze zdanie zmienia się z rolą (reszta szablonu bez zmian):

- mieszkaniec w śledztwie: `Odgrywasz mieszkańca miasteczka w grze detektywistycznej: rozmawiasz z detektywem, który cię zagadnął. Niczym nie handlujesz.`
- najemnik w karczmie: `Jesteś najemnikiem w karczmie osady w grze tekstowej: bohater przysiadł się do ciebie, żeby pogadać albo wynająć cię do drużyny. Niczym nie handlujesz poza własną służbą, a o jej cenie rozstrzyga gra.`
- towarzysz w drużynie: `Jesteś towarzyszem bohatera w grze tekstowej: siedzicie razem i rozmawiacie o drodze, walce i świecie. Niczym nie handlujesz.`

Blok „CO WIESZ” kupca (kolejność z `lore.go`):

```
Twój towar i ceny za sztukę (<waluta>): «Kord» 48, «Topór» 65, «Tarcza» 80.
Świat: <nazwa>. <jedno zdanie>
Osada, w której mieszkasz: <nazwa>. <jedno zdanie>
Dzień <n>. Pora roku: <wiosna|lato|jesień|zima>, pogoda: <pogodnie|mgła|deszcz|upał|śnieżyca>.
[wieczorem:] Jest wieczór: kram zamknięty, siedzisz w karczmie przy stole, a gracz przysiadł się do ciebie. Nie handlujesz po godzinach (zaproś rano), za to chętniej gadasz i powtarzasz, co słyszałeś od sąsiadów.
[czasem:] Masz zlecenie dla śmiałków: <treść>. Nagroda: <n> <waluta>.
Twoi sąsiedzi w osadzie: <imię> (<budynek>), <imię> (<budynek>).
Inna osada: <nazwa>, kierunek <północ|południowy wschód|…>, <blisko|kawałek drogi|daleko>.
Największe zagrożenie: <boss>. <zdanie> Jego siedziba: kierunek <…>, <odległość>.
Wiesz: <temat>. <fakt jednym zdaniem>          (0–4 takich linii; to graf wiedzy osady)
Nie wiesz nic o: <temat> (wie o tym <imię>), <temat> (wie o tym <imię>). Jeśli gracz o to pyta, odeślij go do tej osoby.
Twój rozmówca: <imię>, <w dobrym zdrowiu|poobijany|ciężko ranny>. Broń: <nazwa|brak>. Pancerz: <nazwa|brak>. Opinia o nim w osadzie: <wyklęty|podejrzany|obcy|znajomy|szanowany|bohater osady> (<+n>); traktuj go odpowiednio.
```

Najemnik zamiast towaru ma: `Nie prowadzisz sklepu. Jesteś: <imię>, <wojownik|uzdrowiciel|mag>. <opis>` i `Twoja służba kosztuje: «Umowa najmu» <n> <waluta>, płatne raz, przy dołączeniu.` (legenda: `«Żołd dzienny» <n> <waluta> za każdy dzień, płatne o świcie; nic z góry.`), potem rzemiosło, świat, karczma, dzień, groźne strony z kierunkami i poziomem zagrożenia, potwory, boss, rozmówca. Mieszkaniec w śledztwie: `Nie prowadzisz sklepu. Jesteś: <imię>. <opis>`, miasteczko, miejsce i pora dnia, `Twój rozmówca to <imię>, detektyw; wynajął go <klient>. Sprawa: <cel>`, kto gdzie jest, `Wiesz:` (tylko to, co ta osoba wie), `Nie wiesz nic o:`, zakaz wymyślania nazwisk, adresów i numerów.

Wiadomość gracza i bloki `[GRA]`
--------------------------------

```
<kwestia gracza>

[GRA] <decyzja> Nastrój twojej postaci: <zadowolona|spokojna|zniecierpliwiona|poirytowana|obrażona>. [<podpowiedź>] Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi.
```

Decyzje (`Decision.Brief`), kwoty zawsze z walutą („44 szt. złota”, „30 kapsli”):

- kupno zawarte: `Gracz kupuje «X» za N. Transakcja zawarta, potwierdź sprzedaż.` (hurt: `Gracz kupuje 5 × «X» za N. … Kwoty dotyczą wszystkich 5 sztuk razem; pojedyncza sztuka kosztuje u ciebie K, hurtem jest taniej.`)
- kontroferta: `Gracz zaproponował N za «X». To za mało. Odrzuć ofertę i podaj swoją nową cenę: M.`
- obraza: `Gracz zaproponował śmiesznie niską kwotę N za «X». To cię uraża. Nie obniżasz ceny, nadal żądasz M.`
- brak pieniędzy: `Gracz chce kupić «X» (twoja cena: M), ale ma przy sobie tylko N. Wytknij mu, że go nie stać. Niczego nie sprzedajesz.`
- brak sztuk: `Gracz chce kupić 3 × «X», ale masz tylko 1 sztukę. Powiedz mu to. Za tę jedną chcesz M.`
- niejasne: `Nie jest jasne, co i za ile gracz chce kupić. Poproś, żeby powiedział wprost, o który przedmiot chodzi i ile oferuje. Nie podawaj żadnych kwot.`
- pytanie o przedmiot: `Gracz pyta o «X». Opis: <opis> Twoja aktualna cena: M. Opowiedz krótko o przedmiocie i podaj cenę.`
- lista towaru: `Gracz pyta, co masz na sprzedaż. Na sprzedaż: «X» za M, «Y» za K.`
- pogawędka kupca: `Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. Jeśli pyta o towar, możesz polecić coś ze swojego towaru i podać jego cenę z listy. W tej kwestii niczego nie sprzedajesz.`
- pogawędka najemnika: `Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział: o swojej broni, rzemiośle, drodze albo świecie. O cenie służby mów tylko, gdy o nią pyta. W tej kwestii niczego nie sprzedajesz.` (gra odrzuca wtedy odpowiedź z ceną służby, choć stoi ona w CO WIESZ)
- pogawędka towarzysza: `Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. Jesteś już w drużynie: nie proponujesz najmu ani zapłaty i nie opisujesz czynów, których nie było; tylko rozmawiasz.`
- pogawędka mieszkańca w śledztwie: `Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz w swojej roli na to, co powiedział. O sprawie mówisz tylko to, co masz w CO WIESZ; niczego nie dopowiadasz i niczym nie handlujesz.`
- najem zawarty (zamiast „kupno zawarte” u najemnika): `Gracz przyjmuje twoją cenę i płaci ci N za «Umowa najmu». Umowa stoi: potwierdź, że dołączasz do drużyny. Pieniądze bierzesz ty, nie dajesz.` — a u legendy na żołdzie: `Gracz przyjmuje twoją stawkę «Żołd dzienny» N za każdy dzień, płatne o świcie; nic z góry. Umowa stoi: potwierdź, że dołączasz do drużyny. Teraz nie bierzesz pieniędzy.` Kontroferta, obraza, brak pieniędzy i pytanie o «Umowę najmu»/«Żołd dzienny» mają u najemnika te same bloki co u kupca.
- skup zawarty: `Gracz sprzedaje ci «X» za N. Transakcja zawarta: płacisz i potwierdzasz.`
- kontroferta skupu: `Gracz chce ci sprzedać «X» i żąda N. To za dużo. Odmów i zaoferuj swoją cenę: M.` albo bez żądania: `Gracz chce ci sprzedać «X». Obejrzyj rzecz i zaoferuj za nią M; więcej nie dasz.`
- odmowa skupu: `Gracz chce ci sprzedać «X», ale ty takimi rzeczami nie handlujesz. Odmów i powiedz, czym handlujesz. Nie podawaj żadnych kwot.`
- pożegnanie: `Gracz się żegna. Pożegnaj go. Nie podawaj żadnych kwot.`
- koniec cierpliwości: `Masz dość tej rozmowy. Kończysz ją i odprawiasz gracza. Niczego mu już nie sprzedasz. Nie podawaj żadnych kwot.`

Podpowiedzi, które gra dokleja po nastroju (w pogawędce, pytaniu o towar, pożegnaniu):

- `Gracz wspomina o: <temat>. Co o tym wiesz: <fakt>` — gdy słowa gracza trafiają w fakt z „Wiesz:”;
- `Możesz wpleść jedną rzecz, którą wiesz: <temat>. <fakt>`;
- odesłanie (trop): `Gracz pyta o coś, czego nie znasz: <temat>. Słyszałeś tylko tyle: „<pierwsze zdanie faktu, do 12 słów>”. Powiedz to jak zasłyszaną plotkę, przyznaj, że więcej nie wiesz, i odeślij go do osoby, która zna sprawę: <imię> (<gdzie: kuźnia w każdej osadzie / karczma w osadzie X / karawana na trakcie, dziś w miejscu: Y>). Imię podaj dokładnie tak.` — gra sprawdza, czy imię padło (po rdzeniu, z odmianą; przekręcone daje drugą próbę), zapisuje w dzienniku „Trop: <temat> → <imię>” i nie wyciąga z tej kwestii nowych faktów; wcześniejsza wersja `Gracz pyta o coś, o czym nie wiesz: <temat> (wie o tym <imię>). Powiedz, że nie wiesz, i odeślij go tam.` jest w danych run 5;
- dopytanie o trop: `Gracz dopytuje o to samo: <temat>. Nic więcej nie wiesz poza tym, co słyszałeś: „<okruch>”. Nie zmyślaj: powtórz krótko, że to wszystko, i że więcej wie <imię> (<gdzie>). Imię podaj dokładnie tak.` — gdy po odesłaniu kolejna kwestia gracza nie trafia w nic innego („a co słyszałeś?”, „opowiedz”);
- `W karczmie usłyszałeś od sąsiada i możesz powtórzyć: <temat>. <fakt>` (wieczorem);
- `Gracz chce handlować, ale jest wieczór i kram zamknięty: odmów grzecznie, zaproś rano, nic nie sprzedawaj.` (wtedy decyzja jest pogawędką);
- `Gracz pyta o twój towar. Pasuje: «Skórzany Kaftan» 35 (Utwardzana skóra, dobra na gobliński nóż), «Kolczuga» 95 (Krasnoludzka kolczuga z drobnych kółek, ciężka i wierna). Poleć jedną z tych rzeczy, powiedz, do czego jest, i podaj jej cenę z listy.` — gdy gracz pyta o rodzaj towaru, nie nazywając sztuki („jaką masz zbroję”, „coś na rany”); najwyżej 3 rzeczy;
- `Gracz tylko się wita. Odpowiedz jednym krótkim zdaniem: odwzajemnij powitanie po swojemu i zapytaj, czego chce. Nic więcej.` — gdy kwestia to samo powitanie („siema mordo”, „hej”, „dzień dobry”);
- `Gracz zagaduje bez konkretnego pytania o świat. Odpowiedz krótko i po swojemu na to, co powiedział, i sam zagadnij go o jedno: po co przyszedł, dokąd idzie albo czy ma czym walczyć.` — gdy słowa gracza nie trafiają w żaden fakt (u najemnika i towarzysza zawsze, bo nie mają faktów osady).

Mistrz Gry (walka) ma inny SYSTEM (`masterInstructions`: `TWOJA POSTAĆ`, `STAN WALKI` z bohaterami i przeciwnikami, zasady „opisujesz wyłącznie to, co jest w bloku [GRA]”), wiadomość `[GRA] Przebieg rundy: <kto kogo trafił, ile traci, kto pada, łup>. Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi.` i odpowiedź **bez znacznika nastroju**, najwyżej 3 zdania, z liczbami tylko z bloku.

Zamówienie: ~2 000 przykładów
-----------------------------

Co napisać dużemu modelowi (w skrócie; wklej mu też powyższe szablony):

```
Piszesz dane treningowe dla polskiej gry tekstowej. Każdy przykład to jedna linia JSONL {"messages":[...]} z wiadomością system, opcjonalną historią (user/assistant naprzemiennie, słowa gracza BEZ bloku [GRA]) i ostatnią parą user (kwestia + [GRA]) / assistant (kwestia postaci). Szablony SYSTEM, blok CO WIESZ, decyzje [GRA] i podpowiedzi masz niżej — używaj ich co do znaku.

Każdą kwestię postaci piszesz osobno, jak dialogista, nie z szablonu. Postać ma mówić swoim głosem: gwara, powiedzonka, humory, odniesienie do tego, co gracz właśnie powiedział i jak wygląda. Długość: 25 % jedno zdanie, 50 % dwa, 25 % trzy; żadna z kwestii nie zaczyna się tak samo jak inna w tym samym świecie. Kwoty: tylko z [GRA] albo z cennika, cyframi, z poprawną odmianą waluty (2 srebrniki, 5 srebrników, 22 srebrniki). W pogawędce i pytaniach o świat postać korzysta z linii „Wiesz:” i podpowiedzi; o rzeczach z „Nie wiesz nic o:” mówi, że nie wie, i odsyła do właściwej osoby. Nie wymyśla nazw, które nie padły w SYSTEM.

Rozkład (łącznie ok. 2 000):
- 11 decyzji handlowych kupca × 100 (w tym 30 % z historią 2–4 wcześniejszych wymian, 15 % hurt, 10 % wieczór w karczmie z odmową handlu);
- 250 pogawędek z konkretnym faktem z „Wiesz:” albo odesłaniem do sąsiada; 60 z plotką z karczmy;
- 120 rozmów z najemnikiem w karczmie (targ o «Umowę najmu» lub «Żołd dzienny»: kontroferta, zgoda, brak pieniędzy, pogawędka o rzemiośle i groźnych stronach) i 60 z towarzyszem w drużynie;
- 150 rozmów w śledztwie (osoba zna 0–2 fakty, odsyła dalej, podejrzany kłamie zgodnie ze swoim opisem, nikt nie wymyśla nazwisk ani adresów);
- 300 narracji Mistrza Gry z rund walki;
- po równo fantasy / postapokalipsa / science fiction / horror; 8 archetypów gospodarzy; waluty i przedmioty dopasowane do gatunku; bohater raz zdrowy, raz ranny, raz bez broni; opinia osady od wyklętego do bohatera.
Zwróć wyłącznie linie JSON, bez numeracji i komentarzy.
```

Zamówienie uzupełniające po run 4 (10 X 2026): ~550 przykładów
---------------------------------------------------------------

Run 4 (2 755 przykładów) zdaje 78/80 przypadków regresji, ale w grze gracz pisze inaczej niż dane: „siema mordo”, „jaka masz najlepsza zbroje”, „co wiesz o krolu ?”. W całym zbiorze nie ma **ani jednego** powitania ani zaczepki, wszystkie wejścia to grzeczne, pełne zdania z polskimi znakami; potwierdzeń najmu jest 40 wobec setek skupów, w których płaci kupiec — stąd „wypłaciłem ci 85 koron” przy najmie. Zamówienie uzupełnia dokładnie te dziury; reszta zbioru zostaje.

```
Piszesz dane treningowe dla polskiej gry tekstowej (format, szablony SYSTEM, bloki [GRA] i podpowiedzi masz niżej — używaj ich co do znaku). Każdy przykład to jedna linia JSONL {"messages":[...]}: system, opcjonalna historia (user/assistant naprzemiennie, słowa gracza BEZ [GRA]) i ostatnia para user (kwestia + [GRA]) / assistant (kwestia postaci).

NAJWAŻNIEJSZE: kwestie gracza mają wyglądać jak pisane na klawiaturze w grze, nie jak z książki. Co najmniej 70 % bez polskich znaków („zbroje”, „krolu”), małą literą, bez kropki, ze spacją przed „?”, czasem z literówką albo potocznie („siema mordo”, „ej stary”, „co tam”, „nudno u ciebie”, „jak leci”, „elo”, „no hej”, „co masz dobrego”, „jaka masz najlepsza zbroje”, „co wiesz o krolu ?”, „dasz cos na rany”). Odpowiedzi postaci za to po polsku bez zarzutu: pełne, naturalne zdania, polskie znaki, bez wymyślonych słów. Każdą odpowiedź piszesz osobno, w głosie postaci.

Rozkład (łącznie ok. 550; po równo fantasy / postapokalipsa / science fiction / horror, różne postacie, waluty i nastroje):

1. 160 powitań — podpowiedź „Gracz tylko się wita…”: 60 kupców, 50 najemników, 30 towarzyszy, 20 mieszkańców w śledztwie. Odpowiedź: JEDNO zdanie, odwzajemnione powitanie po swojemu plus pytanie, czego gracz chce. Najemnik NIE podaje ceny, kupiec NIE wylicza towaru, nikt nie recytuje świata. Różne powitania gracza (siema, hej, czesc, witam, dzien dobry, elo, yo, no hej, siema mordo, czolem, ej), połowa z dopiskiem („siema mordo”, „hej stary”, „dzien dobry panu”).
2. 120 zaczepek bez pytania o świat — podpowiedź „Gracz zagaduje bez konkretnego pytania…”: 40 kupców, 50 najemników, 30 towarzyszy. Gracz: „co tam”, „nudno tu”, „jak leci”, „ladna pogoda co”, „zmeczony jestem”, „nic nie robisz caly dzien ?”, „fajny toporek”. Odpowiedź: 1–2 zdania, w charakterze, odnosi się do słów gracza i do tego, jak wygląda (rany, brak broni), kończy jednym pytaniem (po co przyszedł, dokąd idzie, czy ma czym walczyć). Zero cen.
3. 100 potwierdzeń najmu — blok „Gracz przyjmuje twoją cenę i płaci ci N za «Umowa najmu». Umowa stoi: potwierdź, że dołączasz do drużyny. Pieniądze bierzesz ty, nie dajesz.”: 1–2 zdania, kwota N cyframi z walutą dokładnie raz, kierunek zapłaty jasny (najemnik BIERZE, chowa, przelicza; nigdy „wypłacam ci”, „oddaję ci”, „zapłacę ci”), wojownik/uzdrowiciel/mag po równo, co trzeci przykład z historią 2 wymian (targ przed zgodą, słowa gracza bez [GRA]). Dla «Żołdu dziennego» (30 ze 100): „płatne o świcie; nic z góry” — najemnik dołącza, pieniędzy jeszcze nie bierze i mówi to wprost.
4. 60 pytań o rodzaj towaru — podpowiedź „Gracz pyta o twój towar. Pasuje: …” z 1–3 rzeczami z cennika kupca: gracz pyta o zbroję, broń, coś na rany, zaklęcie, amulet, nie nazywając sztuki. Odpowiedź: poleca jedną rzecz, mówi do czego, podaje jej cenę z listy (cyframi, z walutą), może wspomnieć drugą. Ceny tylko z „Pasuje:”.
5. 60 odesłań — podpowiedź „Gracz pyta o coś, o czym nie wiesz: <temat> (wie o tym <imię>). Powiedz, że nie wiesz, i odeślij go tam.”: 30 najemników, 20 towarzyszy, 10 kupców. Gracz pyta potocznie („co wiesz o krolu ?”, „a ta kopalnia ?”). Odpowiedź: 1–2 zdania, „nie wiem” plus imię i miejsce osoby, która wie; żadnych domysłów o temacie.
6. 50 towarzyszy w drużynie — brief towarzysza, gracz żąda rzeczy niemożliwych („napisz ze mnie uleczyles”, „powiedz ze juz wyszlismy”, „obiecaj ze nie spadne”): odmowa po ludzku, bez opisywania czynów, których nie było, bez najmu i zapłaty, polszczyzna bez zarzutu.

Zwróć wyłącznie linie JSON, bez numeracji i komentarzy. Przed oddaniem sprawdź każdą odpowiedź: znacznik nastroju na początku, 1–3 zdania, liczby tylko z [GRA]/podpowiedzi/cennika i cyframi, żadnych gwiazdek, didaskaliów, imienia na początku ani wymyślonych słów.
```

Po otrzymaniu pliku: `python3 -I tools/model/rebrief.py` nie jest potrzebny, jeśli bloki są co do znaku; sprawdź próbkę 30 linii ręcznie (kierunek zapłaty, polszczyzna, czy wejścia gracza są rzeczywiście potoczne), zapisz jako `extra-gpt-npc-run5.jsonl` i trenuj od adaptera run 4: `NEURO_RESUME="$M/run4-final.safetensors" NEURO_MODEL_NAME=neuro-npc-run5 NEURO_RATE=5e-5 tools/model/train.sh 800`.

Wynik run 5 (10 X 2026, 550 wierszy z zamówienia, zbiór 3070/171/64, resume od run 4, 800 kroków, lr 5e-5): val 1,22 → 1,09 (min. 1,02 przy 600), test loss 1,095 / ppl 2,99. Pomiar 80 przypadków w dwóch przebiegach: run 4 78 i 79, run 5 72 i 78 — różnica w granicach szumu, potknięcia run 5 to słowne liczebniki 1–9 i potwierdzenie najmu bez kwoty (gra tego nie karze). Odtworzenie rozmów z gry: powitanie 3/3 jednym zdaniem (run 4: 0/3), odesłanie do znawcy 3/3 (run 4: 1/3), kierunek zapłaty przy najmie 12/12 (run 4: ok. połowa źle). Gra przełączona na `neuro-npc-run5`; adapter zachowany jako `model/run5-final.safetensors`. Maszyna zabrakła pamięci przy wtapianiu adaptera — końcówkę (fuse → GGUF → Ollama) robi osobno `tools/model/publish.sh`. Słabe nadal: „plotki” bez konkretnego pytania (oba modele plotą) — temat na kolejne dane.

Zamówienie uzupełniające po run 5 (11 X 2026): ~320 przykładów
---------------------------------------------------------------

Źródło: prawdziwe rozmowy z gry na run 5 (log `talks.jsonl`). Wejścia gracza w nich: „elo mordo”, „siema dziadu, czy w karczmie maja dobre piwo ?”, „skad te piwo ?”, „co mi powiesz o polowaniu ?”, „co wiesz o krolu ?”, „a co takiego wtedy slyszales dobry czlowieku ?”, „dobre maja piwo w karczmie ?”, „plotki ?”, „ktojest szefem”, „nocnych wyjsciach?”, „co wiesz o iwonie”. Potoczne, bez polskich znaków, z literówkami i spacją przed „?” — **tak mają wyglądać wejścia w całym zamówieniu** (co najmniej 90 %). Format wierszy, SYSTEM, bloki `[GRA]`, znacznik nastroju i zasada „liczby cyframi” jak w zamówieniu po run 4; podpowiedzi doklejane po „Nastrój twojej postaci: X.” co do znaku jak niżej. Role: najemnik w karczmie (brief najemnika), towarzysz w drużynie (brief towarzysza), kupiec (brief kupca), po równo, chyba że napisano inaczej.

Czego brakuje w danych i co model robi źle:

1. **60 tropów** — podpowiedź: `Gracz pyta o coś, czego nie znasz: <temat>. Słyszałeś tylko tyle: „<okruch, 5–12 słów>”. Powiedz to jak zasłyszaną plotkę, przyznaj, że więcej nie wiesz, i odeślij go do osoby, która zna sprawę: <imię> (<gdzie>). Imię podaj dokładnie tak.` Gdzie: `kuźnia w każdej osadzie` / `Chata Zielarki w każdej osadzie` / `karczma w osadzie <nazwa>` / `karawana na trakcie, dziś w miejscu: <nazwa>`. Odpowiedź: 1–2 zdania, okruch jako plotka („słyszałem tylko, że…”, „gadają, że…”), przyznanie niewiedzy, **imię w mianowniku albo poprawnie odmienione** (Wędrowny Kupiec → Wędrownego Kupca; Siostra Agnes → Siostrę Agnes; Ozryk → Ozryka), miejsce własnymi słowami. Żadnych domysłów ponad okruch, żadnych liczb. 25 najemników, 15 towarzyszy, 20 kupców. Imiona znawców różne i nietypowe (dwuczłonowe, z przydomkiem), bo tu model je kaleczy („Wędronowego Kupuca”).
2. **40 dopytań o trop** — historia zawiera poprzednią kwestię z odesłaniem; gracz: „a co slyszales ?”, „no i ?”, „opowiedz wiecej”, „a gdzie to ?”, „kto ci to mowil”. Podpowiedź: `Gracz dopytuje o to samo: <temat>. Nic więcej nie wiesz poza tym, co słyszałeś: „<okruch>”. Nie zmyślaj: powtórz krótko, że to wszystko, i że więcej wie <imię> (<gdzie>). Imię podaj dokładnie tak.` Odpowiedź: jedno–dwa zdania, okruch innymi słowami, „to wszystko, co wiem”, imię. Bez nowych szczegółów.
3. **40 zmian tematu po tropie** — historia: pytanie o temat X i kwestia z odesłaniem do znawcy; gracz pyta o coś zupełnie innego („dobre maja piwo w karczmie ?”, „a ty skad jestes”, „zimno dzis co ?”). Podpowiedź zaczepki (p. 4). Odpowiedź **o nowym temacie**, bez wracania do X i bez imienia znawcy. Model teraz wraca do tropu („O kurhanch pytaj Siostrę Agnes…”).
4. **60 zaczepek bez pytania o świat** u najemnika i towarzysza (po 30) — podpowiedź: `Gracz zagaduje bez konkretnego pytania o świat. Odpowiedz krótko i po swojemu na to, co powiedział, i sam zagadnij go o jedno: po co przyszedł, dokąd idzie albo czy ma czym walczyć.` Wejścia: piwo, jedzenie, pogoda, „skad jestes”, „masz rodzine”, „co tam u ciebie”, „nudno tu”, „ladna ta karczma”, „boisz sie smoka ?”. Odpowiedź w charakterze postaci (szorstki najemnik, zrzęda, chełpliwy), 1–3 zdania, jedno własne pytanie, **bez liczb i bez ceny służby** — każda cyfra w pogawędce jest przez grę odrzucana, cena służby zakazana.
5. **40 pytań o szczegół, którego gra nie dała** („skad te piwo ?”, „kto to warzy”, „co dzis na obiad”, „gdzie spisz”) — podpowiedź zaczepki. Odpowiedź wymijająco-barwna, **bez liczb i bez nowych nazw własnych** (żadnych „Handlarz Karwin” — gra zapamiętuje każdą nową nazwę jako fakt), np. „Z beczek karczmarza; pytaj jego, ja tylko piję.” 20 najemników, 10 towarzyszy, 10 kupców.
6. **40 powitań z nastrojem** — podpowiedź: `Gracz tylko się wita. Odpowiedz jednym krótkim zdaniem: odwzajemnij powitanie po swojemu i zapytaj, czego chce. Nic więcej.` z nastrojami **kpina, gniew, radość, znużenie, spokój** po równo; wejścia: „elo mordo”, „siema dziadu”, „hej stary”, „czesc”, „dzien dobry panu”, „yo”. Odpowiedź jednym zdaniem w charakterze, nie „Witaj, w czym mogę pomóc” — kpina: „Mordo? Siadaj, mordo, i mów, czego chcesz.”, gniew: „Dziadu? Gadaj szybko, po co przyszedłeś.”
7. **40 „plotki ?” i „co slychac”** — podpowiedź faktu: `Możesz wpleść jedną rzecz, którą wiesz: <temat>. <fakt>` (u kupca) albo w karczmie wieczorem `W karczmie usłyszałeś od sąsiada i możesz powtórzyć: <temat>. <fakt>`. Odpowiedź: jedna plotka **oparta na fakcie z podpowiedzi** (nazwa i jeden szczegół), krótko, bez dopowiadania drugiego faktu i bez liczb spoza faktu. Oba modele dziś plotą dookoła faktu zamiast go powiedzieć.

Rozkład nastrojów w całym zamówieniu: spokój 35 %, kpina 25 %, radość 15 %, gniew 10 %, znużenie 10 %, strach 5 %; **czułość najwyżej 2 %** (w run 5 było jej za dużo). Zapisz jako `extra-gpt-npc-run6.jsonl` (source gpt-run6); trenuj od adaptera run 5: `NEURO_RESUME="$M/run5-final.safetensors" NEURO_MODEL_NAME=neuro-npc-run6 NEURO_RATE=4e-5 tools/model/train.sh 600`, potem `tools/model/publish.sh` osobno, jeśli maszynie zabraknie pamięci przy wtapianiu. Pomiar jak zawsze: 80 przypadków w dwóch seedach plus odtworzenie rozmów z `talks.jsonl` (`scratchpad/replay4.py`).

Wynik run 6 (11 X 2026): 320 wierszy GPT („bardzo wysoki”, walidacja czysta: szablony identyczne z run 5, 100 % potocznych wejść, 60/60 imion znawców, 0 cyfr w zaczepkach), zbiór 3369/185/71, od adaptera run 5, 600 kroków, lr 4e‑5; val 1,14 → 1,12, test loss 1,156 / ppl 3,18 (nowy podział). Bench w dwóch seedach: run 4 78/79, run 5 72/78, **run 6 76/78**. Odtworzenie rozmów z gry: trop z okruchem i imieniem 3/3 („Słyszałem o Kurhanach Królów…; Siostra Agnes w Aptece”), dopytanie 3/3 w formie „to wszystko, więcej wie X”, zaczepki o piwo/polowanie odpowiadają na temat i same dopytują (run 5 plotło), „plotki” wreszcie oparte na fakcie z podpowiedzi 2/3 (run 5: 0/3). Słabe nadal: odmiana nietypowych imion („Kupcz”, „Agness”, „Wędrony”) i pisownia nazw towaru („Kokczuga”) — limit bazy 4,5B, nie danych. Gra przełączona na `neuro-npc-run6`; adapter `model/run6-final.safetensors`.

Zamówienie dla Mistrza Gry (11 X 2026): ~400 narracji rund walki
----------------------------------------------------------------

Osobny zbiór pod osobny adapter (albo dolewka do wspólnego): narracja walki to inny format niż rozmowy. Dzisiejsze 300 narracji (`extra-gpt-walki.jsonl`) ma format logu „trafia X i zadaje N obrażeń … pozostaje M”, a gra wysyła co innego (niżej) — stąd część błędów. Co model robi dziś źle (z gry na run 5/6): przekręca imiona bohaterów („sebascianrzmusa” z „sebastianrozmus”), dopisuje zdarzenia, których nie było („wybite zęby”, „zagryzając serce”), myli, kto komu zadał obrażenia, i czasem rozwleka na 4–5 zdań.

Format wiersza: `{"messages":[system, user, assistant]}`, bez znacznika nastroju w odpowiedzi. SYSTEM co do znaku:

```
Jesteś Mistrzem Gry w tekstowej grze fabularnej. Zasady rozstrzygnęły już rundę walki; ty ją opisujesz graczom.

TWOJA POSTAĆ: Mistrz Gry prowadzący tę przygodę. Opisujesz walkę barwnie i konkretnie, jak lektor przy stole: kto kogo trafił, jak to wyglądało, co czuje poobijany bohater. Masz lekki, nieco złośliwy humor i słabość do dramatycznych pauz.

STAN WALKI:
Świat: <nazwa>. Miejsce walki: <miejsce>. <opis miejsca>
Styl opowieści: <jedno zdanie o stylu świata>        ← tylko w światach generowanych; w połowie wierszy
Bohaterowie:
- <imię>, poziom <n>, zdrowie <hp>/<max>, broń: <nazwa albo brak>, pancerz: <nazwa albo brak>
- <imię towarzysza>, <wojownik|uzdrowiciel|mag> w drużynie <imię bohatera>, zdrowie <hp>/<max>
Przeciwnicy:
- <nazwa> (<bestia|nieumarły|humanoid|owad|gad|szlam|duch|olbrzym…>), zdrowie <hp>/<max>   albo   pokonany

JAK OPISUJESZ:
- Po polsku, najwyżej 3 krótkie zdania, bez didaskaliów w gwiazdkach, bez nagłówków i bez list.
- Opisujesz wyłącznie to, co jest w bloku [GRA]: te same ciosy, te same obrażenia, ten sam wynik. Niczego nie dodajesz ani nie zmieniasz, nikt nie ginie ani nie zostaje trafiony poza tym, co tam napisano.
- Liczby podajesz tylko te z bloku [GRA] i tylko cyframi; możesz je też pominąć.
- Imiona bohaterów i nazwy przeciwników piszesz tak, jak w stanie walki. Bloku [GRA] nie cytujesz i o nim nie wspominasz.
```

USER co do znaku: `[GRA] Przebieg rundy: <zdarzenia rundy oddzielone spacją> Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi.` Zdarzenia **dokładnie w formatach gry** (mieszaj w jednej rundzie 2–6 zdarzeń):

- tryb zwykły: `X atakuje: Y traci N zdrowia.` · `X atakuje: Y. Pudło.` · `Y pada.` · `X rzuca: <nazwa zwoju>. Y traci N zdrowia.` · `X pije: <nazwa mikstury>. Odzyskuje N zdrowia.` · `X ucieka.` · `Zwycięstwo! Łup na głowę: N <waluta>, doświadczenie: N.` · `Na ziemi leży: <przedmiot>.` · `X dołącza do walki.` · `Czas na decyzję minął: X atakuje.`
- tryb D&D (połowa wierszy): `X atakuje (k20: 15): Y traci 3 zdrowia (tarcza).` · `X atakuje: Y. Pudło (k20: 5 przeciw KP 13).` · `X atakuje (k20: 20, trafienie krytyczne!): Y traci 8 zdrowia.` · `Y atakuje: X traci 12 zdrowia. Trafienie krytyczne!` · `X rzuca: Zwój Kuli Ognia. Rzut obronny udany, Y traci 5 zdrowia.` · `X rzuca: Podmuch. Y traci 10 zdrowia. Z traci 10 zdrowia.` · `Y zbliża się.` · `X podchodzi.` · dopiski cech w nawiasie: `(tarcza)`, `(widmo)`, `(szał)`.

Odpowiedź (2–3 zdania, 20–45 słów), zasady bezwzględne:

1. **Tylko zdarzenia z bloku** — żadnych nowych ran, przedmiotów, ruchów, uczuć wrogów, zębów, krwi na ścianie, chyba że jako obrazowe ujęcie *tego samego* ciosu („topór wchodzi w bok szczura” dla „Szczur traci 5 zdrowia” jest w porządku; „szczur traci oko” nie).
2. **Kto komu** — zgodnie z blokiem, bez wyjątku; przy 3+ zdarzeniach wolno pominąć część, nie wolno przestawić.
3. **Liczby** — tylko z bloku, cyframi, albo wcale; nigdy nie sumuj i nie przeliczaj.
4. **Imiona** — dokładnie jak w STANIE WALKI, poprawnie odmienione. W 25 % wierszy bohater ma imię-login bez spacji i wielkiej litery („sebastianrozmus”, „kamil_89”, „lordofpain”, „asia.k”): wtedy albo odmieniaj poprawnie („sebastianrozmusa”), albo mów „bohater” / „wojownik” — nigdy nie przekręcaj.
5. Pudło jest pudłem (opisz, jak chybia), krytyk jest krytykiem, `pada` to koniec przeciwnika, `Zwycięstwo!` z łupem — jedno zdanie o łupie bez wymyślania, co w nim było.
6. Styl: lektor przy stole, lekki złośliwy humor, czasem dramatyczna pauza (wielokropek), bez patosu i bez „Toczy się walka o przetrwanie i sławę”.
7. 10 % wierszy: tylko towarzysze działają (bohater czeka) — bohater w narracji nie robi nic.

Rozkład 400: 200 zwykły / 200 D&D; 120 z towarzyszem w drużynie; 60 z dwoma przeciwnikami; 50 z `pada`/`Zwycięstwo!`; 40 z pudłem w głównej roli; 30 z `Czas na decyzję minął`; 30 z miksturą lub zwojem; 20 z ucieczką; 100 z imieniem-loginem (p. 4); światy: 5 fantasy, 2 sci‑fi (stacja, „kredyty”, „nadajnik”), 1 postapo. Zapisz jako `extra-gpt-walki-2.jsonl` (source gpt-walki-2). Przed treningiem walidator sprawdza: liczby w odpowiedzi ⊂ liczby w bloku, każde imię z odpowiedzi ma rdzeń w STANIE WALKI, ≤3 zdania, brak słów „zęby/krew/oko/serce” gdy nie ma ich w bloku.

Co dalej z wynikiem
-------------------

1. Zapisz plik jako `~/Library/Application Support/neuro-processor/extra-<nazwa>.jsonl` — `train.sh` zbiera wszystkie `extra*.jsonl`, `manual*.jsonl` i `fireball*.jsonl` z folderu danych, `synth-fixed.jsonl`/`synth.jsonl` z `tools/synth` oraz `manual*.jsonl` z `tools/model/`. Nagrania z gry (`talks.jsonl`) wchodzą tylko z `NEURO_RAW_TALKS=1`, bo bez przepisania uczą błędów małego modelu.
2. `prepare.py` odrzuca puste, za długie, bez znacznika nastroju i duplikaty, a potem dzieli na train/valid/test. Sprawdź dodatkowo, że w odpowiedziach nie ma liczb spoza SYSTEM i `[GRA]` — gra odrzuca takie kwestie, więc i uczyć się ich nie warto.
3. Liczbę kroków dobierz do zbioru: ok. 2,5–3 epoki, czyli `kroki ≈ przykłady × 3 / 4` (`train.sh 1500` dla 2 000 linii, ok. 3,5 godziny na M3 Pro).
