"""Speech styles (ZAMOWIENIE-style.md): scenarios with a STYL MOWY line and a
check of how much of the style the replies carry.

    python3 style.py build scenarios.jsonl style-scen.jsonl
    python3 style.py check style-scen.jsonl out/style-run7.jsonl [...] --plain out/scen-run7.jsonl
    python3 style.py pack ../model/extra-claude-style-c1-k1.jsonl [...]

`build` takes ten scenarios per pilot style and puts the style's line right
after TWOJA POSTAĆ, the way the game will. `check` counts, per style, the
replies with a word of the style and the anachronisms; with --plain it
counts the same words in replies to the same scenarios without a style, so
that a keeper who talks of anvils anyway is not taken for a styled one.
"""
import json
import re
import sys
import unicodedata
from collections import defaultdict

STYLES = {
    "staropolski": "Mówisz po staropolsku, lekko: waść, jeno, azali, onegdaj, przeto, szyk czasem przestawny. Najwyżej dwa-trzy dawne słowa na kwestię, żeby dało się zrozumieć; żadnych współczesnych słów.",
    "wojskowy": "Mówisz krótko i rzeczowo jak żołnierz: meldujesz, potwierdzasz, oceniasz teren i zagrożenie, używasz słów „przyjąłem”, „rozkaz”, „pozycja”, „cel”. Bez ozdobników.",
    "krasnoludzki": "Mówisz szorstko i rzemieślniczo jak krasnolud: kamień, kowadło, ruda, przodkowie, honor klanu; klniesz „na brody przodków” i nie ufasz elfom.",
}

PICK = {
    "staropolski": ["L02-jagoda", "L14-ola", "S03-jagoda-kaszel", "S04-jagoda-kupno", "S18-jagoda-wieczor", "S21-siemira-wyklety", "S22-siemira-jablko", "S30-ola-strzala", "S31-ola-atak", "S34-runa-zaba"],
    "krasnoludzki": ["L03-torvin", "L04-hakon", "S05-torvin-kolczuga", "S06-torvin-miecze", "S07-hakon-miod", "S08-hakon-futro", "S19-torvin-elfka", "S33-ingrid-wyznanie", "S35-ingrid-pozegnanie", "S10-marea-mapa"],
    "wojskowy": ["L10-rdza", "L13-kobalt", "S23-rdza-najem", "S24-rdza-bieda", "S25-haszim-cena", "S27-kobalt-dusza", "S28-kobalt-najem", "S29-wit-najem", "S11-aksjomat-hurt", "S15-rudy-kawal"],
}

# Words of each style, folded (lowercase, no Polish letters).
MARKERS = {
    "staropolski": r"\b(wasc\w*|wacpan\w*|jeno|azali\w*|onegdaj|przeto|zaiste|wszelako|tedy|niechaj|acan\w*|izali|bowiem|jako zywo|ninie|atoli|iza|mosci|panie bracie|zali)\b",
    "wojskowy": r"\b(melduj\w*|przyjalem|przyjeto|rozkaz\w*|pozycj\w*|teren\w*|cel\w?|zrozumiano|potwierdzam|negatyw\w*|odmeldow\w*|szereg\w*|zadanie|meldunek|odbior|bez odbioru|wykonac|wykonam|osłon\w*|oslon\w*)\b",
    "krasnoludzki": r"\b(kamien\w*|kowadl\w*|rud\w*|przodk\w*|klan\w*|brod\w*|sztoln\w*|mlot\w*|kuzni\w*|elf\w*|kilof\w*|gor\w*)\b",
}
ANACHRONISMS = r"\b(ok|okej|spoko|super|fajn\w*|serio|luz\w*|sorry|totalnie|generalnie)\b"
INVENTED = r"\b(azaliz\w+|onegdajow\w*|waszmosci\w+)\b"


def fold(text):
    text = unicodedata.normalize("NFKD", text.lower().replace("ł", "l"))
    return "".join(c for c in text if not unicodedata.combining(c))


def styled_system(system, style):
    """The system with the style's line after the character's paragraph."""
    line = f"STYL MOWY: {style}. {STYLES[style]}"
    return re.sub(r"(TWOJA POSTAĆ:[^\n]*)", lambda m: m.group(1) + "\n" + line, system, count=1)


def build(source, target):
    scenarios = {s["id"]: s for s in map(json.loads, open(source, encoding="utf-8"))}
    with open(target, "w", encoding="utf-8") as out:
        for style, ids in PICK.items():
            for sid in ids:
                scenario = dict(scenarios[sid])
                scenario["id"] = f"{style}:{sid}"
                scenario["style"] = style
                scenario["plain"] = sid
                scenario["system"] = styled_system(scenario["system"], style)
                out.write(json.dumps(scenario, ensure_ascii=False) + "\n")


