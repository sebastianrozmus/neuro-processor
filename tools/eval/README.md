# Ocena na scenariuszach (tools/eval)

Uzupełnienie benchu (`tools/bench`): zamiast pojedynczych przypadków — 75 rozmów granych jak w grze, z historią. Bench sprawdza format pojedynczej kwestii; tu widać, co model robi po 5–12 turach, w nowych światach i przy trudnych wejściach.

## Scenariusze
`lib.py` (szablony promptów i bloków [GRA], skopiowane 1:1 z danych pack.sh), `chars.py` (postacie i światy), `build.py` → `scenarios.jsonl`:
- 25 długich (do 12 tur) i 50 krótkich rozmów; role: kupiec, najemnik, towarzysz, świadek, Mistrz Gry (walki po 4–6 rund, stan liczony z przebiegu);
- światy z treningu (Słony Brzeg, Szept Mokradeł, Smocza Góra, Rude Niebo, Pas Krzemowy, Siedem Korzeni, Zgasła Jutrznia…) i nowe (Lodowy Szlak, Archipelag Mgieł, Miasto Zegarów, Pustynia Szkła);
- trudne wejścia: gracz po angielsku, „ignoruj instrukcje”, elf u krasnoluda, wyklęty, prośby spoza roli, pokusa zmyślenia imienia/adresu u świadka, zamknięty kram.
`test-scen.jsonl` to test.jsonl z pack.sh w tym samym formacie (`from_test.py`).

## Uruchomienie
```
python3 run.py --model neuro-npc-run9b --scenarios scenarios.jsonl --out out/scen-run9b.jsonl   # OLLAMA_URL, domyślnie PC
python3 check.py scenarios.jsonl out/scen-run9b.jsonl out/scen-run7.jsonl                       # podsumowanie + *.checked.jsonl
python3 report.py scenarios.jsonl out/report.html out/scen-run7.checked.jsonl out/scen-run9b.checked.jsonl
```
Historia jak w grze: wcześniejsze kwestie gracza bez [GRA], model widzi własne poprzednie odpowiedzi. Temperatura 0,7, seed 7 (`--seed`, `--temperature`).

`check.py` (automat zasad): znacznik z listy 8 i zgodny z nastrojem [GRA], ≤3 zdania, bez gwiazdek/cudzysłowów/imienia na początku, liczby tylko z [GRA]/promptu i tylko cyframi, kwota z [GRA] obecna w kwestii, brak kwot gdy zakazane, powtórzone 4-wyrazowe zwroty z wcześniejszych kwestii, u Mistrza Gry brak zmyślonych śmierci. Nie ocenia sensu — to trzeba przeczytać (raport).

## Dane
- `audit.py` / `flags.py` — te same zasady na danych treningowych; `numwords.py` — liczby słownie, długość, >3 zdania w paczkach.
- `GOLD-GUIDE.md` + `gold/*.json` — wzorcowe odpowiedzi na wszystkie 279 tur scenariuszy (styl: żywo i zgodnie z zasadami naraz).
- `step3/gm-audit.json` — audyt 331 narracji Mistrza Gry w zbiorze run 6/7 (36 błędnych z poprawkami), `step3/gm-rewrite-*.json` — barwne wersje pozostałych 295, `step3/auto-flags.jsonl` — wiersze z twardymi błędami.
- `make_step3.py` — nakłada te poprawki na dane z pack.sh i dokłada złote odpowiedzi (wynik dla zbioru run 7: `D:\neuro-train\step3-run7`).

## DPO
`dpo_pack.py` buduje paczkę 3a (`{"id","context","rejected","why"}`) z odpowiedzi modelu i ręcznych notatek (`dpo-notes-*.json`); `dpo_merge.py` łączy ją z odpowiedziami GPT, sprawdza „chosen” automatem i zapisuje pary. Zamówienie: `tools/model/ZAMOWIENIE-krok3.md`.

## Wyniki (17.10.2026)
`out/*.checked.jsonl`, raport `out/report.html`; opis w `tools/model/MODELE.md`.
