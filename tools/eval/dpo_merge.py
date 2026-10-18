"""GPT's "chosen" replies + the DPO pack -> pairs for DPO training.

    python3 dpo_merge.py dpo-run9b-3a.jsonl extra-gpt-dpo-k1.jsonl [k2 ...] --out dpo-pairs.jsonl

Pack lines: {"id","context","rejected","why"}; GPT lines: {"id","chosen"} (chosen may be null).
Every chosen reply goes through check.py's rules for its own [GRA]; failing ones are
listed and left out. Output lines: {"prompt": [messages], "chosen": "...", "rejected": "...", "id"}.
"""
import argparse
import json
import re

from check import TAGS, check

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

TAG = re.compile(r"\[(\w+)\]")
kept = bad = skipped = retagged = 0
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
        # GPT tends to "calm" a valid tag ([zdziwienie] -> [spokój]) even when "why" says nothing about it;
        # DPO then learns that every emotion is a fault (run 10c lost surprise, fear and sadness). The model's
        # own tag goes back when it is one of the 8 and still fits the mood from [GRA].
        old_tag, new_tag = TAG.match(item["rejected"]), TAG.match(reply)
        if (old_tag and new_tag and old_tag.group(1) != new_tag.group(1) and old_tag.group(1) in TAGS
                and not re.search(r"znacznik|nastr", item["why"])):
            restored = old_tag.group(0) + reply[new_tag.end():]
            if not check(scenario, 0, restored, previous):
                reply = restored
                retagged += 1
        issues = check(scenario, 0, reply, previous)
        if issues or reply.strip() == item["rejected"].strip():
            bad += 1
            print(f"{key}: {issues or 'chosen = rejected'}\n   {reply}")
            continue
        out.write(json.dumps({"id": key, "prompt": ctx, "chosen": reply, "rejected": item["rejected"]}, ensure_ascii=False) + "\n")
        kept += 1
missing = [k for k in pack if k not in chosen]
print(f"par: {kept}, odrzuconych przez automat: {bad}, chosen=null: {skipped}, bez odpowiedzi GPT: {len(missing)}, przywrócony znacznik modelu: {retagged}")
