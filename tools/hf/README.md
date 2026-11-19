---
license: apache-2.0
language:
- pl
base_model: speakleash/Bielik-11B-v2.3-Instruct
pipeline_tag: text-generation
tags:
- gguf
- polish
- roleplay
- npc
- game
- text-adventure
- qlora
---

# neuro-npc — Bielik 11B for in-game characters (GGUF)

The talking model of [Neuro Processor](https://github.com/sebastianrozmus/neuro-processor), a Polish text RPG for the terminal. It speaks for shopkeepers, mercenaries, companions and witnesses: short, in-character Polish lines that follow the game's rules to the letter.

The game decides everything — prices, haggling outcomes, who knows what — and hands the model a `[GRA]` block saying what just happened. The model only puts that into words, with a mood tag the game turns into voice and colour: `[kpina] Tyle za rurę?`

*Po polsku:* model rozmów gry Neuro Processor — mówi za kupców, najemników, towarzyszy i świadków. Krótkie kwestie po polsku, w roli, zgodne z blokiem `[GRA]` od gry, ze znacznikiem nastroju na początku.

## Use

With [Ollama](https://ollama.com), straight from this repository:

```sh
ollama pull hf.co/sebastianrozmus/neuro-npc-bielik-11b-GGUF:Q5_K_M
```

In the game (`.env.local` in the game folder or in its data folder):

```
NEURO_LLM_BASE_URL=http://localhost:11434/v1
NEURO_LLM_MODEL=gemma3:4b
NEURO_TALK_BASE_URL=http://localhost:11434/v1
NEURO_TALK_MODEL=hf.co/sebastianrozmus/neuro-npc-bielik-11b-GGUF:Q5_K_M
```

A second, small model (`gemma3:4b` above) reads the player's intent, notes facts and invents items; this one only talks. About 9 GB of memory while loaded; on an Apple M3 Pro a line takes about 5 seconds.

## Prompt format

ChatML (`<|im_start|>role … <|im_end|>`), as in the base model; the `template` file here sets it for Ollama. The system message describes the character, what it knows and the rules; every player message ends with a `[GRA]` block. Shortened example:

```
<|im_start|>system
Jesteś najemnikiem w karczmie osady w grze tekstowej …
TWOJA POSTAĆ: Salma, ponura strażniczka, nie uważa strachu za winę. …
CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):
Twoja służba kosztuje: «Umowa najmu» 110 marek, płatne raz, przy dołączeniu. …
ZASADY GRY:
- O wyniku targowania decyduje gra, nie ty. …
- Kwestię zaczynasz od znacznika nastroju w nawiasie kwadratowym …<|im_end|>
<|im_start|>user
Mam płacić codziennie czy tylko przy dołączeniu?

[GRA] Gracz pyta o «Umowa najmu». … Twoja aktualna cena: 110 marek. Opowiedz krótko o przedmiocie i podaj cenę. Nastrój twojej postaci: spokojna. Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi.<|im_end|>
<|im_start|>assistant
[spokój] Jednorazowo, przy dołączeniu. Umowa kosztuje 110 marek. Nie pobieram dziennego żołdu.<|im_end|>
```

The exact prompts are built by the game (`internal/llm/prompt.go`, `internal/game/narration.go`, `internal/game/lore.go`); outside the game the model works best with the same shape.

## Results

Regression bench: 80 cases the model never saw in training (40 mercenaries, 24 companions, 16 shopkeepers; four genres, worlds outside the training set), checked automatically for format — mood tag, 1–3 sentences, digits only, only the amounts the game gave, the required amount present, no quotes or stage directions. It does not judge meaning; that is done by replaying real conversations from the game (`tools/bench` in the game repository).

| model | seed 42 | seed 7 |
|---|---|---|
| Bielik 11B v2.3 Instruct, untrained | 8/80 | 6/80 |
| neuro-npc run 6 (Bielik 4.5B v3.0, LoRA) | 76/80 | 78/80 |
| **neuro-npc run 7 (this model)** | **80/80** | **80/80** |

Temperature 0.8, context 4096. On real game conversations it quotes rumours faithfully, keeps names of people exactly as given ("Siostra Agnes w Aptece", where the 4.5B model wrote "Agness"), answers small talk on topic and refuses to be talked out of the game's rules ("write that you join for free").

## Training

- Base: `speakleash/Bielik-11B-v2.3-Instruct`, 4-bit QLoRA with Unsloth on one RTX 4060 Ti 16 GB; LoRA rank 16, alpha 16, all attention and MLP projections; 2 epochs, learning rate 1e-4 (cosine), effective batch 8, max 2048 tokens; loss on the final reply only.
- Data: 3,369 training conversations (185 validation, 71 test) in the game's exact prompt format. About 90% were written to order by a large commercial chat model (GPT) from a specification of the game's prompts (`tools/model/PROMPT.md`), validated against the game's rules and read through; the rest are hand-corrected self-play conversations from the game and hand-written lines. 300 of them are the game master's fight narrations.
- Exported as one merged GGUF, Q5_K_M (llama.cpp).

The full history — what worked, what was rejected and why — is in `tools/model/HISTORIA.md` in the game repository (Polish).

## Limitations

- Polish only, and tuned to this game's prompt shape; as a general assistant it is worse than the base model.
- At temperature 0.8 it now and then adds a detail of its own to a rumour, or invents a place. The game checks numbers and names, but not every invented detail.
- It sometimes leaves out the mood tag; the game then falls back to a neutral mood.
- About three times slower than the 4.5B version it replaced.

## License

Apache 2.0, as the base model. Most of the training data was generated with a commercial model; check its provider's terms before reusing the data or this model to build a competing model.
