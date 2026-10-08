"""Merge rewritten replies into recorded talks.

Usage: python3 apply.py IN.jsonl FIXES.jsonl OUT.jsonl

FIXES holds lines {"id": N, "reply": "..."} where N is the line number in
IN (as digest.py numbered them). Lines with a fix get the new reply; lines
without one are copied unchanged, so a partial pass still helps.
"""
import json
import sys

MOODS = ("[spokój]", "[radość]", "[gniew]", "[smutek]", "[strach]", "[kpina]", "[zdziwienie]", "[czułość]")


def main():
    src, fixes, dst = sys.argv[1], sys.argv[2], sys.argv[3]
    replies = {}
    with open(fixes, encoding="utf-8") as handle:
        for line in handle:
            if line.strip():
                fix = json.loads(line)
                reply = fix["reply"].strip()
                if not reply.lower().startswith(MOODS):
                    print(f"id {fix['id']}: brak znacznika nastroju, pomijam")
                    continue
                replies[int(fix["id"])] = reply

    changed = 0
    with open(src, encoding="utf-8") as handle, open(dst, "w", encoding="utf-8") as out:
        for number, line in enumerate(handle):
            if not line.strip():
                continue
            row = json.loads(line)
            if number in replies:
                row["messages"][-1]["content"] = replies[number]
                row["source"] = row.get("source", "") + "+rewritten"
                changed += 1
            out.write(json.dumps(row, ensure_ascii=False) + "\n")
    print(f"poprawiono {changed} z {len(replies)} podanych; wynik: {dst}")


if __name__ == "__main__":
    main()
