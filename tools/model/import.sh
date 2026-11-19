#!/bin/bash
# Registers in Ollama a model trained elsewhere (tools/pc): a GGUF file of
# the whole model, or a LoRA adapter in GGUF on top of a base Ollama has.
#
#   tools/model/import.sh neuro-npc-run7 ~/Downloads/neuro-npc-run7.Q5_K_M.gguf
#   tools/model/import.sh neuro-npc-run7 ~/Downloads/neuro-run7-lora.gguf SpeakLeash/bielik-11b-v2.3-instruct:Q4_K_M
#
# The prompt template, stop token and sampling are the game's, as in train.sh.
set -eu

NAME="$1"
FILE="$2"
BASE="${3:-}"
WORK="$HOME/Library/Application Support/neuro-processor/model"
mkdir -p "$WORK"

if [ -n "$BASE" ]; then
  FROM="FROM $BASE
ADAPTER \"$FILE\""
else
  FROM="FROM \"$FILE\""
fi

cat > "$WORK/Modelfile-$NAME" <<MODELFILE
$FROM
TEMPLATE """{{- range .Messages }}<|im_start|>{{ .Role }}
{{ .Content }}<|im_end|>
{{ end }}<|im_start|>assistant
"""
PARAMETER stop <|im_end|>
PARAMETER temperature 0.8
PARAMETER num_ctx 4096
MODELFILE

ollama create "$NAME" -f "$WORK/Modelfile-$NAME"
echo "Gotowe. Sprawdź modelem: tools/bench/bench.sh $NAME i tools/bench/replay.py (tools/bench/README.md), potem w .env.local: NEURO_TALK_MODEL=$NAME"
