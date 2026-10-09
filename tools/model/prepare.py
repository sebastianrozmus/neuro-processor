"""Build the training set for the game's own model.

Reads the talks the game recorded (talks.jsonl), the synthetic ones
(synth.jsonl) and anything converted from FIREBALL, drops what is too long
or empty, shuffles, and writes train/valid/test.jsonl in the chat format
mlx_lm.lora expects: {"messages": [{"role": ..., "content": ...}, ...]}.

Usage: python3 prepare.py OUT_DIR FILE.jsonl [FILE.jsonl ...]
"""
import json
import os
import random
import sys

MAX_CHARS = 9000
# NEURO_RAW_TALKS=1 lets the game's own unreviewed recordings in.
RAW_TALKS = os.environ.get("NEURO_RAW_TALKS") == "1"
VALID_SHARE = 0.05
TEST_SHARE = 0.02

# Shopkeepers' lines have to open with one of the game's mood tags; the
# game master's narrations (outcome "scene") carry none.
MOODS = ("[spokój]", "[radość]", "[gniew]", "[smutek]", "[strach]", "[kpina]", "[zdziwienie]", "[czułość]")


def load(path):
    rows = []
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError:
                continue
            messages = row.get("messages") if isinstance(row, dict) else row
            if not isinstance(messages, list) or len(messages) < 2:
                continue
            if messages[-1].get("role") != "assistant" or not messages[-1].get("content", "").strip():
                continue
            if sum(len(m.get("content", "")) for m in messages) > MAX_CHARS:
                continue
            reply = messages[-1]["content"].strip()
            if isinstance(row, dict) and row.get("outcome") != "scene" and not reply.lower().startswith(MOODS):
                continue
            # What the game recorded is whatever the small model said at the
            # time, unreviewed; it teaches that model's mistakes. Only talks
            # given a source (synthetic, or rewritten through apply.py) count,
            # unless raw logs are asked for.
            if isinstance(row, dict) and "outcome" in row and "source" not in row and not RAW_TALKS:
                continue
            rows.append({"messages": [{"role": m["role"], "content": m["content"]} for m in messages]})
    return rows


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)

    out, sources = sys.argv[1], sys.argv[2:]
    rows = []
    for path in sources:
        loaded = load(path)
        print(f"{path}: {len(loaded)} wymian")
        rows.extend(loaded)

    # The same prompt and reply recorded twice teach nothing new.
    seen, unique = set(), []
    for row in rows:
        key = json.dumps(row, ensure_ascii=False, sort_keys=True)
        if key not in seen:
            seen.add(key)
            unique.append(row)

    random.Random(7).shuffle(unique)
    valid = max(1, int(len(unique) * VALID_SHARE))
    test = max(1, int(len(unique) * TEST_SHARE))
    splits = {
        "test": unique[:test],
        "valid": unique[test:test + valid],
        "train": unique[test + valid:],
    }

    import os
    os.makedirs(out, exist_ok=True)
    for name, part in splits.items():
        with open(os.path.join(out, f"{name}.jsonl"), "w", encoding="utf-8") as handle:
            for row in part:
                handle.write(json.dumps(row, ensure_ascii=False) + "\n")
        print(f"{name}: {len(part)}")


if __name__ == "__main__":
    main()
