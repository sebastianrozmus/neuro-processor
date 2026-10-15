#!/bin/sh
# Publishes a model from the local Ollama on Hugging Face, with the model
# card (README.md), the Ollama chat template and parameters from this folder.
#
#   tools/hf/upload.sh                                    # neuro-npc-run7 -> sebastianrozmus/neuro-npc-bielik-11b-GGUF
#   tools/hf/upload.sh neuro-npc-run7 user/repo neuro-npc-bielik-11b-Q5_K_M.gguf
#   DRY=1 tools/hf/upload.sh                              # only prepare the folder
#
# Needs the Hugging Face CLI (pip install -U huggingface_hub) and a token with
# write access: hf auth login. Afterwards anyone can run
#   ollama pull hf.co/<repo>:Q5_K_M
# (Ollama picks the file whose name ends with the quantization).
set -e

MODEL="${1:-neuro-npc-run7}"
REPO="${2:-sebastianrozmus/neuro-npc-bielik-11b-GGUF}"
FILE="${3:-neuro-npc-bielik-11b-Q5_K_M.gguf}"

HERE="$(cd "$(dirname "$0")" && pwd)"
STAGE="${STAGE:-${TMPDIR:-/tmp}/neuro-hf}"

BLOB="$(ollama show "$MODEL" --modelfile | sed -n 's/^FROM //p' | head -1)"
[ -f "$BLOB" ] || { echo "nie znalazłem wag modelu $MODEL w Ollamie ($BLOB)" >&2; exit 1; }

mkdir -p "$STAGE"
cp "$HERE/README.md" "$HERE/template" "$HERE/params" "$STAGE/"
# A hard link costs no disk space; across volumes it falls back to a copy.
ln -f "$BLOB" "$STAGE/$FILE" 2>/dev/null || cp "$BLOB" "$STAGE/$FILE"
ls -la "$STAGE"

if [ -n "$DRY" ]; then
	echo "Gotowe do wysłania: $STAGE"
	exit 0
fi

command -v hf >/dev/null || { echo "brak polecenia hf: pip install -U huggingface_hub, potem hf auth login" >&2; exit 1; }
hf repo create "$REPO" --repo-type model --exist-ok
hf upload "$REPO" "$STAGE" . --repo-type model --commit-message "neuro-npc: $MODEL"
echo "Opublikowane: https://huggingface.co/$REPO"
echo "Test: ollama pull hf.co/$REPO:Q5_K_M"
