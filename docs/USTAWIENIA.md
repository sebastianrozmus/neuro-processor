# Ustawienia

Ustawienia czyta się ze zmiennych środowiskowych albo z pliku `.env.local`. Gra szuka go najpierw w katalogu, z którego ją uruchamiasz, potem w katalogu danych gry: `~/Library/Application Support/neuro-processor` (macOS), `~/.config/neuro-processor` (Linux), `%AppData%\neuro-processor` (Windows). W tym samym katalogu leżą światy, zapisy i klucz serwera SSH.

## Modele

Gra rozmawia z każdym serwerem zgodnym z API OpenAI, lokalnym albo w chmurze. Najprościej użyć Ollamy z dwoma modelami: gemma rozumie gracza i generuje światy, a `neuro-npc` mówi za postaci (przepis w [README](../README.pl.md#jak-zacząć)). Ollama musi działać w tle, a gra sama ładuje modele przy starcie.

W grze, w **Esc → Modele językowe**, każda rola ma własny model: rozmowy, walka (Mistrz Gry), umysł (rozumienie gracza, nowe fakty i przedmioty), światy i obrazki. Model możesz też puścić na procesorze: wolniej, ale zwalnia kartę, co dobrze pasuje do umysłu, który pyta rzadko i w tle. Wybór zostaje w `settings.json` w katalogu danych.

Mały model wystarcza do rozmów, ale światy generuje wyraźnie słabsze niż duży model w chmurze.

## Zmienne

| Zmienna | Znaczenie |
|---|---|
| `NEURO_LLM_MODEL`, `NEURO_LLM_BASE_URL`, `NEURO_LLM_API_KEY` | główny model; pusta nazwa to tryb offline |
| `NEURO_TALK_MODEL`, `NEURO_TALK_BASE_URL`, `NEURO_TALK_API_KEY` | osobny model do kwestii postaci |
| `NEURO_GM_MODEL` | osobny model Mistrza Gry (na serwerze `NEURO_TALK_BASE_URL`) |
| `NEURO_WORLD_MODEL` | osobny model do generowania światów (także `_BASE_URL`, `_API_KEY`, `_CONTEXT`) |
| `NEURO_LLM_MAX_TOKENS` | limit długości kwestii, np. `220`, co przyspiesza małe modele |
| `NEURO_LLM_CONTEXT`, `NEURO_TALK_CONTEXT` | okno kontekstu w Ollamie; domyślnie 8192 dla głównego modelu |
| `NEURO_DEBUG` | `on` włącza podgląd pod `http://127.0.0.1:7788/` |
| `NEURO_LOG_TALKS` | `1` zapisuje rozmowy z pełnymi promptami do `talks.jsonl` (materiał do treningu) |
| `NEURO_TTS`, `NEURO_TTS_URL`, `NEURO_TTS_VOICE`, `NEURO_TTS_COMMAND` | głos na start: `1` systemowy, `chatterbox`, albo własne polecenie syntezy |
| `NEURO_STT` | `0` wyłącza mikrofon |
| `NEURO_IMAGES` | `auto` albo `kitty`: prawdziwe obrazki zamiast bloków (niżej) |
| `NEURO_BLOCKS` | znaki obrazków: `quad` (domyślnie), `sextant` (ostrzej, wymaga czcionki), `half` |
| `NEURO_TRUECOLOR` | `0` ogranicza obrazki do 256 kolorów |
| `NEURO_WORLD_ART` | `0` wyłącza malowanie grafiki wygenerowanych światów |
| `NEURO_AVATAR_PYTHON` | Python ze środowiskiem modelu obrazkowego |
| `NEURO_WORLD_DUMP` | plik na surową odpowiedź modelu przy generowaniu świata (do diagnozy) |

**Okno kontekstu.** Ollama domyślnie daje modelowi 4096 tokenów, a projekt śledztwa się w nich ledwo mieści. Gdy serwer okazuje się Ollamą, gra pyta ją jej własnym API i prosi o 8192. Jeśli generowanie kończy się błędem „unexpected end of JSON input”, sprawdź `ollama ps` (kolumna CONTEXT).

## Głos i mikrofon

Głos wybierasz w **Esc → Ustawienia**:

- **Piper**: na procesorze, zdanie w 0,1 s, brzmi naturalnie, ale bez nastrojów. Instalacja: `tools/speech/setup-piper.sh` albo `setup-piper.ps1` (ok. 200 MB).
- **Chatterbox**: słychać nastroje, klonuje głos z próbki. Zajmuje ok. 4 GB karty i kilka sekund na zdanie. Instalacja: `tools/speech/setup-tts.sh`. Próbki (5–15 s czystej polskiej mowy) wgrywasz i przypisujesz postaciom w zakładce **Głos** podglądu.
- **Systemowy**: SAPI na Windows, `say` na macOS. Nic nie trzeba instalować.

**Mikrofon:** F5 zaczyna nagrywanie, drugie F5 kończy. Mowę rozpoznaje lokalnie Whisper, a tekst trafia do wiersza poleceń do poprawienia. Środowisko stawia `tools/avatar/setup.sh`.

## Awatary i malowane światy

`tools/avatar/setup.sh` instaluje SD-Turbo (ok. 2,5 GB). Potem w wyborze postaci jest „Dodaj nowy…”: opisujesz postać, model ją maluje, a gra zamienia obraz na pixel art. Ten sam model maluje w tle tła, sceny i portrety każdego wygenerowanego świata (ok. 25 obrazków, na Macu z układem Apple ok. minuty) w stylu z karty świata.

**Prawdziwe obrazki** (`NEURO_IMAGES=auto`) działają w kitty, WezTerm, Ghostty i nowej Konsoli. W iTerm2 spróbuj `kitty`. Nie działają w Terminal.app, przez tmux ani u gości przez SSH.

## Podgląd w przeglądarce

`NEURO_DEBUG=on` pokazuje trwającą grę pod `http://127.0.0.1:7788/`:

- **Świat, Graf wiedzy, Fakty, Bestiariusz, Przedmioty, Ludzie, Zadania**: wszystko, co gra wie, odświeżane co dwie sekundy;
- **edycja na żywo**: kliknij opis, żeby go poprawić, albo ↻, żeby model napisał go od nowa; fakty, złoto i zdrowie też da się zmienić;
- **skąd**: przy każdym wpisie jego historia, czyli z jakiej kwestii powstał, jaki prompt dostał model i kto co poprawił;
- **Rozmowy**: każda wymiana z decyzją gry, pełnym promptem i odrzuconymi próbami. Złe kwestie zaznaczasz z uwagą, a trafiają do `feedback.jsonl` jako materiał do kolejnego treningu;
- **Wywołania**: każde zapytanie do każdego modelu z czasami, tokenami i pełną treścią.

API to zwykły JSON (`/api/world`, `/api/exchange`, `/api/fact`, …), więc da się z niego korzystać ze skryptów.
