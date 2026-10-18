"""Order A of run 11 (ZAMOWIENIE-run11.md): packs of dry NPC lines for GPT to revive,
and merging GPT's answers back into the training set.

    python3 revive.py pack train.jsonl revive/ [--size 40]
    python3 revive.py merge train.jsonl extra-gpt-revive-k*.jsonl --out train-v2.jsonl

pack writes revive-01.jsonl, revive-02.jsonl...: lines {"id", "messages", "popraw": [k, ...]},
where id is the line number in train.jsonl (from 1) and k counts assistant messages only (from 0).
merge takes GPT lines {"id", "k", "kwestia"} (kwestia null = keep), checks each new line with
tools/eval/check.py, keeps the original when it fails, and lowercases/maps wrong tags everywhere.

Greetings (assistant right after system, no tag) stay as they are: the game puts the shop's
Greeting text into the history untagged (shop.go Remember), and the loss counts only the last reply.
"""
import json
import os
import re
import sys
from collections import Counter

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "eval"))
from check import TAGS, check  # noqa: E402

FREE = "Gracz nie składa oferty"
TAG = re.compile(r"^\[([^\]]+)\]\s*")
FIX = {"obrażona": "gniew", "obrażony": "gniew", "kpię": "kpina", "kpią": "kpina"}
# generic counter-clerk phrases: a short [spokój] line with one of these reads like a form
CLERK = re.compile(
    r"czego (potrzebujesz|chcesz|szukasz|ci trzeba|sobie życzysz|ode mnie)|o czym chcesz|w czym (mogę|pomóc)"
    r"|co cię (tu )?sprowadza|^(rozumiem|nie wiem|dobrze|jasne|w porządku|witaj|witam|cześć|dzień dobry|tak|nie)\b"
    r"|\bnie znam\b|porozmawiaj z|\bpytaj\b|zapytaj|zapraszam|wskaż (przedmiot|towar)|podaj (też )?swoją ofertę"
    r"|nie mam (na to|o tym)|nie wiem")
SHORT_FREE = 70   # free talk shorter than this, without a number -> dry
SHORT_CALM = 90   # [spokój] shorter than this with a clerk phrase -> dry


def load(path):
    with open(path, encoding="utf-8") as handle:
        return [json.loads(line) for line in handle if line.strip()]


def is_gm(row):
    return row["messages"][0]["content"].startswith("Jesteś Mistrzem Gry")


def split(text):
    """(tag or None, speech)."""
    m = TAG.match(text)
    return (m.group(1), text[m.end():]) if m else (None, text)


def gra_of(messages, i):
    """[GRA] block of the player line before message i ("" when there is none)."""
    if i == 0 or messages[i - 1]["role"] != "user":
        return ""
    parts = messages[i - 1]["content"].split("[GRA]", 1)
    return parts[1].strip() if len(parts) == 2 else ""


def assistants(messages):
    return [i for i, m in enumerate(messages) if m["role"] == "assistant"]


def dry(messages, i):
    gra = gra_of(messages, i)
    if not gra:
        return None  # greeting or history line: the model never says it in a trained position
    tag, speech = split(messages[i]["content"])
    speech = speech.strip()
    free = gra.startswith(FREE)
    if free and len(speech) < SHORT_FREE and not re.search(r"\d", speech):
        return "krotka"
    if tag and tag.lower() == "spokój" and len(speech) < SHORT_CALM and CLERK.search(speech.lower()):
        if free or not re.search(r"\d", speech):
            return "urzednik"
    return None


def pack(argv):
    size = 40
    if "--size" in argv:
        at = argv.index("--size")
        size = int(argv[at + 1])
        argv = argv[:at] + argv[at + 2:]
    src, outdir = argv
    rows = load(src)
    os.makedirs(outdir, exist_ok=True)
    picked, why = [], Counter()
    for n, row in enumerate(rows, 1):
        if is_gm(row):
            continue
        messages = row["messages"]
        ks = []
        for k, i in enumerate(assistants(messages)):
            reason = dry(messages, i)
            if reason:
                ks.append(k)
                why[reason] += 1
        if ks:
            picked.append({"id": str(n), "messages": messages, "popraw": ks})
    for p in range(0, len(picked), size):
        with open(os.path.join(outdir, f"revive-{p // size + 1:02d}.jsonl"), "w", encoding="utf-8") as handle:
            for line in picked[p:p + size]:
                handle.write(json.dumps(line, ensure_ascii=False) + "\n")
    turns = sum(len(p["popraw"]) for p in picked)
    print(f"{len(picked)} rozmów, {turns} kwestii ({dict(why)}), {(len(picked) + size - 1) // size} paczek po {size} w {outdir}")


