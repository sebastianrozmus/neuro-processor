#!/bin/bash
# Finishes what train.sh does after training: fuses an adapter into the
# base weights, converts to GGUF and registers the result in Ollama. For
# when the training itself ended but the machine ran out of memory on the
# fuse, or to publish a kept adapter again.
#
#   NEURO_MODEL_NAME=neuro-npc-run5 tools/model/publish.sh [ADAPTER_DIR]
#
# ADAPTER_DIR defaults to the adapters folder train.sh writes. Quit the game
# and anything else holding a model in Ollama first: the dequantised weights
# take about 10 GB on their own.
set -eu

DATA_DIR="$HOME/Library/Application Support/neuro-processor"
VENV="$DATA_DIR/venv"
WORK="$DATA_DIR/model"
BASE="${NEURO_BASE_MODEL:-speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit}"
NAME="${NEURO_MODEL_NAME:-neuro-npc}"
ADAPTERS="${1:-$WORK/adapters}"

echo "Wtapiam adapter $ADAPTERS w wagi..."
rm -rf "$WORK/fused"
"$VENV/bin/python3" -m mlx_lm.fuse --model "$BASE" --adapter-path "$ADAPTERS" --save-path "$WORK/fused" --dequantize

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

rm -rf "$WORK/fused" "$WORK/neuro-npc-q8_0.gguf"
echo "Gotowe. W .env.local ustaw: NEURO_TALK_MODEL=$NAME (albo NEURO_LLM_MODEL=$NAME)"
