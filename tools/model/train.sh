#!/bin/bash
# Fine-tunes a small Polish model on the game's talks with LoRA (MLX, Apple
# Silicon), fuses the adapter into the weights and hands the result to
# Ollama under the name neuro-npc.
#
#   tools/model/train.sh [ITERS]            (NEURO_MODEL_NAME=neuro-npc-run4 for a candidate)
#
# Needs the venv from tools/avatar/setup.sh (mlx-lm is installed there) and
# the data folder prepared by prepare.py (default: ~/Library/Application
# Support/neuro-processor/model/data). Sized for 18 GB of unified memory:
# one sequence at a time, gradients accumulated over four, activations
# recomputed. Quit the game (and anything else holding a model in Ollama)
# before starting. The log lands next to the adapters as train.log.
set -eu

DATA_DIR="$HOME/Library/Application Support/neuro-processor"
VENV="$DATA_DIR/venv"
WORK="$DATA_DIR/model"
BASE="${NEURO_BASE_MODEL:-speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit}"
ITERS="${1:-400}"
BATCH="${NEURO_BATCH:-1}"
ACCUMULATE="${NEURO_ACCUMULATE:-4}"
LAYERS="${NEURO_LAYERS:-24}"
RATE="${NEURO_RATE:-1e-4}"
# NEURO_MODEL_NAME registers the result under another name in Ollama, so a
# new run can be compared with the one the game uses before replacing it.
NAME="${NEURO_MODEL_NAME:-neuro-npc}"
# NEURO_RESUME continues from an adapter file (a short run on new data on
# top of the last model instead of a long one from scratch).
RESUME="${NEURO_RESUME:-}"
# Validation every 100 steps on 50 examples: run 3 scored 25 per point and
# the curve jumped by ±0.3, which said more about the sample than the model.
VAL_BATCHES="${NEURO_VAL_BATCHES:-50}"

mkdir -p "$WORK"
exec > >(tee -a "$WORK/train.log") 2>&1

echo "Zbieram dane..."
MODEL_TOOLS="$(dirname "$0")"
. "$MODEL_TOOLS/sources.sh"
"$VENV/bin/python3" "$(dirname "$0")/prepare.py" "$WORK/data" "${SOURCES[@]}"

# The last adapter is kept aside, so two runs can be compared.
if [ -d "$WORK/adapters" ]; then
  rm -rf "$WORK/adapters-prev"
  mv "$WORK/adapters" "$WORK/adapters-prev"
fi

echo "Trenuję LoRA na $BASE ($ITERS kroków po $BATCH x $ACCUMULATE)..."
# --mask-prompt: the loss counts only the reply, not the thousand tokens
# of rules in front of it, so the model learns to answer, not to recite.
RESUME_ARGS=()
if [ -n "$RESUME" ]; then
  echo "Kontynuuję od adaptera $RESUME"
  RESUME_ARGS=(--resume-adapter-file "$RESUME")
fi
"$VENV/bin/python3" -m mlx_lm.lora \
  --model "$BASE" --train --data "$WORK/data" --mask-prompt \
  --fine-tune-type lora --num-layers "$LAYERS" --batch-size "$BATCH" \
  --grad-accumulation-steps "$ACCUMULATE" --grad-checkpoint --iters "$ITERS" \
  --learning-rate "$RATE" --steps-per-eval 100 --val-batches "$VAL_BATCHES" --save-every 50 \
  --adapter-path "$WORK/adapters" --max-seq-length 2048 "${RESUME_ARGS[@]}"

echo "Sprawdzam na zbiorze testowym..."
"$VENV/bin/python3" -m mlx_lm.lora --model "$BASE" --data "$WORK/data" --adapter-path "$WORK/adapters" --test --mask-prompt

echo "Wtapiam adapter w wagi..."
rm -rf "$WORK/fused"
"$VENV/bin/python3" -m mlx_lm.fuse --model "$BASE" --adapter-path "$WORK/adapters" --save-path "$WORK/fused" --dequantize

# The fused weights come without the tokenizer; it is copied from the base
# model, which is already in the Hugging Face cache.
"$VENV/bin/python3" - "$BASE" "$WORK/fused" <<'PY'
import shutil, sys
from huggingface_hub import snapshot_download
base = snapshot_download(sys.argv[1], allow_patterns=["tokenizer*", "special_tokens_map.json"])
for name in ["tokenizer.json", "tokenizer_config.json", "special_tokens_map.json", "tokenizer.model"]:
    try:
        shutil.copy(f"{base}/{name}", sys.argv[2])
    except FileNotFoundError:
        pass
PY

# Ollama's own importer mangles Bielik's tokenizer, so the weights go
# through llama.cpp's converter to a q8_0 GGUF first.
LLAMA="${NEURO_LLAMA_CPP:-$WORK/llama.cpp}"
if [ ! -f "$LLAMA/convert_hf_to_gguf.py" ]; then
  echo "Pobieram llama.cpp (konwerter do GGUF)..."
  git clone --depth 1 -q https://github.com/ggml-org/llama.cpp.git "$LLAMA"
fi

"$VENV/bin/pip" install -q "$LLAMA/gguf-py"

echo "Konwertuję do GGUF (q8_0)..."
"$VENV/bin/python3" "$LLAMA/convert_hf_to_gguf.py" "$WORK/fused" --outfile "$WORK/neuro-npc-q8_0.gguf" --outtype q8_0

cat > "$WORK/Modelfile" <<MODELFILE
FROM "$WORK/neuro-npc-q8_0.gguf"
TEMPLATE """{{- range .Messages }}<|im_start|>{{ .Role }}
{{ .Content }}<|im_end|>
{{ end }}<|im_start|>assistant
"""
PARAMETER stop <|im_end|>
PARAMETER temperature 0.8
PARAMETER num_ctx 4096
MODELFILE

echo "Rejestruję w Ollamie jako $NAME..."
ollama create "$NAME" -f "$WORK/Modelfile"

# Ollama keeps its own copy of the GGUF, and the fused weights can be made
# again from the adapter, so neither needs to take up 14 GB of disk.
rm -rf "$WORK/fused" "$WORK/neuro-npc-q8_0.gguf"
echo "Gotowe. W .env.local ustaw: NEURO_TALK_MODEL=$NAME (albo NEURO_LLM_MODEL=$NAME)"
