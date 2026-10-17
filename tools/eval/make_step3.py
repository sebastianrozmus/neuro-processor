"""Builds the step 3 training set from pack.sh's data and this evaluation.

    python3 make_step3.py ~/neuro-train/data step3/data

- Game Master rows: the audit's fix for wrong rows, the vivid rewrite for dry ones.
- Other rows: dropped when check.py finds a hard error (wrong or missing amount,
  invented number, broken tag, name prefix, amount where none is allowed);
  numbers written in words are turned into digits.
- Gold replies for the scenarios join the training set twice, except the
  held-out scenarios (HELD_OUT), which stay as a test of run8 against run7.
"""
import glob
import json
import os
import re
import sys

from check import check
from run import user_line

HELD_OUT = {"L03-torvin", "L08-rudy", "L13-kobalt", "L17-runa", "L21-feliks", "L24-gm-wilki",
            "S05-torvin-kolczuga", "S10-marea-mapa", "S15-rudy-kawal", "S20-marea-niejasne", "S25-haszim-cena",
            "S30-ola-strzala", "S35-ingrid-pozegnanie", "S40-helena-adres", "S45-gm-leczenie", "S50-gm-meduza"}
HARD = {"zly-znacznik", "liczba-spoza", "brak-kwoty", "imie-na-poczatku", "kwota-zakazana", "brak-znacznika", "GM:znacznik"}

UNITS = {"jeden": 1, "jedna": 1, "dwa": 2, "dwie": 2, "trzy": 3, "cztery": 4, "pięć": 5, "sześć": 6, "siedem": 7, "osiem": 8,
         "dziewięć": 9, "dziesięć": 10, "jedenaście": 11, "dwanaście": 12, "trzynaście": 13, "czternaście": 14,
         "piętnaście": 15, "szesnaście": 16, "siedemnaście": 17, "osiemnaście": 18, "dziewiętnaście": 19}
TENS = {"dwadzieścia": 20, "trzydzieści": 30, "czterdzieści": 40, "pięćdziesiąt": 50, "sześćdziesiąt": 60,
        "siedemdziesiąt": 70, "osiemdziesiąt": 80, "dziewięćdziesiąt": 90}
HUNDREDS = {"sto": 100, "dwieście": 200, "trzysta": 300, "czterysta": 400, "pięćset": 500}
ALL = {**UNITS, **TENS, **HUNDREDS}
WORD = r"(?:" + "|".join(sorted(ALL, key=len, reverse=True)) + r")"
SEQ = re.compile(r"\b" + WORD + r"(?:\s+" + WORD + r")*\b", re.I)


def digits(text, allowed):
    """Number words -> digits, only when the value is a number the game gave."""
    def swap(match):
        words = match.group(0).lower().split()
        value = sum(ALL[w] for w in words)
        if str(value) not in allowed:
            return match.group(0)
        return str(value)
    return SEQ.sub(swap, text)


def load(path):
    with open(path, encoding="utf-8") as handle:
        return [json.loads(line) for line in handle if line.strip()]


def main():
    src, dst = sys.argv[1], sys.argv[2]
    os.makedirs(dst, exist_ok=True)
    gm_fix = {}
    with open("step3/gm-audit.json", encoding="utf-8") as handle:
        for bad in json.load(handle)["bad"]:
            gm_fix[(bad["file"], bad["line"])] = bad["fix"]
    for path in sorted(glob.glob("step3/gm-rewrite-*.json")):
        with open(path, encoding="utf-8") as handle:
            for item in json.load(handle):
                gm_fix.setdefault((item["file"], item["line"]), item["new"])

    stats = {"gm_poprawione": 0, "usuniete": 0, "liczby_na_cyfry": 0}
    for name in ("train.jsonl", "valid.jsonl"):
        rows = load(os.path.join(src, name))
        out = []
        for i, row in enumerate(rows):
            msgs = row["messages"]
            if msgs[0]["content"].startswith("Jesteś Mistrzem Gry"):
                new = gm_fix.get((name, i))
                if new:
                    msgs[-1]["content"] = new
                    stats["gm_poprawione"] += 1
                out.append(row)
                continue
            player, _, gra = msgs[-2]["content"].partition("[GRA]")
            allowed = set(re.findall(r"\d+", gra)) | set(re.findall(r"\d+", msgs[0]["content"]))
            fixed = digits(msgs[-1]["content"], allowed)
            if fixed != msgs[-1]["content"]:
                stats["liczby_na_cyfry"] += 1
                msgs[-1]["content"] = fixed
            scenario = {"system": msgs[0]["content"], "turns": [{"player": player.strip(), "gra": gra.strip()}]}
            issues = {x.split(":")[0] if not x.startswith("GM:") else x.split(":")[0] + ":" + x.split(":")[1] for x in check(scenario, 0, fixed, [])}
            if issues & HARD:
                stats["usuniete"] += 1
                continue
            out.append(row)
        if name == "train.jsonl":
            gold = load("step3/train-scenarios.jsonl")
            with open("scenarios.jsonl", encoding="utf-8") as handle:
                order = [s["id"] for s in map(json.loads, handle) for _ in s["turns"]]
            kept = [r for r, sid in zip(gold, order) if sid not in HELD_OUT]
            stats["zlote_w_treningu"] = len(kept)
            out += kept * 2
        with open(os.path.join(dst, name), "w", encoding="utf-8") as handle:
            for row in out:
                handle.write(json.dumps(row, ensure_ascii=False) + "\n")
        stats[name] = len(out)
    with open(os.path.join(src, "test.jsonl"), encoding="utf-8") as handle, open(os.path.join(dst, "test.jsonl"), "w", encoding="utf-8") as out:
        out.write(handle.read())
    with open("scenarios.jsonl", encoding="utf-8") as handle, open(os.path.join(dst, "eval-scenarios.jsonl"), "w", encoding="utf-8") as out:
        for line in handle:
            if json.loads(line)["id"] in HELD_OUT:
                out.write(line)
    print(json.dumps(stats, ensure_ascii=False))


if __name__ == "__main__":
    main()
