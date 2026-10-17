"""test.jsonl from pack.sh -> scenarios (history fixed, one reply to generate)."""
import json
import sys

src, dst = sys.argv[1], sys.argv[2]
with open(src, encoding="utf-8") as handle, open(dst, "w", encoding="utf-8") as out:
    for n, line in enumerate(handle, 1):
        messages = json.loads(line)["messages"]
        last = messages[-2]["content"]
        player, _, gra = last.partition("[GRA]")
        out.write(json.dumps({
            "id": f"test-{n:02d}", "kind": "test", "system": messages[0]["content"],
            "history": messages[1:-2],
            "turns": [{"player": player.strip() or None, "gra": gra.strip() or None}],
            "ref": messages[-1]["content"],
        }, ensure_ascii=False) + "\n")
