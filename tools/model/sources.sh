# Lists the training files in the game's data folder into the SOURCES
# array. Sourced by train.sh and pack.sh, so both train on the same set.
# Paths with spaces (the data folder has one) survive only in an array.
SOURCES=("$DATA_DIR/talks.jsonl")
if [ -f "$DATA_DIR/synth-fixed.jsonl" ]; then
  SOURCES+=("$DATA_DIR/synth-fixed.jsonl")
elif [ -f "$DATA_DIR/synth.jsonl" ]; then
  SOURCES+=("$DATA_DIR/synth.jsonl")
fi
for extra in "$DATA_DIR"/fireball*.jsonl "$DATA_DIR"/extra*.jsonl "$DATA_DIR"/manual*.jsonl "$MODEL_TOOLS"/manual*.jsonl; do
  [ -f "$extra" ] && SOURCES+=("$extra")
done
