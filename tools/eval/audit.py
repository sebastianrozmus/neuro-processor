"""Runs check.py's rules over the reference replies of the training data and
counts the stiff, clerk-like phrases the model picks up from it.

    python3 audit.py ~/neuro-train/data/train.jsonl ~/neuro-train/data/valid.jsonl
"""
import json
import re
import sys
from collections import Counter

from check import check

STIFF = [
    r"to propozycja, nie", r"nie zakończony zakup", r"nie jest to już", r"rozliczenie (jest )?zamknięte", r"to ceny sprzedaży",
    r"to koniec targowania", r"zakup nie dochodzi do skutku", r"nie dochodzi do sprzedaży", r"transakcj\w+", r"zapłata przyjęta",
    r"to moja oferta", r"ten interes jest zakończony", r"nie oznacza", r"nie jest potwierdzeniem", r"to już zapłata",
    r"nie zmienia stanu", r"określ", r"wynosi \d+", r"dostępność",
]

rows = []
for path in sys.argv[1:]:
    with open(path, encoding="utf-8") as handle:
        rows += [json.loads(line) for line in handle if line.strip()]

issues, stiff, roles, examples = Counter(), Counter(), Counter(), {}
short = 0
for n, row in enumerate(rows):
    msgs = row["messages"]
    player, _, gra = msgs[-2]["content"].partition("[GRA]")
    scenario = {"system": msgs[0]["content"], "turns": [{"player": player.strip(), "gra": gra.strip()}]}
    reply = msgs[-1]["content"]
    found = check(scenario, 0, reply, [m["content"] for m in msgs[1:-1] if m["role"] == "assistant"])
    role = msgs[0]["content"][:20]
    for issue in found:
        key = issue.split(":")[0]
        issues[key] += 1
        roles[(role, key)] += 1
        examples.setdefault(key, []).append((n, gra.strip()[:200], reply))
    low = reply.lower()
    for pattern in STIFF:
        if re.search(pattern, low):
            stiff[pattern] += 1
    if len(re.findall(r"\w+", reply)) <= 6:
        short += 1

print(f"{len(rows)} wierszy; odpowiedzi z uwagami: {sum(1 for _ in examples.values())} typów")
for key, count in issues.most_common():
    print(f"  {key:22s} {count}")
print(f"bardzo krótkich (≤6 słów): {short}")
print("sztywne frazy:")
for key, count in stiff.most_common():
    print(f"  {count:4d}  {key}")
print("\nprzykłady:")
for key, items in examples.items():
    print(f"--- {key} ({len(items)})")
    for n, gra, reply in items[:4]:
        print(f"   [{n}] G: {gra}\n       R: {reply}")
