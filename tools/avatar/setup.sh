#!/bin/sh
# Prepares the local image model that draws avatars: a Python environment
# with diffusers in the game's data directory and the SD-Turbo weights
# (about 2.5 GB) in the Hugging Face cache. Run once; afterwards the game
# finds the environment by itself.
set -e

DATA="${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/neuro-processor"
VENV="$DATA/venv"

mkdir -p "$DATA"
python3 -m venv "$VENV"
"$VENV/bin/pip" install --quiet --upgrade pip
"$VENV/bin/pip" install --quiet torch diffusers transformers accelerate safetensors pillow numpy sounddevice soundfile mlx-whisper

echo "Pobieram model obrazkowy (pierwszy raz, ok. 2,5 GB)..."
"$VENV/bin/python3" - <<'PY'
import torch
from diffusers import AutoPipelineForText2Image
AutoPipelineForText2Image.from_pretrained("stabilityai/sd-turbo", torch_dtype=torch.float16, variant="fp16")
print("Model obrazkowy gotowy.")
PY

echo "Pobieram model rozpoznawania mowy (ok. 1,6 GB)..."
"$VENV/bin/python3" - <<'PY'
import numpy, soundfile, tempfile, os, mlx_whisper
path = os.path.join(tempfile.gettempdir(), "neuro-silence.wav")
soundfile.write(path, numpy.zeros(16000, dtype="float32"), 16000)
mlx_whisper.transcribe(path, path_or_hf_repo="mlx-community/whisper-large-v3-turbo", language="pl")
print("Gotowe. Rysowanie awatarów jest pod \"Dodaj nowy...\", mikrofon pod F5.")
PY
