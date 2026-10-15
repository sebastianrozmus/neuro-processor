#!/bin/sh
# Measures a model in Ollama on the 80 regression cases, in two seeds.
#
#   tools/bench/bench.sh neuro-npc-run7          # seeds 42 and 7
#   tools/bench/bench.sh neuro-npc-run7 42 7 3   # other seeds
#
# The cases' small-talk briefs are first rewritten to the ones the game sends
# now (tools/model/rebrief.py reads internal/game/narration.go), so the bench
# follows the code. Results land in tools/bench/results/<model>-s<seed>;
# compare.py prints them side by side.
set -e

[ -n "$1" ] || { echo "użycie: $0 MODEL [SEED...]" >&2; exit 2; }
MODEL="$1"; shift
[ $# -gt 0 ] || set -- 42 7

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BENCH="$ROOT/tools/bench"
CASES="$BENCH/eval/regression-80-game.jsonl"

python3 "$ROOT/tools/model/rebrief.py" "$ROOT/internal/game/narration.go" "$BENCH/eval/regression-80.jsonl" "$CASES"

for SEED in "$@"; do
	OUT="$BENCH/results/$MODEL-s$SEED"
	if [ -e "$OUT/summary.json" ]; then
		echo "$OUT już jest — pomijam (usuń folder, żeby zmierzyć od nowa)"
		continue
	fi
	rm -rf "$OUT"
	echo "== $MODEL, seed $SEED"
	python3 "$BENCH/bench_ollama.py" --model "$MODEL" --cases "$CASES" --seed "$SEED" --out "$OUT" \
		--temperature 0.8 --num-predict 220 --num-ctx 4096 --timeout 300 | tail -1
done

python3 "$BENCH/compare.py" "$MODEL"
