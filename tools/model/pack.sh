#!/bin/bash
# Packs the training set for a PC with an NVIDIA card (tools/pc): the same
# sources and the same split train.sh uses, as one archive on the Desktop.
#
#   tools/model/pack.sh            -> ~/Desktop/neuro-data.tar.gz
set -eu

DATA_DIR="$HOME/Library/Application Support/neuro-processor"
VENV="$DATA_DIR/venv"
WORK="$DATA_DIR/model"
MODEL_TOOLS="$(dirname "$0")"
OUT="${1:-$HOME/Desktop/neuro-data.tar.gz}"

. "$MODEL_TOOLS/sources.sh"
"$VENV/bin/python3" "$MODEL_TOOLS/prepare.py" "$WORK/data" "${SOURCES[@]}"

tar -czf "$OUT" -C "$WORK" data
echo "Gotowe: $OUT ($(du -h "$OUT" | cut -f1)). Skopiuj na PC do folderu neuro-train (szczegóły: tools/pc/README.md)."
