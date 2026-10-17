"""GPT's "chosen" replies + the DPO pack -> pairs for DPO training.

    python3 dpo_merge.py dpo-run9b-3a.jsonl extra-gpt-dpo-k1.jsonl [k2 ...] --out dpo-pairs.jsonl

Pack lines: {"id","context","rejected","why"}; GPT lines: {"id","chosen"} (chosen may be null).
Every chosen reply goes through check.py's rules for its own [GRA]; failing ones are
listed and left out. Output lines: {"prompt": [messages], "chosen": "...", "rejected": "...", "id"}.
"""
import argparse
import json

from check import check

parser = argparse.ArgumentParser()
parser.add_argument("pack")
parser.add_argument("gpt", nargs="+")
parser.add_argument("--out", required=True)
args = parser.parse_args()

with open(args.pack, encoding="utf-8") as handle:
    pack = {x["id"]: x for x in map(json.loads, handle)}
chosen = {}
for path in args.gpt:
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line:
                item = json.loads(line)
                chosen[item["id"]] = item.get("chosen")

kept = bad = skipped = 0
with open(args.out, "w", encoding="utf-8") as out:
    for key, item in pack.items():
        reply = chosen.get(key)
        if key not in chosen:
            continue
        if reply is None:
            skipped += 1
            continue
        ctx = item["context"]
        last = ctx[-1]["content"]
        player, _, gra = last.partition("[GRA]")
        scenario = {"system": ctx[0]["content"], "turns": [{"player": player.strip(), "gra": gra.strip()}]}
        previous = [m["content"] for m in ctx[1:-1] if m["role"] == "assistant"]
        issues = check(scenario, 0, reply, previous)
        if issues or reply.strip() == item["rejected"].strip():
            bad += 1
            print(f"{key}: {issues or 'chosen = rejected'}\n   {reply}")
            continue
        out.write(json.dumps({"id": key, "prompt": ctx, "chosen": reply, "rejected": item["rejected"]}, ensure_ascii=False) + "\n")
        kept += 1
missing = [k for k in pack if k not in chosen]
print(f"par: {kept}, odrzuconych przez automat: {bad}, chosen=null: {skipped}, bez odpowiedzi GPT: {len(missing)}")
