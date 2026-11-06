"""Plays scripted conversations against an Ollama model the way the game does
and saves every reply.

    python3 run.py --model neuro-npc-run7 --scenarios scenarios.jsonl --out out/run7.jsonl

A scenario is {"id", "kind", "system", "turns": [{"player", "gra"}], "history"?}.
History sent to the model keeps the earlier player lines without their [GRA]
block, as the game does; only the current line carries it. Like the game
(internal/game/session.go, historyLimit), only the last 12 lines of history
go with a turn (--history). A Game Master fight is a list of rounds, each
sent alone with its own system (state of the fight changes every round):
turns [{"system", "gra"}].

Options as the game sends them to the talk model with NEURO_TALK_MODEL set
and NEURO_TALK_CONTEXT unset (the setup in README.pl.md): no num_ctx, so the
Modelfile's 4096 applies (--num-ctx, sent explicitly here); max_tokens from
NEURO_LLM_MAX_TOKENS, 220 in the README (--num-predict). The game sends no
temperature, top_p or seed, leaving them to Ollama; here they stay fixed
(--temperature, top_p 0.9, --seed) so that runs can be compared.

--warn-ctx reports every turn whose prompt (Ollama's prompt_eval_count) and
--num-predict together do not fit --num-ctx: there Ollama cuts the context
once the reply runs long. Every reply keeps its prompt_eval_count as "prompt".
"""
import argparse
import json
import os
import sys
import time
import urllib.request

HOST = os.environ.get("OLLAMA_URL", "http://192.168.1.156:11434")

# HISTORY_LIMIT is the game's historyLimit: lines of talk a turn carries.
HISTORY_LIMIT = 12


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


def recent(history, limit):
    """The last lines of talk the game would send; all of them for limit 0."""
    return history[-limit:] if limit > 0 else history


def overflow(scenario, turn, data, options):
    """Warns when the prompt and the longest reply allowed overrun the window."""
    window, predict = options.get("num_ctx", 0), options.get("num_predict", 0)
    prompt = data.get("prompt_eval_count", 0)
    if window > 0 and prompt + predict > window:
        print(f"UWAGA okno: {scenario['id']} tura {turn}: prompt {prompt} + num_predict {predict} > num_ctx {window}", file=sys.stderr, flush=True)


def play(model, scenario, options, limit, warn):
    replies = []
    if scenario.get("gm"):
        for n, turn in enumerate(scenario["turns"]):
            messages = [{"role": "system", "content": turn["system"]}, {"role": "user", "content": user_line(None, turn["gra"])}]
            reply, data = chat(model, messages, options)
            if warn:
                overflow(scenario, n, data, options)
            replies.append({"gra": turn["gra"], "reply": reply, "s": round(data["total_duration"] / 1e9, 2), "prompt": data.get("prompt_eval_count")})
        return replies
    system = [{"role": "system", "content": scenario["system"]}]
    history = list(scenario.get("history", []))
    for n, turn in enumerate(scenario["turns"]):
        messages = system + recent(history, limit) + [{"role": "user", "content": user_line(turn.get("player"), turn.get("gra"))}]
        reply, data = chat(model, messages, options)
        if warn:
            overflow(scenario, n, data, options)
        replies.append({"player": turn.get("player"), "gra": turn.get("gra"), "reply": reply, "s": round(data["total_duration"] / 1e9, 2), "prompt": data.get("prompt_eval_count")})
        history = history + [{"role": "user", "content": turn.get("player") or ""}, {"role": "assistant", "content": reply}]
    return replies


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--scenarios", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--temperature", type=float, default=0.7)
    parser.add_argument("--seed", type=int, default=7)
    parser.add_argument("--history", type=int, default=HISTORY_LIMIT, help="lines of talk sent with a turn, as the game's historyLimit; 0 sends all")
    parser.add_argument("--num-ctx", type=int, default=4096, help="context window; 4096 is the Modelfile's, the game's talk model runs in it; 0 leaves it to the server")
    parser.add_argument("--num-predict", type=int, default=220, help="reply cap, as NEURO_LLM_MAX_TOKENS in the README")
    parser.add_argument("--warn-ctx", action="store_true", help="warn about turns whose prompt_eval_count + num_predict exceeds num_ctx")
    parser.add_argument("--repeat-penalty", type=float, default=None, help="Ollama's repeat_penalty; unset leaves it to the Modelfile (1.1 is Ollama's default, the game sends none)")
    args = parser.parse_args()
    options = {"temperature": args.temperature, "top_p": 0.9, "seed": args.seed, "num_predict": args.num_predict}
    if args.num_ctx > 0:
        options["num_ctx"] = args.num_ctx
    if args.repeat_penalty is not None:
        options["repeat_penalty"] = args.repeat_penalty

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
        replies = play(args.model, scenario, options, args.history, args.warn_ctx)
        with open(args.out, "a", encoding="utf-8") as handle:
            handle.write(json.dumps({"id": scenario["id"], "model": args.model, "replies": replies}, ensure_ascii=False) + "\n")
        print(f"{n}/{len(scenarios)} {scenario['id']} {time.time() - started:.0f}s", flush=True)


if __name__ == "__main__":
    main()
