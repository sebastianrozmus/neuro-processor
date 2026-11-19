# Jak to działa

Model nigdy nie decyduje o wyniku i niczego nie liczy. Robi tylko dwie rzeczy: wymyśla treść i ubiera w słowa decyzje gry.

## Światy

Model oddaje projekt świata zgodny ze schematem JSON: nazwy i opisy, a teren, kształty potworów i poziom trudności z zamkniętych list. `internal/world` sprawdza projekt i naprawia, co się da, a potem wylicza mapę, statystyki, działanie przedmiotów i ceny. Projekt odrzucony wraca do modelu raz, z powodem („case.answer nie jest name żadnej osoby z people”). Wbudowane światy to takie same pliki w `internal/world/presets`.

Przed projektem model pisze **kartę stylu**: rozpoznaje inspirację z opisu gracza („w stylu Cowboy Bebop”) i spisuje estetykę, ton rozmów, typy postaci i hasła do obrazków. Karta trafia potem do każdego promptu, więc styl słychać w rozmowach, nie tylko w nazwach.

Śledztwo przerasta mały model w jednym pytaniu, bo gemma 4b oddawała trzy pokoje i żadnej sprawy. Dlatego powstaje w dwóch krokach: najpierw ludzie i sprawa, potem rzeczy, zagadki i zdarzenia.

`go run ./tools/worldgen <rodzaj> "<opis>"` generuje świat z wiersza poleceń, bez ekranów.

## Tura rozmowy

1. **Reguły czytają gracza.** Kwoty (cyframi i słownie), liczbę sztuk, nazwy rzeczy i jasne zwroty („biorę”, „żegnaj”) wyłapuje zwykły kod w `internal/game`. Model pytany jest tylko wtedy, gdy nie wiadomo, czy liczba jest ofertą, albo gdy trzeba ocenić siłę długiego argumentu.
2. **Reguły liczą wynik**: sprzedaż, kontrofertę, obrazę, koniec cierpliwości. Cena minimalna nigdy nie trafia do modelu.
3. **Model pisze kwestię** do gotowej decyzji. Dostaje charakter postaci, jej wiedzę i blok `[GRA]`, który mówi, co się właśnie stało.

**Kontrole.** Kwestia z kwotą, której reguły nie wyliczyły, z liczbą zapisaną słownie, z przekręconym imieniem albo bez imienia, które gra obiecała graczowi, jest odrzucana z powodem. Model dostaje jedną poprawkę, potem pada gotowa kwestia. Odrzucenia widać w podglądzie.

**Wiedza.** Gra trzyma graf faktów i tego, kto je zna. Nowy fakt z rozmowy wchodzi do grafu tylko wtedy, gdy model czytający potrafi go zacytować z kwestii. Tak samo trafiają do biografii fakty, które postać mówi o sobie.

**Mistrz Gry** opisuje rundę walki w tle, z gotowego wyniku. Długie rundy dostaje w kawałkach po najwyżej 3 zdarzenia, bo przy pięciu naraz modele gubiły ciosy trzy razy częściej.

## Wspólny świat

Cały stan trzyma jeden obiekt (`game.Realm`), a każdy gracz, lokalny czy przez SSH, ma tylko własny ekran. Gra jedno- i wieloosobowa to ten sam kod. Grafika to małe obrazki w `assets/px`, składane w scenę i rysowane znakami blokowymi.

## Model `neuro-npc`

Grę obsługuje jeden model: Bielik 11B v3.0 ([SpeakLeash](https://huggingface.co/speakleash)) douczony pod tę grę (QLoRA SFT w dokładnym formacie promptów gry, potem DPO). Mówi za postaci, opisuje walki jako Mistrz Gry i wykonuje zadania Umysłu (JSON). Wydany model z wynikami ślepych ocen jest na [Hugging Face](https://huggingface.co/rozmus/neuro-npc-bielik-11b-v3-GGUF). Każdą nową wersję porównuje się z poprzednią ślepą oceną sędziów na zestawach, których model nie widział, według reguły spisanej przed oceną.

| Ścieżka | Co tam jest |
|---|---|
| `tools/model/HISTORIA.md` | jak powstawał model: co zadziałało, co odrzucono, jak powtórzyć (też jako strona `docs/index.html`) |
| `tools/model/MODELE.md` | karta każdej wersji: dane, ustawienia, wyniki |
| `tools/model/PROMPT.md` | kontrakt promptu i jak zamawiać dane |
| `tools/pc/` | trening na PC z kartą NVIDIA (WSL2 + Unsloth, 11B na 16 GB) |
| `tools/model/` | trening na Macu (MLX), eksport do GGUF i Ollamy |
| `tools/bench/`, `tools/eval/` | bench, odtwarzanie prawdziwych rozmów, walidator danych |
| `tools/synth/` | gra sama ze sobą i zbiera rozmowy, które przechodzą kontrole |
| `tools/hf/` | karta modelu i publikacja na Hugging Face |
| `tools/data/` | konwersja zbioru FIREBALL (CC-BY-4.0) do formatu gry |

Dane do treningu dają też sama gra (`NEURO_LOG_TALKS=1`) i podgląd, w którym oznaczasz złe kwestie z uwagą (`feedback.jsonl`).

## Pokaz

`go run ./tools/demo -out klatki` nagrywa pokaz gry bez terminala i bez modelu, a `python3 tools/demo/render.py klatki demo.mp4` robi z niego film (potrzebne PIL i ffmpeg; część z podglądem nagrywa bezgłowy Chrome).
