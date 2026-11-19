#!/usr/bin/env python3
"""Turn FIREBALL turns (lara-martin/FIREBALL on Hugging Face, CC-BY-4.0) into
training records in the shape the game uses for its own conversations:

    {"system": <who speaks and what they know>,
     "user":   <what was said> + " [GRA] " + <what the rules decided>,
     "assistant": <the line to learn>}

For FIREBALL the speaker is the Dungeon Master, the knowledge is the state
of the fight, the verdict is Avrae's own account of the action and the line
is the narration the players wrote after it. Only turns with all three are
kept, and Discord markup is stripped.

Usage: fireball_to_neuro.py OUT.jsonl IN.jsonl [IN.jsonl ...]
"""
import json
import re
import sys

MIN_LINE, MAX_LINE = 20, 600
MARKUP = re.compile(r"[*_`~>#|]+|<a?:\w+:\d+>|https?://\S+")
HP = re.compile(r"<(\d+)/(\d+) HP; (\w+)>")
# Avrae commands carry display flags (-title, -phrase, -f ...); only the
# action and its target matter.
FLAGS = re.compile(r"\s-(?!t\b)\w+.*$", re.S)

VERDICT_MARK = "[GRA]"


def clean(text):
    text = MARKUP.sub("", text)
    return " ".join(text.split())


def actor_line(actor):
    """One actor of the fight as the game would describe it."""
    hp = HP.search(actor.get("hp") or "")
    health = f"{hp.group(1)}/{hp.group(2)} PW ({hp.group(3)})" if hp else "stan nieznany"
    who = " ".join(part for part in [actor.get("race"), actor.get("class")] if part)
    effects = ", ".join(actor.get("effects") or []) if isinstance(actor.get("effects"), list) else (actor.get("effects") or "")
    line = f"{actor.get('name')}: {who or 'istota'}, {health}"
    if effects:
        line += f", efekty: {effects}"
    return line


def convert(turn, session, index):
    narration = clean(" ".join(turn.get("after_utterances") or []))
    verdict = clean(" ".join(turn.get("automation_results") or []))
    actors = turn.get("combat_state_before") or []
    if not (MIN_LINE <= len(narration) <= MAX_LINE) or not verdict or not actors:
        return None

    # Keep only narration that is about the action: it names the actor or
    # a target. Players also write about everything else between turns.
    names = [a.get("name") for a in [turn.get("current_actor")] + (turn.get("targets_after") or []) if a and a.get("name")]
    first_names = {name.split()[0].lower() for name in names}
    if first_names and not any(name in narration.lower() for name in first_names):
        return None

    state = "\n".join(actor_line(actor) for actor in actors[:12])
    said = clean(" ".join(turn.get("before_utterances") or []))
    commands = " ".join(FLAGS.sub("", command) for command in turn.get("commands_norm") or [])

    return {
        "source": f"fireball/{session}#{index}",
        "system": "Jesteś Mistrzem Gry w sesji Dungeons & Dragons. Opisujesz skutki akcji w walce krótko i obrazowo, "
                  "zgodnie z tym, co rozstrzygnęły zasady.\n\nSTAN WALKI:\n" + state,
        "user": (said + "\n\n" if said else "") + f"{VERDICT_MARK} Akcja: {commands}. Wynik: {verdict}",
        "assistant": narration,
    }


def main():
    out_path, paths = sys.argv[1], sys.argv[2:]
    kept = total = 0
    with open(out_path, "w", encoding="utf-8") as out:
        for path in paths:
            session = path.rsplit("/", 1)[-1].removesuffix(".jsonl")
            with open(path, encoding="utf-8") as handle:
                for index, line in enumerate(handle):
                    if not line.strip():
                        continue
                    total += 1
                    record = convert(json.loads(line), session, index)
                    if record:
                        kept += 1
                        out.write(json.dumps(record, ensure_ascii=False) + "\n")
    print(f"kept {kept} of {total} turns -> {out_path}")


if __name__ == "__main__":
    main()
