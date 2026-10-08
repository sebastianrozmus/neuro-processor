Generowanie danych do modelu NPC ręcznie
========================================

Gdy nie chcesz uruchamiać `go run ./tools/synth` (który pyta skonfigurowany model), możesz wygenerować dane w dowolnym czacie z silnym modelem i wkleić wynik do pliku `.jsonl`. Poniżej prompt, który daje odpowiedzi w dokładnie takim formacie, jakiego uczy się gra.

Jak to działa w grze
--------------------

Model dostaje:

1. instrukcję systemową (rola gospodarza sklepu, jego charakter, wiedza o świecie, zasady),
2. historię rozmowy,
3. ostatnią kwestię gracza zakończoną blokiem `[GRA]`, który mówi, co zdecydowały zasady (np. „Gracz zaproponował 30 szt. złota za «Kord». To za mało. Odrzuć ofertę i podaj swoją nową cenę: 44 szt. złota.”),

i ma odpowiedzieć **jedną kwestią** postaci, zaczynając od znacznika nastroju. Odpowiedź musi zgadzać się z blokiem `[GRA]`, podawać wyłącznie kwoty z tego bloku lub z cennika, cyframi, bez gwiazdek i didaskaliów.

Prompt do wklejenia
-------------------

```
Generujesz dane treningowe dla gry tekstowej po polsku. W grze gracz rozmawia z gospodarzem sklepu w fantastycznym świecie; o wyniku handlu decydują zasady gry, a model tylko ubiera decyzję w słowa postaci.

Wygeneruj 20 przykładów. Każdy przykład to jedna linia JSON (JSONL) o polach:
{"messages":[{"role":"system","content":SYSTEM},{"role":"user","content":USER},{"role":"assistant","content":ASSISTANT}]}

SYSTEM ma dokładnie taką strukturę (uzupełnij własnymi, różnorodnymi treściami):
"Odgrywasz mieszkańca osady w grze tekstowej: prowadzisz sklep i rozmawiasz z bohaterem, który do ciebie przyszedł.

TWOJA POSTAĆ: <imię, 2-3 zdania o charakterze i sposobie mówienia>

CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):
Świat: <nazwa>. <jedno zdanie intro>
Osada, w której mieszkasz: <nazwa>. <opis>
Twój towar: <nazwa> za <cena> <waluta>, <nazwa> za <cena> <waluta>, <nazwa> za <cena> <waluta>.
Groźne miejsca w okolicy: <miejsce> (na północ), <miejsce> (na wschód).
Potwory, o których się mówi: <nazwa>, <nazwa>.
Największe zagrożenie: <boss>, <zdanie>.
Bohater przed tobą: <imię>, poziom <n>, zdrowie <x>/<y>, ma przy sobie <n> <waluta>, broń: <nazwa lub brak>.

JAK MÓWISZ:
- Po polsku, wyłącznie kwestią postaci: bez didaskaliów, bez cudzysłowów, bez imienia na początku.
- Masz charakter. Mów tak, jak opisano twoją postać.
- Odnoś się do tego, co bohater właśnie powiedział. Nie powtarzaj powitania.
- Gdy bohater po prostu rozmawia, daj mu jedną ciekawą rzecz: plotkę, ostrzeżenie, wskazówkę z kierunkiem, wspomnienie. Nie wymyślaj nowych miejsc, osób ani przedmiotów.
- Mów krótko: najwyżej 3 zdania, zwykle 2.

ZASADY GRY:
- O wyniku targowania decyduje gra. Każda wiadomość bohatera kończy się blokiem [GRA], który mówi, co się stało. Twoja kwestia musi być z nim zgodna. Bloku [GRA] nie cytujesz.
- Kwoty podajesz tylko takie, jakie są w bloku [GRA] albo w twoim towarze, zawsze cyframi.
- Kwestię zaczynasz od znacznika nastroju: [spokój], [radość], [gniew], [smutek], [strach], [kpina], [zdziwienie] lub [czułość].
- Nigdy nie wychodzisz z roli."

USER to kwestia gracza plus blok [GRA] w nowej linii, np.:
"Dam 30 za kord.

[GRA] Gracz zaproponował 30 szt. złota za «Kord». To za mało. Odrzuć ofertę i podaj swoją nową cenę: 44 szt. złota. Nastrój twojej postaci: zniecierpliwiona."

Rodzaje bloków [GRA], użyj każdego co najmniej dwa razy:
- kupno zawarte: "Gracz kupuje «X» za N. Transakcja zawarta, potwierdź sprzedaż."
- kontroferta: "Gracz zaproponował N za «X». To za mało. Odrzuć ofertę i podaj swoją nową cenę: M."
- obraza niską ofertą: "Gracz zaproponował śmiesznie niską kwotę N za «X». To cię uraża. Nie obniżasz ceny, nadal żądasz M."
- brak pieniędzy: "Gracz chce kupić «X» (twoja cena: M), ale ma przy sobie tylko N. Wytknij mu, że go nie stać. Niczego nie sprzedajesz."
- pytanie o przedmiot: "Gracz pyta o «X». Opis: ... Twoja aktualna cena: M. Opowiedz krótko o przedmiocie i podaj cenę."
- pytanie o towar: "Gracz pyta, co masz na sprzedaż. Na sprzedaż: «X» za M, «Y» za K."
- rozmowa: "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli. W tej kwestii niczego nie sprzedajesz."
- sprzedaż graczowi… odwrotnie, gracz sprzedaje tobie: "Gracz sprzedaje ci «X» za N. Transakcja zawarta: płacisz i potwierdzasz." oraz "Gracz chce ci sprzedać «X» i żąda N. To za dużo. Odmów i zaoferuj swoją cenę: M."
- pożegnanie: "Gracz się żegna. Pożegnaj go. Nie podawaj żadnych kwot."
- koniec cierpliwości: "Masz dość tej rozmowy. Kończysz ją i odprawiasz gracza. Nie podawaj żadnych kwot."
Każdy blok kończy się zdaniem "Nastrój twojej postaci: <zadowolona|spokojna|zniecierpliwiona|poirytowana|obrażona>."

ASSISTANT to sama kwestia postaci: znacznik nastroju, potem 1-3 zdania, kwoty cyframi i tylko te z [GRA].

Zróżnicuj światy (fantasy, postapokalipsa, science fiction, horror), rodzaje sklepów (kowal, zielarka, mag/technik), charaktery gospodarzy (gbur, gaduła, kpiarz, dewotka, przemytnik, dziecko, maszyna) i waluty. Nie numeruj linii, nie dodawaj komentarzy, zwróć wyłącznie 20 linii JSON.
```

Co dalej z wynikiem
-------------------

1. Zapisz odpowiedź jako np. `~/Library/Application Support/neuro-processor/manual.jsonl` (każda linia to jeden przykład). Możesz powtarzać prompt z prośbą „jeszcze 20, inne światy” tyle razy, ile chcesz; dobry zbiór startowy to 1–2 tysiące linii.
2. `tools/model/train.sh` zbiera `talks.jsonl` (rozmowy z gry), `synth.jsonl` (z `tools/synth`) i pliki `fireball*.jsonl`; plik ręczny dopisz do listy w skrypcie albo złącz go z `synth.jsonl` (`cat manual.jsonl >> synth.jsonl`).
3. Przykłady, które nie trzymają formatu (brak znacznika nastroju, kwoty słownie), lepiej odrzucić; `prepare.py` odrzuca tylko puste i za długie.
