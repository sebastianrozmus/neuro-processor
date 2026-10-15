# Windows counterpart of setup-tts.sh: Chatterbox Multilingual TTS in an
# environment of its own in the game's data directory (%APPDATA%\neuro-processor),
# with the CUDA build of torch 2.6 for an NVIDIA card, and the model weights
# (about 2 GB) in the Hugging Face cache. Run once from PowerShell:
#
#   powershell -ExecutionPolicy Bypass -File tools\speech\setup-tts.ps1
#
# Needs uv (https://docs.astral.sh/uv/) or Python 3.11 from python.org.
$ErrorActionPreference = "Stop"

$data = Join-Path $env:APPDATA "neuro-processor"
$venv = Join-Path $data "venv-tts"
$python = Join-Path $venv "Scripts\python.exe"
New-Item -ItemType Directory -Force (Join-Path $data "voices") | Out-Null

$uv = [bool](Get-Command uv -ErrorAction SilentlyContinue)
if ($uv) {
    uv venv --python 3.11 --quiet $venv
} else {
    py -3.11 -m venv $venv
    & $python -m pip install --quiet --upgrade pip
}

function Install([string[]]$packages) {
    if ($uv) { uv pip install --quiet --python $python @packages }
    else { & $python -m pip install --quiet @packages }
    if ($LASTEXITCODE) { throw "instalacja nie powiodła się: $packages" }
}

Install "chatterbox-tts"
# chatterbox-tts pulls the CPU torch on Windows; the CUDA one of the same
# version makes a line take seconds instead of minutes.
$reinstall = if ($uv) { "--reinstall-package" } else { "--force-reinstall" }
if ($uv) {
    Install @("--reinstall-package", "torch", "--reinstall-package", "torchaudio", "torch==2.6.0", "torchaudio==2.6.0", "--index-url", "https://download.pytorch.org/whl/cu124")
} else {
    Install @($reinstall, "torch==2.6.0", "torchaudio==2.6.0", "--index-url", "https://download.pytorch.org/whl/cu124")
}

Write-Host "Pobieram model mowy (pierwszy raz, ok. 2 GB)..."
@'
import torch
print("CUDA:", torch.cuda.is_available(), torch.cuda.get_device_name(0) if torch.cuda.is_available() else "")
original = torch.load
torch.load = lambda *a, **k: original(*a, **{**k, "map_location": k.get("map_location", "cpu")})
from chatterbox.mtl_tts import ChatterboxMultilingualTTS
try:
    ChatterboxMultilingualTTS.from_pretrained(device="cpu", t3_model="v3")
except TypeError:
    ChatterboxMultilingualTTS.from_pretrained(device="cpu")
print("Model mowy gotowy.")
'@ | & $python -

Write-Host "Gotowe. W .env.local ustaw NEURO_TTS=chatterbox. Próbki głosów (kilka sekund czystej polskiej mowy, WAV) wrzuć do: $data\voices"
