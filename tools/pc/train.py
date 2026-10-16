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


def load(path):
    with open(path, encoding="utf-8") as handle:
        return [json.loads(line) for line in handle if line.strip()]


def encode(rows, tokenizer, limit):
    """Tokens and labels: the prompt is masked (-100), the reply is learnt.
    Rows longer than the limit are dropped, not cut, so no reply is
    learnt without its prompt."""
    encoded, dropped = [], 0
    for row in rows:
        messages = row["messages"]
        if not messages or messages[-1]["role"] != "assistant":
            dropped += 1
            continue
        prompt, reply = render(messages)
        prompt_ids = tokenizer(prompt, add_special_tokens=True)["input_ids"]
        reply_ids = tokenizer(reply, add_special_tokens=False)["input_ids"]
        ids = prompt_ids + reply_ids
        if len(ids) > limit:
            dropped += 1
            continue
        encoded.append({"input_ids": ids, "attention_mask": [1] * len(ids), "labels": [-100] * len(prompt_ids) + reply_ids})
    return encoded, dropped


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
    parser.add_argument("--gguf", default="", help="also write the whole model as GGUF: q5_k_m, q8_0, q4_k_m")
    parser.add_argument("--lora-gguf", action="store_true", help="also write the adapter alone as GGUF")
    parser.add_argument("--dry", action="store_true", help="only check the data and the template, no training")
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

    train, dropped = encode(load(os.path.join(args.data, "train.jsonl")), tokenizer, args.max_len)
    valid, _ = encode(load(os.path.join(args.data, "valid.jsonl")), tokenizer, args.max_len)
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
        use_gradient_checkpointing="unsloth", random_state=7,
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
            logging_steps=10, eval_strategy="steps", eval_steps=max(steps // 8, 25), save_steps=max(steps // 4, 50), save_total_limit=3,
            report_to="none", seed=7,
        ),
    )

    started = time.time()
    trainer.train()
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
