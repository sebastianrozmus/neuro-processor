# Trening głosu gry na PC z kartą NVIDIA (Windows)

Ten folder przenosi trening modelu `neuro-npc` z Maca na PC z kartą NVIDIA. Przy **16 GB** pamięci karty (np. RTX 4060 Ti 16 GB) mieści się **Bielik 11B** w 4 bitach (QLoRA), którego na Macu z 18 GB trenować się nie da. Dane, szablon promptu i sposób oceny zostają te same co na Macu. Wytrenowany model wraca na Maca jako plik GGUF i gra używa go przez Ollamę jak dotąd.

Co tu jest:

| plik | co robi |
|---|---|
| `setup.sh` | jednorazowo przygotowuje Ubuntu w Windows (WSL2): Python z Unsloth i llama.cpp |
| `train.py` | trening QLoRA na Bieliku 11B; zapis adaptera i opcjonalnie GGUF |
| `lora-gguf.sh` | sam adapter jako mały GGUF (gdy brakuje RAM na scalanie) |

Po stronie Maca: `tools/model/pack.sh` pakuje dane, `tools/model/import.sh` rejestruje wynik w Ollamie.

---

## Krok 1 — Ubuntu w Windows (WSL2), raz

1. Zaktualizuj sterownik NVIDIA w Windows (GeForce Experience albo nvidia.com, „Game Ready” albo „Studio”, dowolny z ostatnich miesięcy). **Nie instaluj** sterownika ani CUDA w Ubuntu — WSL korzysta ze sterownika Windows.
2. PowerShell **jako administrator**:
   ```powershell
   wsl --install -d Ubuntu-24.04
   ```
   Uruchom ponownie komputer, jeśli poprosi. Po starcie Ubuntu zapyta o nazwę użytkownika i hasło (dowolne, zapamiętaj hasło — `sudo` o nie pyta).
3. W oknie Ubuntu sprawdź kartę:
   ```bash
   nvidia-smi
   ```
   Ma pokazać RTX 4060 Ti i 16 GB. Jeśli nie — w PowerShell `wsl --shutdown`, zaktualizuj sterownik, spróbuj jeszcze raz.

Miejsce: ok. **60 GB** na dysku C: (Ubuntu, biblioteki, baza 11B ~22 GB w pełnej wersji pobieranej przy scalaniu, wyniki). RAM komputera: do samego treningu wystarczy 16 GB, do scalenia modelu w jeden GGUF (krok 5a) potrzeba **32 GB**; przy 16 GB użyj drogi 5b.

## Krok 2 — narzędzia, raz

Skopiuj ten folder (`tools/pc`) do Ubuntu. Najprościej: w Eksploratorze Windows wpisz w pasku adresu `\\wsl$\Ubuntu-24.04\home\<twoja-nazwa>` i przeciągnij tam folder `pc`. Potem w Ubuntu:

```bash
cd ~/pc
bash setup.sh
```

Trwa 10–20 minut (pobiera PyTorch z CUDA, Unsloth, llama.cpp). Na końcu ma wypisać `CUDA: True NVIDIA GeForce RTX 4060 Ti`.

Bielik 11B może wymagać akceptacji warunków na Hugging Face: wejdź na <https://huggingface.co/speakleash/Bielik-11B-v2.3-Instruct>, zaloguj się, zaakceptuj (jeśli jest przycisk), utwórz token (Settings → Access Tokens, „Read”) i w Ubuntu:

```bash
~/neuro-train/venv/bin/huggingface-cli login
```

## Krok 3 — dane z Maca

Na Macu, w katalogu gry:

```bash
tools/model/pack.sh
```

Na Pulpicie powstaje `neuro-data.tar.gz` (~2 MB: train / valid / test, ten sam podział co przy treningu na Macu). Przenieś go na PC (pendrive, chmura, AirDrop do telefonu…) i w Ubuntu:

```bash
mkdir -p ~/neuro-train && tar -xzf /mnt/c/Users/<twoja-nazwa-w-windows>/Downloads/neuro-data.tar.gz -C ~/neuro-train
ls ~/neuro-train/data        # train.jsonl  valid.jsonl  test.jsonl
```

(`/mnt/c/...` to dysk C: Windows widziany z Ubuntu.)

## Krok 4 — próba na sucho, potem trening

```bash
cd ~/pc
~/neuro-train/venv/bin/python3 train.py --data ~/neuro-train/data --out ~/neuro-train/run7 --dry
```

Pobierze bazę (~7 GB w 4 bitach), sprawdzi, że znaczniki `<|im_start|>`/`<|im_end|>` są w niej pojedynczymi tokenami, zakoduje dane i pokaże koniec promptu z odpowiedzią. Jeśli wypisze błąd o szablonie — zatrzymaj się i daj znać.

Trening:

```bash
~/neuro-train/venv/bin/python3 train.py --data ~/neuro-train/data --out ~/neuro-train/run7 --gguf q5_k_m 2>&1 | tee ~/neuro-train/run7.log
```

