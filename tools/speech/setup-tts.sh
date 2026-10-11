#!/bin/sh
# Prepares Chatterbox Multilingual TTS for the game: a Python environment of
# its own in the game's data directory (chatterbox-tts pins torch 2.6 and
# transformers 5.2, which would break the avatar environment) and the model
# weights (about 2 GB) in the Hugging Face cache. Run once; afterwards
# NEURO_TTS=chatterbox makes the game start the server by itself.
set -e

DATA="${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/neuro-processor"
VENV="$DATA/venv-tts"

mkdir -p "$DATA/voices"
python3 -m venv "$VENV"
"$VENV/bin/pip" install --quiet --upgrade pip
"$VENV/bin/pip" install --quiet chatterbox-tts

echo "Pobieram model mowy (pierwszy raz, ok. 2 GB)..."
"$VENV/bin/python3" - <<'PY'
import torch
original = torch.load
torch.load = lambda *a, **k: original(*a, **{**k, "map_location": k.get("map_location", "cpu")})
from chatterbox.mtl_tts import ChatterboxMultilingualTTS
try:
    ChatterboxMultilingualTTS.from_pretrained(device="cpu", t3_model="v3")
except TypeError:
    ChatterboxMultilingualTTS.from_pretrained(device="cpu")
print("Model mowy gotowy.")
PY

echo "Gotowe. W .env.local ustaw NEURO_TTS=chatterbox. Próbki głosów (kilka sekund czystej polskiej mowy, WAV) wrzuć do: $DATA/voices"
