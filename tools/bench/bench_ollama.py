#!/usr/bin/env python3
"""Run a small NPC regression suite through Ollama /api/chat.
Gold/reference answers are NEVER sent to Ollama. Standard library only.
Automatic checks are syntax/number checks, not a semantic judge.
"""
from __future__ import annotations
import argparse
import collections
import hashlib
import json
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any
from npc_checks import read_jsonl, check_response, validate_messages, allowed_numbers

ROOT = Path(__file__).resolve().parent

def payload_for(case: dict[str, Any], model: str, options: dict[str, Any]) -> dict[str, Any]:
    errors = validate_messages(case["messages"], has_target=False)
    if errors:
        raise ValueError(f"{case.get('id')}: invalid prompt: {errors}")
    # Intentionally select only input messages. Never serialize the whole case.
    return {"model": model, "messages": case["messages"], "stream": False, "options": options}

def request_chat(base_url: str, payload: dict[str, Any], timeout: float) -> dict[str, Any]:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(
        base_url.rstrip("/") + "/api/chat", data=data,
        headers={"Content-Type": "application/json"}, method="POST"
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            result = json.load(response)
    except urllib.error.HTTPError as exc:
        detail = exc.read(4096).decode("utf-8", errors="replace")
        raise RuntimeError(f"Ollama HTTP {exc.code}: {detail}") from exc
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        raise RuntimeError(f"Ollama connection failed: {exc}") from exc
    if not isinstance(result, dict) or result.get("error"):
        raise RuntimeError(f"Ollama error: {result}")
    if not isinstance(result.get("message"), dict) or not isinstance(result["message"].get("content"), str):
        raise RuntimeError("Ollama returned no message.content string")
    return result

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", default="neuro-npc")
    parser.add_argument("--cases", type=Path, default=ROOT/"eval"/"regression-80.jsonl")
    parser.add_argument("--out", type=Path, default=ROOT/"reports"/"ollama-run")
    parser.add_argument("--url", default="http://localhost:11434")
    parser.add_argument("--temperature", type=float, default=0.2)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--num-ctx", type=int, default=4096)
    parser.add_argument("--num-predict", type=int, default=192)
    parser.add_argument("--timeout", type=float, default=180)
    parser.add_argument("--limit", type=int, default=0, help="0 means all cases")
    parser.add_argument("--options-json", type=Path, help="JSON object; overrides generation defaults")
    parser.add_argument("--check-references", action="store_true", help="Offline validation only; no model call")
    args = parser.parse_args()
    if args.limit < 0 or args.num_ctx < 1 or args.num_predict < 1 or args.timeout <= 0:
        parser.error("invalid limit, context size, prediction size or timeout")
    cases = read_jsonl(args.cases)
    if not cases:
        parser.error("case file is empty")
    if args.limit:
        cases = cases[:args.limit]
    ids = [case.get("id") for case in cases]
    if len(set(ids)) != len(ids):
        parser.error("duplicate case IDs")
    for case in cases:
        errors = validate_messages(case["messages"], has_target=False)
        if errors:
            parser.error(f"{case['id']}: {errors}")
        computed = allowed_numbers(case["messages"])
        if set(case["checks"]["allowed_numbers"]) != computed:
            parser.error(f"{case['id']}: allowed-number manifest is stale")
    options = {"temperature": args.temperature, "seed": args.seed,
               "num_ctx": args.num_ctx, "num_predict": args.num_predict}
    if args.options_json:
        extra_options = json.loads(args.options_json.read_text(encoding="utf-8"))
        if not isinstance(extra_options, dict):
            parser.error("--options-json must contain an object")
        options.update(extra_options)
    if args.check_references:
        failures = []
        for case in cases:
            checked = check_response(case["reference"], case["checks"])
            if checked["errors"]:
                failures.append({"id": case["id"], **checked})
        print(json.dumps({"mode": "OFFLINE_REFERENCE_LINT_NOT_MODEL_EVAL",
                          "cases": len(cases), "failures": failures}, ensure_ascii=False, indent=2))
        return 1 if failures else 0
    args.out.mkdir(parents=True, exist_ok=True)
    results_path = args.out / "responses.jsonl"
    if results_path.exists() or (args.out/"summary.json").exists():
        parser.error("output already contains results; choose a new --out directory")
    config = {"model": args.model, "url": args.url, "options": options,
              "case_file_sha256": hashlib.sha256(args.cases.read_bytes()).hexdigest(),
              "requested_cases": len(cases), "references_sent_to_model": False,
              "scope": "Ollama API only; not an in-game test",
              "semantics_automatically_scored": False}
    (args.out/"config.json").write_text(json.dumps(config, ensure_ascii=False, indent=2), encoding="utf-8")
    results = []
    transport_error = None
    try:
        with results_path.open("x", encoding="utf-8") as out:
            for i, case in enumerate(cases, 1):
                payload = payload_for(case, args.model, options)
                started = time.monotonic()
                raw = request_chat(args.url, payload, args.timeout)
                elapsed = time.monotonic() - started
                text = raw["message"]["content"]
                checked = check_response(text, case["checks"])
                if raw.get("done_reason") == "length":
                    checked["errors"].append("generation_truncated")
                row = {"id": case["id"], "genre": case["genre"], "role": case["role"],
                       "kind": case["kind"], "world": case["world"],
                       "user": case["messages"][-1]["content"], "response": text,
                       "reference_FOR_REVIEW_ONLY": case["reference"], "checks": checked,
                       "elapsed_seconds": round(elapsed, 3),
                       "done_reason": raw.get("done_reason"),
                       "prompt_eval_count": raw.get("prompt_eval_count"),
                       "eval_count": raw.get("eval_count"),
                       "manual_review": {"relevance_0_to_2": None, "decision_correct": None,
                                         "knowledge_correct": None, "natural_polish_0_to_2": None,
                                         "notes": None}}
                out.write(json.dumps(row, ensure_ascii=False) + "\n")
                out.flush()
                results.append(row)
                status = "FAIL" if checked["errors"] else "FORMAT_OK"
                print(f"[{i}/{len(cases)}] {case['id']} {status}", flush=True)
    except (RuntimeError, ValueError, KeyError) as exc:
        transport_error = str(exc)
        print(transport_error, file=sys.stderr)
    counts = collections.Counter(error for r in results for error in r["checks"]["errors"])
    by_role = {}
    for role in sorted({r["role"] for r in results}):
        subset = [r for r in results if r["role"] == role]
        by_role[role] = {"completed": len(subset),
                         "automatic_format_pass": sum(not r["checks"]["errors"] for r in subset)}
    summary = {
        **config, "completed_cases": len(results),
        "automatic_format_pass": sum(not r["checks"]["errors"] for r in results),
        "automatic_fail": sum(bool(r["checks"]["errors"]) for r in results),
        "error_counts": dict(counts), "by_role": by_role,
        "transport_error": transport_error,
        "conclusion": "Format/number checks only. Review relevance, payment direction, joining, healing and lore manually.",
        "full_suite_completed": len(results) == len(cases)
    }
    (args.out/"summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    review = ["# Ocena ręczna odpowiedzi", "",
              "Wzorzec nie jest jedyną poprawną odpowiedzią. FORMAT_OK nie oznacza poprawnej decyzji ani trafności.", ""]
    for r in results:
        review.extend([f"## {r['id']} — {r['role']} / {r['kind']}", "",
                       "**Gracz i decyzja:**", r["user"], "",
                       "**Model:**", r["response"], "",
                       "**Przykładowa poprawna odpowiedź:**", r["reference_FOR_REVIEW_ONLY"], "",
                       "**Sprawdź:** ostatni temat; decyzję; kierunek zapłaty; brak dopisanych działań; polszczyznę.",
                       "Ocena: ___; uwagi: ___", ""])
    (args.out/"review.md").write_text("\n".join(review), encoding="utf-8")
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 2 if transport_error else (1 if counts else 0)

if __name__ == "__main__":
    raise SystemExit(main())
