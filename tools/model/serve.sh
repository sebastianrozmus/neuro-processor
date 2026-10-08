#!/bin/bash
# Serves the game's own model (the LoRA adapter trained by train.sh on top
# of the base model) through mlx_lm.server, which speaks the OpenAI API the
# game uses. Ollama's import of these weights breaks the tokenizer, so this
# is the way to play with the model:
#
#   tools/model/serve.sh [PORT]
#
# and in .env.local, keeping the main model for reading intents:
#
#   NEURO_TALK_BASE_URL=http://localhost:8080/v1
#   NEURO_TALK_MODEL=speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit
set -eu

DATA_DIR="$HOME/Library/Application Support/neuro-processor"
VENV="$DATA_DIR/venv"
WORK="$DATA_DIR/model"
BASE="${NEURO_BASE_MODEL:-speakleash/Bielik-4.5B-v3.0-Instruct-MLX-8bit}"
PORT="${1:-8080}"

if [ ! -f "$WORK/adapters/adapters.safetensors" ]; then
  echo "Brak adaptera w $WORK/adapters. Najpierw: tools/model/train.sh" >&2
  exit 1
fi

echo "Serwuję $BASE z adapterem na http://localhost:$PORT/v1 (model: $BASE)."
echo "W .env.local: NEURO_TALK_BASE_URL=http://localhost:$PORT/v1 i NEURO_TALK_MODEL=$BASE"
exec "$VENV/bin/python3" -m mlx_lm.server --model "$BASE" --adapter-path "$WORK/adapters" --port "$PORT" --log-level WARNING
