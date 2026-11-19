"""Preference pairs from the model's own replies: it answers the training
prompts a few times, check.py and a few stricter rules sort the answers, and
a clean answer is paired with a faulty one of about the same length.

    python3 dpo_self.py sample --model neuro-npc-run16 --data train.jsonl --out samples.jsonl [--k 4] [--limit 1500]
    python3 dpo_self.py play --model neuro-npc-run16 --data scenarios.jsonl --out samples.jsonl [--k 4]
    python3 dpo_self.py pairs --samples samples.jsonl --out pairs.jsonl

The pairs teach what the faults are, not that longer is better: run 13 paired
fixes that added missing facts with the replies they fixed, and DPO learnt
length. Here both replies come from the same model at the same temperature,
and a pair is kept only when the two are about as long.
"""
import argparse
import json
import random
import re
import time

from check import check
from run import chat

# What a reply may not do, beyond check.py: these come from the blind reviews
# of runs 14 and 15 (the wage in every line, a fight retold at length).
LONG = 200
TOO_SHORT = 25


def scenario_of(messages):
    """The check.py scenario for the last turn of a prompt."""
    system = messages[0]["content"]
    player, _, gra = messages[-1]["content"].partition("[GRA]")
    gm = system.startswith("Jesteś Mistrzem Gry")
    return {"system": system, "gm": gm, "turns": [{"player": player.strip(), "gra": gra.strip(), "system": system}]}


def faults(messages, reply):
    scenario = scenario_of(messages)
    previous = [m["content"] for m in messages[1:] if m["role"] == "assistant"]
    issues = check(scenario, 0, reply, previous)
    gra = scenario["turns"][0]["gra"]
    speech = re.sub(r"^\[\w+\]\s*", "", reply.strip())
    # A mercenary asked about anything but the fee does not quote it.
    if "O cenie służby mów tylko, gdy o nią pyta" in gra:
        fee = re.search(r"Twoja służba kosztuje: «[^»]+» (\d+)", scenario["system"])
        if fee and re.search(rf"\b{fee.group(1)}\b", speech):
            issues.append("cena-bez-pytania")
    if len(speech) > LONG:
        issues.append("dlugie")
    if len(speech) < TOO_SHORT and not scenario["gm"]:
        issues.append("krotkie")
    if scenario["gm"] and lost_left_out(gra, speech):
        issues.append("GM:pominieta-liczba")
    return issues


def lost_left_out(gra, speech):
    """A narration that skips a blow the rules dealt: the blind judges of run 17
    found this the most common heavy fault of the master (a hit with its number
    gone, "Szczur gryzie" with no 6). A lost number may not be left out even
    though the prompt lets the master skip numbers: a judge reads it as a hit
    that did not happen."""
    lost = re.findall(r"traci (\d+) zdrowia", gra)
    said = set(re.findall(r"\d+", speech))
    return any(n not in said for n in lost)


def sample(args):
    rows = [json.loads(line) for line in open(args.data, encoding="utf-8") if line.strip()]
    rng = random.Random(16)
    rng.shuffle(rows)
    rows = rows[: args.limit] if args.limit else rows
    done = set()
    try:
        done = {json.loads(line)["n"] for line in open(args.out, encoding="utf-8") if line.strip()}
    except FileNotFoundError:
        pass
    started = time.time()
    with open(args.out, "a", encoding="utf-8") as out:
        for n, row in enumerate(rows):
            if n in done:
                continue
            prompt = row["messages"][:-1]
            replies = []
            for k in range(args.k):
                options = {"temperature": args.temperature, "top_p": 0.95, "seed": 1000 + 17 * n + k, "num_predict": 160, "num_ctx": 4096}
                reply, _ = chat(args.model, prompt, options)
                replies.append(reply)
            out.write(json.dumps({"n": n, "prompt": prompt, "gold": row["messages"][-1]["content"], "replies": replies}, ensure_ascii=False) + "\n")
            out.flush()
            if n % 20 == 0:
                print(f"{n + 1}/{len(rows)} {time.time() - started:.0f}s", flush=True)


