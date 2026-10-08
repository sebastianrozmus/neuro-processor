"""Condense recorded talks for rewriting by a stronger model (or a person).

Each shop exchange becomes one compact JSON line with what a better reply
has to respect: the keeper's character, the wares and prices, the last
lines of the talk, the player's line and the [GRA] block. The master's
narrations (outcome "scene") are left out: they need no rewriting.

Usage: python3 digest.py IN.jsonl OUT.jsonl
Then: write {"id": N, "reply": "..."} lines and merge them with apply.py.
"""
import json
import re
import sys


def between(text, start, end):
    begin = text.find(start)
    if begin < 0:
        return ""
    begin += len(start)
    stop = text.find(end, begin) if end else -1
    return text[begin:stop if stop >= 0 else None].strip()


def main():
    src, dst = sys.argv[1], sys.argv[2]
    kept = 0
    with open(src, encoding="utf-8") as handle, open(dst, "w", encoding="utf-8") as out:
        for number, line in enumerate(handle):
            if not line.strip():
                continue
            row = json.loads(line)
            if row.get("outcome") == "scene":
                continue
            messages = row["messages"]
            system = messages[0]["content"]
            history = [m for m in messages[1:-2]]
            user = messages[-2]["content"]
            said, _, game = user.partition("[GRA]")
            digest = {
                "id": number,
                "keeper": row.get("keeper", ""),
                "persona": between(system, "TWOJA POSTAĆ:", "\n\n"),
                "wares": between(system, "Twój towar:", "\n"),
                "hero": between(system, "Bohater przed tobą:", "\n"),
                "history": [("gracz: " if m["role"] == "user" else "kupiec: ") + re.sub(r"\n\n\[GRA\].*", "", m["content"], flags=re.S)[:160] for m in history[-2:]],
                "said": said.strip(),
                "gra": game.strip(),
                "old": messages[-1]["content"],
            }
            out.write(json.dumps(digest, ensure_ascii=False) + "\n")
            kept += 1
    print(f"{kept} wymian do poprawy w {dst}")


if __name__ == "__main__":
    main()
