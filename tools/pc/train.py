"""Trains the game's voice (neuro-npc) on a PC with an NVIDIA card: QLoRA
on Bielik 11B with Unsloth, on the set tools/model/pack.sh packs on the Mac.

Run inside WSL2 (Ubuntu) after tools/pc/setup.sh:

    python3 train.py --data ~/neuro-train/data --out ~/neuro-train/run7

It writes the LoRA adapter (out/adapter), and with --gguf the whole model as
one GGUF file for Ollama (out/neuro-npc-run7.Q5_K_M.gguf), or with
--lora-gguf only the adapter as GGUF, to put on a base Ollama already has.

The prompt is rendered exactly as the game's Ollama template does
(ChatML: <|im_start|>role\\ncontent<|im_end|>), and the loss counts only
the final reply, as mlx_lm's --mask-prompt does on the Mac.

With --dpo it trains preference pairs instead (step 3 of
tools/model/ZAMOWIENIE-run9.md): --data holds train.jsonl and valid.jsonl
with {"prompt": [messages], "chosen", "rejected"} from tools/eval/dpo_merge.py,
and --base is the model being corrected, merged (e.g. run9b/gguf, the 16-bit
folder Unsloth leaves next to the GGUF). A fresh adapter is trained on it, and
the same model with the adapter switched off is the DPO reference.
"""
import argparse
import json
import math
import os
import sys
import time


def render(messages):
    """The conversation as the Ollama template renders it, split into the
    prompt (everything up to the last reply) and the reply itself."""
    prompt = ""
    for message in messages[:-1]:
        prompt += f"<|im_start|>{message['role']}\n{message['content']}<|im_end|>\n"
    prompt += "<|im_start|>assistant\n"
    reply = messages[-1]["content"] + "<|im_end|>"
    return prompt, reply


LLAMA_TOKENIZE = os.path.expanduser("~/.unsloth/llama.cpp/llama-tokenize")


def game_tokenizer(base):
    """The tokenizer as llama.cpp (Ollama, the game) cuts the text: with the
    legacy SentencePiece rule, "▁" before text that follows a special token
    (<|im_start|>▁assistant). Bielik v3.1's newer tokenizer drops that "▁", so
    run26s learnt role tokens the game never feeds; v3.0's kept it. The <s>
    llama.cpp puts first is added in encode (v3.0 adds none of its own)."""
    from transformers import AutoTokenizer
    return AutoTokenizer.from_pretrained(base, legacy=True, from_slow=True)


def with_bos(tokenizer, ids, spans):
    if tokenizer.bos_token_id is not None and ids[:1] != [tokenizer.bos_token_id]:
        return [tokenizer.bos_token_id] + ids, [(0, 0)] + spans
    return ids, spans


