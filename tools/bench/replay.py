#!/usr/bin/env python3
"""Replays real conversations from the game through several models, so two
versions are judged on the same prompts the game actually sent.

The game logs every talk when NEURO_LOG_TALKS=1 (talks.jsonl in its data
directory). For each opening the player used (--keys, matched at the start
of the player's line, case ignored) the latest logged talk is taken, its
last reply dropped, and each model answers it --samples times.

    python3 tools/bench/replay.py neuro-npc-run6,neuro-npc-run7
    python3 tools/bench/replay.py neuro-npc-run7 --keys "co wiesz o krolu,plotki" --samples 5
"""
import argparse
import json
import os
import sys
import urllib.request

KEYS = ["siema mordo", "elo mordo", "co wiesz o krolu", "plotki", "jaka masz najlepsza zbroje", "dobre maja piwo",
        "czy dobre piwo", "skad te piwo", "skad maja tu piwo", "co mi powiesz o polowaniu", "siema dziadu"]


def data_dir():
    base = os.environ.get("XDG_CONFIG_HOME") or os.path.expanduser("~/Library/Application Support")
    return os.path.join(base, "neuro-processor")


def label(model):
    return model.removeprefix("neuro-npc-").split(":")[0]


def ask(url, model, messages, seed, options):
    body = {"model": model, "messages": messages, "stream": False, "options": {**options, "seed": seed}}
    request = urllib.request.Request(url.rstrip("/") + "/api/chat", json.dumps(body).encode(), {"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=300) as response:
        return json.load(response)["message"]["content"].strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("models", help="comma-separated Ollama models")
    parser.add_argument("--talks", default=os.path.join(data_dir(), "talks.jsonl"))
    parser.add_argument("--keys", default=",".join(KEYS), help="comma-separated openings of the player's line")
    parser.add_argument("--samples", type=int, default=3)
    parser.add_argument("--url", default="http://localhost:11434")
    args = parser.parse_args()

    models = args.models.split(",")
    keys = [key.strip().lower() for key in args.keys.split(",") if key.strip()]
    options = {"temperature": 0.8, "num_ctx": 4096, "num_predict": 220}

    picked = {}
    with open(args.talks, encoding="utf-8") as handle:
        for line in handle:
            if not line.strip():
                continue
            talk = json.loads(line)
            user = talk["messages"][-2]["content"].lower()
            for key in keys:
                if user.startswith(key):
                    picked[key] = talk

    for key in keys:
        if key not in picked:
            print(f"## {key} | brak w {args.talks}")
            continue
        messages = picked[key]["messages"][:-1]
        hint = messages[-1]["content"].split("Nastrój twojej postaci: ")[-1][:140]
        print("##", key, "|", hint)
        print(f"  gra powiedziała: {picked[key]['messages'][-1]['content']}")
        for model in models:
            for seed in range(args.samples):
                print(f"  {label(model)} {seed}: {ask(args.url, model, messages, 40 + seed, options)}")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
