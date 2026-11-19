#!/usr/bin/env python3
"""Offline, deliberately limited checks for Polish NPC dialogue.
This is NOT the game's validator and does not prove semantic correctness.
Python 3.10+, standard library only.
"""
from __future__ import annotations
import json
import re
from pathlib import Path
from typing import Any

MOODS = ("spokój", "radość", "gniew", "smutek", "strach", "kpina", "zdziwienie", "czułość")
MOOD_RE = re.compile(r"^\[(" + "|".join(MOODS) + r")\]\s+")
NUMBER_RE = re.compile(r"(?<!\w)\d+(?:[.,]\d+)?(?!\w)")
# Heuristic, not a complete Polish number parser. Intentionally excludes 'raz'
# (e.g. 'płatne raz') and idioms with no definite numeral.
SPELLED_RE = re.compile(
    r"\b(?:zero|jeden|jedna|jedno|jedną|jednego|jednej|jednym|"
    r"dwa|dwie|dwóch|dwoma|dwu|trzy|trzech|trzema|"
    r"cztery|czterech|czterema|pięć|pięciu|pięcioma|"
    r"sześć|sześciu|siedem|siedmiu|osiem|ośmiu|"
    r"dziewięć|dziewięciu|dziesięć|dziesięciu|"
    r"jedenaście|dwanaście|trzynaście|czternaście|piętnaście|"
    r"szesnaście|siedemnaście|osiemnaście|dziewiętnaście|"
    r"dwadzieścia|trzydzieści|czterdzieści|pięćdziesiąt|"
    r"sześćdziesiąt|siedemdziesiąt|osiemdziesiąt|dziewięćdziesiąt|"
    r"sto|dwieście|trzysta|tysiąc)\b", re.IGNORECASE
)

def read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows = []
    with path.open(encoding="utf-8") as fh:
        for line_no, line in enumerate(fh, 1):
            if not line.strip():
                raise ValueError(f"{path}:{line_no}: empty JSONL line")
            try:
                row = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"{path}:{line_no}: invalid JSON: {exc}") from exc
            if not isinstance(row, dict):
                raise ValueError(f"{path}:{line_no}: object required")
            rows.append(row)
    return rows

def numbers(text: str) -> set[str]:
    return {s.replace(",", ".") for s in NUMBER_RE.findall(text)}

def sentences(text: str) -> int:
    body = MOOD_RE.sub("", text.strip(), count=1)
    body = re.sub(r"\b(szt|np|itp|itd)\.", r"\1", body, flags=re.IGNORECASE)
    return len([p for p in re.split(r"[.!?]+(?:\s+|$)", body) if p.strip()])

def allowed_numbers(messages: list[dict[str, str]]) -> set[str]:
    """Only current GRA and actual price lines; never arbitrary SYSTEM numbers."""
    current = messages[-1]["content"]
    if "\n\n[GRA] " not in current:
        raise ValueError("last user message has no final GRA block")
    allowed = numbers(current.split("\n\n[GRA] ", 1)[1])
    for line in messages[0]["content"].splitlines():
        if line.startswith(("Twój towar i ceny za sztukę", "Twoja służba kosztuje:")):
            allowed |= numbers(line)
    return allowed

def validate_messages(messages: list[dict[str, str]], *, has_target: bool) -> list[str]:
    errors = []
    min_count = 3 if has_target else 2
    if not isinstance(messages, list) or len(messages) < min_count:
        return ["messages_too_short"]
    if messages[0].get("role") != "system":
        errors.append("first_role_not_system")
    for index, msg in enumerate(messages):
        if not isinstance(msg, dict) or not isinstance(msg.get("content"), str) or not msg["content"].strip():
            errors.append(f"invalid_message_{index}")
            continue
        expected = "system" if index == 0 else ("user" if index % 2 else "assistant")
        if msg.get("role") != expected:
            errors.append(f"role_order_{index}")
    if messages[-1].get("role") != ("assistant" if has_target else "user"):
        errors.append("wrong_last_role")
    user_index = -2 if has_target else -1
    current = messages[user_index]["content"]
    if current.count("[GRA]") != 1 or "\n\n[GRA] " not in current:
        errors.append("final_gra_count_or_separator")
    if not current.endswith("Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi."):
        errors.append("missing_final_instruction")
    end = len(messages) - (2 if has_target else 1)
    for index in range(1, end):
        if "[GRA]" in messages[index]["content"]:
            errors.append(f"gra_in_history_{index}")
    return errors

def check_response(text: str, checks: dict[str, Any]) -> dict[str, Any]:
    errors, warnings = [], []
    if not isinstance(text, str) or not text.strip():
        return {"errors": ["empty_response"], "warnings": [], "sentence_count": 0,
                "numbers": [], "spelled_numerals": []}
    text = text.strip()
    mood = MOOD_RE.match(text)
    if not mood:
        errors.append("missing_or_invalid_mood")
        body = text
    else:
        body = text[mood.end():]
    if any(token in body for token in ("[", "]", "*", "```", "«", "»", '"', "„", "”")):
        errors.append("markup_or_quotes_in_speech")
    if re.search(r"<\|[^>]+\|>|</?think>", body):
        errors.append("model_control_tokens")
    npc_name = checks.get("npc_name", "")
    if npc_name and re.match(re.escape(npc_name) + r"\s*:", body, re.IGNORECASE):
        errors.append("speaker_signature")
    sentence_count = sentences(text)
    if not 1 <= sentence_count <= 3:
        errors.append("sentence_count")
    found = numbers(body)
    allowed = set(checks.get("allowed_numbers", []))
    if found - allowed:
        errors.append("forbidden_numbers:" + ",".join(sorted(found - allowed)))
    missing = set(checks.get("required_numbers", [])) - found
    if missing:
        errors.append("missing_required_numbers:" + ",".join(sorted(missing)))
    if checks.get("no_numbers") and found:
        errors.append("numbers_in_non_numeric_case")
    spelled = sorted({m.group().lower() for m in SPELLED_RE.finditer(body)})
    if spelled:
        errors.append("spelled_numerals:" + ",".join(spelled))
    hints = checks.get("topic_hints", [])
    if hints and not any(h.lower() in body.lower() for h in hints):
        warnings.append("topic_hint_missing_REQUIRES_HUMAN_REVIEW")
    return {"errors": errors, "warnings": warnings, "sentence_count": sentence_count,
            "numbers": sorted(found), "spelled_numerals": spelled}

def normalize(text: str) -> str:
    return re.sub(r"\s+", " ", text.strip().casefold())
