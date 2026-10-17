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

Wynik pierwszej paczki (17.10): 56 par, 4 odrzucone przez automat za powtórzone zwroty.

Druga paczka: `dpo-run9b-3a-2.jsonl` (321 linii)
-----------------------------------------------

Te same 75 scenariuszy puszczone na run 9b z seedami 11, 23, 42, 101 (`run.py --seed`). Nowe błędy opisane ręcznie w `tools/eval/dpo-notes-run9b-s*.json` (218 notatek) plus automat; id mają przedrostek seeda (`s11:L01-vika#3`), bo ta sama tura wraca w kilku przebiegach z inną historią. Najczęstsze wady: liczby słownie (ok. 70), ponad 3 zdania (25), przekręcone i zmyślone fakty (ok. 25), zasady na głos (11), bełkot i zdania bez sensu (ok. 20), błędy odmiany i rodzaju (ok. 20), Mistrz Gry: złe przypisanie ciosów, druga osoba, zmyślone maksima zdrowia.

Uwaga do rodzaju: prompt gry mówi o rozmówcy zawsze „go/nim” („Marta, detektyw; wynajął go”), więc męskie formy wobec gracza są błędem tylko wtedy, gdy gracz sam mówi o sobie w rodzaju żeńskim („doszłam”).

1. Można w tej samej rozmowie co k1/k2 (GPT ma już kontrakt i prompt 3a). Załącz `dpo-run9b-3a-2.jsonl` i napisz:
   > Następna paczka w tym samym formacie: dpo-run9b-3a-2.jsonl, 321 linii. Zasady jak wcześniej, id przepisuj dokładnie (z przedrostkiem s11:/s23:/…). Po 30 linii na odpowiedź, zaczynając od pierwszej; napiszę „dalej”.
2. Odpowiedzi zapisz jako `extra-gpt-dpo-k3.jsonl` … `extra-gpt-dpo-k13.jsonl` (11 odpowiedzi, ostatnia ma 21 linii).
3. Sprawdź:
   ```
   cd tools/eval
   python3 dpo_merge.py ../model/dpo-run9b-3a-2.jsonl ../model/extra-gpt-dpo-k{3..13}.jsonl --out ../model/dpo-pairs-k3k13.jsonl
   ```

Razem z pierwszą paczką daje to ok. 370 par. Dalsze źródła, gdyby trzeba więcej: odrzucenia z gry i z podglądu (inspektor), 3b (pary syntetyczne) na brakujące typy błędów.

Do treningu potrzebny jest jeszcze wariant `tools/pc/train.py` z `trl.DPOTrainer` (od adaptera run 9b, beta 0,1, 1 epoka, lr 5e-6) — zapowiedziany w `ZAMOWIENIE-run9.md`.
