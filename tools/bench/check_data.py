#!/usr/bin/env python3
"""Checks a new training package (JSONL from an order in tools/model/PROMPT.md)
before it goes into the training set, with the same rules as the bench:
message order, one [GRA] block at the end of the last player line, a mood tag,
1-3 sentences, digits only, and only numbers the game gave. The game
master's narrations get the same number and sentence rules, without a mood.

    python3 tools/bench/check_data.py ~/Library/Application\\ Support/neuro-processor/extra-gpt-npc-run7.jsonl

Rows that fail are listed with their line number; the exit code is 1 when
any row fails. A package that passes still needs reading by eye: payment
direction, staying in role, Polish.
"""
import collections
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from npc_checks import allowed_numbers, check_response, numbers, validate_messages  # noqa: E402

SHOWN = 15


def narration(row):
    """The game master's narrations: the whole player line is the [GRA]
    report of the round, and the reply carries no mood tag."""
    messages = row.get("messages") or []
    return len(messages) >= 3 and messages[0].get("content", "").startswith("Jesteś Mistrzem Gry")


def check_narration(messages):
    """Numbers only from the round's report, digits, 1-3 sentences, no markup."""
    allowed = numbers(messages[-2]["content"])
    errors = check_response("[spokój] " + messages[-1]["content"], {"allowed_numbers": sorted(allowed)})["errors"]
    return [e for e in errors if e != "missing_or_invalid_mood"]


def check(row):
    messages = row.get("messages")
    errors = validate_messages(messages, has_target=True)
    if errors:
        return errors
    try:
        allowed = allowed_numbers(messages[:-1])
    except ValueError as exc:
        return [str(exc)]
    return check_response(messages[-1]["content"], {"allowed_numbers": sorted(allowed)})["errors"]


def main():
    if len(sys.argv) < 2:
        sys.exit(f"użycie: {sys.argv[0]} PLIK.jsonl [PLIK.jsonl ...]")
    failed = 0
    for path in sys.argv[1:]:
        counts, bad, total, narrations = collections.Counter(), [], 0, 0
        with open(path, encoding="utf-8") as handle:
            for number, line in enumerate(handle, 1):
                if not line.strip():
                    continue
                total += 1
                try:
                    row = json.loads(line)
                except json.JSONDecodeError as exc:
                    errors = [f"json: {exc}"]
                else:
                    if narration(row):
                        narrations += 1
                        errors = check_narration(row["messages"])
                    else:
                        errors = check(row)
                if errors:
                    counts.update(e.split(":")[0] for e in errors)
                    bad.append((number, errors, row))
        print(f"{path}: {total - len(bad)}/{total} OK" + (f" (w tym narracji Mistrza Gry: {narrations})" if narrations else ""))
        for error, count in counts.most_common():
            print(f"  {error}: {count}")
        for number, errors, row in bad[:SHOWN]:
            reply = row.get("messages", [{}])[-1].get("content", "") if isinstance(row, dict) else ""
            print(f"  linia {number}: {', '.join(errors)}\n    {reply[:160]}")
        if len(bad) > SHOWN:
            print(f"  … i {len(bad) - SHOWN} więcej")
        failed += len(bad)
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
