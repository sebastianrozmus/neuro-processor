"""Training rows flagged by check.py -> step3/auto-flags.jsonl (file, line, issues, reply)."""
import json, os, sys
from check import check
out = open("step3/auto-flags.jsonl", "w", encoding="utf-8")
n_all = n_bad = 0
for path in sys.argv[1:]:
    for i, line in enumerate(open(path, encoding="utf-8")):
        msgs = json.loads(line)["messages"]
        player, _, gra = msgs[-2]["content"].partition("[GRA]")
        sc = {"system": msgs[0]["content"], "turns": [{"player": player.strip(), "gra": gra.strip()}]}
        issues = [x for x in check(sc, 0, msgs[-1]["content"], []) ]
        n_all += 1
        if issues:
            n_bad += 1
            out.write(json.dumps({"file": os.path.basename(path), "line": i, "issues": issues, "gra": gra.strip()[:300], "reply": msgs[-1]["content"]}, ensure_ascii=False) + "\n")
print(n_all, "wierszy,", n_bad, "oflagowanych")
