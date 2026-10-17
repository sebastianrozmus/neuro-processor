Zamówienie kroku 3 (DPO 3a): pary z prawdziwych błędów run 9b
================================================================

Krok 3 z [ZAMOWIENIE-run9.md](ZAMOWIENIE-run9.md), wariant **3a** (preferowany): złe kwestie pochodzą od naszego modelu, GPT pisze tylko poprawkę.

Skąd paczka
-----------

`dpo-run9b-3a.jsonl` (60 linii) zbudował `tools/eval/dpo_pack.py` z odpowiedzi `neuro-npc-run9b` na 75 scenariuszy oceny (`tools/eval/scenarios.jsonl`, 17.10.2026). Każda linia to `{"id","context","rejected","why"}`:
- `context` — dokładnie to, co widział run 9b: SYSTEM, historia z jego własnymi wcześniejszymi kwestiami (słowa gracza bez [GRA]) i ostatnia wiadomość gracza z [GRA];
- `rejected` — jego kwestia;
- `why` — co w niej złe: 35 linii opisanych ręcznie (zła kwota „Sto osiem” zamiast 18, znacznik [ostrzeżenie], odesłanie do złej osoby, przekręcone i zmyślone fakty, pomylone role i strony transakcji, zmyślone śmierci u Mistrza Gry, zastrzeżenia „nie będę zmyślać” na głos) i 25 wykrytych automatem (głównie liczby słownie, ponad 3 zdania).

Jak zamówić (ChatGPT, najmocniejszy model, rozumowanie „wysokie”)
-----------------------------------------------------------------

1. Nowa rozmowa. Załącz **`tools/model/PROMPT.md`** (kontrakt gry) i **`tools/model/dpo-run9b-3a.jsonl`**.
2. Wklej prompt z sekcji „Krok 3 → 3a” w `ZAMOWIENIE-run9.md` (blok zaczynający się od „Dostajesz linie JSONL {"id","context","rejected","why"}…”) i dopisz na końcu:
   > Linie są w załączonym pliku dpo-run9b-3a.jsonl. Zrób po 30 linii na odpowiedź, zaczynając od pierwszej; napiszę „dalej”.
3. Każdą odpowiedź zapisz jako `extra-gpt-dpo-k1.jsonl`, `-k2` (dwie odpowiedzi po 30 linii).
4. Sprawdź:
   ```
   cd tools/eval
   python3 dpo_merge.py ../model/dpo-run9b-3a.jsonl ../model/extra-gpt-dpo-k1.jsonl ../model/extra-gpt-dpo-k2.jsonl --out ../model/dpo-pairs-run9b.jsonl
   ```
   Wypisuje kwestie, które łamią zasady gry (znacznik, zdania, liczby, kwota z [GRA]) albo są identyczne z `rejected` — te popraw ręcznie albo odeślij GPT z uwagą.
5. Przeczytaj 15 losowych par: czy `chosen` naprawia dokładnie wadę z `why` i zmienia niewiele poza nią.

Za mało na trening
------------------

60 par to za mało na DPO (cel z zamówienia: ok. 300). Ta paczka to pierwsza porcja i wzorzec. Następne:
- te same scenariusze puszczone na run 9b w innych seedach (`tools/eval/run.py --seed 42`, `--seed 1`), nowe błędy opisane w `dpo-notes-*.json` i zbudowane `dpo_pack.py`;
- odrzucenia z gry i z podglądu (inspektor), jak przewiduje 3a;
- ewentualnie 3b (pary syntetyczne) na brakujące typy błędów.

Do treningu potrzebny jest jeszcze wariant `tools/pc/train.py` z `trl.DPOTrainer` (od adaptera run 9b, beta 0,1, 1 epoka, lr 5e-6) — zapowiedziany w `ZAMOWIENIE-run9.md`.
