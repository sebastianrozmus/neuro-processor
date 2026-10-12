#!/bin/bash
# Converts a trained adapter to GGUF, for Ollama's ADAPTER on a base it
# already has (tools/model/import.sh with three arguments). Small (~100 MB)
# and needs no merging, so it works where RAM is short.
#
#   bash lora-gguf.sh ~/neuro-train/run7/adapter neuro-run7-lora.gguf
set -eu

HOME_DIR="${NEURO_TRAIN:-$HOME/neuro-train}"
ADAPTER="$1"
OUT="${2:-$HOME_DIR/neuro-lora.gguf}"
BASE="${NEURO_BASE:-speakleash/Bielik-11B-v2.3-Instruct}"

"$HOME_DIR/venv/bin/python3" "$HOME_DIR/llama.cpp/convert_lora_to_gguf.py" "$ADAPTER" --base-model-id "$BASE" --outtype f16 --outfile "$OUT"
echo "Gotowe: $OUT"
