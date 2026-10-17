"""Gold replies for the scenarios -> training rows in pack.sh format.

    python3 gold_to_train.py scenarios.jsonl gold/ step3/train-scenarios.jsonl

gold/*.json: {"<scenario id>": ["<gold reply for turn 1>", ...], ...}
Every turn becomes one row: the history holds earlier player lines without
their [GRA] block and the gold replies, the last player line carries [GRA].
Game Master rounds are single rows (each round has its own state).
"""
import glob
import json
import os
import sys

from run import user_line

scen_path, gold_dir, out_path = sys.argv[1:]
gold = {}
for path in sorted(glob.glob(os.path.join(gold_dir, "*.json"))):
    with open(path, encoding="utf-8") as handle:
        gold.update(json.load(handle))

rows = 0
os.makedirs(os.path.dirname(out_path) or ".", exist_ok=True)
with open(scen_path, encoding="utf-8") as handle, open(out_path, "w", encoding="utf-8") as out:
    for scenario in map(json.loads, handle):
        replies = gold.get(scenario["id"])
        if isinstance(replies, dict):
            replies = replies.get("gold")
        if not replies:
            print("brak złotych odpowiedzi:", scenario["id"])
            continue
        if len(replies) != len(scenario["turns"]):
            sys.exit(f"{scenario['id']}: {len(replies)} odpowiedzi na {len(scenario['turns'])} tur")
        history = []
        for turn, reply in zip(scenario["turns"], replies):
            if scenario.get("gm"):
                messages = [{"role": "system", "content": turn["system"]}, {"role": "user", "content": user_line(None, turn["gra"])}]
            else:
                messages = [{"role": "system", "content": scenario["system"]}] + history + [
                    {"role": "user", "content": user_line(turn.get("player"), turn.get("gra"))}]
                history = history + [{"role": "user", "content": turn.get("player") or ""}, {"role": "assistant", "content": reply}]
            out.write(json.dumps({"messages": messages + [{"role": "assistant", "content": reply}]}, ensure_ascii=False) + "\n")
            rows += 1
print(f"{rows} wierszy -> {out_path}")