def fix_tag(text):
    tag, speech = split(text)
    if tag is None or tag in TAGS:
        return text, False
    low = tag.lower()
    new = FIX.get(low, low)
    if new not in TAGS:
        return text, False
    return f"[{new}] {speech}", True


def tag_counts(rows):
    """Tags of the trained (last) replies of NPC rows."""
    counts = Counter()
    for row in rows:
        if not is_gm(row) and row["messages"][-1]["role"] == "assistant":
            counts[split(row["messages"][-1]["content"])[0] or "(brak)"] += 1
    return counts


def validate(messages, i, text):
    system = messages[0]["content"]
    gra = gra_of(messages, i)
    scenario = {"system": system, "turns": [{"gra": gra, "system": system}]}
    previous = [m["content"] for m in messages[:i] if m["role"] == "assistant"]
    issues = check(scenario, 0, text, previous)
    speech = split(text)[1]
    if re.search(r'["„”«»]', speech):
        issues.append("cudzyslow")
    return sorted(set(issues))


def merge(argv):
    out = argv[argv.index("--out") + 1]
    argv = [a for j, a in enumerate(argv) if a != "--out" and (j == 0 or argv[j - 1] != "--out")]
    src, answers = argv[0], argv[1:]
    rows = load(src)
    before = tag_counts(rows)

    wanted = {}
    for path in answers:
        for ans in load(path):
            wanted[(str(ans["id"]), int(ans["k"]))] = ans.get("kwestia")

    replaced, kept, rejected, missing = 0, 0, [], []
    for (rid, k), text in sorted(wanted.items(), key=lambda x: (int(x[0][0]), x[0][1])):
        n = int(rid)
        ks = assistants(rows[n - 1]["messages"]) if 1 <= n <= len(rows) else []
        if k >= len(ks):
            missing.append(f"{rid}#{k}")
            continue
        if text is None:
            kept += 1
            continue
        text = fix_tag(text.strip())[0]
        messages = rows[n - 1]["messages"]
        issues = validate(messages, ks[k], text)
        if issues:
            rejected.append((rid, k, issues, text))
            continue
        messages[ks[k]]["content"] = text
        replaced += 1

    fixed = 0
    for row in rows:
        if is_gm(row):
            continue
        for m in row["messages"]:
            if m["role"] == "assistant":
                m["content"], changed = fix_tag(m["content"])
                fixed += changed

    with open(out, "w", encoding="utf-8") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")

    print(f"odpowiedzi {len(wanted)}: zamienione {replaced}, zostawione (null) {kept}, odrzucone {len(rejected)}, złe id/k {len(missing)}")
    print(f"poprawione znaczniki (wielka litera, Obrażona, Kpię): {fixed}")
    reasons = Counter(issue.split(":")[0] for _, _, issues, _ in rejected for issue in issues)
    for reason, count in reasons.most_common():
        print(f"  {reason:20s} {count}")
    for rid, k, issues, text in rejected:
        print(f"  odrzucone {rid}#{k} {','.join(issues)}: {text}")
    if missing:
        print("  złe id/k: " + " ".join(missing))
    after = tag_counts(rows)
    print("znaczniki ostatnich kwestii (przed -> po):")
    for tag in sorted(set(before) | set(after), key=lambda t: -after.get(t, 0)):
        print(f"  {tag:12s} {before.get(tag, 0):5d} -> {after.get(tag, 0):5d}")
    print(f"zapisano {out}")


if __name__ == "__main__":
    if len(sys.argv) < 2 or sys.argv[1] not in ("pack", "merge"):
        sys.exit(__doc__)
    (pack if sys.argv[1] == "pack" else merge)(sys.argv[2:])
