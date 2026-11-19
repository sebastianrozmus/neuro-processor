#!/usr/bin/env python3
"""A table of all bench results in tools/bench/results: model, seed, how
many of the cases kept the format, and the most common errors.

    python3 tools/bench/compare.py            # everything
    python3 tools/bench/compare.py run6 run7  # only results whose folder name contains one of these
"""
import json
import sys
from pathlib import Path

RESULTS = Path(__file__).resolve().parent / "results"


def main():
    wanted = sys.argv[1:]
    rows = []
    for summary in sorted(RESULTS.glob("*/summary.json")):
        name = summary.parent.name
        if wanted and not any(w in name for w in wanted):
            continue
        data = json.loads(summary.read_text(encoding="utf-8"))
        errors = sorted(data.get("error_counts", {}).items(), key=lambda kv: -kv[1])[:3]
        roles = " ".join(f"{role} {r['automatic_format_pass']}/{r['completed']}" for role, r in sorted(data.get("by_role", {}).items()))
        rows.append((name, f"{data['automatic_format_pass']}/{data['completed_cases']}", roles,
                     ", ".join(f"{e}×{n}" for e, n in errors), data.get("case_file_sha256", "")[:8]))
    if not rows:
        sys.exit("brak wyników")
    widths = [max(len(row[i]) for row in rows) for i in range(4)]
    for row in rows:
        print("  ".join(cell.ljust(widths[i]) if i < 4 else cell for i, cell in enumerate(row)))


if __name__ == "__main__":
    main()
