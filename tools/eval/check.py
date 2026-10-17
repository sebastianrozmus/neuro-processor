"""Automatic checks of NPC replies against the game's rules.

    python3 check.py scenarios.jsonl out/scen-run7.jsonl [out/scen-run6.jsonl ...]

Prints a summary per model and writes <results>.checked.jsonl with issues per reply.
"""
import json
import re
import sys
from collections import Counter

TAGS = ["spokój", "radość", "gniew", "smutek", "strach", "kpina", "zdziwienie", "czułość"]
# moods from [GRA] and tags that clearly contradict them
CLASH = {
    "obrażona": {"radość", "czułość"},
    "poirytowana": {"radość", "czułość"},
    "zadowolona": {"gniew", "smutek", "strach"},
    "spokojna": {"gniew"},
}
NUM_WORDS = r"\b(dwa|dwie|trzy|cztery|pięć|sześć|siedem|osiem|dziewięć|dziesięć|jedenaście|dwanaście|piętnaście|dwadzieścia|trzydzieści|czterdzieści|pięćdziesiąt|sześćdziesiąt|siedemdziesiąt|osiemdziesiąt|dziewięćdziesiąt|sto|dwieście|trzysta)\b"
DEATH = r"\b(ginie|zgin\w*|pada|padł\w*|martw\w*|zabit\w*|zabij\w*|pokonan\w*|truchł\w*|nie żyje|ucieka|uciekł\w*|uciec)\b"
META = r"(\[GRA\]|\bblok\w* \[?GRA|\bjako model\b|\bsztuczn\w+ inteligencj\w*|\binstrukcj\w+ (gry|systemu)|\bprompt)"


def sentences(text):
    body = re.sub(r"szt\.", "szt", text)
    return [s for s in re.split(r"(?<=[.!?…])\s+", body.strip()) if s.strip()]


def numbers(text):
    return set(re.findall(r"\d+", text))


def words(text):
    return re.findall(r"[a-ząęółśżźćń]+", text.lower())


def check(scenario, idx, reply, previous):
    turn = scenario["turns"][idx]
    gra = turn.get("gra") or ""
    system = turn.get("system") or scenario.get("system", "")
    gm = bool(scenario.get("gm")) or system.startswith("Jesteś Mistrzem Gry")
    issues = []
    text = reply.strip()
    if not text:
        return ["pusta"]

    m = re.match(r"^\[(\w+)\]\s*", text)
    if gm:
        if m:
            issues.append("GM:znacznik")
    else:
        if not m:
            issues.append("brak-znacznika")
        elif m.group(1) not in TAGS:
            issues.append(f"zly-znacznik:{m.group(1)}")
        else:
            mood = re.search(r"Nastrój twojej postaci: (\w+)", gra)
            if mood and m.group(1) in CLASH.get(mood.group(1), set()):
                issues.append(f"nastroj:{mood.group(1)}->{m.group(1)}")
        if re.search(r"\[\w+\]", text[m.end():] if m else text):
            issues.append("drugi-znacznik")
    speech = text[m.end():] if m else text

    if len(sentences(speech)) > 3:
        issues.append(f"zdania:{len(sentences(speech))}")
    if "*" in speech:
        issues.append("gwiazdki")
    if re.match(r'^["„«]', speech):
        issues.append("cudzyslow")
    if re.search(META, speech, re.I):
        issues.append("meta")
    if re.match(r"^[A-ZŁŚŻ][\w-]+:\s", speech):
        issues.append("imie-na-poczatku")

    allowed = numbers(gra) | numbers(system)
    extra = numbers(speech) - allowed
    if extra:
        issues.append("liczba-spoza:" + ",".join(sorted(extra)))
    if re.search(NUM_WORDS, speech.lower()):
        issues.append("liczba-slownie")
    if "Nie podawaj żadnych kwot" in gra and numbers(speech):
        issues.append("kwota-zakazana")

    # the deal: the price the game settled should be said
    key = None
    for pattern in [r"za (\d+) \S+\. Transakcja zawarta", r"płaci ci (\d+)", r"stawkę «Żołd dzienny» (\d+)",
                    r"podaj swoją nową cenę: (\d+)", r"nadal żądasz (\d+)", r"zaoferuj swoją cenę: (\d+)",
                    r"zaoferuj za nią (\d+)", r"Twoja aktualna cena: (\d+)", r"Za tę jedną chcesz (\d+)"]:
        found = re.search(pattern, gra)
        if found:
            key = found.group(1)
            break
    if key and key not in numbers(speech):
        issues.append(f"brak-kwoty:{key}")

    if gm:
        if re.search(DEATH, speech.lower()) and not re.search(r"ginie|pada|pokonani|ucieka|uciec|ucieczk", gra):
            issues.append("GM:zmyslona-smierc")

    if previous:
        mine = words(speech)
        grams = {tuple(mine[i:i + 4]) for i in range(len(mine) - 3)}
        for prev in previous:
            p = words(re.sub(r"^\[\w+\]\s*", "", prev))
            if grams & {tuple(p[i:i + 4]) for i in range(len(p) - 3)}:
                issues.append("powtorka")
                break
    return issues


def main():
    with open(sys.argv[1], encoding="utf-8") as handle:
        scenarios = {s["id"]: s for s in map(json.loads, handle)}
    for path in sys.argv[2:]:
        with open(path, encoding="utf-8") as handle:
            results = [json.loads(line) for line in handle if line.strip()]
        counts, kinds, total, clean, chars, secs = Counter(), Counter(), 0, 0, 0, 0.0
        out = []
        for res in results:
            scenario = scenarios[res["id"]]
            previous = []
            for idx, rep in enumerate(res["replies"]):
                issues = check(scenario, idx, rep["reply"], previous)
                previous.append(rep["reply"])
                rep["issues"] = issues
                total += 1
                clean += not issues
                chars += len(rep["reply"])
                secs += rep.get("s", 0)
                for issue in issues:
                    counts[issue.split(":")[0]] += 1
                kinds[res["id"][:1] + ("GM" if scenario.get("gm") else "")] += bool(issues)
            out.append(res)
        with open(path.replace(".jsonl", ".checked.jsonl"), "w", encoding="utf-8") as handle:
            for res in out:
                handle.write(json.dumps(res, ensure_ascii=False) + "\n")
        model = results[0]["model"] if results else path
        print(f"\n== {model}: {total} odpowiedzi, bez uwag {clean} ({100 * clean / max(total, 1):.0f}%), śr. {chars / max(total, 1):.0f} znaków, śr. {secs / max(total, 1):.1f} s")
        for issue, n in counts.most_common():
            print(f"  {issue:22s} {n}")


if __name__ == "__main__":
    main()
