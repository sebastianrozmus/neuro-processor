"""Record the microphone until told to stop, then write what was said.

Usage: python3 listen.py --out FILE

Recording lasts until the process gets SIGINT or SIGTERM. The recognised
text (Polish, via a local Whisper model) is written to FILE and printed.
"""
import argparse
import signal
import sys
import tempfile
import os

RATE = 16000
MODEL = "mlx-community/whisper-large-v3-turbo"

# The stop signal may come at any moment, even while the libraries are
# still loading, so it is caught before anything else happens.
stop = {"now": False}


def halt(*_):
    stop["now"] = True


signal.signal(signal.SIGINT, halt)
signal.signal(signal.SIGTERM, halt)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", required=True)
    parser.add_argument("--max-seconds", type=int, default=60)
    args = parser.parse_args()

    import numpy as np
    import sounddevice as sd

    chunks = []

    def collect(indata, frames, time, status):
        chunks.append(indata.copy())
        if len(chunks) * frames >= RATE * args.max_seconds:
            stop["now"] = True

    with sd.InputStream(samplerate=RATE, channels=1, dtype="float32", callback=collect):
        # The game waits for this line before it lets the player stop.
        print("recording", flush=True)

        while not stop["now"]:
            sd.sleep(50)

    if not chunks:
        open(args.out, "w").close()
        return

    audio = np.concatenate(chunks)[:, 0]

    # Too short, or silence (a muted or denied microphone): Whisper would
    # only make something up.
    if len(audio) < RATE // 2 or float(np.abs(audio).max()) < 0.01:
        open(args.out, "w").close()
        return

    import soundfile as sf
    import mlx_whisper

    path = os.path.join(tempfile.gettempdir(), "neuro-listen.wav")
    sf.write(path, audio, RATE)
    result = mlx_whisper.transcribe(path, path_or_hf_repo=MODEL, language="pl")
    text = " ".join(result.get("text", "").split())

    with open(args.out, "w", encoding="utf-8") as handle:
        handle.write(text)
    print(text)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:  # noqa: BLE001 - the game shows this message to the player
        print(f"listen: {error}", file=sys.stderr)
        sys.exit(1)
