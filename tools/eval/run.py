"""Plays scripted conversations against an Ollama model the way the game does
and saves every reply.

    python3 run.py --model neuro-npc-run7 --scenarios scenarios.jsonl --out out/run7.jsonl

A scenario is {"id", "kind", "system", "turns": [{"player", "gra"}], "history"?}.
History sent to the model keeps the earlier player lines without their [GRA]
block, as the game does; only the current line carries it. A Game Master
fight is a list of rounds, each sent alone with its own system (state of the
fight changes every round): turns [{"system", "gra"}].
"""
import argparse
import json
import os
import time
import urllib.request

HOST = os.environ.get("OLLAMA_URL", "http://192.168.1.156:11434")


def chat(model, messages, options):
    body = json.dumps({"model": model, "messages": messages, "stream": False, "options": options}).encode()
    req = urllib.request.Request(HOST + "/api/chat", body, {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as resp:
        data = json.loads(resp.read())
    return data["message"]["content"].strip(), data


def user_line(player, gra):
    if player and gra:
        return f"{player}\n\n[GRA] {gra}"
    if gra:
        return f"[GRA] {gra}"
    return player


def play(model, scenario, options):
    replies = []
    if scenario.get("gm"):
        for turn in scenario["turns"]:
            messages = [{"role": "system", "content": turn["system"]}, {"role": "user", "content": user_line(None, turn["gra"])}]
            reply, data = chat(model, messages, options)
            replies.append({"gra": turn["gra"], "reply": reply, "s": round(data["total_duration"] / 1e9, 2)})
        return replies
    history = [{"role": "system", "content": scenario["system"]}] + scenario.get("history", [])
    for turn in scenario["turns"]:
        messages = history + [{"role": "user", "content": user_line(turn.get("player"), turn.get("gra"))}]
        reply, data = chat(model, messages, options)
        replies.append({"player": turn.get("player"), "gra": turn.get("gra"), "reply": reply, "s": round(data["total_duration"] / 1e9, 2)})
        history = history + [{"role": "user", "content": turn.get("player") or ""}, {"role": "assistant", "content": reply}]
    return replies


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--scenarios", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--temperature", type=float, default=0.7)
    parser.add_argument("--seed", type=int, default=7)
    args = parser.parse_args()
    options = {"temperature": args.temperature, "top_p": 0.9, "seed": args.seed, "num_predict": 220, "num_ctx": 4096}

    done = set()
    if os.path.exists(args.out):
        with open(args.out, encoding="utf-8") as handle:
            done = {json.loads(line)["id"] for line in handle if line.strip()}
    with open(args.scenarios, encoding="utf-8") as handle:
        scenarios = [json.loads(line) for line in handle if line.strip()]
    os.makedirs(os.path.dirname(os.path.abspath(args.out)), exist_ok=True)
    started = time.time()
    for n, scenario in enumerate(scenarios, 1):
        if scenario["id"] in done:
            continue
        replies = play(args.model, scenario, options)
        with open(args.out, "a", encoding="utf-8") as handle:
            handle.write(json.dumps({"id": scenario["id"], "model": args.model, "replies": replies}, ensure_ascii=False) + "\n")
        print(f"{n}/{len(scenarios)} {scenario['id']} {time.time() - started:.0f}s", flush=True)


if __name__ == "__main__":
    main()
