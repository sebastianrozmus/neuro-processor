"""Checks gold replies with check.py's rules.

    python3 validate_gold.py gold/G1.json
"""
import json
import sys

from check import check

with open("scenarios.jsonl", encoding="utf-8") as handle:
    scenarios = {s["id"]: s for s in map(json.loads, handle)}
bad = 0
for path in sys.argv[1:]:
    with open(path, encoding="utf-8") as handle:
        gold = json.load(handle)
    for sid, item in gold.items():
        replies = item["gold"] if isinstance(item, dict) else item
        turns = scenarios[sid]["turns"]
        if len(replies) != len(turns):
            print(f"{sid}: {len(replies)} odpowiedzi na {len(turns)} tur")
            bad += 1
            continue
        for idx, reply in enumerate(replies):
            issues = check(scenarios[sid], idx, reply, replies[:idx])
            if issues:
                bad += 1
                print(f"{sid} tura {idx + 1}: {issues}\n   {reply}")
print("OK" if not bad else f"{bad} uwag")
