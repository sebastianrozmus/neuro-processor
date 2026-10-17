"""Bad replies of a model on the scenarios -> DPO pack for GPT (ZAMOWIENIE-run9.md, step 3a).

    python3 dpo_pack.py out/scen-run9b.checked.jsonl dpo-notes-run9b.json out/dpo-run9b-3a.jsonl

Each line: {"id", "context", "rejected", "why"}. "context" is exactly what the
model saw: system, history with its own earlier replies (player lines without
[GRA]) and the last player line with [GRA]. "why" comes from the hand notes
(dpo-notes-*.json: {"<scenario id>#<turn from 1>": "why"}) and from check.py.
"""
import json
import sys

from run import user_line

AUTO = {
    "liczba-slownie": "liczba słownie zamiast cyframi",
    "zdania": "więcej niż 3 zdania",
    "zly-znacznik": "znacznik spoza listy 8 nastrojów",
    "liczba-spoza": "liczba spoza [GRA] i cennika",
    "brak-kwoty": "nie podaje kwoty z [GRA]",
    "kwota-zakazana": "podaje kwotę, choć [GRA] zabrania",
    "GM:zmyslona-smierc": "opisuje śmierć albo upadek, którego nie ma w [GRA]",
    "nastroj": "znacznik sprzeczny z nastrojem z [GRA]",
}

results_path, notes_path, out_path = sys.argv[1:]
with open("scenarios.jsonl", encoding="utf-8") as handle:
    scenarios = {s["id"]: s for s in map(json.loads, handle)}
with open(notes_path, encoding="utf-8") as handle:
    notes = json.load(handle)

count = 0
with open(results_path, encoding="utf-8") as handle, open(out_path, "w", encoding="utf-8") as out:
    for res in map(json.loads, handle):
        scenario = scenarios[res["id"]]
        history = []
        for idx, (turn, rep) in enumerate(zip(scenario["turns"], res["replies"]), 1):
            key = f"{res['id']}#{idx}"
            if scenario.get("gm"):
                context = [{"role": "system", "content": turn["system"]}, {"role": "user", "content": user_line(None, turn["gra"])}]
            else:
                context = [{"role": "system", "content": scenario["system"]}] + history + [
                    {"role": "user", "content": user_line(turn.get("player"), turn.get("gra"))}]
                history = history + [{"role": "user", "content": turn.get("player") or ""}, {"role": "assistant", "content": rep["reply"]}]
            why = []
            if key in notes:
                why.append(notes[key])
            for issue in ([] if key in notes else rep.get("issues", [])):
                name = issue if issue.startswith("GM:") else issue.split(":")[0]
                text = AUTO.get(name)
                if text and text not in why and not (name == "GM:zmyslona-smierc" and key in notes):
                    why.append(text + (f" ({issue.split(':', 1)[1]})" if ":" in issue and not issue.startswith("GM:") else ""))
            if why:
                out.write(json.dumps({"id": key, "context": context, "rejected": rep["reply"], "why": "; ".join(why)}, ensure_ascii=False) + "\n")
                count += 1
print(f"{count} linii -> {out_path}")