def body(reply):
    return fold(re.sub(r"^\[\w+\]\s*", "", reply.strip()))


def rates(results, styles, key):
    """Per style: replies, with a style word, with an anachronism, invented forms."""
    stats = defaultdict(lambda: [0, 0, 0, 0])
    for res in results:
        style = styles.get(res["id"]) if key == "id" else styles.get(res["id"])
        if not style:
            continue
        for rep in res["replies"]:
            text = body(rep["reply"])
            s = stats[style]
            s[0] += 1
            s[1] += bool(re.search(MARKERS[style], text))
            s[2] += style == "staropolski" and bool(re.search(ANACHRONISMS, text))
            s[3] += bool(re.search(INVENTED, text))
    return stats


def check(scenarios_path, paths, plain_paths):
    scenarios = [json.loads(line) for line in open(scenarios_path, encoding="utf-8")]
    styled = {s["id"]: s["style"] for s in scenarios}
    plain = {s["plain"]: s["style"] for s in scenarios}

    for path in plain_paths:
        results = [json.loads(line) for line in open(path, encoding="utf-8") if line.strip()]
        stats = rates(results, plain, "id")
        print(f"\n== bez stylu: {path}")
        for style, (n, marked, _, _) in stats.items():
            print(f"  {style:13s} słowa stylu w {marked}/{n} ({100 * marked // max(n, 1)}%)")

    for path in paths:
        results = [json.loads(line) for line in open(path, encoding="utf-8") if line.strip()]
        stats = rates(results, styled, "id")
        model = results[0]["model"] if results else path
        print(f"\n== {model} ze stylem")
        for style, (n, marked, anach, invented) in stats.items():
            print(f"  {style:13s} słowa stylu w {marked}/{n} ({100 * marked // max(n, 1)}%), anachronizmy {anach}, wymyślone formy {invented}")


TAILORED = r"^Wiesz: (bro[nń]|uzbrojenie|ma[sś][cć]|bandaż|opatrunek|zioła|[sś]wiatło|leki|sprzęt|towar|pancerz|amunicja)\b"


def pack(paths):
    """Checks a data pack of order C1 (lines {"messages": [...]}): the style
    line word for word, the style heard in styled replies and not in the
    control ones, the game's rules (check.py), one-sentence facts and no
    facts made up to fit the question."""
    sys.path.insert(0, __import__("os").path.dirname(__file__))
    import check as rules

    problems, counts = [], defaultdict(int)
    for path in paths:
        for number, line in enumerate(open(path, encoding="utf-8"), 1):
            messages = json.loads(line)["messages"]
            system, user, reply = messages[0]["content"], messages[-2]["content"], messages[-1]["content"]
            found = re.search(r"^STYL MOWY: (\w+)\. (.*)$", system, re.M)
            style = found.group(1) if found else None
            where = f"{path.rsplit('/', 1)[-1]}#{number}"
            issues = []

            if found and STYLES.get(style) != found.group(2):
                issues.append("linia-stylu-nie-dosłownie")
            if found and not re.search(r"TWOJA POSTAĆ:[^\n]*\nSTYL MOWY:", system):
                issues.append("linia-stylu-nie-po-postaci")
            text = body(reply)
            if style and not re.search(MARKERS[style], text):
                issues.append("brak-słów-stylu")
            if not style and any(re.search(m, text) for s, m in MARKERS.items() if s != "krasnoludzki"):
                issues.append("styl-bez-linii")
            if style == "staropolski" and re.search(ANACHRONISMS, text):
                issues.append("anachronizm")
            if re.search(INVENTED, text):
                issues.append("wymyślona-forma")
            for fact in re.findall(r"^Wiesz: .*$", system, re.M):
                if re.search(TAILORED, fact, re.I):
                    issues.append("fakt-pod-pytanie")
                if len(fact.split()) > 22 or fact.count(". ") > 1:
                    issues.append("fakt-za-długi")

            gra = user.split("[GRA]")[-1] if "[GRA]" in user else ""
            previous = [m["content"] for m in messages[1:-1] if m["role"] == "assistant"]
            issues += rules.check({"turns": [{"gra": gra, "system": system}], "system": system}, 0, reply, previous)

            counts[style or "kontrola"] += 1
            if issues:
                problems.append((where, sorted(set(issues)), reply[:140]))

    print("linie:", dict(counts), "| z uwagami:", len(problems))
    for problem in problems:
        print(*problem)


def main():
    if sys.argv[1] == "build":
        build(sys.argv[2], sys.argv[3])
    elif sys.argv[1] == "pack":
        pack(sys.argv[2:])
    elif sys.argv[1] == "check":
        args = sys.argv[3:]
        plain = []
        if "--plain" in args:
            at = args.index("--plain")
            plain, args = args[at + 1:], args[:at]
        check(sys.argv[2], args, plain)


if __name__ == "__main__":
    main()
