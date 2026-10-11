"""Serves Chatterbox Multilingual TTS to the game over HTTP.

The game starts this (embedded, over stdin) with the Python of the environment tools/speech/setup-tts.sh
makes (chatterbox-tts pins its own torch, so it lives apart from the avatar
environment). The model loads once; every line is then one request:

    POST /speak   {"text": "...", "language": "pl", "voice": "kowal.wav",
                   "exaggeration": 0.5, "cfg_weight": 0.5, "temperature": 0.8}
                  -> audio/wav
    GET  /voices  -> {"voices": ["kowal.wav", ...]}   (files in --voices)
    GET  /health  -> {"ok": true, "device": "mps", "loaded": true}

Usage: python3 chatterbox.py --port 7790 --voices DIR [--device mps|cpu]
"""
import argparse
import io
import json
import os
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

model = None
model_lock = threading.Lock()
device = "cpu"
voices_dir = ""


def log(*parts):
    print(time.strftime("%H:%M:%S"), *parts, file=sys.stderr, flush=True)


def pick_device(wanted):
    import torch
    if wanted and wanted != "auto":
        return wanted
    if torch.backends.mps.is_available():
        return "mps"
    if torch.cuda.is_available():
        return "cuda"
    return "cpu"


def load(wanted):
    """Loads the multilingual model once. On Apple Silicon the checkpoints
    are saved for CUDA, so torch.load is pointed at the chosen device."""
    global model, device
    import torch
    device = pick_device(wanted)
    original = torch.load

    def patched(*args, **kwargs):
        kwargs.setdefault("map_location", torch.device(device))
        return original(*args, **kwargs)

    torch.load = patched
    from chatterbox.mtl_tts import ChatterboxMultilingualTTS
    started = time.time()
    model = ChatterboxMultilingualTTS.from_pretrained(device=device, t3_model="v3")
    log(f"model ready on {device} in {time.time() - started:.0f}s")


def synthesize(body):
    import torch
    import torchaudio
    text = (body.get("text") or "").strip()
    if not text:
        raise ValueError("pusty tekst")
    language = body.get("language") or "pl"
    voice = body.get("voice") or ""
    prompt = None
    if voice:
        candidate = voice if os.path.isabs(voice) else os.path.join(voices_dir, voice)
        if os.path.isfile(candidate):
            prompt = candidate
    kwargs = {
        "language_id": language,
        "exaggeration": float(body.get("exaggeration", 0.5)),
        "cfg_weight": float(body.get("cfg_weight", 0.5)),
        "temperature": float(body.get("temperature", 0.8)),
    }
    if prompt:
        kwargs["audio_prompt_path"] = prompt
    with model_lock:
        started = time.time()
        wav = model.generate(text, **kwargs)
        log(f"{len(text)} chars -> {wav.shape[-1] / model.sr:.1f}s audio in {time.time() - started:.1f}s (voice {voice or 'default'})")
    buffer = io.BytesIO()
    torchaudio.save(buffer, wav.cpu(), model.sr, format="wav")
    return buffer.getvalue()


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def send_json(self, code, payload):
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path.startswith("/health"):
            self.send_json(200, {"ok": True, "device": device, "loaded": model is not None})
        elif self.path.startswith("/voices"):
            names = sorted(n for n in os.listdir(voices_dir) if n.lower().endswith((".wav", ".mp3", ".flac"))) if os.path.isdir(voices_dir) else []
            self.send_json(200, {"voices": names})
        else:
            self.send_json(404, {"error": "nie ma takiej ścieżki"})

    def do_POST(self):
        if not self.path.startswith("/speak"):
            self.send_json(404, {"error": "nie ma takiej ścieżki"})
            return
        if model is None:
            self.send_json(503, {"error": "model jeszcze się ładuje"})
            return
        length = int(self.headers.get("Content-Length") or 0)
        try:
            body = json.loads(self.rfile.read(length) or b"{}")
            data = synthesize(body)
        except Exception as error:  # noqa: BLE001 - every failure is reported to the game
            log("speak failed:", error)
            self.send_json(400, {"error": str(error)})
            return
        self.send_response(200)
        self.send_header("Content-Type", "audio/wav")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def main():
    global voices_dir
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=7790)
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--voices", default="")
    parser.add_argument("--device", default="auto")
    args = parser.parse_args()
    voices_dir = args.voices
    server = ThreadingHTTPServer((args.host, args.port), Handler)
    log(f"listening on http://{args.host}:{args.port}/ (voices: {voices_dir or 'none'})")
    threading.Thread(target=load, args=(args.device,), daemon=True).start()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