- Domyślnie: 2 epoki, lr 1e‑4, LoRA rank 16, akumulacja 8, maks. 2048 tokenów (najdłuższy przykład ma ~1400). Zbiór ma ok. 3,4 mln tokenów na epokę; na 4060 Ti szacunkowo **2–4 godziny** — pierwsze minuty logu pokażą tempo.
- W logu co 10 kroków `loss`, co ~1/8 treningu `eval_loss` — ma spadać (z szumem).
- Komputer w tym czasie może robić inne rzeczy, ale nie gry na tej samej karcie.
- Punkty kontrolne lądują co ~1/4 treningu w `run7/checkpoints/`. Gdy trening się przerwie, można zacząć od wag ostatniego z nich (`--resume ~/neuro-train/run7/checkpoints/checkpoint-<numer>`, najlepiej z `--epochs` zmniejszonym o to, co już przeszło); harmonogram uczenia liczy się wtedy od nowa.

## Krok 5 — model do Ollamy na Macu

**5a. Cały model jako jeden GGUF** (domyślnie, z `--gguf q5_k_m`, potrzeba 32 GB RAM): w `~/neuro-train/run7/gguf/` powstaje plik `…Q5_K_M.gguf` (~7,5 GB). Skopiuj go na Maca i:

```bash
tools/model/import.sh neuro-npc-run7 ~/Downloads/<plik>.gguf
```

`q8_0` (~11,5 GB) jest wierniejszy, ale na Macu z 18 GB razem z gemmą będzie ciasno; `q5_k_m` to rozsądny środek.

**5b. Sam adapter** (gdy RAM nie starczył na scalanie, albo żeby przenosić 100 MB zamiast 7 GB):

```bash
bash ~/pc/lora-gguf.sh ~/neuro-train/run7/adapter ~/neuro-train/neuro-run7-lora.gguf
```

Na Macu Bielik 11B jest już w Ollamie, więc:

```bash
tools/model/import.sh neuro-npc-run7 ~/Downloads/neuro-run7-lora.gguf SpeakLeash/bielik-11b-v2.3-instruct:Q4_K_M
```

Adapter trenowany na bazie 4-bitowej nakłada się na kwantyzowaną bazę Ollamy z niewielką stratą; jeśli wyniki w benchu wyjdą wyraźnie gorsze niż się spodziewamy, wracamy do 5a.

## Krok 6 — ocena i przełączenie (na Macu, jak zawsze)

Bench 80 przypadków w dwóch seedach i odtworzenie prawdziwych rozmów z gry (opis w `tools/model/HISTORIA.md`, „Tutorial”, krok 4), porównanie z `neuro-npc-run6`. Dopiero gdy wygrywa: w `.env.local` `NEURO_TALK_MODEL=neuro-npc-run7`. 11B odpowiada ok. 2× wolniej niż 4,5B.

Kolejne runy: nowe dane → `pack.sh` → na PC trening od poprzedniego adaptera:

```bash
~/neuro-train/venv/bin/python3 train.py --data ~/neuro-train/data --out ~/neuro-train/run8 --resume ~/neuro-train/run7/adapter --epochs 1 --rate 4e-5 --gguf q5_k_m
```

---

## Opcjonalnie: mowa Chatterbox z PC

Domyślnie gra mówi głosem systemowym Maca. Jeśli zechcesz głosu sklonowanego z emocjami w tempie gry (na Macu jedna kwestia to ~2 minuty, na karcie NVIDIA kilka sekund), serwer mowy może działać na PC, a gra na Macu tylko go pyta.

Najprościej w samym Windows (nie w WSL — z WSL port trudniej wystawić do sieci). PowerShell:

```powershell
py -3.11 -m venv $HOME\neuro-tts
$HOME\neuro-tts\Scripts\pip install chatterbox-tts
$HOME\neuro-tts\Scripts\pip install --force-reinstall torch==2.6.0 torchaudio==2.6.0 --index-url https://download.pytorch.org/whl/cu124
mkdir $HOME\neuro-voices
```

Skopiuj z repo `internal/speech/chatterbox.py` do `$HOME` i próbki głosów (WAV, np. `glos-1.wav` z Maca: `~/Library/Application Support/neuro-processor/voices/`) do `$HOME\neuro-voices`. Start:

```powershell
$HOME\neuro-tts\Scripts\python $HOME\chatterbox.py --host 0.0.0.0 --port 7790 --voices $HOME\neuro-voices --device cuda
```

Za pierwszym razem pobierze model (~2 GB) i Windows zapyta o zgodę zapory — zezwól w sieci prywatnej. Na Macu w `.env.local`:

```
NEURO_TTS=chatterbox
NEURO_TTS_URL=http://<adres-IP-PC>:7790
```

(adres PC: `ipconfig` w PowerShell, „Adres IPv4”). Gra widzi działający serwer i nie startuje własnego. Przydział głosów i nastrojów ustawiasz w podglądzie, zakładka Głos; nazwy próbek muszą być te same co w folderze na PC (wgrywanie próbek z podglądu zapisuje je na Macu, więc przy serwerze na PC kopiuj je ręcznie).
