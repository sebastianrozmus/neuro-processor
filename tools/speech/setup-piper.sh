#!/bin/sh
# Prepares Piper, the light voice of the game: the program (MIT) and three
# Polish voices (gosia, darkman, mc_speech; CC0), about 200 MB in all, in
# the game's data directory. Piper runs on the processor and needs no
# graphics card. Afterwards pick it in the game: Esc → Ustawienia → Głos.
set -e

DATA="${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/neuro-processor/piper"
RELEASE="https://github.com/rhasspy/piper/releases/download/2023.11.14-2"
VOICES="https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/pl/pl_PL"

case "$(uname -s)-$(uname -m)" in
    Darwin-arm64) ARCHIVE=piper_macos_aarch64.tar.gz ;;
    Darwin-x86_64) ARCHIVE=piper_macos_x64.tar.gz ;;
    Linux-x86_64) ARCHIVE=piper_linux_x86_64.tar.gz ;;
    Linux-aarch64) ARCHIVE=piper_linux_aarch64.tar.gz ;;
    *) echo "Nie ma gotowego piper na $(uname -s) $(uname -m)."; exit 1 ;;
esac

mkdir -p "$DATA/voices"

echo "Pobieram piper..."
curl -fsSL "$RELEASE/$ARCHIVE" | tar -xz -C "$DATA"

for voice in gosia darkman mc_speech; do
    echo "Pobieram głos $voice..."
    for file in "pl_PL-$voice-medium.onnx" "pl_PL-$voice-medium.onnx.json"; do
        curl -fsSL -o "$DATA/voices/$file" "$VOICES/$voice/medium/$file"
    done
done

echo "Gotowe. W grze: Esc → Ustawienia → Głos: Piper."