def check_llama(tokenizer, rows, gguf, count=24):
    """Stops the run unless the training tokens of a spread of rows equal what
    llama.cpp makes of the same text with the base's GGUF (run26s lesson: the
    rendered text was identical, the tokens were not)."""
    import subprocess
    step = max(1, len(rows) // count)
    for row in rows[::step][:count]:
        prompt, reply = render(row["messages"])
        tokens = tokenizer(prompt + reply, add_special_tokens=True, return_offsets_mapping=True)
        ids, _ = with_bos(tokenizer, tokens["input_ids"], tokens["offset_mapping"])
        out = subprocess.run([LLAMA_TOKENIZE, "-m", gguf, "-p", prompt + reply, "--ids", "--log-disable"],
                             capture_output=True, text=True, check=True).stdout
        ref = json.loads(out.strip().splitlines()[-1])
        if ids != ref:
            k = next((j for j, (a, b) in enumerate(zip(ids, ref)) if a != b), min(len(ids), len(ref)))
            sys.exit(f"tokeny treningu ≠ llama.cpp od pozycji {k}: {tokenizer.convert_ids_to_tokens(ids[k:k + 4])} "
                     f"vs {tokenizer.convert_ids_to_tokens(ref[k:k + 4])}")
    print(f"tokeny zgodne z llama.cpp ({gguf}) na {min(count, len(rows[::step]))} wierszach", flush=True)


def load(path):
    with open(path, encoding="utf-8") as handle:
        return [json.loads(line) for line in handle if line.strip()]


def encode(rows, tokenizer, limit, bos=True, kept=None):
    """Tokens and labels: the prompt is masked (-100), the reply is learnt.
    Rows longer than the limit are dropped, not cut, so no reply is
    learnt without its prompt.

    The whole conversation is tokenized at once, as Ollama does: Bielik's
    SentencePiece tokenizer puts "▁" before any text tokenized alone, so a
    reply tokenized apart would start with "▁[" where the game sees "[".
    The mask is set by character offsets, not by a separate prompt count.

    With bos (the llama.cpp path, run27 on) two more checks, from the Codex
    review of 20.11: the prompt tokenized alone must be the prefix of the whole
    (that is what the game feeds before the reply), and the last token must be
    a learnt <|im_end|>. kept collects the indices of the rows used, so two
    runs can be compared on the same rows."""
    encoded, dropped = [], 0
    end_id = tokenizer("<|im_end|>", add_special_tokens=False)["input_ids"]
    for index, row in enumerate(rows):
        messages = row["messages"]
        if not messages or messages[-1]["role"] != "assistant":
            dropped += 1
            continue
        prompt, reply = render(messages)
        tokens = tokenizer(prompt + reply, add_special_tokens=True, return_offsets_mapping=True)
        ids, spans = tokens["input_ids"], tokens["offset_mapping"]
        if bos:
            ids, spans = with_bos(tokenizer, ids, spans)
        if len(ids) > limit:
            dropped += 1
            continue
        # Special tokens (bos) have the empty span (0, 0) and belong to the prompt.
        # A token across the boundary would teach a split the game never feeds.
        if any(start < len(prompt) < end for start, end in spans):
            dropped += 1
            continue
        labels = [token if start >= len(prompt) and end > start else -100 for token, (start, end) in zip(ids, spans)]
        if all(label == -100 for label in labels):
            dropped += 1
            continue
        if bos:
            alone = tokenizer(prompt, add_special_tokens=True, return_offsets_mapping=True)
            head, _ = with_bos(tokenizer, alone["input_ids"], alone["offset_mapping"])
            if ids[:len(head)] != head or any(label != -100 for label in labels[:len(head)]) \
                    or len(end_id) != 1 or ids[-1] != end_id[0] or labels[-1] != end_id[0]:
                sys.exit(f"wiersz {index}: prompt nie jest prefiksem całości albo końcowy <|im_end|> nie jest uczony")
        if kept is not None:
            kept.append(index)
        encoded.append({"input_ids": ids, "attention_mask": [1] * len(ids), "labels": labels})
    return encoded, dropped


def pairs(path):
    """DPO pairs as plain text in the game's template: the rendered prompt and
    both replies ending with <|im_end|> (the base's eos, so trl adds nothing)."""
    rows = []
    for row in load(path):
        prompt, chosen = render(row["prompt"] + [{"role": "assistant", "content": row["chosen"]}])
        rows.append({"prompt": prompt, "chosen": chosen, "rejected": row["rejected"] + "<|im_end|>"})
    return rows


LOGP = "eval_logps/chosen"


def make_gate(base_class, eval_from, start, max_drop):
    """DPO callback: no checkpoints before --eval-from, and with --stop-logp-drop
    the run ends at the first evaluation whose logp chosen (sum over the reply,
    mean over the valid pairs, as TRL reports it) fell more than max_drop nats
    below the evaluation before training. The Trainer evaluates, then saves,
    then stops, so that point is kept too; the earlier ones are never deleted
    (no save limit), and the picker takes the last one within the limit."""

    class Gate(base_class):
        def __init__(self):
            self.stopped_at, self.missing = None, False

        def on_step_end(self, _args, state, control, **_kwargs):
            if state.global_step < eval_from:
                control.should_save = False
            return control

        def on_evaluate(self, _args, state, control, metrics=None, **_kwargs):
            if not max_drop:
                return control
            if not metrics or LOGP not in metrics:
                self.missing, control.should_training_stop = True, True
                print(f"STOP-BRAK: krok {state.global_step} bez {LOGP} w ewaluacji", flush=True)
                return control
            drop = start - metrics[LOGP]
            print(f"bramka logp: krok {state.global_step}, logp chosen {metrics[LOGP]:.4f}, spadek {drop:.4f}, "
                  f"margines {metrics.get('eval_rewards/margins', float('nan')):.4f}", flush=True)
            if drop > max_drop:
                self.stopped_at, control.should_training_stop = state.global_step, True
                print(f"STOP-LOGP: krok {state.global_step}, spadek {drop:.4f} > {max_drop}", flush=True)
            return control

    return Gate()


def train_dpo(args, model, tokenizer):
    import torch
    from datasets import Dataset
    from transformers import TrainerCallback
    from trl import DPOConfig, DPOTrainer
    from unsloth import FastLanguageModel

    if tokenizer.eos_token != "<|im_end|>":
        sys.exit(f"eos w {args.base} to {tokenizer.eos_token!r}, a odpowiedzi kończą się <|im_end|>")
    train = pairs(os.path.join(args.data, "train.jsonl"))
    valid = pairs(os.path.join(args.data, "valid.jsonl"))
    steps = math.ceil(len(train) * args.epochs / args.accumulate)
    print(f"DPO: train {len(train)} par, valid {len(valid)}; {steps} kroków po {args.accumulate}, lr {args.rate}, beta {args.beta}, rpo_alpha {args.rpo_alpha}", flush=True)
    # Dense by default: at lr 5e-5 the usable window was steps 20-22 (run25d2); evaluations every steps//4 skipped it
    # in run24 and run25d. --eval-every -1 restores the old sparse grid.
    if args.eval_every > 0:
        every, delay = args.eval_every, args.eval_from
    else:
        every, delay = max(steps // 4, 5), 0
    # Every saved point must survive: the picker may need the earliest one (run25d lost step 14 to a limit of 4).
    limit = None if args.save_limit <= 0 else args.save_limit
    print(f"ewaluacja i zapis co {every} kroków od kroku {delay}, limit zapisów {limit}, stop przy spadku logp > {args.stop_logp_drop or '—'}", flush=True)
    if args.dry:
        print("--- szablon (koniec promptu, chosen, rejected) ---")
        print(train[0]["prompt"][-300:] + train[0]["chosen"] + "\n" + train[0]["rejected"])
        return

    model = FastLanguageModel.get_peft_model(
        model, r=args.rank, lora_alpha=args.rank, lora_dropout=0, bias="none",
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        use_gradient_checkpointing="unsloth", random_state=args.seed,
    )
    if args.eval_only:
        return eval_dpo_by_category(args, model, tokenizer, valid, Dataset, DPOConfig, DPOTrainer, torch)
    trainer = DPOTrainer(
        model=model, ref_model=None, processing_class=tokenizer,
        train_dataset=Dataset.from_list(train), eval_dataset=Dataset.from_list(valid),
        args=DPOConfig(
            output_dir=os.path.join(args.out, "checkpoints"), beta=args.beta, max_length=args.max_len,
            # RPO: newer TRL dropped rpo_alpha; the same loss is sigmoid DPO plus weighted NLL on chosen
            **({"loss_type": ["sigmoid", "sft"], "loss_weights": [1.0, args.rpo_alpha]} if args.rpo_alpha > 0 else {}),
            per_device_train_batch_size=1, per_device_eval_batch_size=1, gradient_accumulation_steps=args.accumulate,
            num_train_epochs=args.epochs, learning_rate=args.rate, lr_scheduler_type="cosine", warmup_steps=max(1, round(steps * 0.1)),
            optim="adamw_8bit", weight_decay=0.0, bf16=torch.cuda.is_bf16_supported(), fp16=not torch.cuda.is_bf16_supported(),
            # DPO overshoots quickly: chosen log-probs first rise, then fall with the rejected ones, so every
            # evaluation is saved and the best one can be picked by the eval logps, not only by the margin.
            logging_steps=5, eval_strategy="steps", eval_steps=every, eval_delay=delay, save_strategy="steps",
            save_steps=every, save_total_limit=limit,
            report_to="none", seed=args.seed,
        ),
    )
    before = trainer.evaluate()
    print("przed treningiem:", before, flush=True)
    gate = None
    if args.eval_every > 0 or args.stop_logp_drop:
        if args.stop_logp_drop and LOGP not in before:
            sys.exit(f"brak {LOGP} w ewaluacji przed treningiem — nie ma od czego liczyć spadku")
        gate = make_gate(TrainerCallback, delay, before.get(LOGP), args.stop_logp_drop)
        trainer.add_callback(gate)
    started = time.time()
    trainer.train()
    print(f"trening: {(time.time() - started) / 60:.0f} min", flush=True)
    if gate and gate.missing:
        sys.exit("ewaluacja bez logp chosen — przerwane (błąd techniczny, nie decyzja)")
    if gate and gate.stopped_at:
        # The final weights are past the limit; the picker reads the saved checkpoints.
        print(f"zatrzymane na kroku {gate.stopped_at} (spadek logp chosen > {args.stop_logp_drop}); out/adapter nie zapisany", flush=True)
        return
    print("ewaluacja:", trainer.evaluate(), flush=True)
    adapter = os.path.join(args.out, "adapter")
    model.save_pretrained(adapter)
    tokenizer.save_pretrained(adapter)
    print(f"adapter: {adapter}", flush=True)


def eval_dpo_by_category(args, model, tokenizer, valid, Dataset, DPOConfig, DPOTrainer, torch):
    """--eval-only: the valid pairs per category (--cat-file: one category per
    valid.jsonl line, in order) for the adapter in --resume; the reference is
    the same model with the adapter off, so rewards/chosen / beta is the change
    of logp chosen against the SFT. Also the mean reply length in tokens, for
    logp per token."""
    from peft import set_peft_model_state_dict
    from safetensors.torch import load_file

    result = set_peft_model_state_dict(model, load_file(os.path.join(args.resume, "adapter_model.safetensors")))
    if result.unexpected_keys:
        sys.exit(f"adapter z {args.resume} nie pasuje (rank?): {result.unexpected_keys[:3]}")
    cats = json.load(open(args.cat_file, encoding="utf-8"))
    if len(cats) != len(valid):
        sys.exit(f"{args.cat_file}: {len(cats)} kategorii, a valid ma {len(valid)} par")
    trainer = DPOTrainer(
        model=model, ref_model=None, processing_class=tokenizer, train_dataset=Dataset.from_list(valid[:1]),
        eval_dataset=Dataset.from_list(valid),
        args=DPOConfig(output_dir=os.path.join(args.out, "eval-only"), beta=args.beta, max_length=args.max_len,
                       per_device_eval_batch_size=1, bf16=torch.cuda.is_bf16_supported(), fp16=not torch.cuda.is_bf16_supported(),
                       report_to="none", seed=args.seed),
    )
    groups = {"wszystkie": list(range(len(valid)))}
    for i, cat in enumerate(cats):
        groups.setdefault(cat, []).append(i)
    for name, idx in groups.items():
        rows = [valid[i] for i in idx]
        tokens = sum(len(tokenizer(r["chosen"], add_special_tokens=False)["input_ids"]) for r in rows) / len(rows)
        metrics = trainer.evaluate(eval_dataset=Dataset.from_list(rows), metric_key_prefix="eval")
        print(f"kategoria {name}: n {len(rows)}, tokeny chosen {tokens:.1f}, {json.dumps(metrics, ensure_ascii=False)}", flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--data", required=True, help="folder with train.jsonl and valid.jsonl from pack.sh")
    parser.add_argument("--out", required=True)
    parser.add_argument("--base", default="speakleash/Bielik-11B-v2.3-Instruct")
    parser.add_argument("--name", default="neuro-npc-run7")
    parser.add_argument("--epochs", type=float, default=2)
    parser.add_argument("--rate", type=float, default=1e-4)
    parser.add_argument("--rank", type=int, default=16)
    parser.add_argument("--accumulate", type=int, default=8)
    parser.add_argument("--max-len", type=int, default=2048)
    parser.add_argument("--resume", default="", help="adapter folder to continue from")
    parser.add_argument("--resume-from", default="", help="Trainer checkpoint dir: continue the same run (optimizer, schedule, data order)")
    parser.add_argument("--gguf", default="", help="also write the whole model as GGUF: q5_k_m, q8_0, q4_k_m")
    parser.add_argument("--lora-gguf", action="store_true", help="also write the adapter alone as GGUF")
    parser.add_argument("--dry", action="store_true", help="only check the data and the template, no training")
    parser.add_argument("--dpo", action="store_true", help="train DPO pairs on a merged --base instead of SFT")
    parser.add_argument("--beta", type=float, default=0.1)
    parser.add_argument("--rpo-alpha", type=float, default=0.0, help="DPO: add this weight x NLL on chosen (RPO; TRL loss_type sigmoid+sft)")
    parser.add_argument("--seed", type=int, default=7, help="seed of the trainer (data order, LoRA init)")
    parser.add_argument("--save-parts", type=int, default=4, help="SFT: checkpoints per run (4 = every half epoch of 2)")
    parser.add_argument("--stop-at", type=float, default=0.0, help="SFT: stop after this many epochs, keeping the --epochs schedule")
    parser.add_argument("--eval-every", type=int, default=2, help="DPO: evaluate and save every K steps (-1 = old grid, steps//4)")
    parser.add_argument("--eval-from", type=int, default=10, help="DPO: first step to evaluate and save, with --eval-every > 0")
    parser.add_argument("--save-limit", type=int, default=0, help="DPO: checkpoints kept (0 = all)")
    parser.add_argument("--stop-logp-drop", type=float, default=0.0, help="DPO: end the run once eval logp chosen falls more than this many nats")
    parser.add_argument("--eval-only", action="store_true", help="DPO: no training; evaluate the --resume adapter on valid per --cat-file category")
    parser.add_argument("--cat-file", default="", help="DPO --eval-only: JSON list, one category per valid.jsonl line")
    parser.add_argument("--llama-check", default="", help="SFT: GGUF of the base; stop unless training tokens equal llama.cpp's")
    parser.add_argument("--hf-tokens", action="store_true", help="SFT: tokenize as before run27 (HF tokenizer as loaded, no added <s>)")
    args = parser.parse_args()

    from unsloth import FastLanguageModel  # imported first, as Unsloth asks
    import torch
    from transformers import DataCollatorForSeq2Seq, Trainer, TrainingArguments
    from datasets import Dataset

    model, tokenizer = FastLanguageModel.from_pretrained(model_name=args.base, max_seq_length=args.max_len, load_in_4bit=True, dtype=None)

    for token in ("<|im_start|>", "<|im_end|>"):
        ids = tokenizer(token, add_special_tokens=False)["input_ids"]
        if len(ids) != 1:
            sys.exit(f"{token} nie jest jednym tokenem w {args.base} ({ids}); szablon gry nie pasuje do tej bazy")

    if args.dpo:
        return train_dpo(args, model, tokenizer)

    # SFT tokens as the game's llama.cpp cuts them (run26s lesson); --hf-tokens keeps the old way for repeats.
    cutter = tokenizer if args.hf_tokens else game_tokenizer(args.base)
    rows = load(os.path.join(args.data, "train.jsonl"))
    if args.llama_check:
        check_llama(cutter, rows, args.llama_check)
    elif not args.hf_tokens:
        print("UWAGA: bez --llama-check zgodność tokenów z grą nie jest sprawdzona", flush=True)
    kept_train, kept_valid = [], []
    train, dropped = encode(rows, cutter, args.max_len, bos=not args.hf_tokens, kept=kept_train)
    valid, _ = encode(load(os.path.join(args.data, "valid.jsonl")), cutter, args.max_len, bos=not args.hf_tokens, kept=kept_valid)
    os.makedirs(args.out, exist_ok=True)
    with open(os.path.join(args.out, "rows.json"), "w", encoding="utf-8") as handle:
        json.dump({"train": kept_train, "valid": kept_valid, "hf_tokens": args.hf_tokens}, handle)
    print(f"train {len(train)} (pominięte za długie albo bez odpowiedzi: {dropped}), valid {len(valid)}", flush=True)

    if args.dry:
        prompt, reply = render(load(os.path.join(args.data, "train.jsonl"))[0]["messages"])
        print("--- szablon (koniec promptu i odpowiedź) ---")
        print(prompt[-300:] + reply)
        return

    # Resuming goes through Unsloth too: PeftModel.from_pretrained skips its
    # gradient checkpointing and a 7B model then runs out of 16 GB.
    model = FastLanguageModel.get_peft_model(
        model, r=args.rank, lora_alpha=args.rank, lora_dropout=0, bias="none",
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        use_gradient_checkpointing="unsloth", random_state=args.seed,
    )
    if args.resume:
        from peft import set_peft_model_state_dict
        from safetensors.torch import load_file
        result = set_peft_model_state_dict(model, load_file(os.path.join(args.resume, "adapter_model.safetensors")))
        if result.unexpected_keys:
            sys.exit(f"adapter z {args.resume} nie pasuje (rank?): {result.unexpected_keys[:3]}")
        print(f"wczytany adapter: {args.resume}", flush=True)

    steps = math.ceil(len(train) * args.epochs / args.accumulate)
    print(f"{steps} kroków po {args.accumulate} przykładów, lr {args.rate}, rank {args.rank}", flush=True)

    trainer = Trainer(
        model=model,
        train_dataset=Dataset.from_list(train),
        eval_dataset=Dataset.from_list(valid),
        data_collator=DataCollatorForSeq2Seq(tokenizer, padding=True, label_pad_token_id=-100),
        args=TrainingArguments(
            output_dir=os.path.join(args.out, "checkpoints"),
            per_device_train_batch_size=1, per_device_eval_batch_size=1, gradient_accumulation_steps=args.accumulate,
            num_train_epochs=args.epochs, learning_rate=args.rate, lr_scheduler_type="cosine", warmup_steps=max(1, round(steps * 0.03)),
            optim="adamw_8bit", weight_decay=0.0, bf16=torch.cuda.is_bf16_supported(), fp16=not torch.cuda.is_bf16_supported(),
            logging_steps=10, eval_strategy="steps", eval_steps=max(steps // 8, 25),
            save_steps=max(steps // args.save_parts, 50), save_total_limit=max(6, args.save_parts),  # 6: the end of epoch 1 must survive for the "h" model
            report_to="none", seed=args.seed,
        ),
    )
    if args.stop_at:
        # Keep the schedule of --epochs (warmup, cosine) but end the run early,
        # once the checkpoint at --stop-at epochs is written.
        from transformers import TrainerCallback

        stop_step = math.floor(steps * args.stop_at / args.epochs)

        class StopAt(TrainerCallback):
            def on_step_end(self, _args, state, control, **_kwargs):
                if state.global_step >= stop_step:
                    control.should_save, control.should_training_stop = True, True
                return control

        trainer.add_callback(StopAt())
        print(f"zatrzymanie po {args.stop_at} ep. (krok {stop_step} z {steps})", flush=True)

    started = time.time()
    trainer.train(resume_from_checkpoint=args.resume_from or None)
    print(f"trening: {(time.time() - started) / 60:.0f} min", flush=True)
    print("ewaluacja:", trainer.evaluate(), flush=True)

    adapter = os.path.join(args.out, "adapter")
    model.save_pretrained(adapter)
    tokenizer.save_pretrained(adapter)
    print(f"adapter: {adapter}", flush=True)

    if args.gguf:
        # Unsloth merges the adapter into 16-bit weights (needs ~24 GB RAM
        # and ~25 GB of disk) and runs llama.cpp's converter and quantizer.
        target = os.path.join(args.out, "gguf")
        model.save_pretrained_gguf(target, tokenizer, quantization_method=args.gguf)
        print(f"GGUF w {target}: skopiuj plik .gguf na Maca i zarejestruj tools/model/import.sh {args.name} <plik>", flush=True)

    if args.lora_gguf:
        print("adapter jako GGUF: uruchom tools/pc/lora-gguf.sh (konwerter llama.cpp)", flush=True)


if __name__ == "__main__":
    main()