def play(args):
    """Scenarios (as run.py takes them) played by the model on its own history:
    the first sample of each turn goes on, the others are the alternatives.
    Repetition and drift only show on the model's own history."""
    from run import user_line

    scenarios = [json.loads(line) for line in open(args.data, encoding="utf-8") if line.strip()]
    done = set()
    try:
        done = {json.loads(line)["n"] for line in open(args.out, encoding="utf-8") if line.strip()}
    except FileNotFoundError:
        pass
    started = time.time()
    with open(args.out, "a", encoding="utf-8") as out:
        for s, scenario in enumerate(scenarios):
            gm = bool(scenario.get("gm")) or (scenario.get("system") or "").startswith("Jesteś Mistrzem Gry")
            history = [{"role": "system", "content": scenario.get("system", "")}]
            for t, turn in enumerate(scenario["turns"]):
                n = f"{scenario['id']}#{t}"
                if gm:
                    prompt = [{"role": "system", "content": turn.get("system") or scenario["system"]}, {"role": "user", "content": user_line(None, turn["gra"])}]
                else:
                    prompt = history + [{"role": "user", "content": user_line(turn.get("player"), turn.get("gra"))}]
                replies = []
                for k in range(args.k):
                    options = {"temperature": args.temperature, "top_p": 0.95, "seed": 7000 + 31 * s + 7 * t + k, "num_predict": 160, "num_ctx": 4096}
                    replies.append(chat(args.model, prompt, options)[0])
                if n not in done:
                    out.write(json.dumps({"n": n, "prompt": prompt, "gold": None, "replies": replies}, ensure_ascii=False) + "\n")
                    out.flush()
                if not gm:
                    history = history + [{"role": "user", "content": turn.get("player") or ""}, {"role": "assistant", "content": replies[0]}]
            print(f"{s + 1}/{len(scenarios)} {scenario['id']} {time.time() - started:.0f}s", flush=True)


def close(a, b):
    """Two replies about as long: within a quarter, or 25 characters."""
    return abs(len(a) - len(b)) <= max(25, 0.25 * max(len(a), len(b)))


def pairs(args):
    from collections import Counter

    kinds, kept, lengths = Counter(), [], [0, 0]
    for line in open(args.samples, encoding="utf-8"):
        item = json.loads(line)
        prompt = item["prompt"]
        judged = [(reply, faults(prompt, reply)) for reply in dict.fromkeys(item["replies"])]
        clean = [reply for reply, issues in judged if not issues]
        bad = [(reply, issues) for reply, issues in judged if issues]
        # The gold reply is clean by construction; it stands in when no sample is,
        # but only against a faulty sample of its own length.
        if not clean and item["gold"] and not faults(prompt, item["gold"]):
            clean = [item["gold"]]
        for reply, issues in bad:
            match = [c for c in clean if close(c, reply)]
            if not match:
                continue
            chosen = min(match, key=lambda c: abs(len(c) - len(reply)))
            kind = issues[0].split(":")[0]
            if kinds[kind] >= args.per_kind:
                continue
            kinds[kind] += 1
            lengths[0] += len(chosen)
            lengths[1] += len(reply)
            kept.append({"prompt": prompt, "chosen": chosen, "rejected": reply, "why": issues})
            break
    random.Random(16).shuffle(kept)
    with open(args.out, "w", encoding="utf-8") as out:
        for pair in kept:
            out.write(json.dumps(pair, ensure_ascii=False) + "\n")
    n = max(len(kept), 1)
    print(f"par: {len(kept)}; śr. długość chosen {lengths[0] / n:.0f}, rejected {lengths[1] / n:.0f}")
    for kind, count in kinds.most_common():
        print(f"  {kind:22s} {count}")


def main():
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("sample")
    s.add_argument("--model", required=True)
    s.add_argument("--data", required=True)
    s.add_argument("--out", required=True)
    s.add_argument("--k", type=int, default=4)
    s.add_argument("--limit", type=int, default=0)
    s.add_argument("--temperature", type=float, default=0.9)
    g = sub.add_parser("play")
    g.add_argument("--model", required=True)
    g.add_argument("--data", required=True, help="scenarios as run.py takes them")
    g.add_argument("--out", required=True)
    g.add_argument("--k", type=int, default=4)
    g.add_argument("--temperature", type=float, default=0.9)
    p = sub.add_parser("pairs")
    p.add_argument("--samples", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--per-kind", type=int, default=400)
    args = parser.parse_args()
    {"sample": sample, "play": play, "pairs": pairs}[args.cmd](args)


if __name__ == "__main__":
    main()
