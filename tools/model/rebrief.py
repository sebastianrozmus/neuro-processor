"""Rewrite the small-talk [GRA] brief of mercenary, companion and witness rows
to the one the game sends now (pulled from internal/game/narration.go).

Usage: python3 -I rebrief.py NARRATION_GO IN.jsonl OUT.jsonl
"""
import json, re, sys

go = open(sys.argv[1], encoding="utf-8").read()

def const(name):
    m = re.search(name + r'\s*=\s*"(.*?)"\n', go)
    return m.group(1)

def brief(name):
    m = re.search(name + r':\s*((?:"[^"]*"\s*\+?\s*)+),', go, re.S)
    return "".join(re.findall(r'"([^"]*)"', m.group(1)))

ROLES = {const("RoleMercenary"): brief("RoleMercenary"), const("RoleCompanion"): brief("RoleCompanion"), const("RoleWitness"): brief("RoleWitness")}
OLD = ("Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. "
       "Jeśli pyta o towar, możesz polecić coś ze swojego towaru i podać jego cenę z listy. W tej kwestii niczego nie sprzedajesz.")

changed = kept = 0
with open(sys.argv[2], encoding="utf-8") as src, open(sys.argv[3], "w", encoding="utf-8") as dst:
    for line in src:
        if not line.strip():
            continue
        row = json.loads(line)
        system = row["messages"][0]["content"]
        new = next((b for role, b in ROLES.items() if system.startswith(role)), None)
        if new:
            for m in row["messages"]:
                if m["role"] == "user" and OLD in m["content"]:
                    m["content"] = m["content"].replace(OLD, new)
                    changed += 1
        else:
            kept += 1
        dst.write(json.dumps(row, ensure_ascii=False) + "\n")
print(f"briefs rewritten: {changed}, shopkeeper rows untouched: {kept}")
