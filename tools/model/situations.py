"""Order B of run 11 (ZAMOWIENIE-run11.md, "Sytuacje z gry"): merges GPT's packs into one
training file, without the facts GPT tailored to the player's question.

    python3 situations.py extra-gpt-run11-b-k*.jsonl --out extra-b.jsonl

The game's CO WIESZ holds facts about the world, places, people and clues; never "Wiesz: broń.
X uderza najmocniej" or "Wiesz: sprzęt. Mokra cięciwa twojej kuszy...". Such facts are cut out
of SYSTEM. When the trained (last) reply leans on one (says a word that only that fact gave),
the whole conversation goes (two such words, or the fact's superlative), since the model would learn to say what the game never tells it.
A fact about a person under a generic subject ("Wiesz: pomoc. Ruta w zielarni...") gets the
person's name as subject, the way the game writes it. Last replies that fail tools/eval/check.py
go too. Run rebrief.py and tools/bench/check_data.py on the result.
"""
import json
import os
import re
import sys
import unicodedata
from collections import Counter

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "eval"))
from check import check  # noqa: E402

CUT = os.environ.get("NEURO_SHOW_CUT") == "1"  # print every fact cut out

# subjects of facts about wares and the keeper's own gear, which the game never gives
TAILORED = re.compile(
    r"^(bro[nń]|uzbrojenie|sprz[eę]t|ekwipunek|ma[sś][cć]|ma[sś]ci|banda[zż]e?|opatrunki?|plaster|plastry|"
    r"zio[lł]a|[sś]wiat[lł]o|leki|lekarstwa|towar|pancerz|amunicja|[lł]uk|kusza|rzemios[lł]o|[lł]ata|mikstury?)$")
GENERIC = re.compile(r"^(pomoc|serwis|filtry|leczenie|opieka|naprawa|naprawy)$")
FACT = re.compile(r"^Wiesz: ([^.]+)\. (.*)$", re.M)
COMMON = {"ktore", "ktory", "ktora", "gdzie", "kiedy", "przed", "tylko", "bardzo", "jeszcze", "ponie",
          "dlate", "wiecz", "dobrz", "bedzi", "mozes", "zawsz", "nigdy", "takze", "wtedy", "sobie",
          "przez", "potem", "teraz", "nawet", "inacz", "siebi", "jesli", "swoje", "swoja", "swoim"}


def fold(text):
    text = unicodedata.normalize("NFKD", text.lower().replace("ł", "l"))
    return "".join(c for c in text if not unicodedata.combining(c))


def stems(text):
    return {w[:5] for w in re.findall(r"[a-z]{5,}", fold(text))} - COMMON


def tailored(subject, text):
    subject = subject.strip()
    return subject[:1].islower() and bool(TAILORED.match(subject.lower()))


def renamed(subject, text):
    """'pomoc. Ruta w zielarni...' -> 'Ruta. Ruta w zielarni...'."""
    name = re.match(r"([A-ZĄĆĘŁŃÓŚŹŻ]\w+)", text)
    if GENERIC.match(subject.strip().lower()) and name:
        return name.group(1)
    return subject


def clean(row):
    """(row, why it goes or None, tailored facts cut)."""
    messages = row["messages"]
    system = messages[0]["content"]
    cut = [m for m in FACT.finditer(system) if tailored(m.group(1), m.group(2))]
    if CUT:
        for m in cut:
            print("  wycięty:", m.group(0))
    for m in cut:
        system = system.replace(m.group(0) + "\n", "").replace(m.group(0), "")
    system = FACT.sub(lambda m: f"Wiesz: {renamed(m.group(1), m.group(2))}. {m.group(2)}", system)
    system = system.rstrip("\n")
    messages = [{"role": "system", "content": system}] + messages[1:]
    row = {"messages": messages}

    reply = messages[-1]["content"]
    if cut:
        context = stems(system + "\n" + "\n".join(m["content"] for m in messages[1:-1]))
        only = set().union(*(stems(m.group(2)) for m in cut)) - context
        leaned = only & stems(reply)
        # one shared word is often chance ("z bliska"); two, or the fact's superlative, is not
        if len(leaned) >= 2 or any(w.startswith("naj") for w in leaned):
            return row, "odpowiedz-z-faktu:" + ",".join(sorted(leaned)), len(cut)

    gra = messages[-2]["content"].split("[GRA]", 1)[-1].strip() if "[GRA]" in messages[-2]["content"] else ""
    previous = [m["content"] for m in messages[1:-1] if m["role"] == "assistant"]
    issues = check({"system": system, "turns": [{"gra": gra, "system": system}]}, 0, reply, previous)
    if issues:
        return row, ",".join(sorted(set(issues))), len(cut)
    return row, None, len(cut)


def main(argv):
    out = argv[argv.index("--out") + 1]
    paths = [a for j, a in enumerate(argv) if a != "--out" and (j == 0 or argv[j - 1] != "--out")]
    kept, dropped, facts, seen = [], [], 0, set()
    for path in paths:
        with open(path, encoding="utf-8") as handle:
            for number, line in enumerate(handle, 1):
                if not line.strip():
                    continue
                row, why, cut = clean(json.loads(line))
                facts += cut
                where = f"{os.path.basename(path)}#{number}"
                key = json.dumps(row["messages"], ensure_ascii=False)
                if why is None and key in seen:
                    why = "duplikat"
                if why:
                    dropped.append((where, why, row["messages"][-1]["content"]))
                    continue
                seen.add(key)
                kept.append(row)
    with open(out, "w", encoding="utf-8") as handle:
        for row in kept:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")
    print(f"rozmów {len(kept) + len(dropped)}: zostaje {len(kept)}, odpada {len(dropped)}; wyciętych faktów pod pytanie: {facts}")
    for reason, count in Counter(why.split(":")[0] for _, why, _ in dropped).most_common():
        print(f"  {reason:20s} {count}")
    for where, why, reply in dropped:
        print(f"  {where} {why}: {reply[:150]}")
    tags = Counter(re.match(r"\[(\w+)\]", r["messages"][-1]["content"]).group(1) for r in kept)
    print("znaczniki:", dict(tags.most_common()))
    print(f"zapisano {out}")


if __name__ == "__main__":
    if "--out" not in sys.argv:
        sys.exit(__doc__)
    main(sys.argv[1:])
