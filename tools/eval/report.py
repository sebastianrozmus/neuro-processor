"""Side-by-side HTML report of two (or more) models on the scenarios.

    python3 report.py scenarios.jsonl out/report.html out/scen-run7.checked.jsonl out/scen-run6.checked.jsonl [judge.json]
"""
import html
import json
import re
import sys
from collections import Counter

scen_path, out_path, *paths = sys.argv[1:]
judge = {}
if paths and paths[-1].endswith("judge.json"):
    with open(paths.pop(), encoding="utf-8") as handle:
        judge = json.load(handle)

with open(scen_path, encoding="utf-8") as handle:
    scenarios = [json.loads(line) for line in handle]
models = []
for path in paths:
    with open(path, encoding="utf-8") as handle:
        rows = {r["id"]: r for r in map(json.loads, handle)}
    models.append((next(iter(rows.values()))["model"], rows))


def esc(text):
    return html.escape(text or "")


def summary():
    head = "<tr><th></th>" + "".join(f"<th>{esc(m)}</th>" for m, _ in models) + "</tr>"
    stats = []
    for name, rows in models:
        total = clean = chars = 0
        secs = 0.0
        issues = Counter()
        for r in rows.values():
            for rep in r["replies"]:
                total += 1
                clean += not rep["issues"]
                chars += len(rep["reply"])
                secs += rep.get("s", 0)
                for i in rep["issues"]:
                    issues[i.split(":")[0]] += 1
        stats.append((total, clean, chars, secs, issues))
    keys = sorted({k for s in stats for k in s[4]}, key=lambda k: -sum(s[4][k] for s in stats))
    body = "<tr><td>odpowiedzi</td>" + "".join(f"<td>{s[0]}</td>" for s in stats) + "</tr>"
    body += "<tr><td>bez uwag automatu</td>" + "".join(f"<td><b>{s[1]}</b> ({100 * s[1] / max(s[0], 1):.0f}%)</td>" for s in stats) + "</tr>"
    body += "<tr><td>śr. długość (znaki)</td>" + "".join(f"<td>{s[2] / max(s[0], 1):.0f}</td>" for s in stats) + "</tr>"
    body += "<tr><td>śr. czas (s)</td>" + "".join(f"<td>{s[3] / max(s[0], 1):.1f}</td>" for s in stats) + "</tr>"
    for k in keys:
        body += f"<tr><td class=k>{esc(k)}</td>" + "".join(f"<td>{s[4][k] or ''}</td>" for s in stats) + "</tr>"
    if judge.get("summary"):
        for label, values in judge["summary"].items():
            body += f"<tr><td>{esc(label)}</td>" + "".join(f"<td>{esc(str(values.get(m, '')))}</td>" for m, _ in models) + "</tr>"
    return f"<table class=sum>{head}{body}</table>"


def gra_short(gra):
    return re.sub(r"\s*Odpowiedz najwyżej trzema krótkimi zdaniami\. Liczby pisz cyframi\.", "", gra or "")


parts = []
for s in scenarios:
    sid = s["id"]
    note = judge.get("scenarios", {}).get(sid, "")
    parts.append(f"<section id='{sid}'><h3>{esc(sid)} <small>{esc(s.get('note', ''))}</small></h3>")
    if note:
        parts.append(f"<p class=judge>{esc(note)}</p>")
    for idx, turn in enumerate(s["turns"]):
        player = turn.get("player")
        parts.append("<div class=turn>")
        if player:
            parts.append(f"<div class=player>🧑 {esc(player)}</div>")
        parts.append(f"<div class=gra>[GRA] {esc(gra_short(turn.get('gra')))}</div><div class=replies>")
        for name, rows in models:
            r = rows.get(sid)
            if not r or idx >= len(r["replies"]):
                parts.append("<div class=rep><i>brak</i></div>")
                continue
            rep = r["replies"][idx]
            badges = "".join(f"<span class=bad>{esc(i)}</span>" for i in rep["issues"])
            parts.append(f"<div class=rep><div class=m>{esc(name)}</div>{esc(rep['reply'])}{badges}</div>")
        parts.append("</div></div>")
    parts.append("</section>")

toc = " ".join(f"<a href='#{s['id']}'>{esc(s['id'].split('-')[0])}</a>" for s in scenarios)
page = f"""<!doctype html><html lang=pl><head><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>Porównanie modeli NPC</title><style>
:root{{--bg:#fbfaf7;--fg:#1d1d1b;--mut:#6b6a66;--card:#fff;--line:#e4e1da;--bad:#b42318;--badbg:#fde8e6;--gra:#f3f0e8;--acc:#2f5d8a}}
@media (prefers-color-scheme:dark){{:root:not([data-theme=light]){{--bg:#161615;--fg:#ecebe7;--mut:#a3a19b;--card:#1f1f1d;--line:#34332f;--bad:#ff8a80;--badbg:#3a1d1a;--gra:#262521;--acc:#8ab4e0}}}}
:root[data-theme=dark]{{--bg:#161615;--fg:#ecebe7;--mut:#a3a19b;--card:#1f1f1d;--line:#34332f;--bad:#ff8a80;--badbg:#3a1d1a;--gra:#262521;--acc:#8ab4e0}}
body{{margin:0;background:var(--bg);color:var(--fg);font:15px/1.5 system-ui,sans-serif}}main{{max-width:1200px;margin:auto;padding:16px}}
h1{{font-size:22px}}h3{{margin:28px 0 8px;font-size:17px}}h3 small{{color:var(--mut);font-weight:400}}
.sum{{border-collapse:collapse;margin:12px 0}}.sum td,.sum th{{border:1px solid var(--line);padding:4px 10px;text-align:left}}.k{{font-family:monospace}}
.toc{{line-height:2}}.toc a{{color:var(--acc);margin-right:6px;text-decoration:none}}
.turn{{border-top:1px solid var(--line);padding:8px 0}}.player{{font-weight:600}}.gra{{font-size:12.5px;color:var(--mut);background:var(--gra);padding:4px 8px;border-radius:6px;margin:4px 0}}
.replies{{display:grid;grid-template-columns:repeat({len(models)},1fr);gap:8px}}@media (max-width:700px){{.replies{{grid-template-columns:1fr}}}}
.rep{{background:var(--card);border:1px solid var(--line);border-radius:8px;padding:6px 10px}}.m{{font-size:11px;color:var(--mut)}}
.bad{{display:inline-block;margin:4px 4px 0 0;font-size:11px;color:var(--bad);background:var(--badbg);padding:0 6px;border-radius:4px}}
.judge{{background:var(--gra);padding:8px 10px;border-radius:8px}}
</style></head><body><main><h1>Porównanie modeli NPC: {esc(' vs '.join(m for m, _ in models))}</h1>
{summary()}<p class=toc>{toc}</p>{''.join(parts)}</main></body></html>"""
with open(out_path, "w", encoding="utf-8") as handle:
    handle.write(page)
print("raport:", out_path)
