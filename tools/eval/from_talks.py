"""talks.jsonl from the game (NEURO_LOG_TALKS=1) -> scenarios: each logged reply
becomes one test with the history the game really sent and the reply to redo.

    python3 from_talks.py talks.jsonl game-scen.jsonl [--prefix g1018] [--notes game-notes.json]

Notes are {"<id>": "what a good reply does here"}; only talks with a note are
written when --notes is given, so a session's dull lines stay out of the set.
"ref" keeps what the model said in the game (the line that was judged).
"""
import argparse
import json

parser = argparse.ArgumentParser()
parser.add_argument("talks")
parser.add_argument("out")
parser.add_argument("--prefix", default="g")
parser.add_argument("--notes", default="")
args = parser.parse_args()

notes = {}
if args.notes:
    with open(args.notes, encoding="utf-8") as handle:
        notes = json.load(handle)

with open(args.talks, encoding="utf-8") as handle:
    talks = [json.loads(line) for line in handle if line.strip()]

kept = 0
with open(args.out, "w", encoding="utf-8") as out:
    for n, talk in enumerate(talks):
        key = f"{args.prefix}-{n:02d}"
        if args.notes and key not in notes:
            continue
        messages = talk["messages"]
        if messages[-1]["role"] != "assistant":
            continue
        player, _, gra = messages[-2]["content"].partition("[GRA]")
        scenario = {"id": key, "kind": "game", "keeper": talk["keeper"], "model": talk.get("model"), "system": messages[0]["content"]}
        if messages[0]["content"].startswith("Jesteś Mistrzem Gry"):
            scenario.update(gm=True, turns=[{"system": messages[0]["content"], "gra": gra.strip()}])
        else:
            scenario.update(history=messages[1:-2], turns=[{"player": player.strip() or None, "gra": gra.strip() or None}])
        scenario["ref"] = messages[-1]["content"]
        if key in notes:
            scenario["note"] = notes[key]
        out.write(json.dumps(scenario, ensure_ascii=False) + "\n")
        kept += 1
print(f"scenariuszy: {kept}")
