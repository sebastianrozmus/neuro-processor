#!/bin/bash
# Prepares WSL2 Ubuntu on a Windows PC with an NVIDIA card for training the
# game's voice: system packages, a Python environment with Unsloth (QLoRA,
# CUDA) and llama.cpp for writing GGUF files. Run once, inside Ubuntu:
#
#   bash setup.sh
#
# The NVIDIA driver is the Windows one; nothing is installed for CUDA inside
# Ubuntu. Check first that `nvidia-smi` in Ubuntu shows the card.
set -eu

HOME_DIR="${NEURO_TRAIN:-$HOME/neuro-train}"
mkdir -p "$HOME_DIR"

if ! command -v nvidia-smi >/dev/null || ! nvidia-smi >/dev/null 2>&1; then
  echo "nvidia-smi nie widzi karty. Zaktualizuj sterownik NVIDIA w Windows i uruchom ponownie WSL (wsl --shutdown)." >&2
  exit 1
fi

sudo apt-get update -q
sudo apt-get install -y -q python3-venv python3-dev build-essential cmake git curl libcurl4-openssl-dev

python3 -m venv "$HOME_DIR/venv"
"$HOME_DIR/venv/bin/pip" install -q --upgrade pip
"$HOME_DIR/venv/bin/pip" install -q unsloth datasets huggingface_hub

# llama.cpp: the converter for the adapter alone, and a fallback for the
# whole model when Unsloth's own export fails.
if [ ! -d "$HOME_DIR/llama.cpp" ]; then
  git clone -q --depth 1 https://github.com/ggml-org/llama.cpp.git "$HOME_DIR/llama.cpp"
fi
"$HOME_DIR/venv/bin/pip" install -q -r "$HOME_DIR/llama.cpp/requirements/requirements-convert_hf_to_gguf.txt"

"$HOME_DIR/venv/bin/python3" -c "import torch; print('CUDA:', torch.cuda.is_available(), torch.cuda.get_device_name(0) if torch.cuda.is_available() else '')"
echo "Gotowe. Dalej: tools/pc/README.md, krok 4."
